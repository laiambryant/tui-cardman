package tui

import (
	"database/sql"
	"fmt"

	"gihtub.com/laiambryant/tui-cardman/internal/auth"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Screen represents different views in the application
type Screen int

const (
	ScreenLogin Screen = iota
	ScreenRegister
	ScreenMain
	ScreenCardGameTabs
	ScreenLocalUserSetup
)

// Model is the main application model
type Model struct {
	screen            Screen
	authService       *auth.Service
	userService       IUserService
	cardGameService   ICardGameService
	cardService       ICardService
	collectionService IUserCollectionService
	db                *sql.DB
	user              *auth.User

	// Login/Register screen state
	inputs     []textinput.Model
	focusIndex int
	errorMsg   string
	isSSHMode  bool

	// Main view state
	cardGames []CardGame
	cursor    int

	// Card game tabs state
	cardGameTabs CardGameTabsModel
}

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	noStyle      = lipgloss.NewStyle()
	helpStyle    = blurredStyle
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
)

func NewModel(db *sql.DB, isSSHMode bool) (*Model, error) {
	// Initialize services
	userService := NewUserService(db)
	cardGameService := NewCardGameService(db)
	cardService := NewCardService(db)
	collectionService := NewUserCollectionService(db)
	authSvc := auth.NewService(&dbAdapter{userService: userService})

	// Load card games from database
	cardGames, err := cardGameService.GetAllCardGames()
	if err != nil {
		return nil, fmt.Errorf("failed to load card games: %w", err)
	}

	m := &Model{
		authService:       authSvc,
		userService:       userService,
		cardGameService:   cardGameService,
		cardService:       cardService,
		collectionService: collectionService,
		db:                db,
		isSSHMode:         isSSHMode,
		inputs:            make([]textinput.Model, 2),
		cardGames:         cardGames,
		cursor:            0,
	}
	if isSSHMode {
		m.screen = ScreenLogin
		m.initLoginInputs()
	} else {
		// Local mode - check if users exist
		hasUsers, err := userService.HasUsers()
		if err != nil {
			return nil, fmt.Errorf("failed to check for existing users: %w", err)
		}

		if !hasUsers {
			// No users exist, show local user setup
			m.screen = ScreenLocalUserSetup
			m.initLocalUserSetupInputs()
		} else {
			// Users exist, automatically log in the first user for local mode
			firstUser, err := userService.GetFirstUser()
			if err != nil {
				return nil, fmt.Errorf("failed to get first user for local mode: %w", err)
			}
			m.user = firstUser
			// Update last login for the user
			err = userService.UpdateLastLogin(firstUser.ID)
			if err != nil {
				// Don't fail initialization if we can't update last login
				fmt.Printf("Warning: failed to update last login: %v\n", err)
			}
			m.screen = ScreenMain
		}
	}
	return m, nil
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "shift+tab", "up", "down":
			switch m.screen {
			case ScreenLogin, ScreenRegister, ScreenLocalUserSetup:
				s := msg.String()
				if s == "up" || s == "shift+tab" {
					m.focusIndex--
				} else {
					m.focusIndex++
				}
				if m.focusIndex > len(m.inputs) {
					m.focusIndex = 0
				} else if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs)
				}
				cmds := make([]tea.Cmd, len(m.inputs))
				for i := 0; i <= len(m.inputs)-1; i++ {
					if i == m.focusIndex {
						cmds[i] = m.inputs[i].Focus()
						m.inputs[i].Prompt = focusedStyle.Render("> ")
						m.inputs[i].TextStyle = focusedStyle
					} else {
						m.inputs[i].Blur()
						m.inputs[i].Prompt = blurredStyle.Render("> ")
						m.inputs[i].TextStyle = noStyle
					}
				}
				return m, tea.Batch(cmds...)
			case ScreenMain:
				// Handle navigation in main view
				s := msg.String()
				switch s {
				case "up", "shift+tab":
					if m.cursor > 0 {
						m.cursor--
					}
				case "down", "tab":
					if m.cursor < len(m.cardGames)-1 {
						m.cursor++
					}
				}
				return m, nil
			}

		case "enter":
			switch m.screen {
			case ScreenLogin:
				if m.focusIndex == len(m.inputs) {
					// Switch to register
					m.screen = ScreenRegister
					m.initRegisterInputs()
					m.errorMsg = ""
					return m, nil
				}
				// Attempt login
				return m.handleLogin()
			case ScreenRegister:
				if m.focusIndex == len(m.inputs) {
					// Switch to login
					m.screen = ScreenLogin
					m.initLoginInputs()
					m.errorMsg = ""
					return m, nil
				}
				// Attempt registration
				return m.handleRegister()
			case ScreenLocalUserSetup:
				if m.focusIndex == len(m.inputs) {
					// Submit local user setup
					return m.handleLocalUserSetup()
				}
			case ScreenMain:
				// Select card game and go to tabs view
				if len(m.cardGames) > 0 && m.cursor < len(m.cardGames) {
					selectedGame := &m.cardGames[m.cursor]
					cardGameTabs, err := m.createCardGameTabsModel(selectedGame)
					if err != nil {
						// Could add error handling here, for now just ignore
						return m, nil
					}
					m.cardGameTabs = cardGameTabs
					m.screen = ScreenCardGameTabs
					return m, m.cardGameTabs.Init()
				}
			}
		}
	}

	// Handle screen-specific updates
	switch m.screen {
	case ScreenLogin, ScreenRegister, ScreenLocalUserSetup:
		cmd := m.updateInputs(msg)
		return m, cmd
	case ScreenCardGameTabs:
		var cmd tea.Cmd
		m.cardGameTabs, cmd = m.cardGameTabs.Update(msg)
		// Handle quit from card game tabs - check for specific keys to return to main
		if keyMsg, ok := msg.(tea.KeyMsg); ok && (keyMsg.String() == "q" || keyMsg.String() == "esc") {
			// Return to main screen
			m.screen = ScreenMain
			return m, nil
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
	case ScreenCardGameTabs:
		return m.cardGameTabs.View()
	default:
		return ""
	}
}

type dbAdapter struct {
	userService IUserService
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
func (m *Model) createCardGameTabsModel(selectedGame *CardGame) (CardGameTabsModel, error) {
	cardGameTabs := NewCardGameTabsModel(selectedGame)

	// Load cards for this game
	cards, err := m.cardService.GetCardsByGameID(selectedGame.ID)
	if err != nil {
		return cardGameTabs, fmt.Errorf("failed to load cards: %w", err)
	}
	cardGameTabs.cards = cards
	cardGameTabs.filteredCards = cards

	// Load user collection if user is logged in
	if m.user != nil {
		collections, err := m.collectionService.GetUserCollectionByGameID(m.user.ID, selectedGame.ID)
		if err != nil {
			return cardGameTabs, fmt.Errorf("failed to load user collection: %w", err)
		}
		cardGameTabs.userCollections = collections
		cardGameTabs.filteredCollection = collections
	}

	return cardGameTabs, nil
}
