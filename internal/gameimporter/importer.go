// Package gameimporter implements the import workflow shared by every card game:
// tracking import runs, importing all/new/specific sets and deleting sets. Each
// game only supplies a Source describing how to download and store its data.
package gameimporter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/services/importruns"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

// GameSet is a game-agnostic representation of a card set.
type GameSet struct {
	APIID        string
	Name         string
	Code         string
	PrintedTotal int
	Total        int
}

// GameImporter is the import backend used by the TUI and the CLI.
type GameImporter interface {
	FetchSets(ctx context.Context) ([]GameSet, error)
	GetImportedSetIDs(ctx context.Context) (map[string]bool, error)
	GetImportedSetCounts(ctx context.Context) (map[string]int, error)
	CheckSetInDB(ctx context.Context, apiID string) (inDB, hasCollections bool, err error)
	ImportSet(ctx context.Context, apiID string) error
	DeleteSet(ctx context.Context, apiID string) error
	ImportAll(ctx context.Context) error
	ImportNew(ctx context.Context) error
	ImportSpecific(ctx context.Context, apiIDs []string) error
}

// Source describes how to download one game's data and store a card of type C.
type Source[C any] struct {
	// FetchSets lists the sets available from the game's API.
	FetchSets func(ctx context.Context) ([]GameSet, error)
	// FetchCards downloads every card of a set. It runs outside the database
	// transaction so slow API calls never hold the connection.
	FetchCards func(ctx context.Context, set GameSet) ([]C, error)
	// StoreCard upserts one card (and its game-specific data) inside tx.
	StoreCard func(ctx context.Context, tx *sql.Tx, gameID, setID int64, card C) error
}

// Importer implements GameImporter for a single card game.
type Importer struct {
	db        *sql.DB
	gameID    int64
	gameName  string
	runs      importruns.ImportRunService
	sets      sets.SetService
	fetchSets func(ctx context.Context) ([]GameSet, error)
	importSet func(ctx context.Context, set GameSet) (int, error)
}

// New creates an Importer for the card game stored as gameName in card_games.
func New[C any](database *sql.DB, gameName string, src Source[C]) (*Importer, error) {
	var gameID int64
	if err := database.QueryRow("SELECT id FROM card_games WHERE name = ?", gameName).Scan(&gameID); err != nil {
		return nil, fmt.Errorf("failed to get %s game ID: %w", gameName, err)
	}
	imp := &Importer{
		db:        database,
		gameID:    gameID,
		gameName:  gameName,
		runs:      importruns.NewImportRunService(database),
		sets:      sets.NewSetService(database),
		fetchSets: src.FetchSets,
	}
	imp.importSet = func(ctx context.Context, set GameSet) (int, error) {
		cards, err := src.FetchCards(ctx, set)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch cards for set %s: %w", set.APIID, err)
		}
		total := set.Total
		if total == 0 {
			total = len(cards)
		}
		printedTotal := set.PrintedTotal
		if printedTotal == 0 {
			printedTotal = total
		}
		err = db.WithTransaction(ctx, database, func(tx *sql.Tx) error {
			setID, err := imp.sets.UpsertSetTx(ctx, tx, gameID, set.APIID, set.Code, set.Name, printedTotal, total)
			if err != nil {
				return fmt.Errorf("failed to upsert set %s: %w", set.APIID, err)
			}
			for _, card := range cards {
				if err := src.StoreCard(ctx, tx, gameID, setID, card); err != nil {
					return fmt.Errorf("failed to store card in set %s: %w", set.APIID, err)
				}
			}
			return nil
		})
		if err != nil {
			return 0, err
		}
		return len(cards), nil
	}
	return imp, nil
}

// GameID returns the database ID of the importer's card game.
func (imp *Importer) GameID() int64 { return imp.gameID }

func (imp *Importer) FetchSets(ctx context.Context) ([]GameSet, error) {
	return imp.fetchSets(ctx)
}

func (imp *Importer) GetImportedSetIDs(ctx context.Context) (map[string]bool, error) {
	apiIDs, err := imp.sets.GetAllSetAPIIDs(ctx, imp.gameID)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(apiIDs))
	for _, id := range apiIDs {
		result[id] = true
	}
	return result, nil
}

func (imp *Importer) GetImportedSetCounts(ctx context.Context) (map[string]int, error) {
	return imp.sets.GetAllSetAPIIDsWithCounts(ctx, imp.gameID)
}

func (imp *Importer) CheckSetInDB(ctx context.Context, apiID string) (bool, bool, error) {
	setID, err := imp.sets.GetSetIDByAPIID(ctx, imp.gameID, apiID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("failed to look up set %s: %w", apiID, err)
	}
	hasCollections, err := imp.sets.SetHasUserCollections(ctx, setID)
	if err != nil {
		return true, false, fmt.Errorf("failed to check collections for set %s: %w", apiID, err)
	}
	return true, hasCollections, nil
}

func (imp *Importer) ImportSet(ctx context.Context, apiID string) error {
	return imp.ImportSpecific(ctx, []string{apiID})
}

// DeleteSet removes a set and its cards. Deleting a set that is not in the
// database is a no-op.
func (imp *Importer) DeleteSet(ctx context.Context, apiID string) error {
	setID, err := imp.sets.GetSetIDByAPIID(ctx, imp.gameID, apiID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to look up set %s: %w", apiID, err)
	}
	err = db.WithTransaction(ctx, imp.db, func(tx *sql.Tx) error {
		if _, err := db.ExecContextTx(ctx, tx, "DELETE FROM cards WHERE set_id = ?", setID); err != nil {
			return err
		}
		_, err := db.ExecContextTx(ctx, tx, "DELETE FROM sets WHERE id = ?", setID)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to delete %s set %s: %w", imp.gameName, apiID, err)
	}
	slog.Info("deleted set from database", "game", imp.gameName, "api_id", apiID, "db_id", setID)
	return nil
}

// ImportAll imports every set available from the API.
func (imp *Importer) ImportAll(ctx context.Context) error {
	return imp.run(ctx, "import-full", func(all []GameSet) ([]GameSet, string, error) {
		return all, "", nil
	})
}

// ImportNew imports only the sets that are not in the database yet.
func (imp *Importer) ImportNew(ctx context.Context) error {
	return imp.run(ctx, "import-updates", func(all []GameSet) ([]GameSet, string, error) {
		existing, err := imp.GetImportedSetIDs(ctx)
		if err != nil {
			return nil, "", fmt.Errorf("failed to query existing sets: %w", err)
		}
		var newSets []GameSet
		for _, set := range all {
			if !existing[set.APIID] {
				newSets = append(newSets, set)
			}
		}
		return newSets, "", nil
	})
}

// ImportSpecific imports the sets with the given API IDs. Unknown IDs are
// reported in the import run notes; it fails only if none of them exist.
func (imp *Importer) ImportSpecific(ctx context.Context, apiIDs []string) error {
	return imp.run(ctx, "import-specific", func(all []GameSet) ([]GameSet, string, error) {
		found, notFound := findSets(all, apiIDs)
		if len(found) == 0 {
			return nil, "", fmt.Errorf("none of the specified sets were found: %s", strings.Join(apiIDs, ", "))
		}
		if len(notFound) == 0 {
			return found, "", nil
		}
		slog.Warn("some sets were not found", "game", imp.gameName, "not_found", strings.Join(notFound, ", "))
		return found, "Not found: " + strings.Join(notFound, ", "), nil
	})
}

// run records an import run, selects the sets to import from the API listing
// and imports them one by one. A failing set is recorded and skipped.
func (imp *Importer) run(ctx context.Context, importType string, selectSets func(all []GameSet) ([]GameSet, string, error)) error {
	runID, err := imp.runs.CreateImportRun(ctx, importType)
	if err != nil {
		return fmt.Errorf("failed to create import run: %w", err)
	}
	fail := func(err error) error {
		_ = imp.runs.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, err.Error())
		return err
	}
	all, err := imp.fetchSets(ctx)
	if err != nil {
		return fail(fmt.Errorf("failed to fetch %s sets: %w", imp.gameName, err))
	}
	toImport, extraNote, err := selectSets(all)
	if err != nil {
		return fail(err)
	}
	if len(toImport) == 0 {
		return imp.runs.UpdateImportRun(ctx, runID, "completed", 0, 0, 0, "No new sets found")
	}
	slog.Info("starting import", "game", imp.gameName, "type", importType, "sets", len(toImport))
	result := imp.importSets(ctx, toImport)
	status, notes := result.summary(extraNote)
	if err := imp.runs.UpdateImportRun(ctx, runID, status, result.setsProcessed, result.cardsImported, len(result.errors), notes); err != nil {
		return fmt.Errorf("failed to update import run: %w", err)
	}
	slog.Info("import finished", "game", imp.gameName, "type", importType, "sets", result.setsProcessed, "cards", result.cardsImported, "errors", len(result.errors))
	return nil
}

type importResult struct {
	setsProcessed int
	cardsImported int
	errors        []string
}

func (imp *Importer) importSets(ctx context.Context, toImport []GameSet) importResult {
	var result importResult
	for _, set := range toImport {
		count, err := imp.importSet(ctx, set)
		if err != nil {
			slog.Error("failed to import set", "game", imp.gameName, "set", set.APIID, "error", err)
			result.errors = append(result.errors, fmt.Sprintf("Set %s: %v", set.APIID, err))
			continue
		}
		slog.Info("imported set", "game", imp.gameName, "set", set.APIID, "cards", count)
		result.setsProcessed++
		result.cardsImported += count
	}
	return result
}

// summary returns the import run status and notes.
func (r importResult) summary(extraNote string) (status, notes string) {
	status = "completed"
	notes = fmt.Sprintf("Imported %d sets with %d total cards", r.setsProcessed, r.cardsImported)
	if extraNote != "" {
		notes += ". " + extraNote
	}
	if len(r.errors) > 0 {
		status = "completed_with_errors"
		notes += ". Errors: " + strings.Join(r.errors, "; ")
	}
	return status, notes
}

// findSets returns the sets matching apiIDs (in request order) and the IDs that
// were not found.
func findSets(all []GameSet, apiIDs []string) (found []GameSet, notFound []string) {
	byID := make(map[string]GameSet, len(all))
	for _, set := range all {
		byID[set.APIID] = set
	}
	for _, id := range apiIDs {
		if set, ok := byID[id]; ok {
			found = append(found, set)
		} else {
			notFound = append(notFound, id)
		}
	}
	return found, notFound
}
