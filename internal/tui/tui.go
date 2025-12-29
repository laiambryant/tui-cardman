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
)

// Model is the main application model
type Model struct {
	screen          Screen
	authService     *auth.Service
	userService     IUserService
	cardGameService ICardGameService
	db              *sql.DB
	user            *auth.User

	// Login/Register screen state
	inputs     []textinput.Model
	focusIndex int
	errorMsg   string
	isSSHMode  bool

	// Main view state
	cardGames []CardGame
	cursor    int
}

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle  = focusedStyle
	noStyle      = lipgloss.NewStyle()
	helpStyle    = blurredStyle
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
)

func NewModel(db *sql.DB, isSSHMode bool) (*Model, error) {
	// Initialize services
	userService := NewUserService(db)
	cardGameService := NewCardGameService(db)
	authSvc := auth.NewService(&dbAdapter{userService: userService})

	// Load card games from database
	cardGames, err := cardGameService.GetAllCardGames()
	if err != nil {
		return nil, fmt.Errorf("failed to load card games: %w", err)
	}

	m := &Model{
		authService:     authSvc,
		userService:     userService,
		cardGameService: cardGameService,
		db:              db,
		isSSHMode:       isSSHMode,
		inputs:          make([]textinput.Model, 2),
		cardGames:       cardGames,
		cursor:          0,
	}
	if isSSHMode {
		m.screen = ScreenLogin
		m.initLoginInputs()
	} else {
		m.screen = ScreenMain
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
			if m.screen == ScreenLogin || m.screen == ScreenRegister {
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
			} else if m.screen == ScreenMain {
				// Handle navigation in main view
				s := msg.String()
				if s == "up" || s == "shift+tab" {
					if m.cursor > 0 {
						m.cursor--
					}
				} else if s == "down" || s == "tab" {
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
			}
		}
	}
	if m.screen == ScreenLogin || m.screen == ScreenRegister {
		cmd := m.updateInputs(msg)
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
	case ScreenMain:
		return m.mainView()
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
