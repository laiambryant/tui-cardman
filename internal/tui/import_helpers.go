package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m ImportModel) renderImportView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("CardMan - Import Pokemon TCG Sets") + "\n\n")
	b.WriteString(m.renderCardGameSelector() + "\n\n")
	if m.isLoading {
		b.WriteString(focusedStyle.Render(m.loadingMsg) + "\n")
		return b.String()
	}
	b.WriteString(m.renderSearchInput() + "\n\n")
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderSetsList(),
		"  ",
		m.renderActionsPanel(),
	)
	b.WriteString(mainContent + "\n\n")
	b.WriteString(m.renderStatusBar() + "\n")
	b.WriteString(m.renderHelp() + "\n")
	if m.errorMsg != "" {
		b.WriteString("\n" + errorStyle.Render(m.errorMsg) + "\n")
	}
	if m.statusMsg != "" {
		b.WriteString("\n" + focusedStyle.Render(m.statusMsg) + "\n")
	}
	return b.String()
}

func (m ImportModel) renderCardGameSelector() string {
	label := blurredStyle.Render("Card Game: ")
	value := titleStyle.Render(m.selectedCardGame.Name)
	return label + value
}

func (m ImportModel) renderSearchInput() string {
	label := blurredStyle.Render("Search: ")
	return label + m.searchInput.View()
}

func (m ImportModel) renderSetsList() string {
	listStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styleManager.scheme.Blurred).
		Padding(1, 2).
		Width(50).
		Height(20)
	var b strings.Builder
	b.WriteString(titleStyle.Render("Sets") + "\n\n")
	if len(m.filteredSets) == 0 {
		if m.isLoading {
			b.WriteString(blurredStyle.Render("Loading sets..."))
		} else {
			b.WriteString(blurredStyle.Render("No sets found"))
		}
	} else {
		start := m.cursor - 8
		if start < 0 {
			start = 0
		}
		end := start + 16
		if end > len(m.filteredSets) {
			end = len(m.filteredSets)
		}
		for i := start; i < end; i++ {
			set := m.filteredSets[i]
			prefix := "  "
			if m.cursor == i {
				prefix = "> "
			}
			status := "○"
			if m.databaseSetIDs[set.ID] {
				status = "✓"
			}
			line := fmt.Sprintf("%s%s %s - %s (%d cards)", prefix, status, set.ID, set.Name, set.Total)
			if m.cursor == i && !m.focusOnActions {
				b.WriteString(titleStyle.Render(line) + "\n")
			} else {
				b.WriteString(blurredStyle.Render(line) + "\n")
			}
		}
	}
	return listStyle.Render(b.String())
}

func (m ImportModel) renderActionsPanel() string {
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styleManager.scheme.Blurred).
		Padding(1, 2).
		Width(40).
		Height(20)
	var b strings.Builder
	b.WriteString(titleStyle.Render("Actions") + "\n\n")
	if len(m.filteredSets) > 0 && m.cursor < len(m.filteredSets) {
		selectedSet := m.filteredSets[m.cursor]
		b.WriteString(blurredStyle.Render(fmt.Sprintf("Selected: %s", selectedSet.ID)) + "\n")
		if m.databaseSetIDs[selectedSet.ID] {
			b.WriteString(blurredStyle.Render("Status: Imported") + "\n")
		} else {
			b.WriteString(blurredStyle.Render("Status: Not Imported") + "\n")
		}
		b.WriteString(blurredStyle.Render(fmt.Sprintf("Cards: %d", selectedSet.Total)) + "\n\n")
	}
	actions := m.getAvailableActions()
	for i, action := range actions {
		var line string
		if action.enabled {
			if i == m.actionCursor && m.focusOnActions {
				line = titleStyle.Render(fmt.Sprintf("> %s", action.label))
			} else {
				line = blurredStyle.Render(fmt.Sprintf("  %s", action.label))
			}
		} else {
			line = m.styleManager.GetDisabledStyle().Render(fmt.Sprintf("  %s", action.label))
			if m.selectedSetHasCol && (action.actionType == ActionDelete || action.actionType == ActionReimport) {
				line += " " + errorStyle.Render("(in use)")
			}
		}
		b.WriteString(line + "\n")
	}
	return panelStyle.Render(b.String())
}

func (m ImportModel) renderStatusBar() string {
	totalSets := len(m.availableSets)
	importedSets := len(m.databaseSetIDs)
	filteredCount := len(m.filteredSets)
	status := fmt.Sprintf("Total: %d | Imported: %d | Showing: %d", totalSets, importedSets, filteredCount)
	return blurredStyle.Render(status)
}

func (m ImportModel) renderHelp() string {
	tabKey := "Tab"
	upKey := "↑"
	downKey := "↓"
	enterKey := "Enter"
	backKey := "Q"
	if m.configManager != nil {
		if k := m.configManager.KeyForAction("nav_up"); k != "" {
			upKey = k
		}
		if k := m.configManager.KeyForAction("nav_down"); k != "" {
			downKey = k
		}
		if k := m.configManager.KeyForAction("select"); k != "" {
			enterKey = k
		}
		if k := m.configManager.KeyForAction("back"); k != "" {
			backKey = k
		}
	}
	help := fmt.Sprintf("%s: Switch Panel • %s/%s: Navigate • %s: Execute • %s: Back", tabKey, upKey, downKey, enterKey, backKey)
	return helpStyle.Render(help)
}

func (m ImportModel) renderImportProgress() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Importing Sets...") + "\n\n")
	progressStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styleManager.scheme.Focused).
		Padding(2, 4).
		Width(60).
		Align(lipgloss.Center)
	var content strings.Builder
	if m.importProgress.setID != "" {
		content.WriteString(titleStyle.Render(fmt.Sprintf("Current: %s", m.importProgress.setID)) + "\n\n")
	} else {
		content.WriteString(titleStyle.Render("Starting import...") + "\n\n")
	}
	if m.importProgress.totalSets > 0 {
		completed := m.importProgress.setsCompleted
		total := m.importProgress.totalSets
		percentage := int(float64(completed) / float64(total) * 100)
		barWidth := 40
		filledWidth := int(float64(barWidth) * float64(completed) / float64(total))
		bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)
		content.WriteString(fmt.Sprintf("[%s] %d%%\n\n", bar, percentage))
		content.WriteString(blurredStyle.Render(fmt.Sprintf("Sets: %d / %d completed", completed, total)) + "\n")
		if m.importProgress.cardsImported > 0 {
			content.WriteString(blurredStyle.Render(fmt.Sprintf("Total cards: %d", m.importProgress.cardsImported)) + "\n")
		}
	} else {
		content.WriteString(blurredStyle.Render("Processing...") + "\n\n")
	}
	content.WriteString("\n" + helpStyle.Render("Press Ctrl+C to cancel"))
	b.WriteString(progressStyle.Render(content.String()))
	return b.String()
}
