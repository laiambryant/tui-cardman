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
			prefix := getCursorPrefix(m.cursor == i)
			line := fmt.Sprintf("%s%s", prefix, game.Name)
			if m.cursor == i && m.mainFocusPanel == 0 {
				b.WriteString(titleStyle.Render(line) + "\n")
			} else {
				b.WriteString(blurredStyle.Render(line) + "\n")
			}
		}
	}

	borderColor := m.styleManager.scheme.Blurred
	if m.mainFocusPanel == 0 {
		borderColor = m.styleManager.scheme.Focused
	}

	innerWidth := width - 6
	if innerWidth < 1 {
		innerWidth = 1
	}
	innerHeight := height - 4
	if innerHeight < 1 {
		innerHeight = 1
	}
	return m.styleManager.Box(borderColor, innerWidth, innerHeight, 0, 2, 1).Render(b.String())
}

func (m Model) renderImportPanelMain(width, height int) string {
	if m.importModel == nil {
		return ""
	}
	return m.importModel.renderImportPanel(width, height, m.mainFocusPanel == 1)
}

func (m Model) renderMainMenuHelp() string {
	settingsKey := ResolveKeyBinding(m.configManager, "settings", "F1")
	navUp := ResolveKeyBinding(m.configManager, "nav_up", "↑")
	navDown := ResolveKeyBinding(m.configManager, "nav_down", "↓")
	selectKey := ResolveKeyBinding(m.configManager, "select", "Enter")
	quitKey := ResolveKeyBinding(m.configManager, "quit", "Ctrl+C")
	help := fmt.Sprintf("%s: Settings • Left/Right: Switch Panel • %s/%s: Navigate • %s: Select • %s: Quit", settingsKey, navUp, navDown, selectKey, quitKey)
	return helpStyle.Render(help)
}
