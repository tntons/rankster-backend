package config

import "testing"

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	t.Setenv("PUBLIC_BASE_URL", "")

	cfg := Load()

	if cfg.DatabaseURL != "postgresql://postgres:postgres@localhost:5432/rankster?sslmode=disable" {
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
}

func TestLoadUsesEnvironmentOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://tester@localhost:5432/demo?sslmode=disable")
	t.Setenv("HOST", "127.0.0.1")
	t.Setenv("PORT", "9000")
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:9000")

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
}
