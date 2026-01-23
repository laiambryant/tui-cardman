package tui

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laiambryant/tui-cardman/internal/auth"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
	"github.com/laiambryant/tui-cardman/internal/services/buttonconfig"
)

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

	// Initialize config manager for SSH mode now that we have a user ID
	if m.isSSHMode && m.configManager == nil {
		buttonConfigService := buttonconfig.NewButtonConfigService(m.db)

		// Check if user has existing config, initialize default if not
		ctx := context.Background()
		_, err := buttonConfigService.GetByUserID(ctx, user.ID)
		if err == sql.ErrNoRows {
			if initErr := buttonConfigService.InitializeDefault(ctx, user.ID); initErr != nil {
				fmt.Printf("Warning: failed to initialize default config for user %d: %v\n", user.ID, initErr)
			}
		}

		configPath := runtimecfg.GetConfigPath()
		configManager, err := runtimecfg.NewManager(false, configPath, buttonConfigService, user.ID)
		if err != nil {
			m.errorMsg = fmt.Sprintf("Login successful but failed to load config: %v", err)
			// Continue anyway with default config
		} else {
			m.configManager = configManager
		}
	}

	m.screen = ScreenMain
	m.errorMsg = ""
	return m, nil
}

func (m Model) loginView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("CardMan - Login") + "\n\n")
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
	// Build dynamic help string from config
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
	b.WriteString(helpStyle.Render("Don't have an account? Press Enter on the button below") + "\n")
	registerBtn := "[ Register ]"
	if m.focusIndex == len(m.inputs)+1 {
		registerBtn = focusedStyle.Render("[ Register ]")
	}
	b.WriteString(registerBtn + "\n")
	return b.String()
}
