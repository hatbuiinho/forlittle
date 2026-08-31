package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"forlittle/server/internal/config"
	"forlittle/server/internal/models"
	"forlittle/server/internal/services"
	"forlittle/server/internal/timecontrol"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

const defaultTimezone = "Asia/Ho_Chi_Minh"

type TimeControlHandler struct {
	DB     *gorm.DB
	Cfg    config.Config
	Broker *timecontrol.CommandBroker
}

type timePolicyInput struct {
	Timezone string                       `json:"timezone"`
	Enabled  *bool                        `json:"enabled"`
	Schedule []timecontrol.ScheduleWindow `json:"schedule"`
}

type sharedTimePolicyInput struct {
	Name string `json:"name" binding:"required"`
	timePolicyInput
}

type machinePolicyAssignmentInput struct {
	SharedPolicyID *uint `json:"shared_policy_id"`
}

type effectiveTimePolicyResponse struct {
	Policy   models.TimePolicy           `json:"policy"`
	Schedule []models.TimeScheduleWindow `json:"schedule"`
	Source   string                      `json:"source"`
}

type deviceEnrollmentRequest struct {
	MachineID             string `json:"machine_id" binding:"required"`
	DisplayName           string `json:"display_name"`
	LittleMonkCode        string `json:"little_monk_code" binding:"required"`
	LittleMonkDisplayName string `json:"little_monk_display_name"`
	EnrollmentKey         string `json:"enrollment_key" binding:"required"`
}

var errMachineAssignedToAnotherLittleMonk = errors.New("machine is already assigned to another little monk")

type deviceHeartbeatRequest struct {
	EffectiveState       string     `json:"effective_state" binding:"required"`
	StateReason          string     `json:"state_reason" binding:"required"`
	NextAllowedAt        *time.Time `json:"next_allowed_at"`
	ExtendedUntil        *time.Time `json:"extended_until"`
	AgentHealthy         bool       `json:"agent_healthy"`
	AppliedPolicyVersion int        `json:"applied_policy_version"`
}

type usageBucketInput struct {
	WindowsUser   string    `json:"windows_user" binding:"required"`
	Application   string    `json:"application" binding:"required"`
	UsageDate     time.Time `json:"usage_date" binding:"required"`
	ActiveSeconds int64     `json:"active_seconds"`
	IdleSeconds   int64     `json:"idle_seconds"`
}

type usageRequest struct {
	Buckets []usageBucketInput `json:"buckets" binding:"required"`
}

type commandInput struct {
	Type            string         `json:"type" binding:"required"`
	DurationSeconds int64          `json:"duration_seconds"`
	Reason          string         `json:"reason"`
	Payload         map[string]any `json:"payload"`
}

type commandAckInput struct {
	Status string `json:"status" binding:"required"`
	Error  string `json:"error"`
}

func (h TimeControlHandler) Enroll(c *gin.Context) {
	var input deviceEnrollmentRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}
	if h.Cfg.DeviceEnrollmentKey == "" || input.EnrollmentKey != h.Cfg.DeviceEnrollmentKey {
		unauthorized(c, "invalid enrollment key")
		return
	}

	machineID := strings.TrimSpace(input.MachineID)
	if machineID == "" {
		badRequest(c, errors.New("machine_id is required"))
		return
	}
	littleMonkCode := strings.TrimSpace(input.LittleMonkCode)
	if littleMonkCode == "" {
		badRequest(c, errors.New("little_monk_code is required"))
		return
	}
	now := time.Now().UTC()
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = machineID
	}
	littleMonkDisplayName := strings.TrimSpace(input.LittleMonkDisplayName)
	if littleMonkDisplayName == "" {
		littleMonkDisplayName = displayName
	}

	token, err := services.NewOpaqueToken()
	if err != nil {
		internalServerError(c, "could not create device token")
		return
	}

	var machine models.Machine
	var littleMonk models.LittleMonk
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("code = ?", littleMonkCode).First(&littleMonk).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			littleMonk = models.LittleMonk{Code: littleMonkCode, DisplayName: littleMonkDisplayName, Status: "active"}
			if err := tx.Create(&littleMonk).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		err = tx.Where("machine_id = ?", machineID).First(&machine).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			machine = models.Machine{
				MachineID:       machineID,
				DisplayName:     displayName,
				Status:          "active",
				LittleMonkID:    &littleMonk.ID,
				DeviceTokenHash: "managed-by-device-client",
				LastSeenAt:      &now,
			}
			if err := tx.Create(&machine).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if machine.LittleMonkID != nil && *machine.LittleMonkID != littleMonk.ID {
				return errMachineAssignedToAnotherLittleMonk
			}
			if err := tx.Model(&machine).Updates(map[string]any{
				"display_name":      displayName,
				"little_monk_id":    littleMonk.ID,
				"status":            "active",
				"device_token_hash": "managed-by-device-client",
				"last_seen_at":      now,
			}).Error; err != nil {
				return err
			}
			machine.DisplayName = displayName
			machine.LittleMonkID = &littleMonk.ID
			machine.Status = "active"
			machine.LastSeenAt = &now
		}

		client := models.DeviceClient{MachineID: machineID, ClientType: "windows_service", TokenHash: services.HashToken(token), LastSeenAt: &now}
		return tx.Where(models.DeviceClient{MachineID: machineID, ClientType: "windows_service"}).Assign(client).FirstOrCreate(&client).Error
	})
	if errors.Is(err, errMachineAssignedToAnotherLittleMonk) {
		c.JSON(http.StatusConflict, gin.H{"error": "machine is already assigned to another little monk"})
		return
	}
	if err != nil {
		internalServerError(c, "could not enroll service")
		return
	}

	c.JSON(http.StatusOK, gin.H{"device_token": token, "machine_status": machine.Status, "little_monk": littleMonk, "server_time": now})
}

func (h TimeControlHandler) GetPolicy(c *gin.Context) {
	machineID := c.GetString("machine_id")
	policy, windows, source, err := h.policyForMachine(machineID)
	if err != nil {
		internalServerError(c, "could not load time policy")
		return
	}
	state := h.loadState(machineID)
	c.JSON(http.StatusOK, gin.H{"policy": policy, "schedule": windows, "source": source, "state": state, "server_time": time.Now().UTC()})
}

func (h TimeControlHandler) GetCommands(c *gin.Context) {
	machineID := c.GetString("machine_id")
	now := time.Now().UTC()
	var commands []models.DeviceCommand
	if err := h.DB.Where("machine_id = ? AND status IN ? AND (expires_at IS NULL OR expires_at > ?)", machineID, []string{"PENDING", "RECEIVED"}, now).Order("created_at asc").Find(&commands).Error; err != nil {
		internalServerError(c, "could not load commands")
		return
	}
	c.JSON(http.StatusOK, gin.H{"commands": commands, "server_time": now})
}

func (h TimeControlHandler) Heartbeat(c *gin.Context) {
	var input deviceHeartbeatRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}
	if !isAccessState(input.EffectiveState) {
		badRequest(c, errors.New("invalid effective_state"))
		return
	}
	machineID := c.GetString("machine_id")
	now := time.Now().UTC()
	state := models.MachineTimeState{MachineID: machineID, EffectiveState: input.EffectiveState, StateReason: strings.TrimSpace(input.StateReason), NextAllowedAt: input.NextAllowedAt, ExtendedUntil: input.ExtendedUntil, AgentHealthy: input.AgentHealthy, AppliedPolicyVersion: input.AppliedPolicyVersion, PolicyAppliedAt: &now, LastReportedAt: &now}
	if state.StateReason == "" {
		state.StateReason = "reported"
	}
	if err := h.DB.Where(models.MachineTimeState{MachineID: machineID}).Assign(state).FirstOrCreate(&state).Error; err != nil {
		internalServerError(c, "could not store service state")
		return
	}
	if err := h.DB.Model(&models.Machine{}).Where("machine_id = ?", machineID).Update("last_seen_at", now).Error; err != nil {
		internalServerError(c, "could not update machine")
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "server_time": now})
}

func (h TimeControlHandler) Usage(c *gin.Context) {
	var input usageRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}
	if len(input.Buckets) == 0 || len(input.Buckets) > 500 {
		badRequest(c, errors.New("buckets must contain 1 to 500 items"))
		return
	}
	machineID := c.GetString("machine_id")
	for _, bucket := range input.Buckets {
		if strings.TrimSpace(bucket.WindowsUser) == "" || strings.TrimSpace(bucket.Application) == "" || bucket.ActiveSeconds < 0 || bucket.IdleSeconds < 0 {
			badRequest(c, errors.New("invalid usage bucket"))
			return
		}
		date := time.Date(bucket.UsageDate.Year(), bucket.UsageDate.Month(), bucket.UsageDate.Day(), 0, 0, 0, 0, time.UTC)
		record := models.AppUsage{MachineID: machineID, WindowsUser: strings.TrimSpace(bucket.WindowsUser), Application: strings.TrimSpace(bucket.Application), UsageDate: date}
		if err := h.DB.Where(models.AppUsage{MachineID: record.MachineID, WindowsUser: record.WindowsUser, Application: record.Application, UsageDate: date}).Assign(map[string]any{"active_seconds": bucket.ActiveSeconds, "idle_seconds": bucket.IdleSeconds}).FirstOrCreate(&record).Error; err != nil {
			internalServerError(c, "could not store usage")
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"accepted": len(input.Buckets)})
}

func (h TimeControlHandler) AckCommand(c *gin.Context) {
	var input commandAckInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}
	if !isCommandStatus(input.Status) {
		badRequest(c, errors.New("invalid command acknowledgement status"))
		return
	}
	machineID := c.GetString("machine_id")
	commandID := c.Param("commandId")
	updates := map[string]any{"status": input.Status, "error": strings.TrimSpace(input.Error)}
	if input.Status == "APPLIED" || input.Status == "FAILED" || input.Status == "IGNORED_DUPLICATE" {
		now := time.Now().UTC()
		updates["applied_at"] = now
	}
	result := h.DB.Model(&models.DeviceCommand{}).Where("command_id = ? AND machine_id = ?", commandID, machineID).Updates(updates)
	if result.Error != nil {
		internalServerError(c, "could not acknowledge command")
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h TimeControlHandler) GetPolicyAdmin(c *gin.Context) {
	policy, windows, err := h.policyByLittleMonk(c.Param("littleMonkId"))
	if err != nil {
		internalServerError(c, "could not load time policy")
		return
	}
	c.JSON(http.StatusOK, gin.H{"policy": policy, "schedule": windows})
}

// ListManagedMachinesAdmin returns only computers enrolled by the Windows Time Control service.
// Browser-extension machines and Little Monk records are deliberately excluded.
func (h TimeControlHandler) ListManagedMachinesAdmin(c *gin.Context) {
	var machines []models.Machine
	err := h.DB.Joins("JOIN device_clients ON device_clients.machine_id = machines.machine_id").
		Where("device_clients.client_type = ? AND device_clients.revoked_at IS NULL", "windows_service").
		Order("machines.display_name asc, machines.machine_id asc").Find(&machines).Error
	if err != nil {
		internalServerError(c, "could not load Windows Time Control machines")
		return
	}
	now := time.Now().UTC()
	items := make([]serviceMachineResponse, 0, len(machines))
	for _, machine := range machines {
		items = append(items, serviceMachineResponseFor(machine, now))
	}
	c.JSON(http.StatusOK, items)
}

func (h TimeControlHandler) ListSharedPoliciesAdmin(c *gin.Context) {
	var policies []models.TimePolicy
	if err := h.DB.Where("scope = ?", "shared").Order("name asc, id asc").Find(&policies).Error; err != nil {
		internalServerError(c, "could not load shared schedules")
		return
	}
	items := make([]effectiveTimePolicyResponse, 0, len(policies))
	for _, policy := range policies {
		windows, err := h.scheduleForPolicy(policy.ID)
		if err != nil {
			internalServerError(c, "could not load shared schedule windows")
			return
		}
		items = append(items, effectiveTimePolicyResponse{Policy: policy, Schedule: windows, Source: "shared"})
	}
	c.JSON(http.StatusOK, items)
}

func (h TimeControlHandler) CreateSharedPolicyAdmin(c *gin.Context) {
	var input sharedTimePolicyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}
	timezone, enabled, err := validateTimePolicyInput(input.timePolicyInput)
	if err != nil {
		badRequest(c, err)
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		badRequest(c, errors.New("schedule name is required"))
		return
	}
	policy := models.TimePolicy{Name: name, Scope: "shared", Timezone: timezone, Enabled: enabled, Version: 1}
	if err := h.DB.Transaction(func(tx *gorm.DB) error { return h.replacePolicySchedule(tx, &policy, input.Schedule, true) }); err != nil {
		internalServerError(c, "could not create shared schedule")
		return
	}
	windows, _ := h.scheduleForPolicy(policy.ID)
	c.JSON(http.StatusCreated, effectiveTimePolicyResponse{Policy: policy, Schedule: windows, Source: "shared"})
}

func (h TimeControlHandler) UpdateSharedPolicyAdmin(c *gin.Context) {
	policyID, err := parsePositiveID(c.Param("policyId"))
	if err != nil {
		badRequest(c, err)
		return
	}
	var input sharedTimePolicyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}
	timezone, enabled, err := validateTimePolicyInput(input.timePolicyInput)
	if err != nil {
		badRequest(c, err)
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		badRequest(c, errors.New("schedule name is required"))
		return
	}
	var policy models.TimePolicy
	if err := h.DB.Where("id = ? AND scope = ?", policyID, "shared").First(&policy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "shared schedule not found"})
			return
		}
		internalServerError(c, "could not load shared schedule")
		return
	}
	policy.Name, policy.Timezone, policy.Enabled, policy.Version = name, timezone, enabled, policy.Version+1
	if err := h.DB.Transaction(func(tx *gorm.DB) error { return h.replacePolicySchedule(tx, &policy, input.Schedule, false) }); err != nil {
		internalServerError(c, "could not save shared schedule")
		return
	}
	h.queuePolicyRefreshForSharedPolicy(policy.ID)
	windows, _ := h.scheduleForPolicy(policy.ID)
	c.JSON(http.StatusOK, effectiveTimePolicyResponse{Policy: policy, Schedule: windows, Source: "shared"})
}

func (h TimeControlHandler) DeleteSharedPolicyAdmin(c *gin.Context) {
	policyID, err := parsePositiveID(c.Param("policyId"))
	if err != nil {
		badRequest(c, err)
		return
	}
	var affected []models.MachineTimePolicyAssignment
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var policy models.TimePolicy
		if err := tx.Where("id = ? AND scope = ?", policyID, "shared").First(&policy).Error; err != nil {
			return err
		}
		if err := tx.Where("shared_policy_id = ?", policyID).Find(&affected).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.MachineTimePolicyAssignment{}).Where("shared_policy_id = ?", policyID).Update("shared_policy_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Where("time_policy_id = ?", policyID).Delete(&models.TimeScheduleWindow{}).Error; err != nil {
			return err
		}
		return tx.Delete(&policy).Error
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "shared schedule not found"})
			return
		}
		internalServerError(c, "could not delete shared schedule")
		return
	}
	for _, assignment := range affected {
		h.queuePolicyRefreshForMachine(assignment.MachineID)
	}
	c.Status(http.StatusNoContent)
}

func (h TimeControlHandler) GetMachinePolicyAdmin(c *gin.Context) {
	policy, windows, source, err := h.policyForMachine(c.Param("machineId"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "machine not found"})
			return
		}
		internalServerError(c, "could not load machine schedule")
		return
	}
	var assignment models.MachineTimePolicyAssignment
	_ = h.DB.Where("machine_id = ?", c.Param("machineId")).First(&assignment).Error
	c.JSON(http.StatusOK, gin.H{"policy": policy, "schedule": windows, "source": source, "assignment": assignment})
}

func (h TimeControlHandler) PutMachineSharedPolicyAdmin(c *gin.Context) {
	machineID := strings.TrimSpace(c.Param("machineId"))
	var input machinePolicyAssignmentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}
	if err := h.ensureMachineAndSharedPolicy(machineID, input.SharedPolicyID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "machine or shared schedule not found"})
			return
		}
		internalServerError(c, "could not validate assignment")
		return
	}
	assignment := models.MachineTimePolicyAssignment{MachineID: machineID}
	if err := h.DB.Where(models.MachineTimePolicyAssignment{MachineID: machineID}).Assign(map[string]any{"shared_policy_id": input.SharedPolicyID}).FirstOrCreate(&assignment).Error; err != nil {
		internalServerError(c, "could not assign shared schedule")
		return
	}
	h.queuePolicyRefreshForMachine(machineID)
	h.GetMachinePolicyAdmin(c)
}

func (h TimeControlHandler) PutMachineOverridePolicyAdmin(c *gin.Context) {
	machineID := strings.TrimSpace(c.Param("machineId"))
	var input timePolicyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}
	timezone, enabled, err := validateTimePolicyInput(input)
	if err != nil {
		badRequest(c, err)
		return
	}
	var assignment models.MachineTimePolicyAssignment
	if err := h.DB.Where("machine_id = ?", machineID).First(&assignment).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		internalServerError(c, "could not load machine schedule")
		return
	}
	if err := h.ensureMachineAndSharedPolicy(machineID, nil); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "machine not found"})
			return
		}
		internalServerError(c, "could not load machine")
		return
	}
	var policy models.TimePolicy
	if assignment.OverridePolicyID != nil {
		if err := h.DB.Where("id = ? AND scope = ?", *assignment.OverridePolicyID, "machine_override").First(&policy).Error; err != nil {
			internalServerError(c, "could not load override schedule")
			return
		}
		policy.Timezone, policy.Enabled, policy.Version = timezone, enabled, policy.Version+1
	} else {
		policy = models.TimePolicy{Name: "Lịch riêng: " + machineID, Scope: "machine_override", Timezone: timezone, Enabled: enabled, Version: 1}
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := h.replacePolicySchedule(tx, &policy, input.Schedule, policy.ID == 0); err != nil {
			return err
		}
		return tx.Where(models.MachineTimePolicyAssignment{MachineID: machineID}).Assign(map[string]any{"override_policy_id": policy.ID}).FirstOrCreate(&models.MachineTimePolicyAssignment{MachineID: machineID}).Error
	}); err != nil {
		internalServerError(c, "could not save machine override")
		return
	}
	h.queuePolicyRefreshForMachine(machineID)
	h.GetMachinePolicyAdmin(c)
}

func (h TimeControlHandler) DeleteMachineOverridePolicyAdmin(c *gin.Context) {
	machineID := strings.TrimSpace(c.Param("machineId"))
	var assignment models.MachineTimePolicyAssignment
	if err := h.DB.Where("machine_id = ?", machineID).First(&assignment).Error; err != nil || assignment.OverridePolicyID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "machine override not found"})
		return
	}
	overrideID := *assignment.OverridePolicyID
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.MachineTimePolicyAssignment{}).Where("machine_id = ?", machineID).Update("override_policy_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Where("time_policy_id = ?", overrideID).Delete(&models.TimeScheduleWindow{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.TimePolicy{}, overrideID).Error
	}); err != nil {
		internalServerError(c, "could not remove machine override")
		return
	}
	h.queuePolicyRefreshForMachine(machineID)
	c.Status(http.StatusNoContent)
}

func (h TimeControlHandler) PutPolicyAdmin(c *gin.Context) {
	var input timePolicyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}
	if err := timecontrol.ValidateSchedule(input.Schedule); err != nil {
		badRequest(c, err)
		return
	}
	littleMonkID, err := parsePositiveID(c.Param("littleMonkId"))
	if err != nil {
		badRequest(c, err)
		return
	}
	var littleMonk models.LittleMonk
	if err := h.DB.First(&littleMonk, littleMonkID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "little monk not found"})
			return
		}
		internalServerError(c, "could not load little monk")
		return
	}
	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = defaultTimezone
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		badRequest(c, errors.New("invalid timezone"))
		return
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		policy := models.TimePolicy{LittleMonkID: &littleMonkID}
		err := tx.Where("little_monk_id = ?", littleMonkID).First(&policy).Error
		if err == gorm.ErrRecordNotFound {
			policy = models.TimePolicy{LittleMonkID: &littleMonkID, Timezone: timezone, Version: 1, Enabled: enabled}
			if err := tx.Create(&policy).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if err := tx.Model(&policy).Updates(map[string]any{"timezone": timezone, "enabled": enabled, "version": policy.Version + 1}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("time_policy_id = ?", policy.ID).Delete(&models.TimeScheduleWindow{}).Error; err != nil {
			return err
		}
		for _, window := range input.Schedule {
			if err := tx.Create(&models.TimeScheduleWindow{TimePolicyID: policy.ID, DayOfWeek: window.DayOfWeek, StartMinutes: window.StartMinutes, EndMinutes: window.EndMinutes}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		internalServerError(c, "could not save time policy")
		return
	}
	h.queuePolicyRefresh(littleMonkID)
	h.GetPolicyAdmin(c)
}

func (h TimeControlHandler) CreateCommandAdmin(c *gin.Context) {
	var input commandInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err)
		return
	}
	commandType := strings.ToUpper(strings.TrimSpace(input.Type))
	if !isCommandType(commandType) {
		badRequest(c, errors.New("invalid command type"))
		return
	}
	if (commandType == "UNBLOCK" || commandType == "EXTRA_TIME") && (input.DurationSeconds < 60 || input.DurationSeconds > 24*60*60) {
		badRequest(c, errors.New("temporary commands require duration_seconds from 60 to 86400"))
		return
	}
	machineID := strings.TrimSpace(c.Param("machineId"))
	var machine models.Machine
	if err := h.DB.Where("machine_id = ?", machineID).First(&machine).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "machine not found"})
			return
		}
		internalServerError(c, "could not load machine")
		return
	}
	payload := input.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	payload["reason"] = strings.TrimSpace(input.Reason)
	if input.DurationSeconds > 0 {
		payload["duration_seconds"] = input.DurationSeconds
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		badRequest(c, errors.New("invalid command payload"))
		return
	}
	var expiresAt *time.Time
	if input.DurationSeconds > 0 {
		value := time.Now().UTC().Add(time.Duration(input.DurationSeconds) * time.Second)
		expiresAt = &value
	}
	command := models.DeviceCommand{CommandID: newCommandID(), MachineID: machineID, Type: commandType, PayloadJSON: string(payloadJSON), Status: "PENDING", ExpiresAt: expiresAt}
	if err := h.DB.Create(&command).Error; err != nil {
		internalServerError(c, "could not create command")
		return
	}
	h.notifyCommand(machineID, command.CommandID)
	c.JSON(http.StatusCreated, command)
}

// SyncMachinePolicyAdmin requests an immediate policy fetch from one machine.
// The machine reports the applied version in its next heartbeat.
func (h TimeControlHandler) SyncMachinePolicyAdmin(c *gin.Context) {
	machineID := strings.TrimSpace(c.Param("machineId"))
	var machine models.Machine
	if err := h.DB.Where("machine_id = ?", machineID).First(&machine).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "machine not found"})
			return
		}
		internalServerError(c, "could not load machine")
		return
	}
	payload, _ := json.Marshal(map[string]string{"reason": "sư_chú_yêu_cầu_đồng_bộ"})
	command := models.DeviceCommand{CommandID: newCommandID(), MachineID: machine.MachineID, Type: "REFRESH_POLICY", PayloadJSON: string(payload), Status: "PENDING"}
	if err := h.DB.Create(&command).Error; err != nil {
		internalServerError(c, "could not queue policy refresh")
		return
	}
	h.notifyCommand(machine.MachineID, command.CommandID)
	c.JSON(http.StatusCreated, command)
}

func (h TimeControlHandler) WebSocket(c *gin.Context) {
	if h.Broker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "command notifications unavailable"})
		return
	}
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	connection, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer connection.Close()

	connection.SetReadLimit(4096)
	_ = connection.SetReadDeadline(time.Now().Add(90 * time.Second))
	connection.SetPongHandler(func(string) error { return connection.SetReadDeadline(time.Now().Add(90 * time.Second)) })
	machineID := c.GetString("machine_id")
	messages, unsubscribe := h.Broker.Subscribe(machineID)
	defer unsubscribe()

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, err := connection.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-readDone:
			return
		case message, ok := <-messages:
			if !ok {
				return
			}
			_ = connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := connection.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ping.C:
			_ = connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h TimeControlHandler) ListUsageAdmin(c *gin.Context) {
	machineID := c.Param("machineId")
	date := time.Now().UTC().Truncate(24 * time.Hour)
	if value := c.Query("date"); value != "" {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			badRequest(c, errors.New("date must use YYYY-MM-DD"))
			return
		}
		date = parsed
	}
	var records []models.AppUsage
	if err := h.DB.Where("machine_id = ? AND usage_date = ?", machineID, date).Order("active_seconds desc, application asc").Find(&records).Error; err != nil {
		internalServerError(c, "could not load usage")
		return
	}
	c.JSON(http.StatusOK, records)
}

func (h TimeControlHandler) GetMachineStateAdmin(c *gin.Context) {
	machineID := strings.TrimSpace(c.Param("machineId"))
	var machine models.Machine
	if err := h.DB.Where("machine_id = ?", machineID).First(&machine).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "machine not found"})
			return
		}
		internalServerError(c, "could not load machine")
		return
	}
	c.JSON(http.StatusOK, h.loadState(machineID))
}

func (h TimeControlHandler) policyForMachine(machineID string) (models.TimePolicy, []models.TimeScheduleWindow, string, error) {
	var machine models.Machine
	if err := h.DB.Where("machine_id = ?", machineID).First(&machine).Error; err != nil {
		return models.TimePolicy{}, nil, "", err
	}
	var assignment models.MachineTimePolicyAssignment
	if err := h.DB.Where("machine_id = ?", machineID).First(&assignment).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.TimePolicy{}, nil, "", err
	}
	if assignment.OverridePolicyID != nil {
		policy, windows, err := h.policyByID(*assignment.OverridePolicyID)
		return policy, windows, "machine_override", err
	}
	if assignment.SharedPolicyID != nil {
		policy, windows, err := h.policyByID(*assignment.SharedPolicyID)
		return policy, windows, "shared", err
	}
	return models.TimePolicy{Timezone: defaultTimezone, Version: 0, Enabled: false}, []models.TimeScheduleWindow{}, "none", nil
}

func (h TimeControlHandler) policyByLittleMonk(rawID string) (models.TimePolicy, []models.TimeScheduleWindow, error) {
	id, err := parsePositiveID(rawID)
	if err != nil {
		return models.TimePolicy{}, nil, err
	}
	var policy models.TimePolicy
	err = h.DB.Where("little_monk_id = ?", id).First(&policy).Error
	if err == gorm.ErrRecordNotFound {
		return models.TimePolicy{LittleMonkID: &id, Timezone: defaultTimezone, Version: 0, Enabled: false}, []models.TimeScheduleWindow{}, nil
	}
	if err != nil {
		return models.TimePolicy{}, nil, err
	}
	windows, err := h.scheduleForPolicy(policy.ID)
	return policy, windows, err
}

func validateTimePolicyInput(input timePolicyInput) (string, bool, error) {
	if err := timecontrol.ValidateSchedule(input.Schedule); err != nil {
		return "", false, err
	}
	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = defaultTimezone
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return "", false, errors.New("invalid timezone")
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	return timezone, enabled, nil
}

func (h TimeControlHandler) replacePolicySchedule(tx *gorm.DB, policy *models.TimePolicy, windows []timecontrol.ScheduleWindow, create bool) error {
	if create {
		if err := tx.Create(policy).Error; err != nil {
			return err
		}
	} else if err := tx.Model(policy).Updates(map[string]any{"name": policy.Name, "timezone": policy.Timezone, "enabled": policy.Enabled, "version": policy.Version}).Error; err != nil {
		return err
	}
	if err := tx.Where("time_policy_id = ?", policy.ID).Delete(&models.TimeScheduleWindow{}).Error; err != nil {
		return err
	}
	for _, window := range windows {
		if err := tx.Create(&models.TimeScheduleWindow{TimePolicyID: policy.ID, DayOfWeek: window.DayOfWeek, StartMinutes: window.StartMinutes, EndMinutes: window.EndMinutes}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (h TimeControlHandler) policyByID(policyID uint) (models.TimePolicy, []models.TimeScheduleWindow, error) {
	var policy models.TimePolicy
	if err := h.DB.First(&policy, policyID).Error; err != nil {
		return models.TimePolicy{}, nil, err
	}
	windows, err := h.scheduleForPolicy(policy.ID)
	return policy, windows, err
}

func (h TimeControlHandler) scheduleForPolicy(policyID uint) ([]models.TimeScheduleWindow, error) {
	var windows []models.TimeScheduleWindow
	err := h.DB.Where("time_policy_id = ?", policyID).Order("day_of_week asc, start_minutes asc").Find(&windows).Error
	return windows, err
}

func (h TimeControlHandler) ensureMachineAndSharedPolicy(machineID string, sharedPolicyID *uint) error {
	var machine models.Machine
	if err := h.DB.Where("machine_id = ?", machineID).First(&machine).Error; err != nil {
		return err
	}
	if sharedPolicyID == nil {
		return nil
	}
	var policy models.TimePolicy
	return h.DB.Where("id = ? AND scope = ?", *sharedPolicyID, "shared").First(&policy).Error
}

func (h TimeControlHandler) loadState(machineID string) models.MachineTimeState {
	var state models.MachineTimeState
	if err := h.DB.Where("machine_id = ?", machineID).First(&state).Error; err != nil {
		return models.MachineTimeState{MachineID: machineID, EffectiveState: timecontrol.StateBlocked, StateReason: "not_reported"}
	}
	return state
}

func (h TimeControlHandler) queuePolicyRefresh(littleMonkID uint) {
	var machines []models.Machine
	if err := h.DB.Where("little_monk_id = ?", littleMonkID).Find(&machines).Error; err != nil {
		return
	}
	for _, machine := range machines {
		h.queuePolicyRefreshForMachine(machine.MachineID)
	}
}

func (h TimeControlHandler) queuePolicyRefreshForSharedPolicy(policyID uint) {
	var assignments []models.MachineTimePolicyAssignment
	if h.DB.Where("shared_policy_id = ?", policyID).Find(&assignments).Error != nil {
		return
	}
	for _, assignment := range assignments {
		h.queuePolicyRefreshForMachine(assignment.MachineID)
	}
}

func (h TimeControlHandler) queuePolicyRefreshForMachine(machineID string) {
	payload, _ := json.Marshal(map[string]any{"reason": "policy_updated"})
	command := models.DeviceCommand{CommandID: newCommandID(), MachineID: machineID, Type: "REFRESH_POLICY", PayloadJSON: string(payload), Status: "PENDING"}
	if h.DB.Create(&command).Error == nil {
		h.notifyCommand(machineID, command.CommandID)
	}
}

func (h TimeControlHandler) notifyCommand(machineID, commandID string) {
	if h.Broker == nil {
		return
	}
	payload, err := json.Marshal(map[string]string{"type": "COMMAND_AVAILABLE", "command_id": commandID})
	if err == nil {
		h.Broker.Notify(machineID, payload)
	}
}

func parsePositiveID(value string) (uint, error) {
	var id uint64
	_, err := fmt.Sscan(value, &id)
	if err != nil || id == 0 {
		return 0, errors.New("invalid identifier")
	}
	return uint(id), nil
}

func newCommandID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "cmd-fallback-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return "cmd-" + hex.EncodeToString(value)
}

func isAccessState(value string) bool {
	return value == timecontrol.StateAllowed || value == timecontrol.StateBlocked || value == timecontrol.StateExtended
}
func isCommandStatus(value string) bool {
	return value == "RECEIVED" || value == "APPLIED" || value == "FAILED" || value == "IGNORED_DUPLICATE"
}
func isCommandType(value string) bool {
	switch value {
	case "BLOCK", "UNBLOCK", "EXTRA_TIME", "RESUME_POLICY", "POLICY_UPDATED", "REFRESH_POLICY", "FORCE_LOCK", "FORCE_LOGOUT":
		return true
	default:
		return false
	}
}
