package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/laiambryant/tui-cardman/internal/auth"
	"github.com/laiambryant/tui-cardman/internal/export"
	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
	cardservice "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/deck"
)

type DeckBuilderMode int

const (
	DeckModeNormal DeckBuilderMode = iota
	DeckModeCreate
	DeckModeEdit
)

type DeckBuilderFocus int

const (
	DeckFocusDeckPanel DeckBuilderFocus = iota
	DeckFocusCardPanel
)

// cardPanelSubFocus tracks which sub-panel of the right panel is active.
type cardPanelSubFocus int

const (
	cardSubFocusSearch cardPanelSubFocus = iota // top: search/add cards table
	cardSubFocusDeck                            // bottom: cards currently in deck
)

type DeckBuilderModel struct {
	selectedGame      *model.CardGame
	user              *auth.User
	deckService       deck.DeckService
	cardService       cardservice.CardService
	styleManager      *StyleManager
	configManager     *runtimecfg.Manager
	width             int
	height            int
	decks             []model.Deck
	deckCursor        int
	selectedDeck      *model.Deck
	mode              DeckBuilderMode
	nameInput         textinput.Model
	cards             []model.Card
	filteredCards     []model.Card
	searchInput       textinput.Model
	cardTable         table.Model
	deckContentsTable table.Model
	cardSubFocus      cardPanelSubFocus
	focus             DeckBuilderFocus
	quantities        QuantityTracker
	modal             ModalModel
	shouldGoBack      bool
	exportState       ExportState
	importState       ImportState
	searchCache       *SearchCache
	cardPagination    Pagination
	mutationPending   bool
	mutationStatus    string
}

type (
	saveDeckMsg     struct{}
	createDeckMsg   struct{}
	deleteDeckMsg   struct{}
	deckMutationMsg struct {
		operation string
		err       error
		updates   map[int64]int
		deckID    int64
		name      string
	}
)

func NewDeckBuilderModel(game *model.CardGame, user *auth.User, deckService deck.DeckService, cardSvc cardservice.CardService, cards []model.Card, cfg *runtimecfg.Manager, sm *StyleManager) DeckBuilderModel {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search cards..."
	searchInput.Width = 30
	sm.ApplyTextInputStyles(&searchInput)
	nameInput := textinput.New()
	nameInput.Placeholder = "Deck name..."
	nameInput.Width = 30
	sm.ApplyTextInputStyles(&nameInput)
	columns := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Set", Width: 12},
		{Title: "Rarity", Width: 10},
		{Title: "#", Width: 6},
		{Title: "Qty", Width: 5},
	}
	cardTable := NewStyledTable(columns, 10, true, sm)
	deckContentsColumns := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Set", Width: 12},
		{Title: "Rarity", Width: 10},
		{Title: "#", Width: 6},
		{Title: "Qty", Width: 5},
	}
	deckContentsTable := NewStyledTable(deckContentsColumns, 5, false, sm)
	return DeckBuilderModel{
		selectedGame:      game,
		user:              user,
		deckService:       deckService,
		cardService:       cardSvc,
		styleManager:      sm,
		configManager:     cfg,
		cards:             cards,
		filteredCards:     cards,
		searchInput:       searchInput,
		nameInput:         nameInput,
		cardTable:         cardTable,
		deckContentsTable: deckContentsTable,
		quantities:        newQuantityTracker(),
		searchCache:       NewSearchCache(),
		cardPagination:    NewPagination(50),
	}
}

func (m DeckBuilderModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m DeckBuilderModel) Update(msg tea.Msg) (DeckBuilderModel, tea.Cmd) {
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
	if m.exportState.active {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			cmd := m.exportState.HandleKey(keyMsg.String())
			return m, cmd
		}
	}
	if m.importState.active {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			return m, m.importState.Update(keyMsg)
		}
	}
	switch msg := msg.(type) {
	case exportDoneMsg:
		m.exportState.HandleResult(msg)
		return m, nil
	case importReadyMsg:
		m.importState.HandleResult(msg)
		return m, nil
	case ImportApplyMsg:
		return m.applyImport(msg)
	case importCancelMsg:
		return m, nil
	case saveDeckMsg:
		return m.performSave()
	case createDeckMsg:
		return m.performCreateDeck()
	case deleteDeckMsg:
		return m.performDeleteDeck()
	case deckMutationMsg:
		return m.handleMutationResult(msg)
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.mutationPending {
			return m, nil
		}
		s := msg.String()
		action := GetAction(m.configManager, s)
		if m.mode != DeckModeNormal {
			return m.handleFormKeys(msg)
		}
		searchFocused := m.focus == DeckFocusCardPanel && m.cardSubFocus == cardSubFocusSearch
		// While the search box has focus only Esc goes back, so "q" can be typed.
		if searchFocused && isBackKeyWhileTyping(action, s) || !searchFocused && isBackKey(action, s) {
			if m.focus == DeckFocusCardPanel {
				if m.cardSubFocus == cardSubFocusDeck {
					m.cardSubFocus = cardSubFocusSearch
					m.cardTable.Focus()
					m.deckContentsTable.Blur()
					m.searchInput.Focus()
				} else {
					m.focus = DeckFocusDeckPanel
					m.cardTable.Blur()
					m.deckContentsTable.Blur()
					m.searchInput.Blur()
				}
				return m, nil
			}
			m.shouldGoBack = true
			return m, nil
		}
		if s == "tab" {
			if m.focus == DeckFocusDeckPanel {
				m.focus = DeckFocusCardPanel
				m.cardSubFocus = cardSubFocusSearch
				m.cardTable.Focus()
				m.deckContentsTable.Blur()
				m.searchInput.Focus()
			} else {
				// cycle sub-focus within the right panel; on second tab go back to deck panel
				if m.cardSubFocus == cardSubFocusSearch {
					m.cardSubFocus = cardSubFocusDeck
					m.cardTable.Blur()
					m.deckContentsTable.Focus()
					m.searchInput.Blur()
				} else {
					m.focus = DeckFocusDeckPanel
					m.cardSubFocus = cardSubFocusSearch
					m.cardTable.Blur()
					m.deckContentsTable.Blur()
					m.searchInput.Blur()
				}
			}
			return m, nil
		}
		if m.focus == DeckFocusDeckPanel {
			return m.handleDeckPanelKeys(msg)
		}
		return m.handleCardPanelKeys(msg)
	}
	return m, nil
}

func (m DeckBuilderModel) handleDeckPanelKeys(msg tea.KeyMsg) (DeckBuilderModel, tea.Cmd) {
	s := msg.String()
	action := GetAction(m.configManager, s)
	if action == "nav_up" || s == "up" || s == "k" {
		if m.deckCursor > 0 {
			m.deckCursor--
			m.selectCurrentDeck()
		}
		return m, nil
	}
	if action == "nav_down" || s == "down" || s == "j" {
		if m.deckCursor < len(m.decks)-1 {
			m.deckCursor++
			m.selectCurrentDeck()
		}
		return m, nil
	}
	if action == "create_new" {
		m.mode = DeckModeCreate
		m.nameInput.SetValue("")
		m.nameInput.Focus()
		return m, nil
	}
	if action == "edit" && m.selectedDeck != nil {
		m.mode = DeckModeEdit
		m.nameInput.SetValue(m.selectedDeck.Name)
		m.nameInput.Focus()
		return m, nil
	}
	if action == "delete" && m.selectedDeck != nil {
		m.modal = newConfirmModal("Delete Deck", fmt.Sprintf("Delete deck '%s'?", m.selectedDeck.Name), deleteDeckMsg{}, m.styleManager, m.width, m.height)
		return m, nil
	}
	if isSelectKey(action, s) && len(m.decks) > 0 {
		m.focus = DeckFocusCardPanel
		m.cardTable.Focus()
		m.searchInput.Focus()
		return m, nil
	}
	return m, nil
}

func (m DeckBuilderModel) handleCardPanelKeys(msg tea.KeyMsg) (DeckBuilderModel, tea.Cmd) {
	s := msg.String()
	action := GetAction(m.configManager, s)
	// Arrow navigation — always safe
	if action == "nav_up" || s == "up" {
		if m.cardSubFocus == cardSubFocusDeck {
			m.deckContentsTable, _ = m.deckContentsTable.Update(msg)
		} else {
			m.cardTable, _ = m.cardTable.Update(msg)
		}
		return m, nil
	}
	if action == "nav_down" || s == "down" {
		if m.cardSubFocus == cardSubFocusDeck {
			if m.deckContentsTable.Cursor() < len(m.deckContentsTable.Rows())-1 {
				m.deckContentsTable, _ = m.deckContentsTable.Update(msg)
			}
		} else {
			if m.cardTable.Cursor() < len(m.cardTable.Rows())-1 {
				m.cardTable, _ = m.cardTable.Update(msg)
			}
		}
		return m, nil
	}
	// Modifier-key shortcuts — safe regardless of search focus
	if action == "page_next" {
		m.cardPagination.NextPage()
		m.updateCardTable()
		m.cardTable.SetCursor(0)
		return m, nil
	}
	if action == "page_prev" {
		m.cardPagination.PrevPage()
		m.updateCardTable()
		m.cardTable.SetCursor(0)
		return m, nil
	}
	if action == "increment_quantity" {
		return m.handleIncrement()
	}
	if action == "decrement_quantity" {
		return m.handleDecrement()
	}
	if action == "save" {
		return m.handleSave()
	}
	if action == "export" && m.selectedDeck != nil {
		m.exportState = NewExportState("deck", m.selectedDeck.Name, true, m.selectedDeck.Name, m.buildDeckExportRows)
		return m, nil
	}
	if action == "import" && m.selectedDeck != nil {
		m.importState = NewImportState(m.cardService, m.selectedGame.ID, m.styleManager)
		return m, nil
	}
	// Forward non-modifier printable keys to the search textinput when it is focused
	if m.cardSubFocus == cardSubFocusSearch && !isModifierKey(s) {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.filterCards()
		m.cardPagination.Reset()
		m.updateCardTable()
		m.cardTable.SetCursor(0)
		return m, cmd
	}
	return m, nil
}

func (m DeckBuilderModel) handleFormKeys(msg tea.KeyMsg) (DeckBuilderModel, tea.Cmd) {
	s := msg.String()
	if s == "esc" {
		m.mode = DeckModeNormal
		m.nameInput.Blur()
		return m, nil
	}
	if s == "enter" {
		title := "Create Deck"
		message := fmt.Sprintf("Create deck '%s'?", m.nameInput.Value())
		if m.mode == DeckModeEdit {
			title = "Update Deck"
			message = fmt.Sprintf("Update deck to '%s'?", m.nameInput.Value())
		}
		m.modal = newConfirmModal(title, message, createDeckMsg{}, m.styleManager, m.width, m.height)
		return m, nil
	}
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m *DeckBuilderModel) selectCurrentDeck() {
	if m.deckCursor >= 0 && m.deckCursor < len(m.decks) {
		d := m.decks[m.deckCursor]
		m.selectedDeck = &d
		quantities, err := m.deckService.GetAllQuantitiesForDeck(d.ID)
		if err == nil {
			m.quantities.load(quantities)
		} else {
			m.quantities.reset()
		}
		m.updateCardTable()
	}
}

func (m *DeckBuilderModel) filterCards() {
	m.filteredCards = filterCardsByQueryCached(m.cards, m.searchInput.Value(), m.searchCache)
}

func (m *DeckBuilderModel) deckVCS(width int) VisibleColumnSet {
	return BuildVisibleColumnSet(DeckColumns, GetVisibleColumns(m.configManager), GetColumnOrder(m.configManager), width)
}

func (m *DeckBuilderModel) updateCardTable() {
	m.cardPagination.TotalItems = len(m.filteredCards)
	start, end := m.cardPagination.Slice()
	page := m.filteredCards[start:end]
	vcs := m.deckVCS(80)
	m.cardTable.SetColumns(vcs.Columns)
	m.cardTable.SetRows(buildCardRows(page, m.quantities.db, m.quantities.temp, vcs))
	m.deckContentsTable.SetColumns(vcs.Columns)
	m.updateDeckContentsTable(vcs)
}

func (m *DeckBuilderModel) updateDeckContentsTable(vcs VisibleColumnSet) {
	var rows []table.Row
	for _, card := range m.cards {
		qty := m.quantities.total(card.ID)
		if qty <= 0 {
			continue
		}
		rows = append(rows, vcs.BuildRow(CardToDataMap(card, 0, qty)))
	}
	m.deckContentsTable.SetRows(rows)
}

func (m DeckBuilderModel) getSelectedCard() (model.Card, bool) {
	idx := m.cardTable.Cursor()
	if idx < 0 {
		return model.Card{}, false
	}
	actualIndex := m.cardPagination.CurrentPage*m.cardPagination.PageSize + idx
	if actualIndex >= len(m.filteredCards) {
		return model.Card{}, false
	}
	return m.filteredCards[actualIndex], true
}

func (m DeckBuilderModel) handleIncrement() (DeckBuilderModel, tea.Cmd) {
	if m.selectedDeck == nil {
		return m, nil
	}
	card, ok := m.getSelectedCard()
	if !ok {
		return m, nil
	}
	m.quantities.increment(card.ID)
	m.updateCardTable()
	return m, nil
}

func (m DeckBuilderModel) handleDecrement() (DeckBuilderModel, tea.Cmd) {
	if m.selectedDeck == nil {
		return m, nil
	}
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

func (m DeckBuilderModel) handleSave() (DeckBuilderModel, tea.Cmd) {
	if m.selectedDeck == nil || m.quantities.pendingCount() == 0 {
		return m, nil
	}
	m.modal = newConfirmModal("Save Deck",
		fmt.Sprintf("Save %d card changes to '%s'?", m.quantities.pendingCount(), m.selectedDeck.Name),
		saveDeckMsg{}, m.styleManager, m.width, m.height)
	return m, nil
}

func (m DeckBuilderModel) performSave() (DeckBuilderModel, tea.Cmd) {
	if m.selectedDeck == nil {
		return m, nil
	}
	updates := m.quantities.buildUpdates()
	deckID := m.selectedDeck.ID
	service := m.deckService
	m.mutationPending = true
	m.mutationStatus = "Saving deck changes..."
	return m, func() tea.Msg {
		err := service.UpsertDeckCardBatch(context.Background(), deckID, updates)
		return deckMutationMsg{operation: "save cards", err: err, updates: updates, deckID: deckID}
	}
}

func (m DeckBuilderModel) performCreateDeck() (DeckBuilderModel, tea.Cmd) {
	if m.user == nil || m.selectedGame == nil {
		return m, nil
	}
	name := m.nameInput.Value()
	var editID int64
	if m.mode == DeckModeEdit && m.selectedDeck != nil {
		editID = m.selectedDeck.ID
	}
	userID := m.user.ID
	gameID := m.selectedGame.ID
	service := m.deckService
	operation := "create deck"
	if editID != 0 {
		operation = "update deck"
	}
	m.mutationPending = true
	m.mutationStatus = "Saving deck..."
	return m, func() tea.Msg {
		var err error
		if editID != 0 {
			err = service.UpdateDeck(context.Background(), editID, name)
		} else {
			_, err = service.CreateDeck(context.Background(), userID, gameID, name)
		}
		return deckMutationMsg{operation: operation, err: err, deckID: editID, name: name}
	}
}

func (m DeckBuilderModel) performDeleteDeck() (DeckBuilderModel, tea.Cmd) {
	if m.selectedDeck == nil {
		return m, nil
	}
	deckID := m.selectedDeck.ID
	service := m.deckService
	m.mutationPending = true
	m.mutationStatus = "Deleting deck..."
	return m, func() tea.Msg {
		err := service.DeleteDeck(context.Background(), deckID)
		return deckMutationMsg{operation: "delete deck", err: err, deckID: deckID}
	}
}

func (m DeckBuilderModel) handleMutationResult(msg deckMutationMsg) (DeckBuilderModel, tea.Cmd) {
	m.mutationPending = false
	if msg.err != nil {
		m.mutationStatus = fmt.Sprintf("Failed to %s: %v", msg.operation, msg.err)
		return m, nil
	}

	if msg.operation == "save cards" {
		if m.selectedDeck != nil && m.selectedDeck.ID == msg.deckID {
			m.quantities.commit(msg.updates)
			m.searchCache.Invalidate()
			m.updateCardTable()
		}
		m.mutationStatus = "Deck saved"
		return m, nil
	}

	if msg.operation == "delete deck" && m.selectedDeck != nil && m.selectedDeck.ID == msg.deckID {
		m.selectedDeck = nil
		m.quantities.reset()
	}
	m.mode = DeckModeNormal
	m.nameInput.Blur()
	m.refreshDecks()
	switch msg.operation {
	case "update deck":
		for i, d := range m.decks {
			if d.ID == msg.deckID {
				m.deckCursor = i
				break
			}
		}
	case "create deck":
		for i, d := range m.decks {
			if d.Name == msg.name {
				m.deckCursor = i
				break
			}
		}
	case "delete deck":
		if m.deckCursor >= len(m.decks) && m.deckCursor > 0 {
			m.deckCursor--
		}
	}
	if len(m.decks) > 0 {
		m.selectCurrentDeck()
	}
	m.mutationStatus = "Changes saved"
	return m, nil
}

func (m *DeckBuilderModel) refreshDecks() {
	if m.user == nil || m.selectedGame == nil {
		return
	}
	decks, err := m.deckService.GetDecksByUserAndGame(m.user.ID, m.selectedGame.ID)
	if err == nil {
		m.decks = decks
	}
}

func (m DeckBuilderModel) View() string {
	header := m.renderHeader()
	footer := m.renderFooter()
	return RenderFramedWithModal(header, footer, m.renderBody, m.width, m.height, m.styleManager, &m.modal)
}

func (m DeckBuilderModel) renderHeader() string {
	if m.selectedGame != nil {
		return RenderBreadcrumbs(m.styleManager, m.selectedGame.Name, "Deck Builder")
	}
	return RenderBreadcrumbs(m.styleManager, "Deck Builder")
}

func (m DeckBuilderModel) renderBody(availableHeight int) string {
	contentWidth := m.width - frameBorderSize - framePaddingX*2
	if contentWidth < 20 {
		contentWidth = 20
	}
	leftWidth := contentWidth * 35 / 100
	rightWidth := contentWidth - leftWidth
	leftPanel := m.renderDeckPanel(leftWidth, availableHeight)
	rightPanel := m.renderCardPanel(rightWidth, availableHeight)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m DeckBuilderModel) renderDeckPanel(width, height int) string {
	var b strings.Builder
	focused := m.focus == DeckFocusDeckPanel
	if m.mode == DeckModeCreate || m.mode == DeckModeEdit {
		title := "New Deck"
		if m.mode == DeckModeEdit {
			title = "Edit Deck"
		}
		b.WriteString("Name: " + m.nameInput.View() + "\n")
		return RenderTitledPanel(m.styleManager, title, b.String(), width, height, focused, 1, 0)
	}
	if len(m.decks) == 0 {
		b.WriteString(m.styleManager.GetBlurredStyle().Render("No decks yet. Press 'n' to create.") + "\n")
		return RenderTitledPanel(m.styleManager, "My Decks", b.String(), width, height, focused, 1, 0)
	}
	for i, d := range m.decks {
		b.WriteString(RenderListItem(d.Name, m.deckCursor == i && focused))
	}
	if m.selectedDeck != nil {
		b.WriteString("\n")
		b.WriteString(m.styleManager.GetHelpStyle().Render("Cards ") + m.styleManager.GetSecondaryStyle().Bold(true).Render(fmt.Sprintf("%d", m.quantities.totalCards())) + "\n")
	}
	return RenderTitledPanel(m.styleManager, "My Decks", b.String(), width, height, focused, 1, 0)
}

func (m DeckBuilderModel) renderCardPanel(width, height int) string {
	tableWidth := max(width-4, 20)
	if m.selectedDeck == nil {
		return m.styleManager.GetBlurredStyle().Render("Select a deck to add cards.") + "\n"
	}
	topHeight := max(height*6/10, 5)
	bottomHeight := max(height-topHeight, 5)
	searchFocused := m.cardSubFocus == cardSubFocusSearch
	titleStyle := m.styleManager.GetTitleStyle()
	var top strings.Builder
	if searchFocused {
		top.WriteString(titleStyle.Render("Add Cards: "+m.selectedDeck.Name) + "\n")
	} else {
		top.WriteString(m.styleManager.GetBlurredStyle().Render("Add Cards: "+m.selectedDeck.Name) + "\n")
	}
	top.WriteString(renderSearchPrompt(m.styleManager, m.searchInput.View(), m.cardPagination.StatusText()) + "\n")
	vcs := m.deckVCS(tableWidth)
	if len(m.cardTable.Rows()) == 0 {
		top.WriteString(m.styleManager.GetBlurredStyle().Render("No cards match your search.") + "\n")
	} else {
		searchTableHeight := CalcTableHeight(topHeight, 2, 3)
		m.cardTable.SetColumns(vcs.Columns)
		m.cardTable.SetHeight(searchTableHeight)
		top.WriteString(m.cardTable.View())
	}
	deckFocused := m.cardSubFocus == cardSubFocusDeck
	var bottom strings.Builder
	if deckFocused {
		bottom.WriteString(titleStyle.Render("Deck Contents") + "\n")
	} else {
		bottom.WriteString(m.styleManager.GetBlurredStyle().Render("Deck Contents") + "\n")
	}
	if len(m.deckContentsTable.Rows()) == 0 {
		bottom.WriteString(m.styleManager.GetBlurredStyle().Render("No cards in deck yet.") + "\n")
	} else {
		deckTableHeight := CalcTableHeight(bottomHeight, 1, 3)
		m.deckContentsTable.SetColumns(vcs.Columns)
		m.deckContentsTable.SetHeight(deckTableHeight)
		bottom.WriteString(m.deckContentsTable.View())
	}
	return renderSplitCardPanel(m.styleManager, top.String(), searchFocused, bottom.String(), deckFocused, width, topHeight, bottomHeight)
}

func (m DeckBuilderModel) renderFooter() string {
	if m.exportState.active {
		return m.exportState.Render(m.styleManager)
	}
	if m.importState.active {
		return m.importState.Render(m.styleManager)
	}
	hb := NewHelpBuilder(m.configManager)
	var footer string
	if m.mode != DeckModeNormal {
		footer = JoinHelpHints(
			RenderHelpHint("Enter", "Confirm"),
			RenderHelpHint("Esc", "Cancel"),
		)
	} else if m.focus == DeckFocusCardPanel {
		footer = JoinHelpHints(
			RenderHelpHint("Tab", "Switch panel"),
			hb.Build(
				KeyItem{"increment_quantity", "+", "Add"},
				KeyItem{"decrement_quantity", "Delete", "Remove"},
				KeyItem{"save", "Ctrl+S", "Save"},
			),
			hb.Build(KeyItem{"export", "x", "Export"}, KeyItem{"import", "i", "Import"}),
			hb.Pair("page_next", "Ctrl+N", "page_prev", "Ctrl+P", "Page"),
			hb.Build(KeyItem{"quit_alt", "Esc", "Back"}),
		)
	} else {
		footer = JoinHelpHints(
			RenderHelpHint("Tab", "Switch panel"),
			hb.Build(KeyItem{"create_new", "n", "New"}, KeyItem{"edit", "e", "Edit"}, KeyItem{"delete", "d", "Delete"}),
			hb.Build(KeyItem{"back", "Q", "Back"}),
		)
	}
	if m.exportState.statusMsg != "" {
		footer = m.styleManager.GetHelpStyle().Render(m.exportState.statusMsg) + "  " + footer
	}
	if m.mutationStatus != "" {
		footer = m.styleManager.GetHelpStyle().Render(m.mutationStatus) + "  " + footer
	}
	return footer
}

func (m *DeckBuilderModel) buildDeckExportRows() []export.CardRow {
	return buildCardExportRows(m.cards, m.quantities.db, m.quantities.temp)
}

func (m DeckBuilderModel) applyImport(msg ImportApplyMsg) (DeckBuilderModel, tea.Cmd) {
	if m.selectedDeck == nil {
		return m, nil
	}
	m.quantities.applyImport(msg.Quantities)
	m.updateCardTable()
	return m, nil
}
