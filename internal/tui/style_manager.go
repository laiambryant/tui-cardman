package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
)

var defaultStyleManager *StyleManager

var (
	focusedStyle lipgloss.Style
	blurredStyle lipgloss.Style
	noStyle      lipgloss.Style
	helpStyle    lipgloss.Style
	errorStyle   lipgloss.Style
	titleStyle   lipgloss.Style
)

func init() {
	scheme := runtimecfg.ColorSchemes["default"]
	defaultStyleManager = NewStyleManager(scheme)
	focusedStyle = defaultStyleManager.GetFocusedStyle()
	blurredStyle = defaultStyleManager.GetBlurredStyle()
	noStyle = defaultStyleManager.GetNoStyle()
	helpStyle = defaultStyleManager.GetHelpStyle()
	errorStyle = defaultStyleManager.GetErrorStyle()
	titleStyle = defaultStyleManager.GetTitleStyle()
}

// StyleManager centralizes all TUI styling and applies themes
type StyleManager struct {
	scheme       runtimecfg.ColorScheme
	focusedStyle lipgloss.Style
	blurredStyle lipgloss.Style
	titleStyle   lipgloss.Style
	errorStyle   lipgloss.Style
	helpStyle    lipgloss.Style
	noStyle      lipgloss.Style
}

// NewStyleManager creates a new style manager with the given color scheme
func NewStyleManager(scheme runtimecfg.ColorScheme) *StyleManager {
	sm := &StyleManager{
		scheme: scheme,
	}
	sm.initializeStyles()
	return sm
}

func (sm *StyleManager) initializeStyles() {
	sm.focusedStyle = sm.createForegroundStyle(sm.scheme.Focused)
	sm.blurredStyle = sm.createForegroundStyle(sm.scheme.Blurred)
	sm.titleStyle = sm.createForegroundStyle(sm.scheme.Title).Bold(true)
	sm.errorStyle = sm.createForegroundStyle(sm.scheme.Error)
	sm.helpStyle = sm.blurredStyle
	// Create noStyle with theme background/foreground if set, and inline
	sm.noStyle = lipgloss.NewStyle().Inline(true)
	if sm.scheme.Background != "" {
		sm.noStyle = sm.noStyle.Background(sm.scheme.Background)
	}
	if sm.scheme.Foreground != "" {
		sm.noStyle = sm.noStyle.Foreground(sm.scheme.Foreground)
	}
}

func (sm *StyleManager) createForegroundStyle(color lipgloss.Color) lipgloss.Style {
	style := lipgloss.NewStyle().Foreground(color).Inline(true)
	// Apply theme background if explicitly set
	if sm.scheme.Background != "" {
		style = style.Background(sm.scheme.Background)
	}
	return style
}

// GetFocusedStyle returns the style for focused elements
func (sm *StyleManager) GetFocusedStyle() lipgloss.Style {
	return sm.focusedStyle
}

// GetBlurredStyle returns the style for blurred elements
func (sm *StyleManager) GetBlurredStyle() lipgloss.Style {
	return sm.blurredStyle
}

// GetTitleStyle returns the style for titles
func (sm *StyleManager) GetTitleStyle() lipgloss.Style {
	return sm.titleStyle
}

// GetErrorStyle returns the style for errors
func (sm *StyleManager) GetErrorStyle() lipgloss.Style {
	return sm.errorStyle
}

// GetHelpStyle returns the style for help text
func (sm *StyleManager) GetHelpStyle() lipgloss.Style {
	return sm.helpStyle
}

// GetNoStyle returns an unstyled lipgloss style
func (sm *StyleManager) GetNoStyle() lipgloss.Style {
	return sm.noStyle
}

// GetDisabledStyle returns the style for disabled elements
func (sm *StyleManager) GetDisabledStyle() lipgloss.Style {
	style := lipgloss.NewStyle().Foreground(sm.scheme.Disabled).Strikethrough(true).Inline(true)
	// Apply theme background if explicitly set
	if sm.scheme.Background != "" {
		style = style.Background(sm.scheme.Background)
	}
	return style
}

// GetBoxStyle returns a styled box with optional focus
func (sm *StyleManager) GetBoxStyle(focused bool) lipgloss.Style {
	color := sm.scheme.Blurred
	if focused {
		color = sm.scheme.Focused
	}
	style := sm.createRoundedBorderStyle(color, false).
		Padding(1, 2).
		Width(40).
		Align(lipgloss.Center)
	if focused {
		style = style.Bold(true)
	}
	// Apply theme background/foreground if explicitly set
	if sm.scheme.Background != "" {
		style = style.Background(sm.scheme.Background)
	}
	if sm.scheme.Foreground != "" {
		style = style.Foreground(sm.scheme.Foreground)
	}
	return style
}

func (sm *StyleManager) createRoundedBorderStyle(color lipgloss.Color, pad bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color)
	if pad {
		style = style.Padding(1, 2)
	}
	// Apply theme background to border area if explicitly set
	if sm.scheme.Background != "" {
		style = style.BorderBackground(sm.scheme.Background)
	}
	return style
}

// GetModalStyle returns a styled modal dialog
func (sm *StyleManager) GetModalStyle() lipgloss.Style {
	style := sm.createRoundedBorderStyle(sm.scheme.Title, true).
		Width(50)
	// Apply theme background/foreground if explicitly set
	if sm.scheme.Background != "" {
		style = style.Background(sm.scheme.Background)
	}
	if sm.scheme.Foreground != "" {
		style = style.Foreground(sm.scheme.Foreground)
	}
	return style
}

// GetTableStyles returns styled table configuration
func (sm *StyleManager) GetTableStyles() table.Styles {
	s := table.DefaultStyles()

	// Apply theme background and foreground to header
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(sm.scheme.Blurred).
		BorderBottom(true).
		Bold(true).
		Align(lipgloss.Center)
	if sm.scheme.Background != "" {
		s.Header = s.Header.Background(sm.scheme.Background).BorderBackground(sm.scheme.Background)
	}
	if sm.scheme.Foreground != "" {
		s.Header = s.Header.Foreground(sm.scheme.Foreground)
	}

	// Apply theme background and foreground to cells
	if sm.scheme.Background != "" {
		s.Cell = s.Cell.Background(sm.scheme.Background)
	}
	if sm.scheme.Foreground != "" {
		s.Cell = s.Cell.Foreground(sm.scheme.Foreground)
	}

	selectedForeground, selectedBackground := sm.getTableSelectionColors()
	s.Selected = s.Selected.
		Foreground(selectedForeground).
		Bold(false)
	// Only apply background if explicitly set in the color scheme
	if sm.scheme.TableSelectedBackground != "" {
		s.Selected = s.Selected.Background(selectedBackground)
	}
	return s
}

// GetSettingsSelectedStyle returns the style for selected settings items
func (sm *StyleManager) GetSettingsSelectedStyle() lipgloss.Style {
	selectedForeground, selectedBackground := sm.getTableSelectionColors()
	style := lipgloss.NewStyle().
		Foreground(selectedForeground).
		Bold(false)
	// Only apply background if explicitly set in the color scheme
	if sm.scheme.TableSelectedBackground != "" {
		style = style.Background(selectedBackground)
	}
	return style
}

func (sm *StyleManager) resolveColor(preferred, fallback1, fallback2 lipgloss.Color) lipgloss.Color {
	if preferred != "" {
		return preferred
	}
	if fallback1 != "" {
		return fallback1
	}
	return fallback2
}

func (sm *StyleManager) getTableSelectionColors() (lipgloss.Color, lipgloss.Color) {
	foreground := sm.resolveColor(sm.scheme.TableSelectedForeground, sm.scheme.Foreground, sm.scheme.Focused)
	background := sm.resolveColor(sm.scheme.TableSelectedBackground, sm.scheme.Focused, sm.scheme.Title)
	return foreground, background
}

// GetTableBaseStyle returns the style for the table container
func (sm *StyleManager) GetTableBaseStyle() lipgloss.Style {
	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(sm.scheme.Blurred)
	// Apply theme background to border area
	if sm.scheme.Background != "" {
		style = style.BorderBackground(sm.scheme.Background)
	}
	return style
}

// GetPanelStyle returns a styled panel
func (sm *StyleManager) GetPanelStyle() lipgloss.Style {
	style := sm.createRoundedBorderStyle(sm.scheme.Blurred, true)
	// Apply theme background/foreground if explicitly set
	if sm.scheme.Background != "" {
		style = style.Background(sm.scheme.Background)
	}
	if sm.scheme.Foreground != "" {
		style = style.Foreground(sm.scheme.Foreground)
	}
	return style
}

// GetTabStyle returns a styled tab
func (sm *StyleManager) GetTabStyle(active bool) lipgloss.Style {
	color := sm.scheme.Blurred
	border := sm.createInactiveBorder()
	if active {
		color = sm.scheme.Focused
		border = sm.createActiveBorder()
	}
	return lipgloss.NewStyle().
		Border(border, true).
		BorderForeground(color).
		Padding(0, 1)
}

func (sm *StyleManager) createActiveBorder() lipgloss.Border {
	return lipgloss.Border{Top: "─", Bottom: " ", Left: "│", Right: "│"}
}

func (sm *StyleManager) createInactiveBorder() lipgloss.Border {
	return lipgloss.Border{Top: "─", Bottom: "─", Left: "│", Right: "│"}
}

// ApplyFullBackground applies background to entire view if the scheme has background colors
func (sm *StyleManager) ApplyFullBackground(content string, width, height int) string {
	// Only apply background if the scheme explicitly defines background/foreground colors
	if sm.scheme.Background == "" || sm.scheme.Foreground == "" {
		return content
	}
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Background(sm.scheme.Background).
		Foreground(sm.scheme.Foreground)
	return style.Render(content)
}

// UpdateTheme updates the style manager with a new theme
func (sm *StyleManager) UpdateTheme(scheme runtimecfg.ColorScheme) {
	sm.scheme = scheme
	sm.initializeStyles()
	focusedStyle = sm.GetFocusedStyle()
	blurredStyle = sm.GetBlurredStyle()
	noStyle = sm.GetNoStyle()
	helpStyle = sm.GetHelpStyle()
	errorStyle = sm.GetErrorStyle()
	titleStyle = sm.GetTitleStyle()
}

// ApplyTextInputStyles configures a textinput with theme-aware colors
func (sm *StyleManager) ApplyTextInputStyles(ti *textinput.Model) {
	// Set text and cursor colors
	if sm.scheme.Foreground != "" {
		ti.TextStyle = lipgloss.NewStyle().Foreground(sm.scheme.Foreground)
	}
	if sm.scheme.Focused != "" {
		ti.Cursor.Style = lipgloss.NewStyle().Foreground(sm.scheme.Focused)
	}
	// Set placeholder color
	if sm.scheme.Blurred != "" {
		ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(sm.scheme.Blurred)
	}
	// Prompt color
	if sm.scheme.Focused != "" {
		ti.PromptStyle = lipgloss.NewStyle().Foreground(sm.scheme.Focused)
	}
}
