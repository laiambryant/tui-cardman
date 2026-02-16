package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
)

type CardGameMenuOption int

const (
	MenuMyCollection CardGameMenuOption = iota
	MenuMyLists
	menuOptionCount
)

type CardGameMenuModel struct {
	selectedGame   *model.CardGame
	cursor         int
	styleManager   *StyleManager
	configManager  *runtimecfg.Manager
	width          int
	height         int
	shouldGoBack   bool
	selectedOption CardGameMenuOption
	optionChosen   bool
}

func NewCardGameMenuModel(game *model.CardGame, sm *StyleManager, cfg *runtimecfg.Manager) *CardGameMenuModel {
	return &CardGameMenuModel{
		selectedGame:  game,
		cursor:        0,
		styleManager:  sm,
		configManager: cfg,
	}
}

func (m CardGameMenuModel) Init() tea.Cmd {
	return nil
}

func (m CardGameMenuModel) Update(msg tea.Msg) (CardGameMenuModel, tea.Cmd) {
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		action := MatchActionOrDefault(m.configManager, s, "")
		if action == "quit" || s == "ctrl+c" {
			return m, tea.Quit
		}
		if isBackKey(action, s) {
			m.shouldGoBack = true
			return m, nil
		}
		if action == "nav_up" || s == "up" || s == "k" {
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		}
		if action == "nav_down" || s == "down" || s == "j" || s == "tab" {
			if m.cursor < int(menuOptionCount)-1 {
				m.cursor++
			}
			return m, nil
		}
		if isSelectKey(action, s) {
			m.selectedOption = CardGameMenuOption(m.cursor)
			m.optionChosen = true
			return m, nil
		}
	}
	return m, nil
}

func (m CardGameMenuModel) View() string {
	header := m.renderHeader()
	footer := m.renderFooter()
	return RenderFramedWithModal(header, footer, m.renderBody, m.width, m.height, m.styleManager, nil)
}

func (m CardGameMenuModel) renderHeader() string {
	if m.selectedGame != nil {
		return m.styleManager.GetTitleStyle().Render(m.selectedGame.Name)
	}
	return m.styleManager.GetTitleStyle().Render("Select Mode")
}

func (m CardGameMenuModel) renderBody(availableHeight int) string {
	var b strings.Builder
	b.WriteString(m.styleManager.GetTitleStyle().Render("Select Mode") + "\n\n")
	b.WriteString(RenderListItem("My Collection", m.cursor == 0))
	b.WriteString(RenderListItem("My Lists", m.cursor == 1))
	return b.String()
}

func (m CardGameMenuModel) renderFooter() string {
	hb := NewHelpBuilder(m.configManager)
	return m.styleManager.GetHelpStyle().Render(
		hb.Pair("nav_up", "↑", "nav_down", "↓", "Navigate") + " • " + hb.Build(
			KeyItem{"select", "Enter", "Select"},
			KeyItem{"back", "Q", "Back"},
		),
	)
}
