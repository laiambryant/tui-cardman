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

const Logo = `
     ___   ___   ___   ___   _  _   ___   _  _
    / __| / _ \ | _ \ |   \ | \/ | / _ \ | \| |
   | (__ | (_| ||   / | |) || |\/|| (_| || .  |
    \___| \__._||_|_\ |___/ |_|  | \__._||_|\_|

        +---------+  +---------+  +---------+
        | *     * |  |  /\_/\  |  | ~~   ~~ |
        |    *    |  | ( o.o ) |  |  ~~ ~~  |
        | *     * |  |  > ^ <  |  | ~~   ~~ |
        +---------+  +---------+  +---------+

           ~ Your Card Collection Manager ~
`

var ImportSpinner = spinner.Spinner{
	Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	FPS:    time.Second / 10,
}

var CardSpinner = spinner.Spinner{
	Frames: []string{"★ ", " ★", "  ★", "   ★", "    ★"},
	FPS:    time.Second / 5,
}
