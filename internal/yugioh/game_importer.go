package yugioh

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

// YuGiOhGameImporter implements gameimporter.GameImporter for Yu-Gi-Oh!.
type YuGiOhGameImporter struct {
	client     *Client
	service    *ImportService
	setService sets.SetService
}

// NewYuGiOhGameImporter creates a new YuGiOhGameImporter.
func NewYuGiOhGameImporter(client *Client, service *ImportService, setService sets.SetService) *YuGiOhGameImporter {
	return &YuGiOhGameImporter{
		client:     client,
		service:    service,
		setService: setService,
	}
}

func (i *YuGiOhGameImporter) FetchSets(ctx context.Context) ([]gameimporter.GameSet, error) {
	ygoSets, err := i.client.GetSets(ctx)
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
}

func (i *YuGiOhGameImporter) GetImportedSetIDs(ctx context.Context) (map[string]bool, error) {
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

func (i *YuGiOhGameImporter) CheckSetInDB(ctx context.Context, apiID string) (bool, bool, error) {
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

func (i *YuGiOhGameImporter) ImportSet(ctx context.Context, apiID string) error {
	return i.service.ImportSpecificSets(ctx, []string{apiID})
}

func (i *YuGiOhGameImporter) DeleteSet(ctx context.Context, apiID string) error {
	return i.service.DeleteSetByAPIID(ctx, apiID)
}

func (i *YuGiOhGameImporter) ImportAll(ctx context.Context) error {
	return i.service.ImportAllSets(ctx)
}

func (i *YuGiOhGameImporter) ImportNew(ctx context.Context) error {
	return i.service.ImportNewSets(ctx)
}

func (i *YuGiOhGameImporter) ImportSpecific(ctx context.Context, apiIDs []string) error {
	return i.service.ImportSpecificSets(ctx, apiIDs)
}
