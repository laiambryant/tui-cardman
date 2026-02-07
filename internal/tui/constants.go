package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"time"
)

const (
	headerHeight     = 4
	footerHeight     = 1
	smallScreen      = 130
	paneTitleHeight  = 1
	defaultPaneWidth = 30
	focusedPaneWidth = 50
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

const Logo = `▗▄▄▖ ▗▄▖ ▗▄▄▖ ▗▄▄▄  ▗▖  ▗▖ ▗▄▖ ▗▖  ▗▖
▐▌   ▐▌ ▐▌▐▌ ▐▌▐▌  █ ▐▛▚▞▜▌▐▌ ▐▌▐▛▚▖▐▌
▐▌   ▐▛▀▜▌▐▛▀▚▖▐▌  █ ▐▌  ▐▌▐▛▀▜▌▐▌ ▝▜▌
▝▚▄▄▖▐▌ ▐▌▐▌ ▐▌▐▙▄▄▀ ▐▌  ▐▌▐▌ ▐▌▐▌  ▐▌`

var ImportSpinner = spinner.Spinner{
	Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	FPS:    time.Second / 10,
}

var CardSpinner = spinner.Spinner{
	Frames: []string{"★ ", " ★", "  ★", "   ★", "    ★"},
	FPS:    time.Second / 5,
}
