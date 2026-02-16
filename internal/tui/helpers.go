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
		*modal = modal.SetBackgroundContent(content)
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
