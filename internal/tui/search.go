package tui

import (
	"fmt"
	"sort"

	"github.com/sahilm/fuzzy"

	"github.com/laiambryant/tui-cardman/internal/model"
)

// cardFieldSource implements fuzzy.Source for a single string field of []model.Card.
type cardFieldSource struct {
	cards []model.Card
	field func(model.Card) string
}

func (s cardFieldSource) String(i int) string { return s.field(s.cards[i]) }
func (s cardFieldSource) Len() int            { return len(s.cards) }

// collectionFieldSource implements fuzzy.Source for a single string field of []model.UserCollection.
type collectionFieldSource struct {
	collections []model.UserCollection
	field       func(model.UserCollection) string
}

func (s collectionFieldSource) String(i int) string { return s.field(s.collections[i]) }
func (s collectionFieldSource) Len() int            { return len(s.collections) }

// FuzzySearchResult holds a card and its best fuzzy match score.
type FuzzySearchResult struct {
	Card  model.Card
	Index int
	Score int
}

// fuzzySearchCards searches Name, Number, and Rarity fields independently,
// merges results keeping the best score per card, and returns sorted by score descending.
func fuzzySearchCards(cards []model.Card, query string) []FuzzySearchResult {
	if query == "" || len(cards) == 0 {
		return nil
	}

	bestScore := make(map[int]int) // index → best score

	fields := []func(model.Card) string{
		func(c model.Card) string { return c.Name },
		func(c model.Card) string { return c.Number },
		func(c model.Card) string { return c.Rarity },
		func(c model.Card) string { return c.Artist },
	}

	for _, field := range fields {
		matches := fuzzy.FindFrom(query, cardFieldSource{cards: cards, field: field})
		for _, m := range matches {
			if existing, ok := bestScore[m.Index]; !ok || m.Score > existing {
				bestScore[m.Index] = m.Score
			}
		}
	}

	results := make([]FuzzySearchResult, 0, len(bestScore))
	for idx, score := range bestScore {
		results = append(results, FuzzySearchResult{
			Card:  cards[idx],
			Index: idx,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// fuzzySearchCollections searches Card.Name, Card.Number, Card.Rarity, Condition, Notes.
func fuzzySearchCollections(collections []model.UserCollection, query string) []model.UserCollection {
	if query == "" || len(collections) == 0 {
		return nil
	}

	bestScore := make(map[int]int)

	fields := []func(model.UserCollection) string{
		func(c model.UserCollection) string {
			if c.Card != nil {
				return c.Card.Name
			}
			return ""
		},
		func(c model.UserCollection) string {
			if c.Card != nil {
				return c.Card.Number
			}
			return ""
		},
		func(c model.UserCollection) string {
			if c.Card != nil {
				return c.Card.Rarity
			}
			return ""
		},
		func(c model.UserCollection) string {
			if c.Card != nil {
				return c.Card.Artist
			}
			return ""
		},
		func(c model.UserCollection) string { return c.Condition },
		func(c model.UserCollection) string { return c.Notes },
	}

	for _, field := range fields {
		matches := fuzzy.FindFrom(query, collectionFieldSource{collections: collections, field: field})
		for _, m := range matches {
			if existing, ok := bestScore[m.Index]; !ok || m.Score > existing {
				bestScore[m.Index] = m.Score
			}
		}
	}

	type scored struct {
		index int
		score int
	}
	ranked := make([]scored, 0, len(bestScore))
	for idx, score := range bestScore {
		ranked = append(ranked, scored{idx, score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	results := make([]model.UserCollection, 0, len(ranked))
	for _, r := range ranked {
		results = append(results, collections[r.index])
	}
	return results
}

// --- Search Cache ---

// SearchCache is a simple query→results cache for fuzzy search.
// No mutex needed because Bubble Tea is single-threaded.
type SearchCache struct {
	entries map[string][]FuzzySearchResult
}

// NewSearchCache creates a new empty cache.
func NewSearchCache() *SearchCache {
	return &SearchCache{entries: make(map[string][]FuzzySearchResult)}
}

// Get returns cached results and whether they were found.
func (c *SearchCache) Get(query string) ([]FuzzySearchResult, bool) {
	r, ok := c.entries[query]
	return r, ok
}

// Put stores results for a query.
func (c *SearchCache) Put(query string, results []FuzzySearchResult) {
	c.entries[query] = results
}

// Invalidate clears the entire cache.
func (c *SearchCache) Invalidate() {
	c.entries = make(map[string][]FuzzySearchResult)
}

// filterCardsByQueryCached checks the cache before computing fuzzy results.
func filterCardsByQueryCached(cards []model.Card, query string, cache *SearchCache) []model.Card {
	if query == "" {
		return cards
	}
	if cache != nil {
		if cached, ok := cache.Get(query); ok {
			return fuzzyResultsToCards(cached)
		}
	}
	results := fuzzySearchCards(cards, query)
	if cache != nil {
		cache.Put(query, results)
	}
	return fuzzyResultsToCards(results)
}

func fuzzyResultsToCards(results []FuzzySearchResult) []model.Card {
	cards := make([]model.Card, len(results))
	for i, r := range results {
		cards[i] = r.Card
	}
	return cards
}

// --- Pagination ---

// Pagination provides page-based slicing for UI tables.
type Pagination struct {
	CurrentPage int
	PageSize    int
	TotalItems  int
}

// NewPagination creates a Pagination with the given page size.
func NewPagination(pageSize int) Pagination {
	return Pagination{PageSize: pageSize}
}

// TotalPages returns the total number of pages.
func (p Pagination) TotalPages() int {
	if p.TotalItems <= 0 || p.PageSize <= 0 {
		return 0
	}
	return (p.TotalItems + p.PageSize - 1) / p.PageSize
}

// NextPage advances to the next page if possible.
func (p *Pagination) NextPage() {
	if p.CurrentPage < p.TotalPages()-1 {
		p.CurrentPage++
	}
}

// PrevPage goes to the previous page if possible.
func (p *Pagination) PrevPage() {
	if p.CurrentPage > 0 {
		p.CurrentPage--
	}
}

// Reset returns to the first page.
func (p *Pagination) Reset() {
	p.CurrentPage = 0
}

// Slice returns (start, end) indices for the current page.
func (p Pagination) Slice() (int, int) {
	if p.TotalItems <= 0 || p.PageSize <= 0 {
		return 0, 0
	}
	start := p.CurrentPage * p.PageSize
	if start >= p.TotalItems {
		start = max((p.TotalPages()-1)*p.PageSize, 0)
	}
	end := start + p.PageSize
	if end > p.TotalItems {
		end = p.TotalItems
	}
	return start, end
}

// StatusText returns a human-readable page indicator.
func (p Pagination) StatusText() string {
	total := p.TotalPages()
	if total <= 1 {
		return fmt.Sprintf("(%d results)", p.TotalItems)
	}
	return fmt.Sprintf("Page %d/%d (%d results)", p.CurrentPage+1, total, p.TotalItems)
}
