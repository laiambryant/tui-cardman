package tui

import (
	"fmt"
	"strings"

	"gihtub.com/laiambryant/tui-cardman/internal/auth"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

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

func (m Model) registerView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("CardMan - Register") + "\n\n")
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
	settingsKey := "F1"
	navNext := "Tab"
	navPrev := "Shift+Tab"
	submitKey := "Enter"
	quitKey := "Ctrl+C"
	if m.configManager != nil {
		if k := m.configManager.KeyForAction("settings"); k != "" {
			settingsKey = k
		}
		if k := m.configManager.KeyForAction("nav_next_tab"); k != "" {
			navNext = k
		}
		if k := m.configManager.KeyForAction("nav_prev_tab"); k != "" {
			navPrev = k
		}
		if k := m.configManager.KeyForAction("select"); k != "" {
			submitKey = k
		}
		if k := m.configManager.KeyForAction("quit"); k != "" {
			quitKey = k
		}
	}
	help := fmt.Sprintf("%s: Settings • %s/%s: Navigate • %s: Submit • %s: Quit", settingsKey, navPrev, navNext, submitKey, quitKey)
	b.WriteString(helpStyle.Render(help) + "\n")
	b.WriteString(helpStyle.Render("Already have an account? Press Enter on the button below") + "\n")
	loginBtn := "[ Back to Login ]"
	b.WriteString(loginBtn + "\n")
	return b.String()
}
