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

	// Title keyword lists are stored as JSON arrays and can exceed the old varchar(255) limit.
	return database.Exec("ALTER TABLE policy_rules ALTER COLUMN pattern_value TYPE text").Error
}
