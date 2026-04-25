package models

import "time"

type Admin struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:100;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
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
	LittleMonkID uint      `gorm:"index;not null" json:"little_monk_id"`
	Action       string    `gorm:"size:20;not null" json:"action"`
	PatternType  string    `gorm:"size:50;not null" json:"pattern_type"`
	PatternValue string    `gorm:"size:255;not null" json:"pattern_value"`
	Enabled      bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
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
