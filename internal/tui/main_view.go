package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var modeLabels = []string{
	"My Collection",
	"My Lists",
	"My Decks",
	"Import Sets",
}

func (m Model) mainView() string {
	header := m.renderMainHeader()
	footer := m.renderMainFooter()
	layout := calculateFrameLayout(lipgloss.Height(header), lipgloss.Height(footer), m.width, m.height)
	body := m.renderSplitMainBody(layout.ContentWidth, layout.BodyContentHeight)
	return renderFramedViewWithLayout(header, body, footer, layout, m.styleManager)
}

func (m Model) renderMainHeader() string {
	if m.user != nil {
		return titleStyle.Render(fmt.Sprintf("CardMan — Welcome, %s %s!", m.user.Name, m.user.Surname))
	}
	return titleStyle.Render("CardMan — Card Management TUI")
}

func (m Model) renderMainFooter() string {
	var b strings.Builder
	if m.errorMsg != "" {
		b.WriteString(errorStyle.Render("Error: "+m.errorMsg) + "\n")
	}
	b.WriteString(m.renderMainMenuHelp())
	return b.String()
}

func (m Model) renderSplitMainBody(contentWidth, contentHeight int) string {
	if contentWidth <= 0 || contentHeight <= 0 {
		return ""
	}
	leftWidth := contentWidth * 40 / 100
	rightWidth := contentWidth - leftWidth
	leftPanel := m.renderCardGamesPanel(leftWidth, contentHeight)
	rightPanel := m.renderRightPanel(rightWidth, contentHeight)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m Model) renderCardGamesPanel(width, height int) string {
	var b strings.Builder
	b.WriteString(renderPanelHeading(m.styleManager, "Card Games", m.mainFocusPanel == 0))
	btnMaxWidth := panelButtonMaxWidth(width)
	if len(m.cardGames) == 0 {
		b.WriteString(errorStyle.Render("No card games found.") + "\n")
	} else {
		for i, game := range m.cardGames {
			focused := m.cursor == i && m.mainFocusPanel == 0
			b.WriteString(RenderButtonItem(m.styleManager, game.Name, focused, btnMaxWidth))
		}
	}
	return RenderPanel(m.styleManager, b.String(), width, height, m.mainFocusPanel == 0, 2, 1)
}

// panelButtonMaxWidth derives a button width from the panel width, accounting for
// panel border and padding overhead on each axis.
func panelButtonMaxWidth(panelWidth int) int {
	innerWidth := max(panelWidth-6, 10)
	return max(innerWidth-4, 6)
}

// renderRightPanel renders the right side of the home screen split into
// top (mode selection) and bottom (stats).
func (m Model) renderRightPanel(width, height int) string {
	btnMaxWidth := panelButtonMaxWidth(width)
	topContent := m.renderModesSection(btnMaxWidth)
	bottomContent := m.renderStatsSection()
	body := lipgloss.JoinVertical(lipgloss.Left, topContent, bottomContent)
	return RenderPanel(m.styleManager, body, width, height, m.mainFocusPanel == 1, 2, 1)
}

func (m Model) renderModesSection(btnMaxWidth int) string {
	var b strings.Builder
	b.WriteString(renderPanelHeading(m.styleManager, "Select Mode", m.mainFocusPanel == 1))
	for i, label := range modeLabels {
		focused := m.rightPanelCursor == i && m.mainFocusPanel == 1
		b.WriteString(RenderButtonItem(m.styleManager, label, focused, btnMaxWidth))
	}
	return b.String()
}

func (m Model) renderStatsSection() string {
	sm := m.styleManager
	var b strings.Builder
	b.WriteString(renderPanelHeading(sm, "Stats", false))
	if placeholder, done := m.renderStatsPlaceholder(); done {
		b.WriteString(placeholder)
		return b.String()
	}
	b.WriteString(sm.GetHelpStyle().Render(m.cardGames[m.cursor].Name) + "\n\n")
	if m.statsLoading {
		b.WriteString(sm.GetHelpStyle().Render("Loading stats...") + "\n")
		return b.String()
	}
	b.WriteString(m.renderStatLines())
	return b.String()
}

// renderStatsPlaceholder returns an explanatory message when stats cannot be shown yet.
func (m Model) renderStatsPlaceholder() (string, bool) {
	hs := m.styleManager.GetHelpStyle()
	if m.user == nil {
		return hs.Render("Login to see stats.") + "\n", true
	}
	if len(m.cardGames) == 0 {
		return hs.Render("No card games available.") + "\n", true
	}
	if m.cursor >= len(m.cardGames) {
		return hs.Render("Select a game to see stats.") + "\n", true
	}
	return "", false
}

func (m Model) renderStatLines() string {
	if m.gameStats == nil {
		return m.styleManager.GetHelpStyle().Render("Select a game to see stats.") + "\n"
	}
	s := m.gameStats
	var b strings.Builder
	b.WriteString(m.renderStatLine("Cards owned", fmt.Sprintf("%d", s.TotalCardsOwned)))
	b.WriteString(m.renderStatLine("Lists", fmt.Sprintf("%d", s.ListCount)))
	b.WriteString(m.renderStatLine("Decks", fmt.Sprintf("%d", s.DeckCount)))
	if s.CollectionValue > 0 {
		b.WriteString(m.renderStatLine("Collection value", fmt.Sprintf("$%.2f", s.CollectionValue)))
	}
	if s.TotalSets > 0 {
		b.WriteString(m.renderStatLine("Sets complete", fmt.Sprintf("%d / %d", s.SetsComplete, s.TotalSets)))
	}
	return b.String()
}

func (m Model) renderStatLine(label, value string) string {
	return m.styleManager.GetHelpStyle().Render(label+": ") + m.styleManager.GetTitleStyle().Render(value) + "\n"
}

func (m Model) renderMainMenuHelp() string {
	hb := NewHelpBuilder(m.configManager)
	return helpStyle.Render(joinHelp(
		hb.Build(KeyItem{"settings", "F1", "Settings"}),
		"←/→: Switch Panel",
		hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate"),
		hb.Build(
			KeyItem{"select", "Enter", "Select"},
			KeyItem{"quit", "Ctrl+C", "Quit"},
		),
	))
}
