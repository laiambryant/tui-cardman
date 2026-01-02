package runtimecfg

import (
	"fmt"
	"sync"
)

// Subscriber is a function that gets called when config changes
type Subscriber func(*RuntimeConfig)

// Manager manages the runtime configuration
type Manager struct {
	mu          sync.RWMutex
	config      *RuntimeConfig
	configPath  string
	subscribers []Subscriber
}

// NewManager creates a new config manager
func NewManager(configPath string) (*Manager, error) {
	cfg, err := Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &Manager{
		config:      cfg,
		configPath:  configPath,
		subscribers: make([]Subscriber, 0),
	}, nil
}

// Get returns a copy of the current configuration
func (m *Manager) Get() *RuntimeConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modifications
	cfg := *m.config

	// Deep copy the keybindings map
	cfg.Keybindings = make(map[string]string)
	for k, v := range m.config.Keybindings {
		cfg.Keybindings[k] = v
	}

	return &cfg
}

// Set updates the configuration and notifies subscribers
func (m *Manager) Set(cfg *RuntimeConfig) error {
	m.mu.Lock()

	// Validate keybindings for conflicts
	if err := m.validateKeybindings(cfg.Keybindings); err != nil {
		m.mu.Unlock()
		return err
	}

	m.config = cfg
	subscribers := make([]Subscriber, len(m.subscribers))
	copy(subscribers, m.subscribers)
	m.mu.Unlock()

	// Notify subscribers outside of lock
	for _, sub := range subscribers {
		sub(cfg)
	}

	// Save to disk
	if err := Save(cfg, m.configPath); err != nil {
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

	// If specific actions provided, check only those
	if len(actions) > 0 {
		for _, action := range actions {
			if key, exists := m.config.Keybindings[action]; exists && key == keyPress {
				return action
			}
		}
		return ""
	}

	// Otherwise check all keybindings
	for action, key := range m.config.Keybindings {
		if key == keyPress {
			return action
		}
	}
	return ""
}

// validateKeybindings checks for duplicate key bindings
func (m *Manager) validateKeybindings(bindings map[string]string) error {
	seen := make(map[string]string)

	for action, key := range bindings {
		if key == "" {
			continue // Allow unbound actions
		}

		if existingAction, exists := seen[key]; exists {
			return fmt.Errorf("key '%s' is bound to both '%s' and '%s'", key, existingAction, action)
		}
		seen[key] = action
	}

	return nil
}

// SetKeybinding updates a single keybinding
func (m *Manager) SetKeybinding(action, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a copy of current keybindings
	newBindings := make(map[string]string)
	for k, v := range m.config.Keybindings {
		newBindings[k] = v
	}

	// Update the specific binding
	newBindings[action] = key

	// Validate
	if err := m.validateKeybindings(newBindings); err != nil {
		return err
	}

	// Apply
	m.config.Keybindings[action] = key

	return nil
}
