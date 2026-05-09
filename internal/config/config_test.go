package config

import "testing"

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	t.Setenv("PUBLIC_BASE_URL", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("AUTH_TOKEN_SECRET", "")
	t.Setenv("ENABLE_MOCK_AUTH", "")

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
	if len(cfg.AllowedOrigins) != 0 {
		t.Fatalf("unexpected default CORS_ALLOWED_ORIGINS: %v", cfg.AllowedOrigins)
	}
	if cfg.GoogleClientID != "" {
		t.Fatalf("unexpected default GOOGLE_CLIENT_ID: %s", cfg.GoogleClientID)
	}
	if cfg.AuthTokenSecret != "rankster-dev-secret" {
		t.Fatalf("unexpected default AUTH_TOKEN_SECRET: %s", cfg.AuthTokenSecret)
	}
	if cfg.EnableMockAuth {
		t.Fatal("expected mock auth to be disabled by default")
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
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://rankster-frontend.vercel.app, https://rankster-frontend.vercel.app/ , https://example.com")
	t.Setenv("GOOGLE_CLIENT_ID", "google-client-id")
	t.Setenv("AUTH_TOKEN_SECRET", "super-secret")
	t.Setenv("ENABLE_MOCK_AUTH", "true")

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
	if got, want := cfg.AllowedOrigins, []string{"https://rankster-frontend.vercel.app", "https://example.com"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected CORS_ALLOWED_ORIGINS: %v", got)
	}
	if cfg.GoogleClientID != "google-client-id" {
		t.Fatalf("unexpected GOOGLE_CLIENT_ID: %s", cfg.GoogleClientID)
	}
	if cfg.AuthTokenSecret != "super-secret" {
		t.Fatalf("unexpected AUTH_TOKEN_SECRET: %s", cfg.AuthTokenSecret)
	}
	if !cfg.EnableMockAuth {
		t.Fatal("expected ENABLE_MOCK_AUTH override to enable mock auth")
	}
}
