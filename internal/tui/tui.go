package tui

import (
	"database/sql"
	"fmt"
	"strings"

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

func (m *Model) initLoginInputs() {
	var t textinput.Model

	// Email input
	t = textinput.New()
	t.Placeholder = "email@example.com"
	t.Focus()
	t.CharLimit = 255
	t.Width = 50
	t.Prompt = focusedStyle.Render("> ")
	m.inputs[0] = t

	// Password input
	t = textinput.New()
	t.Placeholder = "password"
	t.CharLimit = 255
	t.Width = 50
	t.EchoMode = textinput.EchoPassword
	t.EchoCharacter = '•'
	t.Prompt = blurredStyle.Render("> ")
	m.inputs[1] = t

	m.focusIndex = 0
}

func (m *Model) initRegisterInputs() {
	m.inputs = make([]textinput.Model, 4)

	// Name input
	t := textinput.New()
	t.Placeholder = "First Name"
	t.Focus()
	t.CharLimit = 100
	t.Width = 50
	t.Prompt = focusedStyle.Render("> ")
	m.inputs[0] = t

	// Surname input
	t = textinput.New()
	t.Placeholder = "Last Name"
	t.CharLimit = 100
	t.Width = 50
	t.Prompt = blurredStyle.Render("> ")
	m.inputs[1] = t

	// Email input
	t = textinput.New()
	t.Placeholder = "email@example.com"
	t.CharLimit = 255
	t.Width = 50
	t.Prompt = blurredStyle.Render("> ")
	m.inputs[2] = t

	// Password input
	t = textinput.New()
	t.Placeholder = "password"
	t.CharLimit = 255
	t.Width = 50
	t.EchoMode = textinput.EchoPassword
	t.EchoCharacter = '•'
	t.Prompt = blurredStyle.Render("> ")
	m.inputs[3] = t

	m.focusIndex = 0
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

func (m *Model) handleLogin() (tea.Model, tea.Cmd) {
	email := m.inputs[0].Value()
	password := m.inputs[1].Value()
	user, err := m.authService.Login(auth.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		m.errorMsg = err.Error()
		return m, nil
	}
	m.user = user
	m.screen = ScreenMain
	m.errorMsg = ""
	return m, nil
}

func (m *Model) handleRegister() (tea.Model, tea.Cmd) {
	name := m.inputs[0].Value()
	surname := m.inputs[1].Value()
	email := m.inputs[2].Value()
	password := m.inputs[3].Value()
	user, err := m.authService.Register(auth.RegisterRequest{
		Name:     name,
		Surname:  surname,
		Email:    email,
		Password: password,
	})
	if err != nil {
		m.errorMsg = err.Error()
		return m, nil
	}
	m.user = user
	m.screen = ScreenMain
	m.errorMsg = ""
	return m, nil
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

func (m Model) loginView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🃏 CardMan - Login") + "\n\n")
	b.WriteString(focusedStyle.Render("Email:") + "\n")
	b.WriteString(m.inputs[0].View() + "\n\n")
	b.WriteString(blurredStyle.Render("Password:") + "\n")
	b.WriteString(m.inputs[1].View() + "\n\n")
	if m.errorMsg != "" {
		b.WriteString(errorStyle.Render("Error: "+m.errorMsg) + "\n\n")
	}
	button := "[ Login ]"
	if m.focusIndex == len(m.inputs) {
		button = focusedStyle.Render("[ Login ]")
	}
	b.WriteString(button + "\n\n")
	b.WriteString(helpStyle.Render("Tab/Shift+Tab: Navigate • Enter: Submit • Ctrl+C: Quit") + "\n")
	b.WriteString(helpStyle.Render("Don't have an account? Press Enter on the button below") + "\n")
	registerBtn := "[ Register ]"
	b.WriteString(registerBtn + "\n")
	return b.String()
}

func (m Model) registerView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🃏 CardMan - Register") + "\n\n")
	b.WriteString(focusedStyle.Render("First Name:") + "\n")
	b.WriteString(m.inputs[0].View() + "\n\n")
	b.WriteString(blurredStyle.Render("Last Name:") + "\n")
	b.WriteString(m.inputs[1].View() + "\n\n")
	b.WriteString(blurredStyle.Render("Email:") + "\n")
	b.WriteString(m.inputs[2].View() + "\n\n")
	b.WriteString(blurredStyle.Render("Password (8+ chars, 1 uppercase, 1 special):") + "\n")
	b.WriteString(m.inputs[3].View() + "\n\n")
	if m.errorMsg != "" {
		b.WriteString(errorStyle.Render("Error: "+m.errorMsg) + "\n\n")
	}
	button := "[ Register ]"
	if m.focusIndex == len(m.inputs) {
		button = focusedStyle.Render("[ Register ]")
	}
	b.WriteString(button + "\n\n")
	b.WriteString(helpStyle.Render("Tab/Shift+Tab: Navigate • Enter: Submit • Ctrl+C: Quit") + "\n")
	b.WriteString(helpStyle.Render("Already have an account? Press Enter on the button below") + "\n")
	loginBtn := "[ Back to Login ]"
	b.WriteString(loginBtn + "\n")
	return b.String()
}

func (m Model) mainView() string {
	var b strings.Builder

	// Title
	if m.user != nil {
		b.WriteString(titleStyle.Render(fmt.Sprintf("🃏 CardMan - Welcome, %s %s!", m.user.Name, m.user.Surname)) + "\n\n")
	} else {
		b.WriteString(titleStyle.Render("🃏 CardMan - Card Games") + "\n\n")
	}

	// Card games list
	b.WriteString(focusedStyle.Render("Select a card game:") + "\n\n")

	if len(m.cardGames) == 0 {
		b.WriteString(errorStyle.Render("No card games found. Please run migrations.") + "\n")
	} else {
		for i, game := range m.cardGames {
			cursor := " "
			if m.cursor == i {
				cursor = focusedStyle.Render(">")
				b.WriteString(fmt.Sprintf("%s %s\n", cursor, focusedStyle.Render(game.Name)))
			} else {
				b.WriteString(fmt.Sprintf("  %s\n", game.Name))
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓: Navigate • Enter: Select • Ctrl+C: Quit") + "\n")

	return b.String()
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
