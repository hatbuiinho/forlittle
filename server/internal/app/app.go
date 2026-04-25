package app

import (
	"fmt"

	"forlittle/server/internal/config"
	"forlittle/server/internal/db"
	"forlittle/server/internal/http/router"
)

func Run() error {
	cfg := config.Load()

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	if err := db.AutoMigrate(database); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	if err := db.Bootstrap(database, cfg); err != nil {
		return fmt.Errorf("bootstrap database: %w", err)
	}

	engine := router.New(cfg, database)
	return engine.Run(":" + cfg.Port)
}
