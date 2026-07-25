package mtg

import (
	"context"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

// NewMTGGameImporter creates a gameimporter.GameImporter for Magic: The Gathering.
func NewMTGGameImporter(client *Client, service *ImportService, setService sets.SetService) *gameimporter.Base {
	return &gameimporter.Base{
		CardGameID: service.mtgGameID,
		Runner:     service,
		SetService: setService,
		FetchSetsFn: func(ctx context.Context) ([]gameimporter.GameSet, error) {
			mtgSets, err := client.GetSets(ctx)
			if err != nil {
				return nil, err
			}
			result := make([]gameimporter.GameSet, 0, len(mtgSets))
			for _, s := range mtgSets {
				result = append(result, gameimporter.GameSet{
					APIID: s.SetCode,
					Name:  s.Name,
					Code:  s.SetCode,
					Total: s.NumCards,
				})
			}
			return result, nil
		},
	}
}
