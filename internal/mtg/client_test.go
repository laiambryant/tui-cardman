package mtg

import (
	"testing"
)

func TestResolveHasMore(t *testing.T) {
	ps := 100

	tests := []struct {
		name       string
		page       int
		fetched    int
		totalCount int
		want       bool
	}{
		{
			name: "first page, more remain (total authoritative)",
			page: 1, fetched: 100, totalCount: 269,
			want: true,
		},
		{
			name: "second page, more remain",
			page: 2, fetched: 100, totalCount: 269,
			want: true,
		},
		{
			name: "third page, none remain (300 > 269)",
			page: 3, fetched: 69, totalCount: 269,
			want: false,
		},
		{
			name: "exactly one full page (100 == 100)",
			page: 1, fetched: 100, totalCount: 100,
			want: false,
		},
		{
			name: "small set fits in one page",
			page: 1, fetched: 42, totalCount: 42,
			want: false,
		},
		{
			name: "total reported as 0 via header (treat as absent)",
			page: 1, fetched: 100, totalCount: 0,
			want: true, // full page → assume more
		},
		{
			name: "no header, full page returned → assume more",
			page: 1, fetched: 100, totalCount: 0,
			want: true,
		},
		{
			name: "no header, partial page → last page",
			page: 1, fetched: 57, totalCount: 0,
			want: false,
		},
		{
			name: "no header, empty page → done",
			page: 2, fetched: 0, totalCount: 0,
			want: false,
		},
		{
			name: "no header, exactly full page on page 2 → assume more",
			page: 2, fetched: 100, totalCount: 0,
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveHasMore(tc.page, ps, tc.fetched, tc.totalCount)
			if got != tc.want {
				t.Errorf("resolveHasMore(page=%d, ps=%d, fetched=%d, total=%d) = %v, want %v",
					tc.page, ps, tc.fetched, tc.totalCount, got, tc.want)
			}
		})
	}
}
