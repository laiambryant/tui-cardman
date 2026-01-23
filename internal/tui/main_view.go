package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) mainView() string {
	var b strings.Builder

	// Title
	if m.user != nil {
		b.WriteString(titleStyle.Render(fmt.Sprintf("CardMan - Welcome, %s %s!", m.user.Name, m.user.Surname)) + "\n\n")
	} else {
		b.WriteString(titleStyle.Render("CardMan - Card Management TUI") + "\n\n")
	}

	// Card games boxes
	b.WriteString(focusedStyle.Render("Select a card game:") + "\n\n")

	if len(m.cardGames) == 0 {
		b.WriteString(errorStyle.Render("No card games found. Please run migrations.") + "\n")
	} else {
		// Define box styles
		selectedBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2).
			Width(40).
			Align(lipgloss.Center).
			Bold(true)

		unselectedBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2).
			Width(40).
			Align(lipgloss.Center)

		// Display boxes in a single centered column
		for i, game := range m.cardGames {
			var box string
			if m.cursor == i {
				box = selectedBoxStyle.Render("🎴 " + game.Name)
			} else {
				box = unselectedBoxStyle.Render("🎴 " + game.Name)
			}
			b.WriteString(box + "\n")
		}
	}

	b.WriteString("\n")
	help := NewHelpBuilder(m.configManager).Build(
		KeyItem{"settings", "F1", "Settings"},
		KeyItem{"nav_up", "↑", "Navigate"},
		KeyItem{"nav_down", "↓", "Navigate"},
		KeyItem{"select", "Enter", "Select"},
		KeyItem{"quit", "Ctrl+C", "Quit"},
	)
	// Adjust to show nav_up/nav_down together
	settingsKey := "F1"
	navUp := "↑"
	navDown := "↓"
	selectKey := "Enter"
	quitKey := "Ctrl+C"
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
	}
	help = fmt.Sprintf("%s: Settings • %s/%s: Navigate • %s: Select • %s: Quit", settingsKey, navUp, navDown, selectKey, quitKey)
	b.WriteString(helpStyle.Render(help) + "\n")
	if m.errorMsg != "" {
		b.WriteString("\n" + errorStyle.Render(m.errorMsg) + "\n")
	}
	return b.String()
}
