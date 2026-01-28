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
			if m.section == sectionUI {
				if m.uiCursor > 0 {
					m.uiCursor--
				}
			} else {
				if m.cursor > 0 {
					m.cursor--
				}
			}
			return m, nil
		}
		if action == "nav_down" || s == "j" {
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
			return m, nil
		}
		if action == "nav_left" || s == "h" {
			if m.section == sectionUI {
				m.handleUISettingChange(-1)
			} else {
				if m.section > 0 {
					m.section--
					m.cursor = 0
					m.uiCursor = 0
				}
			}
			return m, nil
		}
		if action == "nav_right" || s == "l" {
			if m.section == sectionUI {
				m.handleUISettingChange(1)
			} else {
				if m.section < sectionUI {
					m.section++
					m.cursor = 0
					m.uiCursor = 0
				}
			}
			return m, nil
		}
		if action == "select" {
			if m.section == sectionKeybindings && m.cursor < len(m.actions) {
				m.editing = true
				m.editingAction = m.actions[m.cursor]
				m.errorMsg = ""
			} else if m.section == sectionUI {
				m.handleUISettingChange(1)
			}
			return m, nil
		}
	}

	return m, nil
}

func (m *SettingsModel) handleUISettingChange(direction int) {
	switch m.uiCursor {
	case uiSettingTheme:
		themes := runtimecfg.GetColorSchemeNames()
		if len(themes) == 0 {
			return
		}
		currentIndex := -1
		for i, theme := range themes {
			if theme == m.tempConfig.UI.ColorScheme {
				currentIndex = i
				break
			}
		}
		if currentIndex == -1 {
			currentIndex = 0
		}
		newIndex := (currentIndex + direction + len(themes)) % len(themes)
		m.tempConfig.UI.ColorScheme = themes[newIndex]
		m.hasChanges = true
	case uiSettingOpaqueBackground:
		m.tempConfig.UI.OpaqueBackground = !m.tempConfig.UI.OpaqueBackground
		m.hasChanges = true
	case uiSettingBackgroundStyle:
		if !m.tempConfig.UI.OpaqueBackground {
			return
		}
		styles := []string{"none", "components", "full", "both"}
		currentIndex := -1
		for i, style := range styles {
			if style == m.tempConfig.UI.BackgroundStyle {
				currentIndex = i
				break
			}
		}
		if currentIndex == -1 {
			currentIndex = 1
		}
		newIndex := (currentIndex + direction + len(styles)) % len(styles)
		m.tempConfig.UI.BackgroundStyle = styles[newIndex]
		m.hasChanges = true
	}
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

func (m SettingsModel) renderUISection() string {
	var b strings.Builder
	cfg := m.tempConfig
	b.WriteString(m.styleManager.GetTitleStyle().Render("UI Settings") + "\n\n")
	themes := runtimecfg.GetColorSchemeNames()
	bgStyles := []string{"none", "components", "full", "both"}
	cursor := "→ "
	blank := "  "
	themeDisplay := cursor
	if m.uiCursor != uiSettingTheme {
		themeDisplay = blank
	}
	line := themeDisplay + "Theme: " + cfg.UI.ColorScheme
	if m.uiCursor == uiSettingTheme {
		b.WriteString(m.styleManager.GetSettingsSelectedStyle().Render(line) + "\n")
	} else {
		b.WriteString(line + "\n")
	}
	bgDisplay := cursor
	if m.uiCursor != uiSettingOpaqueBackground {
		bgDisplay = blank
	}
	bgValue := "Off"
	if cfg.UI.OpaqueBackground {
		bgValue = "On"
	}
	line = bgDisplay + "Opaque Background: " + bgValue
	if m.uiCursor == uiSettingOpaqueBackground {
		b.WriteString(m.styleManager.GetSettingsSelectedStyle().Render(line) + "\n")
	} else {
		b.WriteString(line + "\n")
	}
	bgStyleDisplay := cursor
	if m.uiCursor != uiSettingBackgroundStyle {
		bgStyleDisplay = blank
	}
	styleValue := cfg.UI.BackgroundStyle
	if !cfg.UI.OpaqueBackground {
		styleValue = m.styleManager.GetBlurredStyle().Render(styleValue)
	}
	line = bgStyleDisplay + "Background Style: " + styleValue
	if m.uiCursor == uiSettingBackgroundStyle {
		b.WriteString(m.styleManager.GetSettingsSelectedStyle().Render(line) + "\n")
	} else {
		b.WriteString(line + "\n")
	}
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
