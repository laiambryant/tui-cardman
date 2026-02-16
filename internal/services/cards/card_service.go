package card

import (
	"context"
	"database/sql"
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

	insertCardQuery = `INSERT INTO cards (api_id, set_id, number, name, rarity, artist, card_game_id, updated_at)
		    VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	updateCardQuery = `UPDATE cards 
		 SET set_id = ?, number = ?, name = ?, rarity = ?, 
			 artist = ?, updated_at = ?
		 WHERE id = ?`
)

// GetCardsByGameID retrieves all cards for a specific card game
func (s *CardServiceImpl) GetCardsByGameID(gameID int64) ([]model.Card, error) {
	rows, err := db.Query(s.db, selectCardsByGameIDQuery, gameID)
	if err != nil {
		return nil, &FailedToQueryCardsByGameIDError{Err: err}
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
		return nil, &FailedToQueryAllCardsError{Err: err}
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
		var set model.Set // New set struct

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
			return nil, &FailedToScanCardError{Err: err}
		}
		if gameCreatedAt.Valid {
			game.CreatedAt = gameCreatedAt.Time
		}
		// Handle nullable strings
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

		// Handle Set
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
			// Ensure SetID matches if it wasn't valid (though it should be if join worked)
			if card.SetID == 0 {
				card.SetID = set.ID
			}
		}

		// Attach card game data
		card.CardGame = &game
		cards = append(cards, card)
	}

	if err := rows.Err(); err != nil {
		return nil, &ErrorIteratingCardsError{Err: err}
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
			return 0, &FailedToInsertCardError{Err: err}
		}
		cardID, err = result.LastInsertId()
		if err != nil {
			slog.Error("failed to get last insert id for card", "api_id", apiID, "name", name, "error", err)
			return 0, &FailedToGetCardIDError{Err: err}
		}
		slog.Debug("inserted new card", "api_id", apiID, "name", name, "card_id", cardID)
		return cardID, nil
	}
	if err != nil {
		slog.Error("failed to query card during upsert", "api_id", apiID, "error", err)
		return 0, &FailedToQueryCardError{Err: err}
	}

	if _, err := db.ExecContextTx(ctx, tx, updateCardQuery, setID, number, name, rarity, artist, time.Now(), cardID); err != nil {
		slog.Error("failed to update card", "api_id", apiID, "name", name, "card_id", cardID, "error", err)
		return 0, &FailedToUpdateCardError{Err: err}
	}
	slog.Debug("updated existing card", "api_id", apiID, "name", name, "card_id", cardID)
	return cardID, nil
}
