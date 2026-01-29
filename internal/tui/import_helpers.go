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
		return "[x]"
	}
	return "[ ]"
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

func splitImportPanelWidths(contentWidth int) (int, int) {
	// Each panel has: border (2 chars) + padding (4 chars) = 6 overhead
	// We need spacing between panels: 1 char
	const panelOverhead = 6 // border + padding per panel
	const spacing = 1
	
	if contentWidth < 20 {
		return max(contentWidth/2, 10), max(contentWidth/2, 10)
	}
	
	// Total overhead: 2 panels worth of overhead + spacing
	totalOverhead := (panelOverhead * 2) + spacing
	available := contentWidth - totalOverhead
	
	if available < 10 {
		return 15, 15
	}
	
	left := available * 3 / 5
	right := available - left
	
	return left, right
}

func (m ImportModel) renderImportView() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}
	header := m.renderImportHeader()
	body := m.renderImportBody()
	footer := m.renderImportFooter()
	return renderFramedView(header, body, footer, m.width, m.height, m.styleManager)
}

func (m ImportModel) renderImportHeader() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("CardMan - Import Pokemon TCG Sets") + "\n")
	b.WriteString(m.renderCardGameSelector() + "\n")
	return b.String()
}

func (m ImportModel) renderImportBody() string {
	var b strings.Builder
	if m.isLoading {
		b.WriteString(focusedStyle.Render(m.loadingMsg))
		return b.String()
	}
	b.WriteString(m.renderSearchInput() + "\n\n")
	
	// Calculate available width for the two panels
	contentWidth := m.width - frameBorderSize - framePaddingX*2
	if contentWidth < 0 {
		contentWidth = 0
	}
	listWidth, actionsWidth := splitImportPanelWidths(contentWidth)
	
	leftPanel := m.renderSetsListPanel(listWidth)
	rightPanel := m.renderActionsPanelContent(actionsWidth)
	
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		" ",
		rightPanel,
	)
	b.WriteString(mainContent)
	return b.String()
}

func (m ImportModel) renderImportFooter() string {
	var b strings.Builder
	if m.errorMsg != "" {
		b.WriteString(errorStyle.Render("Error: "+m.errorMsg) + "\n")
	}
	if m.statusMsg != "" {
		b.WriteString(focusedStyle.Render(m.statusMsg) + "\n")
	}
	contentWidth := m.width - frameBorderSize - framePaddingX*2
	if contentWidth < 0 {
		contentWidth = 0
	}
	b.WriteString(m.renderStatusBar(contentWidth) + "\n")
	b.WriteString(m.renderHelp())
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

func (m ImportModel) renderSetsListPanel(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Sets") + "\n")
	if len(m.filteredSets) == 0 {
		b.WriteString(m.renderEmptySetsList())
	} else {
		itemsPerPage := 15 // Reasonable default
		b.WriteString(m.renderSetsListContent(itemsPerPage))
	}
	
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styleManager.scheme.Blurred).
		Padding(1, 2).
		MaxWidth(width)
	
	return panelStyle.Render(b.String())
}
func (m ImportModel) renderEmptySetsList() string {
	if m.isLoading {
		return blurredStyle.Render("Loading sets...")
	}
	return blurredStyle.Render("No sets found")
}
func (m ImportModel) renderSetsListContent(itemsPerPage int) string {
	var b strings.Builder
	start, end := calculatePaginationRange(m.cursor, len(m.filteredSets), itemsPerPage)
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

func (m ImportModel) renderActionsPanelContent(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Actions") + "\n")
	if len(m.filteredSets) > 0 && m.cursor < len(m.filteredSets) {
		b.WriteString(m.renderSelectedSetInfo())
	}
	b.WriteString(m.renderActionsList())
	
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styleManager.scheme.Blurred).
		Padding(1, 2).
		MaxWidth(width)
	
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

func (m ImportModel) renderStatusBar(contentWidth int) string {
	totalSets := len(m.availableSets)
	importedSets := len(m.databaseSetIDs)
	filteredCount := len(m.filteredSets)

	leftSection := fmt.Sprintf("Total: %d  Imported: %d", totalSets, importedSets)
	rightSection := fmt.Sprintf("Showing: %d", filteredCount)

	// Calculate widths for left and right sections
	leftWidth := contentWidth * 2 / 3
	rightWidth := contentWidth - leftWidth

	if leftWidth < 0 {
		leftWidth = 0
	}
	if rightWidth < 0 {
		rightWidth = 0
	}

	leftStyle := blurredStyle.Copy().Width(leftWidth).Align(lipgloss.Left)
	rightStyle := blurredStyle.Copy().Width(rightWidth).Align(lipgloss.Right)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftStyle.Render(leftSection),
		rightStyle.Render(rightSection),
	)
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
	header := titleStyle.Render("Importing Sets...")
	footer := helpStyle.Render("Press Ctrl+C to cancel")
	layout := calculateFrameLayout(lipgloss.Height(header), lipgloss.Height(footer), m.width, m.height)
	body := m.renderImportProgressBody(layout.ContentWidth, layout.BodyContentHeight)
	return renderFramedViewWithLayout(header, body, footer, layout, m.styleManager)
}

func (m ImportModel) renderImportProgressBody(contentWidth, contentHeight int) string {
	progressStyle := m.createProgressPanelStyle(contentWidth)
	panel := progressStyle.Render(m.renderProgressContent())
	return lipgloss.Place(contentWidth, contentHeight, lipgloss.Center, lipgloss.Center, panel)
}
func (m ImportModel) createProgressPanelStyle(contentWidth int) lipgloss.Style {
	panelWidth := 60
	if contentWidth > 0 {
		panelWidth = min(60, contentWidth)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styleManager.scheme.Focused).
		Padding(2, 4).
		Width(panelWidth).
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
