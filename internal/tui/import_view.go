package tui

import (
	"context"
	"database/sql"
	"fmt"
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
	"log/slog"
	"strings"
)

type ActionType int

const (
	ActionImport ActionType = iota
	ActionDelete
	ActionReimport
	ActionImportAll
	ActionImportUpdates
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
	focusOnActions    bool
	configManager     *runtimecfg.Manager
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
}

func NewImportModel(db *sql.DB, cfg *runtimecfg.Manager, cardGames []model.CardGame) (ImportModel, error) {
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
	selectedGame := &cardGames[0]
	return ImportModel{
		selectedCardGame: selectedGame,
		searchInput:      searchInput,
		cursor:           0,
		actionCursor:     0,
		focusOnActions:   false,
		configManager:    cfg,
		db:               db,
		pokemonClient:    client,
		importService:    importService,
		setService:       setService,
		databaseSetIDs:   make(map[string]bool),
		cardGames:        cardGames,
		cardGameCursor:   0,
	}, nil
}

func (m ImportModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.fetchSetsCmd(),
		m.fetchDatabaseSetsCmd(),
	)
}

func (m ImportModel) Update(msg tea.Msg) (ImportModel, tea.Cmd) {
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
	case checkSetInCollectionSuccessMsg:
		m.selectedSetHasCol = msg.hasCollections
		return m, nil
	case checkSetInCollectionErrorMsg:
		m.errorMsg = fmt.Sprintf("Failed to check set collections: %v", msg.err)
		return m, nil
	case importSetSuccessMsg:
		m.isImporting = false
		m.statusMsg = fmt.Sprintf("Successfully imported set: %s", msg.setID)
		return m, tea.Batch(m.fetchDatabaseSetsCmd(), m.checkSelectedSetInDB())
	case importSetErrorMsg:
		m.isImporting = false
		m.errorMsg = fmt.Sprintf("Failed to import set %s: %v", msg.setID, msg.err)
		return m, nil
	case deleteSetSuccessMsg:
		m.statusMsg = fmt.Sprintf("Successfully deleted set: %s", msg.setID)
		return m, tea.Batch(m.fetchDatabaseSetsCmd(), m.checkSelectedSetInDB())
	case deleteSetErrorMsg:
		m.errorMsg = fmt.Sprintf("Failed to delete set %s: %v", msg.setID, msg.err)
		return m, nil
	case importProgressMsg:
		m.importProgress = msg
		return m, nil
	case importAllSetsSuccessMsg:
		m.isImporting = false
		m.statusMsg = "Successfully imported all sets"
		return m, m.fetchDatabaseSetsCmd()
	case importAllSetsErrorMsg:
		m.isImporting = false
		m.errorMsg = fmt.Sprintf("Failed to import all sets: %v", msg.err)
		return m, nil
	case importNewSetsSuccessMsg:
		m.isImporting = false
		m.statusMsg = "Successfully imported new sets"
		return m, m.fetchDatabaseSetsCmd()
	case importNewSetsErrorMsg:
		m.isImporting = false
		m.errorMsg = fmt.Sprintf("Failed to import new sets: %v", msg.err)
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
	case importProgressMsg:
		m.importProgress = msg
		return m, nil
	case importSetSuccessMsg:
		m.isImporting = false
		m.statusMsg = fmt.Sprintf("Successfully imported set: %s", msg.setID)
		return m, tea.Batch(m.fetchDatabaseSetsCmd(), m.checkSelectedSetInDB())
	case importSetErrorMsg:
		m.isImporting = false
		m.errorMsg = fmt.Sprintf("Failed to import set %s: %v", msg.setID, msg.err)
		return m, nil
	case importAllSetsSuccessMsg:
		m.isImporting = false
		m.statusMsg = "Successfully imported all sets"
		return m, m.fetchDatabaseSetsCmd()
	case importAllSetsErrorMsg:
		m.isImporting = false
		m.errorMsg = fmt.Sprintf("Failed to import all sets: %v", msg.err)
		return m, nil
	case importNewSetsSuccessMsg:
		m.isImporting = false
		m.statusMsg = "Successfully imported new sets"
		return m, m.fetchDatabaseSetsCmd()
	case importNewSetsErrorMsg:
		m.isImporting = false
		m.errorMsg = fmt.Sprintf("Failed to import new sets: %v", msg.err)
		return m, nil
	}
	return m, nil
}

func (m ImportModel) handleKeyMsg(msg tea.KeyMsg) (ImportModel, tea.Cmd) {
	s := msg.String()
	action := ""
	if m.configManager != nil {
		action = m.configManager.MatchAction(s)
	}
	if action == "quit" || s == "ctrl+c" {
		return m, tea.Quit
	}
	if s == "tab" {
		m.focusOnActions = !m.focusOnActions
		if m.focusOnActions {
			m.actionCursor = 0
		}
		return m, nil
	}
	if m.focusOnActions {
		return m.handleActionNavigation(msg)
	}
	return m.handleSetListNavigation(msg)
}

func (m ImportModel) handleSetListNavigation(msg tea.KeyMsg) (ImportModel, tea.Cmd) {
	s := msg.String()
	action := ""
	if m.configManager != nil {
		action = m.configManager.MatchAction(s)
	}
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
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	if m.searchInput.Value() != "" {
		m.filterSets()
	} else {
		m.filteredSets = m.availableSets
	}
	return m, cmd
}

func (m ImportModel) handleActionNavigation(msg tea.KeyMsg) (ImportModel, tea.Cmd) {
	s := msg.String()
	action := ""
	if m.configManager != nil {
		action = m.configManager.MatchAction(s)
	}
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

func (m ImportModel) filterSets() {
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
	switch action.actionType {
	case ActionImport:
		if len(m.filteredSets) > 0 && m.cursor < len(m.filteredSets) {
			selectedSet := m.filteredSets[m.cursor]
			m.isImporting = true
			m.importProgress = importProgressMsg{setID: selectedSet.ID}
			return m, m.importSetCmd(selectedSet.ID)
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
			return m, tea.Sequence(
				m.deleteSetCmd(selectedSet.ID),
				m.importSetCmd(selectedSet.ID),
			)
		}
	case ActionImportAll:
		m.isImporting = true
		return m, m.importAllSetsCmd()
	case ActionImportUpdates:
		m.isImporting = true
		return m, m.importNewSetsCmd()
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
	m.selectedSetInDB = m.databaseSetIDs[selectedSet.ID]
	if !m.selectedSetInDB {
		m.selectedSetHasCol = false
		return nil
	}
	return func() tea.Msg {
		ctx := context.Background()
		dbSetID, err := m.setService.GetSetIDByAPIID(ctx, selectedSet.ID)
		if err != nil {
			return checkSetInCollectionErrorMsg{err}
		}
		hasCollections, err := m.setService.SetHasUserCollections(ctx, dbSetID)
		if err != nil {
			return checkSetInCollectionErrorMsg{err}
		}
		return checkSetInCollectionSuccessMsg{hasCollections}
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

func (m ImportModel) importAllSetsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.importService.ImportAllSets(ctx)
		if err != nil {
			return importAllSetsErrorMsg{err}
		}
		return importAllSetsSuccessMsg{}
	}
}

func (m ImportModel) importNewSetsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.importService.ImportNewSets(ctx)
		if err != nil {
			return importNewSetsErrorMsg{err}
		}
		return importNewSetsSuccessMsg{}
	}
}
