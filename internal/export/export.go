// Package export provides CSV export and import functionality for card collections.
package export

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type CardRow struct {
	Name     string
	SetName  string
	SetCode  string
	Number   string
	Rarity   string
	Quantity int
	// Category is the Pokemon TCG card category ("Pokemon", "Trainer" or
	// "Energy") when known. It is only used by the PTCGO format.
	Category string
}

func ToCSV(rows []CardRow, filepath string) (err error) {
	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close CSV file: %w", closeErr)
		}
	}()
	w := csv.NewWriter(f)
	if err := w.Write([]string{"Name", "Set", "Set Code", "Number", "Rarity", "Quantity"}); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	for _, r := range rows {
		if err := w.Write([]string{r.Name, r.SetName, r.SetCode, r.Number, r.Rarity, fmt.Sprintf("%d", r.Quantity)}); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("failed to flush CSV: %w", err)
	}
	return nil
}

func ToText(rows []CardRow, filepath string) error {
	return writeFile(filepath, func(w *bufio.Writer) {
		fmt.Fprintf(w, "%-30s %-20s %-8s %-8s %-15s %s\n", "Name", "Set", "Code", "Number", "Rarity", "Qty")
		w.WriteString(strings.Repeat("-", 90) + "\n")
		for _, r := range rows {
			fmt.Fprintf(w, "%-30s %-20s %-8s %-8s %-15s %d\n", r.Name, r.SetName, r.SetCode, r.Number, r.Rarity, r.Quantity)
		}
	})
}

// ToPTCGO writes a deck list in the Pokemon TCG Online text format, grouped
// into Pokémon, Trainer and Energy sections.
func ToPTCGO(deckName string, rows []CardRow, filepath string) error {
	sections := []struct {
		title string
		rows  []CardRow
	}{{title: "Pokémon"}, {title: "Trainer Cards"}, {title: "Energy"}}
	for _, r := range rows {
		i := ptcgoSection(r)
		sections[i].rows = append(sections[i].rows, r)
	}
	return writeFile(filepath, func(w *bufio.Writer) {
		fmt.Fprintf(w, "****** %s ******\n", deckName)
		for _, sec := range sections {
			fmt.Fprintf(w, "\n##%s - %d\n\n", sec.title, sumQty(sec.rows))
			for _, r := range sec.rows {
				fmt.Fprintf(w, "* %d %s %s %s\n", r.Quantity, r.Name, r.SetCode, r.Number)
			}
		}
	})
}

// ptcgoSection returns 0 for Pokémon, 1 for trainers and 2 for energy. The
// card category is used when known; otherwise the name is the only hint.
func ptcgoSection(r CardRow) int {
	switch strings.ToLower(r.Category) {
	case "pokemon":
		return 0
	case "trainer":
		return 1
	case "energy":
		return 2
	}
	lower := strings.ToLower(r.Name)
	switch {
	case strings.Contains(lower, "energy"):
		return 2
	case strings.Contains(lower, "supporter") || strings.Contains(lower, "stadium"):
		return 1
	}
	return 0
}

func sumQty(rows []CardRow) int {
	total := 0
	for _, r := range rows {
		total += r.Quantity
	}
	return total
}

// writeFile creates path, lets write fill it and reports write, flush and
// close errors.
func writeFile(path string, write func(w *bufio.Writer)) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close file: %w", closeErr)
		}
	}()
	w := bufio.NewWriter(f)
	write(w)
	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// GenerateFilename builds "<type>_<name>_<date>.<ext>". The name is lowercased
// and every run of characters other than letters and digits becomes a single
// "_", so names like "AC/DC" or "Magic: The Gathering" give valid file names.
func GenerateFilename(exportType, name, format string) string {
	sanitized := strings.Trim(strings.Join(strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), "_"), "_")
	if sanitized == "" {
		sanitized = "export"
	}
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s_%s_%s.%s", exportType, sanitized, date, format)
}

// FromCSV reads a CSV file exported by ToCSV and returns a slice of CardRow.
// The file must have a header row matching: Name, Set, Set Code, Number, Rarity, Quantity
func FromCSV(filepath string) ([]CardRow, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Validate header
	expected := []string{"Name", "Set", "Set Code", "Number", "Rarity", "Quantity"}
	header := records[0]
	if len(header) != len(expected) {
		return nil, fmt.Errorf("unexpected header: got %d columns, want %d", len(header), len(expected))
	}
	for i, col := range expected {
		if !strings.EqualFold(header[i], col) {
			return nil, fmt.Errorf("unexpected column %d: got %q, want %q", i+1, header[i], col)
		}
	}

	var rows []CardRow
	for lineNum, record := range records[1:] {
		if len(record) != 6 {
			return nil, fmt.Errorf("line %d: expected 6 columns, got %d", lineNum+2, len(record))
		}
		qty, err := strconv.Atoi(strings.TrimSpace(record[5]))
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid quantity %q: %w", lineNum+2, record[5], err)
		}
		rows = append(rows, CardRow{
			Name:     strings.TrimSpace(record[0]),
			SetName:  strings.TrimSpace(record[1]),
			SetCode:  strings.TrimSpace(record[2]),
			Number:   strings.TrimSpace(record[3]),
			Rarity:   strings.TrimSpace(record[4]),
			Quantity: qty,
		})
	}
	return rows, nil
}
