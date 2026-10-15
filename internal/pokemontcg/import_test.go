package pokemontcg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/services/pokemoncard"
	"github.com/laiambryant/tui-cardman/internal/testutil"
)

type mockClient struct {
	sets  []Set
	pages map[string][][]Card // set ID -> pages of cards
}

func (m *mockClient) GetSets(context.Context) ([]Set, error) { return m.sets, nil }

func (m *mockClient) GetCardsForSet(_ context.Context, setID string, page int) (*PaginatedResponse, []Card, error) {
	pages := m.pages[setID]
	if page > len(pages) {
		return &PaginatedResponse{Page: page}, nil, nil
	}
	return &PaginatedResponse{Page: page, HasMore: page < len(pages)}, pages[page-1], nil
}

func TestNewImporter_RequiresPokemonGame(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	_, err := newImporter(db, &mockClient{})
	assert.Error(t, err)
}

func TestImporter_StoresCardsPricesAndDetails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	testutil.ApplyTestMigrations(t, db, "../db/migrations")

	hp := Card{
		ID: "base1-4", Name: "Charizard", Number: "4", Rarity: "Rare Holo", Artist: "Mitsuhiro Arita",
		HP: 120, Category: "Pokemon", Stage: "Stage2", Types: []string{"Fire"},
		Attacks: []CardAttack{{Name: "Fire Spin", Cost: []string{"Fire"}, Damage: "100"}},
		TCGPlayer: &TCGPlayerPrices{Prices: map[string]TCGPlayerPrice{
			"normal":  {Market: 300},
			"reverse": {Market: 450},
		}},
		CardMarket: &CardMarketPrices{Prices: map[string]CardMarketPrice{
			"reverseHolo": {Avg: 9, Trend: 9},
			"holo":        {Avg: 5, Trend: 6},
		}},
	}
	client := &mockClient{
		sets: []Set{{ID: "base1", Name: "Base Set", PtcgoCode: "base1", PrintedTotal: 102, Total: 102}},
		pages: map[string][][]Card{
			"base1": {{hp}, {{ID: "base1-5", Name: "Clefairy", Number: "5"}}},
		},
	}
	imp, err := newImporter(db, client)
	require.NoError(t, err)

	require.NoError(t, imp.ImportSet(context.Background(), "base1"))

	var cardID int64
	var name, number, artist string
	require.NoError(t, db.QueryRow(`SELECT id, name, number, artist FROM cards WHERE api_id = 'base1-4'`).
		Scan(&cardID, &name, &number, &artist))
	assert.Equal(t, "Charizard", name)
	assert.Equal(t, "4", number)
	assert.Equal(t, "Mitsuhiro Arita", artist)

	var cardCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM cards`).Scan(&cardCount))
	assert.Equal(t, 2, cardCount, "cards from every page are imported")

	var tcgRows int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM prices_tcgplayer WHERE card_id = ?`, cardID).Scan(&tcgRows))
	assert.Equal(t, 2, tcgRows)

	var cmRows int
	var avg float64
	require.NoError(t, db.QueryRow(`SELECT COUNT(*), MAX(avg_price) FROM prices_cardmarket WHERE card_id = ?`, cardID).Scan(&cmRows, &avg))
	assert.Equal(t, 1, cmRows, "only one CardMarket variant is stored")
	assert.InDelta(t, 5, avg, 0.001, "holo is preferred over reverse holo")

	pc, err := pokemoncard.NewPokemonCardService(db).GetByCardID(cardID)
	require.NoError(t, err)
	assert.Equal(t, 120, pc.HP)
	assert.Equal(t, []string{"Fire"}, pc.Types)
	require.Len(t, pc.Attacks, 1)
	assert.Equal(t, "Fire Spin", pc.Attacks[0].Name)

	// Re-importing replaces prices instead of accumulating them.
	require.NoError(t, imp.ImportSet(context.Background(), "base1"))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM prices_tcgplayer WHERE card_id = ?`, cardID).Scan(&tcgRows))
	assert.Equal(t, 2, tcgRows)
}

func TestPrimaryCardMarketPrice(t *testing.T) {
	_, ok := primaryCardMarketPrice(nil)
	assert.False(t, ok)

	_, ok = primaryCardMarketPrice(&CardMarketPrices{})
	assert.False(t, ok)

	p, ok := primaryCardMarketPrice(&CardMarketPrices{Prices: map[string]CardMarketPrice{
		"holo": {Avg: 2}, "normal": {Avg: 1}, "reverseHolo": {Avg: 3},
	}})
	assert.True(t, ok)
	assert.InDelta(t, 1, p.Avg, 0.001)
}
