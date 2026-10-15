package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/laiambryant/tui-cardman/internal/model"
	listservice "github.com/laiambryant/tui-cardman/internal/services/list"
	"github.com/laiambryant/tui-cardman/internal/services/prices"
)

type CardDetailModel struct {
	card            model.Card
	tcgPrices       []model.TCGPlayerPriceRow
	cardMarketPrice *model.CardMarketPriceRow
	listsContaining []model.UserList
	ownedQuantity   int
	visible         bool
	scroll          int
	maxScroll       int
	width           int
	height          int
	styleManager    *StyleManager
	tcgService      prices.TCGPlayerPriceService
	cmService       prices.CardMarketPriceService
	listService     listservice.ListService
	userID          int64
	renderers       []CardDetailRenderer
	extraData       any
}

type cardDetailLoadedMsg struct {
	tcgPrices       []model.TCGPlayerPriceRow
	cardMarketPrice *model.CardMarketPriceRow
	lists           []model.UserList
	extraData       any
}

func (m *CardDetailModel) Open(card model.Card, ownedQty int) tea.Cmd {
	m.card = card
	m.ownedQuantity = ownedQty
	m.visible = true
	m.scroll = 0
	m.tcgPrices = nil
	m.cardMarketPrice = nil
	m.listsContaining = nil
	m.extraData = nil
	return m.fetchDetailCmd(card.ID)
}

func (m *CardDetailModel) Close() {
	m.visible = false
}

func (m *CardDetailModel) fetchDetailCmd(cardID int64) tea.Cmd {
	tcgService, cmService, listService, userID := m.tcgService, m.cmService, m.listService, m.userID
	renderer := m.activeRenderer()
	return func() tea.Msg {
		var msg cardDetailLoadedMsg
		if tcgService != nil {
			msg.tcgPrices, _ = tcgService.GetLatestPricesForCard(cardID)
		}
		if cmService != nil {
			msg.cardMarketPrice, _ = cmService.GetLatestPriceForCard(cardID)
		}
		if listService != nil && userID > 0 {
			msg.lists, _ = listService.GetListsContainingCard(userID, cardID)
		}
		if renderer != nil {
			msg.extraData = renderer.FetchExtra(cardID)
		}
		return msg
	}
}

func (m *CardDetailModel) applyLoaded(msg cardDetailLoadedMsg) {
	m.tcgPrices = msg.tcgPrices
	m.cardMarketPrice = msg.cardMarketPrice
	m.listsContaining = msg.lists
	m.extraData = msg.extraData
}

func (m *CardDetailModel) handleKey(key string) {
	switch key {
	case "esc", "q":
		m.Close()
	case "up", "k":
		m.scroll = max(m.scroll-1, 0)
	case "down", "j":
		m.scroll = min(m.scroll+1, m.maxScroll)
	}
}

func (m *CardDetailModel) activeRenderer() CardDetailRenderer {
	for _, r := range m.renderers {
		if r.CanRender(m.card.CardGameID) {
			return r
		}
	}
	return nil
}

func (m *CardDetailModel) View() string {
	if !m.visible {
		return ""
	}
	renderer := m.activeRenderer()
	if renderer != nil && m.width >= 70 {
		return m.renderWideView(renderer)
	}
	return m.renderModalView()
}

// renderWideView renders the card detail as a full-screen framed view (header/body/footer).
func (m *CardDetailModel) renderWideView(renderer CardDetailRenderer) string {
	sm := m.styleManager

	// Header: card name
	header := RenderBreadcrumbs(sm, "Card Detail", m.card.Name)

	// Footer: help text
	footer := JoinHelpHints(
		RenderHelpHint("Esc", "Close"),
		RenderHelpHint("↑/↓", "Scroll"),
	)

	// Compute layout
	layout := calculateFrameLayout(lipgloss.Height(header), lipgloss.Height(footer), m.width, m.height)

	leftHalf := layout.ContentWidth / 2
	rightHalf := layout.ContentWidth - leftHalf

	// Left panel: game-specific renderer, fills full body height
	leftPanel := renderer.RenderLeft(m.card, m.extraData, leftHalf, layout.BodyContentHeight, sm)

	// Right panel: scrollable card details
	// RenderPanel with padX=1, padY=0, border=2: inner width = rightHalf-4, inner height = BodyContentHeight-2
	rightInnerWidth := max(rightHalf-4, 10)
	rightInnerHeight := max(layout.BodyContentHeight-2, 1)
	rightContent := m.renderContent(rightInnerWidth)
	rightLines := strings.Split(rightContent, "\n")
	m.maxScroll = max(0, len(rightLines)-rightInnerHeight)
	if m.scroll > m.maxScroll {
		m.scroll = m.maxScroll
	}
	end := min(m.scroll+rightInnerHeight, len(rightLines))
	visibleRight := strings.Join(rightLines[m.scroll:end], "\n")
	rightPanel := RenderPanel(sm, visibleRight, rightHalf, layout.BodyContentHeight, true, 1, 0)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	return renderFramedViewWithLayout(header, body, footer, layout, sm)
}

// renderModalView is the fallback for narrow terminals or cards without a renderer.
func (m *CardDetailModel) renderModalView() string {
	modalWidth := 50
	if m.width > 0 {
		modalWidth = min(50, m.width-10)
	}
	if modalWidth < 30 {
		modalWidth = 30
	}
	innerWidth := modalWidth - 6
	content := m.renderContent(innerWidth)
	lines := strings.Split(content, "\n")
	modalHeight := min(m.height-4, 25)
	if modalHeight < 10 {
		modalHeight = 10
	}
	visibleLines := modalHeight - 4
	m.maxScroll = max(0, len(lines)-visibleLines)
	if m.scroll > m.maxScroll {
		m.scroll = m.maxScroll
	}
	end := min(m.scroll+visibleLines, len(lines))
	visible := strings.Join(lines[m.scroll:end], "\n")
	modalStyle := m.styleManager.Box(m.styleManager.scheme.Focused, modalWidth, modalHeight, 0, 2, 1)
	modal := modalStyle.Render(visible)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m *CardDetailModel) renderContent(width int) string {
	var b strings.Builder
	sm := m.styleManager
	ts := sm.GetSecondaryStyle().Bold(true)
	ns := sm.GetNoStyle()
	bs := sm.GetBlurredStyle()
	field := func(label, value string) string {
		return bs.Render(fmt.Sprintf("%-8s", label)) + ns.Render(value) + "\n"
	}
	b.WriteString(sm.GetTitleStyle().Bold(true).Render(m.card.Name) + "\n")
	b.WriteString(sm.GetSubtleStyle().Render(strings.Repeat("─", min(width, len(m.card.Name)+4))) + "\n")
	setName := "Unknown"
	if m.card.Set != nil {
		setName = m.card.Set.Name
	}
	b.WriteString(field("Set", setName))
	b.WriteString(field("Number", m.card.Number))
	b.WriteString(bs.Render(fmt.Sprintf("%-8s", "Rarity")) + renderRarityText(sm, m.card.Rarity) + "\n")
	if m.card.Artist != "" {
		b.WriteString(field("Artist", m.card.Artist))
	}
	b.WriteString(field("Owned", fmt.Sprintf("%d", m.ownedQuantity)))
	b.WriteString("\n")
	if len(m.tcgPrices) > 0 {
		b.WriteString(ts.Render("TCGPlayer Prices") + "\n")
		for _, p := range m.tcgPrices {
			if p.Market > 0 {
				b.WriteString(bs.Render(fmt.Sprintf("  %-15s ", p.PriceType)) + sm.GetSuccessStyle().Render(fmt.Sprintf("$%.2f", p.Market)) + bs.Render(fmt.Sprintf(" (low $%.2f / high $%.2f)", p.Low, p.High)) + "\n")
			}
		}
		b.WriteString("\n")
	}
	if m.cardMarketPrice != nil && (m.cardMarketPrice.AvgPrice > 0 || m.cardMarketPrice.TrendPrice > 0) {
		b.WriteString(ts.Render("CardMarket Prices") + "\n")
		if m.cardMarketPrice.AvgPrice > 0 {
			b.WriteString(bs.Render("  Average  ") + sm.GetSuccessStyle().Render(fmt.Sprintf("$%.2f", m.cardMarketPrice.AvgPrice)) + "\n")
		}
		if m.cardMarketPrice.TrendPrice > 0 {
			b.WriteString(bs.Render("  Trend    ") + sm.GetSuccessStyle().Render(fmt.Sprintf("$%.2f", m.cardMarketPrice.TrendPrice)) + "\n")
		}
		b.WriteString("\n")
	}
	if len(m.tcgPrices) == 0 && m.cardMarketPrice == nil {
		b.WriteString(bs.Render("No price data available.") + "\n\n")
	}
	if len(m.listsContaining) > 0 {
		b.WriteString(ts.Render("In Lists") + "\n")
		for _, l := range m.listsContaining {
			b.WriteString(ns.Render(fmt.Sprintf("  %s %s", ListSymbol, l.Name)) + "\n")
		}
	}
	b.WriteString("\n" + JoinHelpHints(RenderHelpHint("Esc", "Close"), RenderHelpHint("↑/↓", "Scroll")))
	return b.String()
}

// renderRarityText renders a rarity name tinted with its rarity color.
func renderRarityText(sm *StyleManager, rarity string) string {
	if rarity == "" {
		return sm.GetBlurredStyle().Render("—")
	}
	color, ok := rarityColors[rarity]
	if !ok {
		return sm.GetNoStyle().Render(rarity)
	}
	return sm.applyBGFG(lipgloss.NewStyle().Foreground(color).Inline(true)).Render(rarityIconFor(rarity) + " " + rarity)
}

func rarityIconFor(rarity string) string {
	lower := strings.ToLower(rarity)
	switch {
	case strings.Contains(lower, "ultra") || strings.Contains(lower, "secret") || strings.Contains(lower, "hyper"):
		return UltraRareIcon
	case strings.Contains(lower, "rare"):
		return RareIcon
	case strings.Contains(lower, "uncommon"):
		return UncommonIcon
	default:
		return CommonIcon
	}
}
