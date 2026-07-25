package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
)

const (
	SuccessIcon    = "✓"
	FailureIcon    = "✗"
	PendingIcon    = "○"
	ImportIcon     = "↓"
	PokemonIcon    = "★"
	MTGIcon        = "♦"
	YuGiOhIcon     = "◆"
	Separator      = "│"
	ExpandSymbol   = "▶"
	CollapseSymbol = "▼"
	ListSymbol     = "≡"
	Ellipsis       = "…"
	CommonIcon     = "○"
	UncommonIcon   = "◆"
	RareIcon       = "★"
	UltraRareIcon  = "✦"
)

var ImportSpinner = spinner.Spinner{
	Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	FPS:    time.Second / 10,
}

var CardSpinner = spinner.Spinner{
	Frames: []string{"★ ", " ★", "  ★", "   ★", "    ★"},
	FPS:    time.Second / 5,
}
