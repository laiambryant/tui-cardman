package pokemontcg

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/model"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/importruns"
	"github.com/laiambryant/tui-cardman/internal/services/prices"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
	"github.com/laiambryant/tui-cardman/internal/testutil"
)

// Mock implementations for testing

type mockImportRunService struct {
	createRunFunc func(ctx context.Context, importType string) (int64, error)
	updateRunFunc func(ctx context.Context, runID int64, status string, setsProcessed, cardsImported, errorsCount int, notes string) error
}

func (m *mockImportRunService) CreateImportRun(ctx context.Context, importType string) (int64, error) {
	if m.createRunFunc != nil {
		return m.createRunFunc(ctx, importType)
	}
	return 1, nil
}

func (m *mockImportRunService) UpdateImportRun(ctx context.Context, runID int64, status string, setsProcessed, cardsImported, errorsCount int, notes string) error {
	if m.updateRunFunc != nil {
		return m.updateRunFunc(ctx, runID, status, setsProcessed, cardsImported, errorsCount, notes)
	}
	return nil
}

type mockSetService struct {
	getSetIDFunc     func(ctx context.Context, apiID string) (int64, error)
	upsertSetFunc    func(ctx context.Context, apiID, code, name string, printedTotal, total int) (int64, error)
	getAllAPIIDsFunc func(ctx context.Context) ([]string, error)
}

func (m *mockSetService) GetSetIDByAPIID(ctx context.Context, apiID string) (int64, error) {
	if m.getSetIDFunc != nil {
		return m.getSetIDFunc(ctx, apiID)
	}
	return 1, nil
}

func (m *mockSetService) UpsertSet(ctx context.Context, apiID, code, name string, printedTotal, total int) (int64, error) {
	if m.upsertSetFunc != nil {
		return m.upsertSetFunc(ctx, apiID, code, name, printedTotal, total)
	}
	return 1, nil
}

func (m *mockSetService) GetAllSetAPIIDs(ctx context.Context) ([]string, error) {
	if m.getAllAPIIDsFunc != nil {
		return m.getAllAPIIDsFunc(ctx)
	}
	return []string{}, nil
}

type mockCardService struct {
	upsertCardFunc func(ctx context.Context, tx *sql.Tx, apiID string, setID int64, number, name, rarity, artist string, cardGameID int64) (int64, error)
}

func (m *mockCardService) UpsertCard(ctx context.Context, tx *sql.Tx, apiID string, setID int64, number, name, rarity, artist string, cardGameID int64) (int64, error) {
	if m.upsertCardFunc != nil {
		return m.upsertCardFunc(ctx, tx, apiID, setID, number, name, rarity, artist, cardGameID)
	}
	return 1, nil
}

func (m *mockCardService) GetCardsByGameID(gameID int64) ([]model.Card, error) { return nil, nil }
func (m *mockCardService) GetAllCards() ([]model.Card, error)                  { return nil, nil }
func (m *mockCardService) GetCardIDByAPIID(ctx context.Context, apiID string) (int64, error) {
	return 0, nil
}

type mockTCGPlayerPriceService struct {
	insertPriceFunc  func(ctx context.Context, tx *sql.Tx, cardID int64, priceType string, low, mid, high, market, directLow float64, url, updatedAt string) error
	deletePricesFunc func(ctx context.Context, tx *sql.Tx, cardID int64) error
}

func (m *mockTCGPlayerPriceService) InsertPrice(ctx context.Context, tx *sql.Tx, cardID int64, priceType string, low, mid, high, market, directLow float64, url, updatedAt string) error {
	if m.insertPriceFunc != nil {
		return m.insertPriceFunc(ctx, tx, cardID, priceType, low, mid, high, market, directLow, url, updatedAt)
	}
	return nil
}

func (m *mockTCGPlayerPriceService) DeletePrices(ctx context.Context, tx *sql.Tx, cardID int64) error {
	if m.deletePricesFunc != nil {
		return m.deletePricesFunc(ctx, tx, cardID)
	}
	return nil
}

type mockCardMarketPriceService struct {
	insertPriceFunc  func(ctx context.Context, tx *sql.Tx, cardID int64, avgPrice, trendPrice float64, url string) error
	deletePricesFunc func(ctx context.Context, tx *sql.Tx, cardID int64) error
}

func (m *mockCardMarketPriceService) InsertPrice(ctx context.Context, tx *sql.Tx, cardID int64, avgPrice, trendPrice float64, url string) error {
	if m.insertPriceFunc != nil {
		return m.insertPriceFunc(ctx, tx, cardID, avgPrice, trendPrice, url)
	}
	return nil
}

func (m *mockCardMarketPriceService) DeletePrices(ctx context.Context, tx *sql.Tx, cardID int64) error {
	if m.deletePricesFunc != nil {
		return m.deletePricesFunc(ctx, tx, cardID)
	}
	return nil
}

type mockClient struct {
	getSetsFunc        func(ctx context.Context) ([]Set, error)
	getCardsForSetFunc func(ctx context.Context, setID string, page int) (*PaginatedResponse, []Card, error)
	getCardFunc        func(ctx context.Context, cardID string) (*Card, error)
}

func (m *mockClient) GetSets(ctx context.Context) ([]Set, error) {
	if m.getSetsFunc != nil {
		return m.getSetsFunc(ctx)
	}
	return []Set{}, nil
}

func (m *mockClient) GetCardsForSet(ctx context.Context, setID string, page int) (*PaginatedResponse, []Card, error) {
	if m.getCardsForSetFunc != nil {
		return m.getCardsForSetFunc(ctx, setID, page)
	}
	return &PaginatedResponse{Page: page, PageSize: 250, TotalCount: 0}, []Card{}, nil
}

func (m *mockClient) GetCard(ctx context.Context, cardID string) (*Card, error) {
	if m.getCardFunc != nil {
		return m.getCardFunc(ctx, cardID)
	}
	return &Card{}, nil
}

// Test helper to create a test database with Pokemon game
func setupTestDBWithPokemon(t *testing.T) *sql.DB {
	db := testutil.SetupTestDB(t)
	testutil.ApplyTestMigrations(t, db, "../../internal/db/migrations")
	// Pokemon game is already inserted by migration 002_create_card_games.up.sql
	return db
}

// Helper to create import service with real services
func createTestImportService(db *sql.DB) *ImportService {
	client := NewClient("") // Use real client with no API key
	logger := slog.Default()

	// Use real services for database operations
	importRunService := importruns.NewImportRunService(db)
	setService := sets.NewSetService(db)
	cardService := card.NewCardService(db)
	tcgPlayerService := prices.NewTCGPlayerPriceService(db)
	cardMarketService := prices.NewCardMarketPriceService(db)

	return NewImportService(db, client, logger, importRunService, setService, cardService, tcgPlayerService, cardMarketService)
}

func TestNewImportService(t *testing.T) {
	t.Run("Successfully creates import service with Pokemon game", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		require.NotNil(t, service)
		assert.Equal(t, db, service.db)
		assert.Equal(t, int64(1), service.pokemonGameID)
	})

	t.Run("Pokemon game is initialized from database", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		defer testutil.CleanupTestDB(t, db)
		testutil.ApplyTestMigrations(t, db, "../../internal/db/migrations")

		// Pokemon is inserted by migration
		client := NewClient("")
		logger := slog.Default()
		importRunService := importruns.NewImportRunService(db)
		setService := sets.NewSetService(db)
		cardService := card.NewCardService(db)
		tcgPlayerService := prices.NewTCGPlayerPriceService(db)
		cardMarketService := prices.NewCardMarketPriceService(db)

		service := NewImportService(db, client, logger, importRunService, setService, cardService, tcgPlayerService, cardMarketService)

		require.NotNil(t, service)
		// pokemonGameID should be 1 (the ID from the migration)
		assert.Equal(t, int64(1), service.pokemonGameID)
	})
}

func TestImportService_GetExistingSetAPIIDs(t *testing.T) {
	t.Run("Returns existing set API IDs as map", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		// Insert some sets
		_, err := db.Exec("INSERT INTO sets (api_id, name, code, printed_total, total) VALUES ('base1', 'Base Set', 'BS', 102, 102)")
		require.NoError(t, err)
		_, err = db.Exec("INSERT INTO sets (api_id, name, code, printed_total, total) VALUES ('fossil', 'Fossil', 'FO', 62, 62)")
		require.NoError(t, err)

		service := createTestImportService(db)

		existingSets, err := service.GetExistingSetAPIIDs(context.Background())

		require.NoError(t, err)
		assert.Len(t, existingSets, 2)
		assert.True(t, existingSets["base1"])
		assert.True(t, existingSets["fossil"])
		assert.False(t, existingSets["nonexistent"])
	})

	t.Run("Returns empty map when no sets exist", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		existingSets, err := service.GetExistingSetAPIIDs(context.Background())

		require.NoError(t, err)
		assert.Len(t, existingSets, 0)
	})
}

func TestImportService_FilterNewSets(t *testing.T) {
	t.Run("Filters out existing sets", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		// Insert some existing sets
		_, err := db.Exec("INSERT INTO sets (api_id, name, code, printed_total, total) VALUES ('base1', 'Base Set', 'BS', 102, 102)")
		require.NoError(t, err)
		_, err = db.Exec("INSERT INTO sets (api_id, name, code, printed_total, total) VALUES ('fossil', 'Fossil', 'FO', 62, 62)")
		require.NoError(t, err)

		service := createTestImportService(db)

		allSets := []Set{
			{ID: "base1", Name: "Base Set"},
			{ID: "base2", Name: "Base Set 2"},
			{ID: "fossil", Name: "Fossil"},
			{ID: "jungle", Name: "Jungle"},
		}

		newSets, err := service.filterNewSets(context.Background(), allSets)

		require.NoError(t, err)
		assert.Len(t, newSets, 2)
		assert.Equal(t, "base2", newSets[0].ID)
		assert.Equal(t, "jungle", newSets[1].ID)
	})

	t.Run("Returns empty when all sets exist", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		_, err := db.Exec("INSERT INTO sets (api_id, name, code, printed_total, total) VALUES ('base1', 'Base Set', 'BS', 102, 102)")
		require.NoError(t, err)
		_, err = db.Exec("INSERT INTO sets (api_id, name, code, printed_total, total) VALUES ('base2', 'Base Set 2', 'B2', 130, 130)")
		require.NoError(t, err)

		service := createTestImportService(db)

		allSets := []Set{
			{ID: "base1", Name: "Base Set"},
			{ID: "base2", Name: "Base Set 2"},
		}

		newSets, err := service.filterNewSets(context.Background(), allSets)

		require.NoError(t, err)
		assert.Len(t, newSets, 0)
	})

	t.Run("Returns all sets when none exist", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		allSets := []Set{
			{ID: "base1", Name: "Base Set"},
			{ID: "base2", Name: "Base Set 2"},
		}

		newSets, err := service.filterNewSets(context.Background(), allSets)

		require.NoError(t, err)
		assert.Len(t, newSets, 2)
	})
}

func TestImportService_FindRequestedSets(t *testing.T) {
	t.Run("Finds requested sets and tracks not found", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		allSets := []Set{
			{ID: "base1", Name: "Base Set"},
			{ID: "base2", Name: "Base Set 2"},
			{ID: "fossil", Name: "Fossil"},
		}

		requestedIDs := []string{"base1", "jungle", "fossil"}

		found, notFound := service.findRequestedSets(allSets, requestedIDs)

		assert.Len(t, found, 2)
		assert.Equal(t, "base1", found[0].ID)
		assert.Equal(t, "fossil", found[1].ID)

		assert.Len(t, notFound, 1)
		assert.Equal(t, "jungle", notFound[0])
	})

	t.Run("Returns all requested when all found", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		allSets := []Set{
			{ID: "base1", Name: "Base Set"},
			{ID: "base2", Name: "Base Set 2"},
		}

		requestedIDs := []string{"base1", "base2"}

		found, notFound := service.findRequestedSets(allSets, requestedIDs)

		assert.Len(t, found, 2)
		assert.Len(t, notFound, 0)
	})

	t.Run("Returns all not found when none exist", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		allSets := []Set{
			{ID: "base1", Name: "Base Set"},
		}

		requestedIDs := []string{"fossil", "jungle"}

		found, notFound := service.findRequestedSets(allSets, requestedIDs)

		assert.Len(t, found, 0)
		assert.Len(t, notFound, 2)
	})
}

func TestImportService_BuildImportNotes(t *testing.T) {
	t.Run("Builds completed status with no errors", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		status, notes := service.buildImportNotes(5, 250, 0)

		assert.Equal(t, "completed", status)
		assert.Contains(t, notes, "Imported 5 sets")
		assert.Contains(t, notes, "250 total cards")
	})

	t.Run("Builds completed_with_errors status when errors exist", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		status, notes := service.buildImportNotes(5, 250, 3)

		assert.Equal(t, "completed_with_errors", status)
		assert.Contains(t, notes, "Imported 5 sets")
		assert.Contains(t, notes, "250 total cards")
	})

	t.Run("Adds extra notes when provided", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		status, notes := service.buildImportNotes(5, 250, 0, "Extra information")

		assert.Equal(t, "completed", status)
		assert.Contains(t, notes, "Imported 5 sets")
		assert.Contains(t, notes, "Extra information")
	})

	t.Run("Skips empty extra notes", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		status, notes := service.buildImportNotes(2, 100, 0, "")

		assert.Equal(t, "completed", status)
		assert.Contains(t, notes, "Imported 2 sets")
		assert.Contains(t, notes, "100 total cards")
		assert.NotContains(t, notes, ". .")
	})
}

func TestImportService_UpsertSet(t *testing.T) {
	t.Run("Inserts new set and returns ID", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		set := Set{
			ID:           "base1",
			Name:         "Base Set",
			PrintedTotal: 102,
			Total:        102,
			PtcgoCode:    "BS",
		}

		setID, err := service.UpsertSet(context.Background(), set)

		require.NoError(t, err)
		assert.Greater(t, setID, int64(0))

		// Verify set was inserted
		var name string
		err = db.QueryRow("SELECT name FROM sets WHERE id = ?", setID).Scan(&name)
		require.NoError(t, err)
		assert.Equal(t, "Base Set", name)
	})

	t.Run("Updates existing set", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		// Insert initial set
		set1 := Set{
			ID:           "base1",
			Name:         "Base Set",
			PrintedTotal: 102,
			Total:        102,
			PtcgoCode:    "BS",
		}
		setID1, err := service.UpsertSet(context.Background(), set1)
		require.NoError(t, err)

		// Update with new data
		set2 := Set{
			ID:           "base1",
			Name:         "Base Set Updated",
			PrintedTotal: 103,
			Total:        103,
			PtcgoCode:    "BS1",
		}
		setID2, err := service.UpsertSet(context.Background(), set2)
		require.NoError(t, err)

		// Should return same ID
		assert.Equal(t, setID1, setID2)

		// Verify set was updated
		var name string
		var total int
		err = db.QueryRow("SELECT name, total FROM sets WHERE id = ?", setID2).Scan(&name, &total)
		require.NoError(t, err)
		assert.Equal(t, "Base Set Updated", name)
		assert.Equal(t, 103, total)
	})
}

func TestImportService_CreateAndUpdateImportRun(t *testing.T) {
	t.Run("Creates import run successfully", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		runID, err := service.CreateImportRun(context.Background(), "import-full")

		require.NoError(t, err)
		assert.Greater(t, runID, int64(0))

		// Verify import run was created
		var importType, status string
		err = db.QueryRow("SELECT import_type, status FROM import_runs WHERE id = ?", runID).Scan(&importType, &status)
		require.NoError(t, err)
		assert.Equal(t, "import-full", importType)
		assert.Equal(t, "running", status)
	})

	t.Run("Updates import run with completion data", func(t *testing.T) {
		db := setupTestDBWithPokemon(t)
		defer testutil.CleanupTestDB(t, db)

		service := createTestImportService(db)

		// Create run
		runID, err := service.CreateImportRun(context.Background(), "import-updates")
		require.NoError(t, err)

		// Update run
		err = service.UpdateImportRun(context.Background(), runID, "completed", 5, 250, 0, "Successfully imported")
		require.NoError(t, err)

		// Verify update
		var status string
		var setsProcessed, cardsImported int
		err = db.QueryRow("SELECT status, sets_processed, cards_imported FROM import_runs WHERE id = ?", runID).
			Scan(&status, &setsProcessed, &cardsImported)
		require.NoError(t, err)
		assert.Equal(t, "completed", status)
		assert.Equal(t, 5, setsProcessed)
		assert.Equal(t, 250, cardsImported)
	})
}
