package buttonconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
)

const (
	getByUserIDQuery = `
		SELECT id, user_id, configuration, created_at, updated_at
		FROM button_configuration
		WHERE user_id = ?
	`
	saveConfigQuery = `
		INSERT INTO button_configuration (user_id, configuration, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			configuration = excluded.configuration,
			updated_at = excluded.updated_at
	`
)

type ButtonConfigService interface {
	GetByUserID(ctx context.Context, userID int64) (*model.ButtonConfiguration, error)
	Save(ctx context.Context, userID int64, config *runtimecfg.RuntimeConfig) error
	InitializeDefault(ctx context.Context, userID int64) error
	MigrateLocalToDB(ctx context.Context, userID int64, localPath string) error
}

type ButtonConfigServiceImpl struct {
	db *sql.DB
}

func NewButtonConfigService(db *sql.DB) ButtonConfigService {
	return &ButtonConfigServiceImpl{
		db: db,
	}
}

func (b *ButtonConfigServiceImpl) GetByUserID(ctx context.Context, userID int64) (*model.ButtonConfiguration, error) {
	var config model.ButtonConfiguration
	err := db.QueryRowContext(ctx, b.db, getByUserIDQuery, userID).Scan(
		&config.ID,
		&config.UserID,
		&config.Configuration,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	if err != nil {
		return handleNoConfigErr(err, userID)
	}
	slog.Info("retrieved button configuration", "user_id", userID, "config_id", config.ID)
	return &config, nil
}

func (b *ButtonConfigServiceImpl) Save(ctx context.Context, userID int64, config *runtimecfg.RuntimeConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		slog.Error("failed to marshal configuration", "user_id", userID, "error", err)
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}
	result, err := db.ExecContext(ctx, b.db, saveConfigQuery, userID, string(configJSON), time.Now())
	if err != nil {
		slog.Error("failed to save button configuration", "user_id", userID, "error", err)
		return fmt.Errorf("failed to save button configuration: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		slog.Error("failed to get rows affected", "error", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	slog.Info("saved button configuration", "user_id", userID, "rows_affected", rowsAffected)
	return nil
}

func (b *ButtonConfigServiceImpl) InitializeDefault(ctx context.Context, userID int64) error {
	defaultConfig := runtimecfg.Default()
	return b.Save(ctx, userID, defaultConfig)
}

func (b *ButtonConfigServiceImpl) MigrateLocalToDB(ctx context.Context, userID int64, localPath string) error {
	_, err := b.GetByUserID(ctx, userID)
	if err == nil {
		slog.Info("user already has database configuration, skipping migration", "user_id", userID)
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing config: %w", err)
	}
	localConfig, err := runtimecfg.Load(localPath)
	if err != nil {
		return fmt.Errorf("failed to load local config for migration: %w", err)
	}
	err = b.Save(ctx, userID, localConfig)
	if err != nil {
		return fmt.Errorf("failed to migrate config to database: %w", err)
	}
	if err := os.Remove(localPath); err != nil {
		slog.Warn("failed to remove local config after migration", "path", localPath, "error", err)
	}
	slog.Info("successfully migrated local configuration to database", "user_id", userID, "path", localPath)
	return nil
}

func handleNoConfigErr(err error, userID int64) (*model.ButtonConfiguration, error) {
	if err == sql.ErrNoRows {
		slog.Debug("no button configuration found for user", "user_id", userID)
		return nil, sql.ErrNoRows
	}
	slog.Error("failed to get button configuration", "user_id", userID, "error", err)
	return nil, fmt.Errorf("failed to get button configuration: %w", err)
}
