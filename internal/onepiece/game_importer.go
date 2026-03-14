package onepiece

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

type OnePieceGameImporter struct {
	client     *Client
	service    *ImportService
	setService sets.SetService
}

func NewOnePieceGameImporter(client *Client, service *ImportService, setService sets.SetService) *OnePieceGameImporter {
	return &OnePieceGameImporter{
		client:     client,
		service:    service,
		setService: setService,
	}
}

func (i *OnePieceGameImporter) FetchSets(ctx context.Context) ([]gameimporter.GameSet, error) {
	opSets, err := i.client.GetSets(ctx)
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
}

func (i *OnePieceGameImporter) GetImportedSetIDs(ctx context.Context) (map[string]bool, error) {
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

func (i *OnePieceGameImporter) GetImportedSetCounts(ctx context.Context) (map[string]int, error) {
	return i.setService.GetAllSetAPIIDsWithCounts(ctx)
}

func (i *OnePieceGameImporter) CheckSetInDB(ctx context.Context, apiID string) (bool, bool, error) {
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

func (i *OnePieceGameImporter) ImportSet(ctx context.Context, apiID string) error {
	return i.service.ImportSpecificSets(ctx, []string{apiID})
}

func (i *OnePieceGameImporter) DeleteSet(ctx context.Context, apiID string) error {
	return i.service.DeleteSetByAPIID(ctx, apiID)
}

func (i *OnePieceGameImporter) ImportAll(ctx context.Context) error {
	return i.service.ImportAllSets(ctx)
}

func (i *OnePieceGameImporter) ImportNew(ctx context.Context) error {
	return i.service.ImportNewSets(ctx)
}

func (i *OnePieceGameImporter) ImportSpecific(ctx context.Context, apiIDs []string) error {
	return i.service.ImportSpecificSets(ctx, apiIDs)
}
