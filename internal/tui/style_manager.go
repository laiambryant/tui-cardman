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
	sm := &StyleManager{scheme: scheme}
	sm.initializeStyles()
	return sm
}

func (sm *StyleManager) initializeStyles() {
	sm.focusedStyle = sm.createForegroundStyle(sm.scheme.Focused)
	sm.blurredStyle = sm.createForegroundStyle(sm.scheme.Blurred)
	sm.titleStyle = sm.createForegroundStyle(sm.scheme.Title).Bold(true)
	sm.errorStyle = sm.createForegroundStyle(sm.scheme.Error)
	sm.helpStyle = sm.blurredStyle
	sm.noStyle = sm.applyBGFG(lipgloss.NewStyle().Inline(true))
}

func (sm *StyleManager) createForegroundStyle(color lipgloss.Color) lipgloss.Style {
	return sm.applyBGFG(lipgloss.NewStyle().Foreground(color).Inline(true))
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
	return sm.applyBGFG(lipgloss.NewStyle().Foreground(sm.scheme.Disabled).Strikethrough(true).Inline(true))
}

// GetBoxStyle returns a styled box with optional focus
func (sm *StyleManager) GetBoxStyle(focused bool) lipgloss.Style {
	color := sm.scheme.Blurred
	if focused {
		color = sm.scheme.Focused
	}
	style := sm.createRoundedBorderStyle(color, false).Padding(1, 2).Width(40).Align(lipgloss.Center)
	if focused {
		style = style.Bold(true)
	}
	return sm.applyBGFG(style)
}

func (sm *StyleManager) createRoundedBorderStyle(color lipgloss.Color, pad bool) lipgloss.Style {
	s := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(color)
	if pad {
		s = s.Padding(1, 2)
	}
	// do not apply background/foreground here; Box will handle global theming
	return s
}

// GetModalStyle returns a styled modal dialog
func (sm *StyleManager) GetModalStyle() lipgloss.Style {
	return sm.Box(sm.scheme.Title, 50, 0, 0, 2, 1)
}

// GetTableStyles returns styled table configuration
func (sm *StyleManager) GetTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(sm.scheme.Blurred).BorderBottom(true).Bold(true).Align(lipgloss.Center)
	if sm.scheme.Background != "" {
		s.Header = s.Header.Background(sm.scheme.Background).BorderBackground(sm.scheme.Background)
	}
	if sm.scheme.Foreground != "" {
		s.Header = s.Header.Foreground(sm.scheme.Foreground)
	}
	if sm.scheme.Background != "" {
		s.Cell = s.Cell.Background(sm.scheme.Background)
	}
	if sm.scheme.Foreground != "" {
		s.Cell = s.Cell.Foreground(sm.scheme.Foreground)
	}
	fg, bg := sm.getTableSelectionColors()
	s.Selected = s.Selected.Foreground(fg).Bold(true)
	if sm.scheme.TableSelectedBackground != "" {
		s.Selected = s.Selected.Background(bg)
	}
	return s
}

// GetSettingsSelectedStyle returns the style for selected settings items
func (sm *StyleManager) GetSettingsSelectedStyle() lipgloss.Style {
	fg, bg := sm.getTableSelectionColors()
	style := lipgloss.NewStyle().Foreground(fg).Bold(false)
	if sm.scheme.TableSelectedBackground != "" {
		style = style.Background(bg)
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
	return sm.resolveColor(sm.scheme.TableSelectedForeground, sm.scheme.Foreground, sm.scheme.Focused), sm.resolveColor(sm.scheme.TableSelectedBackground, sm.scheme.Focused, sm.scheme.Title)
}

// GetTableBaseStyle returns the style for the table container
func (sm *StyleManager) GetTableBaseStyle() lipgloss.Style {
	return sm.applyBGFG(lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(sm.scheme.Blurred))
}

// GetPanelStyle returns a styled panel
func (sm *StyleManager) GetPanelStyle() lipgloss.Style {
	return sm.Box(sm.scheme.Blurred, 0, 0, 0, 2, 1)
}

// GetTabStyle returns a styled tab
func (sm *StyleManager) GetTabStyle(active bool) lipgloss.Style {
	color := sm.scheme.Blurred
	border := sm.createInactiveBorder()
	if active {
		color = sm.scheme.Focused
		border = sm.createActiveBorder()
	}
	return sm.applyBGFG(lipgloss.NewStyle().Border(border, true).BorderForeground(color).Padding(0, 1))
}

func (sm *StyleManager) createActiveBorder() lipgloss.Border {
	return lipgloss.Border{Top: "─", Bottom: " ", Left: "│", Right: "│"}
}

func (sm *StyleManager) createInactiveBorder() lipgloss.Border {
	return lipgloss.Border{Top: "─", Bottom: "─", Left: "│", Right: "│"}
}

// GetFullScreenStyle returns a style sized to the full terminal with themed background/foreground
func (sm *StyleManager) GetFullScreenStyle(width, height int) lipgloss.Style {
	return sm.applyBGFG(lipgloss.NewStyle().Width(width).Height(height))
}

// ApplyFullBackground applies background to entire view if the scheme has background colors
func (sm *StyleManager) ApplyFullBackground(content string, width, height int) string {
	if sm.scheme.Background == "" || sm.scheme.Foreground == "" {
		return content
	}
	return sm.applyBGFG(lipgloss.NewStyle().Width(width).Height(height)).Render(content)
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
	ts := sm.applyBGFG(lipgloss.NewStyle())
	ti.TextStyle = ts
	if sm.scheme.Focused != "" {
		ti.Cursor.Style = sm.applyBGFG(lipgloss.NewStyle().Foreground(sm.scheme.Focused))
	}
	ph := sm.applyBGFG(lipgloss.NewStyle().Foreground(sm.scheme.Blurred))
	ti.PlaceholderStyle = ph
	if sm.scheme.Focused != "" {
		ti.PromptStyle = sm.applyBGFG(lipgloss.NewStyle().Foreground(sm.scheme.Focused))
	}
}

func (sm *StyleManager) applyBGFG(s lipgloss.Style) lipgloss.Style {
	if sm.scheme.Background != "" {
		s = s.Background(sm.scheme.Background)
		s = s.BorderBackground(sm.scheme.Background)
	}
	if sm.scheme.Foreground != "" {
		s = s.Foreground(sm.scheme.Foreground)
	}
	return s
}

func (sm *StyleManager) Box(borderColor lipgloss.Color, width, height, maxWidth, padX, padY int) lipgloss.Style {
	s := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderColor).Padding(padY, padX)
	if width > 0 {
		s = s.Width(width)
	}
	if height > 0 {
		s = s.Height(height)
	}
	if maxWidth > 0 {
		s = s.MaxWidth(maxWidth)
	}
	return sm.applyBGFG(s)
}
