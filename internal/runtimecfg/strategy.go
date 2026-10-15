package runtimecfg

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/laiambryant/tui-cardman/internal/model"
)

// ButtonStrategy defines how the configuration is stored.
type ButtonStrategy interface {
	Load() (*RuntimeConfig, error)
	Save(config *RuntimeConfig) error
}

// LocalStrategy stores the configuration in a JSON file.
type LocalStrategy struct {
	configPath string
}

// NewLocalStrategy creates a new local file strategy
func NewLocalStrategy(configPath string) *LocalStrategy {
	return &LocalStrategy{configPath: configPath}
}

func (s *LocalStrategy) Load() (*RuntimeConfig, error) {
	return Load(s.configPath)
}

func (s *LocalStrategy) Save(config *RuntimeConfig) error {
	return Save(config, s.configPath)
}

// ButtonConfigService interface for remote strategy (to avoid import cycle)
type ButtonConfigService interface {
	GetByUserID(ctx context.Context, userID int64) (*model.ButtonConfiguration, error)
	Save(ctx context.Context, userID int64, config *RuntimeConfig) error
	MigrateLocalToDB(ctx context.Context, userID int64, localPath string) error
}

// RemoteStrategy stores a user's configuration in the database.
type RemoteStrategy struct {
	service ButtonConfigService
	userID  int64
}

// NewRemoteStrategy creates a new remote database strategy
func NewRemoteStrategy(service ButtonConfigService, userID int64) *RemoteStrategy {
	return &RemoteStrategy{service: service, userID: userID}
}

// MigrateFromLocal migrates a local config file to the database if the file exists.
func (s *RemoteStrategy) MigrateFromLocal(localPath string) {
	if s.service == nil || localPath == "" {
		return
	}
	if _, err := os.Stat(localPath); err != nil {
		return
	}
	if err := s.service.MigrateLocalToDB(context.Background(), s.userID, localPath); err != nil {
		slog.Warn("failed to migrate local config to database", "user_id", s.userID, "path", localPath, "error", err)
	}
}

// Load returns the user's configuration, or the defaults when there is none
// or it cannot be read.
func (s *RemoteStrategy) Load() (*RuntimeConfig, error) {
	if s.service == nil {
		return Default(), nil
	}
	buttonConfig, err := s.service.GetByUserID(context.Background(), s.userID)
	if err != nil {
		slog.Debug("no database configuration found, using default", "user_id", s.userID, "error", err)
		return Default(), nil
	}
	var cfg RuntimeConfig
	if err := json.Unmarshal([]byte(buttonConfig.Configuration), &cfg); err != nil {
		slog.Error("failed to unmarshal database configuration", "user_id", s.userID, "error", err)
		return Default(), nil
	}
	applyDefaults(&cfg)
	return &cfg, nil
}

func (s *RemoteStrategy) Save(config *RuntimeConfig) error {
	if s.service == nil {
		return ErrNoRemoteServiceAvailable
	}
	if err := s.service.Save(context.Background(), s.userID, config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}
