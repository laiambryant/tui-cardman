package tui

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/laiambryant/tui-cardman/internal/auth"
	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
	"github.com/laiambryant/tui-cardman/internal/services/usercollection"
	"maps"
	"strings"
)

// Tab represents different tabs in the card game view
type Tab int

const (
	TabCollection Tab = iota
	TabCardSearch
	TabUserSearch
)

// CardGameTabsModel represents the state for the card game tabs view
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
}

// NewCardGameTabsModel creates a new card game tabs model
func NewCardGameTabsModel(selectedGame *model.CardGame, cfg *runtimecfg.Manager, styleManager *StyleManager) CardGameTabsModel {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search cards..."
	searchInput.Width = 30

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
		return blurredStyle.Render(messageWhenEmpty) + "\n"
	}
	return blurredStyle.Render(messageNoMatch) + "\n"
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

		// Up / down navigation (keep vim keys as fallbacks)
		if action == "nav_up" || s == "k" || s == "up" {
			if m.currentTab == TabCardSearch || m.currentTab == TabUserSearch || m.currentTab == TabCollection {
				m.cardTable, _ = m.cardTable.Update(msg)
				return m, nil
			}
			return m, nil
		}
		if action == "nav_down" || s == "j" || s == "down" {
			if m.currentTab == TabCardSearch || m.currentTab == TabUserSearch || m.currentTab == TabCollection {
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
	layout := calculateFrameLayout(lipgloss.Height(header), lipgloss.Height(footer), m.width, m.height)
	body := m.renderCardGameTabsBody(layout.BodyContentHeight)
	content := renderFramedViewWithLayout(header, body, footer, layout, m.styleManager)
	if m.modal.IsVisible() {
		m.modal = m.modal.SetBackgroundContent(content)
		return m.modal.View()
	}
	return content
}

func (m CardGameTabsModel) renderCardGameTabsHeader() string {
	var b strings.Builder
	if m.selectedGame != nil {
		b.WriteString(titleStyle.Render(m.selectedGame.Name+" Collection Manager") + "\n")
	}
	tabs := []string{"Collection", "Card Search", "My Collection"}
	var tabStyles []string
	for i, tab := range tabs {
		if Tab(i) == m.currentTab {
			tabStyles = append(tabStyles, titleStyle.Render("[ "+tab+" ]"))
		} else {
			tabStyles = append(tabStyles, blurredStyle.Render("  "+tab+"  "))
		}
	}
	b.WriteString(strings.Join(tabStyles, " "))
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
	return helpStyle.Render(m.buildHelpText())
}
func (m CardGameTabsModel) buildHelpText() string {
	if m.currentTab == TabCardSearch {
		return m.buildCardSearchHelpText()
	}
	return m.buildDefaultHelpText()
}
func (m CardGameTabsModel) buildCardSearchHelpText() string {
	incrementKey := ResolveKeyBinding(m.configManager, "increment_quantity", "+")
	decrementKey := ResolveKeyBinding(m.configManager, "decrement_quantity", "Delete")
	saveKey := ResolveKeyBinding(m.configManager, "save", "Ctrl+S")
	navUp := ResolveKeyBinding(m.configManager, "nav_up", "↑")
	navDown := ResolveKeyBinding(m.configManager, "nav_down", "↓")
	backKey := ResolveKeyBinding(m.configManager, "back", "Q")
	return fmt.Sprintf("%s: Add • %s: Remove • %s: Save • %s/%s: Navigate • %s: Back", incrementKey, decrementKey, saveKey, navUp, navDown, backKey)
}
func (m CardGameTabsModel) buildDefaultHelpText() string {
	settingsKey := ResolveKeyBinding(m.configManager, "settings", "F1")
	nextTab := ResolveKeyBinding(m.configManager, "nav_next_tab", "Tab")
	prevTab := ResolveKeyBinding(m.configManager, "nav_prev_tab", "Shift+Tab")
	navUp := ResolveKeyBinding(m.configManager, "nav_up", "↑")
	navDown := ResolveKeyBinding(m.configManager, "nav_down", "↓")
	backKey := ResolveKeyBinding(m.configManager, "back", "Q")
	quitKey := ResolveKeyBinding(m.configManager, "quit", "Ctrl+C")
	return fmt.Sprintf("%s: Settings • %s/%s: Switch tabs • %s/%s: Navigate • %s: Back • %s: Quit", settingsKey, prevTab, nextTab, navUp, navDown, backKey, quitKey)
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
		m.configureTableColumns(m.getCollectionColumns())
		m.cardTable.SetRows(buildCollectionRows(m.filteredCollection))
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

// renderCollectionTab now uses a table instead of a list
func (m CardGameTabsModel) renderCollectionTab(availableHeight int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Your Collection Summary") + "\n")
	if len(m.filteredCollection) == 0 {
		b.WriteString(blurredStyle.Render("No cards in your collection yet.") + "\n")
		b.WriteString(blurredStyle.Render("Use Card Search to discover cards to add!"))
		return b.String()
	}
	totalCards := 0
	for _, collection := range m.filteredCollection {
		totalCards += collection.Quantity
	}
	b.WriteString(blurredStyle.Render("Total unique cards: ") +
		titleStyle.Render(fmt.Sprintf("%d", len(m.filteredCollection))) + "\n")
	b.WriteString(blurredStyle.Render("Total cards: ") +
		titleStyle.Render(fmt.Sprintf("%d", totalCards)) + "\n")
	b.WriteString(titleStyle.Render("Recent additions:") + "\n")
	tableHeight := 10
	if availableHeight > 0 {
		tableHeight = availableHeight - 4
	}
	if tableHeight < 3 {
		tableHeight = 3
	}
	m.cardTable.SetRows(buildCollectionRows(m.filteredCollection))
	m.cardTable.SetHeight(tableHeight)
	b.WriteString(m.styleManager.GetTableBaseStyle().Render(m.cardTable.View()))
	return b.String()
}

func (m CardGameTabsModel) renderCardSearchTab(availableHeight int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Search All Cards") + "\n")
	b.WriteString(blurredStyle.Render("Search: ") + m.searchInput.View() + "\n")
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
			b.WriteString(blurredStyle.Render("No cards available.") + "\n")
		} else {
			if m.searchInput.Value() == "" {
				b.WriteString(blurredStyle.Render("Type to search for cards...") + "\n")
			} else {
				b.WriteString(blurredStyle.Render("No cards match your search.") + "\n")
			}
		}
		return b.String()
	}
	b.WriteString(titleStyle.Render("Found cards:") + "\n")
	tableHeight := 10
	if availableHeight > 0 {
		tableHeight = availableHeight - 3
	}
	if tableHeight < 3 {
		tableHeight = 3
	}
	m.cardTable.SetRows(rows)
	m.cardTable.SetHeight(tableHeight)
	b.WriteString(m.styleManager.GetTableBaseStyle().Render(m.cardTable.View()))
	return b.String()
}

func (m CardGameTabsModel) renderUserSearchTab(availableHeight int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Search Your Collection") + "\n")
	b.WriteString(blurredStyle.Render("Search: ") + m.searchInput.View() + "\n")
	if len(m.filteredCollection) == 0 {
		b.WriteString(m.renderEmptySearchMessage(m.searchInput.Value(), "Type to search your collection...", "No cards in your collection match your search."))
	} else {
		b.WriteString(titleStyle.Render("Your matching cards:") + "\n")
		tableHeight := 10
		if availableHeight > 0 {
			tableHeight = availableHeight - 3
		}
		if tableHeight < 3 {
			tableHeight = 3
		}
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
		}
	}

	return m, nil
}
