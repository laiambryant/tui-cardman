// Package tui implements the terminal user interface using the Bubble Tea framework.
package tui

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/laiambryant/tui-cardman/internal/auth"
	"github.com/laiambryant/tui-cardman/internal/config"
	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/importers"
	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
	"github.com/laiambryant/tui-cardman/internal/services/cardgame"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/deck"
	listservice "github.com/laiambryant/tui-cardman/internal/services/list"
	"github.com/laiambryant/tui-cardman/internal/services/mtgcard"
	"github.com/laiambryant/tui-cardman/internal/services/onepiececard"
	"github.com/laiambryant/tui-cardman/internal/services/pokemoncard"
	"github.com/laiambryant/tui-cardman/internal/services/prices"
	"github.com/laiambryant/tui-cardman/internal/services/user"
	"github.com/laiambryant/tui-cardman/internal/services/usercollection"
	"github.com/laiambryant/tui-cardman/internal/services/yugiohcard"
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
	ScreenLists
	ScreenDeckBuilder
)

type CardGameMenuOption int

const (
	MenuMyCollection CardGameMenuOption = iota
	MenuMyLists
	MenuMyDecks
	MenuImportSets
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

// Model is the main application model
type Model struct {
	screen               Screen
	settingsReturnScreen Screen
	authService          *auth.Service
	userService          user.UserService
	cardService          card.CardService
	collectionService    usercollection.UserCollectionService
	listService          listservice.ListService
	deckService          deck.DeckService
	tcgPriceService      prices.TCGPlayerPriceService
	cmPriceService       prices.CardMarketPriceService
	db                   *sql.DB
	user                 *auth.User
	configManager        *runtimecfg.Manager
	styleManager         *StyleManager
	inputs               []textinput.Model
	focusIndex           int
	errorMsg             string
	isSSHMode            bool
	cardGames            []model.CardGame
	cursor               int
	cardGameTabs         CardGameTabsModel
	listsModel           *ListsModel
	deckBuilderModel     *DeckBuilderModel
	selectedGame         *model.CardGame
	settingsModel        *SettingsModel
	mainFocusPanel       int
	rightPanelCursor     int
	gameStats            *GameStats
	statsLoading         bool
	importModel          *ImportModel
	importers            map[int64]gameimporter.GameImporter
	postSplashScreen     Screen
	width                int
	height               int
}

func NewModel(db *sql.DB, isSSHMode bool) (*Model, error) {
	userService := user.NewUserService(db)
	cardGames, err := cardgame.NewCardGameService(db).GetAllCardGames()
	if err != nil {
		return nil, fmt.Errorf("failed to load card games: %w", err)
	}
	gameImporters, err := initImporters(db, cardGames)
	if err != nil {
		return nil, err
	}
	m := &Model{
		authService:       auth.NewService(userService),
		userService:       userService,
		cardService:       card.NewCardService(db),
		collectionService: usercollection.NewUserCollectionService(db),
		listService:       listservice.NewListService(db),
		deckService:       deck.NewDeckService(db),
		tcgPriceService:   prices.NewTCGPlayerPriceService(db),
		cmPriceService:    prices.NewCardMarketPriceService(db),
		db:                db,
		isSSHMode:         isSSHMode,
		styleManager:      NewStyleManager(),
		cardGames:         cardGames,
		importers:         gameImporters,
		screen:            ScreenSplash,
	}
	if isSSHMode {
		m.postSplashScreen = ScreenLogin
		m.initLoginInputs()
		return m, nil
	}
	m.initLocalConfigManager()
	firstUser, err := userService.GetFirstUser()
	if errors.Is(err, user.ErrNoUsersFound) {
		m.postSplashScreen = ScreenLocalUserSetup
		m.initLocalUserSetupInputs()
		return m, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get first user for local mode: %w", err)
	}
	if err := userService.UpdateLastLogin(firstUser.ID); err != nil {
		slog.Warn("failed to update last login", "user_id", firstUser.ID, "error", err)
	}
	m.user = firstUser
	m.postSplashScreen = ScreenMain
	return m, nil
}

// initImporters constructs game-specific importers keyed by card game DB ID.
// Games without an importer are skipped.
func initImporters(db *sql.DB, cardGames []model.CardGame) (map[int64]gameimporter.GameImporter, error) {
	result := make(map[int64]gameimporter.GameImporter)
	for _, g := range cardGames {
		importer, err := importers.New(db, g.Name, config.GetAPIKey())
		if errors.Is(err, importers.ErrUnsupportedGame) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("initialize %s importer: %w", g.Name, err)
		}
		result[g.ID] = importer
	}
	return result, nil
}

// enterMainScreen completes login, registration or local profile setup: it
// loads the user's configuration and shows the main screen.
func (m *Model) enterMainScreen(u *auth.User) tea.Cmd {
	m.user = u
	m.errorMsg = ""
	if m.isSSHMode {
		m.initSSHConfigManager(u.ID)
	} else {
		m.initLocalConfigManager()
	}
	m.screen = ScreenMain
	return m.loadHighlightedGameStats()
}

// loadHighlightedGameStats loads the stats shown for the game under the cursor.
func (m *Model) loadHighlightedGameStats() tea.Cmd {
	if m.user == nil || m.cursor >= len(m.cardGames) {
		return nil
	}
	m.gameStats = nil
	m.statsLoading = true
	return m.loadGameStatsCmd(m.cardGames[m.cursor].ID)
}

func isSelectKey(action, s string) bool {
	return action == "select" || s == "enter" || s == "\r" || s == "\n"
}

func isBackKey(action, s string) bool {
	return action == "back" || action == "quit_alt" || s == "q" || s == "esc"
}

// isBackKeyWhileTyping reports whether a key means "back" while a text input
// has focus: only non-printable keys (Esc by default) do, so that letters such
// as "q" can still be typed.
func isBackKeyWhileTyping(action, s string) bool {
	return isBackKey(action, s) && isModifierKey(s)
}

func isQuitKey(action, s string) bool {
	return action == "quit" || s == "ctrl+c"
}

// formFocusStep returns -1 or +1 when the key moves the focus between the
// fields and buttons of the login, register and setup forms, 0 otherwise.
// Left and right are left to the text inputs for moving the cursor.
func formFocusStep(action, s string) int {
	switch {
	case action == "nav_up" || action == "nav_prev_tab" || s == "up" || s == "shift+tab":
		return -1
	case action == "nav_down" || action == "nav_next_tab" || s == "down" || s == "tab":
		return 1
	}
	return 0
}

func (m *Model) moveFormFocus(step int) tea.Cmd {
	// Login and register have two buttons below the inputs, setup has one.
	stops := len(m.inputs) + 2
	if m.screen == ScreenLocalUserSetup {
		stops = len(m.inputs) + 1
	}
	m.focusIndex = (m.focusIndex + step + stops) % stops
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
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

func (m *Model) handleFormSelect() (tea.Model, tea.Cmd) {
	switch {
	case m.focusIndex == len(m.inputs) && m.screen == ScreenLogin:
		return m.handleLogin()
	case m.focusIndex == len(m.inputs) && m.screen == ScreenRegister:
		return m.handleRegister()
	case m.focusIndex == len(m.inputs) && m.screen == ScreenLocalUserSetup:
		return m.handleLocalUserSetup()
	case m.focusIndex == len(m.inputs)+1 && m.screen == ScreenLogin:
		m.screen = ScreenRegister
		m.initRegisterInputs()
		m.errorMsg = ""
	case m.focusIndex == len(m.inputs)+1 && m.screen == ScreenRegister:
		m.screen = ScreenLogin
		m.initLoginInputs()
		m.errorMsg = ""
	}
	return m, nil
}

// applyConfiguredTheme applies the persisted theme to the shared style manager.
func (m *Model) applyConfiguredTheme() {
	if m.configManager == nil {
		return
	}
	if theme := m.configManager.Get().UI.Theme; theme != "" {
		m.styleManager.SetTheme(theme)
	}
}

func (m *Model) openSettings() tea.Cmd {
	m.settingsModel = NewSettingsModel(m.configManager, m.styleManager)
	m.settingsModel.width = m.width
	m.settingsModel.height = m.height
	m.settingsModel.modal = m.settingsModel.modal.SetDimensions(m.width, m.height)
	m.settingsReturnScreen = m.screen
	m.screen = ScreenSettings
	return m.settingsModel.Init()
}

// closeSettings returns to the screen settings were opened from, so open
// forms, unsaved edits and running imports are not lost.
func (m *Model) closeSettings() {
	m.screen = m.settingsReturnScreen
	m.refreshThemedComponents()
}

// refreshThemedComponents re-applies theme-derived styles to long-lived
// components after the theme may have changed in settings.
func (m *Model) refreshThemedComponents() {
	if m.importModel != nil {
		m.importModel.spinner.Style = focusedStyle
		m.styleManager.ApplyTextInputStyles(&m.importModel.searchInput)
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tea.Tick(2500*time.Millisecond, func(time.Time) tea.Msg {
			return splashDoneMsg{}
		}),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case splashDoneMsg:
		m.screen = m.postSplashScreen
		if m.screen == ScreenMain {
			return m, m.loadHighlightedGameStats()
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		s := msg.String()
		action := GetAction(m.configManager, s)
		slog.Debug("tui key pressed", "key", s, "action", action, "screen", m.screen)
		if action == "settings" && m.screen != ScreenSettings && m.screen != ScreenSplash {
			return m, m.openSettings()
		}
		if isQuitKey(action, s) {
			if m.screen == ScreenSettings {
				m.settingsModel.cancelChanges()
				m.closeSettings()
				return m, nil
			}
			return m, tea.Quit
		}
		switch m.screen {
		case ScreenLogin, ScreenRegister, ScreenLocalUserSetup:
			if isSelectKey(action, s) {
				return m.handleFormSelect()
			}
			if step := formFocusStep(action, s); step != 0 {
				return m, m.moveFormFocus(step)
			}
		case ScreenMain:
			return m.handleMainScreenKey(action, s)
		}
	}
	return m.handleScreenUpdates(msg)
}

func (m *Model) handleMainScreenKey(action, s string) (tea.Model, tea.Cmd) {
	switch {
	case isSelectKey(action, s):
		return m.handleMainScreenSelect()
	case m.mainFocusPanel == 1 && (action == "nav_left" || s == "left" || isBackKey(action, s)):
		m.mainFocusPanel = 0
	case action == "nav_right" || s == "right":
		m.mainFocusPanel = 1
	case m.mainFocusPanel == 0:
		prevCursor := m.cursor
		if action == "nav_up" || s == "up" {
			m.cursor = max(m.cursor-1, 0)
		} else if action == "nav_down" || s == "down" || s == "tab" {
			m.cursor = min(m.cursor+1, max(len(m.cardGames)-1, 0))
		}
		if m.cursor != prevCursor {
			return m, m.loadHighlightedGameStats()
		}
	case action == "nav_up" || s == "up" || s == "k":
		m.rightPanelCursor = max(m.rightPanelCursor-1, 0)
	case action == "nav_down" || s == "down" || s == "j" || s == "tab":
		m.rightPanelCursor = min(m.rightPanelCursor+1, len(modeLabels)-1)
	}
	return m, nil
}

func (m *Model) handleMainScreenSelect() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.cardGames) {
		return m, nil
	}
	if m.mainFocusPanel == 0 {
		// Switch focus to the right panel so the user can pick a mode directly.
		m.selectedGame = &m.cardGames[m.cursor]
		m.mainFocusPanel = 1
		m.rightPanelCursor = 0
		m.errorMsg = ""
		return m, nil
	}
	// The mode panel belongs to the game currently highlighted in the left panel.
	return m.navigateToMode(&m.cardGames[m.cursor], m.rightPanelCursor)
}

func (m *Model) navigateToMode(game *model.CardGame, modeIdx int) (tea.Model, tea.Cmd) {
	m.selectedGame = game
	var err error
	switch CardGameMenuOption(modeIdx) {
	case MenuMyCollection:
		var cardGameTabs CardGameTabsModel
		if cardGameTabs, err = m.createCardGameTabsModel(game); err == nil {
			m.cardGameTabs = cardGameTabs
			m.screen = ScreenCardGameTabs
			return m, m.cardGameTabs.Init()
		}
	case MenuMyLists:
		if m.listsModel, err = m.createListsModel(game); err == nil {
			m.screen = ScreenLists
			return m, m.listsModel.Init()
		}
	case MenuMyDecks:
		if m.deckBuilderModel, err = m.createDeckBuilderModel(game); err == nil {
			m.screen = ScreenDeckBuilder
			return m, m.deckBuilderModel.Init()
		}
	case MenuImportSets:
		if m.importModel, err = m.createImportModel(game); err == nil {
			m.screen = ScreenImport
			return m, m.importModel.Init()
		}
	}
	if err != nil {
		slog.Error("failed to open screen", "mode", modeLabels[modeIdx], "game", game.Name, "error", err)
		m.errorMsg = fmt.Sprintf("Failed to open %s: %v", modeLabels[modeIdx], err)
	}
	return m, nil
}

func (m *Model) handleScreenUpdates(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Keep a running import progressing while another screen (settings) is shown.
	if m.screen != ScreenImport && m.importModel != nil && m.importModel.isImporting && isImportMessage(msg) {
		var importCmd tea.Cmd
		*m.importModel, importCmd = m.importModel.Update(msg)
		_, screenCmd := m.dispatchScreenUpdate(msg)
		return m, tea.Batch(importCmd, screenCmd)
	}
	return m.dispatchScreenUpdate(msg)
}

func isImportMessage(msg tea.Msg) bool {
	switch msg.(type) {
	case importSetSuccessMsg, importSetErrorMsg, deleteSetSuccessMsg, deleteSetErrorMsg, spinner.TickMsg:
		return true
	}
	return false
}

func (m *Model) dispatchScreenUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.screen {
	case ScreenLogin, ScreenRegister, ScreenLocalUserSetup:
		cmd = m.updateInputs(msg)
	case ScreenSettings:
		*m.settingsModel, cmd = m.settingsModel.Update(msg)
		if m.settingsModel.shouldClose {
			m.closeSettings()
		}
	case ScreenCardGameTabs:
		m.cardGameTabs, cmd = m.cardGameTabs.Update(msg)
		m.goBackIf(&m.cardGameTabs.shouldGoBack)
	case ScreenLists:
		*m.listsModel, cmd = m.listsModel.Update(msg)
		m.goBackIf(&m.listsModel.shouldGoBack)
	case ScreenDeckBuilder:
		*m.deckBuilderModel, cmd = m.deckBuilderModel.Update(msg)
		m.goBackIf(&m.deckBuilderModel.shouldGoBack)
	case ScreenImport:
		*m.importModel, cmd = m.importModel.Update(msg)
		m.goBackIf(&m.importModel.shouldGoBack)
	case ScreenMain:
		if statsMsg, ok := msg.(gameStatsMsg); ok && m.cursor < len(m.cardGames) && m.cardGames[m.cursor].ID == statsMsg.gameID {
			m.gameStats = &statsMsg.stats
			m.statsLoading = false
		}
	}
	return m, cmd
}

// goBackIf returns to the main screen when a sub-screen asked to go back.
func (m *Model) goBackIf(shouldGoBack *bool) {
	if *shouldGoBack {
		*shouldGoBack = false
		m.screen = ScreenMain
	}
}

func (m *Model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m *Model) View() string {
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
	case ScreenLists:
		return m.listsModel.View()
	case ScreenDeckBuilder:
		return m.deckBuilderModel.View()
	case ScreenImport:
		return m.importModel.View()
	}
	return ""
}

// createCardGameTabsModel creates a card game tabs model with loaded data
func (m *Model) createCardGameTabsModel(selectedGame *model.CardGame) (CardGameTabsModel, error) {
	cardGameTabs := NewCardGameTabsModel(selectedGame, m.configManager, m.styleManager)
	cardGameTabs.width = m.width
	cardGameTabs.height = m.height
	cardGameTabs.modal = cardGameTabs.modal.SetDimensions(m.width, m.height)
	cardGameTabs.collectionService = m.collectionService
	cardGameTabs.user = m.user
	cardGameTabs.cardDetail = &CardDetailModel{
		styleManager: m.styleManager,
		tcgService:   m.tcgPriceService,
		cmService:    m.cmPriceService,
		listService:  m.listService,
		width:        m.width,
		height:       m.height,
		renderers: []CardDetailRenderer{
			NewPokemonCardRenderer(pokemoncard.NewPokemonCardService(m.db), m.gameIDByName("Pokemon")),
			NewYuGiOhCardRenderer(yugiohcard.NewYuGiOhCardService(m.db), m.gameIDByName("Yu-Gi-Oh!")),
			NewMTGCardRenderer(mtgcard.NewMTGCardService(m.db), m.gameIDByName("Magic: The Gathering")),
			NewOnePieceCardRenderer(onepiececard.NewOnePieceCardService(m.db), m.gameIDByName("One Piece")),
		},
	}
	cards, err := m.cardService.GetCardsByGameID(selectedGame.ID)
	if err != nil {
		return cardGameTabs, fmt.Errorf("failed to load cards: %w", err)
	}
	cardGameTabs.cards = cards
	cardGameTabs.filteredCards = cards
	if m.user == nil {
		return cardGameTabs, nil
	}
	userID := m.user.ID
	cardGameTabs.cardDetail.userID = userID
	collections, err := m.collectionService.GetUserCollectionByGameID(userID, selectedGame.ID)
	if err != nil {
		return cardGameTabs, fmt.Errorf("failed to load user collection: %w", err)
	}
	cardGameTabs.userCollections = collections
	cardGameTabs.filteredCollection = collections
	if quantities, err := m.collectionService.GetAllQuantitiesForGame(userID, selectedGame.ID); err != nil {
		slog.Error("failed to load quantities for game", "user_id", userID, "game_id", selectedGame.ID, "error", err)
	} else {
		cardGameTabs.quantities.load(quantities)
	}
	cardGameTabs.computeCollectionStats()
	if err := m.collectionService.SnapshotCollectionValue(context.Background(), userID, selectedGame.ID); err != nil {
		slog.Error("failed to snapshot collection value", "error", err)
	}
	if cardGameTabs.valueHistory, err = m.collectionService.GetCollectionValueHistory(userID, selectedGame.ID); err != nil {
		slog.Error("failed to load value history", "error", err)
	}
	if cardGameTabs.collectionValue, err = m.collectionService.GetCollectionValue(userID, selectedGame.ID); err != nil {
		slog.Error("failed to load collection value", "user_id", userID, "game_id", selectedGame.ID, "error", err)
	}
	return cardGameTabs, nil
}

func (m *Model) gameIDByName(name string) int64 {
	for _, g := range m.cardGames {
		if g.Name == name {
			return g.ID
		}
	}
	return 0
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
		if lists, err := m.listService.GetListsByUserAndGame(m.user.ID, selectedGame.ID); err != nil {
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
		if decks, err := m.deckService.GetDecksByUserAndGame(m.user.ID, selectedGame.ID); err != nil {
			slog.Error("failed to load decks", "error", err)
		} else if deckModel.decks = decks; len(decks) > 0 {
			deckModel.selectCurrentDeck()
		}
	}
	return &deckModel, nil
}

func (m *Model) createImportModel(game *model.CardGame) (*ImportModel, error) {
	importer, ok := m.importers[game.ID]
	if !ok {
		return nil, fmt.Errorf("no importer registered for game %q (id=%d)", game.Name, game.ID)
	}
	importModel := NewImportModel(importer, game, m.configManager, m.styleManager)
	importModel.width = m.width
	importModel.height = m.height
	importModel.modal = importModel.modal.SetDimensions(m.width, m.height)
	return &importModel, nil
}

// loadGameStatsCmd loads stats for a given card game asynchronously.
func (m *Model) loadGameStatsCmd(gameID int64) tea.Cmd {
	userID := m.user.ID
	collectionSvc := m.collectionService
	listSvc := m.listService
	deckSvc := m.deckService
	cardSvc := m.cardService
	return func() tea.Msg {
		stats := GameStats{}
		quantities, err := collectionSvc.GetAllQuantitiesForGame(userID, gameID)
		if err != nil {
			slog.Error("failed to load quantities for stats", "game_id", gameID, "error", err)
		}
		for _, qty := range quantities {
			stats.TotalCardsOwned += qty
		}
		if stats.CollectionValue, err = collectionSvc.GetCollectionValue(userID, gameID); err != nil {
			slog.Error("failed to load collection value for stats", "game_id", gameID, "error", err)
		}
		if lists, err := listSvc.GetListsByUserAndGame(userID, gameID); err != nil {
			slog.Error("failed to load lists for stats", "game_id", gameID, "error", err)
		} else {
			stats.ListCount = len(lists)
		}
		if decks, err := deckSvc.GetDecksByUserAndGame(userID, gameID); err != nil {
			slog.Error("failed to load decks for stats", "game_id", gameID, "error", err)
		} else {
			stats.DeckCount = len(decks)
		}
		// A set is complete when every card of it in the database is owned.
		cards, err := cardSvc.GetCardsByGameID(gameID)
		if err != nil {
			slog.Error("failed to load cards for stats", "game_id", gameID, "error", err)
		}
		missing := map[int64]bool{}
		for _, c := range cards {
			if c.Set != nil {
				missing[c.Set.ID] = missing[c.Set.ID] || quantities[c.ID] <= 0
			}
		}
		stats.TotalSets = len(missing)
		for _, isMissing := range missing {
			if !isMissing {
				stats.SetsComplete++
			}
		}
		return gameStatsMsg{gameID: gameID, stats: stats}
	}
}
