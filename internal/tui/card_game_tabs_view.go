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
	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
	"github.com/laiambryant/tui-cardman/internal/services/usercollection"
)

// Tab represents different tabs in the card game view
type Tab int

const (
	TabCollection Tab = iota
	TabCardSearch
	TabUserSearch
)

type setCompletionData struct {
	SetID   int64
	SetName string
	Owned   int
	Total   int
	Percent float64
}
type CardGameTabsModel struct {
	selectedGame        *model.CardGame
	currentTab          Tab
	searchInput         textinput.Model
	cards               []model.Card
	userCollections     []model.UserCollection
	filteredCards       []model.Card
	filteredCollection  []model.UserCollection
	cursor              int
	cardTable           table.Model
	configManager       *runtimecfg.Manager
	styleManager        *StyleManager
	width               int
	height              int
	tempQuantityChanges map[int64]int
	dbQuantities        map[int64]int
	collectionService   usercollection.UserCollectionService
	user                *auth.User
	modal               ModalModel
	setCompletionTable  table.Model
	setCompletions      []setCompletionData
	spotlightSetID      int64
	collectionTabFocus  int
	spotlightScroll     int
	collectionValue     float64
	rarityBreakdown     string
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

	return CardGameTabsModel{
		selectedGame:        selectedGame,
		currentTab:          TabCollection,
		searchInput:         searchInput,
		cursor:              0,
		cardTable:           cardTable,
		configManager:       cfg,
		styleManager:        styleManager,
		tempQuantityChanges: make(map[int64]int),
		dbQuantities:        make(map[int64]int),
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
	showAll := m.searchInput.Value() == ""
	if showAll {
		if selectedRow >= len(m.cards) {
			return model.Card{}, false
		}
		return m.cards[selectedRow], true
	}
	if selectedRow >= len(m.filteredCards) {
		return model.Card{}, false
	}
	return m.filteredCards[selectedRow], true
}
func buildCollectionRows(collections []model.UserCollection) []table.Row {
	var rows []table.Row
	for _, collection := range collections {
		rows = append(rows, collectionToRow(collection))
	}
	return rows
}
func (m CardGameTabsModel) renderEmptySearchMessage(searchValue string, messageWhenEmpty string, messageNoMatch string) string {
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
		return m, nil
	}
	if m.modal.IsVisible() {
		var cmd tea.Cmd
		m.modal, cmd = m.modal.Update(msg)
		return m, cmd
	}
	switch msg := msg.(type) {
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
			m.currentTab = (m.currentTab + 1) % 3
			m = m.updateTableForTab()
			if m.currentTab == TabCardSearch || m.currentTab == TabUserSearch {
				m.searchInput.Focus()
			} else {
				m.searchInput.Blur()
			}
			return m, nil
		}

		if action == "nav_prev_tab" || action == "nav_left" || s == "left" {
			if m.currentTab == 0 {
				m.currentTab = 2
			} else {
				m.currentTab--
			}
			m = m.updateTableForTab()
			if m.currentTab == TabCardSearch || m.currentTab == TabUserSearch {
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
			if m.currentTab == TabCardSearch || m.currentTab == TabUserSearch {
				m.cardTable, _ = m.cardTable.Update(msg)
				return m, nil
			}
			return m, nil
		}
		if action == "nav_down" || s == "j" || s == "down" {
			if m.currentTab == TabCollection {
				if m.collectionTabFocus == 0 {
					m.setCompletionTable, _ = m.setCompletionTable.Update(msg)
					m.updateSpotlightFromSetTable()
				} else {
					m.spotlightScroll++
				}
				return m, nil
			}
			if m.currentTab == TabCardSearch || m.currentTab == TabUserSearch {
				m.cardTable, _ = m.cardTable.Update(msg)
				return m, nil
			}
			return m, nil
		}
		// Quantity management (only in Card Search tab)
		if m.currentTab == TabCardSearch {
			if action == "increment_quantity" {
				return m.handleIncrementQuantity()
			}
			if action == "decrement_quantity" {
				return m.handleDecrementQuantity()
			}
			if action == "save" {
				return m.handleSaveCollection()
			}
		}
	}
	// Update search input if in search tabs (but not for navigation keys)
	if m.currentTab == TabCardSearch || m.currentTab == TabUserSearch {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			s := keyMsg.String()
			action := MatchActionOrDefault(m.configManager, s, "")
			// Don't pass navigation keys to search input
			if action != "nav_up" && action != "nav_down" && s != "k" && s != "j" {
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				// Filter results based on search
				switch m.currentTab {
				case TabCardSearch:
					m.filteredCards = m.filterCards(m.searchInput.Value())
					m.updateCardTable()
				case TabUserSearch:
					m.filteredCollection = m.filterUserCollection(m.searchInput.Value())
					// Also update table rows for user search
					var rows []table.Row
					for _, collection := range m.filteredCollection {
						rows = append(rows, collectionToRow(collection))
					}
					m.cardTable.SetRows(rows)
				}
				// Reset cursor when search changes
				m.cursor = 0
				m.cardTable.SetCursor(0)
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m CardGameTabsModel) View() string {
	header := m.renderCardGameTabsHeader()
	footer := m.renderCardGameTabsFooter()
	return RenderFramedWithModal(header, footer, m.renderCardGameTabsBody, m.width, m.height, m.styleManager, &m.modal)
}

func (m CardGameTabsModel) renderCardGameTabsHeader() string {
	var b strings.Builder
	if m.selectedGame != nil {
		b.WriteString(m.styleManager.GetTitleStyle().Render(m.selectedGame.Name+" Collection Manager") + "\n")
	}
	b.WriteString(RenderTabBar(m.styleManager, []string{"Collection", "Card Search", "My Collection"}, int(m.currentTab)))
	return b.String()
}

func (m CardGameTabsModel) renderCardGameTabsBody(availableHeight int) string {
	switch m.currentTab {
	case TabCollection:
		return m.renderCollectionTab(availableHeight)
	case TabCardSearch:
		return m.renderCardSearchTab(availableHeight)
	case TabUserSearch:
		return m.renderUserSearchTab(availableHeight)
	}
	return ""
}

func (m CardGameTabsModel) renderCardGameTabsFooter() string {
	return m.styleManager.GetHelpStyle().Render(m.buildHelpText())
}
func (m CardGameTabsModel) buildHelpText() string {
	hb := NewHelpBuilder(m.configManager)
	if m.currentTab == TabCardSearch {
		return hb.Build(
			KeyItem{"increment_quantity", "+", "Add"},
			KeyItem{"decrement_quantity", "Delete", "Remove"},
			KeyItem{"save", "Ctrl+S", "Save"},
		) + " • " + hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate") + " • " + hb.Build(KeyItem{"back", "Q", "Back"})
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
	case TabUserSearch:
		m.configureTableColumns(m.getCollectionColumns())
		m.cardTable.SetRows(buildCollectionRows(m.filteredCollection))
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
func (m CardGameTabsModel) getCollectionColumns() []table.Column {
	return []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Expansion", Width: 15},
		{Title: "Rarity", Width: 12},
		{Title: "Amount", Width: 8},
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
	topLeft := fmt.Sprintf("Total unique: %d  Total: %d\nRarity: %s\nValue: ", len(m.userCollections), totalCards, m.rarityBreakdown)
	if m.collectionValue > 0 {
		topLeft += fmt.Sprintf("$%.2f", m.collectionValue)
	} else {
		topLeft += "N/A"
	}
	topLeftPanel := RenderPanel(m.styleManager, m.styleManager.GetTitleStyle().Render("Your Collection Summary\n")+topLeft, leftWidth, 4, false, 1, 0)
	bottomLeftHeight := availableHeight - 6
	if bottomLeftHeight < 3 {
		bottomLeftHeight = 3
	}
	m.setCompletionTable.SetHeight(bottomLeftHeight - 1)
	bottomLeft := m.setCompletionTable.View()
	if len(m.setCompletions) == 0 {
		bottomLeft = m.styleManager.GetBlurredStyle().Render("No sets found.")
	}
	bottomLeftPanel := RenderPanel(m.styleManager, bottomLeft, leftWidth, bottomLeftHeight, m.collectionTabFocus == 0, 1, 0)
	leftColumn := lipgloss.JoinVertical(lipgloss.Left, topLeftPanel, bottomLeftPanel)
	spotlightHeight := availableHeight - 2
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
		var spotCards []struct {
			name  string
			owned bool
		}
		for i := range m.cards {
			if m.cards[i].Set != nil && m.cards[i].Set.ID == m.spotlightSetID {
				spotCards = append(spotCards, struct {
					name  string
					owned bool
				}{m.cards[i].Name, ownedCards[m.cards[i].ID]})
			}
		}
		visibleLines := spotlightHeight - 3
		if visibleLines < 1 {
			visibleLines = 1
		}
		start := m.spotlightScroll
		if start > len(spotCards)-visibleLines {
			start = len(spotCards) - visibleLines
		}
		if start < 0 {
			start = 0
		}
		end := start + visibleLines
		if end > len(spotCards) {
			end = len(spotCards)
		}
		var sb strings.Builder
		sb.WriteString(m.styleManager.GetTitleStyle().Render(selected.SetName) + "\n")
		sb.WriteString(m.styleManager.GetBlurredStyle().Render(fmt.Sprintf("%d/%d (%.0f%%)", selected.Owned, selected.Total, selected.Percent)) + "\n")
		for _, sc := range spotCards[start:end] {
			if sc.owned {
				sb.WriteString(m.styleManager.GetFocusedStyle().Render("✓ " + sc.name + "\n"))
			} else {
				sb.WriteString(m.styleManager.GetBlurredStyle().Render("✗ " + sc.name + "\n"))
			}
		}
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
	var b strings.Builder
	b.WriteString(m.styleManager.GetTitleStyle().Render("Search All Cards") + "\n")
	b.WriteString(m.styleManager.GetBlurredStyle().Render("Search: ") + m.styleManager.GetNoStyle().Render(m.searchInput.View()) + "\n")
	showAll := m.searchInput.Value() == ""
	var rows []table.Row
	var any bool
	if showAll {
		for _, card := range m.cards {
			dbQty := m.dbQuantities[card.ID]
			tempDelta := m.tempQuantityChanges[card.ID]
			rows = append(rows, cardToRow(card, dbQty, tempDelta))
		}
		any = len(rows) > 0
	} else {
		for _, card := range m.filteredCards {
			dbQty := m.dbQuantities[card.ID]
			tempDelta := m.tempQuantityChanges[card.ID]
			rows = append(rows, cardToRow(card, dbQty, tempDelta))
		}
		any = len(rows) > 0
	}
	if !any {
		if showAll {
			b.WriteString(m.styleManager.GetBlurredStyle().Render("No cards available.") + "\n")
		} else {
			if m.searchInput.Value() == "" {
				b.WriteString(m.styleManager.GetBlurredStyle().Render("Type to search for cards...") + "\n")
			} else {
				b.WriteString(m.styleManager.GetBlurredStyle().Render("No cards match your search.") + "\n")
			}
		}
		return b.String()
	}
	b.WriteString(m.styleManager.GetTitleStyle().Render("Found cards:") + "\n")
	tableHeight := CalcTableHeight(availableHeight, 3, 3)
	m.cardTable.SetRows(rows)
	m.cardTable.SetHeight(tableHeight)
	b.WriteString(m.styleManager.GetTableBaseStyle().Render(m.cardTable.View()))
	return b.String()
}

func (m CardGameTabsModel) renderUserSearchTab(availableHeight int) string {
	var b strings.Builder
	b.WriteString(m.styleManager.GetTitleStyle().Render("Search Your Collection") + "\n")
	b.WriteString(m.styleManager.GetBlurredStyle().Render("Search: ") + m.styleManager.GetNoStyle().Render(m.searchInput.View()) + "\n")
	if len(m.filteredCollection) == 0 {
		b.WriteString(m.renderEmptySearchMessage(m.searchInput.Value(), "Type to search your collection...", "No cards in your collection match your search."))
	} else {
		b.WriteString(m.styleManager.GetTitleStyle().Render("Your matching cards:") + "\n")
		tableHeight := CalcTableHeight(availableHeight, 3, 3)
		m.cardTable.SetRows(buildCollectionRows(m.filteredCollection))
		m.cardTable.SetHeight(tableHeight)
		b.WriteString(m.styleManager.GetTableBaseStyle().Render(m.cardTable.View()))
	}
	return b.String()
}

// filterCards filters cards based on search query using fuzzy matching
func (m CardGameTabsModel) filterCards(query string) []model.Card {
	if query == "" {
		return m.cards
	}
	var filtered []model.Card
	query = strings.ToLower(query)
	for _, card := range m.cards {
		if strings.Contains(strings.ToLower(card.Name), query) ||
			strings.Contains(strings.ToLower(card.Number), query) ||
			strings.Contains(strings.ToLower(card.Rarity), query) {
			filtered = append(filtered, card)
		}
	}

	return filtered
}

// filterUserCollection filters user collection based on search query
func (m CardGameTabsModel) filterUserCollection(query string) []model.UserCollection {
	if query == "" {
		return m.userCollections
	}
	var filtered []model.UserCollection
	query = strings.ToLower(query)
	for _, collection := range m.userCollections {
		if collection.Card != nil {
			if strings.Contains(strings.ToLower(collection.Card.Name), query) ||
				strings.Contains(strings.ToLower(collection.Card.Number), query) ||
				strings.Contains(strings.ToLower(collection.Card.Rarity), query) ||
				strings.Contains(strings.ToLower(collection.Condition), query) ||
				strings.Contains(strings.ToLower(collection.Notes), query) {
				filtered = append(filtered, collection)
			}
		}
	}
	return filtered
}

// updateCardTable updates the table with current filtered cards
func (m *CardGameTabsModel) updateCardTable() {
	var rows []table.Row
	for _, card := range m.filteredCards {
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
	m.modal = NewModalModel(
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
		m.styleManager,
	)
	if m.width > 0 && m.height > 0 {
		m.modal = m.modal.SetDimensions(m.width, m.height)
	}
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
	m.updateCardTable()

	if m.selectedGame != nil {
		collections, err := m.collectionService.GetUserCollectionByGameID(m.user.ID, m.selectedGame.ID)
		if err == nil {
			m.userCollections = collections
			m.filteredCollection = m.filterUserCollection(m.searchInput.Value())
			m.computeCollectionStats()
		}
	}
	return m, nil
}
