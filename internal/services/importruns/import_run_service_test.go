package importruns

import (
	"context"
	"database/sql"
	"testing"
	"time"

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

func TestNewImportRunService(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)

	assert.NotNil(t, service)
	assert.IsType(t, &ImportRunServiceImpl{}, service)
}

func TestCreateImportRun(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)
	ctx := context.Background()

	// Create import run
	runID, err := service.CreateImportRun(ctx, "import_all")

	require.NoError(t, err)
	assert.Greater(t, runID, int64(0))

	// Verify import run was created
	var importType, status string
	var setsProcessed, cardsImported, errorsCount int
	var startedAt time.Time
	var completedAt sql.NullTime
	var notes sql.NullString

	err = db.QueryRow(`SELECT import_type, status, sets_processed, cards_imported, errors_count, started_at, completed_at, notes 
		FROM import_runs WHERE id = ?`, runID).
		Scan(&importType, &status, &setsProcessed, &cardsImported, &errorsCount, &startedAt, &completedAt, &notes)
	require.NoError(t, err)

	assert.Equal(t, "import_all", importType)
	assert.Equal(t, "running", status)
	assert.Equal(t, 0, setsProcessed)
	assert.Equal(t, 0, cardsImported)
	assert.Equal(t, 0, errorsCount)
	assert.False(t, startedAt.IsZero())
	assert.False(t, completedAt.Valid)
	assert.False(t, notes.Valid)
}

func TestCreateImportRun_DifferentTypes(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)
	ctx := context.Background()

	importTypes := []string{
		"import_all",
		"import_new",
		"import_specific",
		"manual_import",
	}

	var runIDs []int64

	for _, importType := range importTypes {
		runID, err := service.CreateImportRun(ctx, importType)
		require.NoError(t, err)
		assert.Greater(t, runID, int64(0))
		runIDs = append(runIDs, runID)
	}

	// Verify all runs are different
	assert.Len(t, runIDs, 4)
	for i := 0; i < len(runIDs); i++ {
		for j := i + 1; j < len(runIDs); j++ {
			assert.NotEqual(t, runIDs[i], runIDs[j])
		}
	}

	// Verify all were inserted correctly
	for i, runID := range runIDs {
		var importType string
		err := db.QueryRow(`SELECT import_type FROM import_runs WHERE id = ?`, runID).Scan(&importType)
		require.NoError(t, err)
		assert.Equal(t, importTypes[i], importType)
	}
}

func TestCreateImportRun_ContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail due to canceled context
	_, err := service.CreateImportRun(ctx, "import_all")
	assert.Error(t, err)
}

func TestUpdateImportRun(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)
	ctx := context.Background()

	// Create import run
	runID, err := service.CreateImportRun(ctx, "import_all")
	require.NoError(t, err)

	// Update import run
	err = service.UpdateImportRun(ctx, runID, "completed", 10, 250, 0, "Successfully imported all sets")
	require.NoError(t, err)

	// Verify update
	var status string
	var setsProcessed, cardsImported, errorsCount int
	var completedAt sql.NullTime
	var notes sql.NullString

	err = db.QueryRow(`SELECT status, sets_processed, cards_imported, errors_count, completed_at, notes 
		FROM import_runs WHERE id = ?`, runID).
		Scan(&status, &setsProcessed, &cardsImported, &errorsCount, &completedAt, &notes)
	require.NoError(t, err)

	assert.Equal(t, "completed", status)
	assert.Equal(t, 10, setsProcessed)
	assert.Equal(t, 250, cardsImported)
	assert.Equal(t, 0, errorsCount)
	assert.True(t, completedAt.Valid)
	assert.False(t, completedAt.Time.IsZero())
	assert.True(t, notes.Valid)
	assert.Equal(t, "Successfully imported all sets", notes.String)
}

func TestUpdateImportRun_WithErrors(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)
	ctx := context.Background()

	// Create import run
	runID, err := service.CreateImportRun(ctx, "import_new")
	require.NoError(t, err)

	// Update with errors
	err = service.UpdateImportRun(ctx, runID, "completed_with_errors", 5, 100, 3, "Imported with some errors")
	require.NoError(t, err)

	// Verify update
	var status string
	var errorsCount int
	var notes sql.NullString

	err = db.QueryRow(`SELECT status, errors_count, notes FROM import_runs WHERE id = ?`, runID).
		Scan(&status, &errorsCount, &notes)
	require.NoError(t, err)

	assert.Equal(t, "completed_with_errors", status)
	assert.Equal(t, 3, errorsCount)
	assert.True(t, notes.Valid)
	assert.Equal(t, "Imported with some errors", notes.String)
}

func TestUpdateImportRun_EmptyNotes(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)
	ctx := context.Background()

	// Create import run
	runID, err := service.CreateImportRun(ctx, "import_all")
	require.NoError(t, err)

	// Update with empty notes
	err = service.UpdateImportRun(ctx, runID, "completed", 10, 250, 0, "")
	require.NoError(t, err)

	// Verify empty notes
	var notes sql.NullString
	err = db.QueryRow(`SELECT notes FROM import_runs WHERE id = ?`, runID).Scan(&notes)
	require.NoError(t, err)

	assert.True(t, notes.Valid)
	assert.Equal(t, "", notes.String)
}

func TestUpdateImportRun_NonExistentRun(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)
	ctx := context.Background()

	// Try to update non-existent run
	err := service.UpdateImportRun(ctx, 99999, "completed", 10, 250, 0, "Test")

	// Should succeed but affect 0 rows (SQLite doesn't error)
	require.NoError(t, err)
}

func TestUpdateImportRun_ContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)
	ctx := context.Background()

	// Create import run
	runID, err := service.CreateImportRun(ctx, "import_all")
	require.NoError(t, err)

	// Create canceled context
	ctxCanceled, cancel := context.WithCancel(context.Background())
	cancel()

	// Try to update with canceled context
	err = service.UpdateImportRun(ctxCanceled, runID, "completed", 10, 250, 0, "Test")
	assert.Error(t, err)
}

func TestUpdateImportRun_MultipleUpdates(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)
	ctx := context.Background()

	// Create import run
	runID, err := service.CreateImportRun(ctx, "import_all")
	require.NoError(t, err)

	// First update
	err = service.UpdateImportRun(ctx, runID, "running", 5, 100, 0, "Halfway done")
	require.NoError(t, err)

	var status string
	var setsProcessed, cardsImported int
	err = db.QueryRow(`SELECT status, sets_processed, cards_imported FROM import_runs WHERE id = ?`, runID).
		Scan(&status, &setsProcessed, &cardsImported)
	require.NoError(t, err)
	assert.Equal(t, "running", status)
	assert.Equal(t, 5, setsProcessed)
	assert.Equal(t, 100, cardsImported)

	// Second update
	err = service.UpdateImportRun(ctx, runID, "completed", 10, 250, 0, "Finished")
	require.NoError(t, err)

	err = db.QueryRow(`SELECT status, sets_processed, cards_imported FROM import_runs WHERE id = ?`, runID).
		Scan(&status, &setsProcessed, &cardsImported)
	require.NoError(t, err)
	assert.Equal(t, "completed", status)
	assert.Equal(t, 10, setsProcessed)
	assert.Equal(t, 250, cardsImported)
}

func TestUpdateImportRun_ZeroValues(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)
	ctx := context.Background()

	// Create import run
	runID, err := service.CreateImportRun(ctx, "import_all")
	require.NoError(t, err)

	// Update with all zero values
	err = service.UpdateImportRun(ctx, runID, "failed", 0, 0, 0, "No data processed")
	require.NoError(t, err)

	// Verify zeros were stored
	var status string
	var setsProcessed, cardsImported, errorsCount int
	err = db.QueryRow(`SELECT status, sets_processed, cards_imported, errors_count FROM import_runs WHERE id = ?`, runID).
		Scan(&status, &setsProcessed, &cardsImported, &errorsCount)
	require.NoError(t, err)

	assert.Equal(t, "failed", status)
	assert.Equal(t, 0, setsProcessed)
	assert.Equal(t, 0, cardsImported)
	assert.Equal(t, 0, errorsCount)
}

func TestUpdateImportRun_LargeValues(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)
	ctx := context.Background()

	// Create import run
	runID, err := service.CreateImportRun(ctx, "import_all")
	require.NoError(t, err)

	// Update with large values
	err = service.UpdateImportRun(ctx, runID, "completed", 1000, 50000, 150, "Large import completed")
	require.NoError(t, err)

	// Verify large values were stored
	var setsProcessed, cardsImported, errorsCount int
	err = db.QueryRow(`SELECT sets_processed, cards_imported, errors_count FROM import_runs WHERE id = ?`, runID).
		Scan(&setsProcessed, &cardsImported, &errorsCount)
	require.NoError(t, err)

	assert.Equal(t, 1000, setsProcessed)
	assert.Equal(t, 50000, cardsImported)
	assert.Equal(t, 150, errorsCount)
}

func TestImportRunService_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)
	ctx := context.Background()

	// Simulate complete import workflow

	// Start import
	runID, err := service.CreateImportRun(ctx, "import_all")
	require.NoError(t, err)

	// Verify initial state
	var status string
	err = db.QueryRow(`SELECT status FROM import_runs WHERE id = ?`, runID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "running", status)

	// Simulate progress update
	time.Sleep(10 * time.Millisecond)
	err = service.UpdateImportRun(ctx, runID, "running", 5, 120, 0, "Processing...")
	require.NoError(t, err)

	// Complete import
	time.Sleep(10 * time.Millisecond)
	err = service.UpdateImportRun(ctx, runID, "completed", 10, 250, 0, "Import completed successfully")
	require.NoError(t, err)

	// Verify final state
	var finalStatus string
	var setsProcessed, cardsImported, errorsCount int
	var startedAt, completedAt time.Time
	var notes sql.NullString

	err = db.QueryRow(`SELECT status, sets_processed, cards_imported, errors_count, started_at, completed_at, notes 
		FROM import_runs WHERE id = ?`, runID).
		Scan(&finalStatus, &setsProcessed, &cardsImported, &errorsCount, &startedAt, &completedAt, &notes)
	require.NoError(t, err)

	assert.Equal(t, "completed", finalStatus)
	assert.Equal(t, 10, setsProcessed)
	assert.Equal(t, 250, cardsImported)
	assert.Equal(t, 0, errorsCount)
	assert.False(t, startedAt.IsZero())
	assert.False(t, completedAt.IsZero())
	assert.True(t, completedAt.After(startedAt) || completedAt.Equal(startedAt))
	assert.True(t, notes.Valid)
	assert.Equal(t, "Import completed successfully", notes.String)
}

func TestImportRunService_ConcurrentRuns(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)
	ctx := context.Background()

	// Create multiple concurrent import runs
	runID1, err := service.CreateImportRun(ctx, "import_all")
	require.NoError(t, err)

	runID2, err := service.CreateImportRun(ctx, "import_new")
	require.NoError(t, err)

	runID3, err := service.CreateImportRun(ctx, "import_specific")
	require.NoError(t, err)

	// All should have different IDs
	assert.NotEqual(t, runID1, runID2)
	assert.NotEqual(t, runID2, runID3)
	assert.NotEqual(t, runID1, runID3)

	// Update each independently
	err = service.UpdateImportRun(ctx, runID1, "completed", 10, 250, 0, "Run 1 complete")
	require.NoError(t, err)

	err = service.UpdateImportRun(ctx, runID2, "running", 5, 100, 0, "Run 2 in progress")
	require.NoError(t, err)

	err = service.UpdateImportRun(ctx, runID3, "failed", 0, 0, 5, "Run 3 failed")
	require.NoError(t, err)

	// Verify each has correct state
	var status1, status2, status3 string
	err = db.QueryRow(`SELECT status FROM import_runs WHERE id = ?`, runID1).Scan(&status1)
	require.NoError(t, err)
	err = db.QueryRow(`SELECT status FROM import_runs WHERE id = ?`, runID2).Scan(&status2)
	require.NoError(t, err)
	err = db.QueryRow(`SELECT status FROM import_runs WHERE id = ?`, runID3).Scan(&status3)
	require.NoError(t, err)

	assert.Equal(t, "completed", status1)
	assert.Equal(t, "running", status2)
	assert.Equal(t, "failed", status3)
}

func TestUpdateImportRun_LongNotes(t *testing.T) {
	db := setupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewImportRunService(db)
	ctx := context.Background()

	// Create import run
	runID, err := service.CreateImportRun(ctx, "import_all")
	require.NoError(t, err)

	// Update with very long notes
	longNotes := "This is a very long note that contains detailed information about the import process. " +
		"It includes timestamps, error messages, warnings, and other diagnostic information. " +
		"The note can be quite lengthy and should be stored correctly in the database. " +
		"This tests that TEXT fields can handle large amounts of data without truncation."

	err = service.UpdateImportRun(ctx, runID, "completed", 10, 250, 0, longNotes)
	require.NoError(t, err)

	// Verify long notes were stored correctly
	var notes sql.NullString
	err = db.QueryRow(`SELECT notes FROM import_runs WHERE id = ?`, runID).Scan(&notes)
	require.NoError(t, err)

	assert.True(t, notes.Valid)
	assert.Equal(t, longNotes, notes.String)
	assert.Greater(t, len(notes.String), 200)
}
