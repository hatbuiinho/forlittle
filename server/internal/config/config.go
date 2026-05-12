package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv               string
	Port                 string
	DatabaseURL          string
	ExtensionReleasesDir string
	AdminEmail           string
	AdminPassword        string
	AdminDisplayName     string
	AdminSessionTTLHours int
	AdminCookieSecure    bool
	AdminCookieSameSite  string
	CORSAllowedOrigins   []string
}

func Load() Config {
	return Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/forlittle?sslmode=disable"),
		ExtensionReleasesDir: getEnv("EXTENSION_RELEASES_DIR", "extension-releases"),
		AdminEmail:           getEnv("ADMIN_EMAIL", "admin@example.com"),
		AdminPassword:        getEnv("ADMIN_PASSWORD", "admin123"),
		AdminDisplayName:     getEnv("ADMIN_DISPLAY_NAME", "Temple Admin"),
		AdminSessionTTLHours: getEnvInt("ADMIN_SESSION_TTL_HOURS", 24),
		AdminCookieSecure:    getEnvBool("ADMIN_COOKIE_SECURE", false),
		AdminCookieSameSite:  strings.ToLower(getEnv("ADMIN_COOKIE_SAME_SITE", "lax")),
		CORSAllowedOrigins:   getEnvList("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvList(key string, fallback string) []string {
	rawValue := getEnv(key, fallback)
	parts := strings.Split(rawValue, ",")
	values := make([]string, 0, len(parts))

	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}

	return values
}
