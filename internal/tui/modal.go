package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ModalModel struct {
	title             string
	message           string
	onConfirm         func() tea.Cmd
	onCancel          func() tea.Cmd
	confirmed         bool
	selected          int
	visible           bool
	width             int
	height            int
	backgroundContent string
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
	modalBox := m.renderModalBox()
	if m.width == 0 || m.height == 0 {
		return modalBox
	}
	overlay := m.renderOverlay()
	centered := lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modalBox,
	)
	return m.overlayContent(overlay, centered)
}

func (m ModalModel) renderModalBox() string {
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

func (m ModalModel) renderOverlay() string {
	if m.backgroundContent == "" {
		return ""
	}
	lines := strings.Split(m.backgroundContent, "\n")
	var overlayLines []string
	shadeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	for _, line := range lines {
		plainLine := stripANSI(line)
		shadedLine := strings.Repeat("░", len(plainLine))
		overlayLines = append(overlayLines, shadeStyle.Render(shadedLine))
	}
	return strings.Join(overlayLines, "\n")
}

func (m ModalModel) overlayContent(background, foreground string) string {
	bgLines := strings.Split(background, "\n")
	fgLines := strings.Split(foreground, "\n")
	maxLines := len(bgLines)
	if len(fgLines) > maxLines {
		maxLines = len(fgLines)
	}
	var result []string
	for i := 0; i < maxLines; i++ {
		var bgLine, fgLine string
		if i < len(bgLines) {
			bgLine = bgLines[i]
		}
		if i < len(fgLines) {
			fgLine = fgLines[i]
		}
		combinedLine := m.mergeLines(bgLine, fgLine)
		result = append(result, combinedLine)
	}
	return strings.Join(result, "\n")
}

func (m ModalModel) mergeLines(bg, fg string) string {
	if fg == "" {
		return bg
	}
	bgPlain := stripANSI(bg)
	fgPlain := stripANSI(fg)
	if len(fgPlain) == 0 {
		return bg
	}
	bgRunes := []rune(bgPlain)
	fgRunes := []rune(fgPlain)
	var result strings.Builder
	for i := 0; i < len(bgRunes); i++ {
		if i < len(fgRunes) && fgRunes[i] != ' ' {
			result.WriteRune(fgRunes[i])
		} else {
			result.WriteRune(bgRunes[i])
		}
	}
	return result.String()
}

func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
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

func (m ModalModel) SetDimensions(width, height int) ModalModel {
	m.width = width
	m.height = height
	return m
}

func (m ModalModel) SetBackgroundContent(content string) ModalModel {
	m.backgroundContent = content
	return m
}
