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
	initErr     error
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

func (m *mockButtonConfigService) InitializeDefault(_ context.Context, _ int64) error {
	return m.initErr
}

func (m *mockButtonConfigService) MigrateLocalToDB(_ context.Context, _ int64, localPath string) error {
	m.migratePath = localPath
	return m.migrateErr
}

func TestRemoteStrategy_Load_NilService(t *testing.T) {
	defaultCfg := Default()
	strategy := NewRemoteStrategy(nil, 1, defaultCfg)
	cfg, err := strategy.Load()
	require.NoError(t, err)
	assert.Equal(t, defaultCfg, cfg)
}

func TestRemoteStrategy_Load_ServiceError(t *testing.T) {
	defaultCfg := Default()
	svc := &mockButtonConfigService{getErr: errors.New("db error")}
	strategy := NewRemoteStrategy(svc, 1, defaultCfg)
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
	strategy := NewRemoteStrategy(svc, 1, Default())
	cfg, err := strategy.Load()
	require.NoError(t, err)
	assert.Equal(t, "ctrl+q", cfg.Keybindings["quit"])
}

func TestRemoteStrategy_Load_InvalidJSON(t *testing.T) {
	defaultCfg := Default()
	svc := &mockButtonConfigService{
		config: &model.ButtonConfiguration{Configuration: "not json"},
	}
	strategy := NewRemoteStrategy(svc, 1, defaultCfg)
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
	strategy := NewRemoteStrategy(svc, 1, defaults)
	cfg, err := strategy.Load()
	require.NoError(t, err)
	assert.Equal(t, "ctrl+q", cfg.Keybindings["quit"])
	assert.Equal(t, defaults.Keybindings["nav_up"], cfg.Keybindings["nav_up"])
}

func TestRemoteStrategy_Save_NilService(t *testing.T) {
	strategy := NewRemoteStrategy(nil, 1, Default())
	err := strategy.Save(Default())
	assert.ErrorIs(t, err, ErrNoRemoteServiceAvailable)
}

func TestRemoteStrategy_Save_Success(t *testing.T) {
	svc := &mockButtonConfigService{}
	strategy := NewRemoteStrategy(svc, 1, Default())
	strategy.MarkUnsaved()
	assert.True(t, strategy.HasUnsavedChanges())
	err := strategy.Save(Default())
	require.NoError(t, err)
	assert.True(t, svc.saveCalled)
	assert.False(t, strategy.HasUnsavedChanges())
}

func TestRemoteStrategy_Save_Error(t *testing.T) {
	svc := &mockButtonConfigService{saveErr: errors.New("save failed")}
	strategy := NewRemoteStrategy(svc, 1, Default())
	err := strategy.Save(Default())
	assert.Error(t, err)
}

func TestRemoteStrategy_HasUnsavedChanges(t *testing.T) {
	strategy := NewRemoteStrategy(nil, 1, Default())
	assert.False(t, strategy.HasUnsavedChanges())
	strategy.MarkUnsaved()
	assert.True(t, strategy.HasUnsavedChanges())
}

func TestRemoteStrategy_IsAvailable(t *testing.T) {
	assert.False(t, NewRemoteStrategy(nil, 1, Default()).IsAvailable())
	assert.True(t, NewRemoteStrategy(&mockButtonConfigService{}, 1, Default()).IsAvailable())
}

func TestRemoteStrategy_MigrateFromLocal_NilService(t *testing.T) {
	strategy := NewRemoteStrategy(nil, 1, Default())
	strategy.MigrateFromLocal("/some/path")
}

func TestRemoteStrategy_MigrateFromLocal_EmptyPath(t *testing.T) {
	svc := &mockButtonConfigService{}
	strategy := NewRemoteStrategy(svc, 1, Default())
	strategy.MigrateFromLocal("")
	assert.Empty(t, svc.migratePath)
}

func TestRemoteStrategy_MigrateFromLocal_FileNotExist(t *testing.T) {
	svc := &mockButtonConfigService{}
	strategy := NewRemoteStrategy(svc, 1, Default())
	strategy.MigrateFromLocal("/nonexistent/path.json")
	assert.Empty(t, svc.migratePath)
}

func TestRemoteStrategy_MigrateFromLocal_FileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	err := os.WriteFile(path, []byte(`{}`), 0o644)
	require.NoError(t, err)
	svc := &mockButtonConfigService{}
	strategy := NewRemoteStrategy(svc, 1, Default())
	strategy.MigrateFromLocal(path)
	assert.Equal(t, path, svc.migratePath)
}

func TestRemoteStrategy_MigrateFromLocal_MigrateError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	err := os.WriteFile(path, []byte(`{}`), 0o644)
	require.NoError(t, err)
	svc := &mockButtonConfigService{migrateErr: errors.New("migrate failed")}
	strategy := NewRemoteStrategy(svc, 1, Default())
	strategy.MigrateFromLocal(path)
}

func TestNewManager_Local(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	m, err := NewManager(true, path, nil, 0)
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.False(t, m.IsUserMode())
}

func TestNewManager_Remote(t *testing.T) {
	svc := &mockButtonConfigService{getErr: errors.New("no config")}
	m, err := NewManager(false, "", svc, 42)
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.True(t, m.IsUserMode())
}

func TestManager_Set_NotifiesAndSaves(t *testing.T) {
	svc := &mockButtonConfigService{}
	strategy := NewRemoteStrategy(svc, 1, Default())
	cfg := Default()
	m := &Manager{
		config:      cfg,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(cfg.Keybindings),
	}
	notified := false
	m.Subscribe(func(c *RuntimeConfig) { notified = true })
	newCfg := Default()
	newCfg.Keybindings["quit"] = "ctrl+q"
	err := m.Set(newCfg)
	require.NoError(t, err)
	assert.True(t, notified)
	assert.True(t, svc.saveCalled)
	assert.Equal(t, "ctrl+q", m.Get().Keybindings["quit"])
}

func TestManager_Set_ValidationFailure(t *testing.T) {
	svc := &mockButtonConfigService{}
	strategy := NewRemoteStrategy(svc, 1, Default())
	cfg := Default()
	m := &Manager{
		config:      cfg,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(cfg.Keybindings),
	}
	badCfg := &RuntimeConfig{
		Keybindings: map[string]string{
			"quit":   "ctrl+c",
			"select": "ctrl+c",
		},
	}
	err := m.Set(badCfg)
	assert.Error(t, err)
	assert.False(t, svc.saveCalled)
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
