package deck

import (
	"context"
	"database/sql"
	"log/slog"
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
	ValidateDeck(cards []model.Card, quantities map[int64]int) []DeckValidationError
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
		return nil, &FailedToCreateDeckError{Err: err}
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, &FailedToCreateDeckError{Err: err}
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
		return nil, &FailedToQueryDecksError{Err: err}
	}
	defer rows.Close()
	var decks []model.Deck
	for rows.Next() {
		var d model.Deck
		if err := rows.Scan(&d.ID, &d.UserID, &d.CardGameID, &d.Name, &d.Format, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, &FailedToScanDeckError{Err: err}
		}
		decks = append(decks, d)
	}
	if err := rows.Err(); err != nil {
		return nil, &FailedToQueryDecksError{Err: err}
	}
	return decks, nil
}

func (s *DeckServiceImpl) GetDeckByID(deckID int64) (*model.Deck, error) {
	var d model.Deck
	err := db.QueryRow(s.db, selectDeckByIDQuery, deckID).Scan(&d.ID, &d.UserID, &d.CardGameID, &d.Name, &d.Format, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, &FailedToQueryDecksError{Err: err}
	}
	return &d, nil
}

func (s *DeckServiceImpl) UpdateDeck(ctx context.Context, deckID int64, name, format string) error {
	_, err := db.ExecContext(ctx, s.db, updateDeckQuery, name, format, deckID)
	if err != nil {
		slog.Error("failed to update deck", "deck_id", deckID, "error", err)
		return &FailedToUpdateDeckError{Err: err}
	}
	return nil
}

func (s *DeckServiceImpl) DeleteDeck(ctx context.Context, deckID int64) error {
	_, err := db.ExecContext(ctx, s.db, deleteDeckQuery, deckID)
	if err != nil {
		slog.Error("failed to delete deck", "deck_id", deckID, "error", err)
		return &FailedToDeleteDeckError{Err: err}
	}
	return nil
}

func (s *DeckServiceImpl) GetAllQuantitiesForDeck(deckID int64) (map[int64]int, error) {
	rows, err := db.Query(s.db, selectAllQuantitiesForDeckQuery, deckID)
	if err != nil {
		return nil, &FailedToGetDeckQuantitiesError{Err: err}
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
		return nil, &FailedToGetDeckQuantitiesError{Err: err}
	}
	return quantities, nil
}

func (s *DeckServiceImpl) UpsertDeckCardBatch(ctx context.Context, deckID int64, updates map[int64]int) error {
	return db.WithTransaction(ctx, s.db, func(tx *sql.Tx) error {
		for cardID, quantity := range updates {
			if quantity <= 0 {
				_, err := db.ExecContextTx(ctx, tx, deleteDeckCardQuery, deckID, cardID)
				if err != nil {
					return &FailedToUpsertDeckCardError{Err: err}
				}
				continue
			}
			_, err := db.ExecContextTx(ctx, tx, upsertDeckCardQuery, deckID, cardID, quantity)
			if err != nil {
				return &FailedToUpsertDeckCardError{Err: err}
			}
		}
		return nil
	})
}

func (s *DeckServiceImpl) ValidateDeck(cards []model.Card, quantities map[int64]int) []DeckValidationError {
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
			Message: strings.Replace("Deck must have exactly 60 cards (currently COUNT)", "COUNT", strings.Replace(strings.Replace("X", "X", string(rune('0'+totalCards/10)), 1), string(rune('0'+totalCards/10)), "", 0), 1),
		})
		errors[len(errors)-1].Message = "Deck must have exactly 60 cards (currently " + itoa(totalCards) + ")"
	}
	for name, qty := range nameQty {
		if qty > 4 && !isBasicEnergy(name) {
			errors = append(errors, DeckValidationError{
				Type:    "duplicate_limit",
				Message: name + ": max 4 copies allowed (" + itoa(qty) + " found)",
			})
		}
	}
	return errors
}

func isBasicEnergy(name string) bool {
	energyNames := []string{
		"Grass Energy", "Fire Energy", "Water Energy",
		"Lightning Energy", "Psychic Energy", "Fighting Energy",
		"Darkness Energy", "Metal Energy", "Fairy Energy",
		"Basic Grass Energy", "Basic Fire Energy", "Basic Water Energy",
		"Basic Lightning Energy", "Basic Psychic Energy", "Basic Fighting Energy",
		"Basic Darkness Energy", "Basic Metal Energy", "Basic Fairy Energy",
	}
	lower := strings.ToLower(name)
	for _, e := range energyNames {
		if strings.ToLower(e) == lower {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
