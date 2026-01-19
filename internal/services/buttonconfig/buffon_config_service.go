package buttonconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/laiambryant/tui-cardman/internal/logging"
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
	slog.Debug("executing query", "query", logging.SanitizeQuery(getByUserIDQuery), "args", []any{userID})
	var config model.ButtonConfiguration
	err := b.db.QueryRowContext(ctx, getByUserIDQuery, userID).Scan(
		&config.ID,
		&config.UserID,
		&config.Configuration,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	if err != nil {
		return handle_no_config_err(err, userID)
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
	slog.Debug("executing query", "query", logging.SanitizeQuery(saveConfigQuery), "args", []any{userID, string(configJSON), time.Now()})
	result, err := b.db.ExecContext(ctx, saveConfigQuery, userID, string(configJSON), time.Now())
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

func handle_no_config_err(err error, userID int64) (*model.ButtonConfiguration, error) {
	if err == sql.ErrNoRows {
		slog.Debug("no button configuration found for user", "user_id", userID)
		return nil, sql.ErrNoRows
	}
	slog.Error("failed to get button configuration", "user_id", userID, "error", err)
	return nil, fmt.Errorf("failed to get button configuration: %w", err)
}
