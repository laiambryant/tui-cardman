package sets

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/laiambryant/tui-cardman/internal/logging"
)

// SetService defines the interface for set-related operations
type SetService interface {
	GetSetIDByAPIID(ctx context.Context, apiID string) (int64, error)
	UpsertSet(ctx context.Context, apiID, code, name string, printedTotal, total int) (int64, error)
	GetAllSetAPIIDs(ctx context.Context) ([]string, error)
	SetHasUserCollections(ctx context.Context, setID int64) (bool, error)
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

	checkSetHasUserCollectionsQuery = `SELECT COUNT(*) > 0 FROM user_collections uc
		JOIN cards c ON uc.card_id = c.id
		WHERE c.set_id = ?`
)

// GetSetIDByAPIID retrieves the database ID for a set by its API ID
func (s *SetServiceImpl) GetSetIDByAPIID(ctx context.Context, apiID string) (int64, error) {
	slog.Debug("query row", "query", logging.SanitizeQuery(selectSetIDQuery), "args", []any{apiID})
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
		return 0, &FailedToBeginTransactionError{Err: err}
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	slog.Debug("query row (tx)", "query", logging.SanitizeQuery(selectSetIDQuery), "args", []any{apiID})
	var setID int64
	err = tx.QueryRowContext(ctx, selectSetIDQuery, apiID).Scan(&setID)
	if err == sql.ErrNoRows {
		slog.Debug("exec (tx)", "query", logging.SanitizeQuery(insertSetQuery), "args", []any{apiID, code, name, printedTotal, total, time.Now()})
		result, err := tx.ExecContext(ctx, insertSetQuery,
			apiID, code, name, printedTotal, total,
			time.Now())
		if err != nil {
			slog.Error("failed to insert set", "api_id", apiID, "error", err)
			return 0, &FailedToInsertSetError{Err: err}
		}
		setID, err = result.LastInsertId()
		if err != nil {
			slog.Error("failed to get last insert id after insert", "api_id", apiID, "error", err)
			return 0, &FailedToGetLastInsertIDError{Err: err}
		}
	} else if err != nil {
		slog.Error("failed to query set during upsert", "api_id", apiID, "error", err)
		return 0, &FailedToQuerySetError{Err: err}
	} else {
		slog.Debug("exec (tx)", "query", logging.SanitizeQuery(updateSetQuery), "args", []any{code, name, printedTotal, total, time.Now(), setID})
		_, err = tx.ExecContext(ctx, updateSetQuery,
			code, name, printedTotal, total,
			time.Now(), setID)
		if err != nil {
			slog.Error("failed to update set", "api_id", apiID, "id", setID, "error", err)
			return 0, &FailedToUpdateSetError{Err: err}
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("failed to commit transaction for upsert set", "api_id", apiID, "id", setID, "error", err)
		return 0, &FailedToCommitTransactionError{Err: err}
	}
	return setID, nil
}

// GetAllSetAPIIDs retrieves all set API IDs from the database
func (s *SetServiceImpl) GetAllSetAPIIDs(ctx context.Context) ([]string, error) {
	slog.Debug("query", "query", logging.SanitizeQuery(selectAllSetAPIIDsQuery), "args", []any{})
	rows, err := s.db.QueryContext(ctx, selectAllSetAPIIDsQuery)
	if err != nil {
		slog.Error("failed to query existing sets", "error", err)
		return nil, &FailedToQueryExistingSetsError{Err: err}
	}
	defer rows.Close()

	var apiIDs []string
	for rows.Next() {
		var apiID string
		if err := rows.Scan(&apiID); err != nil {
			slog.Error("failed to scan set api_id", "error", err)
			return nil, &FailedToScanSetAPIIDError{Err: err}
		}
		apiIDs = append(apiIDs, apiID)
	}

	if err := rows.Err(); err != nil {
		slog.Error("error iterating set rows", "error", err)
		return nil, &ErrorIteratingSetsError{Err: err}
	}

	return apiIDs, nil
}

// SetHasUserCollections checks if any user has cards from this set in their collection
func (s *SetServiceImpl) SetHasUserCollections(ctx context.Context, setID int64) (bool, error) {
	slog.Debug("query row", "query", logging.SanitizeQuery(checkSetHasUserCollectionsQuery), "args", []any{setID})
	var hasCollections bool
	err := s.db.QueryRowContext(ctx, checkSetHasUserCollectionsQuery, setID).Scan(&hasCollections)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		slog.Error("failed to check if set has user collections", "set_id", setID, "error", err)
		return false, &FailedToCheckSetCollectionsError{Err: err}
	}
	return hasCollections, nil
}
