package onepiece

import (
	"context"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

// NewOnePieceGameImporter creates a gameimporter.GameImporter for One Piece.
func NewOnePieceGameImporter(client *Client, service *ImportService, setService sets.SetService) *gameimporter.Base {
	return &gameimporter.Base{
		Runner:     service,
		SetService: setService,
		FetchSetsFn: func(ctx context.Context) ([]gameimporter.GameSet, error) {
			opSets, err := client.GetSets(ctx)
			if err != nil {
				return nil, err
			}
			result := make([]gameimporter.GameSet, 0, len(opSets))
			for _, s := range opSets {
				result = append(result, gameimporter.GameSet{
					APIID: s.SetID,
					Name:  s.SetName,
					Code:  s.SetID,
				})
			}
			return result, nil
		},
	}
}
