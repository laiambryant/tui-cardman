package tui

import (
	"github.com/charmbracelet/bubbles/table"
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
	defaultStyleManager = NewStyleManager(scheme, false, "components")
	focusedStyle = defaultStyleManager.GetFocusedStyle()
	blurredStyle = defaultStyleManager.GetBlurredStyle()
	noStyle = defaultStyleManager.GetNoStyle()
	helpStyle = defaultStyleManager.GetHelpStyle()
	errorStyle = defaultStyleManager.GetErrorStyle()
	titleStyle = defaultStyleManager.GetTitleStyle()
}

// StyleManager centralizes all TUI styling and applies themes
type StyleManager struct {
	scheme           runtimecfg.ColorScheme
	opaqueBackground bool
	backgroundStyle  string
	focusedStyle     lipgloss.Style
	blurredStyle     lipgloss.Style
	titleStyle       lipgloss.Style
	errorStyle       lipgloss.Style
	helpStyle        lipgloss.Style
	noStyle          lipgloss.Style
}

// NewStyleManager creates a new style manager with the given color scheme and settings
func NewStyleManager(scheme runtimecfg.ColorScheme, opaqueBackground bool, backgroundStyle string) *StyleManager {
	sm := &StyleManager{
		scheme:           scheme,
		opaqueBackground: opaqueBackground,
		backgroundStyle:  backgroundStyle,
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
	sm.noStyle = lipgloss.NewStyle()
}

func (sm *StyleManager) createForegroundStyle(color lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(color)
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
	return lipgloss.NewStyle().Foreground(sm.scheme.Disabled).Strikethrough(true)
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
	return sm.applyBackground(style, "box")
}

func (sm *StyleManager) createRoundedBorderStyle(color lipgloss.Color, pad bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color)
	if pad {
		style = style.Padding(1, 2)
	}
	return style
}

// GetModalStyle returns a styled modal dialog
func (sm *StyleManager) GetModalStyle() lipgloss.Style {
	style := sm.createRoundedBorderStyle(sm.scheme.Title, true).
		Width(50)
	return sm.applyBackground(style, "modal")
}

// GetTableStyles returns styled table configuration
func (sm *StyleManager) GetTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(sm.scheme.Blurred).
		BorderBottom(true).
		Bold(true).
		Align(lipgloss.Center)
	selectedForeground, selectedBackground := sm.getTableSelectionColors()
	s.Selected = s.Selected.
		Foreground(selectedForeground).
		Background(selectedBackground).
		Bold(false)
	return s
}

// GetSettingsSelectedStyle returns the style for selected settings items
func (sm *StyleManager) GetSettingsSelectedStyle() lipgloss.Style {
	selectedForeground, selectedBackground := sm.getTableSelectionColors()
	return lipgloss.NewStyle().
		Foreground(selectedForeground).
		Background(selectedBackground).
		Bold(false)
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
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(sm.scheme.Blurred)
}

// GetPanelStyle returns a styled panel
func (sm *StyleManager) GetPanelStyle() lipgloss.Style {
	style := sm.createRoundedBorderStyle(sm.scheme.Blurred, true)
	return sm.applyBackground(style, "panel")
}

// GetTabStyle returns a styled tab
func (sm *StyleManager) GetTabStyle(active bool) lipgloss.Style {
	color := sm.scheme.Blurred
	border := sm.createInactiveBorder()
	if active {
		color = sm.scheme.Focused
		border = sm.createActiveBorder()
	}
	style := lipgloss.NewStyle().
		Border(border, true).
		BorderForeground(color).
		Padding(0, 1)
	return sm.applyBackground(style, "tab")
}

func (sm *StyleManager) createActiveBorder() lipgloss.Border {
	return lipgloss.Border{Top: "─", Bottom: " ", Left: "│", Right: "│"}
}

func (sm *StyleManager) createInactiveBorder() lipgloss.Border {
	return lipgloss.Border{Top: "─", Bottom: "─", Left: "│", Right: "│"}
}

// ApplyFullBackground applies background to entire view if enabled
func (sm *StyleManager) ApplyFullBackground(content string, width, height int) string {
	if !sm.shouldApplyFullBackground() {
		return content
	}
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Background(sm.scheme.Background).
		Foreground(sm.scheme.Foreground)
	return style.Render(content)
}

func (sm *StyleManager) applyBackground(style lipgloss.Style, component string) lipgloss.Style {
	if !sm.shouldApplyComponentBackground(component) {
		return style
	}
	return style.Background(sm.scheme.Background).Foreground(sm.scheme.Foreground)
}

func (sm *StyleManager) shouldApplyComponentBackground(_ string) bool {
	if !sm.opaqueBackground {
		return false
	}
	switch sm.backgroundStyle {
	case "none":
		return false
	case "components", "both":
		return true
	case "full":
		return false
	default:
		return false
	}
}

func (sm *StyleManager) shouldApplyFullBackground() bool {
	if !sm.opaqueBackground {
		return false
	}
	return sm.backgroundStyle == "full" || sm.backgroundStyle == "both"
}

// UpdateTheme updates the style manager with a new theme
func (sm *StyleManager) UpdateTheme(scheme runtimecfg.ColorScheme, opaqueBackground bool, backgroundStyle string) {
	sm.scheme = scheme
	sm.opaqueBackground = opaqueBackground
	sm.backgroundStyle = backgroundStyle
	sm.initializeStyles()
	focusedStyle = sm.GetFocusedStyle()
	blurredStyle = sm.GetBlurredStyle()
	noStyle = sm.GetNoStyle()
	helpStyle = sm.GetHelpStyle()
	errorStyle = sm.GetErrorStyle()
	titleStyle = sm.GetTitleStyle()
}
