package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/laiambryant/tui-cardman/internal/tui/art"
)

func (m Model) splashView() string {
	logo := art.RenderLogo(
		m.width, m.height,
		m.styleManager.GetTitleStyle(),
		m.styleManager.GetFocusedStyle(),
		m.styleManager.GetBlurredStyle(),
	)
	fullStyle := m.styleManager.GetFullScreenStyle(m.width, m.height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)
	return fullStyle.Render(logo)
}
