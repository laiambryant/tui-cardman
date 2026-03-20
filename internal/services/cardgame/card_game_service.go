// Package cardgame provides services for managing card game data.
package cardgame

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

var ErrGameNotFound = errors.New("game not found")

// CardGameService defines the interface for card game-related operations.
type CardGameService interface {
	GetAllCardGames() ([]model.CardGame, error)
	GetCardGameByName(name string) (*model.CardGame, error)
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
	selectCardGameByNameQuery = `
		SELECT id, name, created_at
		FROM card_games
		WHERE LOWER(name) = LOWER(?)
		LIMIT 1
	`
)

// GetAllCardGames retrieves all card games from the database
func (s *CardGameServiceImpl) GetAllCardGames() ([]model.CardGame, error) {
	rows, err := db.Query(s.db, selectAllCardGamesQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query card games: %w", err)
	}
	defer rows.Close()

	var games []model.CardGame
	for rows.Next() {
		var game model.CardGame
		if err := rows.Scan(&game.ID, &game.Name, &game.CreatedAt); err != nil {
			slog.Error("failed to scan card game", "error", err)
			return nil, fmt.Errorf("failed to scan card game: %w", err)
		}
		games = append(games, game)
	}

	if err = rows.Err(); err != nil {
		slog.Error("error iterating card games", "error", err)
		return nil, fmt.Errorf("error iterating card games: %w", err)
	}

	slog.Debug("retrieved all card games", "count", len(games))
	return games, nil
}

func (s *CardGameServiceImpl) GetCardGameByName(name string) (*model.CardGame, error) {
	row := db.QueryRow(s.db, selectCardGameByNameQuery, name)
	var game model.CardGame
	if err := row.Scan(&game.ID, &game.Name, &game.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGameNotFound
		}
		return nil, fmt.Errorf("failed to scan card game: %w", err)
	}
	return &game, nil
}
