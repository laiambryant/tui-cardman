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
	return func() tea.Msg {
		var tcgPrices []model.TCGPlayerPriceRow
		var cmPrice *model.CardMarketPriceRow
		var lists []model.UserList
		if m.tcgService != nil {
			tcgPrices, _ = m.tcgService.GetLatestPricesForCard(cardID)
		}
		if m.cmService != nil {
			cmPrice, _ = m.cmService.GetLatestPriceForCard(cardID)
		}
		if m.listService != nil && m.userID > 0 {
			lists, _ = m.listService.GetListsContainingCard(m.userID, cardID)
		}
		var extraData any
		for _, r := range m.renderers {
			if r.CanRender(m.card.CardGameID) {
				extraData = r.FetchExtra(cardID)
				break
			}
		}
		return cardDetailLoadedMsg{
			tcgPrices:       tcgPrices,
			cardMarketPrice: cmPrice,
			lists:           lists,
			extraData:       extraData,
		}
	}
}

func (m *CardDetailModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case cardDetailLoadedMsg:
		m.tcgPrices = msg.tcgPrices
		m.cardMarketPrice = msg.cardMarketPrice
		m.listsContaining = msg.lists
		m.extraData = msg.extraData
		return nil
	case tea.KeyMsg:
		s := msg.String()
		if s == "esc" || s == "q" {
			m.Close()
			return nil
		}
		if s == "up" || s == "k" {
			if m.scroll > 0 {
				m.scroll--
			}
			return nil
		}
		if s == "down" || s == "j" {
			if m.scroll < m.maxScroll {
				m.scroll++
			}
			return nil
		}
	}
	return nil
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
	header := sm.GetTitleStyle().Render(m.card.Name)

	// Footer: help text
	footer := sm.GetBlurredStyle().Render("Esc: Close • ↑/↓: Scroll")

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
	ts := m.styleManager.GetTitleStyle()
	ns := m.styleManager.GetNoStyle()
	bs := m.styleManager.GetBlurredStyle()
	b.WriteString(ts.Render(m.card.Name) + "\n")
	b.WriteString(strings.Repeat("─", min(width, len(m.card.Name)+4)) + "\n")
	setName := "Unknown"
	if m.card.Set != nil {
		setName = m.card.Set.Name
	}
	b.WriteString(ns.Render(fmt.Sprintf("Set:    %s", setName)) + "\n")
	b.WriteString(ns.Render(fmt.Sprintf("Number: %s", m.card.Number)) + "\n")
	b.WriteString(ns.Render(fmt.Sprintf("Rarity: %s", m.card.Rarity)) + "\n")
	if m.card.Artist != "" {
		b.WriteString(ns.Render(fmt.Sprintf("Artist: %s", m.card.Artist)) + "\n")
	}
	b.WriteString(ns.Render(fmt.Sprintf("Owned:  %d", m.ownedQuantity)) + "\n")
	b.WriteString("\n")
	if len(m.tcgPrices) > 0 {
		b.WriteString(ts.Render("TCGPlayer Prices") + "\n")
		for _, p := range m.tcgPrices {
			if p.Market > 0 {
				b.WriteString(ns.Render(fmt.Sprintf("  %-15s $%.2f (low $%.2f / high $%.2f)", p.PriceType, p.Market, p.Low, p.High)) + "\n")
			}
		}
		b.WriteString("\n")
	}
	if m.cardMarketPrice != nil && (m.cardMarketPrice.AvgPrice > 0 || m.cardMarketPrice.TrendPrice > 0) {
		b.WriteString(ts.Render("CardMarket Prices") + "\n")
		if m.cardMarketPrice.AvgPrice > 0 {
			b.WriteString(ns.Render(fmt.Sprintf("  Average: $%.2f", m.cardMarketPrice.AvgPrice)) + "\n")
		}
		if m.cardMarketPrice.TrendPrice > 0 {
			b.WriteString(ns.Render(fmt.Sprintf("  Trend:   $%.2f", m.cardMarketPrice.TrendPrice)) + "\n")
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
	b.WriteString("\n" + bs.Render("Esc: Close • ↑/↓: Scroll"))
	return b.String()
}
