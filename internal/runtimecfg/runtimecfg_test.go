package runtimecfg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- RuntimeConfig / Load / Save tests ---

func TestDefault_HasRequiredKeybindings(t *testing.T) {
	cfg := Default()
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Keybindings)

	requiredActions := []string{
		"quit", "quit_alt", "settings",
		"nav_up", "nav_down", "nav_left", "nav_right",
		"nav_prev_tab", "nav_next_tab",
		"select", "back",
		"tab_all_cards", "tab_collection", "tab_search",
		"switch_tab_left", "switch_tab_right",
		"search_focus", "search_clear",
		"save", "increment_quantity", "decrement_quantity",
	}

	for _, action := range requiredActions {
		_, exists := cfg.Keybindings[action]
		assert.True(t, exists, "default config missing keybinding for action: %s", action)
	}
}

func TestDefault_NoEmptyKeys(t *testing.T) {
	cfg := Default()
	for action, key := range cfg.Keybindings {
		assert.NotEmpty(t, key, "default keybinding for action %q must not be empty", action)
	}
}

func TestDefault_UIDefaults(t *testing.T) {
	cfg := Default()
	assert.False(t, cfg.UI.CompactLists)
}

func TestLoad_MissingFile_ReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")

	cfg, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	defaults := Default()
	assert.Equal(t, defaults.Keybindings, cfg.Keybindings)
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{
		"keybindings": {
			"quit": "ctrl+q",
			"select": "space"
		},
		"ui": {
			"compact_lists": true
		}
	}`
	err := os.WriteFile(path, []byte(data), 0o644)
	require.NoError(t, err)

	cfg, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "ctrl+q", cfg.Keybindings["quit"])
	assert.Equal(t, "space", cfg.Keybindings["select"])
	assert.True(t, cfg.UI.CompactLists)
}

func TestLoad_MergesDefaults_ForMissingKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "partial.json")

	// Only override one keybinding; the rest should come from defaults
	data := `{"keybindings": {"quit": "ctrl+q"}}`
	err := os.WriteFile(path, []byte(data), 0o644)
	require.NoError(t, err)

	cfg, err := Load(path)
	require.NoError(t, err)

	defaults := Default()
	// The overridden key
	assert.Equal(t, "ctrl+q", cfg.Keybindings["quit"])
	// A non-overridden key should still have the default
	assert.Equal(t, defaults.Keybindings["nav_up"], cfg.Keybindings["nav_up"])
}

func TestLoad_NullKeybindings_UsesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "null_kb.json")

	data := `{"keybindings": null, "ui": {"compact_lists": false}}`
	err := os.WriteFile(path, []byte(data), 0o644)
	require.NoError(t, err)

	cfg, err := Load(path)
	require.NoError(t, err)

	defaults := Default()
	assert.Equal(t, defaults.Keybindings, cfg.Keybindings)
}

func TestLoad_InvalidJSON_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.json")

	err := os.WriteFile(path, []byte(`{not valid json`), 0o644)
	require.NoError(t, err)

	_, err = Load(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestSave_WritesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "saved.json")

	cfg := Default()
	cfg.Keybindings["quit"] = "ctrl+q"
	cfg.UI.CompactLists = true

	err := Save(cfg, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var loaded RuntimeConfig
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, "ctrl+q", loaded.Keybindings["quit"])
	assert.True(t, loaded.UI.CompactLists)
}

func TestSave_CreatesIntermediateDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "nested", "config.json")

	cfg := Default()
	err := Save(cfg, path)
	require.NoError(t, err)

	_, err = os.Stat(path)
	assert.NoError(t, err, "config file should exist after Save")
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.json")

	original := Default()
	original.Keybindings["quit"] = "ctrl+x"
	original.UI.CompactLists = true

	err := Save(original, path)
	require.NoError(t, err)

	loaded, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, original.Keybindings, loaded.Keybindings)
	assert.Equal(t, original.UI, loaded.UI)
}

func TestGetConfigPath_DefaultPath(t *testing.T) {
	// Ensure the env var is not set
	t.Setenv("CARDMAN_CONFIG", "")

	path := GetConfigPath()
	assert.Equal(t, ".cardman.json", path)
}

func TestGetConfigPath_EnvOverride(t *testing.T) {
	t.Setenv("CARDMAN_CONFIG", "/custom/path/config.json")

	path := GetConfigPath()
	assert.Equal(t, "/custom/path/config.json", path)
}

// --- Manager tests ---

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	m, err := NewManager(true, filepath.Join(t.TempDir(), "config.json"), nil, 0)
	require.NoError(t, err)
	return m
}

func TestManager_Get_ReturnsDefensiveCopy(t *testing.T) {
	m := newTestManager(t)

	got1 := m.Get()
	got1.Keybindings["quit"] = "MUTATED"
	got1.UI.VisibleColumns["artist"] = true
	got1.UI.ColumnOrder[0] = "MUTATED"

	got2 := m.Get()
	assert.NotEqual(t, "MUTATED", got2.Keybindings["quit"], "Get() should return a defensive copy")
	assert.False(t, got2.UI.VisibleColumns["artist"])
	assert.Equal(t, "name", got2.UI.ColumnOrder[0])
}

func TestManager_KeyForAction(t *testing.T) {
	m := newTestManager(t)
	assert.Equal(t, Default().Keybindings["quit"], m.KeyForAction("quit"))
	assert.Empty(t, m.KeyForAction("nonexistent_action"))
}

func TestManager_MatchAction_AllBindings(t *testing.T) {
	m := newTestManager(t)
	for action, key := range Default().Keybindings {
		assert.Equal(t, action, m.MatchAction(key), "key %q should match action %q", key, action)
	}
	assert.Empty(t, m.MatchAction("ctrl+shift+z"))
}

func TestManager_Set_SavesAndRebuildsIndex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	m, err := NewManager(true, path, nil, 0)
	require.NoError(t, err)

	cfg := m.Get()
	cfg.Keybindings["quit"] = "ctrl+q"
	require.NoError(t, m.Set(cfg))

	assert.Equal(t, "quit", m.MatchAction("ctrl+q"))
	assert.Empty(t, m.MatchAction("ctrl+c"))
	saved, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "ctrl+q", saved.Keybindings["quit"])
}

func TestManager_Set_ValidationFailure(t *testing.T) {
	m := newTestManager(t)
	err := m.Set(&RuntimeConfig{Keybindings: map[string]string{"quit": "ctrl+c", "select": "ctrl+c"}})
	assert.Error(t, err)
	assert.Equal(t, "quit", m.MatchAction("ctrl+c"), "a rejected config is not applied")
}

func TestManager_ConcurrentGet(t *testing.T) {
	m := newTestManager(t)
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NotNil(t, m.Get())
		}()
	}
	wg.Wait()
}

// --- LocalStrategy tests ---

func TestLocalStrategy_Load_CreatesDefault_WhenMissing(t *testing.T) {
	strategy := NewLocalStrategy(filepath.Join(t.TempDir(), "config.json"))
	cfg, err := strategy.Load()
	require.NoError(t, err)
	assert.Equal(t, Default(), cfg)
}

func TestLocalStrategy_Save_And_Load(t *testing.T) {
	strategy := NewLocalStrategy(filepath.Join(t.TempDir(), "config.json"))
	cfg := Default()
	cfg.Keybindings["quit"] = "ctrl+q"
	require.NoError(t, strategy.Save(cfg))

	loaded, err := strategy.Load()
	require.NoError(t, err)
	assert.Equal(t, "ctrl+q", loaded.Keybindings["quit"])
}

// --- validateKeybindings tests ---

func TestValidateKeybindings(t *testing.T) {
	err := validateKeybindings(map[string]string{"quit": "", "select": "enter"})
	assert.ErrorContains(t, err, "not bound to any key")

	err = validateKeybindings(map[string]string{"quit": "ctrl+c", "select": "ctrl+c"})
	assert.ErrorContains(t, err, "bound to both")

	assert.NoError(t, validateKeybindings(map[string]string{"quit": "ctrl+c", "select": "enter"}))
}
