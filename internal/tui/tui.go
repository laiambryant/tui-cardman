// Package tui implements the terminal user interface using the Bubble Tea framework.
package tui

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/laiambryant/tui-cardman/internal/auth"
	"github.com/laiambryant/tui-cardman/internal/config"
	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/mtg"
	"github.com/laiambryant/tui-cardman/internal/onepiece"
	"github.com/laiambryant/tui-cardman/internal/pokemontcg"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
	"github.com/laiambryant/tui-cardman/internal/services/cardgame"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/deck"
	"github.com/laiambryant/tui-cardman/internal/services/importruns"
	listservice "github.com/laiambryant/tui-cardman/internal/services/list"
	"github.com/laiambryant/tui-cardman/internal/services/mtgcard"
	"github.com/laiambryant/tui-cardman/internal/services/onepiececard"
	"github.com/laiambryant/tui-cardman/internal/services/pokemoncard"
	"github.com/laiambryant/tui-cardman/internal/services/prices"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
	"github.com/laiambryant/tui-cardman/internal/services/user"
	"github.com/laiambryant/tui-cardman/internal/services/usercollection"
	"github.com/laiambryant/tui-cardman/internal/services/yugiohcard"
	"github.com/laiambryant/tui-cardman/internal/yugioh"
)

// Screen represents different views in the application
type Screen int

const (
	ScreenLogin Screen = iota
	ScreenRegister
	ScreenMain
	ScreenCardGameTabs
	ScreenLocalUserSetup
	ScreenSettings
	ScreenImport
	ScreenSplash
	ScreenCardGameMenu
	ScreenLists
	ScreenDeckBuilder
)

type splashDoneMsg struct{}

// GameStats holds statistics for a specific card game for the currently highlighted game
type GameStats struct {
	TotalCardsOwned int
	ListCount       int
	DeckCount       int
	CollectionValue float64
	SetsComplete    int
	TotalSets       int
}

// gameStatsMsg is sent when stats have been loaded for a card game
type gameStatsMsg struct {
	gameID int64
	stats  GameStats
}

// cardsReloadedMsg is sent when the card list for a game has been refreshed from the database
type cardsReloadedMsg struct {
	gameID int64
	cards  []model.Card
}

// Model is the main application model
type Model struct {
	screen            Screen
	authService       *auth.Service
	userService       user.UserService
	cardGameService   cardgame.CardGameService
	cardService       card.CardService
	collectionService usercollection.UserCollectionService
	listService       listservice.ListService
	deckService       deck.DeckService
	tcgPriceService   prices.TCGPlayerPriceService
	cmPriceService    prices.CardMarketPriceService
	db                *sql.DB
	user              *auth.User
	configManager     *runtimecfg.Manager
	styleManager      *StyleManager
	inputs            []textinput.Model
	focusIndex        int
	errorMsg          string
	isSSHMode         bool
	cardGames         []model.CardGame
	cursor            int
	cardGameTabs      CardGameTabsModel
	cardGameMenuModel *CardGameMenuModel
	listsModel        *ListsModel
	deckBuilderModel  *DeckBuilderModel
	selectedGame      *model.CardGame
	settingsModel     *SettingsModel
	mainFocusPanel    int
	rightPanelCursor  int
	gameStats         *GameStats
	statsLoading      bool
	importModel       *ImportModel
	importers         map[int64]gameimporter.GameImporter
	postSplashScreen  Screen
	width             int
	height            int
}

func NewModel(db *sql.DB, isSSHMode bool) (*Model, error) {
	configPath := runtimecfg.GetConfigPath()
	configManager, err := runtimecfg.NewManager(true, configPath, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config manager: %w", err)
	}
	styleManager := NewStyleManager()
	userService, cardGameService, cardService, collectionService, listSvc, deckSvc, tcgPriceSvc, cmPriceSvc, authSvc := initServices(db)
	cardGames, err := cardGameService.GetAllCardGames()
	if err != nil {
		return nil, fmt.Errorf("failed to load card games: %w", err)
	}
	importers := initImporters(db, cardGames, slog.Default())
	m := &Model{
		authService:       authSvc,
		userService:       userService,
		cardGameService:   cardGameService,
		cardService:       cardService,
		collectionService: collectionService,
		listService:       listSvc,
		deckService:       deckSvc,
		tcgPriceService:   tcgPriceSvc,
		cmPriceService:    cmPriceSvc,
		db:                db,
		isSSHMode:         isSSHMode,
		configManager:     configManager,
		styleManager:      styleManager,
		inputs:            make([]textinput.Model, 2),
		cardGames:         cardGames,
		cursor:            0,
		importers:         importers,
	}
	if isSSHMode {
		m.postSplashScreen = ScreenLogin
		m.initLoginInputs()
	} else {
		hasUsers, err := userService.HasUsers()
		if err != nil {
			return nil, fmt.Errorf("failed to check for existing users: %w", err)
		}
		if !hasUsers {
			m.postSplashScreen = ScreenLocalUserSetup
			m.initLocalUserSetupInputs()
		} else {
			firstUser, err := userService.GetFirstUser()
			if err != nil {
				return nil, fmt.Errorf("failed to get first user for local mode: %w", err)
			}
			m.user = firstUser
			err = userService.UpdateLastLogin(firstUser.ID)
			if err != nil {
				fmt.Printf("Warning: failed to update last login: %v\n", err)
			}
			m.postSplashScreen = ScreenMain
			m.initMainScreenImport()
		}
	}
	m.screen = ScreenSplash
	return m, nil
}

// initImporters constructs game-specific importers keyed by card game DB ID.
func initImporters(db *sql.DB, cardGames []model.CardGame, logger *slog.Logger) map[int64]gameimporter.GameImporter {
	importers := make(map[int64]gameimporter.GameImporter)

	setService := sets.NewSetService(db)
	cardService := card.NewCardService(db)
	importRunService := importruns.NewImportRunService(db)

	// Pokemon
	apiKey := config.GetAPIKey()
	ptcgClient := pokemontcg.NewClient(apiKey)
	pokemonCardSvc := pokemoncard.NewPokemonCardService(db)
	tcgPriceSvc := prices.NewTCGPlayerPriceService(db)
	cmPriceSvc := prices.NewCardMarketPriceService(db)
	ptcgService := pokemontcg.NewImportService(db, ptcgClient, logger,
		importRunService, setService, cardService,
		tcgPriceSvc, cmPriceSvc, pokemonCardSvc)
	ptcgImporter := pokemontcg.NewPokemonGameImporter(ptcgClient, ptcgService, setService)

	// Yu-Gi-Oh!
	ygoClient := yugioh.NewClient()
	yugiohCardSvc := yugiohcard.NewYuGiOhCardService(db)
	ygoService := yugioh.NewImportService(db, ygoClient, logger,
		importRunService, setService, cardService, yugiohCardSvc)
	ygoImporter := yugioh.NewYuGiOhGameImporter(ygoClient, ygoService, setService)

	// Magic: The Gathering
	mtgClient := mtg.NewClient()
	mtgCardSvc := mtgcard.NewMTGCardService(db)
	mtgService := mtg.NewImportService(db, mtgClient, logger,
		importRunService, setService, cardService, mtgCardSvc)
	mtgImporter := mtg.NewMTGGameImporter(mtgClient, mtgService, setService)

	opClient := onepiece.NewClient()
	opCardSvc := onepiececard.NewOnePieceCardService(db)
	opService := onepiece.NewImportService(db, opClient, logger,
		importRunService, setService, cardService, opCardSvc)
	opImporter := onepiece.NewOnePieceGameImporter(opClient, opService, setService)

	for i := range cardGames {
		g := &cardGames[i]
		switch g.Name {
		case "Pokemon":
			importers[g.ID] = ptcgImporter
		case "Yu-Gi-Oh!":
			importers[g.ID] = ygoImporter
		case "Magic: The Gathering":
			importers[g.ID] = mtgImporter
		case "One Piece":
			importers[g.ID] = opImporter
		}
	}
	return importers
}

func initServices(db *sql.DB) (user.UserService, cardgame.CardGameService, card.CardService, usercollection.UserCollectionService, listservice.ListService, deck.DeckService, prices.TCGPlayerPriceService, prices.CardMarketPriceService, *auth.Service) {
	userService := user.NewUserService(db)
	cardGameService := cardgame.NewCardGameService(db)
	cardService := card.NewCardService(db)
	collectionService := usercollection.NewUserCollectionService(db)
	listSvc := listservice.NewListService(db)
	deckSvc := deck.NewDeckService(db)
	tcgPriceSvc := prices.NewTCGPlayerPriceService(db)
	cmPriceSvc := prices.NewCardMarketPriceService(db)
	authSvc := auth.NewService(&dbAdapter{userService: userService})
	return userService, cardGameService, cardService, collectionService, listSvc, deckSvc, tcgPriceSvc, cmPriceSvc, authSvc
}

func isNavigationKey(action, s string) bool {
	return action == "nav_prev_tab" || action == "nav_next_tab" || action == "nav_up" || action == "nav_down" || action == "nav_left" || action == "nav_right" || s == "tab" || s == "shift+tab" || s == "up" || s == "down" || s == "left" || s == "right"
}

func isSelectKey(action, s string) bool {
	return action == "select" || s == "enter" || s == "\r" || s == "\n"
}

func isBackKey(action, s string) bool {
	return action == "back" || action == "quit_alt" || s == "q" || s == "esc"
}

func isQuitKey(action, s string) bool {
	return action == "quit" || action == "quit_alt" || s == "ctrl+c"
}

func (m *Model) updateInputFocus(cmds []tea.Cmd) tea.Cmd {
	for i := 0; i <= len(m.inputs)-1; i++ {
		if i == m.focusIndex {
			cmds[i] = m.inputs[i].Focus()
			m.inputs[i].Prompt = m.styleManager.GetFocusedStyle().Render("> ")
			m.inputs[i].TextStyle = m.styleManager.GetFocusedStyle()
		} else {
			m.inputs[i].Blur()
			m.inputs[i].Prompt = m.styleManager.GetBlurredStyle().Render("> ")
			m.inputs[i].TextStyle = m.styleManager.GetNoStyle()
		}
	}
	return tea.Batch(cmds...)
}

func (m *Model) cycleFocusIndex(forward bool) {
	if forward {
		m.focusIndex++
	} else {
		m.focusIndex--
	}
	maxIndex := len(m.inputs) + 1
	if m.focusIndex > maxIndex {
		m.focusIndex = 0
	} else if m.focusIndex < 0 {
		m.focusIndex = maxIndex
	}
}

func (m *Model) handleInputScreenNavigation(action, s string) tea.Cmd {
	if action == "nav_up" || s == "up" || s == "shift+tab" || action == "nav_left" || s == "left" {
		m.cycleFocusIndex(false)
	} else if action == "nav_down" || s == "down" || action == "nav_next_tab" || s == "tab" || action == "nav_right" || s == "right" {
		m.cycleFocusIndex(true)
	}
	cmds := make([]tea.Cmd, len(m.inputs))
	return m.updateInputFocus(cmds)
}

func (m *Model) handleMainScreenCursor(action, s string) {
	if action == "nav_up" || s == "up" {
		if m.cursor > 0 {
			m.cursor--
		}
	} else if action == "nav_down" || s == "down" || s == "tab" {
		if m.cursor < len(m.cardGames)-1 {
			m.cursor++
		}
	}
}

func (m *Model) handleLoginSelect() (tea.Model, tea.Cmd) {
	if m.focusIndex == len(m.inputs) {
		return m.handleLogin()
	} else if m.focusIndex == len(m.inputs)+1 {
		m.screen = ScreenRegister
		m.initRegisterInputs()
		m.errorMsg = ""
		return m, nil
	}
	return m, nil
}

func (m *Model) handleRegisterSelect() (tea.Model, tea.Cmd) {
	if m.focusIndex == len(m.inputs) {
		return m.handleRegister()
	} else if m.focusIndex == len(m.inputs)+1 {
		m.screen = ScreenLogin
		m.initLoginInputs()
		m.errorMsg = ""
		return m, nil
	}
	return m, nil
}

func (m *Model) handleLocalUserSetupSelect() (tea.Model, tea.Cmd) {
	if m.focusIndex == len(m.inputs) {
		return m.handleLocalUserSetup()
	}
	return m, nil
}

func (m *Model) selectCardGame() (tea.Model, tea.Cmd) {
	if len(m.cardGames) > 0 && m.cursor < len(m.cardGames) {
		selectedGame := &m.cardGames[m.cursor]
		m.selectedGame = selectedGame
		// Switch focus to right panel so user can pick a mode directly
		m.mainFocusPanel = 1
		m.rightPanelCursor = 0
		m.errorMsg = ""
		return m, nil
	}
	return m, nil
}

func (m *Model) initMainScreenImport() {
	if len(m.cardGames) == 0 {
		return
	}
	importModel, err := m.createImportModel(&m.cardGames[0])
	if err != nil {
		slog.Error("failed to create import model", "error", err)
		m.errorMsg = fmt.Sprintf("Failed to load import panel: %v", err)
		return
	}
	m.importModel = &importModel
}

func (m *Model) handleSettingsKey(action string) (tea.Model, tea.Cmd) {
	if action == "settings" && m.screen != ScreenSettings {
		m.settingsModel = NewSettingsModel(m.configManager, m.styleManager)
		m.settingsModel.width = m.width
		m.settingsModel.height = m.height
		m.settingsModel.modal = m.settingsModel.modal.SetDimensions(m.width, m.height)
		m.screen = ScreenSettings
		return m, m.settingsModel.Init()
	}
	return m, nil
}

func (m *Model) handleQuitKey(action, s string) (tea.Model, tea.Cmd) {
	if isQuitKey(action, s) {
		if m.screen == ScreenSettings {
			m.screen = ScreenMain
			return m, nil
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tea.Tick(2500*time.Millisecond, func(t time.Time) tea.Msg {
			return splashDoneMsg{}
		}),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case splashDoneMsg:
		m.screen = m.postSplashScreen
		if m.screen == ScreenMain {
			var cmds []tea.Cmd
			// Load initial stats for the first highlighted card game
			if m.user != nil && len(m.cardGames) > 0 {
				m.statsLoading = true
				cmds = append(cmds, m.loadGameStatsCmd(m.cardGames[m.cursor].ID))
			}
			if m.importModel != nil {
				cmds = append(cmds, m.importModel.Init())
			}
			return m, tea.Batch(cmds...)
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.importModel != nil && m.screen != ScreenImport {
			m.importModel.width = msg.Width
			m.importModel.height = msg.Height
			m.importModel.modal = m.importModel.modal.SetDimensions(msg.Width, msg.Height)
		}
		return m.handleScreenUpdates(msg)
	case tea.KeyMsg:
		s := msg.String()
		action := GetAction(m.configManager, s)
		slog.Debug("tui key pressed", "key", s, "action", action, "screen", m.screen)
		if newModel, cmd := m.handleSettingsKey(action); cmd != nil {
			return newModel, cmd
		}
		// When right panel is focused on main screen, back key switches back to left panel
		if m.screen == ScreenMain && m.mainFocusPanel == 1 {
			if action == "nav_left" || s == "left" {
				m.mainFocusPanel = 0
				return m, nil
			}
			if isBackKey(action, s) {
				m.mainFocusPanel = 0
				return m, nil
			}
		}
		if newModel, cmd := m.handleQuitKey(action, s); cmd != nil {
			return newModel, cmd
		}
		if isNavigationKey(action, s) {
			if m.screen == ScreenLogin || m.screen == ScreenRegister || m.screen == ScreenLocalUserSetup || m.screen == ScreenMain {
				return m.handleNavigationKeys(action, s)
			}
		}
		if isSelectKey(action, s) {
			if m.screen == ScreenLogin || m.screen == ScreenRegister || m.screen == ScreenLocalUserSetup || m.screen == ScreenMain {
				return m.handleSelectKeys()
			}
		}
	}
	return m.handleScreenUpdates(msg)
}

func (m Model) handleNavigationKeys(action, s string) (tea.Model, tea.Cmd) {
	switch m.screen {
	case ScreenLogin, ScreenRegister, ScreenLocalUserSetup:
		return m, m.handleInputScreenNavigation(action, s)
	case ScreenMain:
		return m.handleMainScreenNavigation(action, s)
	}
	return m, nil
}

func (m *Model) handleMainScreenNavigation(action, s string) (tea.Model, tea.Cmd) {
	if action == "nav_right" || s == "right" {
		m.mainFocusPanel = 1
		return m, nil
	}
	if action == "nav_left" || s == "left" {
		m.mainFocusPanel = 0
		return m, nil
	}
	if m.mainFocusPanel == 0 {
		prevCursor := m.cursor
		m.handleMainScreenCursor(action, s)
		// If cursor changed and we have a selected game, load stats for the new highlighted game
		if m.cursor != prevCursor && m.user != nil && len(m.cardGames) > 0 && m.cursor < len(m.cardGames) {
			m.gameStats = nil
			m.statsLoading = true
			return m, m.loadGameStatsCmd(m.cardGames[m.cursor].ID)
		}
	} else {
		// Navigate mode options in right panel
		if action == "nav_up" || s == "up" || s == "k" {
			if m.rightPanelCursor > 0 {
				m.rightPanelCursor--
			}
		} else if action == "nav_down" || s == "down" || s == "j" || s == "tab" {
			const numModes = 4
			if m.rightPanelCursor < numModes-1 {
				m.rightPanelCursor++
			}
		}
	}
	return m, nil
}

func (m Model) handleSelectKeys() (tea.Model, tea.Cmd) {
	switch m.screen {
	case ScreenLogin:
		return m.handleLoginSelect()
	case ScreenRegister:
		return m.handleRegisterSelect()
	case ScreenLocalUserSetup:
		return m.handleLocalUserSetupSelect()
	case ScreenMain:
		return m.handleMainScreenSelect()
	}
	return m, nil
}

func (m *Model) handleMainScreenSelect() (tea.Model, tea.Cmd) {
	if m.mainFocusPanel == 0 {
		return m.selectCardGame()
	}
	// Right panel: navigate to selected mode for the currently highlighted/selected game
	if m.mainFocusPanel == 1 {
		// Use selectedGame if set, otherwise use cursor game
		game := m.selectedGame
		if game == nil && len(m.cardGames) > 0 && m.cursor < len(m.cardGames) {
			game = &m.cardGames[m.cursor]
			m.selectedGame = game
		}
		if game == nil {
			return m, nil
		}
		return m.navigateToMode(game, m.rightPanelCursor)
	}
	return m, nil
}

func (m *Model) navigateToMode(game *model.CardGame, modeIdx int) (tea.Model, tea.Cmd) {
	switch CardGameMenuOption(modeIdx) {
	case MenuMyCollection:
		cardGameTabs, err := m.createCardGameTabsModel(game)
		if err != nil {
			slog.Error("failed to create card game tabs model", "error", err)
			m.errorMsg = fmt.Sprintf("Failed to load card game: %v", err)
			return m, nil
		}
		m.cardGameTabs = cardGameTabs
		m.screen = ScreenCardGameTabs
		return m, m.cardGameTabs.Init()
	case MenuMyLists:
		listsModel, err := m.createListsModel(game)
		if err != nil {
			slog.Error("failed to create lists model", "error", err)
			m.errorMsg = fmt.Sprintf("Failed to load lists: %v", err)
			return m, nil
		}
		m.listsModel = listsModel
		m.screen = ScreenLists
		return m, m.listsModel.Init()
	case MenuMyDecks:
		deckModel, err := m.createDeckBuilderModel(game)
		if err != nil {
			slog.Error("failed to create deck builder model", "error", err)
			m.errorMsg = fmt.Sprintf("Failed to load decks: %v", err)
			return m, nil
		}
		m.deckBuilderModel = deckModel
		m.screen = ScreenDeckBuilder
		return m, m.deckBuilderModel.Init()
	case MenuImportSets:
		newImport, err := m.createImportModel(game)
		if err != nil {
			slog.Error("failed to create import model", "error", err)
			m.errorMsg = fmt.Sprintf("Failed to load import panel: %v", err)
			return m, nil
		}
		m.importModel = &newImport
		m.screen = ScreenImport
		return m, m.importModel.Init()
	}
	return m, nil
}

func (m Model) handleScreenUpdates(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.importModel != nil && m.importModel.queueProcessing && m.screen != ScreenImport && m.screen != ScreenMain {
		if m.isImportMessage(msg) {
			var importCmd tea.Cmd
			*m.importModel, importCmd = m.importModel.Update(msg)
			screenModel, screenCmd := m.dispatchScreenUpdate(msg)
			return screenModel, tea.Batch(importCmd, screenCmd)
		}
	}
	return m.dispatchScreenUpdate(msg)
}

func (m *Model) isImportMessage(msg tea.Msg) bool {
	switch msg.(type) {
	case importSetSuccessMsg, importSetErrorMsg, importProgressMsg,
		importAllSetsSuccessMsg, importAllSetsErrorMsg,
		importNewSetsSuccessMsg, importNewSetsErrorMsg:
		return true
	}
	return false
}

func (m *Model) isImportSuccessMessage(msg tea.Msg) bool {
	switch msg.(type) {
	case importSetSuccessMsg, importAllSetsSuccessMsg, importNewSetsSuccessMsg:
		return true
	}
	return false
}

func (m Model) dispatchScreenUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	// cardsReloadedMsg is handled regardless of current screen so the card
	// search tab stays current after an import finishes in the background.
	if reloadMsg, ok := msg.(cardsReloadedMsg); ok {
		if m.selectedGame != nil && m.selectedGame.ID == reloadMsg.gameID {
			m.cardGameTabs.cards = reloadMsg.cards
			m.cardGameTabs.filteredCards = reloadMsg.cards
			m.cardGameTabs.updateCardTable()
			if m.deckBuilderModel != nil {
				m.deckBuilderModel.cards = reloadMsg.cards
				m.deckBuilderModel.filteredCards = reloadMsg.cards
				m.deckBuilderModel.updateCardTable()
			}
			if m.listsModel != nil {
				m.listsModel.cards = reloadMsg.cards
				m.listsModel.filteredCards = reloadMsg.cards
				m.listsModel.updateListCardTable()
			}
		}
		return m, nil
	}
	switch m.screen {
	case ScreenLogin, ScreenRegister, ScreenLocalUserSetup:
		return m, m.updateInputs(msg)
	case ScreenSettings:
		return m.updateSettings(msg)
	case ScreenCardGameTabs:
		return m.updateCardGameTabs(msg)
	case ScreenCardGameMenu:
		return m.updateCardGameMenu(msg)
	case ScreenLists:
		return m.updateLists(msg)
	case ScreenDeckBuilder:
		return m.updateDeckBuilder(msg)
	case ScreenImport:
		return m.updateImport(msg)
	case ScreenMain:
		// Handle stats message on main screen
		if statsMsg, ok := msg.(gameStatsMsg); ok {
			if len(m.cardGames) > 0 && m.cursor < len(m.cardGames) && m.cardGames[m.cursor].ID == statsMsg.gameID {
				statsCopy := statsMsg.stats
				m.gameStats = &statsCopy
				m.statsLoading = false
			}
		}
	}
	return m, nil
}

func (m *Model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	*m.settingsModel, cmd = m.settingsModel.Update(msg)
	if m.settingsModel.shouldClose {
		m.screen = ScreenMain
	}
	return m, cmd
}

func (m *Model) updateCardGameTabs(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.cardGameTabs, cmd = m.cardGameTabs.Update(msg)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		action := GetAction(m.configManager, keyMsg.String())
		if isBackKey(action, keyMsg.String()) {
			m.screen = ScreenMain
			return m, nil
		}
	}
	return m, cmd
}

func (m *Model) updateCardGameMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.cardGameMenuModel == nil {
		return m, nil
	}
	var cmd tea.Cmd
	*m.cardGameMenuModel, cmd = m.cardGameMenuModel.Update(msg)
	if m.cardGameMenuModel.shouldGoBack {
		m.screen = ScreenMain
		return m, nil
	}
	if m.cardGameMenuModel.optionChosen {
		m.cardGameMenuModel.optionChosen = false
		switch m.cardGameMenuModel.selectedOption {
		case MenuMyCollection:
			cardGameTabs, err := m.createCardGameTabsModel(m.selectedGame)
			if err != nil {
				slog.Error("failed to create card game tabs model", "error", err)
				m.errorMsg = fmt.Sprintf("Failed to load card game: %v", err)
				return m, nil
			}
			m.cardGameTabs = cardGameTabs
			m.screen = ScreenCardGameTabs
			return m, m.cardGameTabs.Init()
		case MenuMyLists:
			listsModel, err := m.createListsModel(m.selectedGame)
			if err != nil {
				slog.Error("failed to create lists model", "error", err)
				m.errorMsg = fmt.Sprintf("Failed to load lists: %v", err)
				return m, nil
			}
			m.listsModel = listsModel
			m.screen = ScreenLists
			return m, m.listsModel.Init()
		case MenuMyDecks:
			deckModel, err := m.createDeckBuilderModel(m.selectedGame)
			if err != nil {
				slog.Error("failed to create deck builder model", "error", err)
				m.errorMsg = fmt.Sprintf("Failed to load decks: %v", err)
				return m, nil
			}
			m.deckBuilderModel = deckModel
			m.screen = ScreenDeckBuilder
			return m, m.deckBuilderModel.Init()
		}
	}
	return m, cmd
}

func (m *Model) updateLists(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.listsModel == nil {
		return m, nil
	}
	var cmd tea.Cmd
	*m.listsModel, cmd = m.listsModel.Update(msg)
	if m.listsModel.shouldGoBack {
		m.listsModel.shouldGoBack = false
		m.screen = ScreenMain
		return m, nil
	}
	return m, cmd
}

func (m *Model) updateDeckBuilder(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.deckBuilderModel == nil {
		return m, nil
	}
	var cmd tea.Cmd
	*m.deckBuilderModel, cmd = m.deckBuilderModel.Update(msg)
	if m.deckBuilderModel.shouldGoBack {
		m.deckBuilderModel.shouldGoBack = false
		m.screen = ScreenMain
		return m, nil
	}
	return m, cmd
}

func (m *Model) updateImport(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.importModel != nil {
		var cmd tea.Cmd
		*m.importModel, cmd = m.importModel.Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			action := GetAction(m.configManager, keyMsg.String())
			if isBackKey(action, keyMsg.String()) && !m.importModel.isImporting {
				m.screen = ScreenMain
				return m, nil
			}
		}
		// After a successful import, reload cards so the card search tab stays current
		if m.isImportSuccessMessage(msg) && m.selectedGame != nil {
			return m, tea.Batch(cmd, m.reloadCardsCmd(m.selectedGame.ID))
		}
		return m, cmd
	}
	return m, nil
}

// reloadCardsCmd fetches fresh cards for the given game and sends a cardsReloadedMsg.
func (m *Model) reloadCardsCmd(gameID int64) tea.Cmd {
	svc := m.cardService
	return func() tea.Msg {
		cards, err := svc.GetCardsByGameID(gameID)
		if err != nil {
			slog.Error("failed to reload cards after import", "game_id", gameID, "error", err)
			return nil
		}
		return cardsReloadedMsg{gameID: gameID, cards: cards}
	}
}

func (m *Model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m Model) View() string {
	switch m.screen {
	case ScreenSplash:
		return m.splashView()
	case ScreenLogin:
		return m.loginView()
	case ScreenRegister:
		return m.registerView()
	case ScreenLocalUserSetup:
		return m.localUserSetupView()
	case ScreenMain:
		return m.mainView()
	case ScreenSettings:
		return m.settingsModel.View()
	case ScreenCardGameTabs:
		return m.cardGameTabs.View()
	case ScreenCardGameMenu:
		if m.cardGameMenuModel != nil {
			return m.cardGameMenuModel.View()
		}
		return ""
	case ScreenLists:
		if m.listsModel != nil {
			return m.listsModel.View()
		}
		return ""
	case ScreenDeckBuilder:
		if m.deckBuilderModel != nil {
			return m.deckBuilderModel.View()
		}
		return ""
	case ScreenImport:
		if m.importModel != nil {
			return m.importModel.View()
		}
		return ""
	default:
		return ""
	}
}

type dbAdapter struct {
	userService user.UserService
}

func (a *dbAdapter) CreateUser(req auth.RegisterRequest, passwordHash string) (*auth.User, error) {
	return a.userService.CreateUser(req, passwordHash)
}

func (a *dbAdapter) GetUserByEmail(email string) (*auth.User, error) {
	return a.userService.GetUserByEmail(email)
}

func (a *dbAdapter) UpdateLastLogin(userID int64) error {
	return a.userService.UpdateLastLogin(userID)
}

// createCardGameTabsModel creates a card game tabs model with loaded data
func (m *Model) createCardGameTabsModel(selectedGame *model.CardGame) (CardGameTabsModel, error) {
	cardGameTabs := NewCardGameTabsModel(selectedGame, m.configManager, m.styleManager)
	cardGameTabs.width = m.width
	cardGameTabs.height = m.height
	cardGameTabs.modal = cardGameTabs.modal.SetDimensions(m.width, m.height)
	cardGameTabs.collectionService = m.collectionService
	cardGameTabs.user = m.user
	pokemonCardSvc := pokemoncard.NewPokemonCardService(m.db)
	pokemonRenderer := NewPokemonCardRenderer(pokemonCardSvc, selectedGame.ID)
	yugiohCardSvc := yugiohcard.NewYuGiOhCardService(m.db)
	yugiohRenderer := NewYuGiOhCardRenderer(yugiohCardSvc, selectedGame.ID)
	mtgCardSvc := mtgcard.NewMTGCardService(m.db)
	mtgRenderer := NewMTGCardRenderer(mtgCardSvc, selectedGame.ID)
	opCardSvc := onepiececard.NewOnePieceCardService(m.db)
	opRenderer := NewOnePieceCardRenderer(opCardSvc, selectedGame.ID)
	cardGameTabs.cardDetail = &CardDetailModel{
		styleManager: m.styleManager,
		tcgService:   m.tcgPriceService,
		cmService:    m.cmPriceService,
		listService:  m.listService,
		width:        m.width,
		height:       m.height,
		renderers:    []CardDetailRenderer{pokemonRenderer, yugiohRenderer, mtgRenderer, opRenderer},
	}
	if m.user != nil {
		cardGameTabs.cardDetail.userID = m.user.ID
	}
	cards, err := m.cardService.GetCardsByGameID(selectedGame.ID)
	if err != nil {
		return cardGameTabs, fmt.Errorf("failed to load cards: %w", err)
	}
	cardGameTabs.cards = cards
	cardGameTabs.filteredCards = cards
	if m.user != nil {
		collections, err := m.collectionService.GetUserCollectionByGameID(m.user.ID, selectedGame.ID)
		if err != nil {
			return cardGameTabs, fmt.Errorf("failed to load user collection: %w", err)
		}
		cardGameTabs.userCollections = collections
		cardGameTabs.filteredCollection = collections
		quantities, err := m.collectionService.GetAllQuantitiesForGame(m.user.ID, selectedGame.ID)
		if err != nil {
			slog.Error("failed to load quantities for game", "user_id", m.user.ID, "game_id", selectedGame.ID, "error", err)
		} else {
			cardGameTabs.quantities.load(quantities)
		}
		cardGameTabs.computeCollectionStats()
		_ = m.collectionService.SnapshotCollectionValue(context.Background(), m.user.ID, selectedGame.ID)
		valueHistory, err := m.collectionService.GetCollectionValueHistory(m.user.ID, selectedGame.ID)
		if err != nil {
			slog.Error("failed to load value history", "error", err)
		} else {
			cardGameTabs.valueHistory = valueHistory
		}
		value, err := m.collectionService.GetCollectionValue(m.user.ID, selectedGame.ID)
		if err != nil {
			slog.Error("failed to load collection value", "user_id", m.user.ID, "game_id", selectedGame.ID, "error", err)
		} else {
			cardGameTabs.collectionValue = value
		}
	}
	return cardGameTabs, nil
}

func (m *Model) createListsModel(selectedGame *model.CardGame) (*ListsModel, error) {
	cards, err := m.cardService.GetCardsByGameID(selectedGame.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load cards: %w", err)
	}
	listsModel := NewListsModel(selectedGame, m.user, m.listService, m.cardService, cards, m.configManager, m.styleManager)
	listsModel.width = m.width
	listsModel.height = m.height
	listsModel.modal = listsModel.modal.SetDimensions(m.width, m.height)
	if m.user != nil {
		lists, err := m.listService.GetListsByUserAndGame(m.user.ID, selectedGame.ID)
		if err != nil {
			slog.Error("failed to load lists", "error", err)
		} else {
			listsModel.lists = lists
		}
	}
	return &listsModel, nil
}

func (m *Model) createDeckBuilderModel(selectedGame *model.CardGame) (*DeckBuilderModel, error) {
	cards, err := m.cardService.GetCardsByGameID(selectedGame.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load cards: %w", err)
	}
	deckModel := NewDeckBuilderModel(selectedGame, m.user, m.deckService, m.cardService, cards, m.configManager, m.styleManager)
	deckModel.width = m.width
	deckModel.height = m.height
	deckModel.modal = deckModel.modal.SetDimensions(m.width, m.height)
	if m.user != nil {
		decks, err := m.deckService.GetDecksByUserAndGame(m.user.ID, selectedGame.ID)
		if err != nil {
			slog.Error("failed to load decks", "error", err)
		} else {
			deckModel.decks = decks
			if len(decks) > 0 {
				deckModel.selectCurrentDeck()
			}
		}
	}
	return &deckModel, nil
}

func (m *Model) createImportModel(game *model.CardGame) (ImportModel, error) {
	importer, ok := m.importers[game.ID]
	if !ok {
		return ImportModel{}, fmt.Errorf("no importer registered for game %q (id=%d)", game.Name, game.ID)
	}
	importModel, err := NewImportModel(importer, game, m.configManager, m.styleManager, m.cardGames)
	if err != nil {
		return ImportModel{}, err
	}
	importModel.width = m.width
	importModel.height = m.height
	return importModel, nil
}

// loadGameStatsCmd loads stats for a given card game asynchronously.
func (m *Model) loadGameStatsCmd(gameID int64) tea.Cmd {
	userID := int64(0)
	if m.user != nil {
		userID = m.user.ID
	}
	collectionSvc := m.collectionService
	listSvc := m.listService
	deckSvc := m.deckService
	cardSvc := m.cardService

	return func() tea.Msg {
		stats := GameStats{}

		// Total cards owned (sum of quantities)
		if userID != 0 {
			quantities, err := collectionSvc.GetAllQuantitiesForGame(userID, gameID)
			if err != nil {
				slog.Error("failed to load quantities for stats", "game_id", gameID, "error", err)
			} else {
				total := 0
				for _, qty := range quantities {
					total += qty
				}
				stats.TotalCardsOwned = total
			}

			// Collection value
			value, err := collectionSvc.GetCollectionValue(userID, gameID)
			if err != nil {
				slog.Error("failed to load collection value for stats", "game_id", gameID, "error", err)
			} else {
				stats.CollectionValue = value
			}

			// List count
			lists, err := listSvc.GetListsByUserAndGame(userID, gameID)
			if err != nil {
				slog.Error("failed to load lists for stats", "game_id", gameID, "error", err)
			} else {
				stats.ListCount = len(lists)
			}

			// Deck count
			decks, err := deckSvc.GetDecksByUserAndGame(userID, gameID)
			if err != nil {
				slog.Error("failed to load decks for stats", "game_id", gameID, "error", err)
			} else {
				stats.DeckCount = len(decks)
			}

			// Set completion: count cards grouped by set, compare owned vs total per set
			cards, err := cardSvc.GetCardsByGameID(gameID)
			if err != nil {
				slog.Error("failed to load cards for stats", "game_id", gameID, "error", err)
			} else if quantities != nil {
				// Build a set -> total cards map and owned set tracker
				type setInfo struct {
					total int
					owned int
				}
				setMap := map[int64]*setInfo{}
				for _, c := range cards {
					if c.Set != nil {
						if _, ok := setMap[c.Set.ID]; !ok {
							setMap[c.Set.ID] = &setInfo{}
						}
						setMap[c.Set.ID].total++
						if quantities[c.ID] > 0 {
							setMap[c.Set.ID].owned++
						}
					}
				}
				totalSets := len(setMap)
				completeSets := 0
				for _, si := range setMap {
					if si.total > 0 && si.owned >= si.total {
						completeSets++
					}
				}
				stats.TotalSets = totalSets
				stats.SetsComplete = completeSets
			}
		}

		return gameStatsMsg{gameID: gameID, stats: stats}
	}
}
