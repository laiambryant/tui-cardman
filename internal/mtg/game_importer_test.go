package mtg

import (
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
