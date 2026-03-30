package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/laiambryant/tui-cardman/internal/auth"
	"github.com/laiambryant/tui-cardman/internal/export"
	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
	"github.com/laiambryant/tui-cardman/internal/services/usercollection"
)

// Tab represents different tabs in the card game view
type Tab int

const (
	TabCollection Tab = iota
	TabCardSearch
	TabValueHistory
	tabCount
)

type setCompletionData struct {
	SetID   int64
	SetName string
	Owned   int
	Total   int
	Percent float64
}

type CardGameTabsModel struct {
	selectedGame         *model.CardGame
	currentTab           Tab
	searchInput          textinput.Model
	cards                []model.Card
	userCollections      []model.UserCollection
	filteredCards        []model.Card
	filteredCollection   []model.UserCollection
	cursor               int
	cardTable            table.Model
	configManager        *runtimecfg.Manager
	styleManager         *StyleManager
	width                int
	height               int
	quantities           QuantityTracker
	collectionService    usercollection.UserCollectionService
	user                 *auth.User
	modal                ModalModel
	setCompletionTable   table.Model
	setCompletions       []setCompletionData
	spotlightSetID       int64
	collectionTabFocus   int
	spotlightScroll      int
	collectionValue      float64
	rarityBreakdown      string
	userSearchTable      table.Model
	userSearchInput      textinput.Model
	searchTabFocus       int
	cardDetail           *CardDetailModel
	valueHistory         []usercollection.ValueSnapshot
	exportState          ExportState
	searchCache          *SearchCache
	collectionCache      *SearchCache
	cardPagination       Pagination
	collectionPagination Pagination
}

// NewCardGameTabsModel creates a new card game tabs model
func NewCardGameTabsModel(selectedGame *model.CardGame, cfg *runtimecfg.Manager, styleManager *StyleManager) CardGameTabsModel {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search cards..."
	searchInput.Width = 30

	// Apply theme colors to search input
	styleManager.ApplyTextInputStyles(&searchInput)

	// Initialize table with Collection Tab columns (default tab)
	columns := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Expansion", Width: 15},
		{Title: "Rarity", Width: 12},
		{Title: "Amount", Width: 8},
	}

	cardTable := NewStyledTable(columns, 10, true, styleManager)

	userSearchInput := textinput.New()
	userSearchInput.Placeholder = "Search collection..."
	userSearchInput.Width = 30
	styleManager.ApplyTextInputStyles(&userSearchInput)

	userSearchColumns := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Expansion", Width: 15},
		{Title: "Rarity", Width: 12},
		{Title: "Amount", Width: 8},
	}
	userSearchTable := NewStyledTable(userSearchColumns, 10, false, styleManager)

	return CardGameTabsModel{
		selectedGame:         selectedGame,
		currentTab:           TabCollection,
		searchInput:          searchInput,
		cursor:               0,
		cardTable:            cardTable,
		configManager:        cfg,
		styleManager:         styleManager,
		quantities:           newQuantityTracker(),
		userSearchTable:      userSearchTable,
		userSearchInput:      userSearchInput,
		searchCache:          NewSearchCache(),
		collectionCache:      NewSearchCache(),
		cardPagination:       NewPagination(50),
		collectionPagination: NewPagination(50),
	}
}

func (m *CardGameTabsModel) computeCollectionStats() {
	rarityCounts := make(map[string]int)
	for _, uc := range m.userCollections {
		if uc.Card != nil {
			rarityCounts[uc.Card.Rarity] += uc.Quantity
		}
	}
	var parts []string
	for r, count := range rarityCounts {
		label := r
		if len(label) > 2 {
			label = label[:1]
		}
		parts = append(parts, fmt.Sprintf("%s:%d", label, count))
	}
	if len(parts) > 0 {
		m.rarityBreakdown = strings.Join(parts, " ")
	} else {
		m.rarityBreakdown = "N/A"
	}
	setMap := make(map[int64]*struct {
		id           int64
		name         string
		printedTotal int
	})
	for i := range m.cards {
		if m.cards[i].Set != nil && m.cards[i].Set.ID > 0 {
			if _, ok := setMap[m.cards[i].Set.ID]; !ok {
				setMap[m.cards[i].Set.ID] = &struct {
					id           int64
					name         string
					printedTotal int
				}{m.cards[i].Set.ID, m.cards[i].Set.Name, m.cards[i].Set.PrintedTotal}
			}
		}
	}
	ownedPerSet := make(map[int64]int)
	for _, uc := range m.userCollections {
		if uc.Card != nil && uc.Card.Set != nil {
			ownedPerSet[uc.Card.Set.ID]++
		}
	}
	m.setCompletions = nil
	for setID, owned := range ownedPerSet {
		if si, ok := setMap[setID]; ok {
			total := si.printedTotal
			if total == 0 {
				total = owned
			}
			pct := 0.0
			if total > 0 {
				pct = float64(owned) / float64(total) * 100
			}
			m.setCompletions = append(m.setCompletions, setCompletionData{
				SetID:   setID,
				SetName: si.name,
				Owned:   owned,
				Total:   total,
				Percent: pct,
			})
		}
	}
	sort.Slice(m.setCompletions, func(i, j int) bool {
		return m.setCompletions[i].SetName < m.setCompletions[j].SetName
	})
	m.setCompletionTable = NewStyledTable([]table.Column{
		{Title: "Set", Width: 20},
		{Title: "Cards", Width: 10},
		{Title: "Progress", Width: 20},
	}, 5, true, m.styleManager)
	var rows []table.Row
	for _, sc := range m.setCompletions {
		rows = append(rows, table.Row{
			Truncate(sc.SetName, 20),
			fmt.Sprintf("%d/%d", sc.Owned, sc.Total),
			RenderProgressBar(sc.Percent, 16, m.styleManager),
		})
	}
	m.setCompletionTable.SetRows(rows)
	if len(m.setCompletions) > 0 {
		m.spotlightSetID = m.setCompletions[0].SetID
		m.spotlightScroll = 0
	}
}

func (m CardGameTabsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *CardGameTabsModel) configureTableColumns(columns []table.Column) {
	m.cardTable.SetColumns(columns)
	m.cardTable.SetStyles(m.styleManager.GetTableStyles())
	m.cardTable.Focus()
}

func (m CardGameTabsModel) getSelectedCard() (model.Card, bool) {
	selectedRow := m.cardTable.Cursor()
	if selectedRow < 0 {
		return model.Card{}, false
	}
	source := m.filteredCards
	if m.searchInput.Value() == "" {
		source = m.cards
	}
	actualIndex := m.cardPagination.CurrentPage*m.cardPagination.PageSize + selectedRow
	if actualIndex >= len(source) {
		return model.Card{}, false
	}
	return source[actualIndex], true
}

func (m CardGameTabsModel) cardSearchVCS(width int) VisibleColumnSet {
	return BuildVisibleColumnSet(CardSearchColumns, GetVisibleColumns(m.configManager), GetColumnOrder(m.configManager), width)
}

func (m CardGameTabsModel) collectionVCS(width int) VisibleColumnSet {
	return BuildVisibleColumnSet(CollectionColumns, GetVisibleColumns(m.configManager), GetColumnOrder(m.configManager), width)
}

func buildCardRows(cards []model.Card, dbQtys, tempDeltas map[int64]int, vcs VisibleColumnSet) []table.Row {
	var rows []table.Row
	for _, card := range cards {
		rows = append(rows, vcs.BuildRow(CardToDataMap(card, dbQtys[card.ID], tempDeltas[card.ID])))
	}
	return rows
}

func buildCollectionRows(collections []model.UserCollection, vcs VisibleColumnSet) []table.Row {
	var rows []table.Row
	for _, collection := range collections {
		rows = append(rows, vcs.BuildRow(CollectionToDataMap(collection)))
	}
	return rows
}

func (m CardGameTabsModel) renderEmptySearchMessage(searchValue, messageWhenEmpty, messageNoMatch string) string {
	if searchValue == "" {
		return m.styleManager.GetBlurredStyle().Render(messageWhenEmpty) + "\n"
	}
	return m.styleManager.GetBlurredStyle().Render(messageNoMatch) + "\n"
}

func (m CardGameTabsModel) Update(msg tea.Msg) (CardGameTabsModel, tea.Cmd) {
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
		m.modal = m.modal.SetDimensions(sizeMsg.Width, sizeMsg.Height)
		if m.cardDetail != nil {
			m.cardDetail.width = sizeMsg.Width
			m.cardDetail.height = sizeMsg.Height
		}
		return m, nil
	}
	if m.cardDetail != nil && m.cardDetail.visible {
		if _, ok := msg.(cardDetailLoadedMsg); ok {
			updated, cmd := m.cardDetail.Update(msg)
			*m.cardDetail = updated
			return m, cmd
		}
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			updated, cmd := m.cardDetail.Update(keyMsg)
			*m.cardDetail = updated
			return m, cmd
		}
		return m, nil
	}
	if m.modal.IsVisible() {
		var cmd tea.Cmd
		m.modal, cmd = m.modal.Update(msg)
		return m, cmd
	}
	if m.exportState.active {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			cmd := m.exportState.HandleKey(keyMsg.String())
			return m, cmd
		}
	}
	switch msg := msg.(type) {
	case exportDoneMsg:
		m.exportState.HandleResult(msg)
		return m, nil
	case cardDetailLoadedMsg:
		if m.cardDetail != nil {
			updated, _ := m.cardDetail.Update(msg)
			*m.cardDetail = updated
		}
		return m, nil
	case saveCollectionMsg:
		return m.performSaveCollection()
	case tea.KeyMsg:
		s := msg.String()
		action := MatchActionOrDefault(m.configManager, s, "")
		if action == "quit" || s == "ctrl+c" {
			return m, tea.Quit
		}
		if action == "back" || action == "quit_alt" {
			return m, nil
		}
		if m, ok := m.handleTabSwitch(action, s); ok {
			return m, nil
		}
		switch m.currentTab {
		case TabCollection:
			return m.handleCollectionTabKeys(action, s, msg)
		case TabCardSearch:
			return m.handleCardSearchTabKeys(action, s, msg)
		}
	}
	return m, nil
}

func (m CardGameTabsModel) handleTabSwitch(action, s string) (CardGameTabsModel, bool) {
	switch {
	case action == "nav_next_tab" || action == "nav_right" || s == "right":
		m.currentTab = (m.currentTab + 1) % tabCount
	case action == "nav_prev_tab" || action == "nav_left" || s == "left":
		if m.currentTab == 0 {
			m.currentTab = tabCount - 1
		} else {
			m.currentTab--
		}
	default:
		return m, false
	}
	m = m.updateTableForTab()
	if m.currentTab == TabCardSearch {
		m.searchInput.Focus()
	} else {
		m.searchInput.Blur()
	}
	return m, true
}

func (m CardGameTabsModel) handleCollectionTabKeys(action, s string, msg tea.KeyMsg) (CardGameTabsModel, tea.Cmd) {
	if s == "tab" {
		m.collectionTabFocus = (m.collectionTabFocus + 1) % 2
		if m.collectionTabFocus == 0 {
			m.setCompletionTable.Focus()
		} else {
			m.setCompletionTable.Blur()
		}
		return m, nil
	}
	if action == "nav_up" || s == "up" {
		if m.collectionTabFocus == 0 {
			m.setCompletionTable, _ = m.setCompletionTable.Update(msg)
			m.updateSpotlightFromSetTable()
		} else if m.spotlightScroll > 0 {
			m.spotlightScroll--
		}
		return m, nil
	}
	if action == "nav_down" || s == "down" {
		if m.collectionTabFocus == 0 {
			if m.setCompletionTable.Cursor() < len(m.setCompletionTable.Rows())-1 {
				m.setCompletionTable, _ = m.setCompletionTable.Update(msg)
				m.updateSpotlightFromSetTable()
			}
		} else {
			m.spotlightScroll++
		}
		return m, nil
	}
	return m, nil
}

func (m CardGameTabsModel) handleCardSearchTabKeys(action, s string, msg tea.KeyMsg) (CardGameTabsModel, tea.Cmd) {
	if s == "tab" {
		m.searchTabFocus = (m.searchTabFocus + 1) % 2
		if m.searchTabFocus == 0 {
			m.cardTable.Focus()
			m.userSearchTable.Blur()
			m.searchInput.Focus()
			m.userSearchInput.Blur()
		} else {
			m.cardTable.Blur()
			m.userSearchTable.Focus()
			m.searchInput.Blur()
			m.userSearchInput.Focus()
		}
		return m, nil
	}
	if action == "nav_up" || s == "up" {
		if m.searchTabFocus == 0 {
			m.cardTable, _ = m.cardTable.Update(msg)
		} else {
			m.userSearchTable, _ = m.userSearchTable.Update(msg)
		}
		return m, nil
	}
	if action == "nav_down" || s == "down" {
		if m.searchTabFocus == 0 {
			if m.cardTable.Cursor() < len(m.cardTable.Rows())-1 {
				m.cardTable, _ = m.cardTable.Update(msg)
			}
		} else {
			if m.userSearchTable.Cursor() < len(m.userSearchTable.Rows())-1 {
				m.userSearchTable, _ = m.userSearchTable.Update(msg)
			}
		}
		return m, nil
	}
	if action == "export" {
		m.exportState = NewExportState("collection", m.selectedGame.Name, false, "", m.buildCollectionExportRows)
		return m, nil
	}
	if action == "page_next" {
		if m.searchTabFocus == 0 {
			m.cardPagination.NextPage()
			m.updateCardTable()
			m.cardTable.SetCursor(0)
		} else {
			m.collectionPagination.NextPage()
			m.filteredCollection = m.filterUserCollection(m.userSearchInput.Value())
			m.updateUserSearchTable(m.paginateCollections(m.filteredCollection))
			m.userSearchTable.SetCursor(0)
		}
		return m, nil
	}
	if action == "page_prev" {
		if m.searchTabFocus == 0 {
			m.cardPagination.PrevPage()
			m.updateCardTable()
			m.cardTable.SetCursor(0)
		} else {
			m.collectionPagination.PrevPage()
			m.filteredCollection = m.filterUserCollection(m.userSearchInput.Value())
			m.updateUserSearchTable(m.paginateCollections(m.filteredCollection))
			m.userSearchTable.SetCursor(0)
		}
		return m, nil
	}
	if m.searchTabFocus == 0 {
		if action == "increment_quantity" {
			return m.handleIncrementQuantity()
		}
		if action == "decrement_quantity" {
			return m.handleDecrementQuantity()
		}
		if action == "save" {
			return m.handleSaveCollection()
		}
		if action == "select" && m.cardDetail != nil {
			card, ok := m.getSelectedCard()
			if ok {
				qty := m.quantities.total(card.ID)
				cmd := m.cardDetail.Open(card, qty)
				return m, cmd
			}
		}
	}
	if !isModifierKey(s) {
		if m.searchTabFocus == 0 {
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.filteredCards = m.filterCards(m.searchInput.Value())
			m.cardPagination.Reset()
			m.updateCardTable()
			m.cursor = 0
			m.cardTable.SetCursor(0)
			return m, cmd
		}
		var cmd tea.Cmd
		m.userSearchInput, cmd = m.userSearchInput.Update(msg)
		m.filteredCollection = m.filterUserCollection(m.userSearchInput.Value())
		m.collectionPagination.Reset()
		m.collectionPagination.TotalItems = len(m.filteredCollection)
		m.updateUserSearchTable(m.paginateCollections(m.filteredCollection))
		m.userSearchTable.SetCursor(0)
		return m, cmd
	}
	return m, nil
}

func (m CardGameTabsModel) View() string {
	if m.cardDetail != nil && m.cardDetail.visible {
		return m.cardDetail.View()
	}
	header := m.renderCardGameTabsHeader()
	footer := m.renderCardGameTabsFooter()
	return RenderFramedWithModal(header, footer, m.renderCardGameTabsBody, m.width, m.height, m.styleManager, &m.modal)
}

func (m CardGameTabsModel) renderCardGameTabsHeader() string {
	var b strings.Builder
	if m.selectedGame != nil {
		b.WriteString(m.styleManager.GetTitleStyle().Render(m.selectedGame.Name+" Collection Manager") + "\n")
	}
	b.WriteString(RenderTabBar(m.styleManager, []string{"Collection", "Card Search", "Value"}, int(m.currentTab)))
	return b.String()
}

func (m CardGameTabsModel) renderCardGameTabsBody(availableHeight int) string {
	switch m.currentTab {
	case TabCollection:
		return m.renderCollectionTab(availableHeight)
	case TabCardSearch:
		return m.renderCardSearchTab(availableHeight)
	case TabValueHistory:
		return m.renderValueHistoryTab(availableHeight)
	}
	return ""
}

func (m CardGameTabsModel) renderValueHistoryTab(availableHeight int) string {
	contentWidth := m.width - frameBorderSize - framePaddingX*2
	if contentWidth < 20 {
		contentWidth = 20
	}
	var b strings.Builder
	ts := m.styleManager.GetTitleStyle()
	ns := m.styleManager.GetNoStyle()
	b.WriteString(ts.Render("Collection Value History") + "\n\n")
	valueStr := "N/A"
	if m.collectionValue > 0 {
		valueStr = fmt.Sprintf("$%.2f", m.collectionValue)
	}
	b.WriteString(ts.Render(valueStr) + "\n")
	if len(m.valueHistory) >= 2 {
		if change7d, ok := calcValueChange(m.valueHistory, 7); ok {
			changeStyle := m.styleManager.applyBGFG(lipgloss.NewStyle().Foreground(lipgloss.Color("34")))
			if change7d < 0 {
				changeStyle = m.styleManager.applyBGFG(lipgloss.NewStyle().Foreground(lipgloss.Color("160")))
			}
			b.WriteString(ns.Render("7d: ") + changeStyle.Render(fmt.Sprintf("%+.1f%%", change7d)))
			b.WriteString("  ")
		}
		if change30d, ok := calcValueChange(m.valueHistory, 30); ok {
			changeStyle := m.styleManager.applyBGFG(lipgloss.NewStyle().Foreground(lipgloss.Color("34")))
			if change30d < 0 {
				changeStyle = m.styleManager.applyBGFG(lipgloss.NewStyle().Foreground(lipgloss.Color("160")))
			}
			b.WriteString(ns.Render("30d: ") + changeStyle.Render(fmt.Sprintf("%+.1f%%", change30d)))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	chartHeight := availableHeight - 8
	if chartHeight < 3 {
		chartHeight = 3
	}
	b.WriteString(renderValueChart(m.valueHistory, contentWidth, chartHeight, m.styleManager))
	return b.String()
}

func (m CardGameTabsModel) renderCardGameTabsFooter() string {
	if m.exportState.active {
		return m.exportState.Render(m.styleManager)
	}
	footer := m.styleManager.GetHelpStyle().Render(m.buildHelpText())
	if m.exportState.statusMsg != "" {
		footer = m.styleManager.GetHelpStyle().Render(m.exportState.statusMsg) + "  " + footer
	}
	return footer
}

func (m CardGameTabsModel) buildHelpText() string {
	hb := NewHelpBuilder(m.configManager)
	if m.currentTab == TabCardSearch {
		return strings.Join([]string{
			"Tab: Switch panel",
			hb.Build(
				KeyItem{"increment_quantity", "+", "Add"},
				KeyItem{"decrement_quantity", "Delete", "Remove"},
				KeyItem{"save", "Ctrl+S", "Save"},
			),
			hb.Build(KeyItem{"export", "x", "Export"}),
			hb.Pair("page_next", "Ctrl+N", "page_prev", "Ctrl+P", "Page"),
			hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate"),
			hb.Build(KeyItem{"back", "Q", "Back"}),
		}, " | ")
	}
	if m.currentTab == TabCollection {
		return strings.Join([]string{
			"Tab: Switch panel",
			hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate"),
			hb.Pair("nav_prev_tab", "Shift+Tab", "nav_next_tab", "Tab", "Switch tabs"),
			hb.Build(KeyItem{"back", "Q", "Back"}),
		}, " | ")
	}
	return strings.Join([]string{
		hb.Build(KeyItem{"settings", "F1", "Settings"}),
		hb.Pair("nav_prev_tab", "Shift+Tab", "nav_next_tab", "Tab", "Switch tabs"),
		hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate"),
		hb.Build(KeyItem{"back", "Q", "Back"}, KeyItem{"quit", "Ctrl+C", "Quit"}),
	}, " | ")
}

func (m CardGameTabsModel) updateTableForTab() CardGameTabsModel {
	m.cardTable.SetRows([]table.Row{})
	switch m.currentTab {
	case TabCardSearch:
		m.configureTableColumns(m.getCardSearchColumns())
		m.updateCardTable()
		m.searchTabFocus = 0
		m.cardTable.Focus()
		m.userSearchTable.Blur()
		m.filteredCollection = m.filterUserCollection(m.userSearchInput.Value())
		m.updateUserSearchTable(m.filteredCollection)
	case TabCollection:
		m.setCompletionTable.Focus()
		m.collectionTabFocus = 0
	}
	m.cardTable.SetCursor(0)
	return m
}

func (m CardGameTabsModel) getCardSearchColumns() []table.Column {
	return []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Expansion", Width: 15},
		{Title: "Rarity", Width: 12},
		{Title: "Card #", Width: 8},
		{Title: "Quantity", Width: 8},
	}
}

func (m CardGameTabsModel) renderCollectionTab(availableHeight int) string {
	if len(m.userCollections) == 0 {
		var b strings.Builder
		b.WriteString(m.styleManager.GetTitleStyle().Render("Your Collection Summary") + "\n")
		b.WriteString(m.styleManager.GetBlurredStyle().Render("No cards in your collection yet."))
		return b.String()
	}
	contentWidth := m.width - frameBorderSize - framePaddingX*2
	if contentWidth < 20 {
		contentWidth = 20
	}
	leftWidth := contentWidth / 2
	rightWidth := contentWidth - leftWidth
	totalCards := 0
	for _, u := range m.userCollections {
		totalCards += u.Quantity
	}
	valueStr := "N/A"
	if m.collectionValue > 0 {
		valueStr = fmt.Sprintf("$%.2f", m.collectionValue)
	}
	ns := m.styleManager.GetNoStyle()
	topLeft := m.styleManager.GetTitleStyle().Render("Your Collection Summary") + "\n" +
		ns.Render(fmt.Sprintf("Total unique: %d  Total: %d", len(m.userCollections), totalCards)) + "\n" +
		ns.Render(fmt.Sprintf("Rarity: %s", m.rarityBreakdown)) + "\n" +
		ns.Render(fmt.Sprintf("Value: %s", valueStr))
	topLeftHeight := 6
	topLeftPanel := RenderPanel(m.styleManager, topLeft, leftWidth, topLeftHeight, false, 1, 0)
	topLeftRenderedHeight := lipgloss.Height(topLeftPanel)
	bottomLeftHeight := availableHeight - topLeftRenderedHeight
	if bottomLeftHeight < 3 {
		bottomLeftHeight = 3
	}
	m.setCompletionTable.SetHeight(max(bottomLeftHeight-4, 1))
	bottomLeft := m.setCompletionTable.View()
	if len(m.setCompletions) == 0 {
		bottomLeft = m.styleManager.GetBlurredStyle().Render("No sets found.")
	}
	bottomLeftPanel := RenderPanel(m.styleManager, bottomLeft, leftWidth, bottomLeftHeight, m.collectionTabFocus == 0, 1, 0)
	leftColumn := lipgloss.JoinVertical(lipgloss.Left, topLeftPanel, bottomLeftPanel)
	spotlightHeight := availableHeight
	if spotlightHeight < 3 {
		spotlightHeight = 3
	}
	var selected *setCompletionData
	for i := range m.setCompletions {
		if m.setCompletions[i].SetID == m.spotlightSetID {
			selected = &m.setCompletions[i]
			break
		}
	}
	spotContent := m.styleManager.GetBlurredStyle().Render("No set selected.")
	if selected != nil {
		ownedCards := make(map[int64]bool)
		for _, uc := range m.userCollections {
			if uc.Card != nil && uc.Card.Set != nil && uc.Card.Set.ID == m.spotlightSetID {
				ownedCards[uc.Card.ID] = true
			}
		}
		var setCards []model.Card
		for i := range m.cards {
			if m.cards[i].Set != nil && m.cards[i].Set.ID == m.spotlightSetID {
				setCards = append(setCards, m.cards[i])
			}
		}
		var sb strings.Builder
		sb.WriteString(m.styleManager.GetTitleStyle().Render(selected.SetName) + "\n")
		sb.WriteString(RenderProgressBar(selected.Percent, min(rightWidth-8, 20), m.styleManager) + " ")
		sb.WriteString(m.styleManager.GetBlurredStyle().Render(fmt.Sprintf("%d/%d (%.0f%%)", selected.Owned, selected.Total, selected.Percent)) + "\n")
		gridWidth := rightWidth - 6
		if gridWidth < 10 {
			gridWidth = 10
		}
		sb.WriteString(renderSetProgressGrid(setCards, ownedCards, gridWidth, spotlightHeight-4, m.styleManager))
		spotContent = sb.String()
	}
	rightPanel := RenderPanel(m.styleManager, spotContent, rightWidth, spotlightHeight, m.collectionTabFocus == 1, 1, 0)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightPanel)
}

func (m *CardGameTabsModel) updateSpotlightFromSetTable() {
	idx := m.setCompletionTable.Cursor()
	if idx >= 0 && idx < len(m.setCompletions) {
		m.spotlightSetID = m.setCompletions[idx].SetID
		m.spotlightScroll = 0
	}
}

func (m CardGameTabsModel) renderCardSearchTab(availableHeight int) string {
	contentWidth := m.width - frameBorderSize - framePaddingX*2
	if contentWidth < 20 {
		contentWidth = 20
	}
	leftWidth := contentWidth / 2
	rightWidth := contentWidth - leftWidth
	// panelPadX=1, borderOverhead=2: inner table area = panelWidth - 4
	leftTableWidth := max(leftWidth-4, 20)
	rightTableWidth := max(rightWidth-4, 20)
	tableHeight := CalcTableHeight(availableHeight, 2, 3)
	leftContent := m.renderSearchLeftPanel(tableHeight, leftTableWidth)
	leftPanel := RenderPanel(m.styleManager, leftContent, leftWidth, availableHeight, m.searchTabFocus == 0, 1, 0)
	rightContent := m.renderSearchRightPanel(tableHeight, rightTableWidth)
	rightPanel := RenderPanel(m.styleManager, rightContent, rightWidth, availableHeight, m.searchTabFocus == 1, 1, 0)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m CardGameTabsModel) renderSearchLeftPanel(tableHeight, tableWidth int) string {
	var b strings.Builder
	b.WriteString(m.styleManager.GetTitleStyle().Render("Search All Cards") + "\n")
	b.WriteString(m.styleManager.GetBlurredStyle().Render("Search: ") + m.styleManager.GetNoStyle().Render(m.searchInput.View()) + " " + m.styleManager.GetBlurredStyle().Render(m.cardPagination.StatusText()) + "\n")
	if len(m.cardTable.Rows()) == 0 {
		if m.searchInput.Value() == "" {
			b.WriteString(m.styleManager.GetBlurredStyle().Render("No cards available.") + "\n")
		} else {
			b.WriteString(m.styleManager.GetBlurredStyle().Render("No cards match your search.") + "\n")
		}
		return b.String()
	}
	visible := GetVisibleColumns(m.configManager)
	vcs := BuildVisibleColumnSet(CardSearchColumns, visible, GetColumnOrder(m.configManager), tableWidth)
	m.cardTable.SetColumns(vcs.Columns)
	m.cardTable.SetHeight(tableHeight)
	b.WriteString(m.cardTable.View())
	return b.String()
}

func (m CardGameTabsModel) renderSearchRightPanel(tableHeight, tableWidth int) string {
	var b strings.Builder
	b.WriteString(m.styleManager.GetTitleStyle().Render("My Collection") + "\n")
	b.WriteString(m.styleManager.GetBlurredStyle().Render("Search: ") + m.styleManager.GetNoStyle().Render(m.userSearchInput.View()) + " " + m.styleManager.GetBlurredStyle().Render(m.collectionPagination.StatusText()) + "\n")
	if len(m.filteredCollection) == 0 {
		b.WriteString(m.renderEmptySearchMessage(m.userSearchInput.Value(), "No cards in your collection yet.", "No cards match your search."))
		return b.String()
	}
	visible := GetVisibleColumns(m.configManager)
	vcs := BuildVisibleColumnSet(CollectionColumns, visible, GetColumnOrder(m.configManager), tableWidth)
	m.userSearchTable.SetColumns(vcs.Columns)
	m.userSearchTable.SetRows(buildCollectionRows(m.paginateCollections(m.filteredCollection), vcs))
	m.userSearchTable.SetHeight(tableHeight)
	b.WriteString(m.userSearchTable.View())
	return b.String()
}

// filterCards filters cards based on search query using fuzzy matching with caching
func (m CardGameTabsModel) filterCards(query string) []model.Card {
	return filterCardsByQueryCached(m.cards, query, m.searchCache)
}

// paginateCollections applies the collectionPagination to a collection slice.
func (m *CardGameTabsModel) paginateCollections(collections []model.UserCollection) []model.UserCollection {
	m.collectionPagination.TotalItems = len(collections)
	start, end := m.collectionPagination.Slice()
	return collections[start:end]
}

// filterUserCollection filters user collection based on search query using fuzzy matching
func (m CardGameTabsModel) filterUserCollection(query string) []model.UserCollection {
	if query == "" {
		return m.userCollections
	}
	return fuzzySearchCollections(m.userCollections, query)
}

// updateCardTable updates the table with current cards, applying pagination.
func (m *CardGameTabsModel) updateUserSearchTable(collections []model.UserCollection) {
	vcs := m.collectionVCS(80)
	m.userSearchTable.SetColumns(vcs.Columns)
	m.userSearchTable.SetRows(buildCollectionRows(collections, vcs))
}

func (m *CardGameTabsModel) updateCardTable() {
	source := m.filteredCards
	if m.searchInput.Value() == "" {
		source = m.cards
	}
	m.cardPagination.TotalItems = len(source)
	start, end := m.cardPagination.Slice()
	page := source[start:end]
	vcs := m.cardSearchVCS(80)
	m.cardTable.SetColumns(vcs.Columns)
	m.cardTable.SetRows(buildCardRows(page, m.quantities.db, m.quantities.temp, vcs))
}

func (m CardGameTabsModel) handleIncrementQuantity() (CardGameTabsModel, tea.Cmd) {
	card, ok := m.getSelectedCard()
	if !ok {
		return m, nil
	}
	m.quantities.increment(card.ID)
	m.updateCardTable()
	return m, nil
}

func (m CardGameTabsModel) handleDecrementQuantity() (CardGameTabsModel, tea.Cmd) {
	card, ok := m.getSelectedCard()
	if !ok {
		return m, nil
	}
	if !m.quantities.decrement(card.ID) {
		return m, nil
	}
	m.updateCardTable()
	return m, nil
}

func (m CardGameTabsModel) handleSaveCollection() (CardGameTabsModel, tea.Cmd) {
	if m.quantities.pendingCount() == 0 || m.collectionService == nil || m.user == nil {
		return m, nil
	}
	message := fmt.Sprintf("Save %d card quantity changes to your collection?", m.quantities.pendingCount())
	m.modal = newModal(
		"Confirm Save",
		message,
		func() tea.Cmd {
			return func() tea.Msg {
				return saveCollectionMsg{}
			}
		},
		func() tea.Cmd {
			return nil
		},
		m.styleManager, m.width, m.height,
	)
	return m, nil
}

type saveCollectionMsg struct{}

func (m CardGameTabsModel) performSaveCollection() (CardGameTabsModel, tea.Cmd) {
	updates := m.quantities.buildUpdates()
	ctx := context.Background()
	err := m.collectionService.UpsertCollectionBatch(ctx, m.user.ID, updates)
	if err != nil {
		return m, nil
	}
	m.quantities.commit(updates)
	m.searchCache.Invalidate()
	m.collectionCache.Invalidate()
	m.updateCardTable()

	if m.selectedGame != nil {
		collections, err := m.collectionService.GetUserCollectionByGameID(m.user.ID, m.selectedGame.ID)
		if err == nil {
			m.userCollections = collections
			m.filteredCollection = m.filterUserCollection(m.userSearchInput.Value())
			m.computeCollectionStats()
		}
	}
	return m, nil
}

func (m *CardGameTabsModel) buildCollectionExportRows() []export.CardRow {
	var rows []export.CardRow
	for _, uc := range m.userCollections {
		if uc.Card == nil || uc.Quantity <= 0 {
			continue
		}
		setName := ""
		setCode := ""
		if uc.Card.Set != nil {
			setName = uc.Card.Set.Name
			setCode = uc.Card.Set.Code
		}
		rows = append(rows, export.CardRow{
			Name:     uc.Card.Name,
			SetName:  setName,
			SetCode:  setCode,
			Number:   uc.Card.Number,
			Rarity:   uc.Card.Rarity,
			Quantity: uc.Quantity,
		})
	}
	return rows
}
