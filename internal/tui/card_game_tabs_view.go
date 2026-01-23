package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	tempQuantityChanges map[int64]int
	dbQuantities        map[int64]int
	collectionService   usercollection.UserCollectionService
	user                *auth.User
	modal               ModalModel
}

// NewCardGameTabsModel creates a new card game tabs model
func NewCardGameTabsModel(selectedGame *model.CardGame, cfg *runtimecfg.Manager) CardGameTabsModel {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search cards..."
	searchInput.Width = 30

	// Initialize table
	columns := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Expansion", Width: 15},
		{Title: "Rarity", Width: 12},
		{Title: "Card #", Width: 8},
		{Title: "Quantity", Width: 8},
	}

	cardTable := NewStyledTable(columns, 10, true)

	return CardGameTabsModel{
		selectedGame:        selectedGame,
		currentTab:          TabCollection,
		searchInput:         searchInput,
		cursor:              0,
		cardTable:           cardTable,
		configManager:       cfg,
		tempQuantityChanges: make(map[int64]int),
		dbQuantities:        make(map[int64]int),
	}
}

func (m CardGameTabsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m CardGameTabsModel) Update(msg tea.Msg) (CardGameTabsModel, tea.Cmd) {
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
		action := ""
		if m.configManager != nil {
			action = m.configManager.MatchAction(s)
		}

		// Quit handling
		if action == "quit" || s == "ctrl+c" {
			return m, tea.Quit
		}

		// Back / close (use configured bindings)
		if action == "back" || action == "quit_alt" {
			return m, nil
		}

		// Tab navigation
		if action == "nav_next_tab" {
			m.currentTab = (m.currentTab + 1) % 3
			if m.currentTab == TabCardSearch || m.currentTab == TabUserSearch {
				m.searchInput.Focus()
			} else {
				m.searchInput.Blur()
			}
			return m, nil
		}

		if action == "nav_prev_tab" {
			if m.currentTab == 0 {
				m.currentTab = 2
			} else {
				m.currentTab--
			}
			if m.currentTab == TabCardSearch || m.currentTab == TabUserSearch {
				m.searchInput.Focus()
			} else {
				m.searchInput.Blur()
			}
			return m, nil
		}

		// Up / down navigation (keep vim keys as fallbacks)
		if action == "nav_up" || s == "k" {
			if m.currentTab == TabCardSearch {
				m.cardTable, _ = m.cardTable.Update(msg)
				return m, nil
			}
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		}

		if action == "nav_down" || s == "j" {
			if m.currentTab == TabCardSearch {
				m.cardTable, _ = m.cardTable.Update(msg)
				return m, nil
			}
			maxItems := 0
			switch m.currentTab {
			case TabCollection:
				maxItems = len(m.filteredCollection)
			case TabUserSearch:
				maxItems = len(m.filteredCollection)
			}
			if m.cursor < maxItems-1 {
				m.cursor++
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

	// Update search input if in search tabs
	if m.currentTab == TabCardSearch || m.currentTab == TabUserSearch {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)

		// Filter results based on search
		switch m.currentTab {
		case TabCardSearch:
			m.filteredCards = m.filterCards(m.searchInput.Value())
			m.updateCardTable()
		case TabUserSearch:
			m.filteredCollection = m.filterUserCollection(m.searchInput.Value())
		}

		// Reset cursor when search changes
		m.cursor = 0
		return m, cmd
	}

	return m, nil
}

func (m CardGameTabsModel) View() string {
	var b strings.Builder

	// Header with game name
	if m.selectedGame != nil {
		b.WriteString(titleStyle.Render(m.selectedGame.Name+" Collection Manager") + "\n\n")
	}

	// Tab navigation
	tabs := []string{"Collection", "Card Search", "My Collection"}
	var tabStyles []string

	for i, tab := range tabs {
		if Tab(i) == m.currentTab {
			tabStyles = append(tabStyles, focusedStyle.Render("[ "+tab+" ]"))
		} else {
			tabStyles = append(tabStyles, blurredStyle.Render("  "+tab+"  "))
		}
	}

	b.WriteString(strings.Join(tabStyles, " ") + "\n\n")

	// Tab content
	switch m.currentTab {
	case TabCollection:
		b.WriteString(m.renderCollectionTab())
	case TabCardSearch:
		b.WriteString(m.renderCardSearchTab())
	case TabUserSearch:
		b.WriteString(m.renderUserSearchTab())
	}

	// Help text
	b.WriteString("\n")
	// Build dynamic help text from config if available
	settingsKey := "F1"
	nextTab := "Tab"
	prevTab := "Shift+Tab"
	navUp := "↑"
	navDown := "↓"
	backKey := "Q"
	quitKey := "Ctrl+C"
	if m.configManager != nil {
		if k := m.configManager.KeyForAction("settings"); k != "" {
			settingsKey = k
		}
		if k := m.configManager.KeyForAction("nav_next_tab"); k != "" {
			nextTab = k
		}
		if k := m.configManager.KeyForAction("nav_prev_tab"); k != "" {
			prevTab = k
		}
		if k := m.configManager.KeyForAction("nav_up"); k != "" {
			navUp = k
		}
		if k := m.configManager.KeyForAction("nav_down"); k != "" {
			navDown = k
		}
		if k := m.configManager.KeyForAction("back"); k != "" {
			backKey = k
		}
		if k := m.configManager.KeyForAction("quit"); k != "" {
			quitKey = k
		}
	}
	help := fmt.Sprintf("%s: Settings • %s/%s: Switch tabs • %s/%s: Navigate • %s: Back • %s: Quit", settingsKey, prevTab, nextTab, navUp, navDown, backKey, quitKey)
	b.WriteString(helpStyle.Render(help) + "\n")

	content := b.String()
	if m.modal.IsVisible() {
		return content + "\n\n" + m.modal.View()
	}
	return content
}

func (m CardGameTabsModel) renderCollectionTab() string {
	var b strings.Builder

	b.WriteString(focusedStyle.Render("Your Collection Summary") + "\n\n")

	if len(m.filteredCollection) == 0 {
		b.WriteString(blurredStyle.Render("No cards in your collection yet.") + "\n")
		b.WriteString(blurredStyle.Render("Use Card Search to discover cards to add!") + "\n")
	} else {
		// Summary stats
		totalCards := 0
		for _, collection := range m.filteredCollection {
			totalCards += collection.Quantity
		}

		b.WriteString(blurredStyle.Render("Total unique cards: ") +
			focusedStyle.Render(fmt.Sprintf("%d", len(m.filteredCollection))) + "\n")
		b.WriteString(blurredStyle.Render("Total cards: ") +
			focusedStyle.Render(fmt.Sprintf("%d", totalCards)) + "\n\n")

		// Collection list
		b.WriteString(focusedStyle.Render("Recent additions:") + "\n")
		for i, collection := range m.filteredCollection {
			if i >= 10 { // Show only first 10
				break
			}

			style := blurredStyle
			if i == m.cursor {
				style = focusedStyle
			}

			cardName := "Unknown Card"
			if collection.Card != nil {
				cardName = collection.Card.Name
			}

			line := style.Render(cardName + " x" + fmt.Sprintf("%d", collection.Quantity))
			if collection.Condition != "" {
				line += blurredStyle.Render(" (" + collection.Condition + ")")
			}
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}

func (m CardGameTabsModel) renderCardSearchTab() string {
	var b strings.Builder
	b.WriteString(focusedStyle.Render("Search All Cards") + "\n\n")
	b.WriteString(blurredStyle.Render("Search: ") + m.searchInput.View() + "\n\n")
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

	b.WriteString(focusedStyle.Render("Found cards:") + "\n")
	m.cardTable.SetRows(rows)
	b.WriteString(m.cardTable.View())
	return b.String()
}

func (m CardGameTabsModel) renderUserSearchTab() string {
	var b strings.Builder
	b.WriteString(focusedStyle.Render("Search Your Collection") + "\n\n")
	b.WriteString(blurredStyle.Render("Search: ") + m.searchInput.View() + "\n\n")
	if len(m.filteredCollection) == 0 {
		if m.searchInput.Value() == "" {
			b.WriteString(blurredStyle.Render("Type to search your collection...") + "\n")
		} else {
			b.WriteString(blurredStyle.Render("No cards in your collection match your search.") + "\n")
		}
	} else {
		b.WriteString(focusedStyle.Render("Your matching cards:") + "\n")
		var rows []table.Row
		for _, collection := range m.filteredCollection {
			rows = append(rows, collectionToRow(collection))
		}
		m.cardTable.SetRows(rows)
		b.WriteString(m.cardTable.View())
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
	if card.SetID > 0 {
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
	cardNum := ""
	if c.Card != nil {
		name = c.Card.Name
		if c.Card.SetID > 0 {
			setDisplay = fmt.Sprintf("Set#%d", c.Card.SetID)
		}
		rarity = c.Card.Rarity
		cardNum = c.Card.Number
	}
	nameWithQty := FormatNameWithQty(name, c.Quantity)
	return table.Row{
		Truncate(nameWithQty, 25),
		Truncate(setDisplay, 15),
		Truncate(rarity, 12),
		Truncate(cardNum, 8),
	}
}

// handleIncrementQuantity increments the quantity of the selected card
func (m CardGameTabsModel) handleIncrementQuantity() (CardGameTabsModel, tea.Cmd) {
	if len(m.filteredCards) == 0 {
		return m, nil
	}
	selectedRow := m.cardTable.Cursor()
	if selectedRow >= len(m.filteredCards) {
		return m, nil
	}
	card := m.filteredCards[selectedRow]
	if m.tempQuantityChanges == nil {
		m.tempQuantityChanges = make(map[int64]int)
	}
	m.tempQuantityChanges[card.ID]++
	m.updateCardTable()
	return m, nil
}

// handleDecrementQuantity decrements the quantity of the selected card
func (m CardGameTabsModel) handleDecrementQuantity() (CardGameTabsModel, tea.Cmd) {
	if len(m.filteredCards) == 0 {
		return m, nil
	}
	selectedRow := m.cardTable.Cursor()
	if selectedRow >= len(m.filteredCards) {
		return m, nil
	}
	card := m.filteredCards[selectedRow]
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
	for cardID, newQty := range updates {
		m.dbQuantities[cardID] = newQty
	}
	m.tempQuantityChanges = make(map[int64]int)
	m.updateCardTable()
	return m, nil
}
