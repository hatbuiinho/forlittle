package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"forlittle/server/internal/config"
	"forlittle/server/internal/models"
	"forlittle/server/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminHandler struct {
	DB  *gorm.DB
	Cfg config.Config
}

type AdminLoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

const AdminSessionCookieName = "forlittle_admin_session"

type VisitLogGroup struct {
	GroupID           string    `json:"group_id"`
	MachineID         string    `json:"machine_id"`
	ProfileInstanceID string    `json:"profile_instance_id"`
	Domain            string    `json:"domain"`
	URL               string    `json:"url"`
	Title             string    `json:"title"`
	Action            string    `json:"action"`
	VisitCount        int       `json:"visit_count"`
	FirstVisitedAt    time.Time `json:"first_visited_at"`
	LastVisitedAt     time.Time `json:"last_visited_at"`
}

func (h AdminHandler) Login(c *gin.Context) {
	var input AdminLoginRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}

	var user models.User
	if err := h.DB.Where("email = ? AND status = ?", strings.ToLower(strings.TrimSpace(input.Email)), "active").First(&user).Error; err != nil {
		unauthorized(c, "invalid credentials")
		return
	}

	if user.Role != "admin" || !services.CheckPassword(user.PasswordHash, input.Password) {
		unauthorized(c, "invalid credentials")
		return
	}

	token, err := services.NewOpaqueToken()
	if err != nil {
		internalServerError(c, "could not create session")
		return
	}

	session := models.UserSession{
		UserID:    user.ID,
		TokenHash: services.HashToken(token),
		ExpiresAt: time.Now().UTC().Add(time.Duration(h.Cfg.AdminSessionTTLHours) * time.Hour),
	}
	if err := h.DB.Create(&session).Error; err != nil {
		internalServerError(c, "could not store session")
		return
	}

	setAdminSessionCookie(c, h.Cfg, token, session.ExpiresAt)
	c.JSON(http.StatusOK, gin.H{"user": userResponse(user)})
}

func (h AdminHandler) Logout(c *gin.Context) {
	token, err := c.Cookie(AdminSessionCookieName)
	if err == nil && token != "" {
		_ = h.DB.Where("token_hash = ?", services.HashToken(token)).Delete(&models.UserSession{}).Error
	}

	clearAdminSessionCookie(c, h.Cfg)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h AdminHandler) Me(c *gin.Context) {
	userValue, exists := c.Get("admin_user")
	if !exists {
		unauthorized(c, "unauthorized")
		return
	}

	user, ok := userValue.(models.User)
	if !ok {
		unauthorized(c, "unauthorized")
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": userResponse(user)})
}

func userResponse(user models.User) gin.H {
	return gin.H{
		"id":           user.ID,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"role":         user.Role,
		"status":       user.Status,
	}
}

func setAdminSessionCookie(c *gin.Context, cfg config.Config, token string, expiresAt time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     AdminSessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   cfg.AdminCookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearAdminSessionCookie(c *gin.Context, cfg config.Config) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     AdminSessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.AdminCookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h AdminHandler) ListLittleMonks(c *gin.Context) {
	var items []models.LittleMonk
	if err := h.DB.Order("id asc").Find(&items).Error; err != nil {
		internalServerError(c, "could not list little monks")
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h AdminHandler) CreateLittleMonk(c *gin.Context) {
	var input models.LittleMonk
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}

	if input.Status == "" {
		input.Status = "active"
	}

	if err := h.DB.Create(&input).Error; err != nil {
		internalServerError(c, "could not create little monk")
		return
	}

	c.JSON(http.StatusCreated, input)
}

func (h AdminHandler) ListMachines(c *gin.Context) {
	var items []models.Machine
	if err := h.DB.Order("id asc").Find(&items).Error; err != nil {
		internalServerError(c, "could not list machines")
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h AdminHandler) AssignMachine(c *gin.Context) {
	machineID := c.Param("machineId")

	var payload struct {
		LittleMonkID uint `json:"little_monk_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		badRequest(c, err)
		return
	}

	if err := h.DB.Model(&models.Machine{}).Where("machine_id = ?", machineID).Updates(map[string]any{
		"little_monk_id": payload.LittleMonkID,
		"status":         "active",
	}).Error; err != nil {
		internalServerError(c, "could not assign machine")
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h AdminHandler) ListRules(c *gin.Context) {
	var items []models.PolicyRule
	query := h.DB.Order("id asc")

	if value := c.Query("little_monk_id"); value != "" {
		if id, err := strconv.Atoi(value); err == nil {
			query = query.Where("little_monk_id = ?", id)
		}
	} else {
		query = query.Where("little_monk_id IS NULL")
	}

	if err := query.Find(&items).Error; err != nil {
		internalServerError(c, "could not list rules")
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h AdminHandler) GetPolicyConfig(c *gin.Context) {
	config, err := getPolicyConfig(h.DB)
	if err != nil {
		internalServerError(c, "could not load policy config")
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h AdminHandler) UpdatePolicyConfig(c *gin.Context) {
	var input struct {
		DefaultAction string `json:"default_action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}

	if !isValidDefaultAction(input.DefaultAction) {
		badRequest(c, errors.New("invalid default action"))
		return
	}

	config, err := getPolicyConfig(h.DB)
	if err != nil {
		internalServerError(c, "could not load policy config")
		return
	}

	config.DefaultAction = input.DefaultAction
	if err := h.DB.Save(&config).Error; err != nil {
		internalServerError(c, "could not update policy config")
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h AdminHandler) CreateRule(c *gin.Context) {
	var input struct {
		Action       string `json:"action" binding:"required"`
		PatternType  string `json:"pattern_type" binding:"required"`
		PatternValue string `json:"pattern_value" binding:"required"`
		Enabled      *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}

	patternValue := strings.TrimSpace(input.PatternValue)
	if !isValidRuleAction(input.Action) || !isValidPatternType(input.PatternType) || patternValue == "" {
		badRequest(c, errInvalidRule)
		return
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	rule := models.PolicyRule{
		Action:       input.Action,
		PatternType:  input.PatternType,
		PatternValue: patternValue,
		Enabled:      enabled,
	}

	if err := h.DB.Create(&rule).Error; err != nil {
		internalServerError(c, "could not create rule")
		return
	}

	c.JSON(http.StatusCreated, rule)
}

func (h AdminHandler) UpdateRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("ruleId"))
	if err != nil || id <= 0 {
		badRequest(c, errors.New("invalid rule id"))
		return
	}

	var input struct {
		Action       *string `json:"action"`
		PatternType  *string `json:"pattern_type"`
		PatternValue *string `json:"pattern_value"`
		Enabled      *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}

	var rule models.PolicyRule
	if err := h.DB.Where("id = ? AND little_monk_id IS NULL", id).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			notFound(c, "rule not found")
			return
		}
		internalServerError(c, "could not load rule")
		return
	}

	if input.Action != nil {
		if !isValidRuleAction(*input.Action) {
			badRequest(c, errInvalidRule)
			return
		}
		rule.Action = *input.Action
	}

	if input.PatternType != nil {
		if !isValidPatternType(*input.PatternType) {
			badRequest(c, errInvalidRule)
			return
		}
		rule.PatternType = *input.PatternType
	}

	if input.PatternValue != nil {
		patternValue := strings.TrimSpace(*input.PatternValue)
		if patternValue == "" {
			badRequest(c, errInvalidRule)
			return
		}
		rule.PatternValue = patternValue
	}

	if input.Enabled != nil {
		rule.Enabled = *input.Enabled
	}

	if len(rule.PatternValue) > 255 || rule.PatternType == "title_contains_any" {
		if err := h.DB.Exec("ALTER TABLE policy_rules ALTER COLUMN pattern_value TYPE text").Error; err != nil {
			internalServerErrorWithCause(c, "could not migrate rule storage", err)
			return
		}
	}

	if err := h.DB.Save(&rule).Error; err != nil {
		internalServerErrorWithCause(c, "could not update rule", err)
		return
	}

	c.JSON(http.StatusOK, rule)
}

func (h AdminHandler) DeleteRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("ruleId"))
	if err != nil || id <= 0 {
		badRequest(c, errors.New("invalid rule id"))
		return
	}

	result := h.DB.Where("id = ? AND little_monk_id IS NULL", id).Delete(&models.PolicyRule{})
	if result.Error != nil {
		internalServerError(c, "could not delete rule")
		return
	}

	if result.RowsAffected == 0 {
		notFound(c, "rule not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h AdminHandler) ListLogs(c *gin.Context) {
	var items []models.VisitLog
	query := buildVisitLogQuery(h.DB, c)

	if action := c.Query("action"); action != "" && !isValidLogAction(action) {
		badRequest(c, errors.New("invalid log action"))
		return
	}

	limit := parsePositiveInt(c.Query("limit"), 50)
	if limit > 200 {
		limit = 200
	}
	offset := parsePositiveInt(c.Query("offset"), 0)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		internalServerError(c, "could not count logs")
		return
	}

	if err := query.Order("visited_at desc").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		internalServerError(c, "could not list logs")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h AdminHandler) ListLogGroups(c *gin.Context) {
	if action := c.Query("action"); action != "" && !isValidLogAction(action) {
		badRequest(c, errors.New("invalid log action"))
		return
	}

	var logs []models.VisitLog
	query := buildVisitLogQueryWithoutDomain(h.DB, c)
	if err := query.Order("machine_id asc, visited_at asc, id asc").Find(&logs).Error; err != nil {
		internalServerError(c, "could not list logs")
		return
	}

	logs = filterVisitLogsBySearch(logs, visitLogSearchQuery(c))
	groups := groupAdjacentVisitLogs(logs)
	limit := parsePositiveInt(c.Query("limit"), 50)
	if limit > 200 {
		limit = 200
	}
	offset := parsePositiveInt(c.Query("offset"), 0)
	total := len(groups)

	if offset > total {
		offset = total
	}

	end := offset + limit
	if end > total {
		end = total
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  groups[offset:end],
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func buildVisitLogQuery(database *gorm.DB, c *gin.Context) *gorm.DB {
	query := buildVisitLogQueryWithoutDomain(database, c)

	if search := visitLogSearchQuery(c); search != "" {
		keyword := "%" + strings.TrimSpace(search) + "%"
		query = query.Where("domain ILIKE ? OR title ILIKE ?", keyword, keyword)
	}

	return query
}

func buildVisitLogQueryWithoutDomain(database *gorm.DB, c *gin.Context) *gorm.DB {
	query := database.Model(&models.VisitLog{})

	if machineID := c.Query("machine_id"); machineID != "" {
		query = query.Where("machine_id = ?", machineID)
	}

	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}

	if from := c.Query("from"); from != "" {
		query = query.Where("visited_at >= ?", from)
	}

	if to := c.Query("to"); to != "" {
		query = query.Where("visited_at <= ?", to)
	}

	return query
}

func visitLogSearchQuery(c *gin.Context) string {
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		return search
	}

	return c.Query("domain")
}

func filterVisitLogsBySearch(logs []models.VisitLog, search string) []models.VisitLog {
	normalizedSearch := normalizeSearchText(search)
	if normalizedSearch == "" {
		return logs
	}

	filtered := make([]models.VisitLog, 0, len(logs))
	for _, log := range logs {
		if strings.Contains(normalizeSearchText(log.Domain), normalizedSearch) ||
			strings.Contains(normalizeSearchText(log.Title), normalizedSearch) {
			filtered = append(filtered, log)
		}
	}

	return filtered
}

func groupAdjacentVisitLogs(logs []models.VisitLog) []VisitLogGroup {
	groups := make([]VisitLogGroup, 0, len(logs))

	for _, log := range logs {
		normalizedURL := normalizeVisitURL(log.URL)
		if normalizedURL == "" {
			normalizedURL = log.URL
		}

		if len(groups) > 0 {
			lastIndex := len(groups) - 1
			last := &groups[lastIndex]
			if canMergeVisitLogGroup(*last, log, normalizedURL) {
				last.VisitCount++
				last.LastVisitedAt = log.VisitedAt
				last.ProfileInstanceID = log.ProfileInstanceID
				last.Title = pickGroupTitle(last.Title, log.Title)
				continue
			}
		}

		groups = append(groups, VisitLogGroup{
			GroupID:           strconv.FormatUint(uint64(log.ID), 10),
			MachineID:         log.MachineID,
			ProfileInstanceID: log.ProfileInstanceID,
			Domain:            log.Domain,
			URL:               normalizedURL,
			Title:             log.Title,
			Action:            log.Action,
			VisitCount:        1,
			FirstVisitedAt:    log.VisitedAt,
			LastVisitedAt:     log.VisitedAt,
		})
	}

	for i, j := 0, len(groups)-1; i < j; i, j = i+1, j-1 {
		groups[i], groups[j] = groups[j], groups[i]
	}

	return groups
}

func canMergeVisitLogGroup(group VisitLogGroup, log models.VisitLog, normalizedURL string) bool {
	return group.MachineID == log.MachineID &&
		group.URL == normalizedURL &&
		group.Action == log.Action &&
		sameLocalDay(group.LastVisitedAt, log.VisitedAt)
}

func normalizeVisitURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimSpace(rawURL)
	}

	parsed.Fragment = ""
	query := parsed.Query()
	for key := range query {
		lowerKey := strings.ToLower(key)
		if strings.HasPrefix(lowerKey, "utm_") || lowerKey == "fbclid" || lowerKey == "gclid" || lowerKey == "mc_cid" || lowerKey == "mc_eid" {
			query.Del(key)
		}
	}
	parsed.RawQuery = query.Encode()

	if parsed.Path != "/" {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}

	return parsed.String()
}

func sameLocalDay(first time.Time, second time.Time) bool {
	firstYear, firstMonth, firstDay := first.Local().Date()
	secondYear, secondMonth, secondDay := second.Local().Date()
	return firstYear == secondYear && firstMonth == secondMonth && firstDay == secondDay
}

func pickGroupTitle(current string, next string) string {
	if strings.TrimSpace(next) != "" {
		return next
	}
	return current
}

func getPolicyConfig(database *gorm.DB) (models.PolicyConfig, error) {
	config := models.PolicyConfig{ID: 1}
	err := database.Where("id = ?", 1).Attrs(models.PolicyConfig{
		DefaultAction: "allow",
	}).FirstOrCreate(&config).Error
	return config, err
}

func isValidDefaultAction(value string) bool {
	return value == "allow" || value == "block"
}

func normalizeSearchText(value string) string {
	var builder strings.Builder
	lastWasDash := false

	for _, current := range strings.ToLower(strings.TrimSpace(value)) {
		normalized := removeVietnameseAccent(current)
		if (normalized >= 'a' && normalized <= 'z') || (normalized >= '0' && normalized <= '9') {
			builder.WriteRune(normalized)
			lastWasDash = false
			continue
		}

		if normalized == '-' || normalized == '.' {
			builder.WriteRune(normalized)
			lastWasDash = normalized == '-'
			continue
		}

		if !lastWasDash {
			builder.WriteRune('-')
			lastWasDash = true
		}
	}

	return strings.Trim(builder.String(), "-")
}

func removeVietnameseAccent(value rune) rune {
	if value == 'đ' {
		return 'd'
	}

	for _, group := range vietnameseAccentGroups {
		if strings.ContainsRune(group.from, value) {
			return group.to
		}
	}

	if value > unicode.MaxASCII {
		return '-'
	}

	return value
}

var vietnameseAccentGroups = []struct {
	from string
	to   rune
}{
	{from: "àáãảạăằắẳẵặâầấẩẫậä", to: 'a'},
	{from: "èéẻẽẹêềếểễệë", to: 'e'},
	{from: "ìíỉĩịïî", to: 'i'},
	{from: "òóỏõọôồốổỗộơờớởỡợö", to: 'o'},
	{from: "ùúủũụưừứửữựüû", to: 'u'},
	{from: "ýỳỹỵỷ", to: 'y'},
	{from: "ñ", to: 'n'},
	{from: "ç", to: 'c'},
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}

	return parsed
}
