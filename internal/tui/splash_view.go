package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m Model) splashView() string {
	// Use GetTitleStyle() but disable Inline for multi-line logo rendering
	logoStyle := m.styleManager.GetTitleStyle().Inline(false)
	logo := logoStyle.Render(Logo)

	// Use GetFullScreenStyle for centering with themed background
	fullStyle := m.styleManager.GetFullScreenStyle(m.width, m.height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	return fullStyle.Render(logo)
}
