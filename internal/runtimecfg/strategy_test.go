package runtimecfg

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/model"
)

// mockButtonConfigService implements ButtonConfigService for testing
type mockButtonConfigService struct {
	config      *model.ButtonConfiguration
	getErr      error
	saveErr     error
	migrateErr  error
	saveCalled  bool
	savedConfig *RuntimeConfig
	migratePath string
}

func (m *mockButtonConfigService) GetByUserID(_ context.Context, _ int64) (*model.ButtonConfiguration, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.config, nil
}

func (m *mockButtonConfigService) Save(_ context.Context, _ int64, config *RuntimeConfig) error {
	m.saveCalled = true
	m.savedConfig = config
	return m.saveErr
}

func (m *mockButtonConfigService) MigrateLocalToDB(_ context.Context, _ int64, localPath string) error {
	m.migratePath = localPath
	return m.migrateErr
}

func TestRemoteStrategy_Load_NilService(t *testing.T) {
	defaultCfg := Default()
	strategy := NewRemoteStrategy(nil, 1)
	cfg, err := strategy.Load()
	require.NoError(t, err)
	assert.Equal(t, defaultCfg, cfg)
}

func TestRemoteStrategy_Load_ServiceError(t *testing.T) {
	defaultCfg := Default()
	svc := &mockButtonConfigService{getErr: errors.New("db error")}
	strategy := NewRemoteStrategy(svc, 1)
	cfg, err := strategy.Load()
	require.NoError(t, err)
	assert.Equal(t, defaultCfg, cfg)
}

func TestRemoteStrategy_Load_ValidConfig(t *testing.T) {
	dbCfg := Default()
	dbCfg.Keybindings["quit"] = "ctrl+q"
	data, err := json.Marshal(dbCfg)
	require.NoError(t, err)
	svc := &mockButtonConfigService{
		config: &model.ButtonConfiguration{Configuration: string(data)},
	}
	strategy := NewRemoteStrategy(svc, 1)
	cfg, err := strategy.Load()
	require.NoError(t, err)
	assert.Equal(t, "ctrl+q", cfg.Keybindings["quit"])
}

func TestRemoteStrategy_Load_InvalidJSON(t *testing.T) {
	defaultCfg := Default()
	svc := &mockButtonConfigService{
		config: &model.ButtonConfiguration{Configuration: "not json"},
	}
	strategy := NewRemoteStrategy(svc, 1)
	cfg, err := strategy.Load()
	require.NoError(t, err)
	assert.Equal(t, defaultCfg, cfg)
}

func TestRemoteStrategy_Load_MergesDefaults(t *testing.T) {
	partial := &RuntimeConfig{
		Keybindings: map[string]string{"quit": "ctrl+q"},
	}
	data, err := json.Marshal(partial)
	require.NoError(t, err)
	svc := &mockButtonConfigService{
		config: &model.ButtonConfiguration{Configuration: string(data)},
	}
	defaults := Default()
	strategy := NewRemoteStrategy(svc, 1)
	cfg, err := strategy.Load()
	require.NoError(t, err)
	assert.Equal(t, "ctrl+q", cfg.Keybindings["quit"])
	assert.Equal(t, defaults.Keybindings["nav_up"], cfg.Keybindings["nav_up"])
	// Configurations saved before the UI options existed get the UI defaults too.
	assert.Equal(t, DefaultVisibleColumns(), cfg.UI.VisibleColumns)
	assert.Equal(t, DefaultColumnOrder(), cfg.UI.ColumnOrder)
	assert.Equal(t, DefaultTheme, cfg.UI.Theme)
}

func TestRemoteStrategy_Save_NilService(t *testing.T) {
	strategy := NewRemoteStrategy(nil, 1)
	err := strategy.Save(Default())
	assert.ErrorIs(t, err, ErrNoRemoteServiceAvailable)
}

func TestRemoteStrategy_Save_Success(t *testing.T) {
	svc := &mockButtonConfigService{}
	strategy := NewRemoteStrategy(svc, 1)
	err := strategy.Save(Default())
	require.NoError(t, err)
	assert.True(t, svc.saveCalled)
}

func TestRemoteStrategy_Save_Error(t *testing.T) {
	svc := &mockButtonConfigService{saveErr: errors.New("save failed")}
	strategy := NewRemoteStrategy(svc, 1)
	err := strategy.Save(Default())
	assert.Error(t, err)
}

func TestRemoteStrategy_MigrateFromLocal_NilService(t *testing.T) {
	strategy := NewRemoteStrategy(nil, 1)
	strategy.MigrateFromLocal("/some/path")
}

func TestRemoteStrategy_MigrateFromLocal_EmptyPath(t *testing.T) {
	svc := &mockButtonConfigService{}
	strategy := NewRemoteStrategy(svc, 1)
	strategy.MigrateFromLocal("")
	assert.Empty(t, svc.migratePath)
}

func TestRemoteStrategy_MigrateFromLocal_FileNotExist(t *testing.T) {
	svc := &mockButtonConfigService{}
	strategy := NewRemoteStrategy(svc, 1)
	strategy.MigrateFromLocal("/nonexistent/path.json")
	assert.Empty(t, svc.migratePath)
}

func TestRemoteStrategy_MigrateFromLocal_FileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	err := os.WriteFile(path, []byte(`{}`), 0o644)
	require.NoError(t, err)
	svc := &mockButtonConfigService{}
	strategy := NewRemoteStrategy(svc, 1)
	strategy.MigrateFromLocal(path)
	assert.Equal(t, path, svc.migratePath)
}

func TestRemoteStrategy_MigrateFromLocal_MigrateError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	err := os.WriteFile(path, []byte(`{}`), 0o644)
	require.NoError(t, err)
	svc := &mockButtonConfigService{migrateErr: errors.New("migrate failed")}
	strategy := NewRemoteStrategy(svc, 1)
	strategy.MigrateFromLocal(path)
}

func TestNewManager_Local(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	m, err := NewManager(true, path, nil, 0)
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.IsType(t, &LocalStrategy{}, m.strategy)
}

func TestNewManager_Remote(t *testing.T) {
	svc := &mockButtonConfigService{getErr: errors.New("no config")}
	m, err := NewManager(false, "", svc, 42)
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.IsType(t, &RemoteStrategy{}, m.strategy)
	assert.Equal(t, Default(), m.Get(), "falls back to defaults without a stored config")
}

func TestManager_Set_SavesRemote(t *testing.T) {
	svc := &mockButtonConfigService{getErr: errors.New("no config")}
	m, err := NewManager(false, "", svc, 1)
	require.NoError(t, err)
	newCfg := Default()
	newCfg.Keybindings["quit"] = "ctrl+q"
	require.NoError(t, m.Set(newCfg))
	assert.True(t, svc.saveCalled)
	assert.Equal(t, "ctrl+q", svc.savedConfig.Keybindings["quit"])
}

func TestBuildKeyToActionMap(t *testing.T) {
	bindings := map[string]string{
		"quit":   "ctrl+c",
		"select": "enter",
	}
	m := buildKeyToActionMap(bindings)
	assert.Equal(t, "quit", m["ctrl+c"])
	assert.Equal(t, "select", m["enter"])
	assert.Empty(t, m["nonexistent"])
}
