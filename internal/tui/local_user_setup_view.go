package tui

import (
	"fmt"
	"strings"

	"gihtub.com/laiambryant/tui-cardman/internal/auth"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
		submitButton = focusedStyle.Render("[ Create Profile ]")
	} else {
		submitButton = blurredStyle.Render(submitButton)
	}
	b.WriteString(submitButton + "\n\n")

	b.WriteString(helpStyle.Render("↑/↓: Navigate • Enter: Submit • Ctrl+C: Quit") + "\n")

	return b.String()
}

func (m *Model) handleLocalUserSetup() (tea.Model, tea.Cmd) {
	// Validate inputs
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

	// Basic email validation
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		m.errorMsg = "Please enter a valid email address"
		return m, nil
	}

	// Create user with empty password (local mode)
	req := auth.RegisterRequest{
		Name:     name,
		Surname:  surname,
		Email:    email,
		Password: "", // No password for local mode
	}

	user, err := m.userService.CreateUser(req, "") // Empty password hash
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			m.errorMsg = "This email is already registered"
		} else {
			m.errorMsg = fmt.Sprintf("Failed to create profile: %v", err)
		}
		return m, nil
	}

	// Set the user as current user and go to main screen
	m.user = user

	// Create sample collection data for the new local user
	err = m.collectionService.CreateSampleCollectionData(user.ID)
	if err != nil {
		// Don't fail user creation if sample data fails, just log it
		m.errorMsg = fmt.Sprintf("Profile created but failed to add sample cards: %v", err)
		// Still proceed to main screen after a brief delay
		m.screen = ScreenMain
		return m, nil
	}

	m.screen = ScreenMain
	m.errorMsg = ""

	return m, nil
}
