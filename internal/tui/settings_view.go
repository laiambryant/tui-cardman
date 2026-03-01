package tui

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
)

type (
	saveConfirmedMsg struct{}
	saveCancelledMsg struct{}
)

type SettingsModel struct {
	configManager  *runtimecfg.Manager
	styleManager   *StyleManager
	originalConfig *runtimecfg.RuntimeConfig
	tempConfig     *runtimecfg.RuntimeConfig
	hasChanges     bool
	modal          ModalModel
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
func NewSettingsModel(configManager *runtimecfg.Manager, styleManager *StyleManager) *SettingsModel {
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
		styleManager:   styleManager,
		originalConfig: cfg,
		tempConfig:     tempConfig,
		hasChanges:     false,
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
		styleManager,
	)
	model.modal = model.modal.Hide()
	return model
}

func (m SettingsModel) Init() tea.Cmd {
	return nil
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
		m.modal = m.modal.SetDimensions(sizeMsg.Width, sizeMsg.Height)
		return m, nil
	}
	if m.modal.IsVisible() {
		var cmd tea.Cmd
		m.modal, cmd = m.modal.Update(msg)
		return m, cmd
	}
	switch msg := msg.(type) {
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
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

func (m SettingsModel) handleKeyMsg(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	s := msg.String()
	action := GetAction(m.configManager, s)
	if m.editing {
		return m.handleEditingKey(action, s)
	}
	return m.handleNormalKey(action, s)
}

func (m SettingsModel) handleEditingKey(action, s string) (SettingsModel, tea.Cmd) {
	if action == "quit_alt" || action == "back" {
		m.stopEditing()
		return m, nil
	}
	if m.tempConfig.Keybindings == nil {
		m.tempConfig.Keybindings = make(map[string]string)
	}
	m.tempConfig.Keybindings[m.editingAction] = s
	m.hasChanges = true
	m.stopEditing()
	return m, nil
}

func (m SettingsModel) handleNormalKey(action, s string) (SettingsModel, tea.Cmd) {
	switch {
	case (action == "back" || action == "quit_alt") && !m.hasChanges:
		m.shouldClose = true
		return m, nil
	case s == "ctrl+s" && m.hasChanges:
		m.initiateSave()
		return m, nil
	case action == "back" || action == "quit_alt":
		m.initiateSave()
		return m, nil
	case action == "nav_up" || s == "k" || s == "up":
		m.navigateUp()
		return m, nil
	case action == "nav_down" || s == "j" || s == "down":
		m.navigateDown()
		return m, nil
	case action == "select":
		m.handleSelectKey()
		return m, nil
	}
	return m, nil
}

func (m *SettingsModel) navigateUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *SettingsModel) navigateDown() {
	maxCursor := len(m.actions) - 1
	if m.cursor < maxCursor {
		m.cursor++
	}
}

func (m *SettingsModel) stopEditing() {
	m.editing = false
	m.editingAction = ""
	m.errorMsg = ""
}

func (m *SettingsModel) handleSelectKey() {
	if m.cursor < len(m.actions) {
		m.editing = true
		m.editingAction = m.actions[m.cursor]
		m.errorMsg = ""
	}
}

func (m SettingsModel) View() string {
	header := m.renderSettingsHeader()
	footer := m.renderSettingsFooter()
	return RenderFramedWithModal(header, footer, m.renderSettingsBody, m.width, m.height, m.styleManager, &m.modal)
}

func (m SettingsModel) renderSettingsHeader() string {
	var b strings.Builder
	title := "Settings"
	if m.hasChanges {
		title += " *"
	}
	b.WriteString(m.styleManager.GetTitleStyle().Render(title) + "\n")
	b.WriteString(RenderTabBar(m.styleManager, []string{"Keybindings"}, 0))
	return b.String()
}

func (m SettingsModel) renderSettingsBody(maxLines int) string {
	var b strings.Builder
	availableLines := maxLines
	if m.errorMsg != "" {
		b.WriteString(m.styleManager.GetErrorStyle().Render("Error: "+m.errorMsg) + "\n")
		availableLines--
	}
	if availableLines < 1 {
		availableLines = 1
	}
	b.WriteString(m.renderKeybindingsSection(availableLines))
	return b.String()
}

func (m SettingsModel) renderSettingsFooter() string {
	if m.editing {
		cancelKey := "Esc"
		if m.configManager != nil {
			if k := m.configManager.KeyForAction("quit_alt"); k != "" {
				cancelKey = k
			}
		}
		return m.styleManager.GetHelpStyle().Render(fmt.Sprintf("Press any key to bind (%s to cancel)", cancelKey))
	}
	help := m.buildHelpText()
	return m.styleManager.GetHelpStyle().Render(help)
}

func (m SettingsModel) renderKeybindingsSection(maxLines int) string {
	var b strings.Builder
	if m.editing {
		b.WriteString(m.styleManager.GetTitleStyle().Render(fmt.Sprintf("Editing: %s", m.editingAction)) + "\n")
		b.WriteString(m.styleManager.GetBlurredStyle().Render("Press the key you want to bind...") + "\n")
		return b.String()
	}
	cfg := m.tempConfig
	headerSpace := m.styleManager.GetNoStyle().Render(strings.Repeat(" ", 25))
	b.WriteString(m.styleManager.GetTitleStyle().Render("Action") + headerSpace + m.styleManager.GetTitleStyle().Render("Key") + "\n")
	separator := m.styleManager.GetNoStyle().Render(strings.Repeat("─", 50))
	b.WriteString(separator + "\n")
	visibleStart := 0
	visibleEnd := len(m.actions)
	maxVisible := 15
	if maxLines > 0 {
		maxVisible = maxLines - 2
		if len(m.actions) > maxVisible {
			maxVisible--
		}
		if maxVisible < 1 {
			maxVisible = 1
		}
	}
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
			b.WriteString(m.styleManager.GetSettingsSelectedStyle().Render("> "+actionDisplay+" "+keyDisplay) + "\n")
		} else {
			separator := m.styleManager.GetNoStyle().Render(" ")
			b.WriteString(m.styleManager.GetBlurredStyle().Render("  "+actionDisplay) + separator + m.styleManager.GetNoStyle().Render(keyDisplay) + "\n")
		}
	}
	if len(m.actions) > maxVisible {
		b.WriteString(m.styleManager.GetBlurredStyle().Render(fmt.Sprintf("\n  [Showing %d-%d of %d]", visibleStart+1, visibleEnd, len(m.actions))) + "\n")
	}
	return b.String()
}

func (m SettingsModel) buildHelpText() string {
	hb := NewHelpBuilder(m.configManager)
	help := hb.Build(KeyItem{"settings", "F1", "Settings"}) + " • " + hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate") + " • " + hb.Build(
		KeyItem{"select", "Enter", "Edit"},
		KeyItem{"quit_alt", "Esc", "Close"},
	)
	if m.hasChanges {
		help += " • Ctrl+S: Save"
	}
	return help
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
	if err := m.configManager.Set(m.tempConfig); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	m.originalConfig = copyConfig(m.tempConfig)
	m.hasChanges = false
	return nil
}

func (m *SettingsModel) cancelChanges() {
	m.tempConfig = copyConfig(m.originalConfig)
	m.hasChanges = false
}
