package runtimecfg

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

// RuntimeConfig holds all application configuration
type RuntimeConfig struct {
	Keybindings map[string]string `json:"keybindings"`
	UI          UIConfig          `json:"ui"`
}

// UIConfig holds UI-related settings
type UIConfig struct {
	CompactLists bool   `json:"compact_lists"`
	ColorScheme  string `json:"color_scheme"`
}

// ColorScheme defines a color palette for the TUI
type ColorScheme struct {
	Name                    string
	Focused                 lipgloss.Color
	Blurred                 lipgloss.Color
	Error                   lipgloss.Color
	Title                   lipgloss.Color
	Background              lipgloss.Color
	Foreground              lipgloss.Color
	TableSelectedForeground lipgloss.Color
	TableSelectedBackground lipgloss.Color
	Disabled                lipgloss.Color
}

// ColorSchemes contains predefined color schemes
var ColorSchemes = map[string]ColorScheme{
	"default": {
		Name:                    "Default",
		Focused:                 lipgloss.Color("205"),
		Blurred:                 lipgloss.Color("240"),
		Error:                   lipgloss.Color("9"),
		Title:                   lipgloss.Color("170"),
		Background:              lipgloss.Color(""),
		Foreground:              lipgloss.Color(""),
		TableSelectedForeground: lipgloss.Color(""),
		TableSelectedBackground: lipgloss.Color(""),
		Disabled:                lipgloss.Color("240"),
	},
	"dark": {
		Name:                    "Dark",
		Focused:                 lipgloss.Color("15"),
		Blurred:                 lipgloss.Color("8"),
		Error:                   lipgloss.Color("1"),
		Title:                   lipgloss.Color("12"),
		Background:              lipgloss.Color("0"),
		Foreground:              lipgloss.Color("15"),
		TableSelectedForeground: lipgloss.Color(""),
		TableSelectedBackground: lipgloss.Color(""),
		Disabled:                lipgloss.Color("8"),
	},
	"light": {
		Name:                    "Light",
		Focused:                 lipgloss.Color("4"),
		Blurred:                 lipgloss.Color("7"),
		Error:                   lipgloss.Color("1"),
		Title:                   lipgloss.Color("2"),
		Background:              lipgloss.Color("15"),
		Foreground:              lipgloss.Color("0"),
		TableSelectedForeground: lipgloss.Color(""),
		TableSelectedBackground: lipgloss.Color(""),
		Disabled:                lipgloss.Color("7"),
	},
}

// GetColorScheme returns a color scheme by name, or default if not found
func GetColorScheme(name string) ColorScheme {
	allSchemes := GetAllColorSchemes()
	if scheme, exists := allSchemes[name]; exists {
		return scheme
	}
	return allSchemes["default"]
}

// GetAllColorSchemes returns all color schemes (hardcoded + loaded from files)
func GetAllColorSchemes() map[string]ColorScheme {
	loaded, err := LoadThemesFromDirectory(GetThemesPath())
	if err != nil {
		return ColorSchemes
	}
	return MergeThemes(ColorSchemes, loaded)
}

// GetColorSchemeNames returns all available color scheme names
func GetColorSchemeNames() []string {
	allSchemes := GetAllColorSchemes()
	names := make([]string, 0, len(allSchemes))
	for name := range allSchemes {
		names = append(names, name)
	}
	return names
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
		},
		UI: UIConfig{
			CompactLists: false,
			ColorScheme:  "default",
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
		return nil, &FailedToReadConfigFileError{Err: err}
	}
	var cfg RuntimeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, &FailedToParseConfigFileError{Err: err}
	}
	defaults := Default()
	initializeKeybindings(&cfg, defaults)
	initializeUISettings(&cfg, defaults)
	return &cfg, nil
}

func initializeKeybindings(cfg *RuntimeConfig, defaults *RuntimeConfig) {
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

func initializeUISettings(cfg *RuntimeConfig, defaults *RuntimeConfig) {
	if cfg.UI.ColorScheme == "" {
		cfg.UI.ColorScheme = defaults.UI.ColorScheme
	}
}

// Save writes configuration to a file path
func Save(cfg *RuntimeConfig, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &FailedToCreateConfigDirectoryError{Err: err}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return &FailedToMarshalConfigError{Err: err}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return &FailedToWriteConfigFileError{Err: err}
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
