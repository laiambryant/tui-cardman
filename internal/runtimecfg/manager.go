package runtimecfg

import (
	"fmt"
	"maps"
	"sync"
)

// Manager holds the runtime configuration and persists changes through a
// storage strategy (a local file or the user's database row).
type Manager struct {
	mu          sync.RWMutex
	config      *RuntimeConfig
	strategy    ButtonStrategy
	keyToAction map[string]string // Reverse index for O(1) key→action lookups
}

// NewManager creates a config manager. Local mode reads configPath; remote
// (SSH) mode loads the user's configuration from the database, first
// migrating configPath into it when the user has no configuration yet.
func NewManager(isLocal bool, configPath string, service ButtonConfigService, userID int64) (*Manager, error) {
	var strategy ButtonStrategy
	if isLocal {
		strategy = NewLocalStrategy(configPath)
	} else {
		remote := NewRemoteStrategy(service, userID)
		remote.MigrateFromLocal(configPath)
		strategy = remote
	}
	config, err := strategy.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &Manager{
		config:      config,
		strategy:    strategy,
		keyToAction: buildKeyToActionMap(config.Keybindings),
	}, nil
}

// Get returns a copy of the current configuration.
func (m *Manager) Get() *RuntimeConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg := *m.config
	cfg.Keybindings = maps.Clone(m.config.Keybindings)
	cfg.UI.VisibleColumns = maps.Clone(m.config.UI.VisibleColumns)
	cfg.UI.ColumnOrder = append([]string(nil), m.config.UI.ColumnOrder...)
	return &cfg
}

// Set validates, applies and saves a new configuration.
func (m *Manager) Set(cfg *RuntimeConfig) error {
	if err := validateKeybindings(cfg.Keybindings); err != nil {
		return err
	}
	m.mu.Lock()
	m.config = cfg
	m.keyToAction = buildKeyToActionMap(cfg.Keybindings)
	m.mu.Unlock()
	if err := m.strategy.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

// KeyForAction returns the key binding for the given action
func (m *Manager) KeyForAction(action string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Keybindings[action]
}

// MatchAction returns the action bound to the key press, or "" if none is.
func (m *Manager) MatchAction(keyPress string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.keyToAction[keyPress]
}

func validateKeybindings(bindings map[string]string) error {
	seen := make(map[string]string)
	for action, key := range bindings {
		if key == "" {
			return fmt.Errorf("action %q is not bound to any key", action)
		}
		if existingAction, exists := seen[key]; exists {
			return fmt.Errorf("key %q is bound to both %q and %q", key, existingAction, action)
		}
		seen[key] = action
	}
	return nil
}

func buildKeyToActionMap(keybindings map[string]string) map[string]string {
	keyToAction := make(map[string]string, len(keybindings))
	for action, key := range keybindings {
		keyToAction[key] = action
	}
	return keyToAction
}
