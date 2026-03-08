package tui

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/laiambryant/tui-cardman/internal/config"
	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/pokemontcg"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/importruns"
	"github.com/laiambryant/tui-cardman/internal/services/prices"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

type ActionType int

const (
	ActionImport ActionType = iota
	ActionDelete
	ActionReimport
	ActionImportAll
	ActionImportUpdates
)

type importFocus int

const (
	importFocusSets    importFocus = iota
	importFocusActions importFocus = iota
	importFocusQueue   importFocus = iota
)

type ActionItem struct {
	label       string
	description string
	actionType  ActionType
	enabled     bool
}

type ImportModel struct {
	selectedCardGame  *model.CardGame
	availableSets     []pokemontcg.Set
	databaseSetIDs    map[string]bool
	filteredSets      []pokemontcg.Set
	searchInput       textinput.Model
	cursor            int
	actionCursor      int
	queueCursor       int
	focus             importFocus
	configManager     *runtimecfg.Manager
	styleManager      *StyleManager
	db                *sql.DB
	pokemonClient     *pokemontcg.Client
	importService     *pokemontcg.ImportService
	setService        sets.SetService
	isLoading         bool
	loadingMsg        string
	errorMsg          string
	statusMsg         string
	isImporting       bool
	importProgress    importProgressMsg
	selectedSetInDB   bool
	selectedSetHasCol bool
	cardGameCursor    int
	cardGames         []model.CardGame
	width             int
	height            int
	modal             ModalModel
	pendingAction     *ActionItem
	spinner           spinner.Model
	importQueue       []importQueueItem
	queueProcessing   bool
	queueCurrentIndex int
}

func NewImportModel(db *sql.DB, cfg *runtimecfg.Manager, styleManager *StyleManager, cardGames []model.CardGame) (ImportModel, error) {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search sets..."
	searchInput.Width = 30
	apiKey := config.GetAPIKey()
	client := pokemontcg.NewClient(apiKey)
	importRunService := importruns.NewImportRunService(db)
	setService := sets.NewSetService(db)
	cardService := card.NewCardService(db)
	tcgPlayerPriceService := prices.NewTCGPlayerPriceService(db)
	cardMarketPriceService := prices.NewCardMarketPriceService(db)
	importService := pokemontcg.NewImportService(
		db, client, slog.Default(),
		importRunService, setService, cardService,
		tcgPlayerPriceService, cardMarketPriceService,
	)
	s := spinner.New()
	s.Spinner = ImportSpinner
	s.Style = focusedStyle
	selectedGame := &cardGames[0]
	return ImportModel{
		selectedCardGame: selectedGame,
		searchInput:      searchInput,
		cursor:           0,
		actionCursor:     0,
		queueCursor:      0,
		focus:            importFocusSets,
		configManager:    cfg,
		styleManager:     styleManager,
		db:               db,
		pokemonClient:    client,
		importService:    importService,
		setService:       setService,
		databaseSetIDs:   make(map[string]bool),
		cardGames:        cardGames,
		cardGameCursor:   0,
		spinner:          s,
	}, nil
}

func (m ImportModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.fetchSetsCmd(),
		m.fetchDatabaseSetsCmd(),
	)
}

func (m ImportModel) handleImportSetResult(success bool, setID string, err error) (ImportModel, tea.Cmd) {
	if m.queueProcessing {
		return m.handleQueueItemResult(success, setID, err)
	}
	if success {
		m.isImporting = false
		m.statusMsg = fmt.Sprintf("Successfully imported set: %s", setID)
		return m, tea.Batch(m.fetchDatabaseSetsCmd(), m.checkSelectedSetInDB())
	}
	m.isImporting = false
	m.errorMsg = fmt.Sprintf("Failed to import set %s: %v", setID, err)
	return m, nil
}

func (m ImportModel) handleQueueItemResult(success bool, setID string, err error) (ImportModel, tea.Cmd) {
	if m.queueCurrentIndex < len(m.importQueue) {
		if success {
			m.importQueue[m.queueCurrentIndex].status = queueStatusCompleted
		} else {
			m.importQueue[m.queueCurrentIndex].status = queueStatusError
			m.importQueue[m.queueCurrentIndex].err = err
		}
	}
	m.queueCurrentIndex++
	if m.queueCurrentIndex < len(m.importQueue) {
		next := m.importQueue[m.queueCurrentIndex]
		m.importQueue[m.queueCurrentIndex].status = queueStatusImporting
		m.importProgress = importProgressMsg{
			setID:         next.setID,
			setsCompleted: m.queueCurrentIndex,
			totalSets:     len(m.importQueue),
		}
		return m, m.importSetCmd(next.setID)
	}
	m.queueProcessing = false
	m.isImporting = false
	completed := 0
	errored := 0
	for _, item := range m.importQueue {
		if item.status == queueStatusCompleted {
			completed++
		}
		if item.status == queueStatusError {
			errored++
		}
	}
	m.statusMsg = fmt.Sprintf("Queue complete: %d imported, %d errors", completed, errored)
	return m, tea.Batch(m.fetchDatabaseSetsCmd(), m.checkSelectedSetInDB())
}

func (m ImportModel) handleImportAllResult(success bool, operation string, err error) (ImportModel, tea.Cmd) {
	if success {
		m.isImporting = false
		m.statusMsg = fmt.Sprintf("Successfully imported %s", operation)
		return m, m.fetchDatabaseSetsCmd()
	}
	m.isImporting = false
	m.errorMsg = fmt.Sprintf("Failed to import %s: %v", operation, err)
	return m, nil
}

func (m ImportModel) Update(msg tea.Msg) (ImportModel, tea.Cmd) {
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
	if _, ok := msg.(importConfirmedMsg); ok {
		return m.executeConfirmedAction()
	}
	if m.isImporting {
		return m.handleImportingState(msg)
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case fetchSetsSuccessMsg:
		m.availableSets = msg.sets
		m.filteredSets = msg.sets
		m.isLoading = false
		m.statusMsg = fmt.Sprintf("Loaded %d sets from API", len(msg.sets))
		return m, nil
	case fetchSetsErrorMsg:
		m.isLoading = false
		m.errorMsg = fmt.Sprintf("Failed to fetch sets: %v", msg.err)
		return m, nil
	case fetchDatabaseSetsSuccessMsg:
		m.databaseSetIDs = make(map[string]bool)
		for _, id := range msg.apiIDs {
			m.databaseSetIDs[id] = true
		}
		return m, nil
	case fetchDatabaseSetsErrorMsg:
		m.errorMsg = fmt.Sprintf("Failed to fetch database sets: %v", msg.err)
		return m, nil
	case deleteSetSuccessMsg:
		m.statusMsg = fmt.Sprintf("Successfully deleted set: %s", msg.setID)
		return m, tea.Batch(m.fetchDatabaseSetsCmd(), m.checkSelectedSetInDB())
	case deleteSetErrorMsg:
		m.errorMsg = fmt.Sprintf("Failed to delete set %s: %v", msg.setID, msg.err)
		return m, nil
	case checkSetInDBMsg:
		m.selectedSetInDB = true
		m.selectedSetHasCol = msg.hasCollections
		return m, nil
	case checkSetNotInDBMsg:
		m.selectedSetInDB = false
		m.selectedSetHasCol = false
		return m, nil
	case checkSetInCollectionErrorMsg:
		m.errorMsg = fmt.Sprintf("Failed to check set collections: %v", msg.err)
		return m, nil
	}
	return m, nil
}

func (m ImportModel) handleImportingState(msg tea.Msg) (ImportModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case importProgressMsg:
		m.importProgress = msg
		return m, nil
	case importSetSuccessMsg:
		return m.handleImportSetResult(true, msg.setID, nil)
	case importSetErrorMsg:
		return m.handleImportSetResult(false, msg.setID, msg.err)
	case importAllSetsSuccessMsg:
		return m.handleImportAllResult(true, "all sets", nil)
	case importAllSetsErrorMsg:
		return m.handleImportAllResult(false, "all sets", msg.err)
	case importNewSetsSuccessMsg:
		return m.handleImportAllResult(true, "new sets", nil)
	case importNewSetsErrorMsg:
		return m.handleImportAllResult(false, "new sets", msg.err)
	}
	return m, nil
}

func (m ImportModel) handleKeyMsg(msg tea.KeyMsg) (ImportModel, tea.Cmd) {
	s := msg.String()
	action := GetAction(m.configManager, s)
	if action == "quit" || s == "ctrl+c" {
		return m, tea.Quit
	}
	if s == "tab" {
		switch m.focus {
		case importFocusSets:
			m.focus = importFocusActions
			m.actionCursor = 0
		case importFocusActions:
			if len(m.importQueue) > 0 {
				m.focus = importFocusQueue
				m.queueCursor = 0
			} else {
				m.focus = importFocusSets
			}
		case importFocusQueue:
			m.focus = importFocusSets
		}
		return m, nil
	}
	switch m.focus {
	case importFocusActions:
		return m.handleActionNavigation(msg)
	case importFocusQueue:
		return m.handleQueueNavigation(msg)
	}
	return m.handleSetListNavigation(msg)
}

func (m ImportModel) handleSetListNavigation(msg tea.KeyMsg) (ImportModel, tea.Cmd) {
	s := msg.String()
	action := GetAction(m.configManager, s)
	if action == "nav_up" || s == "up" || s == "k" {
		if m.cursor > 0 {
			m.cursor--
			return m, m.checkSelectedSetInDB()
		}
	}
	if action == "nav_down" || s == "down" || s == "j" {
		if m.cursor < len(m.filteredSets)-1 {
			m.cursor++
			return m, m.checkSelectedSetInDB()
		}
	}
	if s == "a" && len(m.filteredSets) > 0 && m.cursor < len(m.filteredSets) {
		set := m.filteredSets[m.cursor]
		if !m.databaseSetIDs[set.ID] {
			m.addToQueue(set.ID, set.Name)
			m.statusMsg = fmt.Sprintf("Added %s to queue (%d pending)", set.Name, m.queuePendingCount())
		}
		return m, nil
	}
	if s == "r" && len(m.filteredSets) > 0 && m.cursor < len(m.filteredSets) {
		set := m.filteredSets[m.cursor]
		m.removeFromQueue(set.ID)
		m.statusMsg = fmt.Sprintf("Removed %s from queue (%d pending)", set.Name, m.queuePendingCount())
		return m, nil
	}
	if s == "s" && len(m.importQueue) > 0 && !m.queueProcessing {
		return m.startQueueProcessing()
	}
	if s == "c" && !m.queueProcessing {
		m.clearCompletedFromQueue()
		m.statusMsg = "Cleared completed items from queue"
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	if m.searchInput.Value() != "" {
		m = m.filterSets()
	} else {
		m.filteredSets = m.availableSets
	}
	return m, cmd
}

func (m ImportModel) startQueueProcessing() (ImportModel, tea.Cmd) {
	m.queueCurrentIndex = 0
	for m.queueCurrentIndex < len(m.importQueue) {
		if m.importQueue[m.queueCurrentIndex].status == queueStatusPending {
			break
		}
		m.queueCurrentIndex++
	}
	if m.queueCurrentIndex >= len(m.importQueue) {
		m.statusMsg = "No pending items in queue"
		return m, nil
	}
	m.queueProcessing = true
	m.isImporting = true
	m.importQueue[m.queueCurrentIndex].status = queueStatusImporting
	first := m.importQueue[m.queueCurrentIndex]
	m.importProgress = importProgressMsg{
		setID:         first.setID,
		setsCompleted: 0,
		totalSets:     len(m.importQueue),
	}
	return m, tea.Batch(m.spinner.Tick, m.importSetCmd(first.setID))
}

func (m ImportModel) handleActionNavigation(msg tea.KeyMsg) (ImportModel, tea.Cmd) {
	s := msg.String()
	action := GetAction(m.configManager, s)
	actions := m.getAvailableActions()
	if action == "nav_up" || s == "up" || s == "k" {
		if m.actionCursor > 0 {
			m.actionCursor--
		}
	}
	if action == "nav_down" || s == "down" || s == "j" {
		if m.actionCursor < len(actions)-1 {
			m.actionCursor++
		}
	}
	if action == "select" || s == "enter" || s == "\r" || s == "\n" {
		if m.actionCursor < len(actions) {
			return m.executeAction(actions[m.actionCursor])
		}
	}
	return m, nil
}

func (m ImportModel) handleQueueNavigation(msg tea.KeyMsg) (ImportModel, tea.Cmd) {
	s := msg.String()
	action := GetAction(m.configManager, s)
	if action == "nav_up" || s == "up" || s == "k" {
		if m.queueCursor > 0 {
			m.queueCursor--
		}
		return m, nil
	}
	if action == "nav_down" || s == "down" || s == "j" {
		if m.queueCursor < len(m.importQueue)-1 {
			m.queueCursor++
		}
		return m, nil
	}
	if s == "r" {
		if m.queueCursor < len(m.importQueue) {
			item := m.importQueue[m.queueCursor]
			m.removeFromQueue(item.setID)
			if m.queueCursor >= len(m.importQueue) && m.queueCursor > 0 {
				m.queueCursor--
			}
			m.statusMsg = fmt.Sprintf("Removed %s from queue (%d pending)", item.setName, m.queuePendingCount())
		}
		return m, nil
	}
	if s == "s" && !m.queueProcessing {
		return m.startQueueProcessing()
	}
	if s == "c" && !m.queueProcessing {
		m.clearCompletedFromQueue()
		m.statusMsg = "Cleared completed items from queue"
		return m, nil
	}
	return m, nil
}

func (m ImportModel) filterSets() ImportModel {
	query := strings.ToLower(m.searchInput.Value())
	var filtered []pokemontcg.Set
	for _, set := range m.availableSets {
		if strings.Contains(strings.ToLower(set.Name), query) ||
			strings.Contains(strings.ToLower(set.ID), query) {
			filtered = append(filtered, set)
		}
	}
	m.filteredSets = filtered
	if m.cursor >= len(m.filteredSets) {
		m.cursor = len(m.filteredSets) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return m
}

func (m ImportModel) getAvailableActions() []ActionItem {
	var actions []ActionItem
	if len(m.filteredSets) > 0 && m.cursor < len(m.filteredSets) {
		selectedSet := m.filteredSets[m.cursor]
		if m.databaseSetIDs[selectedSet.ID] {
			actions = append(actions, ActionItem{
				label:       "Reimport Set",
				description: "Delete and reimport this set",
				actionType:  ActionReimport,
				enabled:     !m.selectedSetHasCol,
			})
			actions = append(actions, ActionItem{
				label:       "Delete Set",
				description: "Remove this set from database",
				actionType:  ActionDelete,
				enabled:     !m.selectedSetHasCol,
			})
		} else {
			actions = append(actions, ActionItem{
				label:       "Import Set",
				description: "Import this set into database",
				actionType:  ActionImport,
				enabled:     true,
			})
		}
	}
	actions = append(actions, ActionItem{
		label:       "Import All Sets",
		description: "Import all available sets",
		actionType:  ActionImportAll,
		enabled:     true,
	})
	actions = append(actions, ActionItem{
		label:       "Import New Sets",
		description: "Import only new sets",
		actionType:  ActionImportUpdates,
		enabled:     true,
	})
	return actions
}

func (m ImportModel) executeAction(action ActionItem) (ImportModel, tea.Cmd) {
	if !action.enabled {
		m.errorMsg = "This action is disabled"
		return m, nil
	}
	m.pendingAction = &action
	message := fmt.Sprintf("Are you sure you want to %s?", action.label)
	if len(m.filteredSets) > 0 && m.cursor < len(m.filteredSets) {
		selectedSet := m.filteredSets[m.cursor]
		if action.actionType == ActionImport || action.actionType == ActionDelete || action.actionType == ActionReimport {
			message = fmt.Sprintf("%s\nSet: %s - %s", message, selectedSet.ID, selectedSet.Name)
		}
	}
	m.modal = newModal(
		"Confirm "+action.label,
		message,
		func() tea.Cmd {
			return func() tea.Msg { return importConfirmedMsg{} }
		},
		func() tea.Cmd { return nil },
		m.styleManager, m.width, m.height,
	)
	return m, nil
}

func (m ImportModel) executeConfirmedAction() (ImportModel, tea.Cmd) {
	if m.pendingAction == nil {
		return m, nil
	}
	action := *m.pendingAction
	m.pendingAction = nil
	switch action.actionType {
	case ActionImport:
		if len(m.filteredSets) > 0 && m.cursor < len(m.filteredSets) {
			selectedSet := m.filteredSets[m.cursor]
			m.isImporting = true
			m.importProgress = importProgressMsg{setID: selectedSet.ID}
			return m, tea.Batch(m.spinner.Tick, m.importSetCmd(selectedSet.ID))
		}
	case ActionDelete:
		if len(m.filteredSets) > 0 && m.cursor < len(m.filteredSets) {
			selectedSet := m.filteredSets[m.cursor]
			return m, m.deleteSetCmd(selectedSet.ID)
		}
	case ActionReimport:
		if len(m.filteredSets) > 0 && m.cursor < len(m.filteredSets) {
			selectedSet := m.filteredSets[m.cursor]
			m.isImporting = true
			m.importProgress = importProgressMsg{setID: selectedSet.ID}
			return m, tea.Batch(m.spinner.Tick, tea.Sequence(
				m.deleteSetCmd(selectedSet.ID),
				m.importSetCmd(selectedSet.ID),
			))
		}
	case ActionImportAll:
		for _, set := range m.availableSets {
			m.addToQueue(set.ID, set.Name)
		}
		m.statusMsg = fmt.Sprintf("Added %d sets to queue. Press 's' to start.", len(m.availableSets))
		return m, nil
	case ActionImportUpdates:
		added := 0
		for _, set := range m.availableSets {
			if !m.databaseSetIDs[set.ID] {
				m.addToQueue(set.ID, set.Name)
				added++
			}
		}
		m.statusMsg = fmt.Sprintf("Added %d new sets to queue. Press 's' to start.", added)
		return m, nil
	}
	return m, nil
}

func (m ImportModel) View() string {
	if m.isImporting {
		return m.renderImportProgress()
	}
	return m.renderImportView()
}

func (m ImportModel) fetchSetsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		sets, err := m.pokemonClient.GetSets(ctx)
		if err != nil {
			return fetchSetsErrorMsg{err}
		}
		return fetchSetsSuccessMsg{sets}
	}
}

func (m ImportModel) fetchDatabaseSetsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		apiIDs, err := m.setService.GetAllSetAPIIDs(ctx)
		if err != nil {
			return fetchDatabaseSetsErrorMsg{err}
		}
		return fetchDatabaseSetsSuccessMsg{apiIDs}
	}
}

func (m ImportModel) checkSelectedSetInDB() tea.Cmd {
	if len(m.filteredSets) == 0 || m.cursor >= len(m.filteredSets) {
		return nil
	}
	selectedSet := m.filteredSets[m.cursor]
	return func() tea.Msg {
		ctx := context.Background()
		dbSetID, err := m.setService.GetSetIDByAPIID(ctx, selectedSet.ID)
		if err == sql.ErrNoRows {
			return checkSetNotInDBMsg{}
		}
		if err != nil {
			return checkSetInCollectionErrorMsg{err}
		}
		hasCollections, err := m.setService.SetHasUserCollections(ctx, dbSetID)
		if err != nil {
			return checkSetInCollectionErrorMsg{err}
		}
		return checkSetInDBMsg{hasCollections}
	}
}

func (m ImportModel) importSetCmd(setID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.importService.ImportSpecificSets(ctx, []string{setID})
		if err != nil {
			return importSetErrorMsg{setID, err}
		}
		return importSetSuccessMsg{setID}
	}
}

func (m ImportModel) deleteSetCmd(setID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.importService.DeleteSetByAPIID(ctx, setID)
		if err != nil {
			return deleteSetErrorMsg{setID, err}
		}
		return deleteSetSuccessMsg{setID}
	}
}
