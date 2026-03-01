package cardgame

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/testutil"
)

func setupCardGameTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := testutil.SetupTestDB(t)
	testutil.ApplyTestMigrations(t, db, "../../db/migrations")
	return db
}

func TestNewCardGameService(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewCardGameService(db)
	assert.NotNil(t, svc)
	assert.IsType(t, &CardGameServiceImpl{}, svc)
}

func TestGetAllCardGames_MigrationsPopulated(t *testing.T) {
	db := setupCardGameTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewCardGameService(db)

	games, err := svc.GetAllCardGames()
	require.NoError(t, err)

	// Migrations insert Pokemon, Magic: The Gathering, Yu-Gi-Oh!
	assert.GreaterOrEqual(t, len(games), 1)

	names := make([]string, len(games))
	for i, g := range games {
		names[i] = g.Name
	}
	assert.Contains(t, names, "Pokemon")
}

func TestGetAllCardGames_OrderedByName(t *testing.T) {
	db := setupCardGameTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewCardGameService(db)

	games, err := svc.GetAllCardGames()
	require.NoError(t, err)

	for i := 1; i < len(games); i++ {
		assert.LessOrEqual(t, games[i-1].Name, games[i].Name, "games should be ordered by name ASC")
	}
}

func TestGetAllCardGames_FieldsPopulated(t *testing.T) {
	db := setupCardGameTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	svc := NewCardGameService(db)

	games, err := svc.GetAllCardGames()
	require.NoError(t, err)

	for _, g := range games {
		assert.Positive(t, g.ID, "card game ID should be positive")
		assert.NotEmpty(t, g.Name, "card game name should not be empty")
		assert.False(t, g.CreatedAt.IsZero(), "created_at should be set")
	}
}

func TestGetAllCardGames_EmptyDatabase(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	// Minimal schema without seeded data
	_, err := db.Exec(`
		CREATE TABLE card_games (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL
		)
	`)
	require.NoError(t, err)

	svc := NewCardGameService(db)

	games, err := svc.GetAllCardGames()
	require.NoError(t, err)
	assert.Empty(t, games)
}

func TestGetAllCardGames_CustomGame(t *testing.T) {
	db := setupCardGameTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	_, err := db.Exec(`INSERT INTO card_games (name) VALUES ('Custom TCG')`)
	require.NoError(t, err)

	svc := NewCardGameService(db)

	games, err := svc.GetAllCardGames()
	require.NoError(t, err)

	names := make([]string, len(games))
	for i, g := range games {
		names[i] = g.Name
	}
	assert.Contains(t, names, "Custom TCG")
}
