package db

import (
	"fmt"

	"forlittle/server/internal/config"
	"forlittle/server/internal/models"
	"forlittle/server/internal/services"

	"gorm.io/gorm"
)

func Bootstrap(database *gorm.DB, cfg config.Config) error {
	return seedAdmin(database, cfg)
}

func seedAdmin(database *gorm.DB, cfg config.Config) error {
	var admin models.Admin
	err := database.Where("username = ?", cfg.AdminUser).First(&admin).Error
	if err == nil {
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("find admin: %w", err)
	}

	passwordHash, err := services.HashPassword(cfg.AdminPass)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	admin = models.Admin{
		Username:     cfg.AdminUser,
		PasswordHash: passwordHash,
	}

	if err := database.Create(&admin).Error; err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	return nil
}
