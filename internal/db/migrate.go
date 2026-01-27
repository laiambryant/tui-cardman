package db

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/laiambryant/tui-cardman/internal/logging"
)

const (
	createMigrationsTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL
	)`
	checkMigrationApplied = `SELECT COUNT(1) FROM schema_migrations WHERE version = ?`
	recordMigration       = `INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)`
)

func ApplyMigrations(db *sql.DB, migrationsDir string) error {
	slog.Debug("starting migration process", "dir", migrationsDir)
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	migrations, err := loadPendingMigrations(db, migrationsDir)
	if err != nil {
		return err
	}
	slog.Debug("pending migrations loaded", "count", len(migrations))
	for _, m := range migrations {
		if err := applyMigration(db, m); err != nil {
			return err
		}
	}
	slog.Debug("all migrations applied successfully")
	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	slog.Debug("ensuring schema_migrations table exists")
	_, err := db.Exec(createMigrationsTable)
	return err
}

func loadPendingMigrations(db *sql.DB, dir string) ([]migration, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	slog.Debug("found migration files", "count", len(files))
	var pending []migration
	for _, path := range files {
		version := extractVersion(path)
		if version == "" {
			slog.Debug("skipping file with invalid version", "path", path)
			continue
		}
		if applied, err := isApplied(db, version); err != nil {
			return nil, err
		} else if applied {
			slog.Debug("migration already applied", "version", version)
			continue
		}
		slog.Debug("migration pending", "version", version, "path", path)
		pending = append(pending, migration{version: version, path: path})
	}
	return pending, nil
}

func extractVersion(path string) string {
	parts := strings.SplitN(filepath.Base(path), "_", 2)
	if len(parts) < 1 {
		return ""
	}
	return parts[0]
}

func isApplied(db *sql.DB, version string) (bool, error) {
	var count int
	err := db.QueryRow(checkMigrationApplied, version).Scan(&count)
	return count > 0, err
}

func applyMigration(db *sql.DB, m migration) error {
	slog.Debug("applying migration", "version", m.version, "path", m.path)
	content, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			slog.Debug("tx rollback", "err", err, "version", m.version)
		}
	}()

	// Execute migration SQL
	slog.Debug("tx exec", "version", m.version, "path", m.path)
	if _, err := tx.Exec(string(content)); err != nil {
		slog.Debug("tx exec failed", "version", m.version, "err", err)
		return &ApplyMigrationError{Path: m.path, Err: err}
	}

	// Record migration
	slog.Debug("tx exec", "query", logging.SanitizeQuery(recordMigration), "args", []any{m.version, time.Now().UTC()})
	if _, err := tx.Exec(recordMigration, m.version, time.Now().UTC()); err != nil {
		slog.Debug("record migration failed", "version", m.version, "err", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		slog.Debug("tx commit failed", "version", m.version, "err", err)
		return err
	}
	slog.Debug("tx commit", "version", m.version)
	slog.Debug("migration applied successfully", "version", m.version)
	return nil
}

type migration struct {
	version string
	path    string
}
