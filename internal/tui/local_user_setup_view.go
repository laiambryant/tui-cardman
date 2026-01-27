package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laiambryant/tui-cardman/internal/auth"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
)

func (m *Model) initLocalUserSetupInputs() {
	var inputs []textinput.Model

	// Name input
	nameInput := textinput.New()
	nameInput.Placeholder = "First Name"
	nameInput.Width = 30
	nameInput.Focus()
	nameInput.Prompt = focusedStyle.Render("> ")
	nameInput.TextStyle = focusedStyle

	// Surname input
	surnameInput := textinput.New()
	surnameInput.Placeholder = "Last Name"
	surnameInput.Width = 30
	surnameInput.Prompt = blurredStyle.Render("> ")

	// Email input
	emailInput := textinput.New()
	emailInput.Placeholder = "email@example.com"
	emailInput.Width = 30
	emailInput.Prompt = blurredStyle.Render("> ")

	inputs = append(inputs, nameInput, surnameInput, emailInput)
	m.inputs = inputs
	m.focusIndex = 0
}

func (m Model) localUserSetupView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🎴 Welcome to CardMan!") + "\n\n")
	b.WriteString(focusedStyle.Render("Let's set up your local profile to get started.") + "\n")
	b.WriteString(blurredStyle.Render("This will be used to manage your card collections.") + "\n\n")

	if m.errorMsg != "" {
		b.WriteString(errorStyle.Render("Error: "+m.errorMsg) + "\n\n")
	}

	// Form fields
	inputs := []string{"First Name:", "Last Name:", "Email:"}
	for i := range m.inputs {
		b.WriteString(inputs[i] + "\n")
		b.WriteString(m.inputs[i].View() + "\n\n")
	}

	// Submit button
	submitButton := "[ Create Profile ]"
	if m.focusIndex == len(m.inputs) {
		submitButton = titleStyle.Render("[ Create Profile ]")
	} else {
		submitButton = blurredStyle.Render(submitButton)
	}
	b.WriteString(submitButton + "\n\n")

	settingsKey := "F1"
	navUp := "↑"
	navDown := "↓"
	submitKey := "Enter"
	quitKey := "Ctrl+C"
	if m.configManager != nil {
		if k := m.configManager.KeyForAction("settings"); k != "" {
			settingsKey = k
		}
		if k := m.configManager.KeyForAction("nav_up"); k != "" {
			navUp = k
		}
		if k := m.configManager.KeyForAction("nav_down"); k != "" {
			navDown = k
		}
		if k := m.configManager.KeyForAction("select"); k != "" {
			submitKey = k
		}
		if k := m.configManager.KeyForAction("quit"); k != "" {
			quitKey = k
		}
	}
	help := fmt.Sprintf("%s: Settings • %s/%s: Navigate • %s: Submit • %s: Quit", settingsKey, navUp, navDown, submitKey, quitKey)
	b.WriteString(helpStyle.Render(help) + "\n")

	return b.String()
}

func (m *Model) handleLocalUserSetup() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.inputs[0].Value())
	surname := strings.TrimSpace(m.inputs[1].Value())
	email := strings.TrimSpace(m.inputs[2].Value())
	if name == "" {
		m.errorMsg = "First name is required"
		return m, nil
	}
	if surname == "" {
		m.errorMsg = "Last name is required"
		return m, nil
	}
	if email == "" {
		m.errorMsg = "Email is required"
		return m, nil
	}
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		m.errorMsg = "Please enter a valid email address"
		return m, nil
	}
	req := createUserRequestForLocalMode(name, surname, email)
	user, err := m.userService.CreateUser(req, "")
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			m.errorMsg = "This email is already registered"
		} else {
			m.errorMsg = fmt.Sprintf("Failed to create profile: %v", err)
		}
		return m, nil
	}
	m.user = user
	if m.configManager == nil {
		configPath := runtimecfg.GetConfigPath()
		configManager, cfgErr := runtimecfg.NewManager(true, configPath, nil, 0)
		if cfgErr != nil {
			fmt.Printf("Warning: failed to initialize config manager, will fallback to default configuration: %v\n", cfgErr)
		} else {
			m.configManager = configManager
		}
	}
	err = m.collectionService.CreateSampleCollectionData(user.ID)
	if err != nil {
		m.errorMsg = fmt.Sprintf("Profile created but failed to add sample collection cards: %v", err)
		m.screen = ScreenMain
		return m, nil
	}
	m.screen = ScreenMain
	m.errorMsg = ""
	return m, nil
}

func createUserRequestForLocalMode(name string, surname string, email string) auth.RegisterRequest {
	req := auth.RegisterRequest{
		Name:     name,
		Surname:  surname,
		Email:    email,
		Password: "",
	}
	return req
}
