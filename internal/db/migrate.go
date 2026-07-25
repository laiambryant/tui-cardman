package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
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

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

// ApplyEmbeddedMigrations applies the migrations compiled into the binary.
// This works regardless of the process working directory.
func ApplyEmbeddedMigrations(db *sql.DB) error {
	sub, err := fs.Sub(embeddedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}
	return applyMigrationsFS(db, sub)
}

// ApplyMigrations applies migrations from a directory on disk.
func ApplyMigrations(db *sql.DB, migrationsDir string) error {
	return applyMigrationsFS(db, os.DirFS(migrationsDir))
}

func applyMigrationsFS(db *sql.DB, fsys fs.FS) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	migrations, err := loadPendingMigrations(db, fsys)
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
	slog.Debug("exec", "query", logging.SanitizeQuery(createMigrationsTable), "args", []any{})
	_, err := db.Exec(createMigrationsTable)
	return err
}

func loadPendingMigrations(db *sql.DB, fsys fs.FS) ([]migration, error) {
	files, err := fs.Glob(fsys, "*.up.sql")
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
		pending = append(pending, migration{version: version, path: path, fsys: fsys})
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
	slog.Debug("query row", "query", logging.SanitizeQuery(checkMigrationApplied), "args", []any{version})
	var count int
	err := db.QueryRow(checkMigrationApplied, version).Scan(&count)
	return count > 0, err
}

func applyMigration(db *sql.DB, m migration) error {
	slog.Debug("applying migration", "version", m.version, "path", m.path)
	content, err := fs.ReadFile(m.fsys, m.path)
	if err != nil {
		return err
	}
	foreignKeysOff := strings.Contains(string(content), "-- cardman:foreign_keys_off")
	var (
		tx   *sql.Tx
		conn *sql.Conn
	)
	if foreignKeysOff {
		conn, err = db.Conn(context.Background())
		if err != nil {
			return fmt.Errorf("open dedicated migration connection: %w", err)
		}
		defer conn.Close()
		if _, err = conn.ExecContext(context.Background(), "PRAGMA foreign_keys = OFF"); err != nil {
			return fmt.Errorf("disable foreign keys for %s: %w", m.path, err)
		}
		defer func() {
			if _, pragmaErr := conn.ExecContext(context.Background(), "PRAGMA foreign_keys = ON"); pragmaErr != nil {
				slog.Error("failed to restore foreign key enforcement", "migration", m.path, "error", pragmaErr)
			}
		}()
		tx, err = conn.BeginTx(context.Background(), nil)
	} else {
		tx, err = db.Begin()
	}
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", m.path, err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			slog.Debug("tx rollback", "err", err, "version", m.version)
		}
	}()
	slog.Debug("exec (tx)", "query", logging.SanitizeQuery(string(content)), "args", []any{})
	slog.Debug("tx exec", "version", m.version, "path", m.path)
	if _, err := tx.Exec(string(content)); err != nil {
		slog.Debug("tx exec failed", "version", m.version, "err", err)
		return fmt.Errorf("apply %s: %w", m.path, err)
	}
	if foreignKeysOff {
		if err := verifyForeignKeys(tx); err != nil {
			return fmt.Errorf("verify %s: %w", m.path, err)
		}
	}
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

func verifyForeignKeys(tx *sql.Tx) error {
	rows, err := tx.Query("PRAGMA foreign_key_check")
	if err != nil {
		return fmt.Errorf("run foreign key check: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return rows.Err()
	}
	var (
		table  string
		rowID  sql.NullInt64
		parent string
		fkID   int
	)
	if err := rows.Scan(&table, &rowID, &parent, &fkID); err != nil {
		return fmt.Errorf("scan foreign key violation: %w", err)
	}
	return fmt.Errorf("foreign key violation in table %s row %v referencing %s (constraint %d)", table, rowID, parent, fkID)
}

type migration struct {
	version string
	path    string
	fsys    fs.FS
}
