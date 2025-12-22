package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type m struct {
}

func (m m) Init() tea.Cmd { return nil }
func (m m) Update(msg tea.Msg) (model tea.Model, cmd tea.Cmd) {
	return
}
func (m m) View() string {
	return ""
}

func NewModel() (model tea.Model, err error) {
	return m{}, nil
}
