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
	screen      Screen
	authService *auth.Service
	db          *sql.DB
	user        *auth.User

	// Login/Register screen state
	inputs     []textinput.Model
	focusIndex int
	errorMsg   string
	isSSHMode  bool
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
	authSvc := auth.NewService(&dbAdapter{db: db})
	m := &Model{
		authService: authSvc,
		db:          db,
		isSSHMode:   isSSHMode,
		inputs:      make([]textinput.Model, 2),
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
	if m.user != nil {
		return fmt.Sprintf("Welcome, %s %s!\n\nYour card collection will be here.\n\nPress Ctrl+C to quit.", m.user.Name, m.user.Surname)
	}
	return "Welcome to CardMan!\n\nYour card collection will be here.\n\nPress Ctrl+C to quit."
}

type dbAdapter struct {
	db *sql.DB
}

func (a *dbAdapter) CreateUser(req auth.RegisterRequest, passwordHash string) (*auth.User, error) {
	return createUser(a.db, req, passwordHash)
}

func (a *dbAdapter) GetUserByEmail(email string) (*auth.User, error) {
	return getUserByEmail(a.db, email)
}

func (a *dbAdapter) UpdateLastLogin(userID int64) error {
	return updateLastLogin(a.db, userID)
}
