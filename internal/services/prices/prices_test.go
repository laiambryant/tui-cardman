package prices

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/testutil"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db := testutil.SetupTestDB(t)
	testutil.ApplyTestMigrations(t, db, "../../db/migrations")
	return db
}

// seedTestCard creates a test card and returns its ID
func seedTestCard(t *testing.T, db *sql.DB) int64 {
	// Create a card game
	result, err := db.Exec(`INSERT INTO card_games (name) VALUES (?)`, "Pokemon TCG")
	require.NoError(t, err)
	gameID, err := result.LastInsertId()
	require.NoError(t, err)

	// Create a set
	result, err = db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`,
		"base1", "Base Set")
	require.NoError(t, err)
	setID, err := result.LastInsertId()
	require.NoError(t, err)

	// Create a card
	result, err = db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, is_placeholder)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		gameID, setID, "base1-1", "Pikachu", "Common", "1", 0)
	require.NoError(t, err)
	cardID, err := result.LastInsertId()
	require.NoError(t, err)

	return cardID
}

// TestTCGPlayerPriceService tests
func TestNewTCGPlayerPriceService(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewTCGPlayerPriceService(db)
	assert.NotNil(t, service)
	assert.IsType(t, &TCGPlayerPriceServiceImpl{}, service)
}

func TestTCGPlayerInsertPrice(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	cardID := seedTestCard(t, db)
	service := NewTCGPlayerPriceService(db)

	// Begin transaction
	tx, err := db.Begin()
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	// Insert price
	err = service.InsertPrice(context.Background(), tx, cardID, "normal",
		10.50, 12.00, 15.00, 11.50, 9.00,
		"https://tcgplayer.com/product/123", "2024-01-15T10:00:00Z")
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Verify price was inserted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM prices_tcgplayer WHERE card_id = ?`, cardID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify price values
	var low, mid, high, market, directLow sql.NullFloat64
	err = db.QueryRow(`SELECT low, mid, high, market, direct_low FROM prices_tcgplayer WHERE card_id = ?`,
		cardID).Scan(&low, &mid, &high, &market, &directLow)
	require.NoError(t, err)
	assert.True(t, low.Valid)
	assert.Equal(t, 10.50, low.Float64)
	assert.True(t, mid.Valid)
	assert.Equal(t, 12.00, mid.Float64)
}

func TestTCGPlayerInsertPrice_NullHandling(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	cardID := seedTestCard(t, db)
	service := NewTCGPlayerPriceService(db)

	tx, err := db.Begin()
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	// Insert price with zero values (should be NULL)
	err = service.InsertPrice(context.Background(), tx, cardID, "foil",
		0, 0, 25.00, 20.00, 0,
		"https://tcgplayer.com/product/124", "2024-01-16T10:00:00Z")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify NULL handling
	var low, mid, directLow sql.NullFloat64
	err = db.QueryRow(`SELECT low, mid, direct_low FROM prices_tcgplayer WHERE card_id = ? AND price_type = 'foil'`,
		cardID).Scan(&low, &mid, &directLow)
	require.NoError(t, err)
	assert.False(t, low.Valid, "Zero value should be stored as NULL")
	assert.False(t, mid.Valid, "Zero value should be stored as NULL")
	assert.False(t, directLow.Valid, "Zero value should be stored as NULL")
}

func TestTCGPlayerDeletePrices(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	cardID := seedTestCard(t, db)
	service := NewTCGPlayerPriceService(db)

	// Insert some prices first
	tx, err := db.Begin()
	require.NoError(t, err)

	err = service.InsertPrice(context.Background(), tx, cardID, "normal",
		10.00, 12.00, 15.00, 11.00, 9.00,
		"https://tcgplayer.com/product/123", "2024-01-15T10:00:00Z")
	require.NoError(t, err)

	err = service.InsertPrice(context.Background(), tx, cardID, "foil",
		20.00, 25.00, 30.00, 22.00, 18.00,
		"https://tcgplayer.com/product/124", "2024-01-15T10:00:00Z")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify prices exist
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM prices_tcgplayer WHERE card_id = ?`, cardID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Delete prices
	tx, err = db.Begin()
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	err = service.DeletePrices(context.Background(), tx, cardID)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify prices were deleted
	err = db.QueryRow(`SELECT COUNT(*) FROM prices_tcgplayer WHERE card_id = ?`, cardID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// TestCardMarketPriceService tests
func TestNewCardMarketPriceService(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewCardMarketPriceService(db)
	assert.NotNil(t, service)
	assert.IsType(t, &CardMarketPriceServiceImpl{}, service)
}

func TestCardMarketInsertPrice(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	cardID := seedTestCard(t, db)
	service := NewCardMarketPriceService(db)

	tx, err := db.Begin()
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	// Insert price
	err = service.InsertPrice(context.Background(), tx, cardID,
		8.50, 9.25, "https://cardmarket.com/product/123")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify price was inserted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM prices_cardmarket WHERE card_id = ?`, cardID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify price values
	var avgPrice, trendPrice sql.NullFloat64
	var url string
	err = db.QueryRow(`SELECT avg_price, trend_price, url FROM prices_cardmarket WHERE card_id = ?`,
		cardID).Scan(&avgPrice, &trendPrice, &url)
	require.NoError(t, err)
	assert.True(t, avgPrice.Valid)
	assert.Equal(t, 8.50, avgPrice.Float64)
	assert.True(t, trendPrice.Valid)
	assert.Equal(t, 9.25, trendPrice.Float64)
	assert.Equal(t, "https://cardmarket.com/product/123", url)
}

func TestCardMarketInsertPrice_NullHandling(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	cardID := seedTestCard(t, db)
	service := NewCardMarketPriceService(db)

	tx, err := db.Begin()
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	// Insert price with zero values (should be NULL)
	err = service.InsertPrice(context.Background(), tx, cardID,
		0, 15.00, "https://cardmarket.com/product/124")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify NULL handling
	var avgPrice sql.NullFloat64
	err = db.QueryRow(`SELECT avg_price FROM prices_cardmarket WHERE card_id = ?`, cardID).Scan(&avgPrice)
	require.NoError(t, err)
	assert.False(t, avgPrice.Valid, "Zero value should be stored as NULL")
}

func TestCardMarketDeletePrices(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	cardID := seedTestCard(t, db)
	service := NewCardMarketPriceService(db)

	// Insert a price first
	tx, err := db.Begin()
	require.NoError(t, err)

	err = service.InsertPrice(context.Background(), tx, cardID,
		10.00, 11.00, "https://cardmarket.com/product/123")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify price exists
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM prices_cardmarket WHERE card_id = ?`, cardID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Delete price
	tx, err = db.Begin()
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	err = service.DeletePrices(context.Background(), tx, cardID)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify price was deleted
	err = db.QueryRow(`SELECT COUNT(*) FROM prices_cardmarket WHERE card_id = ?`, cardID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestTCGPlayerInsertPrice_ContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	cardID := seedTestCard(t, db)
	service := NewTCGPlayerPriceService(db)

	tx, err := db.Begin()
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Try to insert price with canceled context
	err = service.InsertPrice(ctx, tx, cardID, "normal",
		10.50, 12.00, 15.00, 11.50, 9.00,
		"https://tcgplayer.com/product/123", "2024-01-15T10:00:00Z")

	// Should fail due to context cancellation
	assert.Error(t, err)
}

func TestTCGPlayerDeletePrices_NonExistentCard(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewTCGPlayerPriceService(db)

	tx, err := db.Begin()
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	// Delete prices for non-existent card (should succeed - deletes 0 rows)
	err = service.DeletePrices(context.Background(), tx, 99999)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)
}

func TestTCGPlayerInsertPrice_MultipleTypes(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	cardID := seedTestCard(t, db)
	service := NewTCGPlayerPriceService(db)

	tx, err := db.Begin()
	require.NoError(t, err)

	// Insert normal price
	err = service.InsertPrice(context.Background(), tx, cardID, "normal",
		10.00, 12.00, 15.00, 11.00, 9.00,
		"https://tcgplayer.com/product/123", "2024-01-15T10:00:00Z")
	require.NoError(t, err)

	// Insert holofoil price
	err = service.InsertPrice(context.Background(), tx, cardID, "holofoil",
		20.00, 25.00, 30.00, 22.00, 18.00,
		"https://tcgplayer.com/product/124", "2024-01-15T10:00:00Z")
	require.NoError(t, err)

	// Insert reverse holofoil price
	err = service.InsertPrice(context.Background(), tx, cardID, "reverseHolofoil",
		15.00, 18.00, 22.00, 16.50, 14.00,
		"https://tcgplayer.com/product/125", "2024-01-15T10:00:00Z")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify all three prices exist
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM prices_tcgplayer WHERE card_id = ?`, cardID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Verify each price type
	var priceTypes []string
	rows, err := db.Query(`SELECT price_type FROM prices_tcgplayer WHERE card_id = ? ORDER BY price_type`, cardID)
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var priceType string
		err = rows.Scan(&priceType)
		require.NoError(t, err)
		priceTypes = append(priceTypes, priceType)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"holofoil", "normal", "reverseHolofoil"}, priceTypes)
}

func TestCardMarketInsertPrice_ContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	cardID := seedTestCard(t, db)
	service := NewCardMarketPriceService(db)

	tx, err := db.Begin()
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Try to insert price with canceled context
	err = service.InsertPrice(ctx, tx, cardID,
		8.50, 9.25, "https://cardmarket.com/product/123")

	// Should fail due to context cancellation
	assert.Error(t, err)
}

func TestCardMarketDeletePrices_NonExistentCard(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewCardMarketPriceService(db)

	tx, err := db.Begin()
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	// Delete prices for non-existent card (should succeed - deletes 0 rows)
	err = service.DeletePrices(context.Background(), tx, 99999)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)
}

func TestCardMarketInsertPrice_MultipleRecords(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	cardID := seedTestCard(t, db)
	service := NewCardMarketPriceService(db)

	tx, err := db.Begin()
	require.NoError(t, err)

	// Insert multiple price snapshots
	err = service.InsertPrice(context.Background(), tx, cardID,
		8.50, 9.25, "https://cardmarket.com/product/123")
	require.NoError(t, err)

	err = service.InsertPrice(context.Background(), tx, cardID,
		8.75, 9.50, "https://cardmarket.com/product/123")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify both snapshots exist
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM prices_cardmarket WHERE card_id = ?`, cardID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestCardMarketInsertPrice_BothNullPrices(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	cardID := seedTestCard(t, db)
	service := NewCardMarketPriceService(db)

	tx, err := db.Begin()
	require.NoError(t, err)
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	// Insert price with both values as zero (both NULL)
	err = service.InsertPrice(context.Background(), tx, cardID,
		0, 0, "https://cardmarket.com/product/126")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify both are NULL
	var avgPrice, trendPrice sql.NullFloat64
	err = db.QueryRow(`SELECT avg_price, trend_price FROM prices_cardmarket WHERE card_id = ?`,
		cardID).Scan(&avgPrice, &trendPrice)
	require.NoError(t, err)
	assert.False(t, avgPrice.Valid, "Zero avg_price should be NULL")
	assert.False(t, trendPrice.Valid, "Zero trend_price should be NULL")
}

func TestTCGPlayerInsertPrice_LargeValues(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	cardID := seedTestCard(t, db)
	service := NewTCGPlayerPriceService(db)

	tx, err := db.Begin()
	require.NoError(t, err)

	// Insert prices with large values
	err = service.InsertPrice(context.Background(), tx, cardID, "normal",
		9999.99, 10500.50, 12000.00, 10250.25, 9500.00,
		"https://tcgplayer.com/product/999", "2024-01-15T10:00:00Z")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify large values are stored correctly
	var market sql.NullFloat64
	err = db.QueryRow(`SELECT market FROM prices_tcgplayer WHERE card_id = ?`, cardID).Scan(&market)
	require.NoError(t, err)
	assert.True(t, market.Valid)
	assert.Equal(t, 10250.25, market.Float64)
}

func TestCardMarketInsertPrice_LargeValues(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	cardID := seedTestCard(t, db)
	service := NewCardMarketPriceService(db)

	tx, err := db.Begin()
	require.NoError(t, err)

	// Insert prices with large values
	err = service.InsertPrice(context.Background(), tx, cardID,
		5555.55, 6000.99, "https://cardmarket.com/product/999")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify large values are stored correctly
	var avgPrice sql.NullFloat64
	err = db.QueryRow(`SELECT avg_price FROM prices_cardmarket WHERE card_id = ?`, cardID).Scan(&avgPrice)
	require.NoError(t, err)
	assert.True(t, avgPrice.Valid)
	assert.Equal(t, 5555.55, avgPrice.Float64)
}

func TestTCGPlayerDeletePrices_PartialDelete(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	// Create two cards within the same game and set
	gameID, err := db.Exec(`INSERT INTO card_games (name) VALUES (?)`, "Test Game")
	require.NoError(t, err)
	gID, err := gameID.LastInsertId()
	require.NoError(t, err)

	setID, err := db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`, "test-set", "Test Set")
	require.NoError(t, err)
	sID, err := setID.LastInsertId()
	require.NoError(t, err)

	result1, err := db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, is_placeholder)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, gID, sID, "card-1", "Card 1", "Common", "1", 0)
	require.NoError(t, err)
	cardID1, err := result1.LastInsertId()
	require.NoError(t, err)

	result2, err := db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, is_placeholder)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, gID, sID, "card-2", "Card 2", "Common", "2", 0)
	require.NoError(t, err)
	cardID2, err := result2.LastInsertId()
	require.NoError(t, err)

	service := NewTCGPlayerPriceService(db)

	// Insert prices for both cards
	tx, err := db.Begin()
	require.NoError(t, err)

	err = service.InsertPrice(context.Background(), tx, cardID1, "normal",
		10.00, 12.00, 15.00, 11.00, 9.00,
		"https://tcgplayer.com/product/123", "2024-01-15T10:00:00Z")
	require.NoError(t, err)

	err = service.InsertPrice(context.Background(), tx, cardID2, "normal",
		20.00, 22.00, 25.00, 21.00, 19.00,
		"https://tcgplayer.com/product/124", "2024-01-15T10:00:00Z")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Delete prices for only first card
	tx, err = db.Begin()
	require.NoError(t, err)

	err = service.DeletePrices(context.Background(), tx, cardID1)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify only card1 prices were deleted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM prices_tcgplayer WHERE card_id = ?`, cardID1).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	err = db.QueryRow(`SELECT COUNT(*) FROM prices_tcgplayer WHERE card_id = ?`, cardID2).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCardMarketDeletePrices_PartialDelete(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	// Create two cards within the same game and set
	gameID, err := db.Exec(`INSERT INTO card_games (name) VALUES (?)`, "Test Game 2")
	require.NoError(t, err)
	gID, err := gameID.LastInsertId()
	require.NoError(t, err)

	setID, err := db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`, "test-set-2", "Test Set 2")
	require.NoError(t, err)
	sID, err := setID.LastInsertId()
	require.NoError(t, err)

	result1, err := db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, is_placeholder)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, gID, sID, "card-3", "Card 3", "Common", "3", 0)
	require.NoError(t, err)
	cardID1, err := result1.LastInsertId()
	require.NoError(t, err)

	result2, err := db.Exec(`INSERT INTO cards (card_game_id, set_id, api_id, name, rarity, number, is_placeholder)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, gID, sID, "card-4", "Card 4", "Common", "4", 0)
	require.NoError(t, err)
	cardID2, err := result2.LastInsertId()
	require.NoError(t, err)

	service := NewCardMarketPriceService(db)

	// Insert prices for both cards
	tx, err := db.Begin()
	require.NoError(t, err)

	err = service.InsertPrice(context.Background(), tx, cardID1,
		8.50, 9.25, "https://cardmarket.com/product/123")
	require.NoError(t, err)

	err = service.InsertPrice(context.Background(), tx, cardID2,
		15.00, 16.50, "https://cardmarket.com/product/124")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Delete prices for only first card
	tx, err = db.Begin()
	require.NoError(t, err)

	err = service.DeletePrices(context.Background(), tx, cardID1)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify only card1 prices were deleted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM prices_cardmarket WHERE card_id = ?`, cardID1).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	err = db.QueryRow(`SELECT COUNT(*) FROM prices_cardmarket WHERE card_id = ?`, cardID2).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// TestNullFloat64Helper tests the helper function
func TestNullFloat64(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected sql.NullFloat64
	}{
		{
			name:     "Zero value should be NULL",
			value:    0,
			expected: sql.NullFloat64{Valid: false},
		},
		{
			name:     "Positive value should be valid",
			value:    10.50,
			expected: sql.NullFloat64{Float64: 10.50, Valid: true},
		},
		{
			name:     "Negative value should be valid",
			value:    -5.25,
			expected: sql.NullFloat64{Float64: -5.25, Valid: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullFloat64(tt.value)
			assert.Equal(t, tt.expected.Valid, result.Valid)
			if tt.expected.Valid {
				assert.Equal(t, tt.expected.Float64, result.Float64)
			}
		})
	}
}
