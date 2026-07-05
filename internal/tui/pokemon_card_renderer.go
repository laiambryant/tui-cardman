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

// pokemonHeaderLine renders the card name left-aligned with HP right-aligned when available.
func pokemonHeaderLine(pc *model.PokemonCard, card model.Card, cw int) string {
	if pc != nil && pc.HP > 0 {
		return rightAlign(card.Name, fmt.Sprintf("HP %d", pc.HP), cw)
	}
	return card.Name
}

// pokemonSubtitle combines stage, category, and energy types into one line.
func pokemonSubtitle(pc *model.PokemonCard) string {
	if pc == nil {
		return ""
	}
	subtitle := pc.Category
	if pc.Stage != "" && pc.Stage != pc.Category {
		subtitle = pc.Stage + " " + subtitle
	}
	if len(pc.Types) > 0 {
		subtitle += " · " + energyCodes(pc.Types)
	}
	return subtitle
}

func pokemonAbilityLines(pc *model.PokemonCard, cw int, sm *StyleManager) []string {
	var lines []string
	for _, a := range pc.Abilities {
		lines = append(lines, "")
		lines = append(lines, sm.GetTitleStyle().Render(fmt.Sprintf("■ %s: %s", a.Type, a.Name)))
		if a.Effect != "" {
			lines = append(lines, wrapText(a.Effect, cw, "  "))
		}
	}
	return lines
}

func pokemonAttackLines(pc *model.PokemonCard, cw int) []string {
	var lines []string
	for _, a := range pc.Attacks {
		lines = append(lines, "")
		atkName := energyCodes(a.Cost) + " " + a.Name
		if a.Damage != "" {
			lines = append(lines, rightAlign(atkName, a.Damage, cw))
		} else {
			lines = append(lines, atkName)
		}
		if a.Effect != "" {
			lines = append(lines, wrapText(a.Effect, cw, "  "))
		}
	}
	return lines
}

func pokemonTypeValuePairs(label string, pairs []model.PokemonCardWeakRes) string {
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, energyCode(p.Type)+p.Value)
	}
	return label + strings.Join(parts, ", ")
}

func pokemonStatLines(pc *model.PokemonCard, cw int, sm *StyleManager) []string {
	lines := []string{"", sm.GetBlurredStyle().Render(strings.Repeat("─", cw))}
	var stats []string
	if len(pc.Weaknesses) > 0 {
		stats = append(stats, pokemonTypeValuePairs("Weak: ", pc.Weaknesses))
	}
	if len(pc.Resistances) > 0 {
		stats = append(stats, pokemonTypeValuePairs("Resist: ", pc.Resistances))
	}
	if pc.Retreat > 0 {
		stats = append(stats, "Retreat: "+strings.Repeat("[C]", pc.Retreat))
	}
	if len(stats) > 0 {
		lines = append(lines, strings.Join(stats, "  "))
	}
	return lines
}

// RenderLeft renders the Pokemon card panel. width/height are the OUTER panel dimensions;
// the wrap width accounts for RenderPanel's border and padding overhead.
func (r *PokemonCardRenderer) RenderLeft(card model.Card, extra any, width, height int, sm *StyleManager) string {
	pc, _ := extra.(*model.PokemonCard)
	cw := max(width-6, 10)
	lines := []string{
		pokemonHeaderLine(pc, card, cw),
		sm.GetBlurredStyle().Render(pokemonSubtitle(pc)),
		"",
		artPlaceholder(cw, 5, sm),
		"",
	}
	if pc == nil {
		lines = append(lines, sm.GetBlurredStyle().Render("(no Pokemon data)"))
	} else {
		lines = append(lines, pokemonAbilityLines(pc, cw, sm)...)
		lines = append(lines, pokemonAttackLines(pc, cw)...)
		lines = append(lines, pokemonStatLines(pc, cw, sm)...)
	}
	return RenderPanel(sm, strings.Join(lines, "\n"), width, height, false, 1, 0)
}

// rightAlign places left-text on the left and right-text on the right within width cols.
// Uses non-breaking spaces ( ) for padding so lipgloss won't word-wrap across the gap.
func rightAlign(left, right string, width int) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	padding := max(width-lw-rw, 1)
	return left + strings.Repeat(" ", padding) + right
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
	emptyRow := "│" + strings.Repeat(" ", inner) + "│"
	pad := (inner - lw) / 2
	extra := inner - lw - pad
	centerRow := "│" + strings.Repeat(" ", pad) + label + strings.Repeat(" ", extra) + "│"
	rows := []string{top}
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

func appendWrappedWord(lines []string, current, word, indent string, width int) ([]string, string) {
	if lipgloss.Width(current)+lipgloss.Width(word)+1 > width && current != indent {
		return append(lines, current), indent + word
	}
	if current == indent {
		return lines, current + word
	}
	return lines, current + " " + word
}

// wrapText wraps text at width characters, prefixing continuation lines with indent.
func wrapText(text string, width int, indent string) string {
	if width <= lipgloss.Width(indent) {
		return indent + text
	}
	var lines []string
	current := indent
	for _, word := range strings.Fields(text) {
		lines, current = appendWrappedWord(lines, current, word, indent, width)
	}
	if current != indent {
		lines = append(lines, current)
	}
	return strings.Join(lines, "\n")
}
