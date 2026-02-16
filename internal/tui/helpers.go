package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
)

// HelpBuilder constructs dynamic help text from config manager
type HelpBuilder struct {
	cfg *runtimecfg.Manager
}

// NewHelpBuilder creates a new help builder
func NewHelpBuilder(cfg *runtimecfg.Manager) *HelpBuilder {
	return &HelpBuilder{cfg: cfg}
}

func (h *HelpBuilder) resolveKey(action string, defaultKey string) string {
	if h.cfg != nil {
		if k := h.cfg.KeyForAction(action); k != "" {
			return k
		}
	}
	return defaultKey
}
func (h *HelpBuilder) formatKeyItem(item KeyItem) string {
	key := h.resolveKey(item.Action, item.DefaultKey)
	return fmt.Sprintf("%s: %s", key, item.Description)
}

// KeyItem represents a single key binding in help text
type KeyItem struct {
	Action      string
	DefaultKey  string
	Description string
}

// Build constructs help text from a list of key items
func (h *HelpBuilder) Build(items ...KeyItem) string {
	var parts []string
	for _, item := range items {
		parts = append(parts, h.formatKeyItem(item))
	}
	return strings.Join(parts, " • ")
}

// Truncate shortens strings to `max` characters, appending an ellipsis when truncated
func Truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

// FormatNameWithQty formats a name with quantity suffix
func FormatNameWithQty(name string, qty int) string {
	return fmt.Sprintf("%s x%d", name, qty)
}

// NewStyledTable creates a table with consistent styling
func NewStyledTable(columns []table.Column, height int, focused bool, styleManager *StyleManager) table.Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(height),
		table.WithFocused(focused),
	)
	s := styleManager.GetTableStyles()
	t.SetStyles(s)
	return t
}

type ConfigManagerProvider interface {
	GetConfigManager() *runtimecfg.Manager
}

func ResolveKeyBinding(cfg *runtimecfg.Manager, action string, defaultKey string) string {
	if cfg != nil {
		if k := cfg.KeyForAction(action); k != "" {
			return k
		}
	}
	return defaultKey
}
func MatchActionOrDefault(cfg *runtimecfg.Manager, keyString string, fallback string) string {
	if cfg != nil {
		return cfg.MatchAction(keyString)
	}
	return fallback
}
func RenderProgressBar(percent float64, width int, sm *StyleManager) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	if width < 5 {
		width = 5
	}
	barWidth := width - 5
	if barWidth < 1 {
		barWidth = 1
	}
	filled := int(percent / 100.0 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled
	return sm.GetFocusedStyle().Render(strings.Repeat("█", filled)) + sm.GetBlurredStyle().Render(strings.Repeat("░", empty)) + fmt.Sprintf(" %3.0f%%", percent)
}
func RenderActiveTab(label string) string {
	return titleStyle.Copy().Padding(0, 2).Render("[ " + label + " ]")
}
func RenderInactiveTab(label string) string {
	return blurredStyle.Copy().Padding(0, 2).Render("  " + label + "  ")
}
func RenderTitle(title string) string {
	return titleStyle.Render(title) + "\n\n"
}
func RenderSectionTitle(title string) string {
	return titleStyle.Render(title) + "\n"
}
func RenderFocusedLabel(label string) string {
	return focusedStyle.Render(label)
}
func RenderBlurredLabel(label string) string {
	return blurredStyle.Render(label)
}
func RenderConditionalLabel(isFocused bool, label string) string {
	if isFocused {
		return RenderFocusedLabel(label)
	}
	return RenderBlurredLabel(label)
}
