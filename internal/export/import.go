package export

import "fmt"

// ImportResult summarizes the outcome of resolving CSV rows against local cards.
type ImportResult struct {
	// Imported is the number of CSV rows that were matched to a local card.
	Imported int
	// Skipped is the number of CSV rows that could not be matched.
	Skipped int
	// SkippedRows holds the unmatched rows for display to the user.
	SkippedRows []CardRow
	// TotalRows is the total number of data rows in the CSV (excluding header).
	TotalRows int
}

// Summary returns a human-readable one-line description of the result.
func (r *ImportResult) Summary() string {
	if r.Skipped == 0 {
		return fmt.Sprintf("%d card(s) imported", r.Imported)
	}
	return fmt.Sprintf("%d card(s) imported, %d skipped (not found in local database)", r.Imported, r.Skipped)
}

// CardLookupFn is a function that resolves a set code + number to a card ID.
// It should return (0, nil) when the card is not found.
type CardLookupFn func(setCode, number string) (int64, error)

// ResolveCSVToCardQuantities matches rows from a CSV import against local cards
// and returns a quantity map suitable for UpsertDeckCardBatch / UpsertListCardBatch.
//
// Quantities in the returned map represent the amounts from the CSV only; the
// caller is responsible for merging them with any existing quantities.
func ResolveCSVToCardQuantities(rows []CardRow, lookup CardLookupFn) (*ImportResult, map[int64]int, error) {
	result := &ImportResult{TotalRows: len(rows)}
	quantities := make(map[int64]int)

	for _, row := range rows {
		if row.SetCode == "" || row.Number == "" {
			result.Skipped++
			result.SkippedRows = append(result.SkippedRows, row)
			continue
		}

		cardID, err := lookup(row.SetCode, row.Number)
		if err != nil {
			return nil, nil, fmt.Errorf("lookup failed for %q %s/%s: %w", row.Name, row.SetCode, row.Number, err)
		}
		if cardID == 0 {
			result.Skipped++
			result.SkippedRows = append(result.SkippedRows, row)
			continue
		}

		quantities[cardID] += row.Quantity
		result.Imported++
	}

	return result, quantities, nil
}
