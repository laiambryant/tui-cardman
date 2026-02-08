package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const frameBorderSize = 2
const framePaddingX = 1
const framePaddingY = 0

type FrameLayout struct {
	Width               int
	Height              int
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
	headerBox := renderFrameSection(header, layout.Width, layout.HeaderHeight, styleManager)
	bodyBox := renderFrameSection(body, layout.Width, layout.BodyHeight, styleManager)
	footerBox := renderFrameSection(footer, layout.Width, layout.FooterHeight, styleManager)
	frame := lipgloss.JoinVertical(lipgloss.Top, headerBox, bodyBox, footerBox)
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
	headerLines = max(headerLines, 1)
	footerLines = max(footerLines, 1)
	headerHeight := headerLines + frameBorderSize + framePaddingY*2
	footerHeight := footerLines + frameBorderSize + framePaddingY*2
	bodyHeight := height - headerHeight - footerHeight
	if bodyHeight < minSectionHeight {
		deficit := minSectionHeight - bodyHeight
		headerHeight, footerHeight = reduceSectionHeights(headerHeight, footerHeight, deficit, minSectionHeight)
		bodyHeight = height - headerHeight - footerHeight
		if bodyHeight < 1 {
			bodyHeight = 1
		}
	}
	contentWidth := max(width-frameBorderSize-framePaddingX*2, 1)
	headerContentHeight := max(headerHeight-frameBorderSize-framePaddingY*2, 0)
	bodyContentHeight := max(bodyHeight-frameBorderSize-framePaddingY*2, 0)
	footerContentHeight := max(footerHeight-frameBorderSize-framePaddingY*2, 0)
	return FrameLayout{Width: width, Height: height, HeaderHeight: headerHeight, BodyHeight: bodyHeight, FooterHeight: footerHeight, ContentWidth: contentWidth, HeaderContentHeight: headerContentHeight, BodyContentHeight: bodyContentHeight, FooterContentHeight: footerContentHeight}
}

func renderFrameSection(content string, width, height int, styleManager *StyleManager) string {
	contentWidth := max(width-frameBorderSize-framePaddingX*2, 0)
	contentHeight := max(height-frameBorderSize-framePaddingY*2, 0)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(framePaddingY, framePaddingX).
		Width(contentWidth).
		Height(contentHeight)

	if styleManager != nil {
		style = style.BorderForeground(styleManager.scheme.Blurred)
		if styleManager.scheme.Background != "" {
			style = style.Background(styleManager.scheme.Background).BorderBackground(styleManager.scheme.Background)
		}
		if styleManager.scheme.Foreground != "" {
			style = style.Foreground(styleManager.scheme.Foreground)
		}
	}

	return style.Render(content)
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
	// Ensure we never reduce below 3 to maintain at least 1 line of visible content
	// (3 = 1 content line + 2 for border)
	minAllowedHeight := max(minSectionHeight, 3)

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
