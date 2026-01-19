package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
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
		key := item.DefaultKey
		if h.cfg != nil {
			if k := h.cfg.KeyForAction(item.Action); k != "" {
				key = k
			}
		}
		parts = append(parts, fmt.Sprintf("%s: %s", key, item.Description))
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
func NewStyledTable(columns []table.Column, height int, focused bool) table.Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(height),
		table.WithFocused(focused),
	)

	// Apply consistent styling
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return t
}
