package export_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/export"
)

// ---------------------------------------------------------------------------
// FromCSV additional edge cases
// ---------------------------------------------------------------------------

func TestFromCSV_HeaderOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "header_only.csv")
	err := os.WriteFile(path, []byte("Name,Set,Set Code,Number,Rarity,Quantity\n"), 0644)
	require.NoError(t, err)

	rows, err := export.FromCSV(path)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestFromCSV_WrongColumnCount_TooMany(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "too_many.csv")
	content := "Name,Set,Set Code,Number,Rarity,Quantity,Extra\nCharizard,Base,BASE,4,Rare,1,extra\n"
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)

	_, err = export.FromCSV(path)
	assert.Error(t, err)
}

func TestFromCSV_InvalidQuantity_Float(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "float_qty.csv")
	content := "Name,Set,Set Code,Number,Rarity,Quantity\nCharizard,Base Set,BASE,4,Rare Holo,1.5\n"
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)

	_, err = export.FromCSV(path)
	assert.Error(t, err)
}

func TestFromCSV_SpecialCharactersInFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "special.csv")

	rows := []export.CardRow{
		{Name: `Card, "with" commas`, SetName: "Set\twith\ttabs", SetCode: "X1", Number: "1", Rarity: "Rare", Quantity: 1},
	}
	err := export.ToCSV(rows, path)
	require.NoError(t, err)

	loaded, err := export.FromCSV(path)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, rows[0].Name, loaded[0].Name)
}

func TestFromCSV_ZeroQuantity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zero_qty.csv")
	content := "Name,Set,Set Code,Number,Rarity,Quantity\nCharizard,Base Set,BASE,4,Rare Holo,0\n"
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)

	rows, err := export.FromCSV(path)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 0, rows[0].Quantity)
}

func TestFromCSV_HeaderCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "case.csv")
	content := "name,set,set code,number,rarity,quantity\nCharizard,Base Set,BASE,4,Rare Holo,2\n"
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)

	rows, err := export.FromCSV(path)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Charizard", rows[0].Name)
}

func TestFromCSV_WhitespaceAroundValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "whitespace.csv")
	content := "Name,Set,Set Code,Number,Rarity,Quantity\n  Charizard  , Base Set , BASE , 4 , Rare Holo , 3 \n"
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)

	rows, err := export.FromCSV(path)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Charizard", rows[0].Name)
	assert.Equal(t, 3, rows[0].Quantity)
}

func TestFromCSV_LargeDataset(t *testing.T) {
	const count = 5000
	dir := t.TempDir()
	path := filepath.Join(dir, "large.csv")

	rows := make([]export.CardRow, count)
	for i := range rows {
		rows[i] = export.CardRow{
			Name: "Card", SetName: "Set", SetCode: "S1",
			Number: "1", Rarity: "Common", Quantity: i + 1,
		}
	}
	err := export.ToCSV(rows, path)
	require.NoError(t, err)

	loaded, err := export.FromCSV(path)
	require.NoError(t, err)
	assert.Len(t, loaded, count)
}

// ---------------------------------------------------------------------------
// ToCSV edge cases
// ---------------------------------------------------------------------------

func TestToCSV_EmptyRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty_rows.csv")

	err := export.ToCSV(nil, path)
	require.NoError(t, err)

	// File should exist with just the header
	rows, err := export.FromCSV(path)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestToCSV_InvalidPath(t *testing.T) {
	err := export.ToCSV(nil, "/nonexistent/directory/file.csv")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// ToText edge cases
// ---------------------------------------------------------------------------

func TestToText_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "export.txt")

	rows := []export.CardRow{
		{Name: "Charizard", SetName: "Base Set", SetCode: "BASE", Number: "4", Rarity: "Rare Holo", Quantity: 2},
	}
	err := export.ToText(rows, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "Charizard")
	assert.Contains(t, content, "Base Set")
}

func TestToText_EmptyRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")

	err := export.ToText(nil, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Name")
}

// ---------------------------------------------------------------------------
// ToPTCGO edge cases
// ---------------------------------------------------------------------------

func TestToPTCGO_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deck.txt")

	rows := []export.CardRow{
		{Name: "Charizard", SetCode: "BASE", Number: "4", Rarity: "Rare Holo", Quantity: 2},
		{Name: "Fire Energy", SetCode: "BASE", Number: "98", Rarity: "", Quantity: 10},
	}
	err := export.ToPTCGO("My Deck", rows, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "My Deck")
	assert.Contains(t, content, "Charizard")
	assert.Contains(t, content, "Fire Energy")
}

func TestToPTCGO_EnergySeparation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "energy_split.txt")

	rows := []export.CardRow{
		{Name: "Pikachu", SetCode: "JU", Number: "60", Rarity: "Common", Quantity: 4},
		{Name: "Water Energy", SetCode: "BASE", Number: "99", Rarity: "", Quantity: 8},
	}
	err := export.ToPTCGO("Test", rows, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	pikachuIdx := strings.Index(content, "Pikachu")
	energyIdx := strings.Index(content, "Water Energy")
	assert.Less(t, pikachuIdx, energyIdx, "Pokemon should appear before Energy section")
}

// ---------------------------------------------------------------------------
// GenerateFilename
// ---------------------------------------------------------------------------

func TestGenerateFilename_ContainsComponents(t *testing.T) {
	name := export.GenerateFilename("collection", "My Deck Name", "csv")

	assert.Contains(t, name, "collection")
	assert.Contains(t, name, "my_deck_name")
	assert.Contains(t, name, ".csv")
}

func TestGenerateFilename_SpacesReplaced(t *testing.T) {
	name := export.GenerateFilename("deck", "Fire Deck Two", "txt")
	assert.NotContains(t, name, " ")
	assert.Contains(t, name, "fire_deck_two")
}

// ---------------------------------------------------------------------------
// ResolveCSVToCardQuantities additional cases
// ---------------------------------------------------------------------------

func TestResolveCSVToCardQuantities_SkipsMissingSetCode(t *testing.T) {
	rows := []export.CardRow{
		{Name: "No Set Code", SetCode: "", Number: "4", Quantity: 1},
		{Name: "No Number", SetCode: "BASE", Number: "", Quantity: 1},
	}

	lookup := func(setCode, number string) (int64, error) {
		return 100, nil // should never be called
	}

	result, quantities, err := export.ResolveCSVToCardQuantities(rows, lookup)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Imported)
	assert.Equal(t, 2, result.Skipped)
	assert.Empty(t, quantities)
}

func TestResolveCSVToCardQuantities_LookupError(t *testing.T) {
	rows := []export.CardRow{
		{Name: "Card", SetCode: "BASE", Number: "1", Quantity: 1},
	}

	lookup := func(setCode, number string) (int64, error) {
		return 0, assert.AnError
	}

	_, _, err := export.ResolveCSVToCardQuantities(rows, lookup)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lookup failed")
}
