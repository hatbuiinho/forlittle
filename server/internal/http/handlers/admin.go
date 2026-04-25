package handlers

import (
	"net/http"
	"strconv"

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
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h AdminHandler) Login(c *gin.Context) {
	var input AdminLoginRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}

	var admin models.Admin
	if err := h.DB.Where("username = ?", input.Username).First(&admin).Error; err != nil {
		unauthorized(c, "invalid credentials")
		return
	}

	if !services.CheckPassword(admin.PasswordHash, input.Password) {
		unauthorized(c, "invalid credentials")
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": h.Cfg.AdminToken})
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
	}

	if err := query.Find(&items).Error; err != nil {
		internalServerError(c, "could not list rules")
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h AdminHandler) CreateRule(c *gin.Context) {
	var input models.PolicyRule
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}

	if !isValidRuleAction(input.Action) || !isValidPatternType(input.PatternType) {
		badRequest(c, errInvalidRule)
		return
	}

	if err := h.DB.Create(&input).Error; err != nil {
		internalServerError(c, "could not create rule")
		return
	}

	c.JSON(http.StatusCreated, input)
}

func (h AdminHandler) ListLogs(c *gin.Context) {
	var items []models.VisitLog
	query := h.DB.Order("visited_at desc").Limit(200)

	if machineID := c.Query("machine_id"); machineID != "" {
		query = query.Where("machine_id = ?", machineID)
	}

	if domain := c.Query("domain"); domain != "" {
		query = query.Where("domain = ?", domain)
	}

	if err := query.Find(&items).Error; err != nil {
		internalServerError(c, "could not list logs")
		return
	}

	c.JSON(http.StatusOK, items)
}
