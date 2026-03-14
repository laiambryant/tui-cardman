package sets

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/testutil"
)

func TestNewSetService(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewSetService(db)

	assert.NotNil(t, service)
	assert.IsType(t, &SetServiceImpl{}, service)
}

func TestGetSetIDByAPIID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	// Create a set first
	result, err := db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`, "base1", "Base Set")
	require.NoError(t, err)
	expectedID, err := result.LastInsertId()
	require.NoError(t, err)

	// Get set ID by API ID
	setID, err := service.GetSetIDByAPIID(ctx, "base1")

	require.NoError(t, err)
	assert.Equal(t, expectedID, setID)
}

func TestGetSetIDByAPIID_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	setID, err := service.GetSetIDByAPIID(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Equal(t, int64(0), setID)
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestUpsertSet_Insert(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	// Insert a new set
	setID, err := service.UpsertSet(ctx, "base1", "BS", "Base Set", 102, 102)

	require.NoError(t, err)
	assert.Greater(t, setID, int64(0))

	// Verify the set was inserted
	var apiID, code, name string
	var printedTotal, total int
	err = db.QueryRow(`SELECT api_id, code, name, printed_total, total FROM sets WHERE id = ?`, setID).
		Scan(&apiID, &code, &name, &printedTotal, &total)
	require.NoError(t, err)

	assert.Equal(t, "base1", apiID)
	assert.Equal(t, "BS", code)
	assert.Equal(t, "Base Set", name)
	assert.Equal(t, 102, printedTotal)
	assert.Equal(t, 102, total)
}

func TestUpsertSet_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	// Insert initial set
	setID1, err := service.UpsertSet(ctx, "base1", "BS", "Base Set", 102, 102)
	require.NoError(t, err)

	// Update the same set with different values
	setID2, err := service.UpsertSet(ctx, "base1", "BS1", "Base Set Unlimited", 103, 103)
	require.NoError(t, err)

	// Should return the same ID
	assert.Equal(t, setID1, setID2)

	// Verify the set was updated
	var apiID, code, name string
	var printedTotal, total int
	err = db.QueryRow(`SELECT api_id, code, name, printed_total, total FROM sets WHERE id = ?`, setID2).
		Scan(&apiID, &code, &name, &printedTotal, &total)
	require.NoError(t, err)

	assert.Equal(t, "base1", apiID) // api_id should not change
	assert.Equal(t, "BS1", code)
	assert.Equal(t, "Base Set Unlimited", name)
	assert.Equal(t, 103, printedTotal)
	assert.Equal(t, 103, total)
}

func TestUpsertSet_MultipleSets(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	sets := []struct {
		apiID        string
		code         string
		name         string
		printedTotal int
		total        int
	}{
		{"base1", "BS", "Base Set", 102, 102},
		{"jungle", "JU", "Jungle", 64, 64},
		{"fossil", "FO", "Fossil", 62, 62},
	}

	var setIDs []int64

	// Insert multiple sets
	for _, s := range sets {
		setID, err := service.UpsertSet(ctx, s.apiID, s.code, s.name, s.printedTotal, s.total)
		require.NoError(t, err)
		setIDs = append(setIDs, setID)
	}

	// Verify all sets are different
	assert.Len(t, setIDs, 3)
	assert.NotEqual(t, setIDs[0], setIDs[1])
	assert.NotEqual(t, setIDs[1], setIDs[2])
	assert.NotEqual(t, setIDs[0], setIDs[2])

	// Verify count
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM sets`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestUpsertSet_EmptyStrings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	// Insert set with empty code
	setID, err := service.UpsertSet(ctx, "test-set", "", "Test Set", 0, 0)

	require.NoError(t, err)
	assert.Greater(t, setID, int64(0))

	// Verify the set was inserted with empty code
	var code string
	err = db.QueryRow(`SELECT code FROM sets WHERE id = ?`, setID).Scan(&code)
	require.NoError(t, err)
	assert.Equal(t, "", code)
}

func TestUpsertSet_ZeroValues(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	// Insert set with zero totals
	setID, err := service.UpsertSet(ctx, "zero-set", "ZS", "Zero Set", 0, 0)

	require.NoError(t, err)
	assert.Greater(t, setID, int64(0))

	// Verify the set was inserted with zero values
	var printedTotal, total int
	err = db.QueryRow(`SELECT printed_total, total FROM sets WHERE id = ?`, setID).
		Scan(&printedTotal, &total)
	require.NoError(t, err)
	assert.Equal(t, 0, printedTotal)
	assert.Equal(t, 0, total)
}

func TestUpsertSet_TransactionRollback(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)

	// Use a canceled context to force transaction failure
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := service.UpsertSet(ctx, "canceled-set", "CS", "Canceled Set", 10, 10)

	assert.Error(t, err)

	// Verify no set was inserted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM sets WHERE api_id = ?`, "canceled-set").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestGetAllSetAPIIDs(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	// Insert multiple sets
	sets := []struct {
		apiID string
		name  string
	}{
		{"base1", "Base Set"},
		{"jungle", "Jungle"},
		{"fossil", "Fossil"},
		{"base2", "Base Set 2"},
	}

	for _, s := range sets {
		_, err := db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`, s.apiID, s.name)
		require.NoError(t, err)
	}

	// Get all set API IDs
	apiIDs, err := service.GetAllSetAPIIDs(ctx)

	require.NoError(t, err)
	assert.Len(t, apiIDs, 4)
	assert.Contains(t, apiIDs, "base1")
	assert.Contains(t, apiIDs, "jungle")
	assert.Contains(t, apiIDs, "fossil")
	assert.Contains(t, apiIDs, "base2")
}

func TestGetAllSetAPIIDs_Empty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	apiIDs, err := service.GetAllSetAPIIDs(ctx)

	require.NoError(t, err)
	assert.Empty(t, apiIDs) // Will be nil slice when no rows (Go SQL pattern)
}

func TestGetAllSetAPIIDs_SingleSet(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	// Insert single set
	_, err := db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`, "single", "Single Set")
	require.NoError(t, err)

	apiIDs, err := service.GetAllSetAPIIDs(ctx)

	require.NoError(t, err)
	assert.Len(t, apiIDs, 1)
	assert.Equal(t, "single", apiIDs[0])
}

func TestSetService_Integration(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	// Test complete flow: upsert (insert) -> get by api_id -> upsert (update) -> get all

	// Insert new set
	setID1, err := service.UpsertSet(ctx, "integration-test", "IT", "Integration Test", 50, 50)
	require.NoError(t, err)
	assert.Greater(t, setID1, int64(0))

	// Get set by API ID
	retrievedID, err := service.GetSetIDByAPIID(ctx, "integration-test")
	require.NoError(t, err)
	assert.Equal(t, setID1, retrievedID)

	// Update the set
	setID2, err := service.UpsertSet(ctx, "integration-test", "IT2", "Integration Test Updated", 75, 75)
	require.NoError(t, err)
	assert.Equal(t, setID1, setID2) // Should be same ID

	// Insert another set
	_, err = service.UpsertSet(ctx, "second-set", "SS", "Second Set", 25, 25)
	require.NoError(t, err)

	// Get all API IDs
	apiIDs, err := service.GetAllSetAPIIDs(ctx)
	require.NoError(t, err)
	assert.Len(t, apiIDs, 2)
	assert.Contains(t, apiIDs, "integration-test")
	assert.Contains(t, apiIDs, "second-set")
}

func TestUpsertSet_ConcurrentTransactions(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	// First transaction: insert
	setID1, err := service.UpsertSet(ctx, "concurrent", "C1", "Concurrent Test 1", 10, 10)
	require.NoError(t, err)

	// Second transaction: update
	setID2, err := service.UpsertSet(ctx, "concurrent", "C2", "Concurrent Test 2", 20, 20)
	require.NoError(t, err)

	// Should be same ID
	assert.Equal(t, setID1, setID2)

	// Verify final state
	var code string
	var printedTotal int
	err = db.QueryRow(`SELECT code, printed_total FROM sets WHERE api_id = ?`, "concurrent").
		Scan(&code, &printedTotal)
	require.NoError(t, err)
	assert.Equal(t, "C2", code)
	assert.Equal(t, 20, printedTotal)
}

func TestUpsertSet_UpdatedAtTimestamp(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	// Insert set
	setID, err := service.UpsertSet(ctx, "timestamp-test", "TT", "Timestamp Test", 10, 10)
	require.NoError(t, err)

	// Get initial updated_at
	var updatedAt1 sql.NullTime
	err = db.QueryRow(`SELECT updated_at FROM sets WHERE id = ?`, setID).Scan(&updatedAt1)
	require.NoError(t, err)
	assert.True(t, updatedAt1.Valid)

	// Update set
	_, err = service.UpsertSet(ctx, "timestamp-test", "TT2", "Timestamp Test Updated", 20, 20)
	require.NoError(t, err)

	// Get new updated_at
	var updatedAt2 sql.NullTime
	err = db.QueryRow(`SELECT updated_at FROM sets WHERE id = ?`, setID).Scan(&updatedAt2)
	require.NoError(t, err)
	assert.True(t, updatedAt2.Valid)

	// Second timestamp should be equal or after first (might be same due to speed)
	assert.True(t, !updatedAt2.Time.Before(updatedAt1.Time))
}

func TestUpsertSet_UniqueAPIID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	// Insert first set
	setID1, err := service.UpsertSet(ctx, "unique-test", "UT1", "Unique Test 1", 10, 10)
	require.NoError(t, err)

	// "Insert" again with same api_id (should update)
	setID2, err := service.UpsertSet(ctx, "unique-test", "UT2", "Unique Test 2", 20, 20)
	require.NoError(t, err)

	// Should be same ID (update, not insert)
	assert.Equal(t, setID1, setID2)

	// Verify only one row exists
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM sets WHERE api_id = ?`, "unique-test").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestGetAllSetAPIIDsWithCounts(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	_, err := service.UpsertSet(ctx, "base1", "BS", "Base Set", 102, 102)
	require.NoError(t, err)
	_, err = service.UpsertSet(ctx, "jungle", "JU", "Jungle", 64, 64)
	require.NoError(t, err)
	_, err = service.UpsertSet(ctx, "fossil", "FO", "Fossil", 62, 62)
	require.NoError(t, err)

	counts, err := service.GetAllSetAPIIDsWithCounts(ctx)

	require.NoError(t, err)
	assert.Len(t, counts, 3)
	assert.Equal(t, 102, counts["base1"])
	assert.Equal(t, 64, counts["jungle"])
	assert.Equal(t, 62, counts["fossil"])
}

func TestGetAllSetAPIIDsWithCounts_Empty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	counts, err := service.GetAllSetAPIIDsWithCounts(ctx)

	require.NoError(t, err)
	assert.Empty(t, counts)
}

func TestGetAllSetAPIIDsWithCounts_ZeroTotal(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	_, err := service.UpsertSet(ctx, "zero-set", "ZS", "Zero Set", 0, 0)
	require.NoError(t, err)

	counts, err := service.GetAllSetAPIIDsWithCounts(ctx)

	require.NoError(t, err)
	assert.Len(t, counts, 1)
	assert.Equal(t, 0, counts["zero-set"])
}

func TestGetAllSetAPIIDsWithCounts_AfterUpdate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	_, err := service.UpsertSet(ctx, "update-test", "UT", "Update Test", 0, 0)
	require.NoError(t, err)

	_, err = service.UpsertSet(ctx, "update-test", "UT", "Update Test", 150, 150)
	require.NoError(t, err)

	counts, err := service.GetAllSetAPIIDsWithCounts(ctx)

	require.NoError(t, err)
	assert.Equal(t, 150, counts["update-test"])
}

func TestGetAllSetAPIIDs_Ordering(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewSetService(db)
	ctx := context.Background()

	// Insert sets in specific order
	apiIDs := []string{"charlie", "alpha", "bravo", "delta"}
	for _, apiID := range apiIDs {
		_, err := db.Exec(`INSERT INTO sets (api_id, name) VALUES (?, ?)`, apiID, apiID)
		require.NoError(t, err)
	}

	// Get all API IDs
	retrievedIDs, err := service.GetAllSetAPIIDs(ctx)
	require.NoError(t, err)

	// Verify all IDs are present (order doesn't matter based on query)
	assert.Len(t, retrievedIDs, 4)
	for _, apiID := range apiIDs {
		assert.Contains(t, retrievedIDs, apiID)
	}
}
