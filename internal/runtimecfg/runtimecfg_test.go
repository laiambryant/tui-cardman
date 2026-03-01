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
	err := os.WriteFile(path, []byte(data), 0644)
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
	err := os.WriteFile(path, []byte(data), 0644)
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
	err := os.WriteFile(path, []byte(data), 0644)
	require.NoError(t, err)

	cfg, err := Load(path)
	require.NoError(t, err)

	defaults := Default()
	assert.Equal(t, defaults.Keybindings, cfg.Keybindings)
}

func TestLoad_InvalidJSON_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.json")

	err := os.WriteFile(path, []byte(`{not valid json`), 0644)
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

func TestManager_Get_ReturnsDefensiveCopy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	strategy := NewLocalStrategy(path)
	cfg := Default()

	m := &Manager{
		config:      cfg,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(cfg.Keybindings),
	}

	got1 := m.Get()
	got1.Keybindings["quit"] = "MUTATED"

	got2 := m.Get()
	assert.NotEqual(t, "MUTATED", got2.Keybindings["quit"], "Get() should return a defensive copy")
}

func TestManager_KeyForAction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	strategy := NewLocalStrategy(path)
	cfg := Default()

	m := &Manager{
		config:      cfg,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(cfg.Keybindings),
	}

	key := m.KeyForAction("quit")
	assert.Equal(t, cfg.Keybindings["quit"], key)
}

func TestManager_KeyForAction_UnknownAction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	strategy := NewLocalStrategy(path)
	cfg := Default()

	m := &Manager{
		config:      cfg,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(cfg.Keybindings),
	}

	key := m.KeyForAction("nonexistent_action")
	assert.Empty(t, key)
}

func TestManager_MatchAction_AllBindings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	strategy := NewLocalStrategy(path)
	cfg := Default()

	m := &Manager{
		config:      cfg,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(cfg.Keybindings),
	}

	// Every keybinding should match its action
	for action, key := range cfg.Keybindings {
		matched := m.MatchAction(key)
		assert.Equal(t, action, matched, "key %q should match action %q", key, action)
	}
}

func TestManager_MatchAction_WithFilteredActions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	strategy := NewLocalStrategy(path)
	cfg := Default()

	m := &Manager{
		config:      cfg,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(cfg.Keybindings),
	}

	quitKey := cfg.Keybindings["quit"]

	// Match against a list that includes "quit"
	matched := m.MatchAction(quitKey, "quit", "select")
	assert.Equal(t, "quit", matched)

	// Match against a list that does NOT include "quit"
	notMatched := m.MatchAction(quitKey, "select", "nav_up")
	assert.Empty(t, notMatched)
}

func TestManager_SetKeybinding_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	strategy := NewLocalStrategy(path)
	cfg := Default()

	m := &Manager{
		config:      cfg,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(cfg.Keybindings),
	}

	err := m.SetKeybinding("quit", "ctrl+q")
	require.NoError(t, err)
	assert.Equal(t, "ctrl+q", m.KeyForAction("quit"))
}

func TestManager_SetKeybinding_ConflictRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	strategy := NewLocalStrategy(path)
	cfg := Default()

	m := &Manager{
		config:      cfg,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(cfg.Keybindings),
	}

	// "quit" is currently bound to ctrl+c; try binding "select" to the same key
	quitKey := cfg.Keybindings["quit"]
	err := m.SetKeybinding("select", quitKey)
	assert.Error(t, err, "binding two actions to the same key should fail")
}

func TestManager_Subscribe_Notified(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	strategy := NewLocalStrategy(path)
	cfg := Default()

	m := &Manager{
		config:      cfg,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(cfg.Keybindings),
	}

	var notified bool
	m.Subscribe(func(c *RuntimeConfig) {
		notified = true
	})

	newCfg := Default()
	newCfg.Keybindings["quit"] = "ctrl+q"
	err := m.Set(newCfg)
	require.NoError(t, err)

	assert.True(t, notified, "subscriber should have been called on Set()")
}

func TestManager_ConcurrentGet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	strategy := NewLocalStrategy(path)
	cfg := Default()

	m := &Manager{
		config:      cfg,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(cfg.Keybindings),
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got := m.Get()
			assert.NotNil(t, got)
		}()
	}
	wg.Wait()
}

func TestManager_HasUnsavedChanges_LocalStrategy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	strategy := NewLocalStrategy(path)
	cfg := Default()

	m := &Manager{
		config:      cfg,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(cfg.Keybindings),
	}

	// LocalStrategy.HasUnsavedChanges always returns false
	assert.False(t, m.HasUnsavedChanges())
}

func TestManager_IsUserMode_LocalStrategy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	strategy := NewLocalStrategy(path)
	cfg := Default()

	m := &Manager{
		config:      cfg,
		strategy:    strategy,
		subscribers: make([]Subscriber, 0),
		keyToAction: buildKeyToActionMap(cfg.Keybindings),
	}

	assert.False(t, m.IsUserMode())
}

// --- LocalStrategy tests ---

func TestLocalStrategy_Load_CreatesDefault_WhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	strategy := NewLocalStrategy(path)
	cfg, err := strategy.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	defaults := Default()
	assert.Equal(t, defaults.Keybindings, cfg.Keybindings)
}

func TestLocalStrategy_Save_And_Load(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	strategy := NewLocalStrategy(path)
	cfg := Default()
	cfg.Keybindings["quit"] = "ctrl+q"

	err := strategy.Save(cfg)
	require.NoError(t, err)

	loaded, err := strategy.Load()
	require.NoError(t, err)
	assert.Equal(t, "ctrl+q", loaded.Keybindings["quit"])
}

func TestLocalStrategy_HasUnsavedChanges_AlwaysFalse(t *testing.T) {
	strategy := NewLocalStrategy("/tmp/test.json")
	assert.False(t, strategy.HasUnsavedChanges())
	strategy.MarkUnsaved()
	assert.False(t, strategy.HasUnsavedChanges())
}

func TestLocalStrategy_IsAvailable(t *testing.T) {
	strategy := NewLocalStrategy("/tmp/test.json")
	assert.True(t, strategy.IsAvailable())
}

// --- validateKeybindings tests ---

func TestValidateKeybindings_EmptyKey_Rejected(t *testing.T) {
	m := &Manager{
		config:      Default(),
		subscribers: make([]Subscriber, 0),
	}

	err := m.validateKeybindings(map[string]string{
		"quit":   "",
		"select": "enter",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not bound to any key")
}

func TestValidateKeybindings_DuplicateKeys_Rejected(t *testing.T) {
	m := &Manager{
		config:      Default(),
		subscribers: make([]Subscriber, 0),
	}

	err := m.validateKeybindings(map[string]string{
		"quit":   "ctrl+c",
		"select": "ctrl+c",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bound to both")
}

func TestValidateKeybindings_Valid(t *testing.T) {
	m := &Manager{
		config:      Default(),
		subscribers: make([]Subscriber, 0),
	}

	err := m.validateKeybindings(map[string]string{
		"quit":   "ctrl+c",
		"select": "enter",
	})
	assert.NoError(t, err)
}
