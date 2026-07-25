package deck

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	return userID, cardGameID, cardID1, cardID2
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

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "My Deck")
	require.NoError(t, err)
	require.NotNil(t, deck)

	assert.Positive(t, deck.ID)
	assert.Equal(t, userID, deck.UserID)
	assert.Equal(t, cardGameID, deck.CardGameID)
	assert.Equal(t, "My Deck", deck.Name)
}

func TestCreateDeck_MultipleDecks(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedDeckTestData(t, db)

	deck1, err := svc.CreateDeck(ctx, userID, cardGameID, "Deck Alpha")
	require.NoError(t, err)

	deck2, err := svc.CreateDeck(ctx, userID, cardGameID, "Deck Beta")
	require.NoError(t, err)

	assert.NotEqual(t, deck1.ID, deck2.ID)
}

func TestGetDecksByUserAndGame(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedDeckTestData(t, db)

	_, err := svc.CreateDeck(ctx, userID, cardGameID, "Deck A")
	require.NoError(t, err)
	_, err = svc.CreateDeck(ctx, userID, cardGameID, "Deck B")
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

	_, err = svc.CreateDeck(ctx, userID, cardGameID, "User1 Deck")
	require.NoError(t, err)
	_, err = svc.CreateDeck(ctx, otherUserID, cardGameID, "User2 Deck")
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

	created, err := svc.CreateDeck(ctx, userID, cardGameID, "My Deck")
	require.NoError(t, err)

	fetched, err := svc.GetDeckByID(created.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched)

	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, created.Name, fetched.Name)
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

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "Old Name")
	require.NoError(t, err)

	err = svc.UpdateDeck(ctx, deck.ID, "New Name")
	require.NoError(t, err)

	updated, err := svc.GetDeckByID(deck.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
}

func TestDeleteDeck(t *testing.T) {
	db := setupDeckTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewDeckService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedDeckTestData(t, db)

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "To Delete")
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

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "Test Deck")
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

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "Test Deck")
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

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "Test Deck")
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

	deck, err := svc.CreateDeck(ctx, userID, cardGameID, "Empty Deck")
	require.NoError(t, err)

	quantities, err := svc.GetAllQuantitiesForDeck(deck.ID)
	require.NoError(t, err)
	assert.Empty(t, quantities)
}
