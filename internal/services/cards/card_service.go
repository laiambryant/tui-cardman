// Package card provides services for managing individual card data.
package card

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

// CardService defines the interface for card-related operations
type CardService interface {
	GetCardsByGameID(gameID int64) ([]model.Card, error)
	GetAllCards() ([]model.Card, error)
	GetCardIDByAPIID(ctx context.Context, apiID string) (int64, error)
	UpsertCard(ctx context.Context, tx *sql.Tx, apiID string, setID int64, number, name, rarity, artist string, cardGameID int64) (int64, error)
	GetCardBySetCodeAndNumber(setCode, number string) (*model.Card, error)
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
		SELECT c.id, c.card_game_id, c.name, c.rarity, c.is_placeholder, c.created_at,
		       c.api_id, c.set_id, c.number, c.artist, c.updated_at,
		       cg.id, cg.name, cg.created_at,
		       s.id, s.name, s.code, s.printed_total, s.total
		FROM cards c
		JOIN card_games cg ON c.card_game_id = cg.id
		LEFT JOIN sets s ON c.set_id = s.id
		WHERE c.card_game_id = ?
		ORDER BY c.name ASC
	`

	selectAllCardsQuery = `
		SELECT c.id, c.card_game_id, c.name, c.rarity, 
			c.is_placeholder, c.created_at,
			c.api_id, c.set_id, c.number, c.artist, c.updated_at,
			cg.id, cg.name, cg.created_at,
			s.id, s.name, s.code, s.printed_total, s.total
		FROM cards c
		JOIN card_games cg ON c.card_game_id = cg.id
		LEFT JOIN sets s ON c.set_id = s.id
		ORDER BY cg.name ASC, c.name ASC
	`

	selectCardIDQuery = `SELECT id FROM cards WHERE api_id = ?`

	selectCardBySetCodeAndNumberQuery = `
		SELECT c.id, c.card_game_id, c.name, c.rarity, c.is_placeholder, c.created_at,
		       c.api_id, c.set_id, c.number, c.artist, c.updated_at,
		       cg.id, cg.name, cg.created_at,
		       s.id, s.name, s.code, s.printed_total, s.total
		FROM cards c
		JOIN card_games cg ON c.card_game_id = cg.id
		JOIN sets s ON c.set_id = s.id
		WHERE s.code = ? AND c.number = ?
		LIMIT 1
	`

	insertCardQuery = `INSERT INTO cards (api_id, set_id, number, name, rarity, artist, card_game_id, updated_at)
		    VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	updateCardQuery = `UPDATE cards
		 SET set_id = ?, number = ?, name = ?, rarity = ?,
			 artist = ?, card_game_id = ?, updated_at = ?
		 WHERE id = ?`
)

// GetCardsByGameID retrieves all cards for a specific card game
func (s *CardServiceImpl) GetCardsByGameID(gameID int64) ([]model.Card, error) {
	rows, err := db.Query(s.db, selectCardsByGameIDQuery, gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cards by game ID: %w", err)
	}
	defer rows.Close()
	cards, err := s.scanCards(rows)
	if err != nil {
		slog.Error("failed to scan cards", "game_id", gameID, "error", err)
		return nil, err
	}
	slog.Debug("retrieved cards by game ID", "game_id", gameID, "count", len(cards))
	return cards, nil
}

// GetAllCards retrieves all cards from the database
func (s *CardServiceImpl) GetAllCards() ([]model.Card, error) {
	rows, err := db.Query(s.db, selectAllCardsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query all cards: %w", err)
	}
	defer rows.Close()

	cards, err := s.scanCards(rows)
	if err != nil {
		slog.Error("failed to scan all cards", "error", err)
		return nil, err
	}
	slog.Debug("retrieved all cards", "count", len(cards))
	return cards, nil
}

// scanCards is a helper function to scan card rows
func (s *CardServiceImpl) scanCards(rows *sql.Rows) ([]model.Card, error) {
	var cards []model.Card
	for rows.Next() {
		var card model.Card
		var game model.CardGame
		var set model.Set

		var gameCreatedAt, updatedAt sql.NullTime
		var apiID, number, artist sql.NullString
		var setID sql.NullInt64

		// Set fields
		var sID sql.NullInt64
		var sName, sCode sql.NullString
		var sPrintedTotal, sTotal sql.NullInt64

		err := rows.Scan(
			&card.ID, &card.CardGameID, &card.Name, &card.Rarity,
			&card.IsPlaceholder, &card.CreatedAt,
			&apiID, &setID, &number, &artist, &updatedAt,
			&game.ID, &game.Name, &gameCreatedAt,
			&sID, &sName, &sCode, &sPrintedTotal, &sTotal,
		)
		if err != nil {
			slog.Error("failed to scan card", "error", err)
			return nil, fmt.Errorf("failed to scan card: %w", err)
		}
		if gameCreatedAt.Valid {
			game.CreatedAt = gameCreatedAt.Time
		}
		if apiID.Valid {
			card.APIID = apiID.String
		}
		if number.Valid {
			card.Number = number.String
		}
		if artist.Valid {
			card.Artist = artist.String
		}
		if setID.Valid {
			card.SetID = setID.Int64
		}

		if sID.Valid {
			set.ID = sID.Int64
			set.Name = sName.String
			set.Code = sCode.String
			if sPrintedTotal.Valid {
				set.PrintedTotal = int(sPrintedTotal.Int64)
			}
			if sTotal.Valid {
				set.Total = int(sTotal.Int64)
			}
			card.Set = &set
			if card.SetID == 0 {
				card.SetID = set.ID
			}
		}

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
	err := db.QueryRowContext(ctx, s.db, selectCardIDQuery, apiID).Scan(&cardID)
	if err != nil {
		if err == sql.ErrNoRows {
			slog.Debug("card not found by API ID", "api_id", apiID)
		} else {
			slog.Error("failed to query card ID by API ID", "api_id", apiID, "error", err)
		}
		return 0, err
	}
	slog.Debug("found card ID by API ID", "api_id", apiID, "card_id", cardID)
	return cardID, nil
}

// UpsertCard inserts or updates a card within a transaction and returns its database ID
func (s *CardServiceImpl) UpsertCard(ctx context.Context, tx *sql.Tx, apiID string, setID int64, number, name, rarity, artist string, cardGameID int64) (int64, error) {
	var cardID int64
	err := db.QueryRowContextTx(ctx, tx, selectCardIDQuery, apiID).Scan(&cardID)
	if err == sql.ErrNoRows {
		result, err := db.ExecContextTx(ctx, tx, insertCardQuery, apiID, setID, number, name, rarity, artist, cardGameID, time.Now())
		if err != nil {
			slog.Error("failed to insert card", "api_id", apiID, "name", name, "error", err)
			return 0, fmt.Errorf("failed to insert card: %w", err)
		}
		cardID, err = result.LastInsertId()
		if err != nil {
			slog.Error("failed to get last insert id for card", "api_id", apiID, "name", name, "error", err)
			return 0, fmt.Errorf("failed to get card ID: %w", err)
		}
		slog.Debug("inserted new card", "api_id", apiID, "name", name, "card_id", cardID)
		return cardID, nil
	}
	if err != nil {
		slog.Error("failed to query card during upsert", "api_id", apiID, "error", err)
		return 0, fmt.Errorf("failed to query card: %w", err)
	}

	if _, err := db.ExecContextTx(ctx, tx, updateCardQuery, setID, number, name, rarity, artist, cardGameID, time.Now(), cardID); err != nil {
		slog.Error("failed to update card", "api_id", apiID, "name", name, "card_id", cardID, "error", err)
		return 0, fmt.Errorf("failed to update card: %w", err)
	}
	slog.Debug("updated existing card", "api_id", apiID, "name", name, "card_id", cardID)
	return cardID, nil
}

// GetCardBySetCodeAndNumber retrieves a card by its set code and card number
func (s *CardServiceImpl) GetCardBySetCodeAndNumber(setCode, number string) (*model.Card, error) {
	rows, err := db.Query(s.db, selectCardBySetCodeAndNumberQuery, setCode, number)
	if err != nil {
		return nil, fmt.Errorf("failed to query card: %w", err)
	}
	defer rows.Close()
	cards, err := s.scanCards(rows)
	if err != nil {
		return nil, err
	}
	if len(cards) == 0 {
		return nil, nil
	}
	return &cards[0], nil
}
