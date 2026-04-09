package testutil

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"gorm.io/gorm"

	appdb "rankster-backend/internal/db"
)

func NewTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	baseDSN := os.Getenv("DATABASE_URL")
	if baseDSN == "" {
		baseDSN = defaultDatabaseURL()
	}

	adminDSN, err := withDatabase(baseDSN, "postgres")
	if err != nil {
		t.Fatalf("build admin dsn: %v", err)
	}

	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		t.Fatalf("open admin db: %v", err)
	}
	t.Cleanup(func() {
		_ = adminDB.Close()
	})

	dbName := fmt.Sprintf("rankster_test_%d", time.Now().UnixNano())
	if _, err := adminDB.Exec(`CREATE DATABASE ` + quoteIdentifier(dbName)); err != nil {
		t.Fatalf("create test database: %v", err)
	}

	testDSN, err := withDatabase(baseDSN, dbName)
	if err != nil {
		t.Fatalf("build test dsn: %v", err)
	}

	database, err := appdb.Connect(testDSN)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}

	if err := appdb.EnsureDatabase(database, "http://localhost:8000"); err != nil {
		t.Fatalf("bootstrap test database: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := database.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		_, _ = adminDB.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, dbName)
		_, _ = adminDB.Exec(`DROP DATABASE IF EXISTS ` + quoteIdentifier(dbName))
	})

	return database
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

	return "postgresql://" + url.QueryEscape(username) + "@localhost:5432/rankster?sslmode=disable"
}

func withDatabase(dsn, dbName string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	parsed.Path = "/" + dbName
	return parsed.String(), nil
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
