package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type CardRow struct {
	Name     string
	SetName  string
	SetCode  string
	Number   string
	Rarity   string
	Quantity int
}

func ToCSV(rows []CardRow, filepath string) error {
	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{"Name", "Set", "Set Code", "Number", "Rarity", "Quantity"}); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	for _, r := range rows {
		if err := w.Write([]string{r.Name, r.SetName, r.SetCode, r.Number, r.Rarity, fmt.Sprintf("%d", r.Quantity)}); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}
	return nil
}

func ToText(rows []CardRow, filepath string) error {
	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()
	header := fmt.Sprintf("%-30s %-20s %-8s %-8s %-15s %s\n", "Name", "Set", "Code", "Number", "Rarity", "Qty")
	if _, err := f.WriteString(header); err != nil {
		return err
	}
	if _, err := f.WriteString(strings.Repeat("-", 90) + "\n"); err != nil {
		return err
	}
	for _, r := range rows {
		line := fmt.Sprintf("%-30s %-20s %-8s %-8s %-15s %d\n", r.Name, r.SetName, r.SetCode, r.Number, r.Rarity, r.Quantity)
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}

func ToPTCGO(deckName string, rows []CardRow, filepath string) error {
	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()
	pokemon := []CardRow{}
	trainers := []CardRow{}
	energy := []CardRow{}
	for _, r := range rows {
		lower := strings.ToLower(r.Name)
		if strings.Contains(lower, "energy") {
			energy = append(energy, r)
		} else if strings.Contains(lower, "supporter") || strings.Contains(lower, "stadium") || strings.Contains(lower, "item") || isTrainerRarity(r.Rarity) {
			trainers = append(trainers, r)
		} else {
			pokemon = append(pokemon, r)
		}
	}
	if _, err := f.WriteString("****** " + deckName + " ******\n\n"); err != nil {
		return err
	}
	if _, err := f.WriteString("##Pokémon - " + fmt.Sprintf("%d", sumQty(pokemon)) + "\n\n"); err != nil {
		return err
	}
	for _, r := range pokemon {
		if _, err := fmt.Fprintf(f, "* %d %s %s %s\n", r.Quantity, r.Name, r.SetCode, r.Number); err != nil {
			return err
		}
	}
	if _, err := f.WriteString("\n##Trainer Cards - " + fmt.Sprintf("%d", sumQty(trainers)) + "\n\n"); err != nil {
		return err
	}
	for _, r := range trainers {
		if _, err := fmt.Fprintf(f, "* %d %s %s %s\n", r.Quantity, r.Name, r.SetCode, r.Number); err != nil {
			return err
		}
	}
	if _, err := f.WriteString("\n##Energy - " + fmt.Sprintf("%d", sumQty(energy)) + "\n\n"); err != nil {
		return err
	}
	for _, r := range energy {
		if _, err := fmt.Fprintf(f, "* %d %s %s %s\n", r.Quantity, r.Name, r.SetCode, r.Number); err != nil {
			return err
		}
	}
	return nil
}

func isTrainerRarity(rarity string) bool {
	lower := strings.ToLower(rarity)
	return strings.Contains(lower, "trainer") || strings.Contains(lower, "uncommon") || lower == ""
}

func sumQty(rows []CardRow) int {
	total := 0
	for _, r := range rows {
		total += r.Quantity
	}
	return total
}

func GenerateFilename(exportType, name, format string) string {
	sanitized := strings.ReplaceAll(strings.ToLower(name), " ", "_")
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
