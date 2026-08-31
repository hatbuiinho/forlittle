package handlers

import (
	"errors"
	"net/http"
	"time"

	"forlittle/server/internal/models"
	"forlittle/server/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AgentHandler struct {
	DB *gorm.DB
}

type RegisterRequest struct {
	MachineID         string `json:"machine_id" binding:"required"`
	DisplayName       string `json:"display_name"`
	ProfileInstanceID string `json:"profile_instance_id" binding:"required"`
	ExtensionVersion  string `json:"extension_version"`
	Platform          string `json:"platform"`
	Browser           string `json:"browser"`
	BrowserVersion    string `json:"browser_version"`
}

type HeartbeatRequest struct {
	ProfileInstanceID string    `json:"profile_instance_id" binding:"required"`
	SentAt            time.Time `json:"sent_at" binding:"required"`
}

type LogEvent struct {
	ProfileInstanceID string    `json:"profile_instance_id" binding:"required"`
	TabID             int       `json:"tab_id"`
	URL               string    `json:"url" binding:"required"`
	Domain            string    `json:"domain" binding:"required"`
	Title             string    `json:"title" binding:"required"`
	VisitedAt         time.Time `json:"visited_at" binding:"required"`
	Action            string    `json:"action" binding:"required"`
}

type LogBatchRequest struct {
	Events []LogEvent `json:"events" binding:"required"`
}

var errInvalidRule = errors.New("invalid rule action or pattern type")

func (h AgentHandler) Register(c *gin.Context) {
	var input RegisterRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}

	now := time.Now().UTC()
	var token string
	displayName := input.DisplayName
	if displayName == "" {
		displayName = input.MachineID
	}
	token, err := services.NewOpaqueToken()
	if err != nil {
		internalServerError(c, "could not create device token")
		return
	}

	client := models.ExtensionClient{}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var machine models.Machine
		err := tx.Where("machine_id = ?", input.MachineID).First(&machine).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			machine = models.Machine{
				MachineID:       input.MachineID,
				DisplayName:     displayName,
				Status:          "pending",
				DeviceTokenHash: "managed-by-extension-client",
				LastSeenAt:      &now,
			}
			if err := tx.Create(&machine).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		client = models.ExtensionClient{MachineID: input.MachineID}
		if err := tx.Where("machine_id = ?", input.MachineID).Assign(models.ExtensionClient{
			DisplayName: displayName,
			Status:      "active",
			TokenHash:   services.HashToken(token),
			LastSeenAt:  &now,
		}).FirstOrCreate(&client).Error; err != nil {
			return err
		}

		profile := models.BrowserProfile{MachineID: input.MachineID, ProfileInstanceID: input.ProfileInstanceID, FirstSeenAt: now, LastSeenAt: now}
		return tx.Where(models.BrowserProfile{ProfileInstanceID: input.ProfileInstanceID}).Assign(models.BrowserProfile{
			MachineID: input.MachineID, LastSeenAt: now,
		}).FirstOrCreate(&profile).Error
	}); err != nil {
		internalServerErrorWithCause(c, "could not register Chrome Extension", err)
		return
	}

	response := gin.H{
		"machine_status": client.Status,
		"device_token":   token,
	}

	c.JSON(http.StatusOK, response)
}

func (h AgentHandler) Heartbeat(c *gin.Context) {
	var input HeartbeatRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}

	machineID := c.GetString("machine_id")
	now := time.Now().UTC()

	if err := h.DB.Model(&models.ExtensionClient{}).Where("machine_id = ?", machineID).Update("last_seen_at", now).Error; err != nil {
		internalServerErrorWithCause(c, "could not update Chrome Extension heartbeat", err)
		return
	}

	if err := h.DB.Model(&models.BrowserProfile{}).Where("profile_instance_id = ?", input.ProfileInstanceID).Update("last_seen_at", now).Error; err != nil {
		internalServerError(c, "could not update profile heartbeat")
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h AgentHandler) Policy(c *gin.Context) {
	machineID := c.GetString("machine_id")

	var machine models.Machine
	if err := h.DB.Where("machine_id = ?", machineID).First(&machine).Error; err != nil {
		unauthorized(c, "machine not found")
		return
	}

	rules := []models.PolicyRule{}
	if err := h.DB.Where("little_monk_id IS NULL AND enabled = ?", true).Order("id asc").Find(&rules).Error; err != nil {
		internalServerError(c, "could not load policy")
		return
	}

	config, err := getPolicyConfig(h.DB)
	if err != nil {
		internalServerError(c, "could not load policy config")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"policy_version": time.Now().UTC().Unix(),
		"default_action": config.DefaultAction,
		"rules":          rules,
	})
}

func (h AgentHandler) LogsBatch(c *gin.Context) {
	var input LogBatchRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}

	if len(input.Events) == 0 {
		badRequest(c, errors.New("events must not be empty"))
		return
	}

	machineID := c.GetString("machine_id")
	logs := make([]models.VisitLog, 0, len(input.Events))

	for _, event := range input.Events {
		if !isValidLogAction(event.Action) {
			badRequest(c, errors.New("invalid log action"))
			return
		}

		logs = append(logs, models.VisitLog{
			MachineID:         machineID,
			ProfileInstanceID: event.ProfileInstanceID,
			TabID:             event.TabID,
			URL:               event.URL,
			Domain:            event.Domain,
			Title:             event.Title,
			VisitedAt:         event.VisitedAt,
			Action:            event.Action,
		})
	}

	if len(logs) > 0 {
		if err := h.DB.CreateInBatches(logs, 100).Error; err != nil {
			internalServerError(c, "could not store logs")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"accepted": len(logs)})
}

func isValidRuleAction(value string) bool {
	return value == "allow" || value == "block"
}

func isValidPatternType(value string) bool {
	return value == "domain_exact" || value == "domain_suffix" || value == "domain_contains" || value == "title_contains_any"
}

func isValidLogAction(value string) bool {
	switch value {
	case "allowed", "allowed_whitelist", "blocked_blacklist", "blocked_default", "blocked_title":
		return true
	default:
		return false
	}
}
