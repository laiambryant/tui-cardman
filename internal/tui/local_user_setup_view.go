package tui

import (
	"fmt"
	"log/slog"
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

func (m *Model) localUserSetupView() string {
	header := RenderBreadcrumbs(m.styleManager, "Welcome")
	body := m.renderLocalUserSetupBody()
	footer := m.renderLocalUserSetupFooter()
	return renderFramedView(header, body, footer, m.width, m.height, m.styleManager)
}

func (m *Model) renderLocalUserSetupBody() string {
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

func (m *Model) renderLocalUserSetupFooter() string {
	return helpStyle.Render(m.buildLocalSetupHelpText())
}

func (m *Model) buildLocalSetupHelpText() string {
	hb := NewHelpBuilder(m.configManager)
	return JoinHelpHints(
		hb.Build(KeyItem{"settings", "F1", "Settings"}),
		hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate"),
		hb.Build(
			KeyItem{"select", "Enter", "Submit"},
			KeyItem{"quit", "Ctrl+C", "Quit"},
		),
	)
}

func (m *Model) initLocalConfigManager() {
	if m.configManager != nil {
		return
	}
	configManager, err := runtimecfg.NewManager(true, runtimecfg.GetConfigPath(), nil, 0)
	if err != nil {
		slog.Warn("failed to initialize config manager, using default keybindings", "error", err)
		return
	}
	m.configManager = configManager
	m.applyConfiguredTheme()
}

func (m *Model) handleLocalUserSetup() (tea.Model, tea.Cmd) {
	req := auth.RegisterRequest{
		Name:    strings.TrimSpace(m.inputs[0].Value()),
		Surname: strings.TrimSpace(m.inputs[1].Value()),
		Email:   strings.TrimSpace(m.inputs[2].Value()),
	}
	switch {
	case req.Name == "":
		m.errorMsg = "First name is required"
	case req.Surname == "":
		m.errorMsg = "Last name is required"
	case req.Email == "":
		m.errorMsg = "Email is required"
	case auth.ValidateEmail(req.Email) != nil:
		m.errorMsg = "Please enter a valid email address"
	default:
		u, err := m.userService.CreateUser(req, "")
		if err != nil {
			m.errorMsg = fmt.Sprintf("Failed to create profile: %v", err)
			return m, nil
		}
		return m, m.enterMainScreen(u)
	}
	return m, nil
}
