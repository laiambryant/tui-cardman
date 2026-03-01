package buttonconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
	"github.com/laiambryant/tui-cardman/internal/testutil"
)

func setupButtonConfigTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := testutil.SetupTestDB(t)
	testutil.ApplyTestMigrations(t, db, "../../db/migrations")
	return db
}

func seedButtonConfigUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	res, err := db.Exec(`
		INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active)
		VALUES ('Config', 'User', 'config@example.com', 'hash', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
	`)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

func TestNewButtonConfigService(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewButtonConfigService(db)
	assert.NotNil(t, svc)
	assert.IsType(t, &ButtonConfigServiceImpl{}, svc)
}

func TestGetByUserID_NoRows(t *testing.T) {
	db := setupButtonConfigTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewButtonConfigService(db)
	ctx := context.Background()
	userID := seedButtonConfigUser(t, db)

	result, err := svc.GetByUserID(ctx, userID)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestSave_CreatesNewConfig(t *testing.T) {
	db := setupButtonConfigTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewButtonConfigService(db)
	ctx := context.Background()
	userID := seedButtonConfigUser(t, db)

	cfg := runtimecfg.Default()
	err := svc.Save(ctx, userID, cfg)
	require.NoError(t, err)

	// Should now be retrievable
	saved, err := svc.GetByUserID(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, userID, saved.UserID)
	assert.NotEmpty(t, saved.Configuration)
}

func TestSave_UpdatesExistingConfig(t *testing.T) {
	db := setupButtonConfigTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewButtonConfigService(db)
	ctx := context.Background()
	userID := seedButtonConfigUser(t, db)

	// Initial save with default config
	cfg := runtimecfg.Default()
	err := svc.Save(ctx, userID, cfg)
	require.NoError(t, err)

	// Modify a keybinding and save again
	cfg.Keybindings["quit"] = "ctrl+q"
	err = svc.Save(ctx, userID, cfg)
	require.NoError(t, err)

	saved, err := svc.GetByUserID(ctx, userID)
	require.NoError(t, err)

	var loadedCfg runtimecfg.RuntimeConfig
	err = json.Unmarshal([]byte(saved.Configuration), &loadedCfg)
	require.NoError(t, err)
	assert.Equal(t, "ctrl+q", loadedCfg.Keybindings["quit"])
}

func TestSave_ConfigIsValidJSON(t *testing.T) {
	db := setupButtonConfigTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewButtonConfigService(db)
	ctx := context.Background()
	userID := seedButtonConfigUser(t, db)

	cfg := runtimecfg.Default()
	err := svc.Save(ctx, userID, cfg)
	require.NoError(t, err)

	saved, err := svc.GetByUserID(ctx, userID)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(saved.Configuration), &parsed)
	require.NoError(t, err, "saved configuration must be valid JSON")
}

func TestInitializeDefault(t *testing.T) {
	db := setupButtonConfigTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewButtonConfigService(db)
	ctx := context.Background()
	userID := seedButtonConfigUser(t, db)

	err := svc.InitializeDefault(ctx, userID)
	require.NoError(t, err)

	saved, err := svc.GetByUserID(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, saved)

	// Verify the saved config matches the default
	var savedCfg runtimecfg.RuntimeConfig
	err = json.Unmarshal([]byte(saved.Configuration), &savedCfg)
	require.NoError(t, err)

	defaults := runtimecfg.Default()
	for action, key := range defaults.Keybindings {
		assert.Equal(t, key, savedCfg.Keybindings[action], "keybinding mismatch for action: %s", action)
	}
}

func TestInitializeDefault_IdempotentForSameUser(t *testing.T) {
	db := setupButtonConfigTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewButtonConfigService(db)
	ctx := context.Background()
	userID := seedButtonConfigUser(t, db)

	// Calling twice should not error (ON CONFLICT DO UPDATE)
	err := svc.InitializeDefault(ctx, userID)
	require.NoError(t, err)

	err = svc.InitializeDefault(ctx, userID)
	require.NoError(t, err)
}

func TestSave_MultipleUsers(t *testing.T) {
	db := setupButtonConfigTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewButtonConfigService(db)
	ctx := context.Background()

	res1, err := db.Exec(`
		INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active)
		VALUES ('User', 'One', 'user1@example.com', 'hash', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
	`)
	require.NoError(t, err)
	userID1, _ := res1.LastInsertId()

	res2, err := db.Exec(`
		INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active)
		VALUES ('User', 'Two', 'user2@example.com', 'hash', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
	`)
	require.NoError(t, err)
	userID2, _ := res2.LastInsertId()

	cfg1 := runtimecfg.Default()
	cfg1.Keybindings["quit"] = "ctrl+1"
	cfg2 := runtimecfg.Default()
	cfg2.Keybindings["quit"] = "ctrl+2"

	err = svc.Save(ctx, userID1, cfg1)
	require.NoError(t, err)
	err = svc.Save(ctx, userID2, cfg2)
	require.NoError(t, err)

	saved1, err := svc.GetByUserID(ctx, userID1)
	require.NoError(t, err)
	saved2, err := svc.GetByUserID(ctx, userID2)
	require.NoError(t, err)

	var loadedCfg1, loadedCfg2 runtimecfg.RuntimeConfig
	require.NoError(t, json.Unmarshal([]byte(saved1.Configuration), &loadedCfg1))
	require.NoError(t, json.Unmarshal([]byte(saved2.Configuration), &loadedCfg2))

	assert.Equal(t, "ctrl+1", loadedCfg1.Keybindings["quit"])
	assert.Equal(t, "ctrl+2", loadedCfg2.Keybindings["quit"])
}
