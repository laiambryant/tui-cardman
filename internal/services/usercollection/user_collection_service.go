// Package usercollection provides services for managing a user's card collection.
package usercollection

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

// UserCollectionService defines the interface for user collection operations
type UserCollectionService interface {
	GetUserCollectionByGameID(userID, gameID int64) ([]model.UserCollection, error)
	GetCardQuantity(userID, cardID int64) (int, error)
	UpsertCollectionBatch(ctx context.Context, userID int64, updates map[int64]int) error
	GetAllQuantitiesForGame(userID, gameID int64) (map[int64]int, error)
	GetCollectionValue(userID, gameID int64) (float64, error)
	GetCollectionValueHistory(userID, gameID int64) ([]ValueSnapshot, error)
	SnapshotCollectionValue(ctx context.Context, userID, gameID int64) error
}

type ValueSnapshot struct {
	Date  time.Time
	Value float64
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
	selectUserCollectionByGameIDQuery = `
		SELECT uc.id, uc.user_id, uc.card_id, uc.quantity, uc.condition,
		       uc.acquired_date, uc.notes, uc.created_at, uc.updated_at,
		       c.id, c.card_game_id, c.name, c.rarity, c.is_placeholder, c.created_at,
		       c.api_id, c.number, c.artist,
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
		SELECT quantity FROM user_collections WHERE user_id = ? AND card_id = ?
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

	// A card can carry several TCGPlayer price variants (normal, reverse, ...).
	// Value each owned copy once, preferring the "normal" variant.
	selectCollectionValueQuery = `
		SELECT COALESCE(SUM(uc.quantity * (
			SELECT p.market FROM prices_tcgplayer p
			WHERE p.card_id = uc.card_id AND p.market IS NOT NULL
			ORDER BY p.price_type = 'normal' DESC, p.snapshot_at DESC, p.id
			LIMIT 1
		)), 0)
		FROM user_collections uc
		JOIN cards c ON uc.card_id = c.id
		WHERE uc.user_id = ? AND c.card_game_id = ?
	`
	selectValueHistoryQuery = `
		SELECT snapshot_date, total_value
		FROM collection_value_snapshots
		WHERE user_id = ? AND card_game_id = ?
		ORDER BY snapshot_date ASC
	`
	upsertValueSnapshotQuery = `
		INSERT INTO collection_value_snapshots (user_id, card_game_id, total_value, snapshot_date)
		VALUES (?, ?, ?, DATE('now'))
		ON CONFLICT(user_id, card_game_id, snapshot_date)
		DO UPDATE SET total_value = excluded.total_value, created_at = CURRENT_TIMESTAMP
	`
)

// GetUserCollectionByGameID retrieves collection entries for a specific user and card game
func (s *UserCollectionServiceImpl) GetUserCollectionByGameID(userID, gameID int64) ([]model.UserCollection, error) {
	rows, err := db.Query(s.db, selectUserCollectionByGameIDQuery, userID, gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user collection by game: %w", err)
	}
	defer rows.Close()

	var collections []model.UserCollection
	for rows.Next() {
		collection, err := scanUserCollection(rows)
		if err != nil {
			return nil, err
		}
		collections = append(collections, collection)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user collections: %w", err)
	}
	slog.Debug("retrieved user collection by game", "user_id", userID, "game_id", gameID, "count", len(collections))
	return collections, nil
}

func scanUserCollection(rows *sql.Rows) (model.UserCollection, error) {
	var (
		collection                           model.UserCollection
		card                                 model.Card
		game                                 model.CardGame
		acquiredDate, gameCreatedAt          sql.NullTime
		notes, rarity, apiID, number, artist sql.NullString
		setID                                sql.NullInt64
		setName, setCode                     sql.NullString
	)
	err := rows.Scan(
		&collection.ID, &collection.UserID, &collection.CardID, &collection.Quantity, &collection.Condition,
		&acquiredDate, &notes, &collection.CreatedAt, &collection.UpdatedAt,
		&card.ID, &card.CardGameID, &card.Name, &rarity, &card.IsPlaceholder, &card.CreatedAt,
		&apiID, &number, &artist,
		&game.ID, &game.Name, &gameCreatedAt,
		&setID, &setName, &setCode,
	)
	if err != nil {
		return collection, fmt.Errorf("failed to scan user collection: %w", err)
	}
	collection.AcquiredDate = acquiredDate.Time
	collection.Notes = notes.String
	game.CreatedAt = gameCreatedAt.Time
	card.Rarity = rarity.String
	card.APIID = apiID.String
	card.Number = number.String
	card.Artist = artist.String
	if setID.Valid {
		card.SetID = setID.Int64
		card.Set = &model.Set{ID: setID.Int64, Name: setName.String, Code: setCode.String}
	}
	card.CardGame = &game
	collection.Card = &card
	return collection, nil
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
		return 0, fmt.Errorf("failed to get card quantity: %w", err)
	}
	return quantity, nil
}

// GetAllQuantitiesForGame retrieves all card quantities for a specific game
func (s *UserCollectionServiceImpl) GetAllQuantitiesForGame(userID, gameID int64) (map[int64]int, error) {
	quantities, err := db.QueryQuantities(s.db, selectAllQuantitiesForGameQuery, userID, gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get quantities for game: %w", err)
	}
	return quantities, nil
}

func (s *UserCollectionServiceImpl) GetCollectionValue(userID, gameID int64) (float64, error) {
	var value float64
	err := db.QueryRow(s.db, selectCollectionValueQuery, userID, gameID).Scan(&value)
	if err != nil {
		slog.Error("failed to get collection value", "user_id", userID, "game_id", gameID, "error", err)
		return 0, err
	}
	return value, nil
}

// UpsertCollectionBatch sets the quantity of each card; quantities <= 0 remove the card.
func (s *UserCollectionServiceImpl) UpsertCollectionBatch(ctx context.Context, userID int64, updates map[int64]int) error {
	if err := db.SetQuantities(ctx, s.db, userID, updates, upsertCollectionQuery, deleteCollectionQuery); err != nil {
		return fmt.Errorf("failed to upsert collection: %w", err)
	}
	return nil
}

func (s *UserCollectionServiceImpl) GetCollectionValueHistory(userID, gameID int64) ([]ValueSnapshot, error) {
	rows, err := db.Query(s.db, selectValueHistoryQuery, userID, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var snapshots []ValueSnapshot
	for rows.Next() {
		var snap ValueSnapshot
		if err := rows.Scan(&snap.Date, &snap.Value); err != nil {
			return nil, fmt.Errorf("failed to scan value snapshot: %w", err)
		}
		snapshots = append(snapshots, snap)
	}
	return snapshots, rows.Err()
}

func (s *UserCollectionServiceImpl) SnapshotCollectionValue(ctx context.Context, userID, gameID int64) error {
	value, err := s.GetCollectionValue(userID, gameID)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, s.db, upsertValueSnapshotQuery, userID, gameID, value)
	return err
}
