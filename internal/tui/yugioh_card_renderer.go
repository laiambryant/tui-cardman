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

// RenderLeft renders the Yu-Gi-Oh! card panel.
func (r *YuGiOhCardRenderer) RenderLeft(card model.Card, extra any, width, height int, sm *StyleManager) string {
	yc, _ := extra.(*model.YuGiOhCard)

	cw := max(width-6, 10)
	var lines []string

	// ── Name ─────────────────────────────────────────────────────────────────
	lines = append(lines, card.Name)

	// ── Type / Frame subtitle ─────────────────────────────────────────────────
	if yc != nil {
		subtitle := yc.CardType
		if yc.FrameType != "" && yc.FrameType != yc.CardType {
			subtitle += " (" + yc.FrameType + ")"
		}
		lines = append(lines, sm.GetBlurredStyle().Render(subtitle))
	} else {
		lines = append(lines, sm.GetBlurredStyle().Render(card.Rarity))
	}

	// ── Art placeholder ───────────────────────────────────────────────────────
	lines = append(lines, "")
	lines = append(lines, artPlaceholder(cw, 5, sm))
	lines = append(lines, "")

	if yc == nil {
		lines = append(lines, sm.GetBlurredStyle().Render("(no Yu-Gi-Oh! data)"))
	} else {
		// ── Attribute / Race / Level ─────────────────────────────────────────
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
		if len(statParts) > 0 {
			lines = append(lines, sm.GetBlurredStyle().Render(strings.Join(statParts, " / ")))
		}

		// ── ATK / DEF ─────────────────────────────────────────────────────────
		isMonster := yc.ATK != nil || yc.DEF != nil
		if isMonster {
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
			lines = append(lines, rightAlign(atkDef, card.Rarity, cw))
		}

		// ── Link markers ──────────────────────────────────────────────────────
		if len(yc.LinkMarkers) > 0 {
			lines = append(lines, sm.GetBlurredStyle().Render("Markers: "+strings.Join(yc.LinkMarkers, " ")))
		}

		// ── Description ───────────────────────────────────────────────────────
		if yc.Description != "" {
			lines = append(lines, "")
			lines = append(lines, sm.GetBlurredStyle().Render(strings.Repeat("─", cw)))
			lines = append(lines, wrapText(yc.Description, cw, ""))
		}
	}

	content := strings.Join(lines, "\n")
	return RenderPanel(sm, content, width, height, false, 1, 0)
}
