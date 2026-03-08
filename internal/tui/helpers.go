package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/laiambryant/tui-cardman/internal/export"
	"github.com/laiambryant/tui-cardman/internal/model"
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

func (h *HelpBuilder) resolveKey(action, defaultKey string) string {
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

func (h *HelpBuilder) Pair(action1, default1, action2, default2, description string) string {
	k1 := h.resolveKey(action1, default1)
	k2 := h.resolveKey(action2, default2)
	return fmt.Sprintf("%s/%s: %s", k1, k2, description)
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

func MatchActionOrDefault(cfg *runtimecfg.Manager, keyString, fallback string) string {
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
	return strings.Repeat("█", filled) + strings.Repeat("░", empty) + fmt.Sprintf(" %3.0f%%", percent)
}

func RenderPanel(sm *StyleManager, content string, width, height int, focused bool, padX, padY int) string {
	borderColor := sm.scheme.Blurred
	if focused {
		borderColor = sm.scheme.Focused
	}
	borderOverhead := 2
	innerWidth := max(width-borderOverhead-padX*2, 1)
	innerHeight := max(height-borderOverhead-padY*2, 1)
	return sm.Box(borderColor, innerWidth, innerHeight, 0, padX, padY).Render(content)
}

func RenderTabBar(sm *StyleManager, tabs []string, activeIndex int) string {
	var rendered []string
	for i, tab := range tabs {
		if i == activeIndex {
			rendered = append(rendered, sm.GetTitleStyle().Inline(true).Render("[ "+tab+" ]"))
		} else {
			rendered = append(rendered, sm.GetBlurredStyle().Inline(true).Render("  "+tab+"  "))
		}
	}
	return strings.Join(rendered, sm.GetNoStyle().Render(" "))
}

func RenderFramedWithModal(header, footer string, bodyFn func(availableHeight int) string, width, height int, sm *StyleManager, modal *ModalModel) string {
	layout := calculateFrameLayout(lipgloss.Height(header), lipgloss.Height(footer), width, height)
	body := bodyFn(layout.BodyContentHeight)
	content := renderFramedViewWithLayout(header, body, footer, layout, sm)
	if modal != nil && modal.IsVisible() {
		return modal.View()
	}
	return content
}

func CalcTableHeight(availableHeight, headerLines, minHeight int) int {
	if availableHeight <= 0 {
		return max(10, minHeight)
	}
	return max(availableHeight-headerLines, minHeight)
}

func RenderButton(isFocused bool, label string) string {
	if isFocused {
		return titleStyle.Render(label)
	}
	return blurredStyle.Render(label)
}

func RenderButtonItem(sm *StyleManager, label string, isSelected bool, maxWidth int) string {
	labelWidth := lipgloss.Width(label)
	// Width() sets content area; border(1+1) + padding(1+1) = 4 overhead on top.
	// So content width = maxWidth - 4, but never narrower than the label itself.
	btnWidth := max(maxWidth-4, labelWidth)
	if isSelected {
		return sm.applyBGFG(lipgloss.NewStyle().
			Width(btnWidth).
			Align(lipgloss.Center).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(sm.scheme.Focused).
			Foreground(sm.scheme.Focused).
			Bold(true).
			Padding(0, 1)).Render(label) + "\n"
	}
	return sm.applyBGFG(lipgloss.NewStyle().
		Width(btnWidth).
		Align(lipgloss.Center).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(sm.scheme.Blurred).
		Padding(0, 1)).Render(label) + "\n"
}

func GetAction(cfg *runtimecfg.Manager, s string) string {
	return MatchActionOrDefault(cfg, s, "")
}

func RenderListItem(line string, isSelected bool) string {
	prefix := getCursorPrefix(isSelected)
	if isSelected {
		return titleStyle.Render(prefix+line) + "\n"
	}
	return blurredStyle.Render(prefix+line) + "\n"
}

func RenderConditionalLabel(isFocused bool, label string) string {
	if isFocused {
		return focusedStyle.Render(label)
	}
	return blurredStyle.Render(label)
}

// filterCardsByQuery filters a card slice by name, number, or rarity (case-insensitive).
// Returns the original slice unchanged when query is empty.
func filterCardsByQuery(cards []model.Card, query string) []model.Card {
	if query == "" {
		return cards
	}
	q := strings.ToLower(query)
	var filtered []model.Card
	for _, c := range cards {
		if strings.Contains(strings.ToLower(c.Name), q) ||
			strings.Contains(strings.ToLower(c.Number), q) ||
			strings.Contains(strings.ToLower(c.Rarity), q) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// buildCardExportRows produces a CSV-ready row slice from a card list and quantity maps.
// Cards with a combined quantity of zero or below are omitted.
func buildCardExportRows(cards []model.Card, dbQtys, tempDeltas map[int64]int) []export.CardRow {
	var rows []export.CardRow
	for _, c := range cards {
		qty := dbQtys[c.ID] + tempDeltas[c.ID]
		if qty <= 0 {
			continue
		}
		setName, setCode := "", ""
		if c.Set != nil {
			setName = c.Set.Name
			setCode = c.Set.Code
		}
		rows = append(rows, export.CardRow{
			Name:     c.Name,
			SetName:  setName,
			SetCode:  setCode,
			Number:   c.Number,
			Rarity:   c.Rarity,
			Quantity: qty,
		})
	}
	return rows
}
