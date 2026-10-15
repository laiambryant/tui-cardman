package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/logging"
)

// QueryRowContext executes a single-row query with automatic logging
func QueryRowContext(ctx context.Context, db *sql.DB, query string, args ...any) *sql.Row {
	slog.Debug("query row", "query", logging.SanitizeQuery(query), "args", logging.SanitizeArgs(query, args))
	return db.QueryRowContext(ctx, query, args...)
}

// QueryRowContextTx executes a single-row query within a transaction with automatic logging
func QueryRowContextTx(ctx context.Context, tx *sql.Tx, query string, args ...any) *sql.Row {
	slog.Debug("query row (tx)", "query", logging.SanitizeQuery(query), "args", logging.SanitizeArgs(query, args))
	return tx.QueryRowContext(ctx, query, args...)
}

// ExecContext executes a statement with automatic logging and error wrapping
func ExecContext(ctx context.Context, db *sql.DB, query string, args ...any) (sql.Result, error) {
	slog.Debug("exec", "query", logging.SanitizeQuery(query), "args", logging.SanitizeArgs(query, args))
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		slog.Error("exec failed", "query", logging.SanitizeQuery(query), "error", err)
		return nil, fmt.Errorf("exec failed: %w", err)
	}
	return result, nil
}

// ExecContextTx executes a statement within a transaction with automatic logging
func ExecContextTx(ctx context.Context, tx *sql.Tx, query string, args ...any) (sql.Result, error) {
	slog.Debug("exec (tx)", "query", logging.SanitizeQuery(query), "args", logging.SanitizeArgs(query, args))
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
	slog.Debug("query", "query", logging.SanitizeQuery(query), "args", logging.SanitizeArgs(query, args))
	rows, err := db.Query(query, args...)
	if err != nil {
		slog.Error("query failed", "query", logging.SanitizeQuery(query), "error", err)
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return rows, nil
}

// QueryRow is a non-context version for backward compatibility
func QueryRow(db *sql.DB, query string, args ...any) *sql.Row {
	slog.Debug("query row", "query", logging.SanitizeQuery(query), "args", logging.SanitizeArgs(query, args))
	return db.QueryRow(query, args...)
}

// Exec is a non-context version for backward compatibility
func Exec(db *sql.DB, query string, args ...any) (sql.Result, error) {
	slog.Debug("exec", "query", logging.SanitizeQuery(query), "args", logging.SanitizeArgs(query, args))
	result, err := db.Exec(query, args...)
	if err != nil {
		slog.Error("exec failed", "query", logging.SanitizeQuery(query), "error", err)
		return nil, fmt.Errorf("exec failed: %w", err)
	}
	return result, nil
}

// QueryQuantities runs a query selecting (card_id, quantity) rows and returns them as a map.
func QueryQuantities(db *sql.DB, query string, args ...any) (map[int64]int, error) {
	rows, err := Query(db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	quantities := make(map[int64]int)
	for rows.Next() {
		var cardID int64
		var quantity int
		if err := rows.Scan(&cardID, &quantity); err != nil {
			return nil, fmt.Errorf("scan quantity row: %w", err)
		}
		quantities[cardID] = quantity
	}
	return quantities, rows.Err()
}

// SetQuantities stores per-card quantities for one owner (a user's collection, a deck
// or a list) in a single transaction. upsertQuery takes (ownerID, cardID, quantity);
// deleteQuery takes (ownerID, cardID) and is used for quantities <= 0.
func SetQuantities(ctx context.Context, db *sql.DB, ownerID int64, updates map[int64]int, upsertQuery, deleteQuery string) error {
	return WithTransaction(ctx, db, func(tx *sql.Tx) error {
		for cardID, quantity := range updates {
			var err error
			if quantity <= 0 {
				_, err = ExecContextTx(ctx, tx, deleteQuery, ownerID, cardID)
			} else {
				_, err = ExecContextTx(ctx, tx, upsertQuery, ownerID, cardID, quantity)
			}
			if err != nil {
				return err
			}
		}
		return nil
	})
}
