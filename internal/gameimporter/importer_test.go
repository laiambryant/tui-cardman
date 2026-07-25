package gameimporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockSetService struct {
	apiIDs []string
	counts map[string]int
}

func (m *mockSetService) GetSetIDByAPIID(_ context.Context, apiID string) (int64, error) {
	return 0, nil
}

func (m *mockSetService) UpsertSet(_ context.Context, _, _, _ string, _, _ int) (int64, error) {
	return 1, nil
}

func (m *mockSetService) GetAllSetAPIIDs(_ context.Context) ([]string, error) {
	return m.apiIDs, nil
}

func (m *mockSetService) GetAllSetAPIIDsWithCounts(_ context.Context) (map[string]int, error) {
	return m.counts, nil
}

func (m *mockSetService) SetHasUserCollections(_ context.Context, _ int64) (bool, error) {
	return false, nil
}

func TestGetImportedSetIDs(t *testing.T) {
	mock := &mockSetService{apiIDs: []string{"KLD", "AER", "M21"}}
	importer := &Base{SetService: mock}
	ctx := context.Background()

	result, err := importer.GetImportedSetIDs(ctx)

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.True(t, result["KLD"])
	assert.True(t, result["AER"])
	assert.True(t, result["M21"])
	assert.False(t, result["XYZ"])
}

func TestGetImportedSetIDs_Empty(t *testing.T) {
	mock := &mockSetService{apiIDs: []string{}}
	importer := &Base{SetService: mock}
	ctx := context.Background()

	result, err := importer.GetImportedSetIDs(ctx)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetImportedSetCounts(t *testing.T) {
	mock := &mockSetService{counts: map[string]int{"KLD": 264, "AER": 184}}
	importer := &Base{SetService: mock}
	ctx := context.Background()

	result, err := importer.GetImportedSetCounts(ctx)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 264, result["KLD"])
	assert.Equal(t, 184, result["AER"])
}

func TestGetImportedSetCounts_Empty(t *testing.T) {
	mock := &mockSetService{counts: map[string]int{}}
	importer := &Base{SetService: mock}
	ctx := context.Background()

	result, err := importer.GetImportedSetCounts(ctx)

	assert.NoError(t, err)
	assert.Empty(t, result)
}
