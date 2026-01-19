package runtimecfg

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
		return nil, fmt.Errorf("failed to load local config: %w", err)
	}
	return cfg, nil
}

func (s *LocalStrategy) Save(config *RuntimeConfig) error {
	if err := Save(config, s.configPath); err != nil {
		return fmt.Errorf("failed to save local config: %w", err)
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
	GetByUserID(ctx context.Context, userID int64) (interface{}, error)
	Save(ctx context.Context, userID int64, config *RuntimeConfig) error
}

// RemoteStrategy handles database-based configuration storage
type RemoteStrategy struct {
	service           ButtonConfigService
	userID            int64
	hasUnsavedChanges bool
	defaultConfig     *RuntimeConfig
}

// NewRemoteStrategy creates a new remote database strategy
func NewRemoteStrategy(service ButtonConfigService, userID int64, defaultConfig *RuntimeConfig) *RemoteStrategy {
	return &RemoteStrategy{
		service:       service,
		userID:        userID,
		defaultConfig: defaultConfig,
	}
}

func (s *RemoteStrategy) Load() (*RuntimeConfig, error) {
	if s.service == nil {
		slog.Debug("no remote service available, using default config")
		return s.defaultConfig, nil
	}
	slog.Info("loading configuration from database", "user_id", s.userID)
	configInterface, err := s.service.GetByUserID(context.Background(), s.userID)
	if err != nil {
		slog.Debug("no database configuration found, using default", "user_id", s.userID, "error", err)
		return s.defaultConfig, nil
	}
	if configStruct, ok := configInterface.(struct {
		Configuration string
	}); ok {
		var dbConfig RuntimeConfig
		if err := json.Unmarshal([]byte(configStruct.Configuration), &dbConfig); err != nil {
			slog.Error("failed to unmarshal database configuration", "user_id", s.userID, "error", err)
			return s.defaultConfig, nil
		}
		if dbConfig.Keybindings == nil {
			dbConfig.Keybindings = s.defaultConfig.Keybindings
		} else {
			for action, key := range s.defaultConfig.Keybindings {
				if _, exists := dbConfig.Keybindings[action]; !exists {
					dbConfig.Keybindings[action] = key
				}
			}
		}
		s.hasUnsavedChanges = false
		slog.Info("loaded configuration from database", "user_id", s.userID)
		return &dbConfig, nil
	}
	slog.Debug("unexpected config format, using default", "user_id", s.userID)
	return s.defaultConfig, nil
}

func (s *RemoteStrategy) Save(config *RuntimeConfig) error {
	if s.service == nil {
		return fmt.Errorf("no remote service available")
	}
	slog.Info("saving configuration to database", "user_id", s.userID)
	err := s.service.Save(context.Background(), s.userID, config)
	if err != nil {
		return fmt.Errorf("failed to save to database: %w", err)
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
