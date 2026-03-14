// Package deck provides services for managing card deck data.
package deck

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

type DeckValidationError struct {
	Type    string
	Message string
}

type DeckService interface {
	CreateDeck(ctx context.Context, userID, cardGameID int64, name, format string) (*model.Deck, error)
	GetDecksByUserAndGame(userID, cardGameID int64) ([]model.Deck, error)
	GetDeckByID(deckID int64) (*model.Deck, error)
	UpdateDeck(ctx context.Context, deckID int64, name, format string) error
	DeleteDeck(ctx context.Context, deckID int64) error
	GetAllQuantitiesForDeck(deckID int64) (map[int64]int, error)
	UpsertDeckCardBatch(ctx context.Context, deckID int64, updates map[int64]int) error
	ValidateDeck(cards []model.Card, quantities map[int64]int, gameName string) []DeckValidationError
}

type DeckServiceImpl struct {
	db *sql.DB
}

func NewDeckService(database *sql.DB) DeckService {
	return &DeckServiceImpl{db: database}
}

const (
	insertDeckQuery = `
		INSERT INTO decks (user_id, card_game_id, name, format)
		VALUES (?, ?, ?, ?)
	`
	selectDecksByUserAndGameQuery = `
		SELECT id, user_id, card_game_id, name, format, created_at, updated_at
		FROM decks
		WHERE user_id = ? AND card_game_id = ?
		ORDER BY name ASC
	`
	selectDeckByIDQuery = `
		SELECT id, user_id, card_game_id, name, format, created_at, updated_at
		FROM decks
		WHERE id = ?
	`
	updateDeckQuery = `
		UPDATE decks SET name = ?, format = ?, updated_at = CURRENT_TIMESTAMP
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

func (s *DeckServiceImpl) CreateDeck(ctx context.Context, userID, cardGameID int64, name, format string) (*model.Deck, error) {
	result, err := db.ExecContext(ctx, s.db, insertDeckQuery, userID, cardGameID, name, format)
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
		Format:     format,
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
		if err := rows.Scan(&d.ID, &d.UserID, &d.CardGameID, &d.Name, &d.Format, &d.CreatedAt, &d.UpdatedAt); err != nil {
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
	err := db.QueryRow(s.db, selectDeckByIDQuery, deckID).Scan(&d.ID, &d.UserID, &d.CardGameID, &d.Name, &d.Format, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to query deck by id: %w", err)
	}
	return &d, nil
}

func (s *DeckServiceImpl) UpdateDeck(ctx context.Context, deckID int64, name, format string) error {
	_, err := db.ExecContext(ctx, s.db, updateDeckQuery, name, format, deckID)
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

func (s *DeckServiceImpl) ValidateDeck(cards []model.Card, quantities map[int64]int, gameName string) []DeckValidationError {
	if strings.EqualFold(gameName, "yu-gi-oh!") || strings.EqualFold(gameName, "yugioh") {
		return validateYGODeck(cards, quantities)
	}
	if strings.EqualFold(gameName, "magic: the gathering") {
		return validateMTGDeck(cards, quantities)
	}
	if strings.EqualFold(gameName, "one piece") {
		return validateOnePieceDeck(cards, quantities)
	}
	return validatePokemonDeck(cards, quantities)
}

func validatePokemonDeck(cards []model.Card, quantities map[int64]int) []DeckValidationError {
	var errors []DeckValidationError
	cardsByID := make(map[int64]model.Card)
	for _, c := range cards {
		cardsByID[c.ID] = c
	}
	totalCards := 0
	nameQty := make(map[string]int)
	for cardID, qty := range quantities {
		if qty <= 0 {
			continue
		}
		totalCards += qty
		if c, ok := cardsByID[cardID]; ok {
			nameQty[c.Name] += qty
		}
	}
	if totalCards != 60 {
		errors = append(errors, DeckValidationError{
			Type:    "card_count",
			Message: "Deck must have exactly 60 cards (currently " + strconv.Itoa(totalCards) + ")",
		})
	}
	for name, qty := range nameQty {
		if qty > 4 && !isBasicEnergy(name) {
			errors = append(errors, DeckValidationError{
				Type:    "duplicate_limit",
				Message: name + ": max 4 copies allowed (" + strconv.Itoa(qty) + " found)",
			})
		}
	}
	return errors
}

func validateYGODeck(cards []model.Card, quantities map[int64]int) []DeckValidationError {
	var errors []DeckValidationError
	cardsByID := make(map[int64]model.Card)
	for _, c := range cards {
		cardsByID[c.ID] = c
	}
	totalCards := 0
	nameQty := make(map[string]int)
	for cardID, qty := range quantities {
		if qty <= 0 {
			continue
		}
		totalCards += qty
		if c, ok := cardsByID[cardID]; ok {
			nameQty[c.Name] += qty
		}
	}
	if totalCards < 40 || totalCards > 60 {
		errors = append(errors, DeckValidationError{
			Type:    "card_count",
			Message: "Main deck must have 40-60 cards (currently " + strconv.Itoa(totalCards) + ")",
		})
	}
	for name, qty := range nameQty {
		if qty > 3 {
			errors = append(errors, DeckValidationError{
				Type:    "duplicate_limit",
				Message: name + ": max 3 copies allowed (" + strconv.Itoa(qty) + " found)",
			})
		}
	}
	return errors
}

var basicEnergyNames = map[string]bool{
	"grass energy":           true,
	"fire energy":            true,
	"water energy":           true,
	"lightning energy":       true,
	"psychic energy":         true,
	"fighting energy":        true,
	"darkness energy":        true,
	"metal energy":           true,
	"fairy energy":           true,
	"basic grass energy":     true,
	"basic fire energy":      true,
	"basic water energy":     true,
	"basic lightning energy": true,
	"basic psychic energy":   true,
	"basic fighting energy":  true,
	"basic darkness energy":  true,
	"basic metal energy":     true,
	"basic fairy energy":     true,
}

func isBasicEnergy(name string) bool {
	return basicEnergyNames[strings.ToLower(name)]
}

func validateMTGDeck(cards []model.Card, quantities map[int64]int) []DeckValidationError {
	var errors []DeckValidationError
	cardsByID := make(map[int64]model.Card)
	for _, c := range cards {
		cardsByID[c.ID] = c
	}
	totalCards := 0
	nameQty := make(map[string]int)
	for cardID, qty := range quantities {
		if qty <= 0 {
			continue
		}
		totalCards += qty
		if c, ok := cardsByID[cardID]; ok {
			nameQty[c.Name] += qty
		}
	}
	if totalCards < 60 {
		errors = append(errors, DeckValidationError{
			Type:    "card_count",
			Message: "Deck must have at least 60 cards (currently " + strconv.Itoa(totalCards) + ")",
		})
	}
	for name, qty := range nameQty {
		if qty > 4 && !isBasicLand(name) {
			errors = append(errors, DeckValidationError{
				Type:    "duplicate_limit",
				Message: name + ": max 4 copies allowed (" + strconv.Itoa(qty) + " found)",
			})
		}
	}
	return errors
}

var basicLandNames = map[string]bool{
	"plains":   true,
	"island":   true,
	"swamp":    true,
	"mountain": true,
	"forest":   true,
}

func isBasicLand(name string) bool {
	return basicLandNames[strings.ToLower(name)]
}

func validateOnePieceDeck(cards []model.Card, quantities map[int64]int) []DeckValidationError {
	var errors []DeckValidationError
	cardsByID := make(map[int64]model.Card)
	for _, c := range cards {
		cardsByID[c.ID] = c
	}
	totalCards := 0
	nameQty := make(map[string]int)
	for cardID, qty := range quantities {
		if qty <= 0 {
			continue
		}
		totalCards += qty
		if c, ok := cardsByID[cardID]; ok {
			nameQty[c.Name] += qty
		}
	}
	if totalCards != 51 {
		errors = append(errors, DeckValidationError{
			Type:    "card_count",
			Message: "Deck must have exactly 51 cards (1 Leader + 50 main deck, currently " + strconv.Itoa(totalCards) + ")",
		})
	}
	for name, qty := range nameQty {
		if qty > 4 {
			errors = append(errors, DeckValidationError{
				Type:    "duplicate_limit",
				Message: name + ": max 4 copies allowed (" + strconv.Itoa(qty) + " found)",
			})
		}
	}
	return errors
}
