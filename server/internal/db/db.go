package db

import (
	"forlittle/server/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Open(databaseURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
}

func AutoMigrate(database *gorm.DB) error {
	return database.AutoMigrate(
		&models.Admin{},
		&models.LittleMonk{},
		&models.Machine{},
		&models.BrowserProfile{},
		&models.PolicyRule{},
		&models.VisitLog{},
	)
}
