package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/laiambryant/tui-cardman/internal/model"
)

func TestIsModifierKey_CtrlCombos(t *testing.T) {
	assert.True(t, isModifierKey("ctrl+c"))
	assert.True(t, isModifierKey("ctrl+s"))
	assert.True(t, isModifierKey("ctrl+a"))
}

func TestIsModifierKey_AltCombos(t *testing.T) {
	assert.True(t, isModifierKey("alt+x"))
	assert.True(t, isModifierKey("alt+enter"))
}

func TestIsModifierKey_SpecialKeys(t *testing.T) {
	specials := []string{
		"tab", "shift+tab", "enter", "\r", "\n",
		"esc", "up", "down", "left", "right",
		"home", "end", "pgup", "pgdown",
		"f1", "f2", "f3", "f4", "f5", "f6",
		"f7", "f8", "f9", "f10", "f11", "f12",
	}
	for _, key := range specials {
		assert.True(t, isModifierKey(key), "expected %q to be a modifier key", key)
	}
}

func TestIsModifierKey_PrintableCharacters(t *testing.T) {
	printable := []string{"a", "z", "1", "0", "/", " ", "+", "-", ".", "!"}
	for _, key := range printable {
		assert.False(t, isModifierKey(key), "expected %q to NOT be a modifier key", key)
	}
}

func TestTruncate_ShortString(t *testing.T) {
	assert.Equal(t, "hello", Truncate("hello", 10))
}

func TestTruncate_ExactLength(t *testing.T) {
	assert.Equal(t, "hello", Truncate("hello", 5))
}

func TestTruncate_LongString(t *testing.T) {
	assert.Equal(t, "hell...", Truncate("hello world", 7))
}

func TestTruncate_VeryShort(t *testing.T) {
	assert.Equal(t, "...", Truncate("hello world", 3))
}

func TestHelpBuilder_Build_NilConfig(t *testing.T) {
	hb := NewHelpBuilder(nil)
	result := hb.Build(
		KeyItem{"quit", "ctrl+c", "Quit"},
		KeyItem{"select", "enter", "Select"},
	)
	assert.Equal(t, "ctrl+c: Quit | enter: Select", result)
}

func TestHelpBuilder_Build_Empty(t *testing.T) {
	hb := NewHelpBuilder(nil)
	result := hb.Build()
	assert.Equal(t, "", result)
}

func TestHelpBuilder_Build_SingleItem(t *testing.T) {
	hb := NewHelpBuilder(nil)
	result := hb.Build(KeyItem{"quit", "ctrl+c", "Quit"})
	assert.Equal(t, "ctrl+c: Quit", result)
}

func TestHelpBuilder_Pair_NilConfig(t *testing.T) {
	hb := NewHelpBuilder(nil)
	result := hb.Pair("page_next", "Ctrl+N", "page_prev", "Ctrl+P", "Page")
	assert.Equal(t, "Ctrl+N / Ctrl+P: Page", result)
}

func TestCardToDataMap_WithSet(t *testing.T) {
	card := model.Card{
		Name:   "Pikachu",
		Rarity: "Common",
		Number: "025",
		Artist: "Ken Sugimori",
		Set:    &model.Set{Name: "Base Set"},
	}
	data := CardToDataMap(card, 3, 1)
	assert.Equal(t, "Pikachu", data["name"])
	assert.Equal(t, "Base Set", data["expansion"])
	assert.Equal(t, "Common", data["rarity"])
	assert.Equal(t, "025", data["number"])
	assert.Equal(t, "4", data["quantity"])
	assert.Equal(t, "Ken Sugimori", data["artist"])
}

func TestCardToDataMap_WithoutSet(t *testing.T) {
	card := model.Card{Name: "Pikachu", SetID: 42}
	data := CardToDataMap(card, 1, 0)
	assert.Equal(t, "Set#42", data["expansion"])
}

func TestCardToDataMap_NoSetInfo(t *testing.T) {
	card := model.Card{Name: "Pikachu"}
	data := CardToDataMap(card, 0, 0)
	assert.Equal(t, "", data["expansion"])
}

func TestCollectionToDataMap_WithCard(t *testing.T) {
	c := model.UserCollection{
		Quantity: 5,
		Card: &model.Card{
			Name:   "Charizard",
			Rarity: "Rare",
			Artist: "Mitsuhiro Arita",
			Set:    &model.Set{Name: "Base Set"},
		},
	}
	data := CollectionToDataMap(c)
	assert.Equal(t, "Charizard", data["name"])
	assert.Equal(t, "Base Set", data["expansion"])
	assert.Equal(t, "Rare", data["rarity"])
	assert.Equal(t, "5", data["quantity"])
	assert.Equal(t, "Mitsuhiro Arita", data["artist"])
}

func TestCollectionToDataMap_NilCard(t *testing.T) {
	c := model.UserCollection{Quantity: 2}
	data := CollectionToDataMap(c)
	assert.Equal(t, "Unknown Card", data["name"])
	assert.Equal(t, "", data["expansion"])
	assert.Equal(t, "", data["rarity"])
	assert.Equal(t, "2", data["quantity"])
}

func TestCollectionToDataMap_CardWithSetID(t *testing.T) {
	c := model.UserCollection{
		Quantity: 1,
		Card:     &model.Card{Name: "Pikachu", SetID: 10},
	}
	data := CollectionToDataMap(c)
	assert.Equal(t, "Set#10", data["expansion"])
}

func TestBuildVisibleColumnSet_AllVisible(t *testing.T) {
	visible := map[string]bool{"name": true, "expansion": true, "rarity": true}
	order := []string{"name", "expansion", "rarity"}
	cols := []ColumnDef{
		{"name", "Name", 40, 8},
		{"expansion", "Expansion", 30, 5},
		{"rarity", "Rarity", 30, 5},
	}
	vcs := BuildVisibleColumnSet(cols, visible, order, 100)
	assert.Len(t, vcs.Columns, 3)
	assert.Len(t, vcs.Keys, 3)
	assert.Equal(t, "name", vcs.Keys[0])
	assert.Equal(t, "expansion", vcs.Keys[1])
	assert.Equal(t, "rarity", vcs.Keys[2])
}

func TestBuildVisibleColumnSet_SubsetVisible(t *testing.T) {
	visible := map[string]bool{"name": true, "rarity": true, "expansion": false}
	order := []string{"name", "expansion", "rarity"}
	cols := []ColumnDef{
		{"name", "Name", 40, 8},
		{"expansion", "Expansion", 30, 5},
		{"rarity", "Rarity", 30, 5},
	}
	vcs := BuildVisibleColumnSet(cols, visible, order, 100)
	assert.Len(t, vcs.Columns, 2)
	assert.Equal(t, "name", vcs.Keys[0])
	assert.Equal(t, "rarity", vcs.Keys[1])
}

func TestBuildVisibleColumnSet_OrderRespected(t *testing.T) {
	visible := map[string]bool{"name": true, "rarity": true, "number": true}
	order := []string{"rarity", "number", "name"}
	cols := []ColumnDef{
		{"name", "Name", 40, 8},
		{"rarity", "Rarity", 30, 5},
		{"number", "#", 30, 4},
	}
	vcs := BuildVisibleColumnSet(cols, visible, order, 100)
	assert.Equal(t, "rarity", vcs.Keys[0])
	assert.Equal(t, "number", vcs.Keys[1])
	assert.Equal(t, "name", vcs.Keys[2])
}

func TestVisibleColumnSet_BuildRow(t *testing.T) {
	vcs := VisibleColumnSet{
		Keys:   []string{"name", "quantity"},
		Widths: map[string]int{"name": 20, "quantity": 5},
	}
	data := map[string]string{"name": "Pikachu", "quantity": "3"}
	row := vcs.BuildRow(data)
	assert.Len(t, row, 2)
	assert.Equal(t, "Pikachu", row[0])
	assert.Equal(t, "3", row[1])
}

func TestVisibleColumnSet_BuildRow_Truncation(t *testing.T) {
	vcs := VisibleColumnSet{
		Keys:   []string{"name"},
		Widths: map[string]int{"name": 5},
	}
	data := map[string]string{"name": "Charizard"}
	row := vcs.BuildRow(data)
	assert.Equal(t, "Ch...", row[0])
}

func TestRenderProgressBar_Boundaries(t *testing.T) {
	result := RenderProgressBar(0, 10, nil)
	assert.Contains(t, result, "  0%")

	result = RenderProgressBar(100, 10, nil)
	assert.Contains(t, result, "100%")

	result = RenderProgressBar(50, 10, nil)
	assert.Contains(t, result, " 50%")
}

func TestRenderProgressBar_ClampNegative(t *testing.T) {
	result := RenderProgressBar(-10, 10, nil)
	assert.Contains(t, result, "  0%")
}

func TestRenderProgressBar_ClampOver100(t *testing.T) {
	result := RenderProgressBar(150, 10, nil)
	assert.Contains(t, result, "100%")
}

func TestRenderProgressBar_MinWidth(t *testing.T) {
	result := RenderProgressBar(50, 1, nil)
	assert.NotEmpty(t, result)
}

func TestCalcTableHeight_Normal(t *testing.T) {
	assert.Equal(t, 18, CalcTableHeight(20, 2, 10))
}

func TestCalcTableHeight_ZeroAvailable(t *testing.T) {
	assert.Equal(t, 10, CalcTableHeight(0, 2, 10))
}

func TestCalcTableHeight_SmallAvailable(t *testing.T) {
	assert.Equal(t, 10, CalcTableHeight(5, 2, 10))
}

func TestFilterCardsByQuerySubstring_EmptyQuery(t *testing.T) {
	cards := makeCards("Pikachu", "Charizard")
	result := filterCardsByQuerySubstring(cards, "")
	assert.Equal(t, cards, result)
}

func TestFilterCardsByQuerySubstring_MatchName(t *testing.T) {
	cards := makeCards("Pikachu", "Charizard", "Bulbasaur")
	result := filterCardsByQuerySubstring(cards, "pika")
	assert.Len(t, result, 1)
	assert.Equal(t, "Pikachu", result[0].Name)
}

func TestFilterCardsByQuerySubstring_CaseInsensitive(t *testing.T) {
	cards := makeCards("Pikachu")
	result := filterCardsByQuerySubstring(cards, "PIKACHU")
	assert.Len(t, result, 1)
}

func TestFilterCardsByQuerySubstring_MatchRarity(t *testing.T) {
	cards := []model.Card{
		{Name: "Pikachu", Rarity: "Common"},
		{Name: "Charizard", Rarity: "Rare Holo"},
	}
	result := filterCardsByQuerySubstring(cards, "rare")
	assert.Len(t, result, 1)
	assert.Equal(t, "Charizard", result[0].Name)
}

func TestFilterCardsByQuerySubstring_NoMatch(t *testing.T) {
	cards := makeCards("Pikachu", "Charizard")
	result := filterCardsByQuerySubstring(cards, "zzzzz")
	assert.Empty(t, result)
}

func TestFilterCardsByQuery_EmptyQuery(t *testing.T) {
	cards := makeCards("Pikachu", "Charizard")
	result := filterCardsByQuery(cards, "")
	assert.Equal(t, cards, result)
}

func TestFilterCardsByQuery_FindsCard(t *testing.T) {
	cards := makeCards("Pikachu", "Charizard", "Bulbasaur")
	result := filterCardsByQuery(cards, "Charizard")
	assert.NotEmpty(t, result)
	assert.Equal(t, "Charizard", result[0].Name)
}

func TestBuildCardExportRows_Basic(t *testing.T) {
	cards := []model.Card{
		{ID: 1, Name: "Pikachu", Number: "025", Rarity: "Common", Set: &model.Set{Name: "Base Set", Code: "BASE"}},
		{ID: 2, Name: "Charizard", Number: "004", Rarity: "Rare", Set: &model.Set{Name: "Base Set", Code: "BASE"}},
	}
	dbQtys := map[int64]int{1: 3, 2: 1}
	tempDeltas := map[int64]int{1: 1}
	rows := buildCardExportRows(cards, dbQtys, tempDeltas)
	assert.Len(t, rows, 2)
	assert.Equal(t, "Pikachu", rows[0].Name)
	assert.Equal(t, 4, rows[0].Quantity)
	assert.Equal(t, "BASE", rows[0].SetCode)
	assert.Equal(t, 1, rows[1].Quantity)
}

func TestBuildCardExportRows_ZeroQuantityOmitted(t *testing.T) {
	cards := []model.Card{
		{ID: 1, Name: "Pikachu", Set: &model.Set{Name: "Base", Code: "B"}},
		{ID: 2, Name: "Charizard", Set: &model.Set{Name: "Base", Code: "B"}},
	}
	dbQtys := map[int64]int{1: 0, 2: 1}
	rows := buildCardExportRows(cards, dbQtys, nil)
	assert.Len(t, rows, 1)
	assert.Equal(t, "Charizard", rows[0].Name)
}

func TestBuildCardExportRows_NilSet(t *testing.T) {
	cards := []model.Card{{ID: 1, Name: "Pikachu"}}
	dbQtys := map[int64]int{1: 2}
	rows := buildCardExportRows(cards, dbQtys, nil)
	assert.Len(t, rows, 1)
	assert.Equal(t, "", rows[0].SetName)
	assert.Equal(t, "", rows[0].SetCode)
}

func TestMatchActionOrDefault_NilConfig(t *testing.T) {
	assert.Equal(t, "fallback", MatchActionOrDefault(nil, "ctrl+c", "fallback"))
}

func TestGetAction_NilConfig(t *testing.T) {
	assert.Equal(t, "", GetAction(nil, "ctrl+c"))
}
