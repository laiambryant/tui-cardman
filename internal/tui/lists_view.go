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

type listCardPanelSubFocus int

const (
	listCardSubFocusSearch listCardPanelSubFocus = iota
	listCardSubFocusContents
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
	selectedGame      *model.CardGame
	user              *auth.User
	listService       listservice.ListService
	cardService       cardservice.CardService
	styleManager      *StyleManager
	configManager     *runtimecfg.Manager
	width             int
	height            int
	lists             []model.UserList
	listCursor        int
	selectedList      *model.UserList
	mode              ListsMode
	nameInput         textinput.Model
	descInput         textinput.Model
	colorIndex        int
	formFocus         int
	cards             []model.Card
	filteredCards     []model.Card
	searchInput       textinput.Model
	cardTable         table.Model
	listContentsTable table.Model
	cardSubFocus      listCardPanelSubFocus
	focus             ListsFocus
	quantities        QuantityTracker
	modal             ModalModel
	shouldGoBack      bool
	editingListID     int64
	exportState       ExportState
	importState       ImportState
	searchCache       *SearchCache
	cardPagination    Pagination
}

func newFormInput(sm *StyleManager, placeholder string, width, charLimit int) textinput.Model {
	input := textinput.New()
	input.Placeholder = placeholder
	input.Width = width
	input.CharLimit = charLimit
	sm.ApplyTextInputStyles(&input)
	return input
}

func NewListsModel(game *model.CardGame, user *auth.User, listSvc listservice.ListService, cardSvc cardservice.CardService, cards []model.Card, cfg *runtimecfg.Manager, sm *StyleManager) ListsModel {
	searchInput := newSearchInput(sm, "Search cards...")
	nameInput := newFormInput(sm, "List name...", 25, 50)
	descInput := newFormInput(sm, "Description...", 25, 100)
	cardTable := NewStyledTable(defaultSearchTableColumns(), 10, true, sm)
	listContentsTable := NewStyledTable(defaultSearchTableColumns(), 5, false, sm)
	return ListsModel{
		selectedGame:      game,
		user:              user,
		listService:       listSvc,
		cardService:       cardSvc,
		styleManager:      sm,
		configManager:     cfg,
		cards:             cards,
		filteredCards:     cards,
		searchInput:       searchInput,
		nameInput:         nameInput,
		descInput:         descInput,
		cardTable:         cardTable,
		listContentsTable: listContentsTable,
		quantities:        newQuantityTracker(),
		searchCache:       NewSearchCache(),
		cardPagination:    NewPagination(50),
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
			m.cardSubFocus = listCardSubFocusSearch
			m.searchInput.Focus()
			m.cardTable.Focus()
			m.listContentsTable.Blur()
		}
		return m, nil
	}
	if isSelectKey(action, s) {
		return m.selectCurrentList()
	}
	if action == "create_new" {
		m.mode = ListsModeCreate
		m.nameInput.SetValue("")
		m.descInput.SetValue("")
		m.colorIndex = 0
		m.formFocus = 0
		m.nameInput.Focus()
		return m, nil
	}
	if action == "edit" && len(m.lists) > 0 && m.listCursor < len(m.lists) {
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
	if action == "delete" && len(m.lists) > 0 && m.listCursor < len(m.lists) {
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

func (m ListsModel) returnToListPanel() ListsModel {
	m.focus = ListsFocusListPanel
	m.cardSubFocus = listCardSubFocusSearch
	m.searchInput.Blur()
	m.cardTable.Blur()
	m.listContentsTable.Blur()
	return m
}

func (m ListsModel) focusListContents() ListsModel {
	m.cardSubFocus = listCardSubFocusContents
	m.cardTable.Blur()
	m.searchInput.Blur()
	m.listContentsTable.Focus()
	return m
}

func (m ListsModel) focusListSearch() ListsModel {
	m.cardSubFocus = listCardSubFocusSearch
	m.listContentsTable.Blur()
	m.cardTable.Focus()
	m.searchInput.Focus()
	return m
}

func (m ListsModel) cycleCardPanelFocus() ListsModel {
	if m.cardSubFocus == listCardSubFocusSearch {
		return m.focusListContents()
	}
	return m.returnToListPanel()
}

// backKeyClosesCardPanel is true only when search is not focused, so Esc closes instead of clearing the search.
func (m ListsModel) backKeyClosesCardPanel(action, s string) bool {
	return isBackKey(action, s) && m.cardSubFocus != listCardSubFocusSearch
}

func (m ListsModel) handleCardPanelBack() ListsModel {
	if m.cardSubFocus == listCardSubFocusContents {
		return m.focusListSearch()
	}
	return m.returnToListPanel()
}

func (m ListsModel) moveCardPanelCursor(msg tea.KeyMsg, down bool) ListsModel {
	if m.cardSubFocus == listCardSubFocusContents {
		if !down || m.listContentsTable.Cursor() < len(m.listContentsTable.Rows())-1 {
			m.listContentsTable, _ = m.listContentsTable.Update(msg)
		}
		return m
	}
	if !down || m.cardTable.Cursor() < len(m.cardTable.Rows())-1 {
		m.cardTable, _ = m.cardTable.Update(msg)
	}
	return m
}

func (m ListsModel) changeCardPage(next bool) ListsModel {
	if next {
		m.cardPagination.NextPage()
	} else {
		m.cardPagination.PrevPage()
	}
	m.updateListCardTable()
	m.cardTable.SetCursor(0)
	return m
}

func (m ListsModel) forwardKeyToListSearch(msg tea.KeyMsg) (ListsModel, tea.Cmd) {
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.filteredCards = m.filterListCards(m.searchInput.Value())
	m.cardPagination.Reset()
	m.updateListCardTable()
	m.cardTable.SetCursor(0)
	return m, cmd
}

func (m ListsModel) handleCardPanelKeys(msg tea.KeyMsg, s, action string) (ListsModel, tea.Cmd) {
	if s == "tab" {
		return m.cycleCardPanelFocus(), nil
	}
	if action == "nav_left" || s == "left" || s == "shift+tab" {
		return m.returnToListPanel(), nil
	}
	if m.backKeyClosesCardPanel(action, s) {
		return m.handleCardPanelBack(), nil
	}
	if action == "nav_up" || s == "up" {
		return m.moveCardPanelCursor(msg, false), nil
	}
	if action == "nav_down" || s == "down" {
		return m.moveCardPanelCursor(msg, true), nil
	}
	if action == "page_next" {
		return m.changeCardPage(true), nil
	}
	if action == "page_prev" {
		return m.changeCardPage(false), nil
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
	if action == "export" && m.selectedList != nil {
		m.exportState = NewExportState("list", m.selectedList.Name, false, "", m.buildListExportRows)
		return m, nil
	}
	if action == "import" && m.selectedList != nil {
		m.importState = NewImportState(m.cardService, m.styleManager)
		return m, nil
	}
	if m.cardSubFocus == listCardSubFocusSearch && !isModifierKey(s) {
		return m.forwardKeyToListSearch(msg)
	}
	return m, nil
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
		m.quantities.reset()
	} else {
		m.quantities.load(quantities)
	}
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
	leftContent := m.renderListPanel(leftWidth, availableHeight)
	rightPanel := m.renderCardPanel(rightWidth, availableHeight)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftContent, rightPanel)
}

func colorDotFor(sm *StyleManager, color string) string {
	return sm.applyBGFG(lipgloss.NewStyle().Foreground(lipgloss.Color(color))).Render("●")
}

func (m ListsModel) renderListPanel(width, height int) string {
	var b strings.Builder
	if m.mode == ListsModeCreate || m.mode == ListsModeEdit {
		b.WriteString(m.renderForm())
	} else {
		b.WriteString(renderPanelHeading(m.styleManager, "Your Lists", m.focus == ListsFocusListPanel))
		b.WriteString(m.renderListEntries())
	}
	return RenderPanel(m.styleManager, b.String(), width, height, m.focus == ListsFocusListPanel, 1, 0)
}

func (m ListsModel) renderListEntries() string {
	if len(m.lists) == 0 {
		return m.styleManager.GetBlurredStyle().Render("No lists yet. Press 'n' to create one.") + "\n"
	}
	var b strings.Builder
	for i, l := range m.lists {
		isSelected := m.listCursor == i && m.focus == ListsFocusListPanel
		label := colorDotFor(m.styleManager, l.Color) + " " + l.Name
		if m.selectedList != nil && m.selectedList.ID == l.ID {
			label += " *"
		}
		b.WriteString(RenderListItem(label, isSelected))
	}
	return b.String()
}

func (m ListsModel) renderColorPicker() string {
	co := listColorOptions[m.colorIndex]
	colorPreview := colorDotFor(m.styleManager, co.Color) + m.styleManager.GetNoStyle().Render(" "+co.Name)
	arrowStyle := m.styleManager.GetBlurredStyle()
	if m.formFocus == 2 {
		arrowStyle = m.styleManager.GetFocusedStyle()
	}
	return arrowStyle.Render("<") + " " + colorPreview + " " + arrowStyle.Render(">")
}

func (m ListsModel) renderForm() string {
	title := "Create List"
	if m.mode == ListsModeEdit {
		title = "Edit List"
	}
	var b strings.Builder
	b.WriteString(m.styleManager.GetTitleStyle().Render(title) + "\n")
	b.WriteString(RenderConditionalLabel(m.formFocus == 0, "Name: ") + m.nameInput.View() + "\n")
	b.WriteString(RenderConditionalLabel(m.formFocus == 1, "Desc: ") + m.descInput.View() + "\n")
	b.WriteString(RenderConditionalLabel(m.formFocus == 2, "Color: ") + m.renderColorPicker() + "\n")
	b.WriteString(m.styleManager.GetBlurredStyle().Render(joinHelp("Tab: next field", "Enter: confirm", "Esc: cancel")))
	return b.String()
}

func (m ListsModel) renderListSearchSection(vcs VisibleColumnSet, topHeight int) string {
	var top strings.Builder
	colorDot := colorDotFor(m.styleManager, m.selectedList.Color)
	top.WriteString(colorDot + " " + RenderConditionalLabel(m.cardSubFocus == listCardSubFocusSearch, m.selectedList.Name) + "\n")
	top.WriteString(renderSearchLine(m.styleManager, m.searchInput.View(), m.cardPagination.StatusText()))
	source := searchSource(m.searchInput.Value(), m.filteredCards, m.cards)
	m.cardPagination.TotalItems = len(source)
	start, end := m.cardPagination.Slice()
	rows := buildCardRows(source[start:end], m.quantities.db, m.quantities.temp, vcs)
	if len(rows) == 0 {
		top.WriteString(m.styleManager.GetBlurredStyle().Render("No cards match your search.") + "\n")
		return top.String()
	}
	m.cardTable.SetColumns(vcs.Columns)
	m.cardTable.SetRows(rows)
	m.cardTable.SetHeight(CalcTableHeight(topHeight, 2, 3))
	top.WriteString(m.cardTable.View())
	return top.String()
}

func (m ListsModel) renderListContentsSection(vcs VisibleColumnSet, bottomHeight int) string {
	var bottom strings.Builder
	bottom.WriteString(renderPanelHeading(m.styleManager, "List Contents", m.cardSubFocus == listCardSubFocusContents))
	if len(m.listContentsTable.Rows()) == 0 {
		bottom.WriteString(m.styleManager.GetBlurredStyle().Render("No cards in list yet.") + "\n")
		return bottom.String()
	}
	m.listContentsTable.SetColumns(vcs.Columns)
	m.listContentsTable.SetHeight(CalcTableHeight(bottomHeight, 1, 3))
	bottom.WriteString(m.listContentsTable.View())
	return bottom.String()
}

func (m ListsModel) renderCardPanel(width, height int) string {
	if m.selectedList == nil {
		placeholder := m.styleManager.GetBlurredStyle().Render("Select a list to manage cards.") + "\n"
		return RenderPanel(m.styleManager, placeholder, width, height, m.focus == ListsFocusCardPanel, 1, 0)
	}
	tableWidth := max(width-4, 20)
	topHeight := max(height*6/10, 5)
	bottomHeight := max(height-topHeight, 5)
	vcs := BuildVisibleColumnSet(CardSearchColumns, GetVisibleColumns(m.configManager), GetColumnOrder(m.configManager), tableWidth)
	top := m.renderListSearchSection(vcs, topHeight)
	bottom := m.renderListContentsSection(vcs, bottomHeight)
	searchFocused := m.cardSubFocus == listCardSubFocusSearch
	contentsFocused := m.cardSubFocus == listCardSubFocusContents
	return renderSplitCardPanel(m.styleManager, top, searchFocused, bottom, contentsFocused, width, topHeight, bottomHeight)
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
		footer = m.styleManager.GetHelpStyle().Render(joinHelp("Tab: next field", "Enter: confirm", "Esc: cancel"))
	} else if m.focus == ListsFocusCardPanel {
		footer = m.styleManager.GetHelpStyle().Render(joinHelp(
			"Tab: Switch panel",
			hb.Build(
				KeyItem{"increment_quantity", "+", "Add"},
				KeyItem{"decrement_quantity", "Delete", "Remove"},
				KeyItem{"save", "Ctrl+S", "Save"},
			),
			hb.Build(KeyItem{"export", "x", "Export"}, KeyItem{"import", "i", "Import"}),
			hb.Pair("page_next", "Ctrl+N", "page_prev", "Ctrl+P", "Page"),
			hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate"),
			"←/Shift+Tab: Lists panel",
		))
	} else {
		footer = m.styleManager.GetHelpStyle().Render(joinHelp(
			hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate"),
			hb.Build(KeyItem{"select", "Enter", "Select list"}),
			hb.Build(
				KeyItem{"create_new", "n", "New"},
				KeyItem{"edit", "e", "Edit"},
				KeyItem{"delete", "d", "Delete"},
			),
			"→/Tab: Cards panel",
			hb.Build(KeyItem{"back", "Q", "Back"}),
		))
	}
	if m.exportState.statusMsg != "" {
		footer = m.styleManager.GetHelpStyle().Render(m.exportState.statusMsg) + "  " + footer
	}
	return footer
}

func (m ListsModel) getSelectedCard() (model.Card, bool) {
	source := searchSource(m.searchInput.Value(), m.filteredCards, m.cards)
	return cardAtTableCursor(m.cardTable.Cursor(), m.cardPagination, source)
}

func (m ListsModel) handleIncrementQuantity() (ListsModel, tea.Cmd) {
	card, ok := m.getSelectedCard()
	if !ok || m.selectedList == nil {
		return m, nil
	}
	m.quantities.increment(card.ID)
	m.updateListCardTable()
	return m, nil
}

func (m ListsModel) handleDecrementQuantity() (ListsModel, tea.Cmd) {
	card, ok := m.getSelectedCard()
	if !ok || m.selectedList == nil {
		return m, nil
	}
	if !m.quantities.decrement(card.ID) {
		return m, nil
	}
	m.updateListCardTable()
	return m, nil
}

func (m ListsModel) handleSaveListCards() (ListsModel, tea.Cmd) {
	if m.quantities.pendingCount() == 0 || m.selectedList == nil {
		return m, nil
	}
	message := fmt.Sprintf("Save %d card changes to list %q?", m.quantities.pendingCount(), m.selectedList.Name)
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
	updates := m.quantities.buildUpdates()
	ctx := context.Background()
	err := m.listService.UpsertListCardBatch(ctx, m.selectedList.ID, updates)
	if err != nil {
		return m, nil
	}
	m.quantities.commit(updates)
	m.searchCache.Invalidate()
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
		m.quantities.reset()
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
	source := searchSource(m.searchInput.Value(), m.filteredCards, m.cards)
	m.cardPagination.TotalItems = len(source)
	start, end := m.cardPagination.Slice()
	vcs := BuildVisibleColumnSet(CardSearchColumns, GetVisibleColumns(m.configManager), GetColumnOrder(m.configManager), 80)
	m.cardTable.SetColumns(vcs.Columns)
	m.cardTable.SetRows(buildCardRows(source[start:end], m.quantities.db, m.quantities.temp, vcs))
	m.listContentsTable.SetColumns(vcs.Columns)
	m.listContentsTable.SetRows(buildPendingQuantityRows(m.cards, &m.quantities, vcs))
}

func (m ListsModel) filterListCards(query string) []model.Card {
	return filterCardsByQueryCached(m.cards, query, m.searchCache)
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
	return buildCardExportRows(m.cards, m.quantities.db, nil)
}

func (m ListsModel) applyImport(msg ImportApplyMsg) (ListsModel, tea.Cmd) {
	if m.selectedList == nil {
		return m, nil
	}
	m.quantities.applyImport(msg.Quantities)
	m.updateListCardTable()
	return m, nil
}
