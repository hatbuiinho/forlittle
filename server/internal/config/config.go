package config

import "os"

type Config struct {
	AppEnv      string
	Port        string
	DatabaseURL string
	AdminUser   string
	AdminPass   string
	AdminToken  string
}

func Load() Config {
	return Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/forlittle?sslmode=disable"),
		AdminUser:   getEnv("ADMIN_USERNAME", "admin"),
		AdminPass:   getEnv("ADMIN_PASSWORD", "admin123"),
		AdminToken:  getEnv("ADMIN_API_TOKEN", "phase1-admin-token"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
