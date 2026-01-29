package tui

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laiambryant/tui-cardman/internal/auth"
	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
	"github.com/laiambryant/tui-cardman/internal/services/cardgame"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
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
)

// Model is the main application model
type Model struct {
	screen            Screen
	authService       *auth.Service
	userService       user.UserService
	cardGameService   cardgame.CardGameService
	cardService       card.CardService
	collectionService usercollection.UserCollectionService
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
	settingsModel     *SettingsModel
	mainMenuTab       int
	importModel       *ImportModel
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
	styleManager := NewStyleManager(scheme, cfg.UI.OpaqueBackground, cfg.UI.BackgroundStyle)
	userService, cardGameService, cardService, collectionService, authSvc := initServices(db)
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
		db:                db,
		isSSHMode:         isSSHMode,
		configManager:     configManager,
		styleManager:      styleManager,
		inputs:            make([]textinput.Model, 2),
		cardGames:         cardGames,
		cursor:            0,
		mainMenuTab:       0,
	}
	configManager.Subscribe(m.onConfigChange)
	if isSSHMode {
		m.screen = ScreenLogin
		m.initLoginInputs()
	} else {
		hasUsers, err := userService.HasUsers()
		if err != nil {
			return nil, &FailedToCheckForExistingUsersError{Err: err}
		}
		if !hasUsers {
			m.screen = ScreenLocalUserSetup
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
			m.screen = ScreenMain
		}
	}
	return m, nil
}

func initServices(db *sql.DB) (user.UserService, cardgame.CardGameService, card.CardService, usercollection.UserCollectionService, *auth.Service) {
	userService := user.NewUserService(db)
	cardGameService := cardgame.NewCardGameService(db)
	cardService := card.NewCardService(db)
	collectionService := usercollection.NewUserCollectionService(db)
	authSvc := auth.NewService(&dbAdapter{userService: userService})
	return userService, cardGameService, cardService, collectionService, authSvc
}

func (m *Model) onConfigChange(cfg *runtimecfg.RuntimeConfig) {
	scheme := runtimecfg.GetColorScheme(cfg.UI.ColorScheme)
	m.styleManager.UpdateTheme(scheme, cfg.UI.OpaqueBackground, cfg.UI.BackgroundStyle)
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

func (m *Model) getAction(s string) string {
	if m.configManager != nil {
		return m.configManager.MatchAction(s)
	}
	return ""
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

func (m *Model) cycleMainMenuTab(forward bool) {
	if forward {
		m.mainMenuTab = (m.mainMenuTab + 1) % 2
	} else {
		m.mainMenuTab = (m.mainMenuTab - 1 + 2) % 2
	}
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
	} else if action == "nav_left" || s == "left" {
		m.cycleMainMenuTab(false)
	} else if action == "nav_right" || s == "right" {
		m.cycleMainMenuTab(true)
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
		cardGameTabs, err := m.createCardGameTabsModel(selectedGame)
		if err != nil {
			slog.Error("failed to create card game tabs model", "error", err)
			m.errorMsg = fmt.Sprintf("Failed to load card game: %v", err)
			return m, nil
		}
		m.cardGameTabs = cardGameTabs
		m.screen = ScreenCardGameTabs
		m.errorMsg = ""
		return m, m.cardGameTabs.Init()
	}
	return m, nil
}

func (m *Model) selectImport() (tea.Model, tea.Cmd) {
	importModel, err := m.createImportModel()
	if err != nil {
		slog.Error("failed to create import model", "error", err)
		m.errorMsg = fmt.Sprintf("Failed to load import screen: %v", err)
		return m, nil
	}
	m.importModel = &importModel
	m.screen = ScreenImport
	m.errorMsg = ""
	return m, m.importModel.Init()
}

func (m *Model) handleSettingsKey(action string) (tea.Model, tea.Cmd) {
	if action == "settings" && m.screen != ScreenSettings {
		m.settingsModel = NewSettingsModel(m.configManager, m.styleManager)
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
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		action := m.getAction(s)
		slog.Debug("tui key pressed", "key", s, "action", action, "screen", m.screen)
		if newModel, cmd := m.handleSettingsKey(action); cmd != nil {
			return newModel, cmd
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
	if action == "nav_next_tab" || (s == "tab" && m.mainMenuTab == 1) {
		m.cycleMainMenuTab(true)
		return m, nil
	}
	if action == "nav_prev_tab" || s == "shift+tab" {
		m.cycleMainMenuTab(false)
		return m, nil
	}
	if m.mainMenuTab == 0 {
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
	switch m.mainMenuTab {
	case 0:
		return m.selectCardGame()
	case 1:
		return m.selectImport()
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
	case ScreenImport:
		return m.updateImport(msg)
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
		action := m.getAction(keyMsg.String())
		if isBackKey(action, keyMsg.String()) {
			m.screen = ScreenMain
			return m, nil
		}
	}
	return m, cmd
}

func (m *Model) updateImport(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.importModel != nil {
		var cmd tea.Cmd
		*m.importModel, cmd = m.importModel.Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			action := m.getAction(keyMsg.String())
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
	}
	return cardGameTabs, nil
}

func (m *Model) createImportModel() (ImportModel, error) {
	return NewImportModel(m.db, m.configManager, m.styleManager, m.cardGames)
}
