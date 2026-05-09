package config

import (
	"fmt"
	"net/url"
	"os"
	"os/user"
	"strings"
)

type Config struct {
	DatabaseURL      string
	Host             string
	Port             string
	PublicBaseURL    string
	AllowedOrigins   []string
	GoogleClientID   string
	AuthTokenSecret  string
	UploadDir        string
	CloudinaryURL    string
	CloudinaryFolder string
}

func Load() Config {
	return Config{
		DatabaseURL:      getEnv("DATABASE_URL", defaultDatabaseURL()),
		Host:             getEnv("HOST", "0.0.0.0"),
		Port:             getEnv("PORT", "8000"),
		PublicBaseURL:    getEnv("PUBLIC_BASE_URL", "http://localhost:8000"),
		AllowedOrigins:   parseCSVEnv("CORS_ALLOWED_ORIGINS"),
		GoogleClientID:   getEnv("GOOGLE_CLIENT_ID", ""),
		AuthTokenSecret:  getEnv("AUTH_TOKEN_SECRET", "rankster-dev-secret"),
		UploadDir:        getEnv("UPLOAD_DIR", "uploads"),
		CloudinaryURL:    getEnv("CLOUDINARY_URL", ""),
		CloudinaryFolder: getEnv("CLOUDINARY_FOLDER", "rankster/uploads"),
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

func parseCSVEnv(key string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		value := strings.TrimRight(strings.TrimSpace(part), "/")
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}
