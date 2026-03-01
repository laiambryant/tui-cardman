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
	m.inputs = make([]textinput.Model, 3)
	m.inputs[0] = m.createTextInput("First Name", 30, 30, true, false)
	m.inputs[1] = m.createTextInput("Last Name", 30, 30, false, false)
	m.inputs[2] = m.createTextInput("email@example.com", 255, 30, false, false)
	m.focusIndex = 0
}

func (m Model) localUserSetupView() string {
	header := titleStyle.Render("Welcome to CardMan!")
	body := m.renderLocalUserSetupBody()
	footer := m.renderLocalUserSetupFooter()
	return renderFramedView(header, body, footer, m.width, m.height, m.styleManager)
}

func (m Model) renderLocalUserSetupBody() string {
	var b strings.Builder
	b.WriteString(focusedStyle.Render("Let's set up your local profile to get started.") + "\n")
	b.WriteString(blurredStyle.Render("This will be used to manage your card collections.") + "\n")
	if m.errorMsg != "" {
		b.WriteString(errorStyle.Render("Error: "+m.errorMsg) + "\n")
	}
	fields := []string{"First Name:", "Last Name:", "Email:"}
	for i := range m.inputs {
		b.WriteString(fields[i] + "\n")
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n" + RenderButton(m.focusIndex == len(m.inputs), "[ Create Profile ]"))
	return b.String()
}

func (m Model) renderLocalUserSetupFooter() string {
	return helpStyle.Render(m.buildLocalSetupHelpText())
}

func (m Model) buildLocalSetupHelpText() string {
	hb := NewHelpBuilder(m.configManager)
	return hb.Build(KeyItem{"settings", "F1", "Settings"}) + " • " + hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate") + " • " + hb.Build(
		KeyItem{"select", "Enter", "Submit"},
		KeyItem{"quit", "Ctrl+C", "Quit"},
	)
}

func (m *Model) validateLocalUserSetupInputs(name, surname, email string) string {
	name = strings.TrimSpace(name)
	surname = strings.TrimSpace(surname)
	email = strings.TrimSpace(email)
	if name == "" {
		return "First name is required"
	}
	if surname == "" {
		return "Last name is required"
	}
	if email == "" {
		return "Email is required"
	}
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return "Please enter a valid email address"
	}
	return ""
}

func (m *Model) initLocalConfigManager() {
	if m.configManager != nil {
		return
	}
	configPath := runtimecfg.GetConfigPath()
	configManager, cfgErr := runtimecfg.NewManager(true, configPath, nil, 0)
	if cfgErr != nil {
		fmt.Printf("Warning: failed to initialize config manager, will fallback to default configuration: %v\n", cfgErr)
	} else {
		m.configManager = configManager
	}
}

func (m *Model) handleLocalUserSetup() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.inputs[0].Value())
	surname := strings.TrimSpace(m.inputs[1].Value())
	email := strings.TrimSpace(m.inputs[2].Value())
	if validationErr := m.validateLocalUserSetupInputs(name, surname, email); validationErr != "" {
		m.errorMsg = validationErr
		return m, nil
	}
	req := auth.RegisterRequest{
		Name:     name,
		Surname:  surname,
		Email:    email,
		Password: "",
	}
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
	m.initLocalConfigManager()
	err = m.collectionService.CreateSampleCollectionData(user.ID)
	if err != nil {
		m.errorMsg = fmt.Sprintf("Profile created but failed to add sample collection cards: %v", err)
		m.screen = ScreenMain
		m.initMainScreenImport()
		if m.importModel != nil {
			return m, m.importModel.Init()
		}
		return m, nil
	}
	m.screen = ScreenMain
	m.errorMsg = ""
	m.initMainScreenImport()
	if m.importModel != nil {
		return m, m.importModel.Init()
	}
	return m, nil
}
