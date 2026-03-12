package pokemontcg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

// PokemonGameImporter implements gameimporter.GameImporter for Pokemon TCG.
type PokemonGameImporter struct {
	client     *Client
	service    *ImportService
	setService sets.SetService
}

// NewPokemonGameImporter creates a new PokemonGameImporter.
func NewPokemonGameImporter(client *Client, service *ImportService, setService sets.SetService) *PokemonGameImporter {
	return &PokemonGameImporter{
		client:     client,
		service:    service,
		setService: setService,
	}
}

func (i *PokemonGameImporter) FetchSets(ctx context.Context) ([]gameimporter.GameSet, error) {
	ptcgSets, err := i.client.GetSets(ctx)
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
}

func (i *PokemonGameImporter) GetImportedSetIDs(ctx context.Context) (map[string]bool, error) {
	apiIDs, err := i.setService.GetAllSetAPIIDs(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(apiIDs))
	for _, id := range apiIDs {
		result[id] = true
	}
	return result, nil
}

func (i *PokemonGameImporter) CheckSetInDB(ctx context.Context, apiID string) (bool, bool, error) {
	dbSetID, err := i.setService.GetSetIDByAPIID(ctx, apiID)
	if err == sql.ErrNoRows {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("failed to look up set %s: %w", apiID, err)
	}
	hasCollections, err := i.setService.SetHasUserCollections(ctx, dbSetID)
	if err != nil {
		return true, false, fmt.Errorf("failed to check collections for set %s: %w", apiID, err)
	}
	return true, hasCollections, nil
}

func (i *PokemonGameImporter) ImportSet(ctx context.Context, apiID string) error {
	return i.service.ImportSpecificSets(ctx, []string{apiID})
}

func (i *PokemonGameImporter) DeleteSet(ctx context.Context, apiID string) error {
	return i.service.DeleteSetByAPIID(ctx, apiID)
}

func (i *PokemonGameImporter) ImportAll(ctx context.Context) error {
	return i.service.ImportAllSets(ctx)
}

func (i *PokemonGameImporter) ImportNew(ctx context.Context) error {
	return i.service.ImportNewSets(ctx)
}

func (i *PokemonGameImporter) ImportSpecific(ctx context.Context, apiIDs []string) error {
	return i.service.ImportSpecificSets(ctx, apiIDs)
}
