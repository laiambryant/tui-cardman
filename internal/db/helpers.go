package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/logging"
)

// QueryContext executes a query with automatic logging and error wrapping
func QueryContext(ctx context.Context, db *sql.DB, query string, args ...any) (*sql.Rows, error) {
	slog.Debug("query", "query", logging.SanitizeQuery(query), "args", args)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		slog.Error("query failed", "query", logging.SanitizeQuery(query), "error", err)
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return rows, nil
}

// QueryContextTx executes a query within a transaction with automatic logging
func QueryContextTx(ctx context.Context, tx *sql.Tx, query string, args ...any) (*sql.Rows, error) {
	slog.Debug("query (tx)", "query", logging.SanitizeQuery(query), "args", args)
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		slog.Error("query failed (tx)", "query", logging.SanitizeQuery(query), "error", err)
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return rows, nil
}

// QueryRowContext executes a single-row query with automatic logging
func QueryRowContext(ctx context.Context, db *sql.DB, query string, args ...any) *sql.Row {
	slog.Debug("query row", "query", logging.SanitizeQuery(query), "args", args)
	return db.QueryRowContext(ctx, query, args...)
}

// QueryRowContextTx executes a single-row query within a transaction with automatic logging
func QueryRowContextTx(ctx context.Context, tx *sql.Tx, query string, args ...any) *sql.Row {
	slog.Debug("query row (tx)", "query", logging.SanitizeQuery(query), "args", args)
	return tx.QueryRowContext(ctx, query, args...)
}

// ExecContext executes a statement with automatic logging and error wrapping
func ExecContext(ctx context.Context, db *sql.DB, query string, args ...any) (sql.Result, error) {
	slog.Debug("exec", "query", logging.SanitizeQuery(query), "args", args)
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		slog.Error("exec failed", "query", logging.SanitizeQuery(query), "error", err)
		return nil, fmt.Errorf("exec failed: %w", err)
	}
	return result, nil
}

// ExecContextTx executes a statement within a transaction with automatic logging
func ExecContextTx(ctx context.Context, tx *sql.Tx, query string, args ...any) (sql.Result, error) {
	slog.Debug("exec (tx)", "query", logging.SanitizeQuery(query), "args", args)
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		slog.Error("exec failed (tx)", "query", logging.SanitizeQuery(query), "error", err)
		return nil, fmt.Errorf("exec failed: %w", err)
	}
	return result, nil
}

// WithTransaction executes a function within a database transaction
// Automatically commits on success or rolls back on error
func WithTransaction(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			slog.Error("failed to rollback transaction", "error", rbErr, "original_error", err)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		slog.Error("failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Query is a non-context version for backward compatibility
func Query(db *sql.DB, query string, args ...any) (*sql.Rows, error) {
	slog.Debug("query", "query", logging.SanitizeQuery(query), "args", args)
	rows, err := db.Query(query, args...)
	if err != nil {
		slog.Error("query failed", "query", logging.SanitizeQuery(query), "error", err)
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return rows, nil
}

// QueryRow is a non-context version for backward compatibility
func QueryRow(db *sql.DB, query string, args ...any) *sql.Row {
	slog.Debug("query row", "query", logging.SanitizeQuery(query), "args", args)
	return db.QueryRow(query, args...)
}

// Exec is a non-context version for backward compatibility
func Exec(db *sql.DB, query string, args ...any) (sql.Result, error) {
	slog.Debug("exec", "query", logging.SanitizeQuery(query), "args", args)
	result, err := db.Exec(query, args...)
	if err != nil {
		slog.Error("exec failed", "query", logging.SanitizeQuery(query), "error", err)
		return nil, fmt.Errorf("exec failed: %w", err)
	}
	return result, nil
}
