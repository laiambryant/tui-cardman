// Package importers builds the importer for each supported card game.
package importers

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/mtg"
	"github.com/laiambryant/tui-cardman/internal/onepiece"
	"github.com/laiambryant/tui-cardman/internal/pokemontcg"
	"github.com/laiambryant/tui-cardman/internal/yugioh"
)

// ErrUnsupportedGame is returned for card games without an importer.
var ErrUnsupportedGame = errors.New("no importer for this card game")

// New returns the importer for the card game stored as gameName in the
// card_games table. apiKey is only used by the Pokemon TCG API.
func New(database *sql.DB, gameName, apiKey string) (*gameimporter.Importer, error) {
	switch gameName {
	case pokemontcg.GameName:
		return pokemontcg.NewImporter(database, apiKey)
	case mtg.GameName:
		return mtg.NewImporter(database)
	case yugioh.GameName:
		return yugioh.NewImporter(database)
	case onepiece.GameName:
		return onepiece.NewImporter(database)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedGame, gameName)
	}
}
