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

var deckFormatOptions = []string{"standard", "expanded", "unlimited"}

type DeckBuilderModel struct {
	selectedGame        *model.CardGame
	user                *auth.User
	deckService         deck.DeckService
	cardService         cardservice.CardService
	styleManager        *StyleManager
	configManager       *runtimecfg.Manager
	width               int
	height              int
	decks               []model.Deck
	deckCursor          int
	selectedDeck        *model.Deck
	mode                DeckBuilderMode
	nameInput           textinput.Model
	formatIndex         int
	cards               []model.Card
	filteredCards       []model.Card
	searchInput         textinput.Model
	cardTable           table.Model
	focus               DeckBuilderFocus
	tempQuantityChanges map[int64]int
	dbQuantities        map[int64]int
	validationErrors    []deck.DeckValidationError
	modal               ModalModel
	shouldGoBack        bool
	formFocus           int
	exportState         ExportState
	importState         ImportState
}

type saveDeckMsg struct{}
type createDeckMsg struct{}
type deleteDeckMsg struct{}

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
	return DeckBuilderModel{
		selectedGame:        game,
		user:                user,
		deckService:         deckService,
		cardService:         cardSvc,
		styleManager:        sm,
		configManager:       cfg,
		cards:               cards,
		filteredCards:       cards,
		searchInput:         searchInput,
		nameInput:           nameInput,
		cardTable:           cardTable,
		tempQuantityChanges: make(map[int64]int),
		dbQuantities:        make(map[int64]int),
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
			cmd := m.importState.HandleTextInput(keyMsg)
			if cmd == nil {
				cmd = m.importState.HandleKey(keyMsg.String())
			} else {
				_ = m.importState.HandleKey(keyMsg.String())
			}
			return m, cmd
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
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		action := MatchActionOrDefault(m.configManager, s, "")
		if action == "quit" || s == "ctrl+c" {
			return m, tea.Quit
		}
		if m.mode != DeckModeNormal {
			return m.handleFormKeys(msg)
		}
		if isBackKey(action, s) {
			if m.focus == DeckFocusCardPanel {
				m.focus = DeckFocusDeckPanel
				return m, nil
			}
			m.shouldGoBack = true
			return m, nil
		}
		if s == "tab" {
			if m.focus == DeckFocusDeckPanel {
				m.focus = DeckFocusCardPanel
				m.cardTable.Focus()
				m.searchInput.Focus()
			} else {
				m.focus = DeckFocusDeckPanel
				m.cardTable.Blur()
				m.searchInput.Blur()
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
	action := MatchActionOrDefault(m.configManager, s, "")
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
	if s == "n" {
		m.mode = DeckModeCreate
		m.nameInput.SetValue("")
		m.nameInput.Focus()
		m.formatIndex = 0
		m.formFocus = 0
		return m, nil
	}
	if s == "e" && m.selectedDeck != nil {
		m.mode = DeckModeEdit
		m.nameInput.SetValue(m.selectedDeck.Name)
		m.nameInput.Focus()
		for i, f := range deckFormatOptions {
			if f == m.selectedDeck.Format {
				m.formatIndex = i
				break
			}
		}
		m.formFocus = 0
		return m, nil
	}
	if s == "d" && m.selectedDeck != nil {
		m.modal = newModal(
			"Delete Deck",
			fmt.Sprintf("Delete deck '%s'?", m.selectedDeck.Name),
			func() tea.Cmd { return func() tea.Msg { return deleteDeckMsg{} } },
			func() tea.Cmd { return nil },
			m.styleManager, m.width, m.height,
		)
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
	action := MatchActionOrDefault(m.configManager, s, "")
	if action == "nav_up" || s == "up" || s == "k" {
		m.cardTable, _ = m.cardTable.Update(msg)
		return m, nil
	}
	if action == "nav_down" || s == "down" || s == "j" {
		m.cardTable, _ = m.cardTable.Update(msg)
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
	if s == "x" && m.selectedDeck != nil {
		m.exportState = NewExportState("deck", m.selectedDeck.Name, true, m.selectedDeck.Name, m.buildDeckExportRows)
		return m, nil
	}
	if s == "i" && m.selectedDeck != nil {
		m.importState = NewImportState(m.cardService, m.styleManager)
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.filterCards()
	m.updateCardTable()
	return m, cmd
}

func (m DeckBuilderModel) handleFormKeys(msg tea.KeyMsg) (DeckBuilderModel, tea.Cmd) {
	s := msg.String()
	if s == "esc" {
		m.mode = DeckModeNormal
		m.nameInput.Blur()
		return m, nil
	}
	if s == "tab" {
		m.formFocus = (m.formFocus + 1) % 2
		if m.formFocus == 0 {
			m.nameInput.Focus()
		} else {
			m.nameInput.Blur()
		}
		return m, nil
	}
	if m.formFocus == 1 {
		if s == "left" {
			if m.formatIndex > 0 {
				m.formatIndex--
			}
			return m, nil
		}
		if s == "right" {
			if m.formatIndex < len(deckFormatOptions)-1 {
				m.formatIndex++
			}
			return m, nil
		}
		if s == "enter" {
			title := "Create Deck"
			message := fmt.Sprintf("Create deck '%s' (%s)?", m.nameInput.Value(), deckFormatOptions[m.formatIndex])
			if m.mode == DeckModeEdit {
				title = "Update Deck"
				message = fmt.Sprintf("Update deck to '%s' (%s)?", m.nameInput.Value(), deckFormatOptions[m.formatIndex])
			}
			m.modal = newModal(
				title, message,
				func() tea.Cmd { return func() tea.Msg { return createDeckMsg{} } },
				func() tea.Cmd { return nil },
				m.styleManager, m.width, m.height,
			)
			return m, nil
		}
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
			m.dbQuantities = quantities
		} else {
			m.dbQuantities = make(map[int64]int)
		}
		m.tempQuantityChanges = make(map[int64]int)
		m.updateValidation()
		m.updateCardTable()
	}
}

func (m *DeckBuilderModel) filterCards() {
	query := strings.ToLower(m.searchInput.Value())
	if query == "" {
		m.filteredCards = m.cards
		return
	}
	var filtered []model.Card
	for _, c := range m.cards {
		if strings.Contains(strings.ToLower(c.Name), query) ||
			strings.Contains(strings.ToLower(c.Number), query) {
			filtered = append(filtered, c)
		}
	}
	m.filteredCards = filtered
}

func (m *DeckBuilderModel) updateCardTable() {
	var rows []table.Row
	for _, card := range m.filteredCards {
		dbQty := m.dbQuantities[card.ID]
		tempDelta := m.tempQuantityChanges[card.ID]
		setName := ""
		if card.Set != nil {
			setName = card.Set.Name
		}
		rows = append(rows, table.Row{
			Truncate(card.Name, 20),
			Truncate(setName, 12),
			Truncate(card.Rarity, 10),
			Truncate(card.Number, 6),
			fmt.Sprintf("%d", dbQty+tempDelta),
		})
	}
	m.cardTable.SetRows(rows)
}

func (m *DeckBuilderModel) updateValidation() {
	combined := make(map[int64]int)
	for id, qty := range m.dbQuantities {
		combined[id] = qty
	}
	for id, delta := range m.tempQuantityChanges {
		combined[id] += delta
	}
	m.validationErrors = m.deckService.ValidateDeck(m.cards, combined)
}

func (m DeckBuilderModel) getSelectedCard() (model.Card, bool) {
	idx := m.cardTable.Cursor()
	if idx < 0 || idx >= len(m.filteredCards) {
		return model.Card{}, false
	}
	return m.filteredCards[idx], true
}

func (m DeckBuilderModel) handleIncrement() (DeckBuilderModel, tea.Cmd) {
	if m.selectedDeck == nil {
		return m, nil
	}
	card, ok := m.getSelectedCard()
	if !ok {
		return m, nil
	}
	m.tempQuantityChanges[card.ID]++
	m.updateValidation()
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
	total := m.dbQuantities[card.ID] + m.tempQuantityChanges[card.ID]
	if total <= 0 {
		return m, nil
	}
	m.tempQuantityChanges[card.ID]--
	m.updateValidation()
	m.updateCardTable()
	return m, nil
}

func (m DeckBuilderModel) handleSave() (DeckBuilderModel, tea.Cmd) {
	if m.selectedDeck == nil || len(m.tempQuantityChanges) == 0 {
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
	m.modal = newModal(
		"Save Deck",
		fmt.Sprintf("Save %d card changes to '%s'?", changeCount, m.selectedDeck.Name),
		func() tea.Cmd { return func() tea.Msg { return saveDeckMsg{} } },
		func() tea.Cmd { return nil },
		m.styleManager, m.width, m.height,
	)
	return m, nil
}

func (m DeckBuilderModel) performSave() (DeckBuilderModel, tea.Cmd) {
	if m.selectedDeck == nil {
		return m, nil
	}
	updates := make(map[int64]int)
	for cardID, delta := range m.tempQuantityChanges {
		updates[cardID] = m.dbQuantities[cardID] + delta
	}
	ctx := context.Background()
	err := m.deckService.UpsertDeckCardBatch(ctx, m.selectedDeck.ID, updates)
	if err != nil {
		return m, nil
	}
	for id, qty := range updates {
		m.dbQuantities[id] = qty
	}
	m.tempQuantityChanges = make(map[int64]int)
	m.updateValidation()
	m.updateCardTable()
	return m, nil
}

func (m DeckBuilderModel) performCreateDeck() (DeckBuilderModel, tea.Cmd) {
	if m.user == nil || m.selectedGame == nil {
		return m, nil
	}
	name := m.nameInput.Value()
	format := deckFormatOptions[m.formatIndex]
	ctx := context.Background()
	if m.mode == DeckModeEdit && m.selectedDeck != nil {
		_ = m.deckService.UpdateDeck(ctx, m.selectedDeck.ID, name, format)
	} else {
		_, _ = m.deckService.CreateDeck(ctx, m.user.ID, m.selectedGame.ID, name, format)
	}
	m.mode = DeckModeNormal
	m.nameInput.Blur()
	m.refreshDecks()
	return m, nil
}

func (m DeckBuilderModel) performDeleteDeck() (DeckBuilderModel, tea.Cmd) {
	if m.selectedDeck == nil {
		return m, nil
	}
	ctx := context.Background()
	_ = m.deckService.DeleteDeck(ctx, m.selectedDeck.ID)
	m.selectedDeck = nil
	m.dbQuantities = make(map[int64]int)
	m.tempQuantityChanges = make(map[int64]int)
	m.refreshDecks()
	if m.deckCursor >= len(m.decks) && m.deckCursor > 0 {
		m.deckCursor--
	}
	if len(m.decks) > 0 {
		m.selectCurrentDeck()
	}
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
	name := ""
	if m.selectedGame != nil {
		name = m.selectedGame.Name + " "
	}
	return m.styleManager.GetTitleStyle().Render(name + "Deck Builder")
}

func (m DeckBuilderModel) renderBody(availableHeight int) string {
	contentWidth := m.width - frameBorderSize - framePaddingX*2
	if contentWidth < 20 {
		contentWidth = 20
	}
	leftWidth := contentWidth * 35 / 100
	rightWidth := contentWidth - leftWidth
	leftContent := m.renderDeckPanel(leftWidth, availableHeight)
	rightContent := m.renderCardPanel(rightWidth, availableHeight)
	leftPanel := RenderPanel(m.styleManager, leftContent, leftWidth, availableHeight, m.focus == DeckFocusDeckPanel, 1, 0)
	rightPanel := RenderPanel(m.styleManager, rightContent, rightWidth, availableHeight, m.focus == DeckFocusCardPanel, 1, 0)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m DeckBuilderModel) renderDeckPanel(width, height int) string {
	var b strings.Builder
	b.WriteString(m.styleManager.GetTitleStyle().Render("My Decks") + "\n")
	if m.mode == DeckModeCreate || m.mode == DeckModeEdit {
		label := "New Deck"
		if m.mode == DeckModeEdit {
			label = "Edit Deck"
		}
		b.WriteString(m.styleManager.GetFocusedStyle().Render(label) + "\n")
		b.WriteString("Name: " + m.nameInput.View() + "\n")
		b.WriteString("Format: ")
		for i, f := range deckFormatOptions {
			if i == m.formatIndex {
				if m.formFocus == 1 {
					b.WriteString(m.styleManager.GetFocusedStyle().Render("< " + f + " >"))
				} else {
					b.WriteString(m.styleManager.GetTitleStyle().Render("[" + f + "]"))
				}
			} else {
				b.WriteString(m.styleManager.GetBlurredStyle().Render(" " + f + " "))
			}
		}
		b.WriteString("\n")
		return b.String()
	}
	if len(m.decks) == 0 {
		b.WriteString(m.styleManager.GetBlurredStyle().Render("No decks yet. Press 'n' to create.") + "\n")
		return b.String()
	}
	for i, d := range m.decks {
		line := fmt.Sprintf("%s (%s)", d.Name, d.Format)
		b.WriteString(RenderListItem(line, m.deckCursor == i && m.focus == DeckFocusDeckPanel))
	}
	if m.selectedDeck != nil {
		b.WriteString("\n")
		totalCards := 0
		for _, qty := range m.dbQuantities {
			totalCards += qty
		}
		for _, delta := range m.tempQuantityChanges {
			totalCards += delta
		}
		b.WriteString(m.styleManager.GetNoStyle().Render(fmt.Sprintf("Cards: %d/60", totalCards)) + "\n")
		if len(m.validationErrors) == 0 && totalCards == 60 {
			b.WriteString(m.styleManager.applyBGFG(lipgloss.NewStyle().Foreground(lipgloss.Color("34"))).Render(SuccessIcon+" Valid deck") + "\n")
		} else {
			for _, ve := range m.validationErrors {
				b.WriteString(m.styleManager.GetErrorStyle().Render(FailureIcon+" "+ve.Message) + "\n")
			}
		}
	}
	return b.String()
}

func (m DeckBuilderModel) renderCardPanel(width, height int) string {
	var b strings.Builder
	if m.selectedDeck == nil {
		b.WriteString(m.styleManager.GetBlurredStyle().Render("Select a deck to add cards.") + "\n")
		return b.String()
	}
	b.WriteString(m.styleManager.GetTitleStyle().Render("Add Cards to: "+m.selectedDeck.Name) + "\n")
	b.WriteString(m.styleManager.GetBlurredStyle().Render("Search: ") + m.searchInput.View() + "\n")
	tableHeight := CalcTableHeight(height, 4, 3)
	// panelPadX=1, borderOverhead=2: inner table area = width - 4
	tableWidth := max(width-4, 20)
	m.cardTable.SetColumns(scaledDeckColumns(tableWidth))
	m.cardTable.SetHeight(tableHeight)
	b.WriteString(m.cardTable.View())
	return b.String()
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
		footer = m.styleManager.GetHelpStyle().Render("Tab: Switch field • Enter: Confirm • Esc: Cancel")
	} else if m.focus == DeckFocusCardPanel {
		footer = m.styleManager.GetHelpStyle().Render("Tab: Switch panel • " + hb.Build(
			KeyItem{"increment_quantity", "+", "Add"},
			KeyItem{"decrement_quantity", "Delete", "Remove"},
			KeyItem{"save", "Ctrl+S", "Save"},
		) + " • x: Export • i: Import • " + hb.Build(KeyItem{"back", "Q", "Back"}))
	} else {
		footer = m.styleManager.GetHelpStyle().Render("Tab: Switch panel • n: New • e: Edit • d: Delete • " + hb.Build(KeyItem{"back", "Q", "Back"}))
	}
	if m.exportState.statusMsg != "" {
		footer = m.styleManager.GetHelpStyle().Render(m.exportState.statusMsg) + "  " + footer
	}
	return footer
}

func (m *DeckBuilderModel) buildDeckExportRows() []export.CardRow {
	return buildCardExportRows(m.cards, m.dbQuantities, m.tempQuantityChanges)
}

// applyImport merges the imported card quantities into tempQuantityChanges.
// Quantities are staged so the user can review before saving with Ctrl+S.
func (m DeckBuilderModel) applyImport(msg ImportApplyMsg) (DeckBuilderModel, tea.Cmd) {
	if m.selectedDeck == nil {
		return m, nil
	}
	for cardID, qty := range msg.Quantities {
		m.tempQuantityChanges[cardID] += qty
	}
	m.updateValidation()
	m.updateCardTable()
	return m, nil
}
