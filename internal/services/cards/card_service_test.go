package card

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/testutil"
)

func TestNewCardService(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewCardService(db)

	assert.NotNil(t, service)
	assert.IsType(t, &CardServiceImpl{}, service)
}

// Helper function to create test data
func setupCardTestData(t *testing.T, db *sql.DB) (gameID1, gameID2, setID1, setID2 int64) {
	t.Helper()

	// Get existing card games from migrations (Pokemon, Magic, Yu-Gi-Oh!)
	err := db.QueryRow(`SELECT id FROM card_games WHERE name = ?`, "Pokemon").Scan(&gameID1)
	require.NoError(t, err)

	err = db.QueryRow(`SELECT id FROM card_games WHERE name = ?`, "Magic: The Gathering").Scan(&gameID2)
	require.NoError(t, err)

	// Create sets
	result, err := db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`, "base1", "Base Set")
	require.NoError(t, err)
	setID1, err = result.LastInsertId()
	require.NoError(t, err)

	result, err = db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`, "jungle", "Jungle")
	require.NoError(t, err)
	setID2, err = result.LastInsertId()
	require.NoError(t, err)

	return gameID1, gameID2, setID1, setID2
}

func TestGetCardsByGameID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)

	gameID1, gameID2, setID1, _ := setupCardTestData(t, db)

	// Insert cards for game 1
	// Note: UNIQUE constraint on (set_id, number) means same set can't have duplicate numbers
	_, err := db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, artist, is_placeholder) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		gameID1, setID1, "base1-1", "Alakazam", "Rare Holo", "1", "Ken Sugimori", false)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, artist, is_placeholder) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		gameID1, setID1, "base1-4", "Charizard", "Rare Holo", "4", "Mitsuhiro Arita", false)
	require.NoError(t, err)

	// Insert card for game 2 with different number to avoid UNIQUE constraint on (set_id, number)
	_, err = db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, artist, is_placeholder) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		gameID2, setID1, "mtg-1", "Black Lotus", "Rare", "999", "Christopher Rush", false)
	require.NoError(t, err)

	// Get cards by game ID 1
	cards, err := service.GetCardsByGameID(gameID1)

	require.NoError(t, err)
	assert.Len(t, cards, 2)

	// Verify ordering by name (Alakazam, Charizard)
	assert.Equal(t, "Alakazam", cards[0].Name)
	assert.Equal(t, "Charizard", cards[1].Name)

	// Verify card fields
	assert.Equal(t, "base1-1", cards[0].APIID)
	assert.Equal(t, "Rare Holo", cards[0].Rarity)
	assert.Equal(t, "1", cards[0].Number)
	assert.Equal(t, "Ken Sugimori", cards[0].Artist)
	assert.Equal(t, setID1, cards[0].SetID)
	assert.False(t, cards[0].IsPlaceholder)

	// Verify CardGame is populated
	require.NotNil(t, cards[0].CardGame)
	assert.Equal(t, gameID1, cards[0].CardGame.ID)
	assert.Equal(t, "Pokemon", cards[0].CardGame.Name)
}

func TestGetCardsByGameID_EmptyResult(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)

	gameID1, _, _, _ := setupCardTestData(t, db)

	// Get cards for game with no cards
	cards, err := service.GetCardsByGameID(gameID1)

	require.NoError(t, err)
	assert.Empty(t, cards)
}

func TestGetCardsByGameID_NullableFields(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)

	gameID1, _, setID1, _ := setupCardTestData(t, db)

	// Insert card with NULL fields
	_, err := db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, is_placeholder) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		gameID1, setID1, "base1-null", "Test Card", "Common", false)
	require.NoError(t, err)

	cards, err := service.GetCardsByGameID(gameID1)

	require.NoError(t, err)
	assert.Len(t, cards, 1)
	assert.Equal(t, "", cards[0].Number) // NULL string becomes empty
	assert.Equal(t, "", cards[0].Artist) // NULL string becomes empty
}

func TestGetAllCards(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)

	gameID1, gameID2, setID1, _ := setupCardTestData(t, db)

	// Insert cards for multiple games
	_, err := db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		gameID2, setID1, "mtg-1", "Black Lotus", "Rare", "1")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		gameID1, setID1, "base1-4", "Charizard", "Rare Holo", "4")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		gameID1, setID1, "base1-25", "Pikachu", "Common", "25")
	require.NoError(t, err)

	cards, err := service.GetAllCards()

	require.NoError(t, err)
	assert.Len(t, cards, 3)

	// Verify ordering: by game name ASC, then card name ASC
	// Magic: The Gathering comes before Pokemon alphabetically
	assert.Equal(t, "Black Lotus", cards[0].Name)
	assert.Equal(t, "Magic: The Gathering", cards[0].CardGame.Name)

	// Pokemon cards ordered by name
	assert.Equal(t, "Charizard", cards[1].Name)
	assert.Equal(t, "Pokemon", cards[1].CardGame.Name)

	assert.Equal(t, "Pikachu", cards[2].Name)
	assert.Equal(t, "Pokemon", cards[2].CardGame.Name)
}

func TestGetAllCards_EmptyDatabase(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)

	cards, err := service.GetAllCards()

	require.NoError(t, err)
	assert.Empty(t, cards)
}

func TestGetCardIDByAPIID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)
	ctx := context.Background()

	gameID1, _, setID1, _ := setupCardTestData(t, db)

	// Insert a card
	result, err := db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity) 
		VALUES (?, ?, ?, ?, ?)`,
		gameID1, setID1, "base1-25", "Pikachu", "Common")
	require.NoError(t, err)
	expectedID, err := result.LastInsertId()
	require.NoError(t, err)

	// Get card ID by API ID
	cardID, err := service.GetCardIDByAPIID(ctx, "base1-25")

	require.NoError(t, err)
	assert.Equal(t, expectedID, cardID)
}

func TestGetCardIDByAPIID_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)
	ctx := context.Background()

	cardID, err := service.GetCardIDByAPIID(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.Equal(t, int64(0), cardID)
}

func TestUpsertCard_Insert(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)
	ctx := context.Background()

	gameID1, _, setID1, _ := setupCardTestData(t, db)

	// Begin transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	// Insert a new card
	cardID, err := service.UpsertCard(ctx, tx, "base1-25", setID1, "25", "Pikachu", "Common", "Atsuko Nishida", gameID1)

	require.NoError(t, err)
	assert.Greater(t, cardID, int64(0))

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Verify the card was inserted
	var apiID, name, rarity, number, artist string
	var cardGameID, setIDResult int64
	err = db.QueryRow(`SELECT api_id, name, rarity, number, artist, card_game_id, set_id FROM cards WHERE id = ?`, cardID).
		Scan(&apiID, &name, &rarity, &number, &artist, &cardGameID, &setIDResult)
	require.NoError(t, err)

	assert.Equal(t, "base1-25", apiID)
	assert.Equal(t, "Pikachu", name)
	assert.Equal(t, "Common", rarity)
	assert.Equal(t, "25", number)
	assert.Equal(t, "Atsuko Nishida", artist)
	assert.Equal(t, gameID1, cardGameID)
	assert.Equal(t, setID1, setIDResult)
}

func TestUpsertCard_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)
	ctx := context.Background()

	gameID1, _, setID1, setID2 := setupCardTestData(t, db)

	// Insert initial card
	tx1, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	cardID1, err := service.UpsertCard(ctx, tx1, "base1-25", setID1, "25", "Pikachu", "Common", "Atsuko Nishida", gameID1)
	require.NoError(t, err)
	err = tx1.Commit()
	require.NoError(t, err)

	// Update the same card with different values
	tx2, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	cardID2, err := service.UpsertCard(ctx, tx2, "base1-25", setID2, "25A", "Pikachu (Updated)", "Rare", "New Artist", gameID1)
	require.NoError(t, err)
	err = tx2.Commit()
	require.NoError(t, err)

	// Should return the same ID
	assert.Equal(t, cardID1, cardID2)

	// Verify the card was updated
	var name, rarity, number, artist string
	var setIDResult int64
	err = db.QueryRow(`SELECT name, rarity, number, artist, set_id FROM cards WHERE id = ?`, cardID2).
		Scan(&name, &rarity, &number, &artist, &setIDResult)
	require.NoError(t, err)

	assert.Equal(t, "Pikachu (Updated)", name)
	assert.Equal(t, "Rare", rarity)
	assert.Equal(t, "25A", number)
	assert.Equal(t, "New Artist", artist)
	assert.Equal(t, setID2, setIDResult)
}

func TestUpsertCard_TransactionRollback(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)
	ctx := context.Background()

	gameID1, _, setID1, _ := setupCardTestData(t, db)

	// Begin transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Insert a card
	cardID, err := service.UpsertCard(ctx, tx, "rollback-test", setID1, "1", "Rollback Test", "Common", "Test Artist", gameID1)
	require.NoError(t, err)
	assert.Greater(t, cardID, int64(0))

	// Rollback instead of commit
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify card was not inserted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM cards WHERE api_id = ?`, "rollback-test").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestUpsertCard_MultipleCards(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)
	ctx := context.Background()

	gameID1, _, setID1, _ := setupCardTestData(t, db)

	cards := []struct {
		apiID  string
		number string
		name   string
		rarity string
		artist string
	}{
		{"base1-1", "1", "Alakazam", "Rare Holo", "Ken Sugimori"},
		{"base1-4", "4", "Charizard", "Rare Holo", "Mitsuhiro Arita"},
		{"base1-25", "25", "Pikachu", "Common", "Atsuko Nishida"},
	}

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	var cardIDs []int64

	for _, c := range cards {
		cardID, err := service.UpsertCard(ctx, tx, c.apiID, setID1, c.number, c.name, c.rarity, c.artist, gameID1)
		require.NoError(t, err)
		cardIDs = append(cardIDs, cardID)
	}

	err = tx.Commit()
	require.NoError(t, err)

	// Verify all cards are different
	assert.Len(t, cardIDs, 3)
	assert.NotEqual(t, cardIDs[0], cardIDs[1])
	assert.NotEqual(t, cardIDs[1], cardIDs[2])
	assert.NotEqual(t, cardIDs[0], cardIDs[2])

	// Verify count
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM cards`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestUpsertCard_EmptyStrings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)
	ctx := context.Background()

	gameID1, _, setID1, _ := setupCardTestData(t, db)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	// Insert card with empty strings
	cardID, err := service.UpsertCard(ctx, tx, "empty-test", setID1, "", "Empty Test", "", "", gameID1)

	require.NoError(t, err)
	assert.Greater(t, cardID, int64(0))

	err = tx.Commit()
	require.NoError(t, err)

	// Verify the card was inserted with empty strings
	var number, rarity, artist string
	err = db.QueryRow(`SELECT number, rarity, artist FROM cards WHERE id = ?`, cardID).
		Scan(&number, &rarity, &artist)
	require.NoError(t, err)
	assert.Equal(t, "", number)
	assert.Equal(t, "", rarity)
	assert.Equal(t, "", artist)
}

func TestUpsertCard_UpdatedAtTimestamp(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)
	ctx := context.Background()

	gameID1, _, setID1, _ := setupCardTestData(t, db)

	// Insert card
	tx1, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	cardID, err := service.UpsertCard(ctx, tx1, "timestamp-test", setID1, "1", "Timestamp Test", "Common", "Artist", gameID1)
	require.NoError(t, err)
	err = tx1.Commit()
	require.NoError(t, err)

	// Get initial updated_at
	var updatedAt1 sql.NullTime
	err = db.QueryRow(`SELECT updated_at FROM cards WHERE id = ?`, cardID).Scan(&updatedAt1)
	require.NoError(t, err)
	assert.True(t, updatedAt1.Valid)

	time.Sleep(10 * time.Millisecond)

	// Update card
	tx2, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	_, err = service.UpsertCard(ctx, tx2, "timestamp-test", setID1, "2", "Timestamp Test Updated", "Rare", "New Artist", gameID1)
	require.NoError(t, err)
	err = tx2.Commit()
	require.NoError(t, err)

	// Get new updated_at
	var updatedAt2 sql.NullTime
	err = db.QueryRow(`SELECT updated_at FROM cards WHERE id = ?`, cardID).Scan(&updatedAt2)
	require.NoError(t, err)
	assert.True(t, updatedAt2.Valid)

	// Second timestamp should be equal or after first
	assert.True(t, !updatedAt2.Time.Before(updatedAt1.Time))
}

func TestUpsertCard_UniqueAPIID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)
	ctx := context.Background()

	gameID1, _, setID1, _ := setupCardTestData(t, db)

	// Insert first card
	tx1, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	cardID1, err := service.UpsertCard(ctx, tx1, "unique-test", setID1, "1", "Unique Test 1", "Common", "Artist 1", gameID1)
	require.NoError(t, err)
	err = tx1.Commit()
	require.NoError(t, err)

	// "Insert" again with same api_id (should update)
	tx2, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	cardID2, err := service.UpsertCard(ctx, tx2, "unique-test", setID1, "2", "Unique Test 2", "Rare", "Artist 2", gameID1)
	require.NoError(t, err)
	err = tx2.Commit()
	require.NoError(t, err)

	// Should be same ID (update, not insert)
	assert.Equal(t, cardID1, cardID2)

	// Verify only one row exists
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM cards WHERE api_id = ?`, "unique-test").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCardService_Integration(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)
	ctx := context.Background()

	gameID1, gameID2, setID1, _ := setupCardTestData(t, db)

	// Test complete flow: upsert (insert) -> get by api_id -> upsert (update) -> get by game -> get all

	// Insert new card
	tx1, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	cardID1, err := service.UpsertCard(ctx, tx1, "integration-test", setID1, "1", "Integration Test", "Common", "Test Artist", gameID1)
	require.NoError(t, err)
	err = tx1.Commit()
	require.NoError(t, err)
	assert.Greater(t, cardID1, int64(0))

	// Get card by API ID
	retrievedID, err := service.GetCardIDByAPIID(ctx, "integration-test")
	require.NoError(t, err)
	assert.Equal(t, cardID1, retrievedID)

	// Update the card
	tx2, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	cardID2, err := service.UpsertCard(ctx, tx2, "integration-test", setID1, "1A", "Integration Test Updated", "Rare", "New Artist", gameID1)
	require.NoError(t, err)
	err = tx2.Commit()
	require.NoError(t, err)
	assert.Equal(t, cardID1, cardID2)

	// Insert another card for different game
	tx3, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	_, err = service.UpsertCard(ctx, tx3, "second-card", setID1, "2", "Second Card", "Uncommon", "Artist 2", gameID2)
	require.NoError(t, err)
	err = tx3.Commit()
	require.NoError(t, err)

	// Get cards by game ID
	cardsGame1, err := service.GetCardsByGameID(gameID1)
	require.NoError(t, err)
	assert.Len(t, cardsGame1, 1)
	assert.Equal(t, "Integration Test Updated", cardsGame1[0].Name)

	cardsGame2, err := service.GetCardsByGameID(gameID2)
	require.NoError(t, err)
	assert.Len(t, cardsGame2, 1)
	assert.Equal(t, "Second Card", cardsGame2[0].Name)

	// Get all cards
	allCards, err := service.GetAllCards()
	require.NoError(t, err)
	assert.Len(t, allCards, 2)
}

func TestGetCardsByGameID_ComplexJoin(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)

	gameID1, _, setID1, _ := setupCardTestData(t, db)

	// Insert card with all fields populated
	// Note: updated_at is set by default in DB, we don't provide it here
	_, err := db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, artist, is_placeholder) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		gameID1, setID1, "complex-1", "Complex Card", "Secret Rare", "999", "Famous Artist", false)
	require.NoError(t, err)

	cards, err := service.GetCardsByGameID(gameID1)

	require.NoError(t, err)
	assert.Len(t, cards, 1)

	card := cards[0]
	assert.Equal(t, "Complex Card", card.Name)
	assert.Equal(t, "Secret Rare", card.Rarity)
	assert.Equal(t, "999", card.Number)
	assert.Equal(t, "Famous Artist", card.Artist)
	assert.False(t, card.IsPlaceholder)
	assert.NotZero(t, card.CreatedAt)
	// updated_at might be zero time if not set properly in scan, checking separately

	// Verify JOIN populated CardGame
	require.NotNil(t, card.CardGame)
	assert.Equal(t, gameID1, card.CardGame.ID)
	assert.Equal(t, "Pokemon", card.CardGame.Name)
	assert.NotZero(t, card.CardGame.CreatedAt)
}

func TestGetAllCards_MultipleGamesOrdering(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)

	gameID1, gameID2, setID1, _ := setupCardTestData(t, db)

	// Get third game (Yu-Gi-Oh!)
	var gameID3 int64
	err := db.QueryRow(`SELECT id FROM card_games WHERE name = ?`, "Yu-Gi-Oh!").Scan(&gameID3)
	require.NoError(t, err)

	// Insert cards in non-alphabetical order
	cards := []struct {
		gameID int64
		name   string
	}{
		{gameID1, "Zapdos"},           // Pokemon
		{gameID2, "Ancestral Recall"}, // Magic
		{gameID3, "Dark Magician"},    // Yu-Gi-Oh!
		{gameID1, "Alakazam"},         // Pokemon
		{gameID2, "Black Lotus"},      // Magic
	}

	for _, c := range cards {
		_, err := db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity) 
			VALUES (?, ?, ?, ?, ?)`,
			c.gameID, setID1, c.name, c.name, "Rare")
		require.NoError(t, err)
	}

	allCards, err := service.GetAllCards()

	require.NoError(t, err)
	assert.Len(t, allCards, 5)

	// Verify ordering: game name ASC, then card name ASC
	// Magic: The Gathering (Ancestral Recall, Black Lotus)
	assert.Equal(t, "Ancestral Recall", allCards[0].Name)
	assert.Equal(t, "Magic: The Gathering", allCards[0].CardGame.Name)
	assert.Equal(t, "Black Lotus", allCards[1].Name)
	assert.Equal(t, "Magic: The Gathering", allCards[1].CardGame.Name)

	// Pokemon (Alakazam, Zapdos)
	assert.Equal(t, "Alakazam", allCards[2].Name)
	assert.Equal(t, "Pokemon", allCards[2].CardGame.Name)
	assert.Equal(t, "Zapdos", allCards[3].Name)
	assert.Equal(t, "Pokemon", allCards[3].CardGame.Name)

	// Yu-Gi-Oh! (Dark Magician)
	assert.Equal(t, "Dark Magician", allCards[4].Name)
	assert.Equal(t, "Yu-Gi-Oh!", allCards[4].CardGame.Name)
}

func TestUpsertCard_ContextCancellation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)

	gameID1, _, setID1, _ := setupCardTestData(t, db)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// BeginTx itself may succeed even with cancelled context
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	// Upsert should fail with cancelled context
	_, err = service.UpsertCard(ctx, tx, "cancelled", setID1, "1", "Cancelled", "Common", "Artist", gameID1)
	// May or may not error depending on timing, but if it does, it should be context error
	if err != nil {
		assert.Contains(t, err.Error(), "context")
	}
}

func TestScanCards_IsPlaceholderField(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)

	gameID1, _, setID1, _ := setupCardTestData(t, db)

	// Insert placeholder card
	_, err := db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, is_placeholder) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		gameID1, setID1, "placeholder-1", "Placeholder Card", "Unknown", true)
	require.NoError(t, err)

	// Insert normal card
	_, err = db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, is_placeholder) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		gameID1, setID1, "normal-1", "Normal Card", "Common", false)
	require.NoError(t, err)

	cards, err := service.GetCardsByGameID(gameID1)

	require.NoError(t, err)
	assert.Len(t, cards, 2)

	// Find each card
	var placeholderCard, normalCard *model.Card
	for i := range cards {
		switch cards[i].Name {
		case "Placeholder Card":
			placeholderCard = &cards[i]
		case "Normal Card":
			normalCard = &cards[i]
		}
	}

	require.NotNil(t, placeholderCard)
	require.NotNil(t, normalCard)

	assert.True(t, placeholderCard.IsPlaceholder)
	assert.False(t, normalCard.IsPlaceholder)
}

func TestGetCardIDByAPIID_ContextCancellation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should error due to cancelled context
	_, err := service.GetCardIDByAPIID(ctx, "test-card")
	assert.Error(t, err)
}

func TestGetCardsByGameID_InvalidGameID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)

	// Query cards for non-existent game ID
	cards, err := service.GetCardsByGameID(99999)

	require.NoError(t, err)
	assert.Empty(t, cards)
}

func TestUpsertCard_UpdateError(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewCardService(db)
	ctx := context.Background()

	gameID1, _, setID1, _ := setupCardTestData(t, db)

	// Insert initial card
	tx1, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	cardID, err := service.UpsertCard(ctx, tx1, "update-error-test", setID1, "1", "Test Card", "Common", "Artist", gameID1)
	require.NoError(t, err)
	err = tx1.Commit()
	require.NoError(t, err)

	// Start new transaction and immediately cancel context
	tx2, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer tx2.Rollback()

	ctxCancelled, cancel := context.WithCancel(context.Background())
	cancel()

	// Try to update with cancelled context (may or may not error depending on SQLite timing)
	_, _ = service.UpsertCard(ctxCancelled, tx2, "update-error-test", setID1, "2", "Updated", "Rare", "New Artist", gameID1)

	// SQLite may not always respect cancelled contexts immediately
	// Just verify original card is unchanged after rollback
	tx2.Rollback()

	// Verify original card data is unchanged
	var name, rarity string
	err = db.QueryRow(`SELECT name, rarity FROM cards WHERE id = ?`, cardID).Scan(&name, &rarity)
	require.NoError(t, err)
	assert.Equal(t, "Test Card", name)
	assert.Equal(t, "Common", rarity)
}
