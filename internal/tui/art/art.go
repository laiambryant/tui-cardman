// Package art provides ASCII art rendering utilities for the TUI.
package art

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const Tagline = "Terminal Card Collection Manager"

// Logo is the plain-text wordmark used outside the TUI (e.g. version output).
const Logo = `
   █▀▀ ▄▀█ █▀█ █▀▄ █▀▄▀█ ▄▀█ █▄ █
   █▄▄ █▀█ █▀▄ █▄▀ █ ▀ █ █▀█ █ ▀█

   ~ ` + Tagline + ` ~
`

// Half-block wordmark, two rows per letter: C A R D M A N.
var wordmark = []string{
	"█▀▀ ▄▀█ █▀█ █▀▄ █▀▄▀█ ▄▀█ █▄ █",
	"█▄▄ █▀█ █▀▄ █▄▀ █ ▀ █ █▀█ █ ▀█",
}

var cardIcons = []string{"★", "♦", "◆", "♠"}

func miniCard(icon string) []string {
	return []string{
		"╭─────╮",
		fmt.Sprintf("│ %s   │", icon),
		"│  ◢  │",
		fmt.Sprintf("│   %s │", icon),
		"╰─────╯",
	}
}

type rgb struct{ r, g, b int }

func parseHexColor(c lipgloss.Color) (rgb, bool) {
	s := string(c)
	if len(s) != 7 || s[0] != '#' {
		return rgb{}, false
	}
	r, err1 := strconv.ParseInt(s[1:3], 16, 32)
	g, err2 := strconv.ParseInt(s[3:5], 16, 32)
	b, err3 := strconv.ParseInt(s[5:7], 16, 32)
	if err1 != nil || err2 != nil || err3 != nil {
		return rgb{}, false
	}
	return rgb{int(r), int(g), int(b)}, true
}

func lerpColor(from, to rgb, t float64) lipgloss.Color {
	r := from.r + int(t*float64(to.r-from.r))
	g := from.g + int(t*float64(to.g-from.g))
	b := from.b + int(t*float64(to.b-from.b))
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, b))
}

// gradientLine colors each rune of a line by interpolating between two colors.
// When either color is not a hex value the whole line is rendered in `from`.
func gradientLine(line string, from, to lipgloss.Color) string {
	fromRGB, okFrom := parseHexColor(from)
	toRGB, okTo := parseHexColor(to)
	if !okFrom || !okTo {
		return lipgloss.NewStyle().Foreground(from).Bold(true).Inline(true).Render(line)
	}
	runes := []rune(line)
	span := max(len(runes)-1, 1)
	var b strings.Builder
	for i, r := range runes {
		if r == ' ' {
			b.WriteRune(r)
			continue
		}
		color := lerpColor(fromRGB, toRGB, float64(i)/float64(span))
		b.WriteString(lipgloss.NewStyle().Foreground(color).Bold(true).Inline(true).Render(string(r)))
	}
	return b.String()
}

func renderCardFan(accents []lipgloss.Color) string {
	var cards []string
	for i, icon := range cardIcons {
		accent := accents[i%len(accents)]
		style := lipgloss.NewStyle().Foreground(accent).Inline(true)
		var lines []string
		for _, line := range miniCard(icon) {
			lines = append(lines, style.Render(line))
		}
		cards = append(cards, strings.Join(lines, "\n"))
	}
	gap := "  "
	row := cards[0]
	for _, card := range cards[1:] {
		row = lipgloss.JoinHorizontal(lipgloss.Center, row, gap, card)
	}
	return row
}

func renderWordmark(accents []lipgloss.Color) string {
	from := accents[0]
	to := from
	if len(accents) > 1 {
		to = accents[1]
	}
	var lines []string
	for _, line := range wordmark {
		lines = append(lines, gradientLine(line, from, to))
	}
	return strings.Join(lines, "\n")
}

func renderTagline(subtle lipgloss.Color, accents []lipgloss.Color) string {
	subtleStyle := lipgloss.NewStyle().Foreground(subtle).Inline(true)
	accentStyle := lipgloss.NewStyle().Foreground(accents[0]).Inline(true)
	return subtleStyle.Render("── ") + accentStyle.Render("✦") + subtleStyle.Render(" "+Tagline+" ") + accentStyle.Render("✦") + subtleStyle.Render(" ──")
}

// RenderLogo renders the splash screen artwork: a fan of game cards, a
// gradient CARDMAN wordmark and a tagline, tinted with the active theme.
func RenderLogo(width, height int, accents []lipgloss.Color, subtle lipgloss.Color) string {
	if len(accents) == 0 {
		accents = []lipgloss.Color{"#FF5FD2"}
	}
	wordmarkBlock := renderWordmark(accents)
	compact := lipgloss.JoinVertical(lipgloss.Center, wordmarkBlock, "", renderTagline(subtle, accents))
	if (width > 0 && width < 44) || (height > 0 && height < 14) {
		return compact
	}
	return lipgloss.JoinVertical(lipgloss.Center,
		renderCardFan(accents),
		"",
		wordmarkBlock,
		"",
		renderTagline(subtle, accents),
	)
}
