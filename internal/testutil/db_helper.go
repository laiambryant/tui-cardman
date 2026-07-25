// Package testutil provides shared test helpers for database setup and teardown.
package testutil

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	// Set connection limits for test consistency
	db.SetMaxOpenConns(1)

	// Verify connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	return db
}

// CleanupTestDB closes the test database connection
func CleanupTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	if err := db.Close(); err != nil {
		t.Errorf("Failed to close database: %v", err)
	}
}

// ApplyTestMigrations applies all migration files to the test database
func ApplyTestMigrations(t *testing.T, db *sql.DB, migrationsDir string) {
	t.Helper()

	// Create migrations table
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL
	)`)
	if err != nil {
		t.Fatalf("Failed to create migrations table: %v", err)
	}

	// Find all migration files
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		t.Fatalf("Failed to find migration files: %v", err)
	}

	// Apply each migration
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("Failed to read migration file %s: %v", file, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			t.Fatalf("Failed to apply migration %s: %v", file, err)
		}
	}
}
