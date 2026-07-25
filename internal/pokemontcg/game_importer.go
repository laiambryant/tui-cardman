package pokemontcg

import (
	"context"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

// NewPokemonGameImporter creates a gameimporter.GameImporter for Pokemon TCG.
func NewPokemonGameImporter(client *Client, service *ImportService, setService sets.SetService) *gameimporter.Base {
	return &gameimporter.Base{
		Runner:     service,
		SetService: setService,
		FetchSetsFn: func(ctx context.Context) ([]gameimporter.GameSet, error) {
			ptcgSets, err := client.GetSets(ctx)
			if err != nil {
				return nil, err
			}
			result := make([]gameimporter.GameSet, 0, len(ptcgSets))
			for _, s := range ptcgSets {
				result = append(result, gameimporter.GameSet{
					APIID: s.ID,
					Name:  s.Name,
					Code:  s.PtcgoCode,
					Total: s.Total,
				})
			}
			return result, nil
		},
	}
}
