package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/laiambryant/gotestutils/ctesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/testutil"
)

// TestApplyMigrations tests the migration application process
func TestApplyMigrations(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	migrationsDir := "./migrations"
	err := ApplyMigrations(db, migrationsDir)
	require.NoError(t, err)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0, "At least one migration should be applied")
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	require.NoError(t, err, "Users table should exist after migrations")
	err = ApplyMigrations(db, migrationsDir)
	require.NoError(t, err)
}

// TestApplyMigrations_NonExistentDirectory tests migration with invalid directory
func TestApplyMigrations_NonExistentDirectory(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	err := ApplyMigrations(db, "/nonexistent/directory")
	require.NoError(t, err)
}

// TestEnsureMigrationsTable tests migrations table creation
func TestEnsureMigrationsTable(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	err := ensureMigrationsTable(db)
	require.NoError(t, err)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Migrations table should be empty initially")
	err = ensureMigrationsTable(db)
	require.NoError(t, err)
}

// TestExtractVersion tests version extraction from migration filenames
func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "standard migration file",
			path:     "/path/to/001_create_users.up.sql",
			expected: "001",
		},
		{
			name:     "migration with underscores in name",
			path:     "/path/to/002_create_card_games.up.sql",
			expected: "002",
		},
		{
			name:     "relative path",
			path:     "./migrations/003_create_sets.up.sql",
			expected: "003",
		},
		{
			name:     "no directory",
			path:     "004_create_cards.up.sql",
			expected: "004",
		},
		{
			name:     "invalid filename",
			path:     "invalid.sql",
			expected: "invalid.sql",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := extractVersion(tt.path)
			assert.Equal(t, tt.expected, version)
		})
	}
}

// TestExtractVersion_Characterization tests version extraction with characterization testing
func TestExtractVersion_Characterization(t *testing.T) {
	tests := []ctesting.CharacterizationTest[string]{
		ctesting.NewCharacterizationTest(
			"001",
			nil,
			func() (string, error) {
				return extractVersion("/path/to/001_create_users.up.sql"), nil
			},
		),
		ctesting.NewCharacterizationTest(
			"002",
			nil,
			func() (string, error) {
				return extractVersion("./migrations/002_create_card_games.up.sql"), nil
			},
		),
		ctesting.NewCharacterizationTest(
			"999",
			nil,
			func() (string, error) {
				return extractVersion("999_future_migration.up.sql"), nil
			},
		),
	}
	ctesting.VerifyCharacterizationTestsAndResults(t, tests, false)
}

// TestIsApplied tests checking if a migration has been applied
func TestIsApplied(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	err := ensureMigrationsTable(db)
	require.NoError(t, err)
	applied, err := isApplied(db, "001")
	require.NoError(t, err)
	assert.False(t, applied)
	_, err = db.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, datetime('now'))", "001")
	require.NoError(t, err)
	applied, err = isApplied(db, "001")
	require.NoError(t, err)
	assert.True(t, applied)
	applied, err = isApplied(db, "002")
	require.NoError(t, err)
	assert.False(t, applied)
}

// TestLoadPendingMigrations tests loading pending migrations
func TestLoadPendingMigrations(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	err := ensureMigrationsTable(db)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, datetime('now'))", "001")
	require.NoError(t, err)
	pending, err := loadPendingMigrations(db, os.DirFS("./migrations"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(pending), 0)
	for _, m := range pending {
		assert.NotEqual(t, "001", m.version)
	}
}

// TestApplyMigration tests applying a single migration
func TestApplyMigration(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	err := ensureMigrationsTable(db)
	require.NoError(t, err)
	tempDir := t.TempDir()
	migrationPath := filepath.Join(tempDir, "001_test_migration.up.sql")
	migrationContent := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);`
	err = os.WriteFile(migrationPath, []byte(migrationContent), 0o644)
	require.NoError(t, err)
	m := migration{
		version: "001",
		path:    filepath.Base(migrationPath),
		fsys:    os.DirFS(tempDir),
	}
	err = applyMigration(db, m)
	require.NoError(t, err)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_table").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	var recordedVersion string
	err = db.QueryRow("SELECT version FROM schema_migrations WHERE version = ?", "001").Scan(&recordedVersion)
	require.NoError(t, err)
	assert.Equal(t, "001", recordedVersion)
}

// TestApplyMigration_InvalidSQL tests applying a migration with invalid SQL
func TestApplyMigration_InvalidSQL(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	err := ensureMigrationsTable(db)
	require.NoError(t, err)
	tempDir := t.TempDir()
	migrationPath := filepath.Join(tempDir, "999_invalid_migration.up.sql")
	migrationContent := `INVALID SQL STATEMENT;`
	err = os.WriteFile(migrationPath, []byte(migrationContent), 0o644)
	require.NoError(t, err)
	m := migration{
		version: "999",
		path:    filepath.Base(migrationPath),
		fsys:    os.DirFS(tempDir),
	}
	err = applyMigration(db, m)
	require.Error(t, err)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", "999").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// TestMigration_Characterization tests migration workflow with characterization testing
func TestMigration_Characterization(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	tests := []ctesting.CharacterizationTest[error]{
		ctesting.NewCharacterizationTest(
			nil,
			nil,
			func() (error, error) {
				return ensureMigrationsTable(db), nil
			},
		),
	}
	ctesting.VerifyCharacterizationTestsAndResults(t, tests, false)
}

// TestApplyEmbeddedMigrations verifies migrations compiled into the binary apply cleanly
func TestApplyEmbeddedMigrations(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	err := ApplyEmbeddedMigrations(db)
	require.NoError(t, err)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0, "Embedded migrations should be applied")
	err = ApplyEmbeddedMigrations(db)
	require.NoError(t, err)
}

func TestScopeSetsMigrationSplitsSharedHistoricalSet(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	_, err := db.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME NOT NULL
		);
		CREATE TABLE card_games (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE
		);
		CREATE TABLE sets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_id TEXT UNIQUE NOT NULL,
			code TEXT,
			name TEXT NOT NULL,
			printed_total INTEGER,
			total INTEGER,
			symbol_url TEXT,
			logo_url TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE cards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			card_game_id INTEGER NOT NULL REFERENCES card_games(id),
			name TEXT NOT NULL,
			set_id INTEGER REFERENCES sets(id)
		);
		INSERT INTO card_games (id, name) VALUES
			(1, 'Pokemon'),
			(2, 'Magic: The Gathering');
		INSERT INTO sets (id, api_id, code, name, total)
			VALUES (10, 'shared', 'SH', 'Shared Set', 2);
		INSERT INTO cards (card_game_id, name, set_id) VALUES
			(1, 'Pokemon Card', 10),
			(2, 'Magic Card', 10);
	`)
	require.NoError(t, err)

	err = applyMigration(db, migration{
		version: "022",
		path:    "022_scope_sets_to_card_games.up.sql",
		fsys:    os.DirFS("./migrations"),
	})
	require.NoError(t, err)

	var setCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sets WHERE api_id = 'shared'").Scan(&setCount))
	assert.Equal(t, 2, setCount)
	var mismatches int
	require.NoError(t, db.QueryRow(`
		SELECT COUNT(*)
		FROM cards c
		JOIN sets s ON s.id = c.set_id
		WHERE c.card_game_id != s.card_game_id
	`).Scan(&mismatches))
	assert.Zero(t, mismatches)
	rows, err := db.Query("PRAGMA foreign_key_check")
	require.NoError(t, err)
	defer rows.Close()
	assert.False(t, rows.Next())
	require.NoError(t, rows.Err())
}
