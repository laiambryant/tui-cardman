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

type settingsSection int

const (
	sectionKeybindings settingsSection = iota
	sectionUI
)

const (
	uiSettingTheme = iota
	uiSettingOpaqueBackground
	uiSettingBackgroundStyle
)

type saveConfirmedMsg struct{}
type saveCancelledMsg struct{}

type SettingsModel struct {
	configManager  *runtimecfg.Manager
	styleManager   *StyleManager
	originalConfig *runtimecfg.RuntimeConfig
	tempConfig     *runtimecfg.RuntimeConfig
	hasChanges     bool
	modal          ModalModel
	section        settingsSection
	cursor         int
	uiCursor       int
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
		section:        sectionKeybindings,
		actions:        actions,
		input:          input,
		cursor:         0,
		uiCursor:       0,
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

func (m SettingsModel) getAction(s string) string {
	return MatchActionOrDefault(m.configManager, s, "")
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
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

func (m SettingsModel) handleKeyMsg(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	s := msg.String()
	action := m.getAction(s)
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
	case action == "nav_left" || s == "h" || s == "left":
		m.navigateHorizontal(-1)
		return m, nil
	case action == "nav_right" || s == "l" || s == "right":
		m.navigateHorizontal(1)
		return m, nil
	case action == "select":
		m.handleSelectKey()
		return m, nil
	}
	return m, nil
}

func cyclicIndex(current, direction, length int) int {
	if length == 0 {
		return 0
	}
	return (current + direction + length) % length
}

func (m *SettingsModel) navigateUp() {
	if m.section == sectionUI {
		if m.uiCursor > 0 {
			m.uiCursor--
		}
	} else {
		if m.cursor > 0 {
			m.cursor--
		}
	}
}

func (m *SettingsModel) navigateDown() {
	if m.section == sectionUI {
		if m.uiCursor < 2 {
			m.uiCursor++
		}
	} else {
		maxCursor := len(m.actions) - 1
		if m.cursor < maxCursor {
			m.cursor++
		}
	}
}

func (m *SettingsModel) navigateHorizontal(direction int) {
	if m.section == sectionUI {
		m.handleUISettingChange(direction)
	} else {
		if direction > 0 && m.section < sectionUI {
			m.section++
			m.resetCursors()
		} else if direction < 0 && m.section > 0 {
			m.section--
			m.resetCursors()
		}
	}
}

func (m *SettingsModel) resetCursors() {
	m.cursor = 0
	m.uiCursor = 0
}

func (m *SettingsModel) stopEditing() {
	m.editing = false
	m.editingAction = ""
	m.errorMsg = ""
}

func (m *SettingsModel) handleSelectKey() {
	if m.section == sectionKeybindings && m.cursor < len(m.actions) {
		m.editing = true
		m.editingAction = m.actions[m.cursor]
		m.errorMsg = ""
	} else if m.section == sectionUI {
		m.handleUISettingChange(1)
	}
}

func findCurrentIndex(items []string, current string) int {
	for i, item := range items {
		if item == current {
			return i
		}
	}
	return 0
}

func (m *SettingsModel) handleUISettingChange(direction int) {
	switch m.uiCursor {
	case uiSettingTheme:
		m.cycleTheme(direction)
	case uiSettingOpaqueBackground:
		m.toggleOpaqueBackground()
	case uiSettingBackgroundStyle:
		m.cycleBackgroundStyle(direction)
	}
}

func (m *SettingsModel) cycleTheme(direction int) {
	themes := runtimecfg.GetColorSchemeNames()
	if len(themes) == 0 {
		return
	}
	currentIndex := findCurrentIndex(themes, m.tempConfig.UI.ColorScheme)
	newIndex := cyclicIndex(currentIndex, direction, len(themes))
	m.tempConfig.UI.ColorScheme = themes[newIndex]
	m.hasChanges = true
}

func (m *SettingsModel) toggleOpaqueBackground() {
	m.tempConfig.UI.OpaqueBackground = !m.tempConfig.UI.OpaqueBackground
	m.hasChanges = true
}

func (m *SettingsModel) cycleBackgroundStyle(direction int) {
	if !m.tempConfig.UI.OpaqueBackground {
		return
	}
	styles := []string{"none", "components", "full", "both"}
	currentIndex := findCurrentIndex(styles, m.tempConfig.UI.BackgroundStyle)
	newIndex := cyclicIndex(currentIndex, direction, len(styles))
	m.tempConfig.UI.BackgroundStyle = styles[newIndex]
	m.hasChanges = true
}

func (m SettingsModel) View() string {
	var b strings.Builder
	title := "⚙️  Settings"
	if m.hasChanges {
		title += " *"
	}
	b.WriteString(m.styleManager.GetTitleStyle().Render(title) + "\n\n")
	tabs := []string{"Keybindings", "UI"}
	var renderedTabs []string
	for i, tab := range tabs {
		if settingsSection(i) == m.section {
			renderedTabs = append(renderedTabs, m.styleManager.GetTitleStyle().Render("[ "+tab+" ]"))
		} else {
			renderedTabs = append(renderedTabs, m.styleManager.GetBlurredStyle().Render("  "+tab+"  "))
		}
	}
	b.WriteString(strings.Join(renderedTabs, " ") + "\n\n")
	if m.errorMsg != "" {
		b.WriteString(m.styleManager.GetErrorStyle().Render("⚠ "+m.errorMsg) + "\n\n")
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
		b.WriteString(m.styleManager.GetHelpStyle().Render(fmt.Sprintf("Press any key to bind (%s to cancel)", cancelKey)) + "\n")
	} else {
		help := m.buildHelpText()
		b.WriteString(m.styleManager.GetHelpStyle().Render(help) + "\n")
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
		b.WriteString(m.styleManager.GetTitleStyle().Render(fmt.Sprintf("Editing: %s", m.editingAction)) + "\n")
		b.WriteString(m.styleManager.GetBlurredStyle().Render("Press the key you want to bind...") + "\n")
		return b.String()
	}
	cfg := m.tempConfig
	b.WriteString(m.styleManager.GetTitleStyle().Render("Action") + strings.Repeat(" ", 25) + m.styleManager.GetTitleStyle().Render("Key") + "\n")
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
			b.WriteString(m.styleManager.GetSettingsSelectedStyle().Render("> "+actionDisplay+" "+keyDisplay) + "\n")
		} else {
			b.WriteString(m.styleManager.GetBlurredStyle().Render("  "+actionDisplay) + " " + m.styleManager.GetNoStyle().Render(keyDisplay) + "\n")
		}
	}
	if len(m.actions) > maxVisible {
		b.WriteString(m.styleManager.GetBlurredStyle().Render(fmt.Sprintf("\n  [Showing %d-%d of %d]", visibleStart+1, visibleEnd, len(m.actions))) + "\n")
	}
	return b.String()
}

func (m SettingsModel) buildHelpText() string {
	settingsKey := ResolveKeyBinding(m.configManager, "settings", "F1")
	navUp := ResolveKeyBinding(m.configManager, "nav_up", "↑")
	navDown := ResolveKeyBinding(m.configManager, "nav_down", "↓")
	navLeft := ResolveKeyBinding(m.configManager, "nav_left", "←")
	navRight := ResolveKeyBinding(m.configManager, "nav_right", "→")
	editKey := ResolveKeyBinding(m.configManager, "select", "Enter")
	closeKey := ResolveKeyBinding(m.configManager, "quit_alt", "Esc")
	help := fmt.Sprintf("%s: Settings • %s/%s: Navigate • %s/%s: Switch sections • %s: Edit • %s: Close", settingsKey, navUp, navDown, navLeft, navRight, editKey, closeKey)
	if m.hasChanges {
		help += " • Ctrl+S: Save"
	}
	return help
}

func (m SettingsModel) renderUILine(isCursor bool, label, value string) string {
	prefix := "  "
	if isCursor {
		prefix = "→ "
	}
	line := prefix + label + value
	if isCursor {
		return m.styleManager.GetSettingsSelectedStyle().Render(line)
	}
	return line
}

func (m SettingsModel) renderUISection() string {
	var b strings.Builder
	cfg := m.tempConfig
	b.WriteString(m.styleManager.GetTitleStyle().Render("UI Settings") + "\n\n")
	themes := runtimecfg.GetColorSchemeNames()
	bgStyles := []string{"none", "components", "full", "both"}
	b.WriteString(m.renderUILine(m.uiCursor == uiSettingTheme, "Theme: ", cfg.UI.ColorScheme) + "\n")
	bgValue := "Off"
	if cfg.UI.OpaqueBackground {
		bgValue = "On"
	}
	b.WriteString(m.renderUILine(m.uiCursor == uiSettingOpaqueBackground, "Opaque Background: ", bgValue) + "\n")
	styleValue := cfg.UI.BackgroundStyle
	if !cfg.UI.OpaqueBackground {
		styleValue = m.styleManager.GetBlurredStyle().Render(styleValue)
	}
	b.WriteString(m.renderUILine(m.uiCursor == uiSettingBackgroundStyle, "Background Style: ", styleValue) + "\n")
	b.WriteString("\n")
	b.WriteString(m.styleManager.GetBlurredStyle().Render("Available themes: "+strings.Join(themes, ", ")) + "\n")
	b.WriteString(m.styleManager.GetBlurredStyle().Render("Background styles: "+strings.Join(bgStyles, ", ")) + "\n")
	b.WriteString("\n")
	b.WriteString(m.styleManager.GetHelpStyle().Render("↑/↓: Navigate • Enter/→/←: Change • Esc: Back") + "\n")
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
