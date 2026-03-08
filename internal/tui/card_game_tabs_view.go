package tui

import (
	"context"
	"fmt"
	"maps"
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
	tempQuantityChanges  map[int64]int
	dbQuantities         map[int64]int
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
		tempQuantityChanges:  make(map[int64]int),
		dbQuantities:         make(map[int64]int),
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

func buildCollectionRows(collections []model.UserCollection) []table.Row {
	var rows []table.Row
	for _, collection := range collections {
		rows = append(rows, collectionToRow(collection))
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
			cmd := m.cardDetail.Update(msg)
			return m, cmd
		}
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			cmd := m.cardDetail.Update(keyMsg)
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
			m.cardDetail.Update(msg)
		}
		return m, nil
	case saveCollectionMsg:
		return m.performSaveCollection()
	case tea.KeyMsg:
		s := msg.String()
		action := MatchActionOrDefault(m.configManager, s, "")

		// Quit handling
		if action == "quit" || s == "ctrl+c" {
			return m, tea.Quit
		}

		// Back / close (use configured bindings)
		if action == "back" || action == "quit_alt" {
			return m, nil
		}

		// Tab navigation
		if action == "nav_next_tab" || action == "nav_right" || s == "right" {
			m.currentTab = (m.currentTab + 1) % tabCount
			m = m.updateTableForTab()
			if m.currentTab == TabCardSearch {
				m.searchInput.Focus()
			} else {
				m.searchInput.Blur()
			}
			return m, nil
		}

		if action == "nav_prev_tab" || action == "nav_left" || s == "left" {
			if m.currentTab == 0 {
				m.currentTab = tabCount - 1
			} else {
				m.currentTab--
			}
			m = m.updateTableForTab()
			if m.currentTab == TabCardSearch {
				m.searchInput.Focus()
			} else {
				m.searchInput.Blur()
			}
			return m, nil
		}

		if s == "tab" && m.currentTab == TabCollection {
			m.collectionTabFocus = (m.collectionTabFocus + 1) % 2
			if m.collectionTabFocus == 0 {
				m.setCompletionTable.Focus()
			} else {
				m.setCompletionTable.Blur()
			}
			return m, nil
		}
		if s == "tab" && m.currentTab == TabCardSearch {
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
		if action == "nav_up" || s == "k" || s == "up" {
			if m.currentTab == TabCollection {
				if m.collectionTabFocus == 0 {
					m.setCompletionTable, _ = m.setCompletionTable.Update(msg)
					m.updateSpotlightFromSetTable()
				} else if m.spotlightScroll > 0 {
					m.spotlightScroll--
				}
				return m, nil
			}
			if m.currentTab == TabCardSearch {
				if m.searchTabFocus == 0 {
					m.cardTable, _ = m.cardTable.Update(msg)
				} else {
					m.userSearchTable, _ = m.userSearchTable.Update(msg)
				}
				return m, nil
			}
			return m, nil
		}
		if action == "nav_down" || s == "j" || s == "down" {
			if m.currentTab == TabCollection {
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
			if m.currentTab == TabCardSearch {
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
			return m, nil
		}
		if s == "x" && m.currentTab == TabCardSearch {
			m.exportState = NewExportState("collection", m.selectedGame.Name, false, "", m.buildCollectionExportRows)
			return m, nil
		}
		if s == "ctrl+n" && m.currentTab == TabCardSearch {
			if m.searchTabFocus == 0 {
				m.cardPagination.NextPage()
				m.updateCardTable()
				m.cardTable.SetCursor(0)
			} else {
				m.collectionPagination.NextPage()
				m.filteredCollection = m.filterUserCollection(m.userSearchInput.Value())
				m.userSearchTable.SetRows(buildCollectionRows(m.paginateCollections(m.filteredCollection)))
				m.userSearchTable.SetCursor(0)
			}
			return m, nil
		}
		if s == "ctrl+p" && m.currentTab == TabCardSearch {
			if m.searchTabFocus == 0 {
				m.cardPagination.PrevPage()
				m.updateCardTable()
				m.cardTable.SetCursor(0)
			} else {
				m.collectionPagination.PrevPage()
				m.filteredCollection = m.filterUserCollection(m.userSearchInput.Value())
				m.userSearchTable.SetRows(buildCollectionRows(m.paginateCollections(m.filteredCollection)))
				m.userSearchTable.SetCursor(0)
			}
			return m, nil
		}
		if m.currentTab == TabCardSearch && m.searchTabFocus == 0 {
			if action == "increment_quantity" {
				return m.handleIncrementQuantity()
			}
			if action == "decrement_quantity" {
				return m.handleDecrementQuantity()
			}
			if action == "save" {
				return m.handleSaveCollection()
			}
			if (action == "select" || s == "enter") && m.cardDetail != nil {
				card, ok := m.getSelectedCard()
				if ok {
					qty := m.dbQuantities[card.ID] + m.tempQuantityChanges[card.ID]
					cmd := m.cardDetail.Open(card, qty)
					return m, cmd
				}
			}
		}
	}
	if m.currentTab == TabCardSearch {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			s := keyMsg.String()
			action := MatchActionOrDefault(m.configManager, s, "")
			if action != "nav_up" && action != "nav_down" && s != "k" && s != "j" {
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
				m.userSearchTable.SetRows(buildCollectionRows(m.paginateCollections(m.filteredCollection)))
				m.userSearchTable.SetCursor(0)
				return m, cmd
			}
		}
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
		return "Tab: Switch panel • " + hb.Build(
			KeyItem{"increment_quantity", "+", "Add"},
			KeyItem{"decrement_quantity", "Delete", "Remove"},
			KeyItem{"save", "Ctrl+S", "Save"},
		) + " • x: Export • Ctrl+N/P: Page • " + hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate") + " • " + hb.Build(KeyItem{"back", "Q", "Back"})
	}
	if m.currentTab == TabCollection {
		return "Tab: Switch panel • " + hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate") + " • " + hb.Pair("nav_prev_tab", "Shift+Tab", "nav_next_tab", "Tab", "Switch tabs") + " • " + hb.Build(KeyItem{"back", "Q", "Back"})
	}
	return hb.Build(KeyItem{"settings", "F1", "Settings"}) + " • " + hb.Pair("nav_prev_tab", "Shift+Tab", "nav_next_tab", "Tab", "Switch tabs") + " • " + hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate") + " • " + hb.Build(
		KeyItem{"back", "Q", "Back"},
		KeyItem{"quit", "Ctrl+C", "Quit"},
	)
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
		m.userSearchTable.SetRows(buildCollectionRows(m.filteredCollection))
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
	m.cardTable.SetColumns(scaledCardSearchColumns(tableWidth))
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
	m.userSearchTable.SetColumns(scaledCollectionColumns(tableWidth))
	m.userSearchTable.SetRows(buildCollectionRows(m.paginateCollections(m.filteredCollection)))
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

// scaledCardSearchColumns returns 5 table columns whose widths sum to availableWidth,
// distributed proportionally (Name 37%, Expansion 22%, Rarity 18%, Card# 12%, Qty 11%).
// Each column has a sensible minimum width so narrow terminals remain usable.
func scaledCardSearchColumns(availableWidth int) []table.Column {
	// Each cell has Padding(0, 1), so each column consumes Width+2 chars.
	// Subtract the total cell padding (5 columns × 2) before distributing.
	availableWidth = max(availableWidth-10, 20)
	// proportions: 37, 22, 18, 12, 11  (sum = 100)
	name := max(availableWidth*37/100, 8)
	exp := max(availableWidth*22/100, 5)
	rar := max(availableWidth*18/100, 5)
	num := max(availableWidth*12/100, 4)
	qty := max(availableWidth-name-exp-rar-num, 3)
	return []table.Column{
		{Title: "Name", Width: name},
		{Title: "Expansion", Width: exp},
		{Title: "Rarity", Width: rar},
		{Title: "Card #", Width: num},
		{Title: "Quantity", Width: qty},
	}
}

// scaledDeckColumns returns 5 table columns for the deck builder card panel,
// proportionally distributed (Name 38%, Set 23%, Rarity 19%, # 11%, Qty 9%).
func scaledDeckColumns(availableWidth int) []table.Column {
	// Each cell has Padding(0, 1), so each column consumes Width+2 chars.
	// Subtract the total cell padding (5 columns × 2) before distributing.
	availableWidth = max(availableWidth-10, 15)
	// proportions: 38, 23, 19, 11, 9 (sum = 100)
	name := max(availableWidth*38/100, 8)
	set := max(availableWidth*23/100, 4)
	rar := max(availableWidth*19/100, 4)
	num := max(availableWidth*11/100, 3)
	qty := max(availableWidth-name-set-rar-num, 3)
	return []table.Column{
		{Title: "Name", Width: name},
		{Title: "Set", Width: set},
		{Title: "Rarity", Width: rar},
		{Title: "#", Width: num},
		{Title: "Qty", Width: qty},
	}
}

// scaledCollectionColumns returns 4 table columns whose widths sum to availableWidth,
// distributed proportionally (Name 42%, Expansion 25%, Rarity 20%, Amount 13%).
func scaledCollectionColumns(availableWidth int) []table.Column {
	// Each cell has Padding(0, 1), so each column consumes Width+2 chars.
	// Subtract the total cell padding (4 columns × 2) before distributing.
	availableWidth = max(availableWidth-8, 16)
	// proportions: 42, 25, 20, 13 (sum = 100)
	name := max(availableWidth*42/100, 8)
	exp := max(availableWidth*25/100, 5)
	rar := max(availableWidth*20/100, 5)
	amt := max(availableWidth-name-exp-rar, 3)
	return []table.Column{
		{Title: "Name", Width: name},
		{Title: "Expansion", Width: exp},
		{Title: "Rarity", Width: rar},
		{Title: "Amount", Width: amt},
	}
}

// filterUserCollection filters user collection based on search query using fuzzy matching
func (m CardGameTabsModel) filterUserCollection(query string) []model.UserCollection {
	if query == "" {
		return m.userCollections
	}
	return fuzzySearchCollections(m.userCollections, query)
}

// updateCardTable updates the table with current cards, applying pagination.
func (m *CardGameTabsModel) updateCardTable() {
	source := m.filteredCards
	if m.searchInput.Value() == "" {
		source = m.cards
	}
	m.cardPagination.TotalItems = len(source)
	start, end := m.cardPagination.Slice()
	page := source[start:end]
	var rows []table.Row
	for _, card := range page {
		dbQty := m.dbQuantities[card.ID]
		tempDelta := m.tempQuantityChanges[card.ID]
		rows = append(rows, cardToRow(card, dbQty, tempDelta))
	}
	m.cardTable.SetRows(rows)
}

// cardToRow converts a Card into a table.Row with appropriate truncation and quantity.
func cardToRow(card model.Card, dbQty, tempDelta int) table.Row {
	setDisplay := ""
	if card.Set != nil {
		setDisplay = card.Set.Name
	} else if card.SetID > 0 {
		setDisplay = fmt.Sprintf("Set#%d", card.SetID)
	}
	totalQty := dbQty + tempDelta
	return table.Row{
		Truncate(card.Name, 25),
		Truncate(setDisplay, 15),
		Truncate(card.Rarity, 12),
		Truncate(card.Number, 8),
		fmt.Sprintf("%d", totalQty),
	}
}

// collectionToRow converts a UserCollection into a table.Row with truncation.
func collectionToRow(c model.UserCollection) table.Row {
	name := "Unknown Card"
	setDisplay := ""
	rarity := ""
	qty := fmt.Sprintf("%d", c.Quantity)

	if c.Card != nil {
		name = c.Card.Name
		if c.Card.Set != nil {
			setDisplay = c.Card.Set.Name
		} else if c.Card.SetID > 0 {
			setDisplay = fmt.Sprintf("Set#%d", c.Card.SetID)
		}
		rarity = c.Card.Rarity
	}

	return table.Row{
		Truncate(name, 25),
		Truncate(setDisplay, 15),
		Truncate(rarity, 12),
		qty,
	}
}

// handleIncrementQuantity increments the quantity of the selected card
func (m CardGameTabsModel) handleIncrementQuantity() (CardGameTabsModel, tea.Cmd) {
	card, ok := m.getSelectedCard()
	if !ok {
		return m, nil
	}
	if m.tempQuantityChanges == nil {
		m.tempQuantityChanges = make(map[int64]int)
	}
	m.tempQuantityChanges[card.ID]++
	m.updateCardTable()
	return m, nil
}

// handleDecrementQuantity decrements the quantity of the selected card
func (m CardGameTabsModel) handleDecrementQuantity() (CardGameTabsModel, tea.Cmd) {
	card, ok := m.getSelectedCard()
	if !ok {
		return m, nil
	}
	dbQty := m.dbQuantities[card.ID]
	tempDelta := m.tempQuantityChanges[card.ID]
	totalQty := dbQty + tempDelta
	if totalQty <= 0 {
		return m, nil
	}
	if m.tempQuantityChanges == nil {
		m.tempQuantityChanges = make(map[int64]int)
	}
	m.tempQuantityChanges[card.ID]--
	m.updateCardTable()
	return m, nil
}

// handleSaveCollection saves the temporary quantity changes to the database
func (m CardGameTabsModel) handleSaveCollection() (CardGameTabsModel, tea.Cmd) {
	if len(m.tempQuantityChanges) == 0 {
		return m, nil
	}
	if m.collectionService == nil || m.user == nil {
		return m, nil
	}
	changeCount := 0
	for _, delta := range m.tempQuantityChanges {
		if delta != 0 {
			changeCount++
		}
	}
	if changeCount == 0 {
		return m, nil
	}
	message := fmt.Sprintf("Save %d card quantity changes to your collection?", changeCount)
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
	updates := make(map[int64]int)
	for cardID, tempDelta := range m.tempQuantityChanges {
		dbQty := m.dbQuantities[cardID]
		newQty := dbQty + tempDelta
		updates[cardID] = newQty
	}
	ctx := context.Background()
	err := m.collectionService.UpsertCollectionBatch(ctx, m.user.ID, updates)
	if err != nil {
		return m, nil
	}
	maps.Copy(m.dbQuantities, updates)
	m.tempQuantityChanges = make(map[int64]int)
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
