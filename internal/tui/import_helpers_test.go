package tui

import (
	"testing"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/stretchr/testify/assert"
)

func TestResolveSetCardCount_APITotalPresent(t *testing.T) {
	m := ImportModel{
		databaseSetIDs:    map[string]bool{},
		databaseSetCounts: map[string]int{},
	}
	set := gameimporter.GameSet{APIID: "base1", Total: 102}

	result := m.resolveSetCardCount(set)

	assert.Equal(t, "102 cards", result)
}

func TestResolveSetCardCount_DBCountPresent(t *testing.T) {
	m := ImportModel{
		databaseSetIDs:    map[string]bool{"KLD": true},
		databaseSetCounts: map[string]int{"KLD": 264},
	}
	set := gameimporter.GameSet{APIID: "KLD", Total: 0}

	result := m.resolveSetCardCount(set)

	assert.Equal(t, "264 cards", result)
}

func TestResolveSetCardCount_ImportedButZeroCounts(t *testing.T) {
	m := ImportModel{
		databaseSetIDs:    map[string]bool{"KLD": true},
		databaseSetCounts: map[string]int{"KLD": 0},
	}
	set := gameimporter.GameSet{APIID: "KLD", Total: 0}

	result := m.resolveSetCardCount(set)

	assert.Equal(t, "imported", result)
}

func TestResolveSetCardCount_NotImported(t *testing.T) {
	m := ImportModel{
		databaseSetIDs:    map[string]bool{},
		databaseSetCounts: map[string]int{},
	}
	set := gameimporter.GameSet{APIID: "KLD", Total: 0}

	result := m.resolveSetCardCount(set)

	assert.Equal(t, "? cards", result)
}

func TestResolveSetCardCount_APITotalOverridesDBCount(t *testing.T) {
	m := ImportModel{
		databaseSetIDs:    map[string]bool{"base1": true},
		databaseSetCounts: map[string]int{"base1": 102},
	}
	set := gameimporter.GameSet{APIID: "base1", Total: 120}

	result := m.resolveSetCardCount(set)

	assert.Equal(t, "120 cards", result)
}

func TestResolveSetCardCount_NilMaps(t *testing.T) {
	m := ImportModel{
		databaseSetIDs:    nil,
		databaseSetCounts: nil,
	}
	set := gameimporter.GameSet{APIID: "KLD", Total: 0}

	result := m.resolveSetCardCount(set)

	assert.Equal(t, "? cards", result)
}

func TestGetSetStatusIcon_Imported(t *testing.T) {
	assert.Equal(t, "[x]", getSetStatusIcon(true))
}

func TestGetSetStatusIcon_NotImported(t *testing.T) {
	assert.Equal(t, "[ ]", getSetStatusIcon(false))
}

func TestCalculatePaginationRange(t *testing.T) {
	tests := []struct {
		name          string
		cursor        int
		totalItems    int
		itemsPerPage  int
		expectedStart int
		expectedEnd   int
	}{
		{"first page", 0, 100, 15, 0, 15},
		{"middle", 50, 100, 15, 43, 58},
		{"near end", 95, 100, 15, 88, 100},
		{"small list", 3, 5, 15, 0, 5},
		{"at boundary", 14, 100, 15, 7, 22},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end := calculatePaginationRange(tc.cursor, tc.totalItems, tc.itemsPerPage)
			assert.Equal(t, tc.expectedStart, start)
			assert.Equal(t, tc.expectedEnd, end)
		})
	}
}

func TestCalculateProgressPercentage(t *testing.T) {
	tests := []struct {
		name      string
		completed int
		total     int
		expected  int
	}{
		{"zero", 0, 10, 0},
		{"half", 5, 10, 50},
		{"complete", 10, 10, 100},
		{"quarter", 1, 4, 25},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateProgressPercentage(tc.completed, tc.total)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCreateProgressBar(t *testing.T) {
	bar := createProgressBar(5, 10, 20)
	assert.Equal(t, 20, len([]rune(bar)))
	assert.Contains(t, bar, "█")
	assert.Contains(t, bar, "░")
}

func TestCreateProgressBar_Full(t *testing.T) {
	bar := createProgressBar(10, 10, 10)
	assert.Equal(t, "██████████", bar)
}

func TestCreateProgressBar_Empty(t *testing.T) {
	bar := createProgressBar(0, 10, 10)
	assert.Equal(t, "░░░░░░░░░░", bar)
}

func TestSplitImportPanelWidths_Normal(t *testing.T) {
	left, right := splitImportPanelWidths(80)
	assert.Greater(t, left, 0)
	assert.Greater(t, right, 0)
	assert.Greater(t, left, right)
}

func TestSplitImportPanelWidths_VerySmall(t *testing.T) {
	left, right := splitImportPanelWidths(10)
	assert.GreaterOrEqual(t, left, 5)
	assert.GreaterOrEqual(t, right, 5)
}

func TestGetCursorPrefix(t *testing.T) {
	assert.Equal(t, "> ", getCursorPrefix(true))
	assert.Equal(t, "  ", getCursorPrefix(false))
}
