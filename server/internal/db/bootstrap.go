package db

import (
	"fmt"

	"forlittle/server/internal/config"
	"forlittle/server/internal/models"
	"forlittle/server/internal/services"

	"gorm.io/gorm"
)

func Bootstrap(database *gorm.DB, cfg config.Config) error {
	return seedAdminUser(database, cfg)
}

func seedAdminUser(database *gorm.DB, cfg config.Config) error {
	var user models.User
	err := database.Where("email = ?", cfg.AdminEmail).First(&user).Error
	if err == nil {
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("find admin user: %w", err)
	}

	passwordHash, err := services.HashPassword(cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	user = models.User{
		Email:        cfg.AdminEmail,
		PasswordHash: passwordHash,
		DisplayName:  cfg.AdminDisplayName,
		Role:         "admin",
		Status:       "active",
	}

	if err := database.Create(&user).Error; err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	return nil
}
