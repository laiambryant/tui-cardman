package tui

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/logging"
)

// ICardGameService defines the interface for card game-related operations
type ICardGameService interface {
	GetAllCardGames() ([]CardGame, error)
}

// CardGameServiceImpl implements the ICardGameService interface
type CardGameServiceImpl struct {
	db *sql.DB
}

// NewCardGameService creates a new instance of CardGameServiceImpl
func NewCardGameService(db *sql.DB) ICardGameService {
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
func (s *CardGameServiceImpl) GetAllCardGames() ([]CardGame, error) {
	slog.Debug("query", "query", logging.SanitizeQuery(selectAllCardGamesQuery))
	rows, err := s.db.Query(selectAllCardGamesQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query card games: %w", err)
	}
	defer rows.Close()

	var games []CardGame
	for rows.Next() {
		var game CardGame
		if err := rows.Scan(&game.ID, &game.Name, &game.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan card game: %w", err)
		}
		games = append(games, game)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating card games: %w", err)
	}

	return games, nil
}
