package db

import (
	"forlittle/server/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Open(databaseURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		SkipDefaultTransaction: true,
	})
}

func AutoMigrate(database *gorm.DB) error {
	if err := database.AutoMigrate(
		&models.Admin{},
		&models.User{},
		&models.UserSession{},
		&models.LittleMonk{},
		&models.Machine{},
		&models.BrowserProfile{},
		&models.ExtensionClient{},
		&models.PolicyRule{},
		&models.PolicyConfig{},
		&models.VisitLog{},
		&models.DeviceClient{},
		&models.TimePolicy{},
		&models.MachineTimePolicyAssignment{},
		&models.TimeScheduleWindow{},
		&models.MachineTimeState{},
		&models.DeviceCommand{},
		&models.AppUsage{},
	); err != nil {
		return err
	}

	// Existing phase-1 databases created policy_rules.little_monk_id as NOT NULL.
	// Global rules need this column nullable while we keep the old field for future scopes.
	if err := database.Exec("ALTER TABLE policy_rules ALTER COLUMN little_monk_id DROP NOT NULL").Error; err != nil {
		return err
	}

	// Chrome Extension records created before ExtensionClient existed share the
	// Machine row with every other client. Backfill their independent lifecycle
	// metadata once, without modifying Windows Time Control records.
	if err := database.Exec(`
		INSERT INTO extension_clients (machine_id, display_name, status, token_hash, last_seen_at, created_at, updated_at)
		SELECT DISTINCT ON (machines.machine_id)
			machines.machine_id,
			machines.display_name,
			machines.status,
			machines.device_token_hash,
			machines.last_seen_at,
			machines.created_at,
			NOW()
		FROM machines
		JOIN browser_profiles ON browser_profiles.machine_id = machines.machine_id
		ORDER BY machines.machine_id
		ON CONFLICT (machine_id) DO NOTHING`).Error; err != nil {
		return err
	}

	// Title keyword lists are stored as JSON arrays and can exceed the old varchar(255) limit.
	return database.Exec("ALTER TABLE policy_rules ALTER COLUMN pattern_value TYPE text").Error
}
