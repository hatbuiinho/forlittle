package models

import "time"

type Admin struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:100;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;size:255;not null" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	DisplayName  string    `gorm:"size:255;not null" json:"display_name"`
	Role         string    `gorm:"size:50;not null;default:admin" json:"role"`
	Status       string    `gorm:"size:50;not null;default:active" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserSession struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"user_id"`
	TokenHash  string     `gorm:"uniqueIndex;size:255;not null" json:"-"`
	ExpiresAt  time.Time  `gorm:"index;not null" json:"expires_at"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type LittleMonk struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"uniqueIndex;size:100;not null" json:"code"`
	DisplayName string    `gorm:"size:255;not null" json:"display_name"`
	Status      string    `gorm:"size:50;not null;default:active" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Machine struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	MachineID       string     `gorm:"uniqueIndex;size:120;not null" json:"machine_id"`
	DisplayName     string     `gorm:"size:255;not null" json:"display_name"`
	Status          string     `gorm:"size:50;not null;default:pending" json:"status"`
	LittleMonkID    *uint      `json:"little_monk_id"`
	DeviceTokenHash string     `gorm:"size:255;not null" json:"-"`
	LastSeenAt      *time.Time `json:"last_seen_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type BrowserProfile struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	MachineID         string    `gorm:"index;size:120;not null" json:"machine_id"`
	ProfileInstanceID string    `gorm:"uniqueIndex;size:120;not null" json:"profile_instance_id"`
	FirstSeenAt       time.Time `json:"first_seen_at"`
	LastSeenAt        time.Time `json:"last_seen_at"`
}

type PolicyRule struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	LittleMonkID *uint     `gorm:"index" json:"little_monk_id"`
	Action       string    `gorm:"size:20;not null" json:"action"`
	PatternType  string    `gorm:"size:50;not null" json:"pattern_type"`
	PatternValue string    `gorm:"type:text;not null" json:"pattern_value"`
	Enabled      bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
}

type PolicyConfig struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	DefaultAction string    `gorm:"size:20;not null;default:allow" json:"default_action"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type VisitLog struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	MachineID         string    `gorm:"index;size:120;not null" json:"machine_id"`
	ProfileInstanceID string    `gorm:"index;size:120;not null" json:"profile_instance_id"`
	TabID             int       `json:"tab_id"`
	URL               string    `gorm:"type:text;not null" json:"url"`
	Domain            string    `gorm:"size:255;not null" json:"domain"`
	Title             string    `gorm:"type:text;not null" json:"title"`
	VisitedAt         time.Time `gorm:"index;not null" json:"visited_at"`
	Action            string    `gorm:"size:50;not null" json:"action"`
}

// DeviceClient stores a revocable credential for a non-browser device client.
// Its token is stored only as a hash.
type DeviceClient struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	MachineID  string     `gorm:"uniqueIndex:idx_device_client;size:120;not null" json:"machine_id"`
	ClientType string     `gorm:"uniqueIndex:idx_device_client;size:50;not null" json:"client_type"`
	TokenHash  string     `gorm:"uniqueIndex;size:255;not null" json:"-"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type TimePolicy struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	LittleMonkID *uint     `gorm:"uniqueIndex" json:"little_monk_id"`
	Timezone     string    `gorm:"size:100;not null" json:"timezone"`
	Version      int       `gorm:"not null;default:1" json:"version"`
	Enabled      bool      `gorm:"not null;default:false" json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TimeScheduleWindow struct {
	ID           uint `gorm:"primaryKey" json:"id"`
	TimePolicyID uint `gorm:"index;not null" json:"time_policy_id"`
	DayOfWeek    int  `gorm:"not null" json:"day_of_week"`
	StartMinutes int  `gorm:"not null" json:"start_minutes"`
	EndMinutes   int  `gorm:"not null" json:"end_minutes"`
}

type MachineTimeState struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	MachineID      string     `gorm:"uniqueIndex;size:120;not null" json:"machine_id"`
	EffectiveState string     `gorm:"size:20;not null" json:"effective_state"`
	StateReason    string     `gorm:"size:100;not null" json:"state_reason"`
	NextAllowedAt  *time.Time `json:"next_allowed_at"`
	ExtendedUntil  *time.Time `json:"extended_until"`
	AgentHealthy   bool       `gorm:"not null;default:false" json:"agent_healthy"`
	LastReportedAt *time.Time `json:"last_reported_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type DeviceCommand struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	CommandID   string     `gorm:"uniqueIndex;size:80;not null" json:"command_id"`
	MachineID   string     `gorm:"index;size:120;not null" json:"machine_id"`
	Type        string     `gorm:"size:50;not null" json:"type"`
	PayloadJSON string     `gorm:"type:text;not null" json:"payload"`
	Status      string     `gorm:"index;size:30;not null" json:"status"`
	Error       string     `gorm:"type:text" json:"error"`
	ExpiresAt   *time.Time `gorm:"index" json:"expires_at"`
	AppliedAt   *time.Time `json:"applied_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type AppUsage struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	MachineID     string    `gorm:"uniqueIndex:idx_app_usage;size:120;not null" json:"machine_id"`
	WindowsUser   string    `gorm:"uniqueIndex:idx_app_usage;size:255;not null" json:"windows_user"`
	Application   string    `gorm:"uniqueIndex:idx_app_usage;size:255;not null" json:"application"`
	UsageDate     time.Time `gorm:"uniqueIndex:idx_app_usage;not null" json:"usage_date"`
	ActiveSeconds int64     `gorm:"not null;default:0" json:"active_seconds"`
	IdleSeconds   int64     `gorm:"not null;default:0" json:"idle_seconds"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
