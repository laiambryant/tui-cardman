package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/laiambryant/tui-cardman/internal/model"
)

func makeCards(names ...string) []model.Card {
	cards := make([]model.Card, len(names))
	for i, name := range names {
		cards[i] = model.Card{ID: int64(i + 1), Name: name, Number: "", Rarity: "Common"}
	}
	return cards
}

// --- fuzzySearchCards tests ---

func TestFuzzySearchCards_EmptyQuery(t *testing.T) {
	cards := makeCards("Pikachu", "Charizard")
	results := fuzzySearchCards(cards, "")
	assert.Nil(t, results)
}

func TestFuzzySearchCards_ExactMatch(t *testing.T) {
	cards := makeCards("Pikachu", "Charizard", "Bulbasaur")
	results := fuzzySearchCards(cards, "Charizard")
	assert.NotEmpty(t, results)
	assert.Equal(t, "Charizard", results[0].Card.Name)
}

func TestFuzzySearchCards_TypoMatch(t *testing.T) {
	cards := makeCards("Pikachu", "Charizard", "Bulbasaur")
	results := fuzzySearchCards(cards, "charzard")
	// Should still find Charizard with a typo
	found := false
	for _, r := range results {
		if r.Card.Name == "Charizard" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected Charizard to be found with typo query 'charzard'")
}

func TestFuzzySearchCards_MultiField(t *testing.T) {
	cards := []model.Card{
		{ID: 1, Name: "Pikachu", Number: "025", Rarity: "Common"},
		{ID: 2, Name: "Charizard", Number: "006", Rarity: "Rare"},
	}
	// Search by number
	results := fuzzySearchCards(cards, "025")
	assert.NotEmpty(t, results)
	assert.Equal(t, "Pikachu", results[0].Card.Name)

	// Search by rarity
	results = fuzzySearchCards(cards, "Rare")
	assert.NotEmpty(t, results)
	found := false
	for _, r := range results {
		if r.Card.Name == "Charizard" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestFuzzySearchCards_RankingOrder(t *testing.T) {
	cards := makeCards("Pikachu", "Pikach", "Pika", "Unrelated")
	results := fuzzySearchCards(cards, "Pikachu")
	assert.NotEmpty(t, results)
	// Exact match should be first
	assert.Equal(t, "Pikachu", results[0].Card.Name)
}

func TestFuzzySearchCards_EmptyCards(t *testing.T) {
	results := fuzzySearchCards(nil, "test")
	assert.Nil(t, results)
}

// --- fuzzySearchCollections tests ---

func TestFuzzySearchCollections_EmptyQuery(t *testing.T) {
	collections := []model.UserCollection{{Card: &model.Card{Name: "Pikachu"}}}
	results := fuzzySearchCollections(collections, "")
	assert.Nil(t, results)
}

func TestFuzzySearchCollections_ByCardName(t *testing.T) {
	collections := []model.UserCollection{
		{Card: &model.Card{Name: "Pikachu", Number: "025", Rarity: "Common"}},
		{Card: &model.Card{Name: "Charizard", Number: "006", Rarity: "Rare"}},
	}
	results := fuzzySearchCollections(collections, "Pikachu")
	assert.NotEmpty(t, results)
	assert.Equal(t, "Pikachu", results[0].Card.Name)
}

func TestFuzzySearchCollections_ByCondition(t *testing.T) {
	collections := []model.UserCollection{
		{Card: &model.Card{Name: "Pikachu"}, Condition: "Mint"},
		{Card: &model.Card{Name: "Charizard"}, Condition: "Played"},
	}
	results := fuzzySearchCollections(collections, "Mint")
	assert.NotEmpty(t, results)
	assert.Equal(t, "Pikachu", results[0].Card.Name)
}

// --- Pagination tests ---

func TestPagination_TotalPages(t *testing.T) {
	p := NewPagination(10)
	p.TotalItems = 25
	assert.Equal(t, 3, p.TotalPages())

	p.TotalItems = 10
	assert.Equal(t, 1, p.TotalPages())

	p.TotalItems = 0
	assert.Equal(t, 0, p.TotalPages())
}

func TestPagination_Slice(t *testing.T) {
	p := NewPagination(10)
	p.TotalItems = 25

	start, end := p.Slice()
	assert.Equal(t, 0, start)
	assert.Equal(t, 10, end)

	p.NextPage()
	start, end = p.Slice()
	assert.Equal(t, 10, start)
	assert.Equal(t, 20, end)

	p.NextPage()
	start, end = p.Slice()
	assert.Equal(t, 20, start)
	assert.Equal(t, 25, end)
}

func TestPagination_Boundaries(t *testing.T) {
	p := NewPagination(10)
	p.TotalItems = 25

	// Can't go before first page
	p.PrevPage()
	assert.Equal(t, 0, p.CurrentPage)

	// Go to last page and try to go beyond
	p.CurrentPage = 2
	p.NextPage()
	assert.Equal(t, 2, p.CurrentPage)
}

func TestPagination_EmptyData(t *testing.T) {
	p := NewPagination(10)
	p.TotalItems = 0
	start, end := p.Slice()
	assert.Equal(t, 0, start)
	assert.Equal(t, 0, end)
	assert.Equal(t, "(0 results)", p.StatusText())
}

func TestPagination_SinglePage(t *testing.T) {
	p := NewPagination(50)
	p.TotalItems = 10
	assert.Equal(t, 1, p.TotalPages())
	start, end := p.Slice()
	assert.Equal(t, 0, start)
	assert.Equal(t, 10, end)
	assert.Equal(t, "(10 results)", p.StatusText())
}

func TestPagination_Reset(t *testing.T) {
	p := NewPagination(10)
	p.TotalItems = 50
	p.NextPage()
	p.NextPage()
	assert.Equal(t, 2, p.CurrentPage)
	p.Reset()
	assert.Equal(t, 0, p.CurrentPage)
}

func TestPagination_StatusText(t *testing.T) {
	p := NewPagination(10)
	p.TotalItems = 25
	assert.Equal(t, "Page 1/3 (25 results)", p.StatusText())
	p.NextPage()
	assert.Equal(t, "Page 2/3 (25 results)", p.StatusText())
}

// --- SearchCache tests ---

func TestSearchCache_HitMiss(t *testing.T) {
	cache := NewSearchCache()
	_, ok := cache.Get("test")
	assert.False(t, ok)

	results := []FuzzySearchResult{{Card: model.Card{Name: "Pikachu"}, Score: 10}}
	cache.Put("test", results)
	got, ok := cache.Get("test")
	assert.True(t, ok)
	assert.Equal(t, 1, len(got))
	assert.Equal(t, "Pikachu", got[0].Card.Name)
}

func TestSearchCache_Invalidate(t *testing.T) {
	cache := NewSearchCache()
	cache.Put("test", []FuzzySearchResult{{Card: model.Card{Name: "Pikachu"}}})
	cache.Invalidate()
	_, ok := cache.Get("test")
	assert.False(t, ok)
}
