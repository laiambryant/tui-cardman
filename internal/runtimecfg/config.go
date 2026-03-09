// Package runtimecfg manages user-editable runtime configuration persisted to disk.
package runtimecfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RuntimeConfig holds all application configuration
type RuntimeConfig struct {
	Keybindings map[string]string `json:"keybindings"`
	UI          UIConfig          `json:"ui"`
}

type UIConfig struct {
	CompactLists   bool            `json:"compact_lists"`
	VisibleColumns map[string]bool `json:"visible_columns"`
	ColumnOrder    []string        `json:"column_order"`
}

// Default returns a RuntimeConfig with sensible defaults
func Default() *RuntimeConfig {
	return &RuntimeConfig{
		Keybindings: map[string]string{
			// Global actions
			"quit":     "ctrl+c",
			"quit_alt": "esc",
			"settings": "f1",

			// Navigation
			"nav_up":       "up",
			"nav_down":     "down",
			"nav_left":     "left",
			"nav_right":    "right",
			"nav_prev_tab": "shift+tab",
			"nav_next_tab": "tab",
			"select":       "enter",
			"back":         "q",

			// Card game tabs
			"tab_all_cards":    "1",
			"tab_collection":   "2",
			"tab_search":       "3",
			"switch_tab_left":  "h",
			"switch_tab_right": "l",

			// Search
			"search_focus": "/",
			"search_clear": "ctrl+u",

			// Collection management
			"save":               "ctrl+s",
			"increment_quantity": "+",
			"decrement_quantity": "delete",

			// CRUD operations (lists, decks)
			"create_new": "n",
			"edit":       "e",
			"delete":     "d",

			// Import / Export
			"export": "x",
			"import": "i",

			// Pagination
			"page_next": "ctrl+n",
			"page_prev": "ctrl+p",

			// Import queue operations (ctrl+ to avoid search conflicts)
			"queue_add":    "ctrl+a",
			"queue_remove": "ctrl+r",
			"queue_start":  "ctrl+g",
			"queue_clear":  "ctrl+l",
		},
		UI: UIConfig{
			CompactLists:   false,
			VisibleColumns: DefaultVisibleColumns(),
			ColumnOrder:    DefaultColumnOrder(),
		},
	}
}

// Load reads configuration from a file path
func Load(path string) (*RuntimeConfig, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Default(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg RuntimeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	defaults := Default()
	initializeKeybindings(&cfg, defaults)
	initializeVisibleColumns(&cfg, defaults)
	initializeColumnOrder(&cfg, defaults)
	return &cfg, nil
}

func DefaultVisibleColumns() map[string]bool {
	return map[string]bool{
		"name":      true,
		"expansion": true,
		"rarity":    true,
		"number":    true,
		"quantity":  true,
		"artist":    false,
	}
}

func DefaultColumnOrder() []string {
	return []string{"name", "expansion", "rarity", "number", "quantity", "artist"}
}

func initializeColumnOrder(cfg, defaults *RuntimeConfig) {
	if len(cfg.UI.ColumnOrder) == 0 {
		cfg.UI.ColumnOrder = defaults.UI.ColumnOrder
		return
	}
	existing := make(map[string]bool, len(cfg.UI.ColumnOrder))
	for _, k := range cfg.UI.ColumnOrder {
		existing[k] = true
	}
	for _, k := range defaults.UI.ColumnOrder {
		if !existing[k] {
			cfg.UI.ColumnOrder = append(cfg.UI.ColumnOrder, k)
		}
	}
}

func initializeVisibleColumns(cfg, defaults *RuntimeConfig) {
	if cfg.UI.VisibleColumns == nil {
		cfg.UI.VisibleColumns = defaults.UI.VisibleColumns
	} else {
		for col, vis := range defaults.UI.VisibleColumns {
			if _, exists := cfg.UI.VisibleColumns[col]; !exists {
				cfg.UI.VisibleColumns[col] = vis
			}
		}
	}
}

func initializeKeybindings(cfg, defaults *RuntimeConfig) {
	if cfg.Keybindings == nil {
		cfg.Keybindings = defaults.Keybindings
	} else {
		for action, key := range defaults.Keybindings {
			if _, exists := cfg.Keybindings[action]; !exists {
				cfg.Keybindings[action] = key
			}
		}
	}
}

// Save writes configuration to a file path
func Save(cfg *RuntimeConfig, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

// GetConfigPath returns the path to the config file. Respects CARDMAN_CONFIG env var, otherwise uses default
func GetConfigPath() string {
	if path := os.Getenv("CARDMAN_CONFIG"); path != "" {
		return path
	}
	return ".cardman.json"
}
