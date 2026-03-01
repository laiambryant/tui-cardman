package tui

import (
	"context"
	"fmt"
	"maps"
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
	listservice "github.com/laiambryant/tui-cardman/internal/services/list"
)

type ListsFocus int

const (
	ListsFocusListPanel ListsFocus = iota
	ListsFocusCardPanel
)

type ListsMode int

const (
	ListsModeNormal ListsMode = iota
	ListsModeCreate
	ListsModeEdit
)

var listColorOptions = []struct {
	Name  string
	Color string
}{
	{"White", "#FFFFFF"},
	{"Red", "#FF5555"},
	{"Green", "#55FF55"},
	{"Blue", "#5555FF"},
	{"Yellow", "#FFFF55"},
	{"Purple", "#FF55FF"},
	{"Cyan", "#55FFFF"},
	{"Orange", "#FFAA55"},
}

type (
	saveListCardsMsg struct{}
	deleteListMsg    struct{ listID int64 }
	createListMsg    struct{}
	updateListMsg    struct{}
)

type ListsModel struct {
	selectedGame        *model.CardGame
	user                *auth.User
	listService         listservice.ListService
	cardService         cardservice.CardService
	styleManager        *StyleManager
	configManager       *runtimecfg.Manager
	width               int
	height              int
	lists               []model.UserList
	listCursor          int
	selectedList        *model.UserList
	mode                ListsMode
	nameInput           textinput.Model
	descInput           textinput.Model
	colorIndex          int
	formFocus           int
	cards               []model.Card
	filteredCards       []model.Card
	searchInput         textinput.Model
	cardTable           table.Model
	focus               ListsFocus
	tempQuantityChanges map[int64]int
	dbQuantities        map[int64]int
	modal               ModalModel
	shouldGoBack        bool
	editingListID       int64
	exportState         ExportState
	importState         ImportState
}

func NewListsModel(game *model.CardGame, user *auth.User, listSvc listservice.ListService, cardSvc cardservice.CardService, cards []model.Card, cfg *runtimecfg.Manager, sm *StyleManager) ListsModel {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search cards..."
	searchInput.Width = 30
	sm.ApplyTextInputStyles(&searchInput)
	nameInput := textinput.New()
	nameInput.Placeholder = "List name..."
	nameInput.Width = 25
	nameInput.CharLimit = 50
	sm.ApplyTextInputStyles(&nameInput)
	descInput := textinput.New()
	descInput.Placeholder = "Description..."
	descInput.Width = 25
	descInput.CharLimit = 100
	sm.ApplyTextInputStyles(&descInput)
	columns := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Expansion", Width: 15},
		{Title: "Rarity", Width: 12},
		{Title: "Card #", Width: 8},
		{Title: "Quantity", Width: 8},
	}
	cardTable := NewStyledTable(columns, 10, true, sm)
	return ListsModel{
		selectedGame:        game,
		user:                user,
		listService:         listSvc,
		cardService:         cardSvc,
		styleManager:        sm,
		configManager:       cfg,
		cards:               cards,
		filteredCards:       cards,
		searchInput:         searchInput,
		nameInput:           nameInput,
		descInput:           descInput,
		cardTable:           cardTable,
		tempQuantityChanges: make(map[int64]int),
		dbQuantities:        make(map[int64]int),
	}
}

func (m ListsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ListsModel) Update(msg tea.Msg) (ListsModel, tea.Cmd) {
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
	case saveListCardsMsg:
		return m.performSaveListCards()
	case deleteListMsg:
		return m.performDeleteList(msg.listID)
	case createListMsg:
		return m.performCreateList()
	case updateListMsg:
		return m.performUpdateList()
	case tea.KeyMsg:
		s := msg.String()
		action := MatchActionOrDefault(m.configManager, s, "")
		if action == "quit" || s == "ctrl+c" {
			return m, tea.Quit
		}
		if m.mode == ListsModeCreate || m.mode == ListsModeEdit {
			return m.handleFormKeys(s, action)
		}
		if m.focus == ListsFocusListPanel {
			return m.handleListPanelKeys(s, action)
		}
		return m.handleCardPanelKeys(msg, s, action)
	}
	if m.focus == ListsFocusCardPanel {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			s := keyMsg.String()
			action := MatchActionOrDefault(m.configManager, s, "")
			if action != "nav_up" && action != "nav_down" && s != "k" && s != "j" {
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.filteredCards = m.filterListCards(m.searchInput.Value())
				m.updateListCardTable()
				m.cardTable.SetCursor(0)
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m ListsModel) handleListPanelKeys(s, action string) (ListsModel, tea.Cmd) {
	if isBackKey(action, s) {
		m.shouldGoBack = true
		return m, nil
	}
	if action == "nav_up" || s == "up" || s == "k" {
		if m.listCursor > 0 {
			m.listCursor--
		}
		return m, nil
	}
	if action == "nav_down" || s == "down" || s == "j" {
		if m.listCursor < len(m.lists)-1 {
			m.listCursor++
		}
		return m, nil
	}
	if action == "nav_right" || s == "right" || s == "tab" {
		if m.selectedList != nil {
			m.focus = ListsFocusCardPanel
			m.searchInput.Focus()
			m.cardTable.Focus()
		}
		return m, nil
	}
	if isSelectKey(action, s) {
		return m.selectCurrentList()
	}
	if s == "n" {
		m.mode = ListsModeCreate
		m.nameInput.SetValue("")
		m.descInput.SetValue("")
		m.colorIndex = 0
		m.formFocus = 0
		m.nameInput.Focus()
		return m, nil
	}
	if s == "e" && len(m.lists) > 0 && m.listCursor < len(m.lists) {
		l := m.lists[m.listCursor]
		m.mode = ListsModeEdit
		m.editingListID = l.ID
		m.nameInput.SetValue(l.Name)
		m.descInput.SetValue(l.Description)
		m.colorIndex = m.findColorIndex(l.Color)
		m.formFocus = 0
		m.nameInput.Focus()
		return m, nil
	}
	if s == "d" && len(m.lists) > 0 && m.listCursor < len(m.lists) {
		l := m.lists[m.listCursor]
		m.modal = newModal(
			"Delete List",
			fmt.Sprintf("Delete list %q? This cannot be undone.", l.Name),
			func() tea.Cmd {
				return func() tea.Msg { return deleteListMsg{listID: l.ID} }
			},
			func() tea.Cmd { return nil },
			m.styleManager, m.width, m.height,
		)
		return m, nil
	}
	return m, nil
}

func (m ListsModel) handleCardPanelKeys(msg tea.KeyMsg, s, action string) (ListsModel, tea.Cmd) {
	if action == "nav_left" || s == "left" || s == "shift+tab" {
		m.focus = ListsFocusListPanel
		m.searchInput.Blur()
		m.cardTable.Blur()
		return m, nil
	}
	if isBackKey(action, s) {
		m.focus = ListsFocusListPanel
		m.searchInput.Blur()
		m.cardTable.Blur()
		return m, nil
	}
	if action == "nav_up" || s == "up" || s == "k" {
		m.cardTable, _ = m.cardTable.Update(msg)
		return m, nil
	}
	if action == "nav_down" || s == "down" || s == "j" {
		m.cardTable, _ = m.cardTable.Update(msg)
		return m, nil
	}
	if action == "increment_quantity" {
		return m.handleIncrementQuantity()
	}
	if action == "decrement_quantity" {
		return m.handleDecrementQuantity()
	}
	if action == "save" {
		return m.handleSaveListCards()
	}
	if s == "x" && m.selectedList != nil {
		m.exportState = NewExportState("list", m.selectedList.Name, false, "", m.buildListExportRows)
		return m, nil
	}
	if s == "i" && m.selectedList != nil {
		m.importState = NewImportState(m.cardService, m.styleManager)
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.filteredCards = m.filterListCards(m.searchInput.Value())
	m.updateListCardTable()
	m.cardTable.SetCursor(0)
	return m, cmd
}

func (m ListsModel) handleFormKeys(s, action string) (ListsModel, tea.Cmd) {
	if s == "esc" {
		m.mode = ListsModeNormal
		m.nameInput.Blur()
		m.descInput.Blur()
		return m, nil
	}
	if s == "tab" || action == "nav_down" || s == "down" {
		m.formFocus = (m.formFocus + 1) % 3
		return m.updateFormFocus(), nil
	}
	if s == "shift+tab" || action == "nav_up" || s == "up" {
		m.formFocus--
		if m.formFocus < 0 {
			m.formFocus = 2
		}
		return m.updateFormFocus(), nil
	}
	if m.formFocus == 2 {
		if s == "left" || action == "nav_left" {
			m.colorIndex--
			if m.colorIndex < 0 {
				m.colorIndex = len(listColorOptions) - 1
			}
			return m, nil
		}
		if s == "right" || action == "nav_right" {
			m.colorIndex = (m.colorIndex + 1) % len(listColorOptions)
			return m, nil
		}
	}
	if isSelectKey(action, s) && m.formFocus == 2 {
		if m.nameInput.Value() == "" {
			return m, nil
		}
		if m.mode == ListsModeCreate {
			m.modal = newModal(
				"Create List",
				fmt.Sprintf("Create list %q?", m.nameInput.Value()),
				func() tea.Cmd { return func() tea.Msg { return createListMsg{} } },
				func() tea.Cmd { return nil },
				m.styleManager, m.width, m.height,
			)
		} else {
			m.modal = newModal(
				"Update List",
				fmt.Sprintf("Update list %q?", m.nameInput.Value()),
				func() tea.Cmd { return func() tea.Msg { return updateListMsg{} } },
				func() tea.Cmd { return nil },
				m.styleManager, m.width, m.height,
			)
		}
		return m, nil
	}
	if m.formFocus == 0 {
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
		return m, cmd
	}
	if m.formFocus == 1 {
		var cmd tea.Cmd
		m.descInput, cmd = m.descInput.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
		return m, cmd
	}
	return m, nil
}

func (m ListsModel) updateFormFocus() ListsModel {
	m.nameInput.Blur()
	m.descInput.Blur()
	switch m.formFocus {
	case 0:
		m.nameInput.Focus()
	case 1:
		m.descInput.Focus()
	}
	return m
}

func (m ListsModel) selectCurrentList() (ListsModel, tea.Cmd) {
	if len(m.lists) == 0 || m.listCursor >= len(m.lists) {
		return m, nil
	}
	selected := &m.lists[m.listCursor]
	m.selectedList = selected
	quantities, err := m.listService.GetAllQuantitiesForList(selected.ID)
	if err != nil {
		m.dbQuantities = make(map[int64]int)
	} else {
		m.dbQuantities = quantities
	}
	m.tempQuantityChanges = make(map[int64]int)
	m.updateListCardTable()
	return m, nil
}

func (m ListsModel) View() string {
	header := m.renderHeader()
	footer := m.renderFooter()
	return RenderFramedWithModal(header, footer, m.renderBody, m.width, m.height, m.styleManager, &m.modal)
}

func (m ListsModel) renderHeader() string {
	var b strings.Builder
	if m.selectedGame != nil {
		b.WriteString(m.styleManager.GetTitleStyle().Render(m.selectedGame.Name + " - Lists"))
	}
	return b.String()
}

func (m ListsModel) renderBody(availableHeight int) string {
	contentWidth := m.width - frameBorderSize - framePaddingX*2
	if contentWidth < 20 {
		contentWidth = 20
	}
	leftWidth := contentWidth * 35 / 100
	rightWidth := contentWidth - leftWidth
	leftPanel := m.renderListPanel(leftWidth, availableHeight)
	rightPanel := m.renderCardPanel(rightWidth, availableHeight)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m ListsModel) renderListPanel(width, height int) string {
	var b strings.Builder
	if m.mode == ListsModeCreate || m.mode == ListsModeEdit {
		b.WriteString(m.renderForm())
	} else {
		b.WriteString(m.styleManager.GetTitleStyle().Render("Your Lists") + "\n")
		if len(m.lists) == 0 {
			b.WriteString(m.styleManager.GetBlurredStyle().Render("No lists yet. Press 'n' to create one.") + "\n")
		} else {
			for i, l := range m.lists {
				isSelected := m.listCursor == i && m.focus == ListsFocusListPanel
				isActive := m.selectedList != nil && m.selectedList.ID == l.ID
				colorDot := m.styleManager.applyBGFG(lipgloss.NewStyle().Foreground(lipgloss.Color(l.Color))).Render("●")
				label := colorDot + " " + l.Name
				if isActive {
					label += " *"
				}
				b.WriteString(RenderListItem(label, isSelected))
			}
		}
	}
	return RenderPanel(m.styleManager, b.String(), width, height, m.focus == ListsFocusListPanel, 1, 0)
}

func (m ListsModel) renderForm() string {
	var b strings.Builder
	title := "Create List"
	if m.mode == ListsModeEdit {
		title = "Edit List"
	}
	b.WriteString(m.styleManager.GetTitleStyle().Render(title) + "\n")
	nameLabel := "Name: "
	if m.formFocus == 0 {
		nameLabel = m.styleManager.GetFocusedStyle().Render(nameLabel)
	} else {
		nameLabel = m.styleManager.GetBlurredStyle().Render(nameLabel)
	}
	b.WriteString(nameLabel + m.nameInput.View() + "\n")
	descLabel := "Desc: "
	if m.formFocus == 1 {
		descLabel = m.styleManager.GetFocusedStyle().Render(descLabel)
	} else {
		descLabel = m.styleManager.GetBlurredStyle().Render(descLabel)
	}
	b.WriteString(descLabel + m.descInput.View() + "\n")
	colorLabel := "Color: "
	if m.formFocus == 2 {
		colorLabel = m.styleManager.GetFocusedStyle().Render(colorLabel)
	} else {
		colorLabel = m.styleManager.GetBlurredStyle().Render(colorLabel)
	}
	co := listColorOptions[m.colorIndex]
	colorPreview := m.styleManager.applyBGFG(lipgloss.NewStyle().Foreground(lipgloss.Color(co.Color))).Render("● " + co.Name)
	arrows := m.styleManager.GetBlurredStyle().Render("<") + " " + colorPreview + " " + m.styleManager.GetBlurredStyle().Render(">")
	if m.formFocus == 2 {
		arrows = m.styleManager.GetFocusedStyle().Render("<") + " " + colorPreview + " " + m.styleManager.GetFocusedStyle().Render(">")
	}
	b.WriteString(colorLabel + arrows + "\n")
	b.WriteString(m.styleManager.GetBlurredStyle().Render("Tab: next field • Enter: confirm • Esc: cancel"))
	return b.String()
}

func (m ListsModel) renderCardPanel(width, height int) string {
	var b strings.Builder
	if m.selectedList == nil {
		b.WriteString(m.styleManager.GetBlurredStyle().Render("Select a list to manage cards.") + "\n")
		return RenderPanel(m.styleManager, b.String(), width, height, m.focus == ListsFocusCardPanel, 1, 0)
	}
	colorDot := m.styleManager.applyBGFG(lipgloss.NewStyle().Foreground(lipgloss.Color(m.selectedList.Color))).Render("●")
	b.WriteString(colorDot + " " + m.styleManager.GetTitleStyle().Render(m.selectedList.Name) + "\n")
	b.WriteString(m.styleManager.GetBlurredStyle().Render("Search: ") + m.styleManager.GetNoStyle().Render(m.searchInput.View()) + "\n")
	showAll := m.searchInput.Value() == ""
	var rows []table.Row
	if showAll {
		for _, card := range m.cards {
			dbQty := m.dbQuantities[card.ID]
			tempDelta := m.tempQuantityChanges[card.ID]
			rows = append(rows, cardToRow(card, dbQty, tempDelta))
		}
	} else {
		for _, card := range m.filteredCards {
			dbQty := m.dbQuantities[card.ID]
			tempDelta := m.tempQuantityChanges[card.ID]
			rows = append(rows, cardToRow(card, dbQty, tempDelta))
		}
	}
	if len(rows) == 0 {
		b.WriteString(m.styleManager.GetBlurredStyle().Render("No cards match your search.") + "\n")
	} else {
		tableHeight := CalcTableHeight(height-2, 3, 3)
		// panelPadX=1, borderOverhead=2: inner table area = width - 4
		tableWidth := max(width-4, 20)
		m.cardTable.SetColumns(scaledCardSearchColumns(tableWidth))
		m.cardTable.SetRows(rows)
		m.cardTable.SetHeight(tableHeight)
		b.WriteString(m.styleManager.GetTableBaseStyle().Render(m.cardTable.View()))
	}
	return RenderPanel(m.styleManager, b.String(), width, height, m.focus == ListsFocusCardPanel, 1, 0)
}

func (m ListsModel) renderFooter() string {
	if m.exportState.active {
		return m.exportState.Render(m.styleManager)
	}
	if m.importState.active {
		return m.importState.Render(m.styleManager)
	}
	hb := NewHelpBuilder(m.configManager)
	var footer string
	if m.mode == ListsModeCreate || m.mode == ListsModeEdit {
		footer = m.styleManager.GetHelpStyle().Render("Tab: next field • Enter: confirm • Esc: cancel")
	} else if m.focus == ListsFocusCardPanel {
		footer = m.styleManager.GetHelpStyle().Render(
			hb.Build(
				KeyItem{"increment_quantity", "+", "Add"},
				KeyItem{"decrement_quantity", "Delete", "Remove"},
				KeyItem{"save", "Ctrl+S", "Save"},
			) + " • x: Export • i: Import • " + hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate") + " • Left/Shift+Tab: Lists panel",
		)
	} else {
		footer = m.styleManager.GetHelpStyle().Render(
			hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate") + " • Enter: Select list • n: New • e: Edit • d: Delete • Right/Tab: Cards panel • " + hb.Build(KeyItem{"back", "Q", "Back"}),
		)
	}
	if m.exportState.statusMsg != "" {
		footer = m.styleManager.GetHelpStyle().Render(m.exportState.statusMsg) + "  " + footer
	}
	return footer
}

func (m ListsModel) getSelectedCard() (model.Card, bool) {
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

func (m ListsModel) handleIncrementQuantity() (ListsModel, tea.Cmd) {
	card, ok := m.getSelectedCard()
	if !ok || m.selectedList == nil {
		return m, nil
	}
	if m.tempQuantityChanges == nil {
		m.tempQuantityChanges = make(map[int64]int)
	}
	m.tempQuantityChanges[card.ID]++
	m.updateListCardTable()
	return m, nil
}

func (m ListsModel) handleDecrementQuantity() (ListsModel, tea.Cmd) {
	card, ok := m.getSelectedCard()
	if !ok || m.selectedList == nil {
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
	m.updateListCardTable()
	return m, nil
}

func (m ListsModel) handleSaveListCards() (ListsModel, tea.Cmd) {
	if len(m.tempQuantityChanges) == 0 || m.selectedList == nil {
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
	message := fmt.Sprintf("Save %d card changes to list %q?", changeCount, m.selectedList.Name)
	m.modal = newModal(
		"Confirm Save",
		message,
		func() tea.Cmd { return func() tea.Msg { return saveListCardsMsg{} } },
		func() tea.Cmd { return nil },
		m.styleManager, m.width, m.height,
	)
	return m, nil
}

func (m ListsModel) performSaveListCards() (ListsModel, tea.Cmd) {
	if m.selectedList == nil {
		return m, nil
	}
	updates := make(map[int64]int)
	for cardID, tempDelta := range m.tempQuantityChanges {
		dbQty := m.dbQuantities[cardID]
		newQty := dbQty + tempDelta
		updates[cardID] = newQty
	}
	ctx := context.Background()
	err := m.listService.UpsertListCardBatch(ctx, m.selectedList.ID, updates)
	if err != nil {
		return m, nil
	}
	maps.Copy(m.dbQuantities, updates)
	m.tempQuantityChanges = make(map[int64]int)
	m.updateListCardTable()
	return m, nil
}

func (m ListsModel) performDeleteList(listID int64) (ListsModel, tea.Cmd) {
	ctx := context.Background()
	err := m.listService.DeleteList(ctx, listID)
	if err != nil {
		return m, nil
	}
	if m.selectedList != nil && m.selectedList.ID == listID {
		m.selectedList = nil
		m.dbQuantities = make(map[int64]int)
		m.tempQuantityChanges = make(map[int64]int)
	}
	return m.refreshLists()
}

func (m ListsModel) performCreateList() (ListsModel, tea.Cmd) {
	ctx := context.Background()
	color := listColorOptions[m.colorIndex].Color
	_, err := m.listService.CreateList(ctx, m.user.ID, m.selectedGame.ID, m.nameInput.Value(), m.descInput.Value(), color)
	if err != nil {
		return m, nil
	}
	m.mode = ListsModeNormal
	m.nameInput.Blur()
	m.descInput.Blur()
	return m.refreshLists()
}

func (m ListsModel) performUpdateList() (ListsModel, tea.Cmd) {
	ctx := context.Background()
	color := listColorOptions[m.colorIndex].Color
	err := m.listService.UpdateList(ctx, m.editingListID, m.nameInput.Value(), m.descInput.Value(), color)
	if err != nil {
		return m, nil
	}
	m.mode = ListsModeNormal
	m.nameInput.Blur()
	m.descInput.Blur()
	if m.selectedList != nil && m.selectedList.ID == m.editingListID {
		m.selectedList.Name = m.nameInput.Value()
		m.selectedList.Description = m.descInput.Value()
		m.selectedList.Color = color
	}
	return m.refreshLists()
}

func (m ListsModel) refreshLists() (ListsModel, tea.Cmd) {
	if m.user == nil || m.selectedGame == nil {
		return m, nil
	}
	lists, err := m.listService.GetListsByUserAndGame(m.user.ID, m.selectedGame.ID)
	if err != nil {
		return m, nil
	}
	m.lists = lists
	if m.listCursor >= len(m.lists) {
		m.listCursor = max(len(m.lists)-1, 0)
	}
	return m, nil
}

func (m *ListsModel) updateListCardTable() {
	var rows []table.Row
	source := m.filteredCards
	if m.searchInput.Value() == "" {
		source = m.cards
	}
	for _, card := range source {
		dbQty := m.dbQuantities[card.ID]
		tempDelta := m.tempQuantityChanges[card.ID]
		rows = append(rows, cardToRow(card, dbQty, tempDelta))
	}
	m.cardTable.SetRows(rows)
}

func (m ListsModel) filterListCards(query string) []model.Card {
	return filterCardsByQuery(m.cards, query)
}

func (m ListsModel) findColorIndex(color string) int {
	for i, co := range listColorOptions {
		if co.Color == color {
			return i
		}
	}
	return 0
}

func (m *ListsModel) buildListExportRows() []export.CardRow {
	return buildCardExportRows(m.cards, m.dbQuantities, nil)
}

// applyImport merges the imported card quantities into tempQuantityChanges.
// Quantities are staged so the user can review before saving with Ctrl+S.
func (m ListsModel) applyImport(msg ImportApplyMsg) (ListsModel, tea.Cmd) {
	if m.selectedList == nil {
		return m, nil
	}
	for cardID, qty := range msg.Quantities {
		m.tempQuantityChanges[cardID] += qty
	}
	m.updateListCardTable()
	return m, nil
}
