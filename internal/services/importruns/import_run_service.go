package importruns

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/laiambryant/tui-cardman/internal/db"
)

// ImportRunService defines the interface for import run-related operations
type ImportRunService interface {
	CreateImportRun(ctx context.Context, importType string) (int64, error)
	UpdateImportRun(ctx context.Context, runID int64, status string, setsProcessed, cardsImported, errorsCount int, notes string) error
}

// ImportRunServiceImpl implements the ImportRunService interface
type ImportRunServiceImpl struct {
	db *sql.DB
}

// NewImportRunService creates a new instance of ImportRunServiceImpl
func NewImportRunService(db *sql.DB) ImportRunService {
	return &ImportRunServiceImpl{db: db}
}

const (
	createImportRunQuery = `INSERT INTO import_runs (import_type, status, started_at) 
	    VALUES (?, ?, ?)`

	updateImportRunQuery = `UPDATE import_runs 
	    SET status = ?, sets_processed = ?, cards_imported = ?, errors_count = ?, 
		   completed_at = ?, notes = ?
	    WHERE id = ?`
)

// CreateImportRun creates a new import run record and returns its ID
func (s *ImportRunServiceImpl) CreateImportRun(ctx context.Context, importType string) (int64, error) {
	result, err := db.ExecContext(ctx, s.db, createImportRunQuery, importType, "running", time.Now())
	if err != nil {
		slog.Error("failed to create import run", "import_type", importType, "error", err)
		return 0, fmt.Errorf("failed to create import run: %w", err)
	}
	runID, err := result.LastInsertId()
	if err != nil {
		slog.Error("failed to get last insert id for import run", "import_type", importType, "error", err)
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}
	slog.Debug("created import run", "run_id", runID, "import_type", importType)
	return runID, nil
}

// UpdateImportRun updates an existing import run record
func (s *ImportRunServiceImpl) UpdateImportRun(ctx context.Context, runID int64, status string, setsProcessed, cardsImported, errorsCount int, notes string) error {
	_, err := db.ExecContext(ctx, s.db, updateImportRunQuery, status, setsProcessed, cardsImported, errorsCount, time.Now(), notes, runID)
	if err != nil {
		slog.Error("failed to update import run", "run_id", runID, "status", status, "error", err)
		return fmt.Errorf("failed to update import run: %w", err)
	}
	slog.Debug("updated import run", "run_id", runID, "status", status, "sets_processed", setsProcessed, "cards_imported", cardsImported, "errors_count", errorsCount)
	return nil
}
