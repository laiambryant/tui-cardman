package usercollection

import (
	"database/sql"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

// UserCollectionService defines the interface for user collection operations
type UserCollectionService interface {
	GetUserCollectionByUserID(userID int64) ([]model.UserCollection, error)
	GetUserCollectionByGameID(userID, gameID int64) ([]model.UserCollection, error)
	CreateSampleCollectionData(userID int64) error
}

// UserCollectionServiceImpl implements the UserCollectionService interface
type UserCollectionServiceImpl struct {
	db *sql.DB
}

// NewUserCollectionService creates a new instance of UserCollectionServiceImpl
func NewUserCollectionService(db *sql.DB) UserCollectionService {
	return &UserCollectionServiceImpl{db: db}
}

const (
	selectUserCollectionByUserIDQuery = `
		SELECT uc.id, uc.user_id, uc.card_id, uc.quantity, uc.condition,
		       uc.acquired_date, uc.notes, uc.created_at, uc.updated_at,
		       c.id, c.card_game_id, c.name, c.rarity, 
					  c.is_placeholder, c.created_at,
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
		       c.id, c.card_game_id, c.name, c.rarity, 
		       c.is_placeholder, c.created_at,
		       cg.id, cg.name, cg.created_at
		FROM user_collections uc
		JOIN cards c ON uc.card_id = c.id
		JOIN card_games cg ON c.card_game_id = cg.id
		WHERE uc.user_id = ? AND c.card_game_id = ?
		ORDER BY uc.created_at DESC
	`
)

// GetUserCollectionByUserID retrieves all collection entries for a specific user
func (s *UserCollectionServiceImpl) GetUserCollectionByUserID(userID int64) ([]model.UserCollection, error) {
	rows, err := db.Query(s.db, selectUserCollectionByUserIDQuery, userID)
	if err != nil {
		return nil, &FailedToQueryUserCollectionError{Err: err}
	}
	defer rows.Close()

	collections, err := s.scanUserCollections(rows)
	if err != nil {
		slog.Error("failed to scan user collections", "user_id", userID, "error", err)
		return nil, err
	}
	slog.Debug("retrieved user collection", "user_id", userID, "count", len(collections))
	return collections, nil
}

// GetUserCollectionByGameID retrieves collection entries for a specific user and card game
func (s *UserCollectionServiceImpl) GetUserCollectionByGameID(userID, gameID int64) ([]model.UserCollection, error) {
	rows, err := db.Query(s.db, selectUserCollectionByGameIDQuery, userID, gameID)
	if err != nil {
		return nil, &FailedToQueryUserCollectionByGameError{Err: err}
	}
	defer rows.Close()

	collections, err := s.scanUserCollections(rows)
	if err != nil {
		slog.Error("failed to scan user collections by game", "user_id", userID, "game_id", gameID, "error", err)
		return nil, err
	}
	slog.Debug("retrieved user collection by game", "user_id", userID, "game_id", gameID, "count", len(collections))
	return collections, nil
}

// scanUserCollections is a helper function to scan user collection rows
func (s *UserCollectionServiceImpl) scanUserCollections(rows *sql.Rows) ([]model.UserCollection, error) {
	var collections []model.UserCollection
	for rows.Next() {
		var collection model.UserCollection
		var card model.Card
		var game model.CardGame
		var acquiredDate, gameCreatedAt sql.NullTime
		err := rows.Scan(
			&collection.ID, &collection.UserID, &collection.CardID, &collection.Quantity, &collection.Condition,
			&acquiredDate, &collection.Notes, &collection.CreatedAt, &collection.UpdatedAt,
			&card.ID, &card.CardGameID, &card.Name, &card.Rarity,
			&card.IsPlaceholder, &card.CreatedAt,
			&game.ID, &game.Name, &gameCreatedAt,
		)
		if err != nil {
			return nil, &FailedToScanUserCollectionError{Err: err}
		}

		// Handle nullable dates
		if acquiredDate.Valid {
			collection.AcquiredDate = acquiredDate.Time
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
		return nil, &ErrorIteratingUserCollectionsError{Err: err}
	}

	return collections, nil
}

// CreateSampleCollectionData creates sample collection entries for a new local user
func (s *UserCollectionServiceImpl) CreateSampleCollectionData(userID int64) error {
	slog.Debug("creating sample collection data", "user_id", userID)
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
			slog.Error("failed to create sample collection data", "user_id", userID, "card_id", data.cardID, "error", err)
			return &FailedToCreateSampleCollectionDataError{Err: err}
		}
	}

	slog.Debug("created sample collection data", "user_id", userID, "sample_count", len(sampleData))
	return nil
}
