package tui

import "github.com/laiambryant/tui-cardman/internal/model"

// CardDetailRenderer renders game-specific content in the card detail view.
type CardDetailRenderer interface {
	CanRender(cardGameID int64) bool
	FetchExtra(cardID int64) any
	RenderLeft(card model.Card, extra any, width, height int, sm *StyleManager) string
	ModalDimensions(termWidth, termHeight int) (int, int)
}
