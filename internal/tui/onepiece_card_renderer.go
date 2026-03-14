package tui

import (
	"fmt"
	"strings"

	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/services/onepiececard"
)

type OnePieceCardRenderer struct {
	service  onepiececard.OnePieceCardService
	opGameID int64
}

func NewOnePieceCardRenderer(service onepiececard.OnePieceCardService, opGameID int64) *OnePieceCardRenderer {
	return &OnePieceCardRenderer{service: service, opGameID: opGameID}
}

func (r *OnePieceCardRenderer) CanRender(cardGameID int64) bool {
	return cardGameID == r.opGameID
}

func (r *OnePieceCardRenderer) FetchExtra(cardID int64) any {
	oc, err := r.service.GetByCardID(cardID)
	if err != nil {
		return nil
	}
	return oc
}

func (r *OnePieceCardRenderer) ModalDimensions(termWidth, termHeight int) (int, int) {
	return max(80, termWidth-6), max(30, termHeight-4)
}

func (r *OnePieceCardRenderer) RenderLeft(card model.Card, extra any, width, height int, sm *StyleManager) string {
	oc, _ := extra.(*model.OnePieceCard)
	cw := max(width-6, 10)
	var lines []string
	lines = append(lines, card.Name)
	lines = append(lines, renderOnePieceSubtitle(oc, card, sm))
	lines = append(lines, "")
	lines = append(lines, artPlaceholder(cw, 5, sm))
	lines = append(lines, "")
	if oc == nil {
		lines = append(lines, sm.GetBlurredStyle().Render("(no One Piece data)"))
	} else {
		lines = append(lines, renderOnePieceStats(oc, cw, card, sm)...)
		lines = append(lines, renderOnePieceText(oc, cw, sm)...)
	}
	content := strings.Join(lines, "\n")
	return RenderPanel(sm, content, width, height, false, 1, 0)
}

func renderOnePieceSubtitle(oc *model.OnePieceCard, card model.Card, sm *StyleManager) string {
	if oc == nil {
		return sm.GetBlurredStyle().Render(card.Rarity)
	}
	subtitle := oc.CardType
	if oc.CardColor != "" {
		subtitle += " (" + oc.CardColor + ")"
	}
	return sm.GetBlurredStyle().Render(subtitle)
}

func renderOnePieceStats(oc *model.OnePieceCard, cw int, card model.Card, sm *StyleManager) []string {
	var lines []string
	var statParts []string
	if oc.SubTypes != "" {
		statParts = append(statParts, oc.SubTypes)
	}
	if oc.Attribute != "" {
		statParts = append(statParts, oc.Attribute)
	}
	if len(statParts) > 0 {
		lines = append(lines, sm.GetBlurredStyle().Render(strings.Join(statParts, " / ")))
	}
	lines = append(lines, renderOnePieceCombatLine(oc, cw, card)...)
	return lines
}

func renderOnePieceCombatLine(oc *model.OnePieceCard, cw int, card model.Card) []string {
	var parts []string
	if oc.CardCost != "" {
		parts = append(parts, fmt.Sprintf("Cost: %s", oc.CardCost))
	}
	if oc.CardPower != "" {
		parts = append(parts, fmt.Sprintf("Power: %s", oc.CardPower))
	}
	if oc.CounterAmount != "" {
		parts = append(parts, fmt.Sprintf("Counter: %s", oc.CounterAmount))
	}
	if oc.Life != "" {
		parts = append(parts, fmt.Sprintf("Life: %s", oc.Life))
	}
	if len(parts) == 0 {
		return nil
	}
	return []string{rightAlign(strings.Join(parts, "  "), card.Rarity, cw)}
}

func renderOnePieceText(oc *model.OnePieceCard, cw int, sm *StyleManager) []string {
	if oc.CardText == "" {
		return nil
	}
	return []string{
		"",
		sm.GetBlurredStyle().Render(strings.Repeat("─", cw)),
		wrapText(oc.CardText, cw, ""),
	}
}
