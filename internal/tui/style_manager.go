package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
)

var defaultStyleManager *StyleManager

func init() {
	scheme := runtimecfg.ColorSchemes["default"]
	defaultStyleManager = NewStyleManager(scheme, false, "components")
}

var (
	focusedStyle = func() lipgloss.Style { return defaultStyleManager.GetFocusedStyle() }()
	blurredStyle = func() lipgloss.Style { return defaultStyleManager.GetBlurredStyle() }()
	noStyle      = func() lipgloss.Style { return defaultStyleManager.GetNoStyle() }()
	helpStyle    = func() lipgloss.Style { return defaultStyleManager.GetHelpStyle() }()
	errorStyle   = func() lipgloss.Style { return defaultStyleManager.GetErrorStyle() }()
	titleStyle   = func() lipgloss.Style { return defaultStyleManager.GetTitleStyle() }()
)

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
	sm.focusedStyle = lipgloss.NewStyle().Foreground(sm.scheme.Focused)
	sm.blurredStyle = lipgloss.NewStyle().Foreground(sm.scheme.Blurred)
	sm.titleStyle = lipgloss.NewStyle().Bold(true).Foreground(sm.scheme.Title)
	sm.errorStyle = lipgloss.NewStyle().Foreground(sm.scheme.Error)
	sm.helpStyle = sm.blurredStyle
	sm.noStyle = lipgloss.NewStyle()
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

// GetBoxStyle returns a styled box with optional focus
func (sm *StyleManager) GetBoxStyle(focused bool) lipgloss.Style {
	borderColor := sm.scheme.Blurred
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(40).
		Align(lipgloss.Center)
	if focused {
		borderColor = sm.scheme.Focused
		style = style.Bold(true)
	}
	style = style.BorderForeground(borderColor)
	return sm.applyBackground(style, "box")
}

// GetModalStyle returns a styled modal dialog
func (sm *StyleManager) GetModalStyle() lipgloss.Style {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(sm.scheme.Title).
		Padding(1, 2).
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
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	return s
}

// GetPanelStyle returns a styled panel
func (sm *StyleManager) GetPanelStyle() lipgloss.Style {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(sm.scheme.Blurred).
		Padding(1, 2)
	return sm.applyBackground(style, "panel")
}

// GetTabStyle returns a styled tab
func (sm *StyleManager) GetTabStyle(active bool) lipgloss.Style {
	if active {
		style := lipgloss.NewStyle().
			Border(lipgloss.Border{Top: "─", Bottom: " ", Left: "│", Right: "│"}, true).
			BorderForeground(sm.scheme.Focused).
			Padding(0, 1)
		return sm.applyBackground(style, "tab")
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.Border{Top: "─", Bottom: "─", Left: "│", Right: "│"}, true).
		BorderForeground(sm.scheme.Blurred).
		Padding(0, 1)
	return sm.applyBackground(style, "tab")
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

func (sm *StyleManager) shouldApplyComponentBackground(component string) bool {
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
