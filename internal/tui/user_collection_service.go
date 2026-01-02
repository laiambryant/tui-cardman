package tui

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/logging"
)

// IUserCollectionService defines the interface for user collection operations
type IUserCollectionService interface {
	GetUserCollectionByUserID(userID int64) ([]UserCollection, error)
	GetUserCollectionByGameID(userID, gameID int64) ([]UserCollection, error)
	CreateSampleCollectionData(userID int64) error
}

// UserCollectionServiceImpl implements the IUserCollectionService interface
type UserCollectionServiceImpl struct {
	db *sql.DB
}

// NewUserCollectionService creates a new instance of UserCollectionServiceImpl
func NewUserCollectionService(db *sql.DB) IUserCollectionService {
	return &UserCollectionServiceImpl{db: db}
}

const (
	selectUserCollectionByUserIDQuery = `
		SELECT uc.id, uc.user_id, uc.card_id, uc.quantity, uc.condition,
		       uc.acquired_date, uc.notes, uc.created_at, uc.updated_at,
		       c.id, c.card_game_id, c.name, c.expansion, c.rarity, 
		       c.card_number, c.release_date, c.is_placeholder, c.created_at,
		       cg.id, cg.name, cg.created_at
		FROM user_collections uc
		JOIN cards c ON uc.card_id = c.id
		JOIN card_games cg ON c.card_game_id = cg.id
		WHERE uc.user_id = ?
		ORDER BY uc.created_at DESC
	`

	selectUserCollectionByGameIDQuery = `
		SELECT uc.id, uc.user_id, uc.card_id, uc.quantity, uc.condition,
		       uc.acquired_date, uc.notes, uc.created_at, uc.updated_at,
		       c.id, c.card_game_id, c.name, c.expansion, c.rarity, 
		       c.card_number, c.release_date, c.is_placeholder, c.created_at,
		       cg.id, cg.name, cg.created_at
		FROM user_collections uc
		JOIN cards c ON uc.card_id = c.id
		JOIN card_games cg ON c.card_game_id = cg.id
		WHERE uc.user_id = ? AND c.card_game_id = ?
		ORDER BY uc.created_at DESC
	`
)

// GetUserCollectionByUserID retrieves all collection entries for a specific user
func (s *UserCollectionServiceImpl) GetUserCollectionByUserID(userID int64) ([]UserCollection, error) {
	slog.Debug("query", "query", logging.SanitizeQuery(selectUserCollectionByUserIDQuery), "args", []any{userID})
	rows, err := s.db.Query(selectUserCollectionByUserIDQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user collection: %w", err)
	}
	defer rows.Close()

	return s.scanUserCollections(rows)
}

// GetUserCollectionByGameID retrieves collection entries for a specific user and card game
func (s *UserCollectionServiceImpl) GetUserCollectionByGameID(userID, gameID int64) ([]UserCollection, error) {
	slog.Debug("query", "query", logging.SanitizeQuery(selectUserCollectionByGameIDQuery), "args", []any{userID, gameID})
	rows, err := s.db.Query(selectUserCollectionByGameIDQuery, userID, gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user collection by game: %w", err)
	}
	defer rows.Close()

	return s.scanUserCollections(rows)
}

// scanUserCollections is a helper function to scan user collection rows
func (s *UserCollectionServiceImpl) scanUserCollections(rows *sql.Rows) ([]UserCollection, error) {
	var collections []UserCollection
	for rows.Next() {
		var collection UserCollection
		var card Card
		var game CardGame
		var acquiredDate, releaseDate, gameCreatedAt sql.NullTime

		err := rows.Scan(
			&collection.ID, &collection.UserID, &collection.CardID, &collection.Quantity, &collection.Condition,
			&acquiredDate, &collection.Notes, &collection.CreatedAt, &collection.UpdatedAt,
			&card.ID, &card.CardGameID, &card.Name, &card.Expansion, &card.Rarity,
			&card.CardNumber, &releaseDate, &card.IsPlaceholder, &card.CreatedAt,
			&game.ID, &game.Name, &gameCreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user collection: %w", err)
		}

		// Handle nullable dates
		if acquiredDate.Valid {
			collection.AcquiredDate = acquiredDate.Time
		}
		if releaseDate.Valid {
			card.ReleaseDate = releaseDate.Time
		}
		if gameCreatedAt.Valid {
			game.CreatedAt = gameCreatedAt.Time
		}

		// Attach related data
		card.CardGame = &game
		collection.Card = &card

		collections = append(collections, collection)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user collections: %w", err)
	}

	return collections, nil
}

// CreateSampleCollectionData creates sample collection entries for a new local user
func (s *UserCollectionServiceImpl) CreateSampleCollectionData(userID int64) error {
	// Sample collection data - a few cards from each game
	sampleData := []struct {
		cardID    int64
		quantity  int
		condition string
		notes     string
	}{
		{1, 3, "Near Mint", "Starter deck pulls"},     // Pikachu
		{2, 1, "Mint", "Lucky booster pack"},          // Charizard
		{6, 2, "Near Mint", "Trade acquisition"},      // Black Lotus
		{7, 4, "Near Mint", "Commons from starter"},   // Lightning Bolt
		{11, 1, "Mint", "Graded card purchase"},       // Blue-Eyes White Dragon
		{12, 2, "Lightly Played", "Collection start"}, // Dark Magician
	}

	for _, data := range sampleData {
		query := `
			INSERT OR IGNORE INTO user_collections 
			(user_id, card_id, quantity, condition, acquired_date, notes)
			VALUES (?, ?, ?, ?, date('2024-01-15'), ?)
		`
		slog.Debug("exec", "query", query, "args", []any{userID, data.cardID, data.quantity, data.condition, data.notes})
		_, err := s.db.Exec(query, userID, data.cardID, data.quantity, data.condition, data.notes)
		if err != nil {
			return fmt.Errorf("failed to create sample collection data: %w", err)
		}
	}

	return nil
}
