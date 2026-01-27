package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) mainView() string {
	var b strings.Builder
	if m.user != nil {
		b.WriteString(titleStyle.Render(fmt.Sprintf("CardMan - Welcome, %s %s!", m.user.Name, m.user.Surname)) + "\n\n")
	} else {
		b.WriteString(titleStyle.Render("CardMan - Card Management TUI") + "\n\n")
	}
	b.WriteString(m.renderMainMenuTabs() + "\n\n")
	if m.mainMenuTab == 0 {
		b.WriteString(m.renderCardGamesTab())
	} else {
		b.WriteString(m.renderImportTab())
	}
	b.WriteString("\n")
	b.WriteString(m.renderMainMenuHelp() + "\n")
	if m.errorMsg != "" {
		b.WriteString("\n" + errorStyle.Render(m.errorMsg) + "\n")
	}
	return b.String()
}

func (m Model) renderMainMenuTabs() string {
	tabStyle := blurredStyle.Copy().Padding(0, 2)
	activeTabStyle := titleStyle.Copy().Padding(0, 2)
	var tabs []string
	if m.mainMenuTab == 0 {
		tabs = []string{
			activeTabStyle.Render("[ Card Games ]"),
			tabStyle.Render("  Import  "),
		}
	} else {
		tabs = []string{
			tabStyle.Render("  Card Games  "),
			activeTabStyle.Render("[ Import ]"),
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m Model) renderCardGamesTab() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Select a card game:") + "\n\n")
	if len(m.cardGames) == 0 {
		b.WriteString(errorStyle.Render("No card games found. Please run migrations.") + "\n")
	} else {
		for i, game := range m.cardGames {
			var box string
			if m.cursor == i {
				box = m.styleManager.GetBoxStyle(true).Render("🎴 " + game.Name)
			} else {
				box = m.styleManager.GetBoxStyle(false).Render("🎴 " + game.Name)
			}
			b.WriteString(box + "\n")
		}
	}
	return b.String()
}

func (m Model) renderImportTab() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Switch to Import tab to manage card sets") + "\n\n")
	b.WriteString(m.styleManager.GetPanelStyle().Render("Press Enter to open Import Manager") + "\n")
	return b.String()
}

func (m Model) renderMainMenuHelp() string {
	settingsKey := "F1"
	navUp := "↑"
	navDown := "↓"
	selectKey := "Enter"
	quitKey := "Ctrl+C"
	tabKey := "Tab"
	if m.configManager != nil {
		if k := m.configManager.KeyForAction("settings"); k != "" {
			settingsKey = k
		}
		if k := m.configManager.KeyForAction("nav_up"); k != "" {
			navUp = k
		}
		if k := m.configManager.KeyForAction("nav_down"); k != "" {
			navDown = k
		}
		if k := m.configManager.KeyForAction("select"); k != "" {
			selectKey = k
		}
		if k := m.configManager.KeyForAction("quit"); k != "" {
			quitKey = k
		}
		if k := m.configManager.KeyForAction("nav_next_tab"); k != "" {
			tabKey = k
		}
	}
	if m.mainMenuTab == 0 {
		help := fmt.Sprintf("%s: Settings • %s/%s: Navigate • %s: Select • %s: Switch Tab • %s: Quit", settingsKey, navUp, navDown, selectKey, tabKey, quitKey)
		return helpStyle.Render(help)
	}
	help := fmt.Sprintf("%s: Settings • %s: Open Import • %s: Switch Tab • %s: Quit", settingsKey, selectKey, tabKey, quitKey)
	return helpStyle.Render(help)
}
