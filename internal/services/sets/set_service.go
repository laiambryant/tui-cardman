package sets

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// SetService defines the interface for set-related operations
type SetService interface {
	GetSetIDByAPIID(ctx context.Context, apiID string) (int64, error)
	UpsertSet(ctx context.Context, apiID, code, name string, printedTotal, total int) (int64, error)
	GetAllSetAPIIDs(ctx context.Context) ([]string, error)
}

// SetServiceImpl implements the SetService interface
type SetServiceImpl struct {
	db *sql.DB
}

// NewSetService creates a new instance of SetServiceImpl
func NewSetService(db *sql.DB) SetService {
	return &SetServiceImpl{db: db}
}

const (
	selectSetIDQuery = `SELECT id FROM sets WHERE api_id = ?`

	insertSetQuery = `INSERT INTO sets (api_id, code, name, printed_total, total, 
				   updated_at)
	    VALUES (?, ?, ?, ?, ?, ?)`

	updateSetQuery = `UPDATE sets 
	    SET code = ?, name = ?, printed_total = ?, total = ?,  updated_at = ?
	    WHERE id = ?`

	selectAllSetAPIIDsQuery = `SELECT api_id FROM sets`
)

// GetSetIDByAPIID retrieves the database ID for a set by its API ID
func (s *SetServiceImpl) GetSetIDByAPIID(ctx context.Context, apiID string) (int64, error) {
	var setID int64
	err := s.db.QueryRowContext(ctx, selectSetIDQuery, apiID).Scan(&setID)
	if err != nil {
		slog.Error("failed to query set id by api id", "api_id", apiID, "error", err)
		return 0, err
	}
	return setID, nil
}

// UpsertSet inserts or updates a set and returns its database ID
func (s *SetServiceImpl) UpsertSet(ctx context.Context, apiID, code, name string, printedTotal, total int) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("failed to begin transaction for upsert set", "api_id", apiID, "error", err)
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var setID int64
	err = tx.QueryRowContext(ctx, selectSetIDQuery, apiID).Scan(&setID)
	if err == sql.ErrNoRows {
		result, err := tx.ExecContext(ctx, insertSetQuery,
			apiID, code, name, printedTotal, total,
			time.Now())
		if err != nil {
			slog.Error("failed to insert set", "api_id", apiID, "error", err)
			return 0, fmt.Errorf("failed to insert set: %w", err)
		}
		setID, err = result.LastInsertId()
		if err != nil {
			slog.Error("failed to get last insert id after insert", "api_id", apiID, "error", err)
			return 0, fmt.Errorf("failed to get last insert ID: %w", err)
		}
	} else if err != nil {
		slog.Error("failed to query set during upsert", "api_id", apiID, "error", err)
		return 0, fmt.Errorf("failed to query set: %w", err)
	} else {
		_, err = tx.ExecContext(ctx, updateSetQuery,
			code, name, printedTotal, total,
			time.Now(), setID)
		if err != nil {
			slog.Error("failed to update set", "api_id", apiID, "id", setID, "error", err)
			return 0, fmt.Errorf("failed to update set: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("failed to commit transaction for upsert set", "api_id", apiID, "id", setID, "error", err)
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return setID, nil
}

// GetAllSetAPIIDs retrieves all set API IDs from the database
func (s *SetServiceImpl) GetAllSetAPIIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, selectAllSetAPIIDsQuery)
	if err != nil {
		slog.Error("failed to query existing sets", "error", err)
		return nil, fmt.Errorf("failed to query existing sets: %w", err)
	}
	defer rows.Close()

	var apiIDs []string
	for rows.Next() {
		var apiID string
		if err := rows.Scan(&apiID); err != nil {
			slog.Error("failed to scan set api_id", "error", err)
			return nil, fmt.Errorf("failed to scan set api_id: %w", err)
		}
		apiIDs = append(apiIDs, apiID)
	}

	if err := rows.Err(); err != nil {
		slog.Error("error iterating set rows", "error", err)
		return nil, fmt.Errorf("error iterating sets: %w", err)
	}

	return apiIDs, nil
}
