package config

import "testing"

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	t.Setenv("PUBLIC_BASE_URL", "")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("AUTH_TOKEN_SECRET", "")

	cfg := Load()

	expectedDSN := "postgresql://tester@localhost:5432/rankster?sslmode=disable"
	t.Setenv("USER", "tester")
	cfg = Load()

	if cfg.DatabaseURL != expectedDSN {
		t.Fatalf("unexpected default DATABASE_URL: %s", cfg.DatabaseURL)
	}
	if cfg.Host != "0.0.0.0" {
		t.Fatalf("unexpected default HOST: %s", cfg.Host)
	}
	if cfg.Port != "8000" {
		t.Fatalf("unexpected default PORT: %s", cfg.Port)
	}
	if cfg.PublicBaseURL != "http://localhost:8000" {
		t.Fatalf("unexpected default PUBLIC_BASE_URL: %s", cfg.PublicBaseURL)
	}
	if cfg.GoogleClientID != "" {
		t.Fatalf("unexpected default GOOGLE_CLIENT_ID: %s", cfg.GoogleClientID)
	}
	if cfg.AuthTokenSecret != "rankster-dev-secret" {
		t.Fatalf("unexpected default AUTH_TOKEN_SECRET: %s", cfg.AuthTokenSecret)
	}
}

func TestDefaultDatabaseURLFallsBackToPostgres(t *testing.T) {
	t.Setenv("USER", "")

	if got := defaultDatabaseURL(); got == "" {
		t.Fatal("defaultDatabaseURL returned an empty string")
	}
}

func TestLoadUsesEnvironmentOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://tester@localhost:5432/demo?sslmode=disable")
	t.Setenv("HOST", "127.0.0.1")
	t.Setenv("PORT", "9000")
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:9000")
	t.Setenv("GOOGLE_CLIENT_ID", "google-client-id")
	t.Setenv("AUTH_TOKEN_SECRET", "super-secret")

	cfg := Load()

	if cfg.DatabaseURL != "postgresql://tester@localhost:5432/demo?sslmode=disable" {
		t.Fatalf("unexpected DATABASE_URL: %s", cfg.DatabaseURL)
	}
	if cfg.Host != "127.0.0.1" {
		t.Fatalf("unexpected HOST: %s", cfg.Host)
	}
	if cfg.Port != "9000" {
		t.Fatalf("unexpected PORT: %s", cfg.Port)
	}
	if cfg.PublicBaseURL != "http://localhost:9000" {
		t.Fatalf("unexpected PUBLIC_BASE_URL: %s", cfg.PublicBaseURL)
	}
	if cfg.GoogleClientID != "google-client-id" {
		t.Fatalf("unexpected GOOGLE_CLIENT_ID: %s", cfg.GoogleClientID)
	}
	if cfg.AuthTokenSecret != "super-secret" {
		t.Fatalf("unexpected AUTH_TOKEN_SECRET: %s", cfg.AuthTokenSecret)
	}
}
