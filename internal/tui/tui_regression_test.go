package tui

import (
	"database/sql"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/services/usercollection"
	"github.com/laiambryant/tui-cardman/internal/testutil"
)

func keyRunes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

var (
	keyEsc       = tea.KeyMsg{Type: tea.KeyEsc}
	keyF1        = tea.KeyMsg{Type: tea.KeyF1}
	keyTab       = tea.KeyMsg{Type: tea.KeyTab}
	keyBackspace = tea.KeyMsg{Type: tea.KeyBackspace}
	keyCtrlC     = tea.KeyMsg{Type: tea.KeyCtrlC}
)

func setupTUIDB(t *testing.T) *sql.DB {
	t.Helper()
	db := testutil.SetupTestDB(t)
	t.Cleanup(func() { testutil.CleanupTestDB(t, db) })
	testutil.ApplyTestMigrations(t, db, "../db/migrations")
	t.Setenv("CARDMAN_CONFIG", t.TempDir()+"/cardman.json")
	return db
}

// update sends msgs to the model in order and returns the resulting model.
func update(t *testing.T, m tea.Model, msgs ...tea.Msg) *Model {
	t.Helper()
	for _, msg := range msgs {
		m, _ = m.Update(msg)
	}
	return m.(*Model)
}

func TestSettingsReturnsToPreviousScreen(t *testing.T) {
	db := setupTUIDB(t)
	m, err := NewModel(db, false) // no users yet: local profile setup
	require.NoError(t, err)
	m = update(t, m, splashDoneMsg{})
	require.Equal(t, ScreenLocalUserSetup, m.screen)

	m = update(t, m, keyF1)
	require.Equal(t, ScreenSettings, m.screen)
	m = update(t, m, keyEsc)
	assert.Equal(t, ScreenLocalUserSetup, m.screen, "closing settings must not skip profile setup")
	assert.Nil(t, m.user)

	m = update(t, m, keyF1, keyCtrlC)
	assert.Equal(t, ScreenLocalUserSetup, m.screen, "ctrl+c in settings closes settings")
}

func TestLocalProfileSetupEntersMainScreenWithStats(t *testing.T) {
	db := setupTUIDB(t)
	m, err := NewModel(db, false)
	require.NoError(t, err)
	m = update(t, m, splashDoneMsg{})
	m.inputs[0].SetValue("Ash")
	m.inputs[1].SetValue("Ketchum")
	m.inputs[2].SetValue("ash@example.com")
	m.focusIndex = len(m.inputs)

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = update(t, next)
	require.Equal(t, ScreenMain, m.screen)
	require.NotNil(t, m.user)
	assert.Empty(t, m.errorMsg)
	assert.True(t, m.statsLoading)
	require.NotNil(t, cmd)
	_, ok := cmd().(gameStatsMsg)
	assert.True(t, ok, "stats for the highlighted game are loaded")
}

func TestSSHRegisterInitializesUserConfig(t *testing.T) {
	db := setupTUIDB(t)
	m, err := NewModel(db, true)
	require.NoError(t, err)
	m = update(t, m, splashDoneMsg{})
	m.screen = ScreenRegister
	m.initRegisterInputs()
	m.inputs[0].SetValue("Misty")
	m.inputs[1].SetValue("Waterflower")
	m.inputs[2].SetValue("misty@example.com")
	m.inputs[3].SetValue("Str0ng!Pass")
	m.focusIndex = len(m.inputs)

	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	require.Equal(t, ScreenMain, m.screen, m.errorMsg)
	require.NotNil(t, m.configManager, "a newly registered SSH user needs keybindings")
	assert.Equal(t, "save", GetAction(m.configManager, "ctrl+s"))
}

func TestInputScreensUseArrowKeysForCursor(t *testing.T) {
	db := setupTUIDB(t)
	m, err := NewModel(db, false)
	require.NoError(t, err)
	m = update(t, m, splashDoneMsg{}, keyRunes("Ah"), tea.KeyMsg{Type: tea.KeyLeft}, keyRunes("s"))
	assert.Equal(t, 0, m.focusIndex, "left/right move the cursor, not the focus")
	assert.Equal(t, "Ash", m.inputs[0].Value())
}

func TestListFormEditingKeys(t *testing.T) {
	lm := NewListsModel(&model.CardGame{ID: 1, Name: "Pokemon"}, nil, nil, nil, nil, nil, NewStyleManager())
	lm.mode = ListsModeCreate
	lm.nameInput.Focus()
	paste := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Wantz"), Paste: true}
	for _, msg := range []tea.Msg{paste, keyBackspace, keyRunes("s"), tea.KeyMsg{Type: tea.KeyLeft}} {
		lm, _ = lm.Update(msg)
	}
	assert.Equal(t, "Wants", lm.nameInput.Value(), "pasted text is not wrapped in brackets")
}

func TestCSVImportPathTyping(t *testing.T) {
	sm := NewStyleManager()
	dm := NewDeckBuilderModel(&model.CardGame{ID: 1}, nil, nil, nil, nil, nil, sm)
	dm.importState = NewImportState(nil, 1, sm)
	for _, r := range "deck.csvx" {
		dm, _ = dm.Update(keyRunes(string(r)))
	}
	dm, _ = dm.Update(keyBackspace)
	assert.Equal(t, "deck.csv", dm.importState.pathInput.Value(), "each key is applied once")

	lm := NewListsModel(&model.CardGame{ID: 1}, nil, nil, nil, nil, nil, sm)
	lm.importState = NewImportState(nil, 1, sm)
	for _, r := range "list.csv" {
		lm, _ = lm.Update(keyRunes(string(r)))
	}
	assert.Equal(t, "list.csv", lm.importState.pathInput.Value())
	lm, _ = lm.Update(keyEsc)
	assert.False(t, lm.importState.active)
}

func TestCardSearchTabKeys(t *testing.T) {
	sm := NewStyleManager()
	ct := NewCardGameTabsModel(&model.CardGame{ID: 1, Name: "Pokemon"}, nil, sm)
	ct.cards = []model.Card{{ID: 1, Name: "Squirtle"}, {ID: 2, Name: "Pikachu"}}
	ct.currentTab = TabCardSearch
	ct = ct.updateTableForTab()

	ct, _ = ct.Update(keyTab)
	assert.Equal(t, TabCardSearch, ct.currentTab, "tab switches panel inside the Card Search tab")
	assert.Equal(t, 1, ct.searchTabFocus)
	ct, _ = ct.Update(keyTab)
	assert.Equal(t, 0, ct.searchTabFocus)

	ct, _ = ct.Update(keyRunes("q"))
	assert.False(t, ct.shouldGoBack, "q is typed into the focused search box")
	assert.Equal(t, "q", ct.searchInput.Value())

	ct, _ = ct.Update(keyEsc)
	assert.True(t, ct.shouldGoBack)
}

func TestTruncateIsRuneSafe(t *testing.T) {
	assert.Equal(t, "Poké...", Truncate("Pokémon Trainer", 7))
	assert.Equal(t, "日本...", Truncate("日本語のカード名", 5))
	assert.NotPanics(t, func() { Truncate("Card Games", 1) })
	assert.NotPanics(t, func() { Truncate("Card Games", 0) })
	assert.Equal(t, "C", Truncate("Card Games", 1))
}

func TestImportModelStartsLoading(t *testing.T) {
	im := NewImportModel(navigationTestImporter{}, &model.CardGame{ID: 1, Name: "Pokemon"}, nil, NewStyleManager())
	assert.True(t, im.isLoading)
	im, _ = im.Update(fetchSetsSuccessMsg{})
	assert.False(t, im.isLoading)
}

func TestCalcValueChangeUsesDates(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC) }
	// Snapshots are only taken on days the collection was opened.
	history := []usercollection.ValueSnapshot{
		{Date: day(1), Value: 100},
		{Date: day(20), Value: 150},
		{Date: day(28), Value: 200},
		{Date: day(30), Value: 220},
	}
	change, ok := calcValueChange(history, 7)
	require.True(t, ok)
	assert.InDelta(t, 46.67, change, 0.01, "compares with the last value at least 7 days old (day 20)")

	change, ok = calcValueChange(history, 30)
	require.True(t, ok)
	assert.InDelta(t, 120, change, 0.01, "falls back to the oldest snapshot")
}

func TestSetProgressGridOrdersNumbersNaturally(t *testing.T) {
	cards := []model.Card{{ID: 1, Number: "10"}, {ID: 2, Number: "2"}, {ID: 3, Number: "1"}, {ID: 4, Number: "SV1"}}
	sortCardsByNumber(cards)
	var got []string
	for _, c := range cards {
		got = append(got, c.Number)
	}
	assert.Equal(t, []string{"1", "2", "10", "SV1"}, got)
}

func TestFuzzySearchIsDeterministic(t *testing.T) {
	cards := []model.Card{{ID: 1, Name: "Pikachu"}, {ID: 2, Name: "Pikachu"}, {ID: 3, Name: "Pikachu"}, {ID: 4, Name: "Pikachu"}}
	for range 20 {
		results := fuzzySearchCards(cards, "pika")
		require.Len(t, results, 4)
		for i, r := range results {
			assert.Equal(t, int64(i+1), r.Card.ID, "equal scores keep the original order")
		}
	}
}

func TestRarityBreakdownIsDeterministic(t *testing.T) {
	ct := NewCardGameTabsModel(&model.CardGame{ID: 1}, nil, NewStyleManager())
	for _, r := range []string{"Rare", "Common", "Uncommon", "Rare Holo"} {
		ct.userCollections = append(ct.userCollections, model.UserCollection{Quantity: 1, Card: &model.Card{Rarity: r}})
	}
	ct.computeCollectionStats()
	first := ct.rarityBreakdown
	for range 20 {
		ct.computeCollectionStats()
		assert.Equal(t, first, ct.rarityBreakdown)
	}
	assert.Equal(t, "C:1 R:1 RH:1 U:1", first, "rarities are abbreviated by word initials")
}
