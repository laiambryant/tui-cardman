package usercollection

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/testutil"
)

// setupTestDB creates an in-memory database with full schema
func setupTestDB(t *testing.T) *sql.DB {
	db := testutil.SetupTestDB(t)
	testutil.ApplyTestMigrations(t, db, "../../db/migrations")
	return db
}

// setupTestData creates test data for collection tests
func setupTestData(t *testing.T, db *sql.DB) (userID, gameID, cardID int64) {
	// Create a test user
	result, err := db.Exec(`INSERT INTO users (email, password_hash, active) VALUES (?, ?, ?)`,
		"test@example.com", "hashed_password", 1)
	require.NoError(t, err)
	userID, err = result.LastInsertId()
	require.NoError(t, err)

	// Create a test card game
	result, err = db.Exec(`INSERT INTO card_games (name) VALUES (?)`, "Pokemon TCG")
	require.NoError(t, err)
	gameID, err = result.LastInsertId()
	require.NoError(t, err)

	// Create a test set
	var setID int64
	result, err = db.Exec(`INSERT INTO sets (card_game_id, api_id, name) VALUES (?, ?, ?)`,
		gameID, "base1", "Base Set")
	require.NoError(t, err)
	setID, err = result.LastInsertId()
	require.NoError(t, err)

	// Create a test card
	result, err = db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, is_placeholder) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		gameID, setID, "base1-1", "Pikachu", "Common", "1", false)
	require.NoError(t, err)
	cardID, err = result.LastInsertId()
	require.NoError(t, err)

	return userID, gameID, cardID
}

func TestNewUserCollectionService(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewUserCollectionService(db)
	assert.NotNil(t, service)
	assert.IsType(t, &UserCollectionServiceImpl{}, service)
}

func TestGetUserCollectionByGameID_PopulatesRelations(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userID, gameID, cardID := setupTestData(t, db)
	service := NewUserCollectionService(db)

	// Insert a collection entry
	acquiredDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO user_collections (user_id, card_id, quantity, condition, acquired_date, notes)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, cardID, 3, "Near Mint", acquiredDate, "Test collection entry")
	require.NoError(t, err)

	collections, err := service.GetUserCollectionByGameID(userID, gameID)
	require.NoError(t, err)
	require.Len(t, collections, 1)

	collection := collections[0]
	assert.Equal(t, userID, collection.UserID)
	assert.Equal(t, cardID, collection.CardID)
	assert.Equal(t, 3, collection.Quantity)
	assert.Equal(t, "Near Mint", collection.Condition)
	assert.Equal(t, "Test collection entry", collection.Notes)

	// Verify card relationship is populated
	require.NotNil(t, collection.Card)
	assert.Equal(t, "Pikachu", collection.Card.Name)
	assert.Equal(t, "Common", collection.Card.Rarity)
	assert.Equal(t, gameID, collection.Card.CardGameID)

	// Verify card game relationship is populated
	require.NotNil(t, collection.Card.CardGame)
	assert.Equal(t, "Pokemon TCG", collection.Card.CardGame.Name)
}

func TestGetUserCollectionByGameID_MultipleEntries(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userID, gameID, cardID := setupTestData(t, db)
	service := NewUserCollectionService(db)

	// Create another card (reuse the existing set from setupTestData)
	var setID int64
	err := db.QueryRow(`SELECT id FROM sets LIMIT 1`).Scan(&setID)
	require.NoError(t, err)

	result, err := db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, is_placeholder) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		gameID, setID, "base1-2", "Charizard", "Rare", "4", false)
	require.NoError(t, err)
	cardID2, err := result.LastInsertId()
	require.NoError(t, err)

	// Insert multiple collection entries
	acquiredDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err = db.Exec(`INSERT INTO user_collections (user_id, card_id, quantity, condition, acquired_date, notes)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, cardID, 3, "Near Mint", acquiredDate, "Entry 1")
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different created_at times
	_, err = db.Exec(`INSERT INTO user_collections (user_id, card_id, quantity, condition, acquired_date, notes)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, cardID2, 1, "Mint", acquiredDate, "Entry 2")
	require.NoError(t, err)

	collections, err := service.GetUserCollectionByGameID(userID, gameID)
	require.NoError(t, err)
	require.Len(t, collections, 2)

	// Verify ordering (most recent first based on created_at)
	assert.Equal(t, "Pikachu", collections[0].Card.Name)
	assert.Equal(t, "Charizard", collections[1].Card.Name)
}

func TestGetUserCollectionByGameID(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userID, gameID, cardID := setupTestData(t, db)
	service := NewUserCollectionService(db)

	// Insert a collection entry
	acquiredDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO user_collections (user_id, card_id, quantity, condition, acquired_date, notes)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, cardID, 2, "Lightly Played", acquiredDate, "Game-specific entry")
	require.NoError(t, err)

	collections, err := service.GetUserCollectionByGameID(userID, gameID)
	require.NoError(t, err)
	require.Len(t, collections, 1)

	collection := collections[0]
	assert.Equal(t, userID, collection.UserID)
	assert.Equal(t, cardID, collection.CardID)
	assert.Equal(t, 2, collection.Quantity)
	assert.Equal(t, "Lightly Played", collection.Condition)

	// Verify card game matches filter
	require.NotNil(t, collection.Card)
	require.NotNil(t, collection.Card.CardGame)
	assert.Equal(t, gameID, collection.Card.CardGame.ID)
}

func TestGetUserCollectionByGameID_FiltersByGame(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userID, gameID1, cardID1 := setupTestData(t, db)
	service := NewUserCollectionService(db)

	// Get the existing "Magic: The Gathering" game from migrations
	var gameID2 int64
	err := db.QueryRow(`SELECT id FROM card_games WHERE name = ?`, "Magic: The Gathering").Scan(&gameID2)
	require.NoError(t, err)

	result, err := db.Exec(`INSERT INTO sets (card_game_id, api_id, name) VALUES (?, ?, ?)`,
		gameID2, "alpha", "Alpha")
	require.NoError(t, err)
	setID2, err := result.LastInsertId()
	require.NoError(t, err)

	result, err = db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, is_placeholder) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		gameID2, setID2, "alpha-1", "Black Lotus", "Rare", "1", false)
	require.NoError(t, err)
	cardID2, err := result.LastInsertId()
	require.NoError(t, err)

	// Insert collection entries for both games
	acquiredDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err = db.Exec(`INSERT INTO user_collections (user_id, card_id, quantity, condition, acquired_date, notes)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, cardID1, 3, "Near Mint", acquiredDate, "Pokemon card")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO user_collections (user_id, card_id, quantity, condition, acquired_date, notes)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, cardID2, 1, "Mint", acquiredDate, "Magic card")
	require.NoError(t, err)

	// Fetch only Pokemon TCG cards
	collections, err := service.GetUserCollectionByGameID(userID, gameID1)
	require.NoError(t, err)
	require.Len(t, collections, 1)
	assert.Equal(t, "Pikachu", collections[0].Card.Name)

	// Fetch only Magic cards
	collections, err = service.GetUserCollectionByGameID(userID, gameID2)
	require.NoError(t, err)
	require.Len(t, collections, 1)
	assert.Equal(t, "Black Lotus", collections[0].Card.Name)
}

func TestGetUserCollectionByGameID_EmptyCollection(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userID, gameID, _ := setupTestData(t, db)
	service := NewUserCollectionService(db)

	collections, err := service.GetUserCollectionByGameID(userID, gameID)
	require.NoError(t, err)
	assert.Empty(t, collections)
}

func TestScanUserCollections_NullableFields(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userID, gameID, cardID := setupTestData(t, db)
	service := NewUserCollectionService(db)

	// Insert entry with NULL acquired_date
	_, err := db.Exec(`INSERT INTO user_collections (user_id, card_id, quantity, condition, notes)
		VALUES (?, ?, ?, ?, ?)`,
		userID, cardID, 1, "Mint", "No acquisition date")
	require.NoError(t, err)

	collections, err := service.GetUserCollectionByGameID(userID, gameID)
	require.NoError(t, err)
	require.Len(t, collections, 1)

	// Verify NULL date handling - should be zero time
	collection := collections[0]
	assert.True(t, collection.AcquiredDate.IsZero())
}

func TestGetUserCollectionByGameID_LoadsCardNumberAndArtist(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userID, gameID, cardID := setupTestData(t, db)
	_, err := db.Exec(`UPDATE cards SET artist = ? WHERE id = ?`, "Mitsuhiro Arita", cardID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_collections (user_id, card_id, quantity) VALUES (?, ?, ?)`, userID, cardID, 2)
	require.NoError(t, err)

	collections, err := NewUserCollectionService(db).GetUserCollectionByGameID(userID, gameID)
	require.NoError(t, err)
	require.Len(t, collections, 1)
	require.NotNil(t, collections[0].Card)
	assert.Equal(t, "1", collections[0].Card.Number, "number is needed to re-import exported collections")
	assert.Equal(t, "Mitsuhiro Arita", collections[0].Card.Artist)
	assert.Equal(t, "base1-1", collections[0].Card.APIID)
}

func TestGetCollectionValue_UsesOnePricePerCard(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userID, gameID, cardID := setupTestData(t, db)
	_, err := db.Exec(`INSERT INTO user_collections (user_id, card_id, quantity) VALUES (?, ?, ?)`, userID, cardID, 2)
	require.NoError(t, err)
	// A card can have several TCGPlayer variants; owning a copy does not mean owning every variant.
	_, err = db.Exec(`INSERT INTO prices_tcgplayer (card_id, price_type, market) VALUES (?, 'reverse', 3.0), (?, 'normal', 1.5)`, cardID, cardID)
	require.NoError(t, err)

	value, err := NewUserCollectionService(db).GetCollectionValue(userID, gameID)
	require.NoError(t, err)
	assert.InDelta(t, 3.0, value, 0.001, "2 copies x normal market price")
}

func TestGetCollectionValue_FallsBackToOtherVariant(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userID, gameID, cardID := setupTestData(t, db)
	_, err := db.Exec(`INSERT INTO user_collections (user_id, card_id, quantity) VALUES (?, ?, ?)`, userID, cardID, 3)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO prices_tcgplayer (card_id, price_type, market) VALUES (?, 'reverse', 2.0)`, cardID)
	require.NoError(t, err)

	value, err := NewUserCollectionService(db).GetCollectionValue(userID, gameID)
	require.NoError(t, err)
	assert.InDelta(t, 6.0, value, 0.001)
}
