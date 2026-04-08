package config

import "os"

type Config struct {
	DatabaseURL   string
	Host          string
	Port          string
	PublicBaseURL string
}

func Load() Config {
	return Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/rankster?sslmode=disable"),
		Host:          getEnv("HOST", "0.0.0.0"),
		Port:          getEnv("PORT", "8000"),
		PublicBaseURL: getEnv("PUBLIC_BASE_URL", "http://localhost:8000"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
