package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
)

type settingsSection int

const (
	sectionKeybindings settingsSection = iota
	sectionUI
)

// SettingsModel represents the settings view
type SettingsModel struct {
	configManager *runtimecfg.Manager
	section       settingsSection
	cursor        int
	actions       []string // Sorted list of actions for keybinding section
	editing       bool
	editingAction string
	input         textinput.Model
	errorMsg      string
	shouldClose   bool
}

// NewSettingsModel creates a new settings model
func NewSettingsModel(configManager *runtimecfg.Manager) *SettingsModel {
	cfg := configManager.Get()
	actions := make([]string, 0, len(cfg.Keybindings))
	for action := range cfg.Keybindings {
		actions = append(actions, action)
	}
	sort.Strings(actions)

	input := textinput.New()
	input.Placeholder = "Press a key..."
	input.Width = 30

	return &SettingsModel{
		configManager: configManager,
		section:       sectionKeybindings,
		actions:       actions,
		input:         input,
		cursor:        0,
	}
}

func (m SettingsModel) Init() tea.Cmd {
	return nil
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editing {
			s := msg.String()
			action := ""
			if m.configManager != nil {
				action = m.configManager.MatchAction(s)
			}
			if action == "quit_alt" || action == "back" {
				m.editing = false
				m.editingAction = ""
				m.errorMsg = ""
				return m, nil
			}
			err := m.configManager.SetKeybinding(m.editingAction, s)
			if err != nil {
				m.errorMsg = err.Error()
				return m, nil
			}
			cfg := m.configManager.Get()
			err = m.configManager.Set(cfg)
			if err != nil {
				m.errorMsg = fmt.Sprintf("Failed to save: %v", err)
				return m, nil
			}
			m.editing = false
			m.editingAction = ""
			m.errorMsg = ""
			return m, nil
		}
		s := msg.String()
		action := ""
		if m.configManager != nil {
			action = m.configManager.MatchAction(s)
		}
		if action == "back" || action == "quit_alt" {
			m.shouldClose = true
			return m, nil
		}
		if action == "nav_up" || s == "k" {
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		}
		if action == "nav_down" || s == "j" {
			maxCursor := len(m.actions) - 1
			if m.cursor < maxCursor {
				m.cursor++
			}
			return m, nil
		}
		if action == "nav_left" || s == "h" {
			if m.section > 0 {
				m.section--
				m.cursor = 0
			}
			return m, nil
		}
		if action == "nav_right" || s == "l" {
			if m.section < sectionUI {
				m.section++
				m.cursor = 0
			}
			return m, nil
		}
		if action == "select" {
			if m.section == sectionKeybindings && m.cursor < len(m.actions) {
				m.editing = true
				m.editingAction = m.actions[m.cursor]
				m.errorMsg = ""
			}
			return m, nil
		}
	}

	return m, nil
}

func (m SettingsModel) View() string {
	var b strings.Builder
	b.WriteString(settingsTitleStyle.Render("⚙️  Settings") + "\n\n")
	tabs := []string{"Keybindings", "UI"}
	var renderedTabs []string
	for i, tab := range tabs {
		if settingsSection(i) == m.section {
			renderedTabs = append(renderedTabs, settingsFocusStyle.Render("[ "+tab+" ]"))
		} else {
			renderedTabs = append(renderedTabs, settingsBlurStyle.Render("  "+tab+"  "))
		}
	}
	b.WriteString(strings.Join(renderedTabs, " ") + "\n\n")
	if m.errorMsg != "" {
		b.WriteString(settingsErrorStyle.Render("⚠ "+m.errorMsg) + "\n\n")
	}
	switch m.section {
	case sectionKeybindings:
		b.WriteString(m.renderKeybindingsSection())
	case sectionUI:
		b.WriteString(m.renderUISection())
	}
	b.WriteString("\n")
	if m.editing {
		cancelKey := "Esc"
		if m.configManager != nil {
			if k := m.configManager.KeyForAction("quit_alt"); k != "" {
				cancelKey = k
			}
		}
		b.WriteString(helpStyle.Render(fmt.Sprintf("Press any key to bind (%s to cancel)", cancelKey)) + "\n")
	} else {
		settingsKey := "F1"
		navUp := "↑"
		navDown := "↓"
		navLeft := "←"
		navRight := "→"
		editKey := "Enter"
		closeKey := "Esc"
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
			if k := m.configManager.KeyForAction("nav_left"); k != "" {
				navLeft = k
			}
			if k := m.configManager.KeyForAction("nav_right"); k != "" {
				navRight = k
			}
			if k := m.configManager.KeyForAction("select"); k != "" {
				editKey = k
			}
			if k := m.configManager.KeyForAction("quit_alt"); k != "" {
				closeKey = k
			}
		}
		help := fmt.Sprintf("%s: Settings • %s/%s: Navigate • %s/%s: Switch sections • %s: Edit • %s: Close", settingsKey, navUp, navDown, navLeft, navRight, editKey, closeKey)
		b.WriteString(helpStyle.Render(help) + "\n")
	}
	return b.String()
}

func (m SettingsModel) renderKeybindingsSection() string {
	var b strings.Builder

	if m.editing {
		b.WriteString(settingsFocusStyle.Render(fmt.Sprintf("Editing: %s", m.editingAction)) + "\n")
		b.WriteString(settingsBlurStyle.Render("Press the key you want to bind...") + "\n")
		return b.String()
	}
	cfg := m.configManager.Get()
	b.WriteString(settingsFocusStyle.Render("Action") + strings.Repeat(" ", 25) + settingsFocusStyle.Render("Key") + "\n")
	b.WriteString(strings.Repeat("─", 50) + "\n")
	visibleStart := 0
	visibleEnd := len(m.actions)
	maxVisible := 15
	if len(m.actions) > maxVisible {
		visibleStart = max(m.cursor-maxVisible/2, 0)
		visibleEnd = visibleStart + maxVisible
		if visibleEnd > len(m.actions) {
			visibleEnd = len(m.actions)
			visibleStart = max(visibleEnd-maxVisible, 0)
		}
	}
	for i := visibleStart; i < visibleEnd; i++ {
		action := m.actions[i]
		key := cfg.Keybindings[action]
		actionDisplay := action
		if len(actionDisplay) > 25 {
			actionDisplay = actionDisplay[:22] + "..."
		} else {
			actionDisplay = actionDisplay + strings.Repeat(" ", 25-len(actionDisplay))
		}
		keyDisplay := key
		if keyDisplay == "" {
			keyDisplay = "<unbound>"
		}
		if i == m.cursor {
			b.WriteString(settingsFocusStyle.Render("> "+actionDisplay) + " " + settingsFocusStyle.Render(keyDisplay) + "\n")
		} else {
			b.WriteString(settingsBlurStyle.Render("  "+actionDisplay) + " " + noStyle.Render(keyDisplay) + "\n")
		}
	}
	if len(m.actions) > maxVisible {
		b.WriteString(blurredStyle.Render(fmt.Sprintf("\n  [Showing %d-%d of %d]", visibleStart+1, visibleEnd, len(m.actions))) + "\n")
	}
	return b.String()
}

func (m SettingsModel) renderUISection() string {
	var b strings.Builder
	cfg := m.configManager.Get()
	b.WriteString(settingsFocusStyle.Render("UI Settings") + "\n\n")
	b.WriteString(fmt.Sprintf("Compact Lists: %v\n", cfg.UI.CompactLists))
	b.WriteString(fmt.Sprintf("Color Scheme: %s\n", cfg.UI.ColorScheme))
	b.WriteString("\n")
	b.WriteString(settingsBlurStyle.Render("(UI settings editing coming soon)") + "\n")
	return b.String()
}

var (
	settingsTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	settingsFocusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	settingsBlurStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	settingsErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)
