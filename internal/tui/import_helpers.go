package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m ImportModel) createPanelStyle(width int, height int, borderColor lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(width).
		Height(height)
}
func getCursorPrefix(isCursor bool) string {
	if isCursor {
		return "> "
	}
	return "  "
}
func getSetStatusIcon(isImported bool) string {
	if isImported {
		return "✓"
	}
	return "○"
}
func calculatePaginationRange(cursor int, totalItems int, itemsPerPage int) (int, int) {
	start := cursor - itemsPerPage/2
	if start < 0 {
		start = 0
	}
	end := start + itemsPerPage
	if end > totalItems {
		end = totalItems
	}
	return start, end
}
func calculateProgressPercentage(completed int, total int) int {
	return int(float64(completed) / float64(total) * 100)
}
func createProgressBar(completed int, total int, barWidth int) string {
	filledWidth := int(float64(barWidth) * float64(completed) / float64(total))
	return strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)
}
func renderStyledLine(style lipgloss.Style, format string, args ...interface{}) string {
	return style.Render(fmt.Sprintf(format, args...)) + "\n"
}

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
	listStyle := m.createPanelStyle(50, 20, m.styleManager.scheme.Blurred)
	var b strings.Builder
	b.WriteString(titleStyle.Render("Sets") + "\n\n")
	if len(m.filteredSets) == 0 {
		b.WriteString(m.renderEmptySetsList())
	} else {
		b.WriteString(m.renderSetsListContent())
	}
	return listStyle.Render(b.String())
}
func (m ImportModel) renderEmptySetsList() string {
	if m.isLoading {
		return blurredStyle.Render("Loading sets...")
	}
	return blurredStyle.Render("No sets found")
}
func (m ImportModel) renderSetsListContent() string {
	var b strings.Builder
	start, end := calculatePaginationRange(m.cursor, len(m.filteredSets), 16)
	for i := start; i < end; i++ {
		b.WriteString(m.renderSetListItem(i))
	}
	return b.String()
}
func (m ImportModel) renderSetListItem(index int) string {
	set := m.filteredSets[index]
	prefix := getCursorPrefix(m.cursor == index)
	status := getSetStatusIcon(m.databaseSetIDs[set.ID])
	line := fmt.Sprintf("%s%s %s - %s (%d cards)", prefix, status, set.ID, set.Name, set.Total)
	if m.cursor == index && !m.focusOnActions {
		return titleStyle.Render(line) + "\n"
	}
	return blurredStyle.Render(line) + "\n"
}

func (m ImportModel) renderActionsPanel() string {
	panelStyle := m.createPanelStyle(40, 20, m.styleManager.scheme.Blurred)
	var b strings.Builder
	b.WriteString(titleStyle.Render("Actions") + "\n\n")
	if len(m.filteredSets) > 0 && m.cursor < len(m.filteredSets) {
		b.WriteString(m.renderSelectedSetInfo())
	}
	b.WriteString(m.renderActionsList())
	return panelStyle.Render(b.String())
}
func (m ImportModel) renderSelectedSetInfo() string {
	var b strings.Builder
	selectedSet := m.filteredSets[m.cursor]
	b.WriteString(renderStyledLine(blurredStyle, "Selected: %s", selectedSet.ID))
	if m.databaseSetIDs[selectedSet.ID] {
		b.WriteString(renderStyledLine(blurredStyle, "Status: Imported"))
	} else {
		b.WriteString(renderStyledLine(blurredStyle, "Status: Not Imported"))
	}
	b.WriteString(renderStyledLine(blurredStyle, "Cards: %d", selectedSet.Total))
	b.WriteString("\n")
	return b.String()
}
func (m ImportModel) renderActionsList() string {
	var b strings.Builder
	actions := m.getAvailableActions()
	for i, action := range actions {
		b.WriteString(m.renderActionItem(i, action))
	}
	return b.String()
}
func (m ImportModel) renderActionItem(index int, action ActionItem) string {
	if action.enabled {
		return m.renderEnabledAction(index, action.label)
	}
	return m.renderDisabledAction(action)
}
func (m ImportModel) renderEnabledAction(index int, label string) string {
	prefix := getCursorPrefix(index == m.actionCursor && m.focusOnActions)
	if index == m.actionCursor && m.focusOnActions {
		return titleStyle.Render(fmt.Sprintf("%s%s", prefix, label)) + "\n"
	}
	return blurredStyle.Render(fmt.Sprintf("%s%s", prefix, label)) + "\n"
}
func (m ImportModel) renderDisabledAction(action ActionItem) string {
	line := m.styleManager.GetDisabledStyle().Render(fmt.Sprintf("  %s", action.label))
	if m.selectedSetHasCol && (action.actionType == ActionDelete || action.actionType == ActionReimport) {
		line += " " + errorStyle.Render("(in use)")
	}
	return line + "\n"
}

func (m ImportModel) renderStatusBar() string {
	totalSets := len(m.availableSets)
	importedSets := len(m.databaseSetIDs)
	filteredCount := len(m.filteredSets)
	status := fmt.Sprintf("Total: %d | Imported: %d | Showing: %d", totalSets, importedSets, filteredCount)
	return blurredStyle.Render(status)
}

func (m ImportModel) renderHelp() string {
	upKey := ResolveKeyBinding(m.configManager, "nav_up", "↑")
	downKey := ResolveKeyBinding(m.configManager, "nav_down", "↓")
	enterKey := ResolveKeyBinding(m.configManager, "select", "Enter")
	backKey := ResolveKeyBinding(m.configManager, "back", "Q")
	help := fmt.Sprintf("Tab: Switch Panel • %s/%s: Navigate • %s: Execute • %s: Back", upKey, downKey, enterKey, backKey)
	return helpStyle.Render(help)
}

func (m ImportModel) renderImportProgress() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Importing Sets...") + "\n\n")
	progressStyle := m.createProgressPanelStyle()
	b.WriteString(progressStyle.Render(m.renderProgressContent()))
	return b.String()
}
func (m ImportModel) createProgressPanelStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styleManager.scheme.Focused).
		Padding(2, 4).
		Width(60).
		Align(lipgloss.Center)
}
func (m ImportModel) renderProgressContent() string {
	var content strings.Builder
	content.WriteString(m.renderCurrentSetStatus())
	if m.importProgress.totalSets > 0 {
		content.WriteString(m.renderProgressBar())
	} else {
		content.WriteString(blurredStyle.Render("Processing...") + "\n\n")
	}
	content.WriteString("\n" + helpStyle.Render("Press Ctrl+C to cancel"))
	return content.String()
}
func (m ImportModel) renderCurrentSetStatus() string {
	if m.importProgress.setID != "" {
		return titleStyle.Render(fmt.Sprintf("Current: %s", m.importProgress.setID)) + "\n\n"
	}
	return titleStyle.Render("Starting import...") + "\n\n"
}
func (m ImportModel) renderProgressBar() string {
	var b strings.Builder
	completed := m.importProgress.setsCompleted
	total := m.importProgress.totalSets
	percentage := calculateProgressPercentage(completed, total)
	bar := createProgressBar(completed, total, 40)
	b.WriteString(fmt.Sprintf("[%s] %d%%\n\n", bar, percentage))
	b.WriteString(renderStyledLine(blurredStyle, "Sets: %d / %d completed", completed, total))
	if m.importProgress.cardsImported > 0 {
		b.WriteString(renderStyledLine(blurredStyle, "Total cards: %d", m.importProgress.cardsImported))
	}
	return b.String()
}
