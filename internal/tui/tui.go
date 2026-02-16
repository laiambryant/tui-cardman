package tui

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laiambryant/tui-cardman/internal/auth"
	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
	"github.com/laiambryant/tui-cardman/internal/services/cardgame"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	listservice "github.com/laiambryant/tui-cardman/internal/services/list"
	"github.com/laiambryant/tui-cardman/internal/services/user"
	"github.com/laiambryant/tui-cardman/internal/services/usercollection"
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
)

type splashDoneMsg struct{}

// Model is the main application model
type Model struct {
	screen            Screen
	authService       *auth.Service
	userService       user.UserService
	cardGameService   cardgame.CardGameService
	cardService       card.CardService
	collectionService usercollection.UserCollectionService
	listService       listservice.ListService
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
	selectedGame      *model.CardGame
	settingsModel     *SettingsModel
	mainFocusPanel    int
	importModel       *ImportModel
	postSplashScreen  Screen
	width             int
	height            int
}

func NewModel(db *sql.DB, isSSHMode bool) (*Model, error) {
	configPath := runtimecfg.GetConfigPath()
	configManager, err := runtimecfg.NewManager(true, configPath, nil, 0)
	if err != nil {
		return nil, &FailedToInitializeConfigManagerError{Err: err}
	}
	cfg := configManager.Get()
	scheme := runtimecfg.GetColorScheme(cfg.UI.ColorScheme)
	styleManager := NewStyleManager(scheme)
	userService, cardGameService, cardService, collectionService, listSvc, authSvc := initServices(db)
	cardGames, err := cardGameService.GetAllCardGames()
	if err != nil {
		return nil, &FailedToLoadCardGamesError{Err: err}
	}
	m := &Model{
		authService:       authSvc,
		userService:       userService,
		cardGameService:   cardGameService,
		cardService:       cardService,
		collectionService: collectionService,
		listService:       listSvc,
		db:                db,
		isSSHMode:         isSSHMode,
		configManager:     configManager,
		styleManager:      styleManager,
		inputs:            make([]textinput.Model, 2),
		cardGames:         cardGames,
		cursor:            0,
	}
	configManager.Subscribe(m.onConfigChange)
	if isSSHMode {
		m.postSplashScreen = ScreenLogin
		m.initLoginInputs()
	} else {
		hasUsers, err := userService.HasUsers()
		if err != nil {
			return nil, &FailedToCheckForExistingUsersError{Err: err}
		}
		if !hasUsers {
			m.postSplashScreen = ScreenLocalUserSetup
			m.initLocalUserSetupInputs()
		} else {
			firstUser, err := userService.GetFirstUser()
			if err != nil {
				return nil, &FailedToGetFirstUserForLocalModeError{Err: err}
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

func initServices(db *sql.DB) (user.UserService, cardgame.CardGameService, card.CardService, usercollection.UserCollectionService, listservice.ListService, *auth.Service) {
	userService := user.NewUserService(db)
	cardGameService := cardgame.NewCardGameService(db)
	cardService := card.NewCardService(db)
	collectionService := usercollection.NewUserCollectionService(db)
	listSvc := listservice.NewListService(db)
	authSvc := auth.NewService(&dbAdapter{userService: userService})
	return userService, cardGameService, cardService, collectionService, listSvc, authSvc
}

func (m *Model) onConfigChange(cfg *runtimecfg.RuntimeConfig) {
	scheme := runtimecfg.GetColorScheme(cfg.UI.ColorScheme)
	m.styleManager.UpdateTheme(scheme)
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
		m.cardGameMenuModel = NewCardGameMenuModel(selectedGame, m.styleManager, m.configManager)
		m.cardGameMenuModel.width = m.width
		m.cardGameMenuModel.height = m.height
		m.screen = ScreenCardGameMenu
		m.errorMsg = ""
		return m, nil
	}
	return m, nil
}

func (m *Model) initMainScreenImport() {
	importModel, err := m.createImportModel()
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
		if m.screen == ScreenMain && m.importModel != nil {
			return m, m.importModel.Init()
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m.handleScreenUpdates(msg)
	case tea.KeyMsg:
		s := msg.String()
		action := GetAction(m.configManager,s)
		slog.Debug("tui key pressed", "key", s, "action", action, "screen", m.screen)
		if newModel, cmd := m.handleSettingsKey(action); cmd != nil {
			return newModel, cmd
		}
		// When import panel is focused on main screen, intercept keys
		if m.screen == ScreenMain && m.mainFocusPanel == 1 && m.importModel != nil {
			// Left arrow switches back to card games panel
			if action == "nav_left" || s == "left" {
				m.mainFocusPanel = 0
				return m, nil
			}
			// Back key switches to card games panel instead of quitting
			if isBackKey(action, s) {
				m.mainFocusPanel = 0
				return m, nil
			}
			// Quit key still quits
			if isQuitKey(action, s) {
				return m, tea.Quit
			}
			// Forward all other keys to import model
			var cmd tea.Cmd
			*m.importModel, cmd = m.importModel.Update(msg)
			return m, cmd
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
		m.handleMainScreenCursor(action, s)
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
	return m, nil
}

func (m Model) handleScreenUpdates(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case ScreenImport:
		return m.updateImport(msg)
	case ScreenMain:
		if m.importModel != nil {
			var cmd tea.Cmd
			*m.importModel, cmd = m.importModel.Update(msg)
			return m, cmd
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
		action := GetAction(m.configManager,keyMsg.String())
		if isBackKey(action, keyMsg.String()) {
			m.screen = ScreenCardGameMenu
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
		m.screen = ScreenCardGameMenu
		return m, nil
	}
	return m, cmd
}

func (m *Model) updateImport(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.importModel != nil {
		var cmd tea.Cmd
		*m.importModel, cmd = m.importModel.Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			action := GetAction(m.configManager,keyMsg.String())
			if isBackKey(action, keyMsg.String()) && !m.importModel.isImporting {
				m.screen = ScreenMain
				return m, nil
			}
		}
		return m, cmd
	}
	return m, nil
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
	cards, err := m.cardService.GetCardsByGameID(selectedGame.ID)
	if err != nil {
		return cardGameTabs, &FailedToLoadCardsError{Err: err}
	}
	cardGameTabs.cards = cards
	cardGameTabs.filteredCards = cards
	if m.user != nil {
		collections, err := m.collectionService.GetUserCollectionByGameID(m.user.ID, selectedGame.ID)
		if err != nil {
			return cardGameTabs, &FailedToLoadUserCollectionError{Err: err}
		}
		cardGameTabs.userCollections = collections
		cardGameTabs.filteredCollection = collections
		quantities, err := m.collectionService.GetAllQuantitiesForGame(m.user.ID, selectedGame.ID)
		if err != nil {
			slog.Error("failed to load quantities for game", "user_id", m.user.ID, "game_id", selectedGame.ID, "error", err)
		} else {
			cardGameTabs.dbQuantities = quantities
		}
		cardGameTabs.computeCollectionStats()
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
		return nil, &FailedToLoadCardsError{Err: err}
	}
	listsModel := NewListsModel(selectedGame, m.user, m.listService, cards, m.configManager, m.styleManager)
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

func (m *Model) createImportModel() (ImportModel, error) {
	importModel, err := NewImportModel(m.db, m.configManager, m.styleManager, m.cardGames)
	if err != nil {
		return ImportModel{}, err
	}
	importModel.width = m.width
	importModel.height = m.height
	return importModel, nil
}
