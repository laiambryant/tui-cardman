package list

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/testutil"
)

func setupListTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := testutil.SetupTestDB(t)
	testutil.ApplyTestMigrations(t, db, "../../db/migrations")
	return db
}

// seedListTestData inserts baseline rows and returns (userID, cardGameID, cardID1, cardID2).
func seedListTestData(t *testing.T, db *sql.DB) (userID, cardGameID, cardID1, cardID2 int64) {
	t.Helper()

	err := db.QueryRow(`SELECT id FROM card_games WHERE name = 'Pokemon'`).Scan(&cardGameID)
	require.NoError(t, err)

	res, err := db.Exec(`
		INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active)
		VALUES ('Test', 'User', 'list@example.com', 'hash', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
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

func TestNewListService(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	assert.NotNil(t, svc)
	assert.IsType(t, &ListServiceImpl{}, svc)
}

func TestCreateList(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedListTestData(t, db)

	list, err := svc.CreateList(ctx, userID, cardGameID, "Want List", "Cards I want", "#ff0000")
	require.NoError(t, err)
	require.NotNil(t, list)

	assert.Positive(t, list.ID)
	assert.Equal(t, userID, list.UserID)
	assert.Equal(t, cardGameID, list.CardGameID)
	assert.Equal(t, "Want List", list.Name)
	assert.Equal(t, "Cards I want", list.Description)
	assert.Equal(t, "#ff0000", list.Color)
}

func TestCreateList_EmptyDescriptionAndColor(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedListTestData(t, db)

	list, err := svc.CreateList(ctx, userID, cardGameID, "Minimal List", "", "")
	require.NoError(t, err)
	assert.Equal(t, "", list.Description)
	assert.Equal(t, "", list.Color)
}

func TestGetListsByUserAndGame(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedListTestData(t, db)

	_, err := svc.CreateList(ctx, userID, cardGameID, "Alpha List", "", "")
	require.NoError(t, err)
	_, err = svc.CreateList(ctx, userID, cardGameID, "Beta List", "", "")
	require.NoError(t, err)

	lists, err := svc.GetListsByUserAndGame(userID, cardGameID)
	require.NoError(t, err)
	assert.Len(t, lists, 2)

	// Ordered by name ASC
	assert.Equal(t, "Alpha List", lists[0].Name)
	assert.Equal(t, "Beta List", lists[1].Name)
}

func TestGetListsByUserAndGame_Empty(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	userID, cardGameID, _, _ := seedListTestData(t, db)

	lists, err := svc.GetListsByUserAndGame(userID, cardGameID)
	require.NoError(t, err)
	assert.Empty(t, lists)
}

func TestGetListsByUserAndGame_IsolatedByUser(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedListTestData(t, db)

	res, err := db.Exec(`
		INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active)
		VALUES ('Other', 'User', 'other-list@example.com', 'hash', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
	`)
	require.NoError(t, err)
	otherUserID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = svc.CreateList(ctx, userID, cardGameID, "My List", "", "")
	require.NoError(t, err)
	_, err = svc.CreateList(ctx, otherUserID, cardGameID, "Their List", "", "")
	require.NoError(t, err)

	lists, err := svc.GetListsByUserAndGame(userID, cardGameID)
	require.NoError(t, err)
	assert.Len(t, lists, 1)
	assert.Equal(t, "My List", lists[0].Name)
}

func TestGetListByID(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedListTestData(t, db)

	created, err := svc.CreateList(ctx, userID, cardGameID, "My List", "desc", "#00ff00")
	require.NoError(t, err)

	fetched, err := svc.GetListByID(created.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched)

	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, "My List", fetched.Name)
	assert.Equal(t, "desc", fetched.Description)
	assert.Equal(t, "#00ff00", fetched.Color)
}

func TestGetListByID_NotFound(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)

	_, err := svc.GetListByID(999999)
	assert.Error(t, err)
}

func TestUpdateList(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedListTestData(t, db)

	list, err := svc.CreateList(ctx, userID, cardGameID, "Old Name", "old desc", "#aaaaaa")
	require.NoError(t, err)

	err = svc.UpdateList(ctx, list.ID, "New Name", "new desc", "#bbbbbb")
	require.NoError(t, err)

	updated, err := svc.GetListByID(list.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "new desc", updated.Description)
	assert.Equal(t, "#bbbbbb", updated.Color)
}

func TestDeleteList(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedListTestData(t, db)

	list, err := svc.CreateList(ctx, userID, cardGameID, "To Delete", "", "")
	require.NoError(t, err)

	err = svc.DeleteList(ctx, list.ID)
	require.NoError(t, err)

	_, err = svc.GetListByID(list.ID)
	assert.Error(t, err)
}

func TestDeleteList_NonExistent(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()

	err := svc.DeleteList(ctx, 999999)
	assert.NoError(t, err)
}

func TestUpsertListCardBatch_Add(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()
	userID, cardGameID, cardID1, cardID2 := seedListTestData(t, db)

	list, err := svc.CreateList(ctx, userID, cardGameID, "Test List", "", "")
	require.NoError(t, err)

	updates := map[int64]int{cardID1: 2, cardID2: 5}
	err = svc.UpsertListCardBatch(ctx, list.ID, updates)
	require.NoError(t, err)

	quantities, err := svc.GetAllQuantitiesForList(list.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, quantities[cardID1])
	assert.Equal(t, 5, quantities[cardID2])
}

func TestUpsertListCardBatch_Update(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()
	userID, cardGameID, cardID1, _ := seedListTestData(t, db)

	list, err := svc.CreateList(ctx, userID, cardGameID, "Test List", "", "")
	require.NoError(t, err)

	err = svc.UpsertListCardBatch(ctx, list.ID, map[int64]int{cardID1: 1})
	require.NoError(t, err)

	err = svc.UpsertListCardBatch(ctx, list.ID, map[int64]int{cardID1: 7})
	require.NoError(t, err)

	quantities, err := svc.GetAllQuantitiesForList(list.ID)
	require.NoError(t, err)
	assert.Equal(t, 7, quantities[cardID1])
}

func TestUpsertListCardBatch_ZeroRemovesCard(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()
	userID, cardGameID, cardID1, cardID2 := seedListTestData(t, db)

	list, err := svc.CreateList(ctx, userID, cardGameID, "Test List", "", "")
	require.NoError(t, err)

	err = svc.UpsertListCardBatch(ctx, list.ID, map[int64]int{cardID1: 3, cardID2: 1})
	require.NoError(t, err)

	err = svc.UpsertListCardBatch(ctx, list.ID, map[int64]int{cardID1: 0})
	require.NoError(t, err)

	quantities, err := svc.GetAllQuantitiesForList(list.ID)
	require.NoError(t, err)
	_, exists := quantities[cardID1]
	assert.False(t, exists, "card with quantity 0 should be removed")
	assert.Equal(t, 1, quantities[cardID2])
}

func TestGetAllQuantitiesForList_Empty(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()
	userID, cardGameID, _, _ := seedListTestData(t, db)

	list, err := svc.CreateList(ctx, userID, cardGameID, "Empty List", "", "")
	require.NoError(t, err)

	quantities, err := svc.GetAllQuantitiesForList(list.ID)
	require.NoError(t, err)
	assert.Empty(t, quantities)
}

func TestGetListsContainingCard(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()
	userID, cardGameID, cardID1, cardID2 := seedListTestData(t, db)

	list1, err := svc.CreateList(ctx, userID, cardGameID, "List One", "", "")
	require.NoError(t, err)
	list2, err := svc.CreateList(ctx, userID, cardGameID, "List Two", "", "")
	require.NoError(t, err)

	// cardID1 is in list1 and list2, cardID2 only in list1
	err = svc.UpsertListCardBatch(ctx, list1.ID, map[int64]int{cardID1: 1, cardID2: 1})
	require.NoError(t, err)
	err = svc.UpsertListCardBatch(ctx, list2.ID, map[int64]int{cardID1: 2})
	require.NoError(t, err)

	listsWithCard1, err := svc.GetListsContainingCard(userID, cardID1)
	require.NoError(t, err)
	assert.Len(t, listsWithCard1, 2)

	listsWithCard2, err := svc.GetListsContainingCard(userID, cardID2)
	require.NoError(t, err)
	assert.Len(t, listsWithCard2, 1)
	assert.Equal(t, list1.ID, listsWithCard2[0].ID)
}

func TestGetListsContainingCard_CardInNoLists(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	userID, _, cardID1, _ := seedListTestData(t, db)

	lists, err := svc.GetListsContainingCard(userID, cardID1)
	require.NoError(t, err)
	assert.Empty(t, lists)
}

func TestGetListsContainingCard_IsolatedByUser(t *testing.T) {
	db := setupListTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewListService(db)
	ctx := context.Background()
	userID, cardGameID, cardID1, _ := seedListTestData(t, db)

	res, err := db.Exec(`
		INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active)
		VALUES ('Other', 'User', 'other-list2@example.com', 'hash', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
	`)
	require.NoError(t, err)
	otherUserID, err := res.LastInsertId()
	require.NoError(t, err)

	// Both users have a list with cardID1
	myList, err := svc.CreateList(ctx, userID, cardGameID, "My List", "", "")
	require.NoError(t, err)
	theirList, err := svc.CreateList(ctx, otherUserID, cardGameID, "Their List", "", "")
	require.NoError(t, err)

	err = svc.UpsertListCardBatch(ctx, myList.ID, map[int64]int{cardID1: 1})
	require.NoError(t, err)
	err = svc.UpsertListCardBatch(ctx, theirList.ID, map[int64]int{cardID1: 1})
	require.NoError(t, err)

	// Query for userID should only return their own list
	lists, err := svc.GetListsContainingCard(userID, cardID1)
	require.NoError(t, err)
	assert.Len(t, lists, 1)
	assert.Equal(t, myList.ID, lists[0].ID)
}
