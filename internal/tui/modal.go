package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
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
	styleManager      *StyleManager
}

func NewModalModel(title, message string, onConfirm, onCancel func() tea.Cmd, styleManager *StyleManager) ModalModel {
	return ModalModel{
		title:        title,
		message:      message,
		onConfirm:    onConfirm,
		onCancel:     onCancel,
		confirmed:    false,
		selected:     0,
		visible:      true,
		styleManager: styleManager,
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
		return m.handleKey(msg.String())
	}
	return m, nil
}

func (m ModalModel) handleKey(key string) (ModalModel, tea.Cmd) {
	switch key {
	case "left", "h":
		m.selected = 0
		return m, nil
	case "right", "l":
		m.selected = 1
		return m, nil
	case "enter":
		return m.handleConfirm()
	case "esc", "q":
		return m.handleCancel()
	}
	return m, nil
}

func (m ModalModel) handleConfirm() (ModalModel, tea.Cmd) {
	m.visible = false
	if m.selected == 1 {
		m.confirmed = true
		if m.onConfirm != nil {
			return m, m.onConfirm()
		}
	} else {
		if m.onCancel != nil {
			return m, m.onCancel()
		}
	}
	return m, nil
}

func (m ModalModel) handleCancel() (ModalModel, tea.Cmd) {
	m.visible = false
	if m.onCancel != nil {
		return m, m.onCancel()
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

func (m ModalModel) createButtonStyle(focused bool) lipgloss.Style {
	style := m.styleManager.GetFocusedStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styleManager.scheme.Focused).
		Padding(0, 2)
	if focused {
		style = style.Bold(true)
	} else {
		style = m.styleManager.GetBlurredStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.styleManager.scheme.Blurred).
			Padding(0, 2)
	}
	return style
}

func (m ModalModel) renderModalBox() string {
	var b strings.Builder
	modalStyle := m.styleManager.GetModalStyle()
	if m.styleManager != nil {
		modalStyle = modalStyle.Foreground(m.styleManager.scheme.Foreground)
	}
	titleStyle := m.styleManager.GetTitleStyle().Align(lipgloss.Center)
	messageStyle := m.styleManager.GetBlurredStyle().Align(lipgloss.Center)
	b.WriteString(titleStyle.Render(m.title) + "\n\n")
	b.WriteString(messageStyle.Render(m.message) + "\n\n")
	b.WriteString(m.renderButtons())
	return modalStyle.Render(b.String())
}

func (m ModalModel) renderButtons() string {
	noStyle := m.createButtonStyle(m.selected == 0)
	yesStyle := m.createButtonStyle(m.selected == 1)
	noButton := noStyle.Render("No")
	yesButton := yesStyle.Render("Yes")
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, noButton, "  ", yesButton)
	return lipgloss.NewStyle().Align(lipgloss.Center).Render(buttons)
}

func (m ModalModel) renderOverlay() string {
	if m.backgroundContent == "" {
		return ""
	}
	return m.backgroundContent
}

func (m ModalModel) overlayContent(background, foreground string) string {
	if background == "" {
		return foreground
	}
	if foreground == "" {
		return background
	}
	width := maxLineWidth(background)
	fgWidth := maxLineWidth(foreground)
	if fgWidth > width {
		width = fgWidth
	}
	height := lineCount(background)
	fgHeight := lineCount(foreground)
	if fgHeight > height {
		height = fgHeight
	}
	if width == 0 || height == 0 {
		return background
	}
	bgBuf := cellbuf.NewBuffer(width, height)
	cellbuf.SetContent(bgBuf, background)
	fgBuf := cellbuf.NewBuffer(width, height)
	cellbuf.SetContent(fgBuf, foreground)
	blank := cellbuf.BlankCell
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell := fgBuf.Cell(x, y)
			if cell == nil || cell.Width == 0 {
				continue
			}
			if cell.Equal(&blank) {
				continue
			}
			bgBuf.SetCell(x, y, cell)
		}
	}
	return strings.ReplaceAll(cellbuf.Render(bgBuf), "\r\n", "\n")
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

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func maxLineWidth(s string) int {
	if s == "" {
		return 0
	}
	lines := strings.Split(s, "\n")
	maxWidth := 0
	for _, line := range lines {
		width := lipgloss.Width(line)
		if width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth
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
