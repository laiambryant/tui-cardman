package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	frameBorderSize = 2
	framePaddingX   = 1
	framePaddingY   = 0
	headerBarWidth  = 1
	frameTopMargin  = 1
)

var headerAccentBorder = lipgloss.Border{Left: "▌"}

type FrameLayout struct {
	Width               int
	Height              int
	TopMargin           int
	HeaderHeight        int
	BodyHeight          int
	FooterHeight        int
	ContentWidth        int
	HeaderContentHeight int
	BodyContentHeight   int
	FooterContentHeight int
}

func renderFramedView(header, body, footer string, width, height int, styleManager *StyleManager) string {
	header = trimTrailingNewlines(header)
	body = trimTrailingNewlines(body)
	footer = trimTrailingNewlines(footer)
	layout := calculateFrameLayout(lipgloss.Height(header), lipgloss.Height(footer), width, height)
	return renderFramedViewWithLayout(header, body, footer, layout, styleManager)
}

func renderFramedViewWithLayout(header, body, footer string, layout FrameLayout, styleManager *StyleManager) string {
	header = trimTrailingNewlines(header)
	body = trimTrailingNewlines(body)
	footer = trimTrailingNewlines(footer)
	if layout.Width <= 0 || layout.Height <= 0 {
		return joinSections(header, body, footer)
	}
	header = limitLines(header, layout.HeaderContentHeight)
	body = limitLines(body, layout.BodyContentHeight)
	footer = limitLines(footer, layout.FooterContentHeight)
	headerBox := renderHeaderSection(header, layout.Width, layout.HeaderHeight, styleManager)
	bodyBox := renderFrameSection(body, layout.Width, layout.BodyHeight, styleManager)
	footerBox := renderFooterSection(footer, layout.Width, layout.FooterHeight, styleManager)
	frame := lipgloss.JoinVertical(lipgloss.Top, headerBox, bodyBox, footerBox)
	if layout.TopMargin > 0 {
		frame = strings.Repeat("\n", layout.TopMargin) + frame
	}
	frame = lipgloss.NewStyle().Width(layout.Width).Height(layout.Height).Render(frame)
	if styleManager == nil {
		return frame
	}
	return styleManager.ApplyFullBackground(frame, layout.Width, layout.Height)
}

func calculateFrameLayout(headerLines, footerLines, width, height int) FrameLayout {
	if width <= 0 || height <= 0 {
		return FrameLayout{Width: width, Height: height}
	}
	minSectionHeight := resolveMinSectionHeight(height)
	topMargin := frameTopMargin
	if height < 10 {
		topMargin = 0
	}
	headerHeight := max(headerLines, 1)
	footerHeight := max(footerLines, 1)
	bodyHeight := height - topMargin - headerHeight - footerHeight
	if bodyHeight < minSectionHeight {
		deficit := minSectionHeight - bodyHeight
		headerHeight, footerHeight = reduceSectionHeights(headerHeight, footerHeight, deficit, minSectionHeight)
		bodyHeight = height - topMargin - headerHeight - footerHeight
		if bodyHeight < 1 {
			bodyHeight = 1
		}
	}
	contentWidth := max(width-frameBorderSize-framePaddingX*2, 1)
	bodyContentHeight := max(bodyHeight-frameBorderSize-framePaddingY*2, 0)
	return FrameLayout{Width: width, Height: height, TopMargin: topMargin, HeaderHeight: headerHeight, BodyHeight: bodyHeight, FooterHeight: footerHeight, ContentWidth: contentWidth, HeaderContentHeight: headerHeight, BodyContentHeight: bodyContentHeight, FooterContentHeight: footerHeight}
}

func renderFrameSection(content string, width, height int, styleManager *StyleManager) string {
	contentWidth := max(width-frameBorderSize-framePaddingX*2, 0)
	contentHeight := max(height-frameBorderSize-framePaddingY*2, 0)

	style := mstyle(styleManager).Box(styleManager.scheme.Subtle, contentWidth, contentHeight, 0, framePaddingX, framePaddingY)
	return style.Render(content)
}

func renderHeaderSection(content string, width, height int, styleManager *StyleManager) string {
	sm := mstyle(styleManager)
	contentWidth := max(width-headerBarWidth-framePaddingX*2, 1)
	style := sm.applyBGFG(lipgloss.NewStyle().
		Border(headerAccentBorder, false, false, false, true).
		BorderForeground(sm.scheme.Focused).
		Padding(0, framePaddingX).
		Width(contentWidth).
		Height(max(height, 1)))
	return style.Render(content)
}

func renderFooterSection(content string, width, height int, styleManager *StyleManager) string {
	sm := mstyle(styleManager)
	contentWidth := max(width-framePaddingX*2-headerBarWidth, 1)
	style := sm.applyBGFG(lipgloss.NewStyle().
		Padding(0, framePaddingX+headerBarWidth, 0, framePaddingX+headerBarWidth).
		Width(contentWidth).
		Height(max(height, 1)))
	return style.Render(content)
}

// helper to safely access styleManager in layout functions
func mstyle(sm *StyleManager) *StyleManager {
	if sm == nil {
		return defaultStyleManager
	}
	return sm
}

func limitLines(s string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}

func joinSections(header, body, footer string) string {
	var parts []string
	if header != "" {
		parts = append(parts, header)
	}
	if body != "" {
		parts = append(parts, body)
	}
	if footer != "" {
		parts = append(parts, footer)
	}
	return strings.Join(parts, "\n")
}

func trimTrailingNewlines(s string) string {
	return strings.TrimRight(s, "\n")
}

func resolveMinSectionHeight(height int) int {
	if height < 6 {
		return 1
	}
	if height < 9 {
		return 2
	}
	return 3
}

func reduceSectionHeights(headerHeight, footerHeight, deficit, minSectionHeight int) (int, int) {
	// Header and footer are borderless bars, so one line is the floor for each.
	minAllowedHeight := 1

	for deficit > 0 && (headerHeight > minAllowedHeight || footerHeight > minAllowedHeight) {
		if headerHeight > minAllowedHeight {
			headerHeight--
			deficit--
		}
		if deficit > 0 && footerHeight > minAllowedHeight {
			footerHeight--
			deficit--
		}
	}
	return headerHeight, footerHeight
}
