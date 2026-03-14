package mtg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

type MTGGameImporter struct {
	client     *Client
	service    *ImportService
	setService sets.SetService
}

func NewMTGGameImporter(client *Client, service *ImportService, setService sets.SetService) *MTGGameImporter {
	return &MTGGameImporter{
		client:     client,
		service:    service,
		setService: setService,
	}
}

func (i *MTGGameImporter) FetchSets(ctx context.Context) ([]gameimporter.GameSet, error) {
	mtgSets, err := i.client.GetSets(ctx)
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
}

func (i *MTGGameImporter) GetImportedSetIDs(ctx context.Context) (map[string]bool, error) {
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

func (i *MTGGameImporter) GetImportedSetCounts(ctx context.Context) (map[string]int, error) {
	return i.setService.GetAllSetAPIIDsWithCounts(ctx)
}

func (i *MTGGameImporter) CheckSetInDB(ctx context.Context, apiID string) (bool, bool, error) {
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

func (i *MTGGameImporter) ImportSet(ctx context.Context, apiID string) error {
	return i.service.ImportSpecificSets(ctx, []string{apiID})
}

func (i *MTGGameImporter) DeleteSet(ctx context.Context, apiID string) error {
	return i.service.DeleteSetByAPIID(ctx, apiID)
}

func (i *MTGGameImporter) ImportAll(ctx context.Context) error {
	return i.service.ImportAllSets(ctx)
}

func (i *MTGGameImporter) ImportNew(ctx context.Context) error {
	return i.service.ImportNewSets(ctx)
}

func (i *MTGGameImporter) ImportSpecific(ctx context.Context, apiIDs []string) error {
	return i.service.ImportSpecificSets(ctx, apiIDs)
}
