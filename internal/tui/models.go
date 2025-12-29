package tui

import "time"

// CardGame represents a card game in the database
type CardGame struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}
