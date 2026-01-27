package tui

import (
	"fmt"
	"maps"
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

type saveConfirmedMsg struct{}
type saveCancelledMsg struct{}

type SettingsModel struct {
	configManager  *runtimecfg.Manager
	originalConfig *runtimecfg.RuntimeConfig
	tempConfig     *runtimecfg.RuntimeConfig
	hasChanges     bool
	modal          ModalModel
	section        settingsSection
	cursor         int
	actions        []string
	editing        bool
	editingAction  string
	input          textinput.Model
	errorMsg       string
	shouldClose    bool
	width          int
	height         int
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
	tempConfig := copyConfig(cfg)
	model := &SettingsModel{
		configManager:  configManager,
		originalConfig: cfg,
		tempConfig:     tempConfig,
		hasChanges:     false,
		section:        sectionKeybindings,
		actions:        actions,
		input:          input,
		cursor:         0,
	}
	model.modal = NewModalModel(
		"Save Changes?",
		"You have unsaved changes. Save before closing?",
		func() tea.Cmd {
			return func() tea.Msg {
				return saveConfirmedMsg{}
			}
		},
		func() tea.Cmd {
			return func() tea.Msg {
				return saveCancelledMsg{}
			}
		},
	)
	model.modal = model.modal.Hide()
	return model
}

func (m SettingsModel) Init() tea.Cmd {
	return nil
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	if m.modal.IsVisible() {
		var cmd tea.Cmd
		m.modal, cmd = m.modal.Update(msg)
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.modal = m.modal.SetDimensions(msg.Width, msg.Height)
		return m, nil
	case saveConfirmedMsg:
		err := m.confirmSaveChanges()
		if err != nil {
			m.errorMsg = fmt.Sprintf("Failed to save: %v", err)
			return m, nil
		}
		m.shouldClose = true
		return m, nil
	case saveCancelledMsg:
		m.cancelChanges()
		m.shouldClose = true
		return m, nil
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
			if m.tempConfig.Keybindings == nil {
				m.tempConfig.Keybindings = make(map[string]string)
			}
			m.tempConfig.Keybindings[m.editingAction] = s
			m.hasChanges = true
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
		if (action == "back" || action == "quit_alt") && !m.hasChanges {
			m.shouldClose = true
			return m, nil
		}
		if s == "ctrl+s" && m.hasChanges {
			m.initiateSave()
			return m, nil
		}
		if action == "back" || action == "quit_alt" {
			m.initiateSave()
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
	title := "⚙️  Settings"
	if m.hasChanges {
		title += " *"
	}
	b.WriteString(settingsTitleStyle.Render(title) + "\n\n")
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
		if m.hasChanges {
			help += " • Ctrl+S: Save"
		}
		b.WriteString(helpStyle.Render(help) + "\n")
	}
	content := b.String()
	if m.modal.IsVisible() {
		m.modal = m.modal.SetBackgroundContent(content)
		return m.modal.View()
	}
	return content
}

func (m SettingsModel) renderKeybindingsSection() string {
	var b strings.Builder

	if m.editing {
		b.WriteString(settingsFocusStyle.Render(fmt.Sprintf("Editing: %s", m.editingAction)) + "\n")
		b.WriteString(settingsBlurStyle.Render("Press the key you want to bind...") + "\n")
		return b.String()
	}
	cfg := m.tempConfig
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
	cfg := m.tempConfig
	b.WriteString(settingsFocusStyle.Render("UI Settings") + "\n\n")
	b.WriteString(fmt.Sprintf("Compact Lists: %v\n", cfg.UI.CompactLists))
	b.WriteString(fmt.Sprintf("Color Scheme: %s\n", cfg.UI.ColorScheme))
	b.WriteString("\n")
	b.WriteString(settingsBlurStyle.Render("(UI settings editing coming soon)") + "\n")
	return b.String()
}

// copyConfig creates a deep copy of a RuntimeConfig
func copyConfig(cfg *runtimecfg.RuntimeConfig) *runtimecfg.RuntimeConfig {
	if cfg == nil {
		return nil
	}
	copy := &runtimecfg.RuntimeConfig{
		UI: cfg.UI,
	}
	if cfg.Keybindings != nil {
		copy.Keybindings = make(map[string]string)
		maps.Copy(copy.Keybindings, cfg.Keybindings)
	}
	return copy
}

func (m *SettingsModel) initiateSave() {
	if m.hasChanges {
		m.modal = m.modal.Show()
	} else {
		m.shouldClose = true
	}
}

func (m *SettingsModel) confirmSaveChanges() error {
	err := m.configManager.Set(m.tempConfig)
	if err != nil {
		return &FailedToSaveConfigurationError{Err: err}
	}
	m.originalConfig = copyConfig(m.tempConfig)
	m.hasChanges = false
	return nil
}

func (m *SettingsModel) cancelChanges() {
	m.tempConfig = copyConfig(m.originalConfig)
	m.hasChanges = false
}

var (
	settingsTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	settingsFocusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	settingsBlurStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	settingsErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)
