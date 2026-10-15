package tui

import (
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/laiambryant/tui-cardman/internal/model"
)

var rarityColors = map[string]lipgloss.Color{
	"Common":            lipgloss.Color("252"),
	"Uncommon":          lipgloss.Color("34"),
	"Rare":              lipgloss.Color("33"),
	"Rare Holo":         lipgloss.Color("33"),
	"Rare Holo EX":      lipgloss.Color("33"),
	"Rare Holo GX":      lipgloss.Color("33"),
	"Rare Holo V":       lipgloss.Color("33"),
	"Rare VMAX":         lipgloss.Color("220"),
	"Rare VSTAR":        lipgloss.Color("220"),
	"Rare Ultra":        lipgloss.Color("220"),
	"Rare Secret":       lipgloss.Color("220"),
	"Rare Rainbow":      lipgloss.Color("213"),
	"Rare Shiny":        lipgloss.Color("220"),
	"Amazing Rare":      lipgloss.Color("213"),
	"LEGEND":            lipgloss.Color("220"),
	"Promo":             lipgloss.Color("252"),
	"Illustration Rare": lipgloss.Color("213"),
	"Special Art Rare":  lipgloss.Color("213"),
	"Hyper Rare":        lipgloss.Color("220"),
	"Double Rare":       lipgloss.Color("220"),
	"Ultra Rare":        lipgloss.Color("220"),
	"Shiny Rare":        lipgloss.Color("220"),
	"ACE SPEC Rare":     lipgloss.Color("220"),
}

func renderSetProgressGrid(cards []model.Card, ownedCardIDs map[int64]bool, width, height int, sm *StyleManager) string {
	if len(cards) == 0 {
		return sm.GetBlurredStyle().Render("No cards in this set.")
	}
	sortCardsByNumber(cards)
	cellWidth := 2
	cols := (width - 4) / cellWidth
	if cols < 1 {
		cols = 1
	}
	var b strings.Builder
	col := 0
	for _, card := range cards {
		owned := ownedCardIDs[card.ID]
		color, ok := rarityColors[card.Rarity]
		if !ok {
			color = lipgloss.Color("252")
		}
		style := sm.applyBGFG(lipgloss.NewStyle().Foreground(color))
		if owned {
			b.WriteString(style.Render("█ "))
		} else {
			dimStyle := sm.applyBGFG(lipgloss.NewStyle().Foreground(lipgloss.Color("239")))
			b.WriteString(dimStyle.Render("░ "))
		}
		col++
		if col >= cols {
			b.WriteString("\n")
			col = 0
		}
	}
	if col > 0 {
		b.WriteString("\n")
	}
	return b.String()
}

// sortCardsByNumber orders cards by collector number, comparing numeric
// numbers by value ("2" before "10") and putting them before other numbers.
func sortCardsByNumber(cards []model.Card) {
	sort.SliceStable(cards, func(i, j int) bool {
		a, aErr := strconv.Atoi(cards[i].Number)
		b, bErr := strconv.Atoi(cards[j].Number)
		switch {
		case aErr == nil && bErr == nil:
			return a < b
		case aErr == nil || bErr == nil:
			return aErr == nil
		}
		return cards[i].Number < cards[j].Number
	})
}
