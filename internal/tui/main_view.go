package tui

import (
	"fmt"
	"strings"
)

func (m Model) mainView() string {
	var b strings.Builder

	// Title
	if m.user != nil {
		b.WriteString(titleStyle.Render(fmt.Sprintf("🃏 CardMan - Welcome, %s %s!", m.user.Name, m.user.Surname)) + "\n\n")
	} else {
		b.WriteString(titleStyle.Render("🃏 CardMan - Card Games") + "\n\n")
	}

	// Card games list
	b.WriteString(focusedStyle.Render("Select a card game:") + "\n\n")

	if len(m.cardGames) == 0 {
		b.WriteString(errorStyle.Render("No card games found. Please run migrations.") + "\n")
	} else {
		for i, game := range m.cardGames {
			cursor := " "
			if m.cursor == i {
				cursor = focusedStyle.Render(">")
				b.WriteString(fmt.Sprintf("%s %s\n", cursor, focusedStyle.Render(game.Name)))
			} else {
				b.WriteString(fmt.Sprintf("  %s\n", game.Name))
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓: Navigate • Enter: Select • Ctrl+C: Quit") + "\n")

	return b.String()
}
