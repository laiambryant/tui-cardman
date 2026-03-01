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
	m.inputs = make([]textinput.Model, 2)
	m.inputs[0] = m.createTextInput("email@example.com", 255, 50, true, false)
	m.inputs[1] = m.createTextInput("password", 255, 50, false, true)
	m.focusIndex = 0
}

func (m *Model) initSSHConfigManager(userID int64) {
	if !m.isSSHMode || m.configManager != nil {
		return
	}
	buttonConfigService := buttonconfig.NewButtonConfigService(m.db)
	ctx := context.Background()
	_, err := buttonConfigService.GetByUserID(ctx, userID)
	if err == sql.ErrNoRows {
		if initErr := buttonConfigService.InitializeDefault(ctx, userID); initErr != nil {
			fmt.Printf("Warning: failed to initialize default config for user %d: %v\n", userID, initErr)
		}
	}
	configPath := runtimecfg.GetConfigPath()
	configManager, err := runtimecfg.NewManager(false, configPath, buttonConfigService, userID)
	if err != nil {
		m.errorMsg = fmt.Sprintf("Login successful but failed to load config: %v", err)
	} else {
		m.configManager = configManager
	}
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
	m.initSSHConfigManager(user.ID)
	m.screen = ScreenMain
	m.errorMsg = ""
	m.initMainScreenImport()
	if m.importModel != nil {
		return m, m.importModel.Init()
	}
	return m, nil
}

func (m Model) loginView() string {
	header := titleStyle.Render("CardMan - Login")
	body := m.renderLoginBody()
	footer := m.renderLoginFooter()
	return renderFramedView(header, body, footer, m.width, m.height, m.styleManager)
}

func (m Model) renderLoginBody() string {
	var b strings.Builder
	b.WriteString(RenderConditionalLabel(true, "Email:") + "\n")
	b.WriteString(m.inputs[0].View() + "\n")
	b.WriteString(RenderConditionalLabel(false, "Password:") + "\n")
	b.WriteString(m.inputs[1].View())
	if m.errorMsg != "" {
		b.WriteString("\n" + errorStyle.Render("Error: "+m.errorMsg))
	}
	b.WriteString("\n" + RenderButton(m.focusIndex == len(m.inputs), "[ Login ]"))
	return b.String()
}

func (m Model) renderLoginFooter() string {
	var b strings.Builder
	b.WriteString(helpStyle.Render("Don't have an account? Press Enter on the button below") + "\n")
	b.WriteString(RenderButton(m.focusIndex == len(m.inputs)+1, "[ Register ]") + "\n")
	b.WriteString(helpStyle.Render(m.buildAuthViewHelpText()))
	return b.String()
}
