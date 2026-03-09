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
	return fmt.Sprintf("%s / %s: %s", k1, k2, description)
}

// Build constructs help text from a list of key items
func (h *HelpBuilder) Build(items ...KeyItem) string {
	var parts []string
	for _, item := range items {
		parts = append(parts, h.formatKeyItem(item))
	}
	return strings.Join(parts, " | ")
}

// isModifierKey returns true for non-printable / modifier key combos that should
// always be routed to shortcut handlers even when a search textinput is focused.
// Printable single characters return false and should be forwarded to the textinput.
func isModifierKey(s string) bool {
	if strings.HasPrefix(s, "ctrl+") || strings.HasPrefix(s, "alt+") {
		return true
	}
	switch s {
	case "tab", "shift+tab", "enter", "\r", "\n",
		"esc", "up", "down", "left", "right",
		"home", "end",
		"pgup", "pgdown", "f1", "f2", "f3", "f4",
		"f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12":
		return true
	}
	return false
}

type ColumnDef struct {
	Key        string
	Title      string
	Proportion int
	MinWidth   int
}

var CardSearchColumns = []ColumnDef{
	{"name", "Name", 37, 8},
	{"expansion", "Expansion", 22, 5},
	{"rarity", "Rarity", 18, 5},
	{"number", "Card #", 12, 4},
	{"quantity", "Quantity", 11, 3},
	{"artist", "Artist", 15, 5},
}

var CollectionColumns = []ColumnDef{
	{"name", "Name", 42, 8},
	{"expansion", "Expansion", 25, 5},
	{"rarity", "Rarity", 20, 5},
	{"quantity", "Amount", 13, 3},
	{"artist", "Artist", 15, 5},
}

var DeckColumns = []ColumnDef{
	{"name", "Name", 38, 8},
	{"expansion", "Set", 23, 4},
	{"rarity", "Rarity", 19, 4},
	{"number", "#", 11, 3},
	{"quantity", "Qty", 9, 3},
	{"artist", "Artist", 15, 5},
}

type VisibleColumnSet struct {
	Columns []table.Column
	Keys    []string
	Widths  map[string]int
}

func BuildVisibleColumnSet(allCols []ColumnDef, visible map[string]bool, order []string, availableWidth int) VisibleColumnSet {
	active := filterActiveDefsOrdered(allCols, visible, order)
	cellPadding := len(active) * 2
	usable := max(availableWidth-cellPadding, len(active)*3)
	totalProportion := sumProportions(active)
	widths := distributeWidths(active, usable, totalProportion)
	return buildColumnSet(active, widths)
}

func filterActiveDefsOrdered(allCols []ColumnDef, visible map[string]bool, order []string) []ColumnDef {
	defMap := make(map[string]ColumnDef, len(allCols))
	for _, c := range allCols {
		defMap[c.Key] = c
	}
	var active []ColumnDef
	added := make(map[string]bool)
	for _, key := range order {
		if !visible[key] {
			continue
		}
		if def, ok := defMap[key]; ok {
			active = append(active, def)
			added[key] = true
		}
	}
	for _, c := range allCols {
		if visible[c.Key] && !added[c.Key] {
			active = append(active, c)
		}
	}
	return active
}

func sumProportions(cols []ColumnDef) int {
	total := 0
	for _, c := range cols {
		total += c.Proportion
	}
	return total
}

func distributeWidths(cols []ColumnDef, usable, totalProportion int) []int {
	widths := make([]int, len(cols))
	remaining := usable
	for i, c := range cols {
		if i == len(cols)-1 {
			widths[i] = max(remaining, c.MinWidth)
			break
		}
		widths[i] = max(usable*c.Proportion/totalProportion, c.MinWidth)
		remaining -= widths[i]
	}
	return widths
}

func buildColumnSet(active []ColumnDef, widths []int) VisibleColumnSet {
	vcs := VisibleColumnSet{
		Columns: make([]table.Column, len(active)),
		Keys:    make([]string, len(active)),
		Widths:  make(map[string]int, len(active)),
	}
	for i, c := range active {
		vcs.Columns[i] = table.Column{Title: c.Title, Width: widths[i]}
		vcs.Keys[i] = c.Key
		vcs.Widths[c.Key] = widths[i]
	}
	return vcs
}

func (vcs VisibleColumnSet) BuildRow(data map[string]string) table.Row {
	row := make(table.Row, len(vcs.Keys))
	for i, key := range vcs.Keys {
		row[i] = Truncate(data[key], vcs.Widths[key])
	}
	return row
}

func CardToDataMap(card model.Card, dbQty, tempDelta int) map[string]string {
	setDisplay := ""
	if card.Set != nil {
		setDisplay = card.Set.Name
	} else if card.SetID > 0 {
		setDisplay = fmt.Sprintf("Set#%d", card.SetID)
	}
	return map[string]string{
		"name":      card.Name,
		"expansion": setDisplay,
		"rarity":    card.Rarity,
		"number":    card.Number,
		"quantity":  fmt.Sprintf("%d", dbQty+tempDelta),
		"artist":    card.Artist,
	}
}

func CollectionToDataMap(c model.UserCollection) map[string]string {
	name := "Unknown Card"
	setDisplay := ""
	rarity := ""
	artist := ""
	if c.Card != nil {
		name = c.Card.Name
		if c.Card.Set != nil {
			setDisplay = c.Card.Set.Name
		} else if c.Card.SetID > 0 {
			setDisplay = fmt.Sprintf("Set#%d", c.Card.SetID)
		}
		rarity = c.Card.Rarity
		artist = c.Card.Artist
	}
	return map[string]string{
		"name":      name,
		"expansion": setDisplay,
		"rarity":    rarity,
		"quantity":  fmt.Sprintf("%d", c.Quantity),
		"artist":    artist,
	}
}

func GetVisibleColumns(cfg *runtimecfg.Manager) map[string]bool {
	if cfg == nil {
		return runtimecfg.DefaultVisibleColumns()
	}
	c := cfg.Get()
	if c.UI.VisibleColumns == nil {
		return runtimecfg.DefaultVisibleColumns()
	}
	return c.UI.VisibleColumns
}

func GetColumnOrder(cfg *runtimecfg.Manager) []string {
	if cfg == nil {
		return runtimecfg.DefaultColumnOrder()
	}
	c := cfg.Get()
	if len(c.UI.ColumnOrder) == 0 {
		return runtimecfg.DefaultColumnOrder()
	}
	return c.UI.ColumnOrder
}

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

// filterCardsByQuerySubstring filters a card slice by substring match (kept for benchmarks).
func filterCardsByQuerySubstring(cards []model.Card, query string) []model.Card {
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

// filterCardsByQuery filters a card slice using fuzzy matching on name, number, and rarity.
// Returns the original slice unchanged when query is empty.
func filterCardsByQuery(cards []model.Card, query string) []model.Card {
	if query == "" {
		return cards
	}
	results := fuzzySearchCards(cards, query)
	return fuzzyResultsToCards(results)
}

func renderSplitCardPanel(sm *StyleManager, topContent string, topFocused bool, bottomContent string, bottomFocused bool, width, topHeight, bottomHeight int) string {
	topPanel := RenderPanel(sm, topContent, width, topHeight, topFocused, 1, 0)
	bottomPanel := RenderPanel(sm, bottomContent, width, bottomHeight, bottomFocused, 1, 0)
	return lipgloss.JoinVertical(lipgloss.Left, topPanel, bottomPanel)
}

func countNonZeroDeltas(deltas map[int64]int) int {
	count := 0
	for _, delta := range deltas {
		if delta != 0 {
			count++
		}
	}
	return count
}

func buildQuantityUpdates(dbQtys, tempDeltas map[int64]int) map[int64]int {
	updates := make(map[int64]int)
	for cardID, delta := range tempDeltas {
		updates[cardID] = dbQtys[cardID] + delta
	}
	return updates
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
