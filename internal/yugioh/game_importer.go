package yugioh

import (
	"context"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

// NewYuGiOhGameImporter creates a gameimporter.GameImporter for Yu-Gi-Oh!.
func NewYuGiOhGameImporter(client *Client, service *ImportService, setService sets.SetService) *gameimporter.Base {
	return &gameimporter.Base{
		Runner:     service,
		SetService: setService,
		FetchSetsFn: func(ctx context.Context) ([]gameimporter.GameSet, error) {
			ygoSets, err := client.GetSets(ctx)
			if err != nil {
				return nil, err
			}
			result := make([]gameimporter.GameSet, 0, len(ygoSets))
			for _, s := range ygoSets {
				result = append(result, gameimporter.GameSet{
					APIID: s.SetCode,
					Name:  s.SetName,
					Code:  s.SetCode,
					Total: s.NumCards,
				})
			}
			return result, nil
		},
	}
}
