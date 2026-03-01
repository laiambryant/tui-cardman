package runtimecfg

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/laiambryant/tui-cardman/internal/model"
)

// ButtonStrategy defines the interface for configuration storage strategies
type ButtonStrategy interface {
	Load() (*RuntimeConfig, error)
	Save(config *RuntimeConfig) error
	HasUnsavedChanges() bool
	MarkUnsaved()
	IsAvailable() bool
}

// LocalStrategy handles file-based configuration storage
type LocalStrategy struct {
	configPath string
}

// NewLocalStrategy creates a new local file strategy
func NewLocalStrategy(configPath string) *LocalStrategy {
	return &LocalStrategy{
		configPath: configPath,
	}
}

func (s *LocalStrategy) Load() (*RuntimeConfig, error) {
	cfg, err := Load(s.configPath)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *LocalStrategy) Save(config *RuntimeConfig) error {
	if err := Save(config, s.configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

func (s *LocalStrategy) HasUnsavedChanges() bool {
	return false
}

func (s *LocalStrategy) MarkUnsaved() {
}

func (s *LocalStrategy) IsAvailable() bool {
	return true
}

// ButtonConfigService interface for remote strategy (to avoid import cycle)
type ButtonConfigService interface {
	GetByUserID(ctx context.Context, userID int64) (*model.ButtonConfiguration, error)
	Save(ctx context.Context, userID int64, config *RuntimeConfig) error
	InitializeDefault(ctx context.Context, userID int64) error
	MigrateLocalToDB(ctx context.Context, userID int64, localPath string) error
}

// RemoteStrategy handles database-based configuration storage
type RemoteStrategy struct {
	service           ButtonConfigService
	userID            int64
	hasUnsavedChanges bool
	defaultConfig     *RuntimeConfig
}

// NewRemoteStrategy creates a new remote database strategy
func NewRemoteStrategy(service ButtonConfigService, userID int64, defaultConfig *RuntimeConfig, localPath string) *RemoteStrategy {
	strategy := &RemoteStrategy{
		service:       service,
		userID:        userID,
		defaultConfig: defaultConfig,
	}
	if service != nil && localPath != "" {
		if _, err := os.Stat(localPath); err == nil {
			ctx := context.Background()
			if migrateErr := service.MigrateLocalToDB(ctx, userID, localPath); migrateErr != nil {
				slog.Warn("failed to migrate local config to database", "user_id", userID, "path", localPath, "error", migrateErr)
			}
		}
	}
	return strategy
}

func (s *RemoteStrategy) Load() (*RuntimeConfig, error) {
	if s.service == nil {
		slog.Debug("no remote service available, using default config")
		return s.defaultConfig, nil
	}
	slog.Info("loading configuration from database", "user_id", s.userID)
	buttonConfig, err := s.service.GetByUserID(context.Background(), s.userID)
	if err != nil {
		slog.Debug("no database configuration found, using default", "user_id", s.userID, "error", err)
		return s.defaultConfig, nil
	}
	var dbConfig RuntimeConfig
	if err := json.Unmarshal([]byte(buttonConfig.Configuration), &dbConfig); err != nil {
		slog.Error("failed to unmarshal database configuration", "user_id", s.userID, "error", err)
		return s.defaultConfig, nil
	}
	populateKeybindings(dbConfig, s)
	s.hasUnsavedChanges = false
	slog.Info("loaded configuration from database", "user_id", s.userID)
	return &dbConfig, nil
}

func populateKeybindings(dbConfig RuntimeConfig, s *RemoteStrategy) {
	if dbConfig.Keybindings == nil {
		dbConfig.Keybindings = s.defaultConfig.Keybindings
	} else {
		for action, key := range s.defaultConfig.Keybindings {
			if _, exists := dbConfig.Keybindings[action]; !exists {
				dbConfig.Keybindings[action] = key
			}
		}
	}
}

func (s *RemoteStrategy) Save(config *RuntimeConfig) error {
	if s.service == nil {
		return ErrNoRemoteServiceAvailable
	}
	slog.Info("saving configuration to database", "user_id", s.userID)
	if err := s.service.Save(context.Background(), s.userID, config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	s.hasUnsavedChanges = false
	slog.Info("configuration saved to database", "user_id", s.userID)
	return nil
}

func (s *RemoteStrategy) HasUnsavedChanges() bool {
	return s.hasUnsavedChanges
}

func (s *RemoteStrategy) MarkUnsaved() {
	s.hasUnsavedChanges = true
	slog.Debug("marked configuration as unsaved", "user_id", s.userID)
}

func (s *RemoteStrategy) IsAvailable() bool {
	return s.service != nil
}
