package gameimporter

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/testutil"
)

type fakeCard struct {
	id, number, name string
}

// fakeSource serves sets and cards from memory and stores cards with the real card service.
type fakeSource struct {
	sets      []GameSet
	cards     map[string][]fakeCard
	failStore string // card id whose StoreCard fails
}

func (f *fakeSource) source(database *sql.DB) Source[fakeCard] {
	cards := card.NewCardService(database)
	return Source[fakeCard]{
		FetchSets: func(context.Context) ([]GameSet, error) { return f.sets, nil },
		FetchCards: func(_ context.Context, set GameSet) ([]fakeCard, error) {
			return f.cards[set.APIID], nil
		},
		StoreCard: func(ctx context.Context, tx *sql.Tx, gameID, setID int64, c fakeCard) error {
			if c.id == f.failStore {
				return errors.New("injected store failure")
			}
			_, err := cards.UpsertCard(ctx, tx, c.id, setID, c.number, c.name, "Common", "", gameID)
			return err
		},
	}
}

func newTestImporter(t *testing.T, src *fakeSource) (*Importer, *sql.DB) {
	t.Helper()
	database := testutil.SetupTestDB(t)
	t.Cleanup(func() { testutil.CleanupTestDB(t, database) })
	testutil.ApplyTestMigrations(t, database, "../db/migrations")
	imp, err := New(database, "Pokemon", src.source(database))
	require.NoError(t, err)
	return imp, database
}

func twoSetSource() *fakeSource {
	return &fakeSource{
		sets: []GameSet{
			{APIID: "base1", Name: "Base Set", Code: "BS", PrintedTotal: 102, Total: 102},
			{APIID: "jungle", Name: "Jungle", Code: "JU"},
		},
		cards: map[string][]fakeCard{
			"base1":  {{"base1-1", "1", "Alakazam"}, {"base1-2", "2", "Blastoise"}},
			"jungle": {{"jungle-1", "1", "Clefable"}},
		},
	}
}

func lastRun(t *testing.T, database *sql.DB) (status, notes string, sets, cards, errs int) {
	t.Helper()
	require.NoError(t, database.QueryRow(
		`SELECT status, COALESCE(notes, ''), sets_processed, cards_imported, errors_count FROM import_runs ORDER BY id DESC LIMIT 1`,
	).Scan(&status, &notes, &sets, &cards, &errs))
	return status, notes, sets, cards, errs
}

func countRows(t *testing.T, database *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, database.QueryRow(query, args...).Scan(&n))
	return n
}

func TestNew_UnknownGame(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, database)
	testutil.ApplyTestMigrations(t, database, "../db/migrations")

	_, err := New(database, "Unknown Game", (&fakeSource{}).source(database))
	assert.Error(t, err)
}

func TestImportAll_ImportsEverySetAndRecordsRun(t *testing.T) {
	imp, database := newTestImporter(t, twoSetSource())

	require.NoError(t, imp.ImportAll(context.Background()))

	assert.Equal(t, 3, countRows(t, database, "SELECT COUNT(*) FROM cards WHERE card_game_id = ?", imp.GameID()))
	status, _, sets, cards, errs := lastRun(t, database)
	assert.Equal(t, "completed", status)
	assert.Equal(t, 2, sets)
	assert.Equal(t, 3, cards)
	assert.Zero(t, errs)

	counts, err := imp.GetImportedSetCounts(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 102, counts["base1"])
	assert.Equal(t, 1, counts["jungle"], "a set without an API total uses its card count")
}

func TestImportNew_SkipsImportedSets(t *testing.T) {
	src := twoSetSource()
	imp, database := newTestImporter(t, src)
	require.NoError(t, imp.ImportSpecific(context.Background(), []string{"base1"}))

	src.cards["base1"] = append(src.cards["base1"], fakeCard{"base1-3", "3", "Chansey"})
	require.NoError(t, imp.ImportNew(context.Background()))

	assert.Zero(t, countRows(t, database, "SELECT COUNT(*) FROM cards WHERE api_id = 'base1-3'"), "existing set must not be re-imported")
	assert.Equal(t, 1, countRows(t, database, "SELECT COUNT(*) FROM cards WHERE api_id = 'jungle-1'"))

	require.NoError(t, imp.ImportNew(context.Background()))
	status, notes, _, _, _ := lastRun(t, database)
	assert.Equal(t, "completed", status)
	assert.Equal(t, "No new sets found", notes)
}

func TestImportSpecific_ReportsUnknownSets(t *testing.T) {
	imp, database := newTestImporter(t, twoSetSource())

	require.NoError(t, imp.ImportSpecific(context.Background(), []string{"jungle", "fossil"}))
	status, notes, sets, _, _ := lastRun(t, database)
	assert.Equal(t, "completed", status)
	assert.Equal(t, 1, sets)
	assert.Contains(t, notes, "Not found: fossil")

	err := imp.ImportSpecific(context.Background(), []string{"fossil"})
	require.Error(t, err)
	status, _, _, _, _ = lastRun(t, database)
	assert.Equal(t, "failed", status)
}

func TestImportSet_RollsBackSetWhenCardFails(t *testing.T) {
	src := twoSetSource()
	src.failStore = "base1-2"
	imp, database := newTestImporter(t, src)

	require.NoError(t, imp.ImportAll(context.Background()))

	assert.Zero(t, countRows(t, database, "SELECT COUNT(*) FROM sets WHERE api_id = 'base1'"))
	assert.Zero(t, countRows(t, database, "SELECT COUNT(*) FROM cards WHERE api_id = 'base1-1'"))
	assert.Equal(t, 1, countRows(t, database, "SELECT COUNT(*) FROM sets WHERE api_id = 'jungle'"))
	status, notes, sets, _, errs := lastRun(t, database)
	assert.Equal(t, "completed_with_errors", status)
	assert.Equal(t, 1, sets)
	assert.Equal(t, 1, errs)
	assert.Contains(t, notes, "Set base1")
}

func TestDeleteSet_RemovesSetAndCards(t *testing.T) {
	imp, database := newTestImporter(t, twoSetSource())
	ctx := context.Background()
	require.NoError(t, imp.ImportAll(ctx))

	inDB, hasCollections, err := imp.CheckSetInDB(ctx, "base1")
	require.NoError(t, err)
	assert.True(t, inDB)
	assert.False(t, hasCollections)

	require.NoError(t, imp.DeleteSet(ctx, "base1"))
	assert.Zero(t, countRows(t, database, "SELECT COUNT(*) FROM cards WHERE api_id LIKE 'base1-%'"))
	ids, err := imp.GetImportedSetIDs(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"jungle": true}, ids)

	inDB, _, err = imp.CheckSetInDB(ctx, "base1")
	require.NoError(t, err)
	assert.False(t, inDB)
	assert.NoError(t, imp.DeleteSet(ctx, "base1"), "deleting a missing set is a no-op")
}

func TestCheckSetInDB_ReportsCollections(t *testing.T) {
	imp, database := newTestImporter(t, twoSetSource())
	ctx := context.Background()
	require.NoError(t, imp.ImportAll(ctx))
	_, err := database.Exec(`INSERT INTO users (email, password_hash) VALUES ('a@b.co', '')`)
	require.NoError(t, err)
	_, err = database.Exec(`INSERT INTO user_collections (user_id, card_id, quantity)
		SELECT 1, id, 1 FROM cards WHERE api_id = 'jungle-1'`)
	require.NoError(t, err)

	inDB, hasCollections, err := imp.CheckSetInDB(ctx, "jungle")
	require.NoError(t, err)
	assert.True(t, inDB)
	assert.True(t, hasCollections)
}

func TestFindSets(t *testing.T) {
	all := []GameSet{{APIID: "base1"}, {APIID: "base2"}, {APIID: "fossil"}}

	found, notFound := findSets(all, []string{"fossil", "jungle", "base1"})

	require.Len(t, found, 2)
	assert.Equal(t, "fossil", found[0].APIID)
	assert.Equal(t, "base1", found[1].APIID)
	assert.Equal(t, []string{"jungle"}, notFound)
}

func TestImportResultSummary(t *testing.T) {
	status, notes := importResult{setsProcessed: 5, cardsImported: 250}.summary("")
	assert.Equal(t, "completed", status)
	assert.Equal(t, "Imported 5 sets with 250 total cards", notes)

	status, notes = importResult{setsProcessed: 1, cardsImported: 2, errors: []string{"Set x: boom"}}.summary("Not found: y")
	assert.Equal(t, "completed_with_errors", status)
	assert.Equal(t, "Imported 1 sets with 2 total cards. Not found: y. Errors: Set x: boom", notes)
}
