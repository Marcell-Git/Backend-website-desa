package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppPort       string
	AppEnv        string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSslMode     string
	JWTSecret     string
	AccessMinutes int
	RefreshDays   int
}

func Load() (*Config, error) {
	cfg := &Config{
		AppPort:       getEnv("APP_PORT", "8080"),
		AppEnv:        getEnv("APP_ENV", "development"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "postgres"),
		DBName:        getEnv("DB_NAME", "rt26_db"),
		DBSslMode:     getEnv("DB_SSLMODE", "disable"),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		AccessMinutes: getEnvInt("JWT_ACCESS_MINUTES", 15),
		RefreshDays:   getEnvInt("JWT_REFRESH_DAYS", 7),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET wajib diisi pada file .env")
	}

	return cfg, nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSslMode,
	)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
