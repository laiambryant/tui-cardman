package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) mainView() string {
	header := m.renderMainHeader()
	footer := m.renderMainFooter()
	layout := calculateFrameLayout(lipgloss.Height(header), lipgloss.Height(footer), m.width, m.height)
	body := m.renderSplitMainBody(layout.ContentWidth, layout.BodyContentHeight)
	return renderFramedViewWithLayout(header, body, footer, layout, m.styleManager)
}

func (m *Model) renderMainHeader() string {
	brand := RenderBrand(m.styleManager)
	divider := subtleStyle.Render("  │  ")
	if m.user != nil {
		return brand + divider + helpStyle.Render(fmt.Sprintf("Welcome back, %s %s", m.user.Name, m.user.Surname))
	}
	return brand + divider + helpStyle.Render("Card Collection Manager")
}

func (m *Model) renderMainFooter() string {
	var b strings.Builder
	if m.errorMsg != "" {
		b.WriteString(errorStyle.Render("Error: "+m.errorMsg) + "\n")
	}
	b.WriteString(m.renderMainMenuHelp())
	return b.String()
}

func (m *Model) renderSplitMainBody(contentWidth, contentHeight int) string {
	if contentWidth <= 0 || contentHeight <= 0 {
		return ""
	}
	leftWidth := contentWidth * 40 / 100
	rightWidth := contentWidth - leftWidth

	leftPanel := m.renderCardGamesPanel(leftWidth, contentHeight)
	rightPanel := m.renderRightPanel(rightWidth, contentHeight)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m *Model) renderCardGamesPanel(width, height int) string {
	var b strings.Builder
	innerWidth := max(width-6, 10)
	btnMaxWidth := max(innerWidth-4, 6)
	if len(m.cardGames) == 0 {
		b.WriteString(errorStyle.Render("No card games found.") + "\n")
	} else {
		for i, game := range m.cardGames {
			focused := m.cursor == i && m.mainFocusPanel == 0
			b.WriteString(RenderButtonItem(m.styleManager, game.Name, focused, btnMaxWidth))
		}
	}
	return RenderTitledPanel(m.styleManager, "Card Games", b.String(), width, height, m.mainFocusPanel == 0, 2, 1)
}

// renderRightPanel renders the right side of the home screen split into
// top (mode selection) and bottom (stats).
func (m *Model) renderRightPanel(width, height int) string {
	// Split right panel: top ~45% for modes, bottom ~55% for stats
	// We render them as a vertical join inside a single outer panel.
	// inner area (accounting for panel border+padding: 2 border + 2 pad each side = 6 per axis)
	innerWidth := max(width-6, 10)
	btnMaxWidth := max(innerWidth-4, 6)

	topHeight := height * 45 / 100
	bottomHeight := height - topHeight

	topContent := m.renderModesSection(innerWidth, btnMaxWidth, topHeight)
	bottomContent := m.renderStatsSection(innerWidth, bottomHeight)

	body := lipgloss.JoinVertical(lipgloss.Left, topContent, bottomContent)
	return RenderTitledPanel(m.styleManager, "Browse", body, width, height, m.mainFocusPanel == 1, 2, 1)
}

var modeLabels = []string{
	"My Collection",
	"My Lists",
	"My Decks",
	"Import Sets",
}

func (m *Model) renderModesSection(innerWidth, btnMaxWidth, availHeight int) string {
	var b strings.Builder
	b.WriteString(secondaryStyle.Bold(true).Render("Select Mode") + "\n")
	for i, label := range modeLabels {
		focused := m.rightPanelCursor == i && m.mainFocusPanel == 1
		b.WriteString(RenderButtonItem(m.styleManager, label, focused, btnMaxWidth))
	}
	_ = availHeight // used for future trimming if needed
	return b.String()
}

func (m *Model) renderStatsSection(innerWidth, availHeight int) string {
	_ = availHeight
	sm := m.styleManager

	var b strings.Builder
	b.WriteString(secondaryStyle.Bold(true).Render("Stats") + "\n")

	if m.user == nil {
		b.WriteString(sm.GetHelpStyle().Render("Login to see stats.") + "\n")
		return b.String()
	}

	if len(m.cardGames) == 0 {
		b.WriteString(sm.GetHelpStyle().Render("No card games available.") + "\n")
		return b.String()
	}

	// Show which game the stats are for
	if m.cursor < len(m.cardGames) {
		gameName := m.cardGames[m.cursor].Name
		b.WriteString(RenderStatusBadge(sm, "info", gameName) + "\n\n")
	}

	if m.statsLoading {
		b.WriteString(sm.GetHelpStyle().Render("Loading stats...") + "\n")
		return b.String()
	}

	if m.gameStats == nil {
		b.WriteString(sm.GetHelpStyle().Render("Select a game to see stats.") + "\n")
		return b.String()
	}

	s := m.gameStats
	labelStyle := sm.GetHelpStyle()
	valueStyle := sm.GetTitleStyle().Bold(true)

	line := func(label, value string) string {
		return labelStyle.Render(fmt.Sprintf("%-18s", label)) + valueStyle.Render(value) + "\n"
	}

	b.WriteString(line("Cards owned", fmt.Sprintf("%d", s.TotalCardsOwned)))
	b.WriteString(line("Lists", fmt.Sprintf("%d", s.ListCount)))
	b.WriteString(line("Decks", fmt.Sprintf("%d", s.DeckCount)))
	if s.CollectionValue > 0 {
		b.WriteString(labelStyle.Render(fmt.Sprintf("%-18s", "Collection value")) + sm.GetSuccessStyle().Bold(true).Render(fmt.Sprintf("$%.2f", s.CollectionValue)) + "\n")
	}
	if s.TotalSets > 0 {
		percent := float64(s.SetsComplete) / float64(s.TotalSets) * 100
		barWidth := min(max(innerWidth-4, 10), 30)
		b.WriteString(line("Sets complete", fmt.Sprintf("%d / %d", s.SetsComplete, s.TotalSets)))
		b.WriteString(RenderProgressBar(percent, barWidth, sm) + "\n")
	}

	return b.String()
}

func (m *Model) renderMainMenuHelp() string {
	hb := NewHelpBuilder(m.configManager)
	return JoinHelpHints(
		hb.Build(KeyItem{"settings", "F1", "Settings"}),
		RenderHelpHint("←/→", "Switch panel"),
		hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate"),
		hb.Build(
			KeyItem{"select", "Enter", "Select"},
			KeyItem{"quit", "Ctrl+C", "Quit"},
		),
	)
}
