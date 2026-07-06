package tui

import (
	"fmt"
	"strings"

	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/services/yugiohcard"
)

// YuGiOhCardRenderer implements CardDetailRenderer for Yu-Gi-Oh! cards.
type YuGiOhCardRenderer struct {
	service      yugiohcard.YuGiOhCardService
	yugiohGameID int64
}

func NewYuGiOhCardRenderer(service yugiohcard.YuGiOhCardService, yugiohGameID int64) *YuGiOhCardRenderer {
	return &YuGiOhCardRenderer{service: service, yugiohGameID: yugiohGameID}
}

func (r *YuGiOhCardRenderer) CanRender(cardGameID int64) bool {
	return cardGameID == r.yugiohGameID
}

func (r *YuGiOhCardRenderer) FetchExtra(cardID int64) any {
	yc, err := r.service.GetByCardID(cardID)
	if err != nil {
		return nil
	}
	return yc
}

func (r *YuGiOhCardRenderer) ModalDimensions(termWidth, termHeight int) (int, int) {
	w := max(80, termWidth-6)
	h := max(30, termHeight-4)
	return w, h
}

func yugiohSubtitle(yc *model.YuGiOhCard, card model.Card, sm *StyleManager) string {
	if yc == nil {
		return sm.GetBlurredStyle().Render(card.Rarity)
	}
	subtitle := yc.CardType
	if yc.FrameType != "" && yc.FrameType != yc.CardType {
		subtitle += " (" + yc.FrameType + ")"
	}
	return sm.GetBlurredStyle().Render(subtitle)
}

func yugiohStatLine(yc *model.YuGiOhCard, sm *StyleManager) []string {
	var statParts []string
	if yc.Attribute != nil {
		statParts = append(statParts, *yc.Attribute)
	}
	if yc.Race != nil && *yc.Race != "" {
		statParts = append(statParts, *yc.Race)
	}
	if yc.Level != nil {
		statParts = append(statParts, fmt.Sprintf("Level %d", *yc.Level))
	} else if yc.LinkVal != nil {
		statParts = append(statParts, fmt.Sprintf("Link-%d", *yc.LinkVal))
	}
	if len(statParts) == 0 {
		return nil
	}
	return []string{sm.GetBlurredStyle().Render(strings.Join(statParts, " / "))}
}

func yugiohCombatLine(yc *model.YuGiOhCard, card model.Card, cw int) []string {
	if yc.ATK == nil && yc.DEF == nil {
		return nil
	}
	atk := "---"
	def := "---"
	if yc.ATK != nil {
		atk = fmt.Sprintf("%d", *yc.ATK)
	}
	if yc.DEF != nil {
		def = fmt.Sprintf("%d", *yc.DEF)
	} else if yc.LinkVal != nil {
		def = fmt.Sprintf("LINK-%d", *yc.LinkVal)
	}
	atkDef := fmt.Sprintf("ATK/%s  DEF/%s", atk, def)
	return []string{rightAlign(atkDef, card.Rarity, cw)}
}

func yugiohMarkerLine(yc *model.YuGiOhCard, sm *StyleManager) []string {
	if len(yc.LinkMarkers) == 0 {
		return nil
	}
	return []string{sm.GetBlurredStyle().Render("Markers: " + strings.Join(yc.LinkMarkers, " "))}
}

func yugiohDescriptionLines(yc *model.YuGiOhCard, cw int, sm *StyleManager) []string {
	if yc.Description == "" {
		return nil
	}
	return []string{
		"",
		sm.GetBlurredStyle().Render(strings.Repeat("─", cw)),
		wrapText(yc.Description, cw, ""),
	}
}

// RenderLeft renders the Yu-Gi-Oh! card panel.
func (r *YuGiOhCardRenderer) RenderLeft(card model.Card, extra any, width, height int, sm *StyleManager) string {
	yc, _ := extra.(*model.YuGiOhCard)
	cw := max(width-6, 10)
	lines := []string{
		card.Name,
		yugiohSubtitle(yc, card, sm),
		"",
		artPlaceholder(cw, 5, sm),
		"",
	}
	if yc == nil {
		lines = append(lines, sm.GetBlurredStyle().Render("(no Yu-Gi-Oh! data)"))
	} else {
		lines = append(lines, yugiohStatLine(yc, sm)...)
		lines = append(lines, yugiohCombatLine(yc, card, cw)...)
		lines = append(lines, yugiohMarkerLine(yc, sm)...)
		lines = append(lines, yugiohDescriptionLines(yc, cw, sm)...)
	}
	return RenderPanel(sm, strings.Join(lines, "\n"), width, height, false, 1, 0)
}
