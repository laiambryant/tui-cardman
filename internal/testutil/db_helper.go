// Package testutil provides shared test helpers for database setup and teardown.
package testutil

import (
	"database/sql"
	"fmt"
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

// CreateTestSchema creates a basic test schema without migrations
func CreateTestSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	// Create users table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			surname TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			last_login DATETIME,
			active INTEGER NOT NULL DEFAULT 1
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Create card_games table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS card_games (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create card_games table: %v", err)
	}

	// Create sets table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_id TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			series TEXT,
			total_cards INTEGER,
			release_date TEXT,
			card_game_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (card_game_id) REFERENCES card_games(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sets table: %v", err)
	}

	// Create cards table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_id TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			set_id INTEGER NOT NULL,
			number TEXT,
			rarity TEXT,
			types TEXT,
			supertype TEXT,
			subtypes TEXT,
			hp TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (set_id) REFERENCES sets(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create cards table: %v", err)
	}

	// Create user_cards table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_cards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			card_id INTEGER NOT NULL,
			quantity INTEGER NOT NULL DEFAULT 1,
			acquired_date DATETIME,
			notes TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (card_id) REFERENCES cards(id),
			UNIQUE(user_id, card_id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create user_cards table: %v", err)
	}
}

// TruncateAllTables truncates all tables in the database (useful for test cleanup)
func TruncateAllTables(t *testing.T, db *sql.DB) {
	t.Helper()

	tables := []string{
		"user_cards",
		"cards",
		"sets",
		"card_games",
		"users",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Errorf("Failed to truncate table %s: %v", table, err)
		}
	}
}
