package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
)

func getCursorPrefix(isCursor bool) string {
	if isCursor {
		return "> "
	}
	return "  "
}

func getSetStatusIcon(isImported bool) string {
	if isImported {
		return SuccessIcon
	}
	return PendingIcon
}

func calculatePaginationRange(cursor, totalItems, itemsPerPage int) (int, int) {
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

func calculateProgressPercentage(completed, total int) int {
	return int(float64(completed) / float64(total) * 100)
}

func createProgressBar(completed, total, barWidth int) string {
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
	footer := m.renderImportFooter()
	return RenderFramedWithModal(header, footer, m.renderImportBody, m.width, m.height, m.styleManager, &m.modal)
}

func (m ImportModel) renderImportHeader() string {
	var b strings.Builder
	if m.selectedCardGame != nil {
		b.WriteString(RenderBreadcrumbs(m.styleManager, m.selectedCardGame.Name, "Import Sets") + "\n")
	} else {
		b.WriteString(RenderBreadcrumbs(m.styleManager, "Import Sets") + "\n")
	}
	b.WriteString(m.renderCardGameSelector() + "\n")
	return b.String()
}

func (m ImportModel) renderImportBody(maxLines int) string {
	var b strings.Builder
	if m.isLoading {
		b.WriteString(m.styleManager.GetFocusedStyle().Render(m.loadingMsg))
		return b.String()
	}
	b.WriteString(m.renderSearchInput() + "\n\n")
	contentWidth := m.width - frameBorderSize - framePaddingX*2
	if contentWidth < 0 {
		contentWidth = 0
	}
	listWidth, actionsWidth := splitImportPanelWidths(contentWidth)
	leftPanel := m.renderSetsListPanel(listWidth)
	rightPanel := lipgloss.JoinVertical(lipgloss.Left,
		m.renderActionsPanel(actionsWidth),
		m.renderQueuePanel(actionsWidth),
	)
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		m.styleManager.GetNoStyle().Render(" "),
		rightPanel,
	)
	b.WriteString(mainContent)
	return b.String()
}

func (m ImportModel) renderImportFooter() string {
	var b strings.Builder
	if m.errorMsg != "" {
		b.WriteString(m.styleManager.GetErrorStyle().Render("Error: "+m.errorMsg) + "\n")
	}
	if m.statusMsg != "" {
		b.WriteString(m.styleManager.GetFocusedStyle().Render(m.statusMsg) + "\n")
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
	label := m.styleManager.GetBlurredStyle().Render("Card Game ")
	return label + RenderStatusBadge(m.styleManager, "info", m.selectedCardGame.Name)
}

func (m ImportModel) renderSearchInput() string {
	return renderSearchPrompt(m.styleManager, m.searchInput.View(), "")
}

func (m ImportModel) renderSetsListPanel(width int) string {
	var b strings.Builder
	if len(m.filteredSets) == 0 {
		b.WriteString(m.renderEmptySetsList())
	} else {
		itemsPerPage := 15 // Reasonable default
		b.WriteString(m.renderSetsListContent(itemsPerPage))
	}
	return renderTitledFlexPanel(m.styleManager, "Sets", b.String(), width, m.focus == importFocusSets, 2, 1)
}

func (m ImportModel) renderEmptySetsList() string {
	if m.isLoading {
		return m.styleManager.GetBlurredStyle().Render("Loading sets...")
	}
	return m.styleManager.GetBlurredStyle().Render("No sets found")
}

func (m ImportModel) renderSetsListContent(itemsPerPage int) string {
	var b strings.Builder
	start, end := calculatePaginationRange(m.cursor, len(m.filteredSets), itemsPerPage)
	for i := start; i < end; i++ {
		b.WriteString(m.renderSetListItem(i))
	}
	return b.String()
}

func (m ImportModel) resolveSetCardCount(set gameimporter.GameSet) string {
	if set.Total > 0 {
		return fmt.Sprintf("%d cards", set.Total)
	}
	if dbCount, ok := m.databaseSetCounts[set.APIID]; ok && dbCount > 0 {
		return fmt.Sprintf("%d cards", dbCount)
	}
	if m.databaseSetIDs[set.APIID] {
		return "imported"
	}
	return "? cards"
}

func (m ImportModel) renderSetListItem(index int) string {
	set := m.filteredSets[index]
	status := getSetStatusIcon(m.databaseSetIDs[set.APIID])
	cardCount := m.resolveSetCardCount(set)
	line := fmt.Sprintf("%s %s - %s (%s)", status, set.APIID, set.Name, cardCount)
	if m.isInQueue(set.APIID) {
		line += " [Q]"
	}
	return RenderListItem(line, m.cursor == index && m.focus == importFocusSets)
}

func (m ImportModel) renderActionsPanel(width int) string {
	var b strings.Builder
	isFocused := m.focus == importFocusActions
	if len(m.filteredSets) > 0 && m.cursor < len(m.filteredSets) {
		b.WriteString(m.renderSelectedSetInfo())
	}
	b.WriteString(m.renderActionsList())
	return renderTitledFlexPanel(m.styleManager, "Actions", b.String(), width, isFocused, 2, 1)
}

func (m ImportModel) renderQueuePanel(width int) string {
	var b strings.Builder
	isFocused := m.focus == importFocusQueue
	title := "Queue"
	if pending := m.queuePendingCount(); pending > 0 {
		title = fmt.Sprintf("Queue (%d pending)", pending)
	}
	if len(m.importQueue) == 0 {
		b.WriteString(m.styleManager.GetBlurredStyle().Render("No items queued") + "\n")
	} else {
		b.WriteString(m.renderQueueList())
	}
	return renderTitledFlexPanel(m.styleManager, title, b.String(), width, isFocused, 2, 1)
}

func (m ImportModel) renderQueueList() string {
	var b strings.Builder
	maxItems := 8
	shown := 0
	for i, item := range m.importQueue {
		if shown >= maxItems {
			remaining := len(m.importQueue) - shown
			b.WriteString(m.styleManager.GetBlurredStyle().Render(fmt.Sprintf("  ... and %d more", remaining)) + "\n")
			break
		}
		icon := queueItemIcon(item.status)
		line := fmt.Sprintf("  %s %s", icon, item.setName)
		if item.status == queueStatusError && item.err != nil {
			line += " " + m.styleManager.GetErrorStyle().Render("(failed)")
		}
		isCursor := i == m.queueCursor && m.focus == importFocusQueue
		if item.status == queueStatusImporting || isCursor {
			b.WriteString(m.styleManager.GetTitleStyle().Render(line) + "\n")
		} else {
			b.WriteString(m.styleManager.GetBlurredStyle().Render(line) + "\n")
		}
		shown++
	}
	return b.String()
}

func (m ImportModel) renderSelectedSetInfo() string {
	var b strings.Builder
	selectedSet := m.filteredSets[m.cursor]
	b.WriteString(renderStyledLine(m.styleManager.GetBlurredStyle(), "Selected: %s", selectedSet.APIID))
	if m.databaseSetIDs[selectedSet.APIID] {
		b.WriteString(renderStyledLine(m.styleManager.GetBlurredStyle(), "Status: Imported"))
	} else {
		b.WriteString(renderStyledLine(m.styleManager.GetBlurredStyle(), "Status: Not Imported"))
	}
	b.WriteString(renderStyledLine(m.styleManager.GetBlurredStyle(), "Cards: %s", m.resolveSetCardCount(selectedSet)))
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
	isActive := index == m.actionCursor && m.focus == importFocusActions
	prefix := getCursorPrefix(isActive)
	if isActive {
		return m.styleManager.GetTitleStyle().Render(fmt.Sprintf("%s%s", prefix, label)) + "\n"
	}
	return m.styleManager.GetBlurredStyle().Render(fmt.Sprintf("%s%s", prefix, label)) + "\n"
}

func (m ImportModel) renderDisabledAction(action ActionItem) string {
	line := m.styleManager.GetDisabledStyle().Render(fmt.Sprintf("  %s", action.label))
	if m.selectedSetHasCol && (action.actionType == ActionDelete || action.actionType == ActionReimport) {
		line += " " + m.styleManager.GetErrorStyle().Render("(in use)")
	}
	return line + "\n"
}

func (m ImportModel) renderStatusBar(contentWidth int) string {
	totalSets := len(m.availableSets)
	importedSets := len(m.databaseSetIDs)
	filteredCount := len(m.filteredSets)

	hs := m.styleManager.GetHelpStyle()
	vs := m.styleManager.GetSecondaryStyle().Bold(true)
	leftSection := hs.Render("Total ") + vs.Render(fmt.Sprintf("%d", totalSets)) + hs.Render("   Imported ") + vs.Render(fmt.Sprintf("%d", importedSets))
	rightSection := hs.Render("Showing ") + vs.Render(fmt.Sprintf("%d", filteredCount))

	// Calculate widths for left and right sections
	leftWidth := contentWidth * 2 / 3
	rightWidth := contentWidth - leftWidth

	if leftWidth < 0 {
		leftWidth = 0
	}
	if rightWidth < 0 {
		rightWidth = 0
	}

	blurred := m.styleManager.GetBlurredStyle()
	leftStyle := blurred.Width(leftWidth).Align(lipgloss.Left)
	rightStyle := blurred.Width(rightWidth).Align(lipgloss.Right)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftStyle.Render(leftSection),
		rightStyle.Render(rightSection),
	)
}

func (m ImportModel) renderHelp() string {
	hb := NewHelpBuilder(m.configManager)
	common := JoinHelpHints(
		RenderHelpHint("Tab", "Switch panel"),
		hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate"),
		hb.Build(KeyItem{"back", "Q", "Back"}),
	)
	var contextual string
	switch m.focus {
	case importFocusSets:
		contextual = hb.Build(
			KeyItem{"queue_add", "Ctrl+A", "Queue set"},
			KeyItem{"queue_remove", "Ctrl+R", "Unqueue set"},
			KeyItem{"queue_start", "Ctrl+G", "Start queue"},
		)
	case importFocusActions:
		contextual = hb.Build(KeyItem{"select", "Enter", "Execute"})
	case importFocusQueue:
		contextual = hb.Build(
			KeyItem{"queue_remove", "Ctrl+R", "Remove"},
			KeyItem{"queue_start", "Ctrl+G", "Start"},
			KeyItem{"queue_clear", "Ctrl+L", "Clear Done"},
		)
	}
	return common + "\n" + contextual
}

func (m ImportModel) renderImportProgress() string {
	header := RenderBreadcrumbs(m.styleManager, "Importing Sets")
	footer := RenderHelpHint("Ctrl+C", "Cancel")
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
	style := m.styleManager.Box(m.styleManager.scheme.Focused, panelWidth, 0, 0, 4, 2).Align(lipgloss.Center)
	return style
}

func (m ImportModel) renderProgressContent() string {
	var content strings.Builder
	content.WriteString(m.renderCurrentSetStatus())
	if m.importProgress.totalSets > 0 {
		content.WriteString(m.renderProgressBar())
	} else {
		content.WriteString(m.spinner.View() + " " + m.styleManager.GetBlurredStyle().Render("Processing...") + "\n\n")
	}
	return content.String()
}

func (m ImportModel) renderCurrentSetStatus() string {
	if m.queueProcessing && m.importProgress.setID != "" {
		queueStatus := fmt.Sprintf("(%d/%d) Downloading: %s", m.queueCurrentIndex+1, len(m.importQueue), m.importProgress.setID)
		return m.spinner.View() + " " + m.styleManager.GetTitleStyle().Render(queueStatus) + "\n\n"
	}
	if m.importProgress.setID != "" {
		return m.spinner.View() + " " + m.styleManager.GetTitleStyle().Render(fmt.Sprintf("Downloading: %s", m.importProgress.setID)) + "\n\n"
	}
	return m.spinner.View() + " " + m.styleManager.GetTitleStyle().Render("Starting import...") + "\n\n"
}

func (m ImportModel) renderProgressBar() string {
	var b strings.Builder
	completed := m.importProgress.setsCompleted
	total := m.importProgress.totalSets
	percentage := calculateProgressPercentage(completed, total)
	b.WriteString(RenderProgressBar(float64(percentage), 45, m.styleManager) + "\n\n")
	b.WriteString(renderStyledLine(m.styleManager.GetBlurredStyle(), "Sets: %d / %d completed", completed, total))
	if m.importProgress.cardsImported > 0 {
		b.WriteString(renderStyledLine(m.styleManager.GetBlurredStyle(), "Total cards: %d", m.importProgress.cardsImported))
	}
	return b.String()
}
