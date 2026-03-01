package deck

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/testutil"
)

func setupDeckTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := testutil.SetupTestDB(t)
	testutil.ApplyTestMigrations(t, db, "../../db/migrations")
	return db
}

// seedDeckTestData inserts a user, card game, set and cards needed by deck tests.
// Returns (userID, cardGameID, cardID1, cardID2).
func seedDeckTestData(t *testing.T, db *sql.DB) (userID, cardGameID, cardID1, cardID2 int64) {
	t.Helper()

	// Grab the Pokemon card game inserted by migrations
	err := db.QueryRow(`SELECT id FROM card_games WHERE name = 'Pokemon'`).Scan(&cardGameID)
	require.NoError(t, err)

	// Insert a test user
	res, err := db.Exec(`
		INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active)
		VALUES ('Test', 'User', 'deck@example.com', 'hash', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
	`)
	require.NoError(t, err)
	userID, err = res.LastInsertId()
	require.NoError(t, err)

	// Insert a set (real schema: api_id, code, name, printed_total, total, symbol_url, logo_url, updated_at)
	var setID int64
	res, err = db.Exec(`INSERT INTO sets (api_id, name) VALUES ('base1', 'Base Set')`)
	require.NoError(t, err)
	setID, err = res.LastInsertId()
	require.NoError(t, err)

	// Insert two cards
	res, err = db.Exec(`
		INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, artist, is_placeholder, created_at, updated_at)
		VALUES (?, ?, 'base1-4', 'Charizard', 'Rare Holo', '4', 'Mitsuhiro Arita', 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, cardGameID, setID)
	require.NoError(t, err)
	cardID1, err = res.LastInsertId()
	require.NoError(t, err)

	res, err = db.Exec(`
		INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, artist, is_placeholder, created_at, updated_at)
		VALUES (?, ?, 'base1-6', 'Blastoise', 'Rare Holo', '6', 'Ken Sugimori', 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, cardGameID, setID)
	require.NoError(t, err)
	cardID2, err = res.LastInsertId()
	require.NoError(t, err)

	return
}

func TestNewDeckService(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	assert.NotNil(t, svc)
	assert.IsType(t, &DeckServiceImpl{}, svc)
}

func TestCreateDeck(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()

	userID, cardGameID, _, _ := seedDeckTestData(t, db)

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "My Deck", "Standard")
	require.NoError(t, err)
	require.NotNil(t, deck)

	assert.Positive(t, deck.ID)
	assert.Equal(t, userID, deck.UserID)
	assert.Equal(t, cardGameID, deck.CardGameID)
	assert.Equal(t, "My Deck", deck.Name)
	assert.Equal(t, "Standard", deck.Format)
}

func TestCreateDeck_MultipleDecks(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedDeckTestData(t, db)

	deck1, err := svc.CreateDeck(ctx, userID, cardGameID, "Deck Alpha", "Standard")
	require.NoError(t, err)

	deck2, err := svc.CreateDeck(ctx, userID, cardGameID, "Deck Beta", "Expanded")
	require.NoError(t, err)

	assert.NotEqual(t, deck1.ID, deck2.ID)
}

func TestGetDecksByUserAndGame(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedDeckTestData(t, db)

	_, err := svc.CreateDeck(ctx, userID, cardGameID, "Deck A", "Standard")
	require.NoError(t, err)
	_, err = svc.CreateDeck(ctx, userID, cardGameID, "Deck B", "Expanded")
	require.NoError(t, err)

	decks, err := svc.GetDecksByUserAndGame(userID, cardGameID)
	require.NoError(t, err)
	assert.Len(t, decks, 2)

	// Results are ordered by name ASC
	assert.Equal(t, "Deck A", decks[0].Name)
	assert.Equal(t, "Deck B", decks[1].Name)
}

func TestGetDecksByUserAndGame_Empty(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	userID, cardGameID, _, _ := seedDeckTestData(t, db)

	decks, err := svc.GetDecksByUserAndGame(userID, cardGameID)
	require.NoError(t, err)
	assert.Empty(t, decks)
}

func TestGetDecksByUserAndGame_IsolatedByUser(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedDeckTestData(t, db)

	// Create a second user
	res, err := db.Exec(`
		INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active)
		VALUES ('Other', 'User', 'other@example.com', 'hash', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
	`)
	require.NoError(t, err)
	otherUserID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = svc.CreateDeck(ctx, userID, cardGameID, "User1 Deck", "Standard")
	require.NoError(t, err)
	_, err = svc.CreateDeck(ctx, otherUserID, cardGameID, "User2 Deck", "Standard")
	require.NoError(t, err)

	decks, err := svc.GetDecksByUserAndGame(userID, cardGameID)
	require.NoError(t, err)
	assert.Len(t, decks, 1)
	assert.Equal(t, "User1 Deck", decks[0].Name)
}

func TestGetDeckByID(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedDeckTestData(t, db)

	created, err := svc.CreateDeck(ctx, userID, cardGameID, "My Deck", "Standard")
	require.NoError(t, err)

	fetched, err := svc.GetDeckByID(created.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched)

	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, created.Name, fetched.Name)
	assert.Equal(t, created.Format, fetched.Format)
	assert.Equal(t, created.UserID, fetched.UserID)
}

func TestGetDeckByID_NotFound(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)

	_, err := svc.GetDeckByID(999999)
	assert.Error(t, err)
}

func TestUpdateDeck(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedDeckTestData(t, db)

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "Old Name", "Standard")
	require.NoError(t, err)

	err = svc.UpdateDeck(ctx, deck.ID, "New Name", "Expanded")
	require.NoError(t, err)

	updated, err := svc.GetDeckByID(deck.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "Expanded", updated.Format)
}

func TestDeleteDeck(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedDeckTestData(t, db)

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "To Delete", "Standard")
	require.NoError(t, err)

	err = svc.DeleteDeck(ctx, deck.ID)
	require.NoError(t, err)

	_, err = svc.GetDeckByID(deck.ID)
	assert.Error(t, err)
}

func TestDeleteDeck_NonExistent(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()

	// Deleting a non-existent deck should not error (DELETE affects 0 rows, no error)
	err := svc.DeleteDeck(ctx, 999999)
	assert.NoError(t, err)
}

func TestUpsertDeckCardBatch_Add(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()
	userID, cardGameID, cardID1, cardID2 := seedDeckTestData(t, db)

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "Test Deck", "Standard")
	require.NoError(t, err)

	updates := map[int64]int{
		cardID1: 3,
		cardID2: 2,
	}
	err = svc.UpsertDeckCardBatch(ctx, deck.ID, updates)
	require.NoError(t, err)

	quantities, err := svc.GetAllQuantitiesForDeck(deck.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, quantities[cardID1])
	assert.Equal(t, 2, quantities[cardID2])
}

func TestUpsertDeckCardBatch_Update(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()
	userID, cardGameID, cardID1, _ := seedDeckTestData(t, db)

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "Test Deck", "Standard")
	require.NoError(t, err)

	err = svc.UpsertDeckCardBatch(ctx, deck.ID, map[int64]int{cardID1: 2})
	require.NoError(t, err)

	// Update to a new quantity
	err = svc.UpsertDeckCardBatch(ctx, deck.ID, map[int64]int{cardID1: 4})
	require.NoError(t, err)

	quantities, err := svc.GetAllQuantitiesForDeck(deck.ID)
	require.NoError(t, err)
	assert.Equal(t, 4, quantities[cardID1])
}

func TestUpsertDeckCardBatch_ZeroRemovesCard(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()
	userID, cardGameID, cardID1, cardID2 := seedDeckTestData(t, db)

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "Test Deck", "Standard")
	require.NoError(t, err)

	err = svc.UpsertDeckCardBatch(ctx, deck.ID, map[int64]int{cardID1: 2, cardID2: 3})
	require.NoError(t, err)

	// Setting quantity to 0 should remove the card
	err = svc.UpsertDeckCardBatch(ctx, deck.ID, map[int64]int{cardID1: 0})
	require.NoError(t, err)

	quantities, err := svc.GetAllQuantitiesForDeck(deck.ID)
	require.NoError(t, err)
	_, exists := quantities[cardID1]
	assert.False(t, exists, "card with quantity 0 should be removed")
	assert.Equal(t, 3, quantities[cardID2])
}

func TestGetAllQuantitiesForDeck_Empty(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedDeckTestData(t, db)

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "Empty Deck", "Standard")
	require.NoError(t, err)

	quantities, err := svc.GetAllQuantitiesForDeck(deck.ID)
	require.NoError(t, err)
	assert.Empty(t, quantities)
}

// --- ValidateDeck tests ---

func TestValidateDeck_Valid60Cards(t *testing.T) {
	svc := NewDeckService(nil) // ValidateDeck is pure, no DB needed

	cards := make([]model.Card, 10)
	for i := range cards {
		cards[i] = model.Card{ID: int64(i + 1), Name: "Unique Card " + string(rune('A'+i))}
	}

	quantities := make(map[int64]int)
	for _, c := range cards {
		quantities[c.ID] = 6 // 10 cards × 6 = 60 (but 6 > 4, so expect duplicate errors)
	}

	errs := svc.ValidateDeck(cards, quantities)
	// Should only have duplicate errors, not count error
	hasCountErr := false
	for _, e := range errs {
		if e.Type == "card_count" {
			hasCountErr = true
		}
	}
	assert.False(t, hasCountErr)
}

func TestValidateDeck_WrongCount(t *testing.T) {
	svc := NewDeckService(nil)

	cards := []model.Card{
		{ID: 1, Name: "Pikachu"},
	}
	quantities := map[int64]int{1: 4} // 4 cards, not 60

	errs := svc.ValidateDeck(cards, quantities)
	require.Len(t, errs, 1)
	assert.Equal(t, "card_count", errs[0].Type)
	assert.Contains(t, errs[0].Message, "4")
}

func TestValidateDeck_Exactly60(t *testing.T) {
	svc := NewDeckService(nil)

	// 15 distinct cards × 4 copies each = 60
	cards := make([]model.Card, 15)
	quantities := make(map[int64]int)
	for i := range cards {
		cards[i] = model.Card{ID: int64(i + 1), Name: "Card " + string(rune('A'+i))}
		quantities[int64(i+1)] = 4
	}

	errs := svc.ValidateDeck(cards, quantities)
	assert.Empty(t, errs)
}

func TestValidateDeck_DuplicateViolation(t *testing.T) {
	svc := NewDeckService(nil)

	// Build a 60-card deck but with 5 copies of one non-energy card
	cards := []model.Card{
		{ID: 1, Name: "Charizard"},
		{ID: 2, Name: "Pikachu"},
	}
	// 5 + 55 = 60 cards total but Charizard exceeds 4-copy limit
	quantities := map[int64]int{
		1: 5,
		2: 55,
	}

	errs := svc.ValidateDeck(cards, quantities)
	hasDuplicateErr := false
	for _, e := range errs {
		if e.Type == "duplicate_limit" && e.Message != "" {
			hasDuplicateErr = true
		}
	}
	assert.True(t, hasDuplicateErr)
}

func TestValidateDeck_BasicEnergyExempt(t *testing.T) {
	svc := NewDeckService(nil)

	// 4 non-energy + 56 basic energy cards = 60 (energy > 4 but exempt)
	cards := []model.Card{
		{ID: 1, Name: "Charizard"},
		{ID: 2, Name: "Fire Energy"},
	}
	quantities := map[int64]int{
		1: 4,
		2: 56,
	}

	errs := svc.ValidateDeck(cards, quantities)
	// Should have no errors at all: count is correct, Charizard ≤ 4, energy is exempt
	assert.Empty(t, errs)
}

func TestValidateDeck_BasicEnergyVariants(t *testing.T) {
	svc := NewDeckService(nil)

	energyNames := []string{
		"Grass Energy", "Fire Energy", "Water Energy", "Lightning Energy",
		"Psychic Energy", "Fighting Energy", "Darkness Energy", "Metal Energy",
		"Fairy Energy", "Basic Grass Energy", "Basic Fire Energy",
	}

	for _, name := range energyNames {
		t.Run(name, func(t *testing.T) {
			cards := []model.Card{{ID: 1, Name: name}}
			// Fill the rest of the 60-card deck
			otherCards := make([]model.Card, 14)
			quantities := map[int64]int{1: 4}
			for i := range otherCards {
				id := int64(i + 2)
				otherCards[i] = model.Card{ID: id, Name: "Filler " + string(rune('A'+i))}
				quantities[id] = 4
			}

			allCards := append(cards, otherCards...)
			errs := svc.ValidateDeck(allCards, quantities)
			for _, e := range errs {
				assert.NotEqual(t, "duplicate_limit", e.Type, "basic energy should be exempt: %s", name)
			}
		})
	}
}

func TestValidateDeck_ZeroQuantityIgnored(t *testing.T) {
	svc := NewDeckService(nil)

	cards := []model.Card{
		{ID: 1, Name: "Charizard"},
		{ID: 2, Name: "Pikachu"},
	}
	// Only pikachu counts; charizard has qty=0
	quantities := map[int64]int{1: 0, 2: 4}

	errs := svc.ValidateDeck(cards, quantities)
	// 4 cards total → count error, but no duplicate error
	hasDuplicateErr := false
	for _, e := range errs {
		if e.Type == "duplicate_limit" {
			hasDuplicateErr = true
		}
	}
	assert.False(t, hasDuplicateErr)
}

func TestValidateDeck_MultipleErrors(t *testing.T) {
	svc := NewDeckService(nil)

	cards := []model.Card{
		{ID: 1, Name: "Charizard"},
		{ID: 2, Name: "Venusaur"},
	}
	// Wrong count AND duplicate violation
	quantities := map[int64]int{1: 5, 2: 5} // total = 10, both > 4

	errs := svc.ValidateDeck(cards, quantities)
	assert.GreaterOrEqual(t, len(errs), 3) // count error + 2 duplicate errors
}
