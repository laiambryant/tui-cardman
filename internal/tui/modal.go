package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ModalModel struct {
	title     string
	message   string
	onConfirm func() tea.Cmd
	onCancel  func() tea.Cmd
	confirmed bool
	selected  int
	visible   bool
}

func NewModalModel(title, message string, onConfirm, onCancel func() tea.Cmd) ModalModel {
	return ModalModel{
		title:     title,
		message:   message,
		onConfirm: onConfirm,
		onCancel:  onCancel,
		confirmed: false,
		selected:  0,
		visible:   true,
	}
}

func (m ModalModel) Init() tea.Cmd {
	return nil
}

func (m ModalModel) Update(msg tea.Msg) (ModalModel, tea.Cmd) {
	if !m.visible {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			m.selected = 0
			return m, nil
		case "right", "l":
			m.selected = 1
			return m, nil
		case "enter":
			if m.selected == 1 {
				m.confirmed = true
				m.visible = false
				if m.onConfirm != nil {
					return m, m.onConfirm()
				}
			} else {
				m.visible = false
				if m.onCancel != nil {
					return m, m.onCancel()
				}
			}
			return m, nil
		case "esc", "q":
			m.visible = false
			if m.onCancel != nil {
				return m, m.onCancel()
			}
			return m, nil
		}
	}
	return m, nil
}

func (m ModalModel) View() string {
	if !m.visible {
		return ""
	}
	var b strings.Builder
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("170")).
		Padding(1, 2).
		Width(50)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		Align(lipgloss.Center)
	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Align(lipgloss.Center)
	buttonNoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 2)
	buttonYesStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 2)
	buttonNoFocusedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 2)
	buttonYesFocusedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 2)
	b.WriteString(titleStyle.Render(m.title) + "\n\n")
	b.WriteString(messageStyle.Render(m.message) + "\n\n")
	noButton := "No"
	yesButton := "Yes"
	if m.selected == 0 {
		noButton = buttonNoFocusedStyle.Render(noButton)
		yesButton = buttonYesStyle.Render(yesButton)
	} else {
		noButton = buttonNoStyle.Render(noButton)
		yesButton = buttonYesFocusedStyle.Render(yesButton)
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, noButton, "  ", yesButton)
	b.WriteString(lipgloss.NewStyle().Align(lipgloss.Center).Render(buttons))
	return modalStyle.Render(b.String())
}

func (m ModalModel) IsVisible() bool {
	return m.visible
}

func (m ModalModel) Hide() ModalModel {
	m.visible = false
	return m
}

func (m ModalModel) Show() ModalModel {
	m.visible = true
	m.selected = 0
	return m
}

func (m ModalModel) SetMessage(title, message string) ModalModel {
	m.title = title
	m.message = message
	return m
}
