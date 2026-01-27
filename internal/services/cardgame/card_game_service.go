package cardgame

import (
	"database/sql"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

// ICardGameService defines the interface for card game-related operations
type CardGameService interface {
	GetAllCardGames() ([]model.CardGame, error)
}

// CardGameServiceImpl implements the ICardGameService interface
type CardGameServiceImpl struct {
	db *sql.DB
}

// NewCardGameService creates a new instance of CardGameServiceImpl
func NewCardGameService(db *sql.DB) CardGameService {
	return &CardGameServiceImpl{db: db}
}

const (
	selectAllCardGamesQuery = `
		SELECT id, name, created_at
		FROM card_games
		ORDER BY name ASC
	`
)

// GetAllCardGames retrieves all card games from the database
func (s *CardGameServiceImpl) GetAllCardGames() ([]model.CardGame, error) {
	rows, err := db.Query(s.db, selectAllCardGamesQuery)
	if err != nil {
		return nil, &FailedToQueryCardGamesError{Err: err}
	}
	defer rows.Close()

	var games []model.CardGame
	for rows.Next() {
		var game model.CardGame
		if err := rows.Scan(&game.ID, &game.Name, &game.CreatedAt); err != nil {
			slog.Error("failed to scan card game", "error", err)
			return nil, &FailedToScanCardGameError{Err: err}
		}
		games = append(games, game)
	}

	if err = rows.Err(); err != nil {
		slog.Error("error iterating card games", "error", err)
		return nil, &ErrorIteratingCardGamesError{Err: err}
	}

	slog.Debug("retrieved all card games", "count", len(games))
	return games, nil
}
