package tui

import (
	"fmt"
	"strings"

	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/services/mtgcard"
)

type MTGCardRenderer struct {
	service   mtgcard.MTGCardService
	mtgGameID int64
}

func NewMTGCardRenderer(service mtgcard.MTGCardService, mtgGameID int64) *MTGCardRenderer {
	return &MTGCardRenderer{service: service, mtgGameID: mtgGameID}
}

func (r *MTGCardRenderer) CanRender(cardGameID int64) bool {
	return cardGameID == r.mtgGameID
}

func (r *MTGCardRenderer) FetchExtra(cardID int64) any {
	mc, err := r.service.GetByCardID(cardID)
	if err != nil {
		return nil
	}
	return mc
}

func (r *MTGCardRenderer) ModalDimensions(termWidth, termHeight int) (int, int) {
	return max(80, termWidth-6), max(30, termHeight-4)
}

func (r *MTGCardRenderer) RenderLeft(card model.Card, extra any, width, height int, sm *StyleManager) string {
	mc, _ := extra.(*model.MagicCard)
	cw := max(width-6, 10)
	var lines []string
	lines = append(lines, card.Name)
	if mc != nil && mc.ManaCost != "" {
		subtitle := mc.ManaCost
		if mc.CMC > 0 {
			subtitle += fmt.Sprintf(" (CMC: %g)", mc.CMC)
		}
		lines = append(lines, sm.GetBlurredStyle().Render(subtitle))
	} else {
		lines = append(lines, sm.GetBlurredStyle().Render(card.Rarity))
	}
	lines = append(lines, "")
	lines = append(lines, artPlaceholder(cw, 5, sm))
	lines = append(lines, "")
	if mc == nil {
		lines = append(lines, sm.GetBlurredStyle().Render("(no Magic card data)"))
	} else {
		if mc.TypeLine != "" {
			lines = append(lines, mc.TypeLine)
		}
		if mc.Power != "" && mc.Toughness != "" {
			lines = append(lines, rightAlign(fmt.Sprintf("%s/%s", mc.Power, mc.Toughness), card.Rarity, cw))
		} else if mc.Loyalty != "" {
			lines = append(lines, rightAlign(fmt.Sprintf("Loyalty: %s", mc.Loyalty), card.Rarity, cw))
		}
		if len(mc.Colors) > 0 {
			lines = append(lines, sm.GetBlurredStyle().Render("Colors: "+strings.Join(mc.Colors, ", ")))
		}
		if mc.Text != "" {
			lines = append(lines, "")
			lines = append(lines, sm.GetBlurredStyle().Render(strings.Repeat("─", cw)))
			lines = append(lines, wrapText(mc.Text, cw, ""))
		}
		if mc.Flavor != "" {
			lines = append(lines, "")
			lines = append(lines, sm.GetBlurredStyle().Render(strings.Repeat("─", cw)))
			lines = append(lines, sm.GetBlurredStyle().Render(wrapText(mc.Flavor, cw, "")))
		}
	}
	content := strings.Join(lines, "\n")
	return RenderPanel(sm, content, width, height, false, 1, 0)
}
