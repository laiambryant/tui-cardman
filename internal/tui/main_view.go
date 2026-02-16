package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) mainView() string {
	header := m.renderMainHeader()
	footer := m.renderMainFooter()
	layout := calculateFrameLayout(lipgloss.Height(header), lipgloss.Height(footer), m.width, m.height)
	body := m.renderSplitMainBody(layout.ContentWidth, layout.BodyContentHeight)
	return renderFramedViewWithLayout(header, body, footer, layout, m.styleManager)
}

func (m Model) renderMainHeader() string {
	if m.user != nil {
		return titleStyle.Render(fmt.Sprintf("CardMan - Welcome, %s %s!", m.user.Name, m.user.Surname))
	}
	return titleStyle.Render("CardMan - Card Management TUI")
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
	rightPanel := m.renderImportPanelMain(rightWidth, contentHeight)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m Model) renderCardGamesPanel(width, height int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Card Games") + "\n")
	if len(m.cardGames) == 0 {
		b.WriteString(errorStyle.Render("No card games found.") + "\n")
	} else {
		for i, game := range m.cardGames {
			b.WriteString(RenderListItem(game.Name, m.cursor == i && m.mainFocusPanel == 0))
		}
	}
	return RenderPanel(m.styleManager, b.String(), width, height, m.mainFocusPanel == 0, 2, 1)
}

func (m Model) renderImportPanelMain(width, height int) string {
	if m.importModel == nil {
		return ""
	}
	return m.importModel.renderImportPanel(width, height, m.mainFocusPanel == 1)
}

func (m Model) renderMainMenuHelp() string {
	hb := NewHelpBuilder(m.configManager)
	help := hb.Build(KeyItem{"settings", "F1", "Settings"}) + " • Left/Right: Switch Panel • " + hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate") + " • " + hb.Build(
		KeyItem{"select", "Enter", "Select"},
		KeyItem{"quit", "Ctrl+C", "Quit"},
	)
	return helpStyle.Render(help)
}
