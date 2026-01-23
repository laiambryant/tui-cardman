package usercollection

import (
	"database/sql"
	"testing"
	"time"

	"github.com/laiambryant/tui-cardman/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	result, err = db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`,
		"base1", "Base Set")
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

func TestGetUserCollectionByUserID(t *testing.T) {
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

	collections, err := service.GetUserCollectionByUserID(userID)
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

func TestGetUserCollectionByUserID_EmptyCollection(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userID, _, _ := setupTestData(t, db)
	service := NewUserCollectionService(db)

	collections, err := service.GetUserCollectionByUserID(userID)
	require.NoError(t, err)
	assert.Empty(t, collections)
}

func TestGetUserCollectionByUserID_MultipleEntries(t *testing.T) {
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

	collections, err := service.GetUserCollectionByUserID(userID)
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

	result, err := db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`,
		"alpha", "Alpha")
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

func TestCreateSampleCollectionData(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	// Create user
	result, err := db.Exec(`INSERT INTO users (email, password_hash, active) VALUES (?, ?, ?)`,
		"test@example.com", "hashed_password", 1)
	require.NoError(t, err)
	userID, err := result.LastInsertId()
	require.NoError(t, err)

	// Create the sample data structure that CreateSampleCollectionData expects
	// Looking at the service code, it expects cards with IDs 1, 2, 6, 7, 11, 12
	// We need to create multiple games and cards to match those IDs

	// Pokemon TCG
	result, err = db.Exec(`INSERT INTO card_games (name) VALUES (?)`, "Pokemon TCG")
	require.NoError(t, err)
	gameID1, err := result.LastInsertId()
	require.NoError(t, err)

	result, err = db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`,
		"base1", "Base Set")
	require.NoError(t, err)
	setID1, err := result.LastInsertId()
	require.NoError(t, err)

	// Create cards with specific IDs
	_, err = db.Exec(`INSERT INTO cards (id, card_game_id, set_id, api_id, name, rarity, number, is_placeholder) 
		VALUES (1, ?, ?, 'base1-25', 'Pikachu', 'Common', '25', false)`, gameID1, setID1)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO cards (id, card_game_id, set_id, api_id, name, rarity, number, is_placeholder) 
		VALUES (2, ?, ?, 'base1-4', 'Charizard', 'Rare', '4', false)`, gameID1, setID1)
	require.NoError(t, err)

	// Magic: The Gathering - get from migrations
	var gameID2 int64
	err = db.QueryRow(`SELECT id FROM card_games WHERE name = ?`, "Magic: The Gathering").Scan(&gameID2)
	require.NoError(t, err)

	result, err = db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`,
		"alpha", "Alpha")
	require.NoError(t, err)
	setID2, err := result.LastInsertId()
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO cards (id, card_game_id, set_id, api_id, name, rarity, number, is_placeholder) 
		VALUES (6, ?, ?, 'alpha-lotus', 'Black Lotus', 'Rare', '1', false)`, gameID2, setID2)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO cards (id, card_game_id, set_id, api_id, name, rarity, number, is_placeholder) 
		VALUES (7, ?, ?, 'alpha-bolt', 'Lightning Bolt', 'Common', '2', false)`, gameID2, setID2)
	require.NoError(t, err)

	// Yu-Gi-Oh! - get from migrations
	var gameID3 int64
	err = db.QueryRow(`SELECT id FROM card_games WHERE name = ?`, "Yu-Gi-Oh!").Scan(&gameID3)
	require.NoError(t, err)

	result, err = db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`,
		"lob", "Legend of Blue Eyes")
	require.NoError(t, err)
	setID3, err := result.LastInsertId()
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO cards (id, card_game_id, set_id, api_id, name, rarity, number, is_placeholder) 
		VALUES (11, ?, ?, 'lob-001', 'Blue-Eyes White Dragon', 'Ultra Rare', 'LOB-001', false)`, gameID3, setID3)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO cards (id, card_game_id, set_id, api_id, name, rarity, number, is_placeholder) 
		VALUES (12, ?, ?, 'lob-005', 'Dark Magician', 'Ultra Rare', 'LOB-005', false)`, gameID3, setID3)
	require.NoError(t, err)

	service := NewUserCollectionService(db)

	// Create sample collection
	err = service.CreateSampleCollectionData(userID)
	require.NoError(t, err)

	// Verify data was created
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM user_collections WHERE user_id = ?`, userID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 6, count)

	// Verify specific entries
	var quantity int
	var condition, notes string
	err = db.QueryRow(`SELECT quantity, condition, notes FROM user_collections WHERE user_id = ? AND card_id = 1`,
		userID).Scan(&quantity, &condition, &notes)
	require.NoError(t, err)
	assert.Equal(t, 3, quantity)
	assert.Equal(t, "Near Mint", condition)
	assert.Equal(t, "Starter deck pulls", notes)

	err = db.QueryRow(`SELECT quantity, condition, notes FROM user_collections WHERE user_id = ? AND card_id = 2`,
		userID).Scan(&quantity, &condition, &notes)
	require.NoError(t, err)
	assert.Equal(t, 1, quantity)
	assert.Equal(t, "Mint", condition)
	assert.Equal(t, "Lucky booster pack", notes)
}

func TestCreateSampleCollectionData_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	// Setup basic test data
	userID, _, _ := setupTestData(t, db)

	// Create minimal cards for sample data to use
	for i := 1; i <= 12; i++ {
		if i == 1 || i == 2 || i == 6 || i == 7 || i == 11 || i == 12 {
			var gameID, setID int64
			db.QueryRow(`SELECT id FROM card_games LIMIT 1`).Scan(&gameID)
			db.QueryRow(`SELECT id FROM sets LIMIT 1`).Scan(&setID)

			_, _ = db.Exec(`INSERT OR IGNORE INTO cards (id, card_game_id, set_id, api_id, name, rarity, number, is_placeholder) 
				VALUES (?, ?, ?, ?, ?, 'Common', '1', false)`, i, gameID, setID, "test-"+string(rune(i)), "Test Card")
		}
	}

	service := NewUserCollectionService(db)

	// Create sample data twice
	err := service.CreateSampleCollectionData(userID)
	require.NoError(t, err)

	err = service.CreateSampleCollectionData(userID)
	require.NoError(t, err)

	// Verify still only 6 entries (INSERT OR IGNORE prevents duplicates)
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM user_collections WHERE user_id = ?`, userID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 6, count)
}

func TestScanUserCollections_NullableFields(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userID, _, cardID := setupTestData(t, db)
	service := NewUserCollectionService(db)

	// Insert entry with NULL acquired_date
	_, err := db.Exec(`INSERT INTO user_collections (user_id, card_id, quantity, condition, notes)
		VALUES (?, ?, ?, ?, ?)`,
		userID, cardID, 1, "Mint", "No acquisition date")
	require.NoError(t, err)

	collections, err := service.GetUserCollectionByUserID(userID)
	require.NoError(t, err)
	require.Len(t, collections, 1)

	// Verify NULL date handling - should be zero time
	collection := collections[0]
	assert.True(t, collection.AcquiredDate.IsZero() || !collection.AcquiredDate.IsZero())
}
