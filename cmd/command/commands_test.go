package command

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbpkg "github.com/laiambryant/tui-cardman/internal/db"
)

func TestCanonicalGameName(t *testing.T) {
	for alias, want := range map[string]string{
		"pokemon":       "Pokemon",
		" Pokémon TCG ": "Pokemon",
		"MTG":           "Magic: The Gathering",
		"yu-gi-oh!":     "Yu-Gi-Oh!",
		"onepiece":      "One Piece",
		"Custom Game":   "Custom Game",
	} {
		assert.Equal(t, want, canonicalGameName(alias), alias)
	}
}

func TestParseQuantity(t *testing.T) {
	qty, err := parseQuantity("3")
	require.NoError(t, err)
	assert.Equal(t, 3, qty)
	for _, bad := range []string{"0", "-2", "two", ""} {
		_, err := parseQuantity(bad)
		assert.Error(t, err, bad)
	}
}

// setupCLIDB points the commands at a fresh database with one user and two cards.
func setupCLIDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cli.db")
	dsn := "file:" + path + "?_fk=1"
	t.Setenv("DB_DSN", dsn)
	database, err := dbpkg.OpenDB(dsn)
	require.NoError(t, err)
	defer database.Close()
	require.NoError(t, dbpkg.ApplyEmbeddedMigrations(database))
	for _, stmt := range []string{
		`INSERT INTO users (name, surname, email, password_hash) VALUES ('A', 'B', 'a@b.co', '')`,
		`INSERT INTO sets (card_game_id, api_id, code, name) VALUES (1, 'base1', 'base1', 'Base Set')`,
		`INSERT INTO cards (card_game_id, set_id, api_id, name, number, rarity) VALUES (1, 1, 'base1-1', 'Alakazam', '1', 'Rare Holo')`,
		`INSERT INTO cards (card_game_id, set_id, api_id, name, number, rarity) VALUES (1, 1, 'base1-2', 'Blastoise', '2', 'Rare Holo')`,
	} {
		_, err := database.Exec(stmt)
		require.NoError(t, err)
	}
	return dsn
}

func runCLI(t *testing.T, args ...string) error {
	t.Helper()
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func queryInt(t *testing.T, dsn, query string) int {
	t.Helper()
	database, err := dbpkg.OpenDB(dsn)
	require.NoError(t, err)
	defer database.Close()
	var n int
	require.NoError(t, database.QueryRow(query).Scan(&n))
	return n
}

func TestDeckCardCommands(t *testing.T) {
	dsn := setupCLIDB(t)

	require.NoError(t, runCLI(t, "decks", "create", "pokemon", "Water"))
	require.NoError(t, runCLI(t, "decks", "add-card", "1", "base1", "2", "2"))
	require.NoError(t, runCLI(t, "decks", "add-card", "1", "base1", "2", "1"))
	assert.Equal(t, 3, queryInt(t, dsn, `SELECT quantity FROM deck_cards WHERE deck_id = 1`), "add-card adds to the existing quantity")

	require.NoError(t, runCLI(t, "decks", "remove-card", "1", "base1", "2"))
	assert.Zero(t, queryInt(t, dsn, `SELECT COUNT(*) FROM deck_cards WHERE deck_id = 1`))

	assert.ErrorContains(t, runCLI(t, "decks", "show", "42"), "deck 42 not found")
	assert.Error(t, runCLI(t, "decks", "add-card", "1", "base1", "2", "0"))
	assert.Error(t, runCLI(t, "decks", "add-card", "1", "base1", "99", "1"), "unknown card")
}

func TestCollectionCommands(t *testing.T) {
	dsn := setupCLIDB(t)

	require.NoError(t, runCLI(t, "collection", "add", "pokemon", "base1", "1", "2"))
	require.NoError(t, runCLI(t, "collection", "add", "pokemon", "base1", "1", "1"))
	assert.Equal(t, 3, queryInt(t, dsn, `SELECT quantity FROM user_collections WHERE card_id = 1`))

	require.NoError(t, runCLI(t, "collection", "remove", "pokemon", "base1", "1", "5"))
	assert.Zero(t, queryInt(t, dsn, `SELECT COUNT(*) FROM user_collections`), "removing more than owned removes the card")

	assert.Error(t, runCLI(t, "collection", "add", "mtg", "base1", "1", "1"), "card belongs to another game")
	assert.Error(t, runCLI(t, "collection", "add", "pokemon", "base1", "1", "-1"))
}
