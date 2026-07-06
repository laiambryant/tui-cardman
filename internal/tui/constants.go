package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
)

const (
	SuccessIcon = "✓"
	FailureIcon = "✗"
	PendingIcon = "○"
	ImportIcon  = "↓"
	ListSymbol  = "≡"
	Ellipsis    = "…"
)

var ImportSpinner = spinner.Spinner{
	Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	FPS:    time.Second / 10,
}
