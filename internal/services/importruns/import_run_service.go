package importruns

import (
	"context"
	"database/sql"
	"fmt"
	"time"
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
	result, err := s.db.ExecContext(ctx, createImportRunQuery, importType, "running", time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to create import run: %w", err)
	}
	return result.LastInsertId()
}

// UpdateImportRun updates an existing import run record
func (s *ImportRunServiceImpl) UpdateImportRun(ctx context.Context, runID int64, status string, setsProcessed, cardsImported, errorsCount int, notes string) error {
	_, err := s.db.ExecContext(ctx, updateImportRunQuery, status, setsProcessed, cardsImported, errorsCount, time.Now(), notes, runID)
	if err != nil {
		return fmt.Errorf("failed to update import run: %w", err)
	}
	return nil
}
