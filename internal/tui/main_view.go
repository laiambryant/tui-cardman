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
	body := m.renderMainBody()
	return renderFramedViewWithLayout(header, body, footer, layout, m.styleManager)
}

func (m Model) renderMainHeader() string {
	var b strings.Builder
	if m.user != nil {
		b.WriteString(titleStyle.Render(fmt.Sprintf("CardMan - Welcome, %s %s!", m.user.Name, m.user.Surname)) + "\n")
	} else {
		b.WriteString(titleStyle.Render("CardMan - Card Management TUI") + "\n")
	}
	b.WriteString(m.renderMainMenuTabs())
	return b.String()
}

func (m Model) renderMainBody() string {
	if m.mainMenuTab == 0 {
		return m.renderCardGamesTab()
	}
	return m.renderImportTab()
}

func (m Model) renderMainFooter() string {
	var b strings.Builder
	if m.errorMsg != "" {
		b.WriteString(errorStyle.Render("Error: "+m.errorMsg) + "\n")
	}
	b.WriteString(m.renderMainMenuHelp())
	return b.String()
}

func (m Model) renderTab(isActive bool, label string) string {
	if isActive {
		return titleStyle.Padding(0, 2).Render("[ " + label + " ]")
	}
	return blurredStyle.Padding(0, 2).Render("  " + label + "  ")
}

func (m Model) renderMainMenuTabs() string {
	tabs := []string{
		m.renderTab(m.mainMenuTab == 0, "Card Games"),
		m.renderTab(m.mainMenuTab == 1, "Import"),
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m Model) renderCardGameBox(index int, name string) string {
	if m.cursor == index {
		return m.styleManager.GetBoxStyle(true).Render(name)
	}
	return m.styleManager.GetBoxStyle(false).Render(name)
}

func (m Model) renderCardGamesTab() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Select a card game:") + "\n")
	if len(m.cardGames) == 0 {
		b.WriteString(errorStyle.Render("No card games found. Please run migrations.") + "\n")
	} else {
		for i, game := range m.cardGames {
			b.WriteString(m.renderCardGameBox(i, game.Name) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) renderImportTab() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Switch to Import tab to manage card sets") + "\n")
	b.WriteString(m.styleManager.GetPanelStyle().Render("Press Enter to open Import Manager") + "\n")
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) renderMainMenuHelp() string {
	settingsKey := ResolveKeyBinding(m.configManager, "settings", "F1")
	navUp := ResolveKeyBinding(m.configManager, "nav_up", "↑")
	navDown := ResolveKeyBinding(m.configManager, "nav_down", "↓")
	selectKey := ResolveKeyBinding(m.configManager, "select", "Enter")
	quitKey := ResolveKeyBinding(m.configManager, "quit", "Ctrl+C")
	tabKey := ResolveKeyBinding(m.configManager, "nav_next_tab", "Tab")
	if m.mainMenuTab == 0 {
		help := fmt.Sprintf("%s: Settings • %s/%s: Navigate • %s: Select • %s: Switch Tab • %s: Quit", settingsKey, navUp, navDown, selectKey, tabKey, quitKey)
		return helpStyle.Render(help)
	}
	help := fmt.Sprintf("%s: Settings • %s: Open Import • %s: Switch Tab • %s: Quit", settingsKey, selectKey, tabKey, quitKey)
	return helpStyle.Render(help)
}
