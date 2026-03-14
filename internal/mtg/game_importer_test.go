package mtg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveHasMore_AllCases(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		ps         int
		fetched    int
		totalCount int
		want       bool
	}{
		{"first page with total, more remain", 1, 100, 100, 269, true},
		{"second page with total, more remain", 2, 100, 100, 269, true},
		{"last page with total", 3, 100, 69, 269, false},
		{"exactly one page", 1, 100, 100, 100, false},
		{"small set", 1, 100, 42, 42, false},
		{"no header full page", 1, 100, 100, 0, true},
		{"no header partial page", 1, 100, 57, 0, false},
		{"no header empty page", 2, 100, 0, 0, false},
		{"single card total", 1, 100, 1, 1, false},
		{"page size 1", 1, 1, 1, 5, true},
		{"page size 1 last", 5, 1, 1, 5, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveHasMore(tc.page, tc.ps, tc.fetched, tc.totalCount)
			assert.Equal(t, tc.want, got)
		})
	}
}

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
	importer := &MTGGameImporter{setService: mock}
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
	importer := &MTGGameImporter{setService: mock}
	ctx := context.Background()

	result, err := importer.GetImportedSetIDs(ctx)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetImportedSetCounts(t *testing.T) {
	mock := &mockSetService{counts: map[string]int{"KLD": 264, "AER": 184}}
	importer := &MTGGameImporter{setService: mock}
	ctx := context.Background()

	result, err := importer.GetImportedSetCounts(ctx)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 264, result["KLD"])
	assert.Equal(t, 184, result["AER"])
}

func TestGetImportedSetCounts_Empty(t *testing.T) {
	mock := &mockSetService{counts: map[string]int{}}
	importer := &MTGGameImporter{setService: mock}
	ctx := context.Background()

	result, err := importer.GetImportedSetCounts(ctx)

	assert.NoError(t, err)
	assert.Empty(t, result)
}
