package timecontrol

import "time"

const (
	StateAllowed  = "ALLOWED"
	StateBlocked  = "BLOCKED"
	StateExtended = "EXTENDED"
)

type ScheduleWindow struct {
	DayOfWeek    int `json:"day_of_week"`
	StartMinutes int `json:"start_minutes"`
	EndMinutes   int `json:"end_minutes"`
}

type Policy struct {
	Version  int              `json:"version"`
	Timezone string           `json:"timezone"`
	Enabled  bool             `json:"enabled"`
	Schedule []ScheduleWindow `json:"schedule"`
}

type Command struct {
	CommandID   string     `json:"command_id"`
	Type        string     `json:"type"`
	PayloadJSON string     `json:"payload"`
	Status      string     `json:"status"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Override struct {
	ForceBlocked      bool       `json:"force_blocked"`
	ManualUnlockUntil *time.Time `json:"manual_unlock_until"`
	ExtendedUntil     *time.Time `json:"extended_until"`
}

type EffectiveState struct {
	State         string     `json:"state"`
	Reason        string     `json:"reason"`
	NextAllowedAt *time.Time `json:"next_allowed_at"`
	ExtendedUntil *time.Time `json:"extended_until"`
}

type PersistedState struct {
	Policy             Policy                 `json:"policy"`
	Override           Override               `json:"override"`
	Effective          EffectiveState         `json:"effective"`
	AppliedCommandIDs  map[string]time.Time   `json:"applied_command_ids"`
	ServerTimeOffsetMs int64                  `json:"server_time_offset_ms"`
	UpdatedAt          time.Time              `json:"updated_at"`
	UsageBuckets       map[string]UsageBucket `json:"usage_buckets"`
}

type StateMessage struct {
	Type          string     `json:"type"`
	State         string     `json:"state"`
	Reason        string     `json:"reason"`
	NextAllowedAt *time.Time `json:"next_allowed_at,omitempty"`
	ExtendedUntil *time.Time `json:"extended_until,omitempty"`
	Timezone      string     `json:"timezone,omitempty"`
	Policy        *Policy    `json:"policy,omitempty"`
}

type AgentMessage struct {
	Type          string `json:"type"`
	WindowsUser   string `json:"windows_user"`
	Application   string `json:"application"`
	ActiveSeconds int64  `json:"active_seconds"`
	IdleSeconds   int64  `json:"idle_seconds"`
}

type UsageBucket struct {
	WindowsUser   string    `json:"windows_user"`
	Application   string    `json:"application"`
	UsageDate     time.Time `json:"usage_date"`
	ActiveSeconds int64     `json:"active_seconds"`
	IdleSeconds   int64     `json:"idle_seconds"`
}
