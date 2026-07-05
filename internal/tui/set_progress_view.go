package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/laiambryant/tui-cardman/internal/model"
)

var (
	rarityCommonColor  = lipgloss.AdaptiveColor{Light: "#5C5F77", Dark: "#BAC2DE"}
	rarityGreenColor   = lipgloss.AdaptiveColor{Light: "#40A02B", Dark: "#A6E3A1"}
	rarityBlueColor    = lipgloss.AdaptiveColor{Light: "#1E66F5", Dark: "#89B4FA"}
	rarityGoldColor    = lipgloss.AdaptiveColor{Light: "#DF8E1D", Dark: "#F9E2AF"}
	raritySpecialColor = lipgloss.AdaptiveColor{Light: "#EA76CB", Dark: "#F5C2E7"}
)

var rarityColors = map[string]lipgloss.TerminalColor{
	"Common":            rarityCommonColor,
	"Uncommon":          rarityGreenColor,
	"Rare":              rarityBlueColor,
	"Rare Holo":         rarityBlueColor,
	"Rare Holo EX":      rarityBlueColor,
	"Rare Holo GX":      rarityBlueColor,
	"Rare Holo V":       rarityBlueColor,
	"Rare VMAX":         rarityGoldColor,
	"Rare VSTAR":        rarityGoldColor,
	"Rare Ultra":        rarityGoldColor,
	"Rare Secret":       rarityGoldColor,
	"Rare Rainbow":      raritySpecialColor,
	"Rare Shiny":        rarityGoldColor,
	"Amazing Rare":      raritySpecialColor,
	"LEGEND":            rarityGoldColor,
	"Promo":             rarityCommonColor,
	"Illustration Rare": raritySpecialColor,
	"Special Art Rare":  raritySpecialColor,
	"Hyper Rare":        rarityGoldColor,
	"Double Rare":       rarityGoldColor,
	"Ultra Rare":        rarityGoldColor,
	"Shiny Rare":        rarityGoldColor,
	"ACE SPEC Rare":     rarityGoldColor,
}

func rarityColor(rarity string) lipgloss.TerminalColor {
	if color, ok := rarityColors[rarity]; ok {
		return color
	}
	return rarityCommonColor
}

func renderProgressCell(card model.Card, owned bool, sm *StyleManager) string {
	if owned {
		return sm.applyBGFG(lipgloss.NewStyle().Foreground(rarityColor(card.Rarity))).Render("█ ")
	}
	return sm.applyBGFG(lipgloss.NewStyle().Foreground(sm.scheme.Disabled)).Render("░ ")
}

func renderSetProgressGrid(cards []model.Card, ownedCardIDs map[int64]bool, width, height int, sm *StyleManager) string {
	if len(cards) == 0 {
		return sm.GetBlurredStyle().Render("No cards in this set.")
	}
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Number < cards[j].Number
	})
	cellWidth := 2
	cols := max((width-4)/cellWidth, 1)
	var b strings.Builder
	col := 0
	for _, card := range cards {
		b.WriteString(renderProgressCell(card, ownedCardIDs[card.ID], sm))
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
