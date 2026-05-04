package tui

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/runtimecfg"
	"github.com/laiambryant/tui-cardman/internal/testutil"
)

func TestInitSSHConfigManagerReplacesExistingManager(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	testutil.ApplyTestMigrations(t, db, "../db/migrations")

	userID := seedTUIUser(t, db)
	configPath := t.TempDir() + "/cardman.json"
	t.Setenv("CARDMAN_CONFIG", configPath)

	localManager, err := newTestLocalConfigManager(configPath)
	require.NoError(t, err)

	m := &Model{
		db:            db,
		isSSHMode:     true,
		configManager: localManager,
	}

	m.initSSHConfigManager(userID)

	require.NotNil(t, m.configManager)
	assert.True(t, m.configManager.IsUserMode())
}

func TestNewModelSSHStartsWithoutConfigManager(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	testutil.ApplyTestMigrations(t, db, "../db/migrations")

	model, err := NewModel(db, true)
	require.NoError(t, err)
	assert.Nil(t, model.configManager)
	assert.Equal(t, ScreenLogin, model.postSplashScreen)
}

func seedTUIUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	res, err := db.Exec(`
		INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active)
		VALUES ('SSH', 'User', 'ssh-user@example.com', 'hash', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
	`)
	require.NoError(t, err)
	userID, err := res.LastInsertId()
	require.NoError(t, err)
	return userID
}

func newTestLocalConfigManager(configPath string) (*runtimecfg.Manager, error) {
	return runtimecfg.NewManager(true, configPath, nil, 0)
}
