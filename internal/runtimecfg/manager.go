package runtimecfg

import (
	"fmt"
	"maps"
	"slices"
	"sync"
)

// Subscriber is a function that gets called when config changes
type Subscriber func(*RuntimeConfig)

// Manager manages the runtime configuration
type Manager struct {
	mu          sync.RWMutex
	config      *RuntimeConfig
	strategy    ButtonStrategy
	subscribers []Subscriber
	keyToAction map[string]string // Reverse index for O(1) key→action lookups
}

// NewManager creates a new config manager
func NewManager(isLocal bool, configPath string, service ButtonConfigService, userID int64) (*Manager, error) {
	var strategy ButtonStrategy
	var config *RuntimeConfig
	var err error
	if isLocal {
		strategy = NewLocalStrategy(configPath)
		config, err = strategy.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load local config: %w", err)
		}
	} else {
		config = Default()
		strategy = NewRemoteStrategy(service, userID, config, configPath)
		dbConfig, loadErr := strategy.Load()
		if loadErr == nil && dbConfig != nil {
			config = dbConfig
		}
	}
	return &Manager{
		config:      config,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(config.Keybindings),
	}, nil
}

func (m *Manager) Get() *RuntimeConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg := *m.config
	cfg.Keybindings = make(map[string]string)
	maps.Copy(cfg.Keybindings, m.config.Keybindings)
	return &cfg
}

// Set updates the configuration and notifies subscribers
func (m *Manager) Set(cfg *RuntimeConfig) error {
	m.mu.Lock()
	if err := m.validateKeybindings(cfg.Keybindings); err != nil {
		m.mu.Unlock()
		return err
	}
	m.config = cfg
	m.keyToAction = buildKeyToActionMap(cfg.Keybindings)
	m.strategy.MarkUnsaved()
	subscribers := make([]Subscriber, len(m.subscribers))
	copy(subscribers, m.subscribers)
	m.mu.Unlock()
	for _, sub := range subscribers {
		sub(cfg)
	}
	if err := m.strategy.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

// Subscribe adds a subscriber that will be called when config changes
func (m *Manager) Subscribe(sub Subscriber) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers = append(m.subscribers, sub)
}

// KeyForAction returns the key binding for the given action
func (m *Manager) KeyForAction(action string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if key, exists := m.config.Keybindings[action]; exists {
		return key
	}
	return ""
}

// MatchAction returns the action name if the keypress matches any configured action
func (m *Manager) MatchAction(keyPress string, actions ...string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(actions) > 0 {
		return m.matchAmongActions(keyPress, actions)
	}
	if action, exists := m.keyToAction[keyPress]; exists {
		return action
	}
	return ""
}

func (m *Manager) matchAmongActions(keyPress string, actions []string) string {
	if action, exists := m.keyToAction[keyPress]; exists {
		if slices.Contains(actions, action) {
			return action
		}
	}
	return ""
}

func (m *Manager) validateKeybindings(bindings map[string]string) error {
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

// SetKeybinding updates a single keybinding
func (m *Manager) SetKeybinding(action, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	newBindings := make(map[string]string)
	maps.Copy(newBindings, m.config.Keybindings)
	newBindings[action] = key
	if err := m.validateKeybindings(newBindings); err != nil {
		return err
	}
	m.config.Keybindings[action] = key
	m.keyToAction = buildKeyToActionMap(m.config.Keybindings)
	return nil
}

// SaveToDatabase saves the current configuration using the strategy
func (m *Manager) SaveToDatabase() error {
	m.mu.RLock()
	config := m.config
	m.mu.RUnlock()
	return m.strategy.Save(config)
}

// LoadFromDatabase loads configuration using the strategy
func (m *Manager) LoadFromDatabase() error {
	config, err := m.strategy.Load()
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.config = config
	m.mu.Unlock()
	return nil
}

// HasUnsavedChanges returns true if there are unsaved changes
func (m *Manager) HasUnsavedChanges() bool {
	return m.strategy.HasUnsavedChanges()
}

// MarkUnsaved marks the configuration as having unsaved changes
func (m *Manager) MarkUnsaved() {
	m.strategy.MarkUnsaved()
}

// IsUserMode returns true if the manager is using remote strategy
func (m *Manager) IsUserMode() bool {
	_, isRemote := m.strategy.(*RemoteStrategy)
	return isRemote
}

func buildKeyToActionMap(keybindings map[string]string) map[string]string {
	keyToAction := make(map[string]string, len(keybindings))
	for action, key := range keybindings {
		keyToAction[key] = action
	}
	return keyToAction
}
