package card

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/laiambryant/tui-cardman/internal/logging"
	"github.com/laiambryant/tui-cardman/internal/model"
)

// CardService defines the interface for card-related operations
type CardService interface {
	GetCardsByGameID(gameID int64) ([]model.Card, error)
	GetAllCards() ([]model.Card, error)
	GetCardIDByAPIID(ctx context.Context, apiID string) (int64, error)
	UpsertCard(ctx context.Context, tx *sql.Tx, apiID string, setID int64, number, name, rarity, artist string, cardGameID int64) (int64, error)
}

// CardServiceImpl implements the CardService interface
type CardServiceImpl struct {
	db *sql.DB
}

// NewCardService creates a new instance of CardServiceImpl
func NewCardService(db *sql.DB) CardService {
	return &CardServiceImpl{db: db}
}

const (
	selectCardsByGameIDQuery = `
		SELECT c.id, c.card_game_id, c.name, c.expansion, c.rarity, 
		       c.card_number, c.release_date, c.is_placeholder, c.created_at,
		       cg.id, cg.name, cg.created_at
		FROM cards c
		JOIN card_games cg ON c.card_game_id = cg.id
		WHERE c.card_game_id = ?
		ORDER BY c.name ASC
	`

	selectAllCardsQuery = `
		SELECT c.id, c.card_game_id, c.name, c.expansion, c.rarity, 
		       c.card_number, c.release_date, c.is_placeholder, c.created_at,
		       cg.id, cg.name, cg.created_at
		FROM cards c
		JOIN card_games cg ON c.card_game_id = cg.id
		ORDER BY cg.name ASC, c.name ASC
	`

	selectCardIDQuery = `SELECT id FROM cards WHERE api_id = ?`

	insertCardQuery = `INSERT INTO cards (api_id, set_id, number, name, rarity, artist, card_game_id, updated_at)
		    VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	updateCardQuery = `UPDATE cards 
		 SET set_id = ?, number = ?, name = ?, rarity = ?, 
			 artist = ?, updated_at = ?
		 WHERE id = ?`
)

// GetCardsByGameID retrieves all cards for a specific card game
func (s *CardServiceImpl) GetCardsByGameID(gameID int64) ([]model.Card, error) {
	slog.Debug("query", "query", logging.SanitizeQuery(selectCardsByGameIDQuery), "args", []any{gameID})
	rows, err := s.db.Query(selectCardsByGameIDQuery, gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cards by game ID: %w", err)
	}
	defer rows.Close()
	return s.scanCards(rows)
}

// GetAllCards retrieves all cards from the database
func (s *CardServiceImpl) GetAllCards() ([]model.Card, error) {
	slog.Debug("query", "query", logging.SanitizeQuery(selectAllCardsQuery))
	rows, err := s.db.Query(selectAllCardsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query all cards: %w", err)
	}
	defer rows.Close()

	return s.scanCards(rows)
}

// scanCards is a helper function to scan card rows
func (s *CardServiceImpl) scanCards(rows *sql.Rows) ([]model.Card, error) {
	var cards []model.Card
	for rows.Next() {
		var card model.Card
		var game model.CardGame
		var releaseDate, gameCreatedAt sql.NullTime
		err := rows.Scan(
			&card.ID, &card.CardGameID, &card.Name, &card.Expansion, &card.Rarity,
			&card.CardNumber, &releaseDate, &card.IsPlaceholder, &card.CreatedAt,
			&game.ID, &game.Name, &gameCreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan card: %w", err)
		}
		// Handle nullable dates
		if releaseDate.Valid {
			card.ReleaseDate = releaseDate.Time
		}
		if gameCreatedAt.Valid {
			game.CreatedAt = gameCreatedAt.Time
		}
		// Attach card game data
		card.CardGame = &game
		cards = append(cards, card)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cards: %w", err)
	}

	return cards, nil
}

// GetCardIDByAPIID retrieves the database ID for a card by its API ID
func (s *CardServiceImpl) GetCardIDByAPIID(ctx context.Context, apiID string) (int64, error) {
	var cardID int64
	err := s.db.QueryRowContext(ctx, selectCardIDQuery, apiID).Scan(&cardID)
	if err != nil {
		return 0, err
	}
	return cardID, nil
}

// UpsertCard inserts or updates a card within a transaction and returns its database ID
func (s *CardServiceImpl) UpsertCard(ctx context.Context, tx *sql.Tx, apiID string, setID int64, number, name, rarity, artist string, cardGameID int64) (int64, error) {
	var cardID int64
	err := tx.QueryRowContext(ctx, selectCardIDQuery, apiID).Scan(&cardID)
	if err == sql.ErrNoRows {
		result, err := tx.ExecContext(ctx, insertCardQuery, apiID, setID, number, name, rarity, artist, cardGameID, time.Now())
		if err != nil {
			return 0, fmt.Errorf("failed to insert card: %w", err)
		}
		cardID, err = result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get card ID: %w", err)
		}
		return cardID, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to query card: %w", err)
	}

	if _, err := tx.ExecContext(ctx, updateCardQuery, setID, number, name, rarity, artist, time.Now(), cardID); err != nil {
		return 0, fmt.Errorf("failed to update card: %w", err)
	}
	return cardID, nil
}
