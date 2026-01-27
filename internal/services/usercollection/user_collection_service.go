package usercollection

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/logging"
	"github.com/laiambryant/tui-cardman/internal/model"
)

// UserCollectionService defines the interface for user collection operations
type UserCollectionService interface {
	GetUserCollectionByUserID(userID int64) ([]model.UserCollection, error)
	GetUserCollectionByGameID(userID, gameID int64) ([]model.UserCollection, error)
	CreateSampleCollectionData(userID int64) error
	GetCardQuantity(userID, cardID int64) (int, error)
	IncrementQuantity(ctx context.Context, userID, cardID int64) error
	DecrementQuantity(ctx context.Context, userID, cardID int64) error
	UpsertCollectionBatch(ctx context.Context, userID int64, updates map[int64]int) error
	GetAllQuantitiesForGame(userID, gameID int64) (map[int64]int, error)
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
		       cg.id, cg.name, cg.created_at,
		       s.id, s.name, s.code
		FROM user_collections uc
		JOIN cards c ON uc.card_id = c.id
		JOIN card_games cg ON c.card_game_id = cg.id
		LEFT JOIN sets s ON c.set_id = s.id
		WHERE uc.user_id = ?
		ORDER BY uc.created_at DESC
	`

	selectUserCollectionByGameIDQuery = `
		SELECT uc.id, uc.user_id, uc.card_id, uc.quantity, uc.condition,
		       uc.acquired_date, uc.notes, uc.created_at, uc.updated_at,
		       c.id, c.card_game_id, c.name, c.rarity, 
		       c.is_placeholder, c.created_at,
		       cg.id, cg.name, cg.created_at,
		       s.id, s.name, s.code
		FROM user_collections uc
		JOIN cards c ON uc.card_id = c.id
		JOIN card_games cg ON c.card_game_id = cg.id
		LEFT JOIN sets s ON c.set_id = s.id
		WHERE uc.user_id = ? AND c.card_game_id = ?
		ORDER BY uc.created_at DESC
	`

	selectCardQuantityQuery = `
		SELECT uc.quantity 
		FROM user_collections uc 
		WHERE uc.user_id = ? AND uc.card_id = ?
	`

	selectAllQuantitiesForGameQuery = `
		SELECT uc.card_id, uc.quantity
		FROM user_collections uc
		JOIN cards c ON uc.card_id = c.id
		WHERE uc.user_id = ? AND c.card_game_id = ?
	`

	upsertCollectionQuery = `
		INSERT INTO user_collections (user_id, card_id, quantity, condition)
		VALUES (?, ?, ?, 'Near Mint')
		ON CONFLICT(user_id, card_id) 
		DO UPDATE SET quantity = excluded.quantity, updated_at = CURRENT_TIMESTAMP
	`

	deleteCollectionQuery = `
		DELETE FROM user_collections 
		WHERE user_id = ? AND card_id = ?
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
		var set model.Set // New set struct

		var acquiredDate, gameCreatedAt sql.NullTime
		var notes sql.NullString
		var setID sql.NullInt64 // New nullable set fields
		var setName sql.NullString
		var setCode sql.NullString

		err := rows.Scan(
			&collection.ID, &collection.UserID, &collection.CardID, &collection.Quantity, &collection.Condition,
			&acquiredDate, &notes, &collection.CreatedAt, &collection.UpdatedAt,
			&card.ID, &card.CardGameID, &card.Name, &card.Rarity,
			&card.IsPlaceholder, &card.CreatedAt,
			&game.ID, &game.Name, &gameCreatedAt,
			&setID, &setName, &setCode, // Scan new fields
		)
		if err != nil {
			return nil, &FailedToScanUserCollectionError{Err: err}
		}

		// Handle nullable dates
		if acquiredDate.Valid {
			collection.AcquiredDate = acquiredDate.Time
		}
		if notes.Valid {
			collection.Notes = notes.String
		}
		if gameCreatedAt.Valid {
			game.CreatedAt = gameCreatedAt.Time
		}

		// Handle set
		if setID.Valid {
			set.ID = setID.Int64
			set.Name = setName.String
			set.Code = setCode.String
			card.Set = &set
			card.SetID = set.ID
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
		slog.Debug("exec", "query", logging.SanitizeQuery(query), "args", []any{userID, data.cardID, data.quantity, data.condition, data.notes})
		_, err := s.db.Exec(query, userID, data.cardID, data.quantity, data.condition, data.notes)
		if err != nil {
			slog.Error("failed to create sample collection data", "user_id", userID, "card_id", data.cardID, "error", err)
			return &FailedToCreateSampleCollectionDataError{Err: err}
		}
	}

	slog.Debug("created sample collection data", "user_id", userID, "sample_count", len(sampleData))
	return nil
}

// GetCardQuantity retrieves the quantity of a specific card in a user's collection
func (s *UserCollectionServiceImpl) GetCardQuantity(userID, cardID int64) (int, error) {
	var quantity int
	err := db.QueryRow(s.db, selectCardQuantityQuery, userID, cardID).Scan(&quantity)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		slog.Error("failed to get card quantity", "user_id", userID, "card_id", cardID, "error", err)
		return 0, &FailedToGetQuantityError{Err: err}
	}
	return quantity, nil
}

// GetAllQuantitiesForGame retrieves all card quantities for a specific game
func (s *UserCollectionServiceImpl) GetAllQuantitiesForGame(userID, gameID int64) (map[int64]int, error) {
	rows, err := db.Query(s.db, selectAllQuantitiesForGameQuery, userID, gameID)
	if err != nil {
		slog.Error("failed to get all quantities for game", "user_id", userID, "game_id", gameID, "error", err)
		return nil, &FailedToGetQuantityError{Err: err}
	}
	defer rows.Close()
	quantities := make(map[int64]int)
	for rows.Next() {
		var cardID int64
		var quantity int
		if err := rows.Scan(&cardID, &quantity); err != nil {
			slog.Error("failed to scan quantity row", "error", err)
			continue
		}
		quantities[cardID] = quantity
	}
	if err := rows.Err(); err != nil {
		slog.Error("error iterating quantity rows", "error", err)
		return nil, &FailedToGetQuantityError{Err: err}
	}
	slog.Debug("retrieved all quantities for game", "user_id", userID, "game_id", gameID, "count", len(quantities))
	return quantities, nil
}

// IncrementQuantity increments the quantity of a card in a user's collection
func (s *UserCollectionServiceImpl) IncrementQuantity(ctx context.Context, userID, cardID int64) error {
	currentQty, err := s.GetCardQuantity(userID, cardID)
	if err != nil {
		return err
	}
	newQty := currentQty + 1
	_, err = db.ExecContext(ctx, s.db, upsertCollectionQuery, userID, cardID, newQty)
	if err != nil {
		slog.Error("failed to increment quantity", "user_id", userID, "card_id", cardID, "error", err)
		return &FailedToIncrementQuantityError{Err: err}
	}
	slog.Debug("incremented card quantity", "user_id", userID, "card_id", cardID, "new_quantity", newQty)
	return nil
}

// DecrementQuantity decrements the quantity of a card in a user's collection
func (s *UserCollectionServiceImpl) DecrementQuantity(ctx context.Context, userID, cardID int64) error {
	currentQty, err := s.GetCardQuantity(userID, cardID)
	if err != nil {
		return err
	}
	if currentQty <= 0 {
		return nil
	}
	newQty := currentQty - 1
	if newQty == 0 {
		_, err = db.ExecContext(ctx, s.db, deleteCollectionQuery, userID, cardID)
		if err != nil {
			slog.Error("failed to delete card from collection", "user_id", userID, "card_id", cardID, "error", err)
			return &FailedToDecrementQuantityError{Err: err}
		}
		slog.Debug("removed card from collection", "user_id", userID, "card_id", cardID)
		return nil
	}
	_, err = db.ExecContext(ctx, s.db, upsertCollectionQuery, userID, cardID, newQty)
	if err != nil {
		slog.Error("failed to decrement quantity", "user_id", userID, "card_id", cardID, "error", err)
		return &FailedToDecrementQuantityError{Err: err}
	}
	slog.Debug("decremented card quantity", "user_id", userID, "card_id", cardID, "new_quantity", newQty)
	return nil
}

// UpsertCollectionBatch updates multiple card quantities in a single transaction
func (s *UserCollectionServiceImpl) UpsertCollectionBatch(ctx context.Context, userID int64, updates map[int64]int) error {
	return db.WithTransaction(ctx, s.db, func(tx *sql.Tx) error {
		for cardID, quantity := range updates {
			if quantity <= 0 {
				_, err := db.ExecContextTx(ctx, tx, deleteCollectionQuery, userID, cardID)
				if err != nil {
					slog.Error("failed to delete card in batch", "user_id", userID, "card_id", cardID, "error", err)
					return &FailedToUpsertCollectionError{Err: err}
				}
				continue
			}
			_, err := db.ExecContextTx(ctx, tx, upsertCollectionQuery, userID, cardID, quantity)
			if err != nil {
				slog.Error("failed to upsert card in batch", "user_id", userID, "card_id", cardID, "error", err)
				return &FailedToUpsertCollectionError{Err: err}
			}
		}
		slog.Debug("batch upserted collection", "user_id", userID, "update_count", len(updates))
		return nil
	})
}
