package config

import (
	"fmt"
	"net/url"
	"os"
	"os/user"
)

type Config struct {
	DatabaseURL     string
	Host            string
	Port            string
	PublicBaseURL   string
	GoogleClientID  string
	AuthTokenSecret string
	UploadDir       string
}

func Load() Config {
	return Config{
		DatabaseURL:     getEnv("DATABASE_URL", defaultDatabaseURL()),
		Host:            getEnv("HOST", "0.0.0.0"),
		Port:            getEnv("PORT", "8000"),
		PublicBaseURL:   getEnv("PUBLIC_BASE_URL", "http://localhost:8000"),
		GoogleClientID:  getEnv("GOOGLE_CLIENT_ID", ""),
		AuthTokenSecret: getEnv("AUTH_TOKEN_SECRET", "rankster-dev-secret"),
		UploadDir:       getEnv("UPLOAD_DIR", "uploads"),
	}
}

func defaultDatabaseURL() string {
	username := os.Getenv("USER")
	if username == "" {
		if currentUser, err := user.Current(); err == nil && currentUser.Username != "" {
			username = currentUser.Username
		}
	}
	if username == "" {
		username = "postgres"
	}

	return fmt.Sprintf(
		"postgresql://%s@localhost:5432/rankster?sslmode=disable",
		url.QueryEscape(username),
	)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
