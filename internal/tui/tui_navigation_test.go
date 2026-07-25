package tui

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/model"
)

type navigationTestImporter struct{}

func (navigationTestImporter) FetchSets(context.Context) ([]gameimporter.GameSet, error) {
	return nil, nil
}

func (navigationTestImporter) GetImportedSetIDs(context.Context) (map[string]bool, error) {
	return nil, nil
}

func (navigationTestImporter) GetImportedSetCounts(context.Context) (map[string]int, error) {
	return nil, nil
}

func (navigationTestImporter) CheckSetInDB(context.Context, string) (bool, bool, error) {
	return false, false, nil
}

func (navigationTestImporter) ImportSet(context.Context, string) error {
	return nil
}

func (navigationTestImporter) DeleteSet(context.Context, string) error {
	return nil
}

func (navigationTestImporter) ImportAll(context.Context) error {
	return nil
}

func (navigationTestImporter) ImportNew(context.Context) error {
	return nil
}

func (navigationTestImporter) ImportSpecific(context.Context, []string) error {
	return nil
}

func TestHandleMainScreenSelectUsesHighlightedGameForImport(t *testing.T) {
	games := []model.CardGame{
		{ID: 1, Name: "Pokemon"},
		{ID: 2, Name: "Magic: The Gathering"},
	}
	importer := navigationTestImporter{}
	m := &Model{
		screen:           ScreenMain,
		cardGames:        games,
		cursor:           1,
		selectedGame:     &games[0],
		mainFocusPanel:   1,
		rightPanelCursor: int(MenuImportSets),
		importers: map[int64]gameimporter.GameImporter{
			games[0].ID: importer,
			games[1].ID: importer,
		},
		styleManager: NewStyleManager(),
	}

	updated, _ := m.handleMainScreenSelect()
	updatedModel, ok := updated.(*Model)
	require.True(t, ok)
	require.NotNil(t, updatedModel.importModel)
	require.NotNil(t, updatedModel.importModel.selectedCardGame)

	assert.Equal(t, ScreenImport, updatedModel.screen)
	assert.Equal(t, games[1].ID, updatedModel.selectedGame.ID)
	assert.Equal(t, games[1].ID, updatedModel.importModel.selectedCardGame.ID)
}
