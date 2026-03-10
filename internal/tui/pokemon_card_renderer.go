package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/services/pokemoncard"
)

// energyAbbrev maps Pokemon energy type names to bracketed abbreviations.
var energyAbbrev = map[string]string{
	"Psychic":   "[P]",
	"Fire":      "[F]",
	"Water":     "[W]",
	"Grass":     "[G]",
	"Lightning": "[L]",
	"Darkness":  "[D]",
	"Metal":     "[M]",
	"Dragon":    "[R]",
	"Fairy":     "[Y]",
	"Colorless": "[C]",
	"Normal":    "[N]",
	"Fighting":  "[Ft]",
}

func energyCode(t string) string {
	if code, ok := energyAbbrev[t]; ok {
		return code
	}
	if len(t) > 0 {
		return "[" + string([]rune(t)[:1]) + "]"
	}
	return "[?]"
}

func energyCodes(types []string) string {
	parts := make([]string, 0, len(types))
	for _, t := range types {
		parts = append(parts, energyCode(t))
	}
	return strings.Join(parts, "")
}

// PokemonCardRenderer implements CardDetailRenderer for Pokemon cards.
type PokemonCardRenderer struct {
	service       pokemoncard.PokemonCardService
	pokemonGameID int64
}

func NewPokemonCardRenderer(service pokemoncard.PokemonCardService, pokemonGameID int64) *PokemonCardRenderer {
	return &PokemonCardRenderer{service: service, pokemonGameID: pokemonGameID}
}

func (r *PokemonCardRenderer) CanRender(cardGameID int64) bool {
	return cardGameID == r.pokemonGameID
}

func (r *PokemonCardRenderer) FetchExtra(cardID int64) any {
	pc, err := r.service.GetByCardID(cardID)
	if err != nil {
		return nil
	}
	return pc
}

func (r *PokemonCardRenderer) ModalDimensions(termWidth, termHeight int) (int, int) {
	w := max(80, termWidth-6)
	h := max(30, termHeight-4)
	return w, h
}

// RenderLeft renders the Pokemon card panel. width/height are the OUTER panel dimensions.
func (r *PokemonCardRenderer) RenderLeft(card model.Card, extra any, width, height int, sm *StyleManager) string {
	pc, _ := extra.(*model.PokemonCard)

	// RenderPanel uses padX=1 → wrapAt = innerWidth - 2 = width - 6
	cw := max(width-6, 10)

	var lines []string

	// ── Header: name left-aligned, HP right-aligned ───────────────────────────
	if pc != nil && pc.HP > 0 {
		hpStr := fmt.Sprintf("HP\u00a0%d", pc.HP)
		lines = append(lines, rightAlign(card.Name, hpStr, cw))
	} else {
		lines = append(lines, card.Name)
	}

	// ── Subtitle: stage + category + type ────────────────────────────────────
	subtitle := ""
	if pc != nil {
		subtitle = pc.Category
		if pc.Stage != "" && pc.Stage != pc.Category {
			subtitle = pc.Stage + " " + subtitle
		}
		if len(pc.Types) > 0 {
			subtitle += " · " + energyCodes(pc.Types)
		}
	}
	lines = append(lines, sm.GetBlurredStyle().Render(subtitle))

	// ── Art placeholder (text-drawn box, 5 content rows + 2 border = 7 rows) ─
	lines = append(lines, "")
	lines = append(lines, artPlaceholder(cw, 5, sm))
	lines = append(lines, "")

	if pc == nil {
		lines = append(lines, sm.GetBlurredStyle().Render("(no Pokemon data)"))
	} else {
		// ── Abilities ────────────────────────────────────────────────────────
		for _, a := range pc.Abilities {
			lines = append(lines, "")
			lines = append(lines, sm.GetTitleStyle().Render(fmt.Sprintf("■ %s: %s", a.Type, a.Name)))
			if a.Effect != "" {
				lines = append(lines, wrapText(a.Effect, cw, "  "))
			}
		}

		// ── Attacks ──────────────────────────────────────────────────────────
		for _, a := range pc.Attacks {
			lines = append(lines, "")
			atkName := energyCodes(a.Cost) + "\u00a0" + a.Name
			if a.Damage != "" {
				lines = append(lines, rightAlign(atkName, a.Damage, cw))
			} else {
				lines = append(lines, atkName)
			}
			if a.Effect != "" {
				lines = append(lines, wrapText(a.Effect, cw, "  "))
			}
		}

		// ── Separator + stats ─────────────────────────────────────────────────
		lines = append(lines, "")
		lines = append(lines, sm.GetBlurredStyle().Render(strings.Repeat("─", cw)))

		var stats []string
		if len(pc.Weaknesses) > 0 {
			parts := make([]string, 0, len(pc.Weaknesses))
			for _, w := range pc.Weaknesses {
				parts = append(parts, energyCode(w.Type)+w.Value)
			}
			stats = append(stats, "Weak: "+strings.Join(parts, ", "))
		}
		if len(pc.Resistances) > 0 {
			parts := make([]string, 0, len(pc.Resistances))
			for _, res := range pc.Resistances {
				parts = append(parts, energyCode(res.Type)+res.Value)
			}
			stats = append(stats, "Resist: "+strings.Join(parts, ", "))
		}
		if pc.Retreat > 0 {
			stats = append(stats, "Retreat: "+strings.Repeat("[C]", pc.Retreat))
		}
		if len(stats) > 0 {
			lines = append(lines, strings.Join(stats, "  "))
		}
	}

	content := strings.Join(lines, "\n")
	return RenderPanel(sm, content, width, height, false, 1, 0)
}

// rightAlign places left-text on the left and right-text on the right within width cols.
// Uses non-breaking spaces (\u00a0) for padding so lipgloss won't word-wrap across the gap.
func rightAlign(left, right string, width int) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	padding := width - lw - rw
	if padding < 1 {
		padding = 1
	}
	return left + strings.Repeat("\u00a0", padding) + right
}

// artPlaceholder draws a plain-text box of outer width=cw and innerRows content rows.
// Uses plain Unicode box-drawing characters (no ANSI sub-styles) to avoid width
// measurement surprises in the containing lipgloss box.
func artPlaceholder(cw, innerRows int, _ *StyleManager) string {
	inner := max(cw-2, 2)
	label := "· · ·"
	lw := lipgloss.Width(label)

	top := "╭" + strings.Repeat("─", inner) + "╮"
	bot := "╰" + strings.Repeat("─", inner) + "╯"
	emptyRow := "│" + strings.Repeat("\u00a0", inner) + "│"

	pad := (inner - lw) / 2
	extra := inner - lw - pad
	centerRow := "│" + strings.Repeat("\u00a0", pad) + label + strings.Repeat("\u00a0", extra) + "│"

	var rows []string
	rows = append(rows, top)
	midIdx := innerRows / 2
	for i := 0; i < innerRows; i++ {
		if i == midIdx {
			rows = append(rows, centerRow)
		} else {
			rows = append(rows, emptyRow)
		}
	}
	rows = append(rows, bot)
	return strings.Join(rows, "\n")
}

// wrapText wraps text at width characters, prefixing continuation lines with indent.
func wrapText(text string, width int, indent string) string {
	if width <= lipgloss.Width(indent) {
		return indent + text
	}
	var lines []string
	words := strings.Fields(text)
	current := indent
	for _, word := range words {
		if lipgloss.Width(current)+lipgloss.Width(word)+1 > width && current != indent {
			lines = append(lines, current)
			current = indent + word
		} else {
			if current == indent {
				current += word
			} else {
				current += " " + word
			}
		}
	}
	if current != indent {
		lines = append(lines, current)
	}
	return strings.Join(lines, "\n")
}
