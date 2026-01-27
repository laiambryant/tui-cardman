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
