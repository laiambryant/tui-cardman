package runtimecfg

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type themeJSON struct {
	Name       string `json:"name"`
	Focused    string `json:"focused"`
	Blurred    string `json:"blurred"`
	Error      string `json:"error"`
	Title      string `json:"title"`
	Background string `json:"background"`
	Foreground string `json:"foreground"`
}

// GetThemesPath returns the path to the themes directory
func GetThemesPath() string {
	return "themes"
}

// LoadThemeFromFile loads a theme from a JSON file
func LoadThemeFromFile(path string) (*ColorScheme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &FailedToReadThemeFileError{Path: path, Err: err}
	}
	var tj themeJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, &FailedToParseThemeFileError{Path: path, Err: err}
	}
	if err := validateTheme(&tj, path); err != nil {
		return nil, err
	}
	return &ColorScheme{
		Name:       tj.Name,
		Focused:    lipgloss.Color(tj.Focused),
		Blurred:    lipgloss.Color(tj.Blurred),
		Error:      lipgloss.Color(tj.Error),
		Title:      lipgloss.Color(tj.Title),
		Background: lipgloss.Color(tj.Background),
		Foreground: lipgloss.Color(tj.Foreground),
	}, nil
}

func validateTheme(tj *themeJSON, path string) error {
	if tj.Name == "" {
		return &InvalidThemeFormatError{Path: path, Field: "name"}
	}
	if tj.Focused == "" {
		return &InvalidThemeFormatError{Path: path, Field: "focused"}
	}
	if tj.Blurred == "" {
		return &InvalidThemeFormatError{Path: path, Field: "blurred"}
	}
	if tj.Error == "" {
		return &InvalidThemeFormatError{Path: path, Field: "error"}
	}
	if tj.Title == "" {
		return &InvalidThemeFormatError{Path: path, Field: "title"}
	}
	return nil
}

// LoadThemesFromDirectory loads all themes from the themes directory
func LoadThemesFromDirectory(dir string) (map[string]ColorScheme, error) {
	themes := make(map[string]ColorScheme)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return themes, &ThemeDirectoryNotFoundError{Path: dir}
		}
		return themes, &FailedToReadThemeFileError{Path: dir, Err: err}
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		themePath := filepath.Join(dir, entry.Name())
		theme, err := LoadThemeFromFile(themePath)
		if err != nil {
			slog.Warn("failed to load theme file, skipping", "path", themePath, "error", err)
			continue
		}
		themeName := strings.TrimSuffix(entry.Name(), ".json")
		themes[themeName] = *theme
	}
	return themes, nil
}

// MergeThemes combines hardcoded and loaded themes, with loaded themes taking precedence
func MergeThemes(hardcoded, loaded map[string]ColorScheme) map[string]ColorScheme {
	merged := make(map[string]ColorScheme)
	for k, v := range hardcoded {
		merged[k] = v
	}
	for k, v := range loaded {
		merged[k] = v
	}
	return merged
}
