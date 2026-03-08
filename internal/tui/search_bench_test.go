package tui

import (
	"fmt"
	"testing"

	"github.com/laiambryant/tui-cardman/internal/model"
)

// generateBenchCards creates n cards with realistic-looking names.
func generateBenchCards(n int) []model.Card {
	names := []string{
		"Pikachu", "Charizard", "Bulbasaur", "Mewtwo", "Eevee",
		"Snorlax", "Gengar", "Dragonite", "Lucario", "Gardevoir",
		"Blastoise", "Venusaur", "Jigglypuff", "Machamp", "Alakazam",
		"Gyarados", "Lapras", "Arcanine", "Umbreon", "Espeon",
	}
	rarities := []string{"Common", "Uncommon", "Rare", "Ultra Rare", "Secret Rare"}
	cards := make([]model.Card, n)
	for i := range n {
		cards[i] = model.Card{
			ID:     int64(i + 1),
			Name:   fmt.Sprintf("%s V%d", names[i%len(names)], i/len(names)),
			Number: fmt.Sprintf("%03d", i+1),
			Rarity: rarities[i%len(rarities)],
		}
	}
	return cards
}

var benchCardCounts = []int{100, 1000, 5000, 17000}

// BenchmarkSubstringSearch benchmarks the old substring filter.
func BenchmarkSubstringSearch(b *testing.B) {
	queries := []string{"pika", "charz", "ultra"}
	for _, count := range benchCardCounts {
		cards := generateBenchCards(count)
		for _, q := range queries {
			b.Run(fmt.Sprintf("cards=%d/q=%s", count, q), func(b *testing.B) {
				for range b.N {
					_ = filterCardsByQuerySubstring(cards, q)
				}
			})
		}
	}
}

// BenchmarkFuzzySearch benchmarks the new fuzzy search.
func BenchmarkFuzzySearch(b *testing.B) {
	queries := []string{"pika", "charz", "ultra"}
	for _, count := range benchCardCounts {
		cards := generateBenchCards(count)
		for _, q := range queries {
			b.Run(fmt.Sprintf("cards=%d/q=%s", count, q), func(b *testing.B) {
				for range b.N {
					_ = fuzzySearchCards(cards, q)
				}
			})
		}
	}
}

// BenchmarkFuzzySearchCached benchmarks fuzzy search with cache hits.
func BenchmarkFuzzySearchCached(b *testing.B) {
	queries := []string{"pika", "charz", "ultra"}
	for _, count := range benchCardCounts {
		cards := generateBenchCards(count)
		for _, q := range queries {
			b.Run(fmt.Sprintf("cards=%d/q=%s", count, q), func(b *testing.B) {
				cache := NewSearchCache()
				// Prime the cache
				_ = filterCardsByQueryCached(cards, q, cache)
				b.ResetTimer()
				for range b.N {
					_ = filterCardsByQueryCached(cards, q, cache)
				}
			})
		}
	}
}
