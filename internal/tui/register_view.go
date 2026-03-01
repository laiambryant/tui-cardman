package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laiambryant/tui-cardman/internal/auth"
)

func (m *Model) createTextInput(placeholder string, charLimit, width int, focused bool, isPassword bool) textinput.Model {
	t := textinput.New()
	t.Placeholder = placeholder
	t.CharLimit = charLimit
	t.Width = width
	if focused {
		t.Focus()
		t.Prompt = focusedStyle.Render("> ")
	} else {
		t.Prompt = blurredStyle.Render("> ")
	}
	if isPassword {
		t.EchoMode = textinput.EchoPassword
		t.EchoCharacter = '•'
	}
	return t
}

func (m *Model) initRegisterInputs() {
	m.inputs = make([]textinput.Model, 4)
	m.inputs[0] = m.createTextInput("First Name", 100, 50, true, false)
	m.inputs[1] = m.createTextInput("Last Name", 100, 50, false, false)
	m.inputs[2] = m.createTextInput("email@example.com", 255, 50, false, false)
	m.inputs[3] = m.createTextInput("password", 255, 50, false, true)
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

func (m Model) buildAuthViewHelpText() string {
	hb := NewHelpBuilder(m.configManager)
	return hb.Build(KeyItem{"settings", "F1", "Settings"}) + " • " + hb.Pair("nav_prev_tab", "Shift+Tab", "nav_next_tab", "Tab", "Navigate") + " • " + hb.Build(
		KeyItem{"select", "Enter", "Submit"},
		KeyItem{"quit", "Ctrl+C", "Quit"},
	)
}

func (m Model) registerView() string {
	header := titleStyle.Render("CardMan - Register")
	body := m.renderRegisterBody()
	footer := m.renderRegisterFooter()
	return renderFramedView(header, body, footer, m.width, m.height, m.styleManager)
}

func (m Model) renderRegisterBody() string {
	var b strings.Builder
	b.WriteString(RenderConditionalLabel(true, "First Name:") + "\n")
	b.WriteString(m.inputs[0].View() + "\n")
	b.WriteString(RenderConditionalLabel(false, "Last Name:") + "\n")
	b.WriteString(m.inputs[1].View() + "\n")
	b.WriteString(RenderConditionalLabel(false, "Email:") + "\n")
	b.WriteString(m.inputs[2].View() + "\n")
	b.WriteString(RenderConditionalLabel(false, "Password (8+ chars, 1 uppercase, 1 special):") + "\n")
	b.WriteString(m.inputs[3].View())
	if m.errorMsg != "" {
		b.WriteString("\n" + errorStyle.Render("Error: "+m.errorMsg))
	}
	b.WriteString("\n" + RenderButton(m.focusIndex == len(m.inputs), "[ Register ]"))
	return b.String()
}

func (m Model) renderRegisterFooter() string {
	var b strings.Builder
	b.WriteString(helpStyle.Render("Already have an account? Press Enter on the button below") + "\n")
	b.WriteString(RenderButton(m.focusIndex == len(m.inputs)+1, "[ Back to Login ]") + "\n")
	b.WriteString(helpStyle.Render(m.buildAuthViewHelpText()))
	return b.String()
}
