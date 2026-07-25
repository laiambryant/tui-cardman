// Package deck provides services for managing card deck data.
package deck

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

type DeckService interface {
	CreateDeck(ctx context.Context, userID, cardGameID int64, name string) (*model.Deck, error)
	GetDecksByUserAndGame(userID, cardGameID int64) ([]model.Deck, error)
	GetDeckByID(deckID int64) (*model.Deck, error)
	UpdateDeck(ctx context.Context, deckID int64, name string) error
	DeleteDeck(ctx context.Context, deckID int64) error
	GetAllQuantitiesForDeck(deckID int64) (map[int64]int, error)
	UpsertDeckCardBatch(ctx context.Context, deckID int64, updates map[int64]int) error
}

type DeckServiceImpl struct {
	db *sql.DB
}

func NewDeckService(database *sql.DB) DeckService {
	return &DeckServiceImpl{db: database}
}

const (
	insertDeckQuery = `
		INSERT INTO decks (user_id, card_game_id, name)
		VALUES (?, ?, ?)
	`
	selectDecksByUserAndGameQuery = `
		SELECT id, user_id, card_game_id, name, created_at, updated_at
		FROM decks
		WHERE user_id = ? AND card_game_id = ?
		ORDER BY name ASC
	`
	selectDeckByIDQuery = `
		SELECT id, user_id, card_game_id, name, created_at, updated_at
		FROM decks
		WHERE id = ?
	`
	updateDeckQuery = `
		UPDATE decks SET name = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	deleteDeckQuery = `
		DELETE FROM decks WHERE id = ?
	`
	selectAllQuantitiesForDeckQuery = `
		SELECT card_id, quantity
		FROM deck_cards
		WHERE deck_id = ?
	`
	upsertDeckCardQuery = `
		INSERT INTO deck_cards (deck_id, card_id, quantity)
		VALUES (?, ?, ?)
		ON CONFLICT(deck_id, card_id)
		DO UPDATE SET quantity = excluded.quantity, updated_at = CURRENT_TIMESTAMP
	`
	deleteDeckCardQuery = `
		DELETE FROM deck_cards
		WHERE deck_id = ? AND card_id = ?
	`
)

func (s *DeckServiceImpl) CreateDeck(ctx context.Context, userID, cardGameID int64, name string) (*model.Deck, error) {
	result, err := db.ExecContext(ctx, s.db, insertDeckQuery, userID, cardGameID, name)
	if err != nil {
		slog.Error("failed to create deck", "user_id", userID, "name", name, "error", err)
		return nil, fmt.Errorf("failed to create deck: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id for deck: %w", err)
	}
	return &model.Deck{
		ID:         id,
		UserID:     userID,
		CardGameID: cardGameID,
		Name:       name,
	}, nil
}

func (s *DeckServiceImpl) GetDecksByUserAndGame(userID, cardGameID int64) ([]model.Deck, error) {
	rows, err := db.Query(s.db, selectDecksByUserAndGameQuery, userID, cardGameID)
	if err != nil {
		return nil, fmt.Errorf("failed to query decks: %w", err)
	}
	defer rows.Close()
	var decks []model.Deck
	for rows.Next() {
		var d model.Deck
		if err := rows.Scan(&d.ID, &d.UserID, &d.CardGameID, &d.Name, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan deck: %w", err)
		}
		decks = append(decks, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to query decks: %w", err)
	}
	return decks, nil
}

func (s *DeckServiceImpl) GetDeckByID(deckID int64) (*model.Deck, error) {
	var d model.Deck
	err := db.QueryRow(s.db, selectDeckByIDQuery, deckID).Scan(&d.ID, &d.UserID, &d.CardGameID, &d.Name, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to query deck by id: %w", err)
	}
	return &d, nil
}

func (s *DeckServiceImpl) UpdateDeck(ctx context.Context, deckID int64, name string) error {
	_, err := db.ExecContext(ctx, s.db, updateDeckQuery, name, deckID)
	if err != nil {
		slog.Error("failed to update deck", "deck_id", deckID, "error", err)
		return fmt.Errorf("failed to update deck: %w", err)
	}
	return nil
}

func (s *DeckServiceImpl) DeleteDeck(ctx context.Context, deckID int64) error {
	_, err := db.ExecContext(ctx, s.db, deleteDeckQuery, deckID)
	if err != nil {
		slog.Error("failed to delete deck", "deck_id", deckID, "error", err)
		return fmt.Errorf("failed to delete deck: %w", err)
	}
	return nil
}

func (s *DeckServiceImpl) GetAllQuantitiesForDeck(deckID int64) (map[int64]int, error) {
	rows, err := db.Query(s.db, selectAllQuantitiesForDeckQuery, deckID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deck quantities: %w", err)
	}
	defer rows.Close()
	quantities := make(map[int64]int)
	for rows.Next() {
		var cardID int64
		var quantity int
		if err := rows.Scan(&cardID, &quantity); err != nil {
			slog.Error("failed to scan deck quantity", "error", err)
			continue
		}
		quantities[cardID] = quantity
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to get deck quantities: %w", err)
	}
	return quantities, nil
}

func (s *DeckServiceImpl) UpsertDeckCardBatch(ctx context.Context, deckID int64, updates map[int64]int) error {
	return db.WithTransaction(ctx, s.db, func(tx *sql.Tx) error {
		for cardID, quantity := range updates {
			if quantity <= 0 {
				_, err := db.ExecContextTx(ctx, tx, deleteDeckCardQuery, deckID, cardID)
				if err != nil {
					return fmt.Errorf("failed to upsert deck card: %w", err)
				}
				continue
			}
			_, err := db.ExecContextTx(ctx, tx, upsertDeckCardQuery, deckID, cardID, quantity)
			if err != nil {
				return fmt.Errorf("failed to upsert deck card: %w", err)
			}
		}
		return nil
	})
}
