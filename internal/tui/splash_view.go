package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/laiambryant/tui-cardman/internal/tui/art"
)

func (m *Model) splashView() string {
	scheme := m.styleManager.scheme
	logo := art.RenderLogo(
		m.width, m.height,
		[]lipgloss.Color{scheme.Focused, scheme.Title, scheme.Secondary, scheme.Success},
		scheme.Subtle,
	)
	fullStyle := m.styleManager.GetFullScreenStyle(m.width, m.height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)
	return fullStyle.Render(logo)
}
