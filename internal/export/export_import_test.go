package export_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/laiambryant/tui-cardman/internal/export"
)

// sampleRows returns a deterministic set of CardRows used across multiple tests.
func sampleRows() []export.CardRow {
	return []export.CardRow{
		{Name: "Charizard", SetName: "Base Set", SetCode: "BASE", Number: "4", Rarity: "Rare Holo", Quantity: 2},
		{Name: "Pikachu", SetName: "Jungle", SetCode: "JU", Number: "60", Rarity: "Common", Quantity: 4},
		{Name: "Blastoise", SetName: "Base Set", SetCode: "BASE", Number: "2", Rarity: "Rare Holo", Quantity: 1},
	}
}

// writeTempCSV exports rows to a temp directory and returns the file path.
func writeTempCSV(t *testing.T, rows []export.CardRow) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "export.csv")
	if err := export.ToCSV(rows, path); err != nil {
		t.Fatalf("ToCSV failed: %v", err)
	}
	return path
}

// ---------------------------------------------------------------------------
// Round-trip: ToCSV -> FromCSV
// ---------------------------------------------------------------------------

func TestRoundTrip_ToCSV_FromCSV(t *testing.T) {
	original := sampleRows()
	path := writeTempCSV(t, original)

	got, err := export.FromCSV(path)
	if err != nil {
		t.Fatalf("FromCSV returned error: %v", err)
	}

	if len(got) != len(original) {
		t.Fatalf("row count mismatch: got %d, want %d", len(got), len(original))
	}

	for i, want := range original {
		row := got[i]
		if row.Name != want.Name {
			t.Errorf("row %d Name: got %q, want %q", i, row.Name, want.Name)
		}
		if row.SetName != want.SetName {
			t.Errorf("row %d SetName: got %q, want %q", i, row.SetName, want.SetName)
		}
		if row.SetCode != want.SetCode {
			t.Errorf("row %d SetCode: got %q, want %q", i, row.SetCode, want.SetCode)
		}
		if row.Number != want.Number {
			t.Errorf("row %d Number: got %q, want %q", i, row.Number, want.Number)
		}
		if row.Rarity != want.Rarity {
			t.Errorf("row %d Rarity: got %q, want %q", i, row.Rarity, want.Rarity)
		}
		if row.Quantity != want.Quantity {
			t.Errorf("row %d Quantity: got %d, want %d", i, row.Quantity, want.Quantity)
		}
	}
}

// ---------------------------------------------------------------------------
// Round-trip with resolve: ToCSV -> FromCSV -> ResolveCSVToCardQuantities
// ---------------------------------------------------------------------------

func TestRoundTrip_WithResolve_AllMatched(t *testing.T) {
	original := sampleRows()
	path := writeTempCSV(t, original)

	rows, err := export.FromCSV(path)
	if err != nil {
		t.Fatalf("FromCSV: %v", err)
	}

	// Fake lookup: every card is found with a deterministic ID derived from
	// set code + number so we can verify the quantity map precisely.
	idMap := map[string]int64{
		"BASE/4": 101,
		"JU/60":  102,
		"BASE/2": 103,
	}
	lookup := func(setCode, number string) (int64, error) {
		return idMap[setCode+"/"+number], nil
	}

	result, quantities, err := export.ResolveCSVToCardQuantities(rows, lookup)
	if err != nil {
		t.Fatalf("ResolveCSVToCardQuantities: %v", err)
	}

	if result.Imported != len(original) {
		t.Errorf("Imported: got %d, want %d", result.Imported, len(original))
	}
	if result.Skipped != 0 {
		t.Errorf("Skipped: got %d, want 0", result.Skipped)
	}
	if result.TotalRows != len(original) {
		t.Errorf("TotalRows: got %d, want %d", result.TotalRows, len(original))
	}

	// Verify quantity map values.
	expected := map[int64]int{
		101: 2,
		102: 4,
		103: 1,
	}
	for id, wantQty := range expected {
		if got := quantities[id]; got != wantQty {
			t.Errorf("card id %d: quantity got %d, want %d", id, got, wantQty)
		}
	}
	if len(quantities) != len(expected) {
		t.Errorf("quantities map length: got %d, want %d", len(quantities), len(expected))
	}
}

// ---------------------------------------------------------------------------
// Partial match: some cards not in local DB -> skipped
// ---------------------------------------------------------------------------

func TestRoundTrip_WithResolve_PartialMatch(t *testing.T) {
	original := sampleRows()
	path := writeTempCSV(t, original)

	rows, err := export.FromCSV(path)
	if err != nil {
		t.Fatalf("FromCSV: %v", err)
	}

	// Only Charizard (BASE/4) is "known" locally.
	lookup := func(setCode, number string) (int64, error) {
		if setCode == "BASE" && number == "4" {
			return 101, nil
		}
		return 0, nil // not found
	}

	result, quantities, err := export.ResolveCSVToCardQuantities(rows, lookup)
	if err != nil {
		t.Fatalf("ResolveCSVToCardQuantities: %v", err)
	}

	if result.Imported != 1 {
		t.Errorf("Imported: got %d, want 1", result.Imported)
	}
	if result.Skipped != 2 {
		t.Errorf("Skipped: got %d, want 2", result.Skipped)
	}
	if result.TotalRows != 3 {
		t.Errorf("TotalRows: got %d, want 3", result.TotalRows)
	}
	if len(result.SkippedRows) != 2 {
		t.Errorf("SkippedRows length: got %d, want 2", len(result.SkippedRows))
	}
	if qty := quantities[101]; qty != 2 {
		t.Errorf("charizard quantity: got %d, want 2", qty)
	}
}

// ---------------------------------------------------------------------------
// Duplicate set-code+number rows: quantities should accumulate
// ---------------------------------------------------------------------------

func TestRoundTrip_WithResolve_DuplicateRowsAccumulate(t *testing.T) {
	rows := []export.CardRow{
		{Name: "Charizard", SetCode: "BASE", Number: "4", Quantity: 2},
		{Name: "Charizard", SetCode: "BASE", Number: "4", Quantity: 3},
	}
	path := writeTempCSV(t, rows)

	imported, err := export.FromCSV(path)
	if err != nil {
		t.Fatalf("FromCSV: %v", err)
	}

	lookup := func(setCode, number string) (int64, error) { return 101, nil }

	result, quantities, err := export.ResolveCSVToCardQuantities(imported, lookup)
	if err != nil {
		t.Fatalf("ResolveCSVToCardQuantities: %v", err)
	}

	if result.Imported != 2 {
		t.Errorf("Imported: got %d, want 2", result.Imported)
	}
	if quantities[101] != 5 {
		t.Errorf("accumulated quantity: got %d, want 5", quantities[101])
	}
}

// ---------------------------------------------------------------------------
// Empty row list: valid header-only CSV
// ---------------------------------------------------------------------------

func TestRoundTrip_EmptyRows(t *testing.T) {
	path := writeTempCSV(t, []export.CardRow{})

	rows, err := export.FromCSV(path)
	if err != nil {
		t.Fatalf("FromCSV on header-only file: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}

	result, quantities, err := export.ResolveCSVToCardQuantities(rows, func(_, _ string) (int64, error) { return 0, nil })
	if err != nil {
		t.Fatalf("ResolveCSVToCardQuantities on empty rows: %v", err)
	}
	if result.TotalRows != 0 {
		t.Errorf("TotalRows: got %d, want 0", result.TotalRows)
	}
	if len(quantities) != 0 {
		t.Errorf("quantities map should be empty")
	}
}

// ---------------------------------------------------------------------------
// Error cases: FromCSV rejects bad files
// ---------------------------------------------------------------------------

func TestFromCSV_FileNotFound(t *testing.T) {
	_, err := export.FromCSV(filepath.Join(t.TempDir(), "nonexistent.csv"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestFromCSV_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.csv")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := export.FromCSV(path)
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
}

func TestFromCSV_WrongHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad_header.csv")
	content := "WrongCol1,WrongCol2,WrongCol3,WrongCol4,WrongCol5,WrongCol6\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := export.FromCSV(path)
	if err == nil {
		t.Fatal("expected error for wrong header, got nil")
	}
}

func TestFromCSV_InvalidQuantity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad_qty.csv")
	content := "Name,Set,Set Code,Number,Rarity,Quantity\nCharizard,Base Set,BASE,4,Rare Holo,notanumber\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := export.FromCSV(path)
	if err == nil {
		t.Fatal("expected error for invalid quantity, got nil")
	}
}

func TestFromCSV_MissingColumns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "short_row.csv")
	// Header is correct but a data row only has 3 columns.
	content := "Name,Set,Set Code,Number,Rarity,Quantity\nCharizard,Base Set,BASE\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := export.FromCSV(path)
	if err == nil {
		t.Fatal("expected error for short data row, got nil")
	}
}

// ---------------------------------------------------------------------------
// Summary string
// ---------------------------------------------------------------------------

func TestImportResult_Summary_NoSkips(t *testing.T) {
	r := &export.ImportResult{Imported: 5, Skipped: 0, TotalRows: 5}
	got := r.Summary()
	want := "5 card(s) imported"
	if got != want {
		t.Errorf("Summary: got %q, want %q", got, want)
	}
}

func TestImportResult_Summary_WithSkips(t *testing.T) {
	r := &export.ImportResult{Imported: 3, Skipped: 2, TotalRows: 5}
	got := r.Summary()
	want := "3 card(s) imported, 2 skipped (not found in local database)"
	if got != want {
		t.Errorf("Summary: got %q, want %q", got, want)
	}
}
