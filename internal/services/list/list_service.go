package list

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

type ListService interface {
	CreateList(ctx context.Context, userID, cardGameID int64, name, description, color string) (*model.UserList, error)
	GetListsByUserAndGame(userID, cardGameID int64) ([]model.UserList, error)
	GetListByID(listID int64) (*model.UserList, error)
	UpdateList(ctx context.Context, listID int64, name, description, color string) error
	DeleteList(ctx context.Context, listID int64) error
	GetAllQuantitiesForList(listID int64) (map[int64]int, error)
	UpsertListCardBatch(ctx context.Context, listID int64, updates map[int64]int) error
	GetListsContainingCard(userID, cardID int64) ([]model.UserList, error)
}

type ListServiceImpl struct {
	db *sql.DB
}

func NewListService(database *sql.DB) ListService {
	return &ListServiceImpl{db: database}
}

const (
	insertListQuery = `
		INSERT INTO user_lists (user_id, card_game_id, name, description, color)
		VALUES (?, ?, ?, ?, ?)
	`
	selectListsByUserAndGameQuery = `
		SELECT id, user_id, card_game_id, name, description, color, created_at, updated_at
		FROM user_lists
		WHERE user_id = ? AND card_game_id = ?
		ORDER BY name ASC
	`
	selectListByIDQuery = `
		SELECT id, user_id, card_game_id, name, description, color, created_at, updated_at
		FROM user_lists
		WHERE id = ?
	`
	updateListQuery = `
		UPDATE user_lists SET name = ?, description = ?, color = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	deleteListQuery = `
		DELETE FROM user_lists WHERE id = ?
	`
	selectAllQuantitiesForListQuery = `
		SELECT card_id, quantity
		FROM user_list_cards
		WHERE list_id = ?
	`
	upsertListCardQuery = `
		INSERT INTO user_list_cards (list_id, card_id, quantity)
		VALUES (?, ?, ?)
		ON CONFLICT(list_id, card_id)
		DO UPDATE SET quantity = excluded.quantity, updated_at = CURRENT_TIMESTAMP
	`
	deleteListCardQuery = `
		DELETE FROM user_list_cards
		WHERE list_id = ? AND card_id = ?
	`
	selectListsContainingCardQuery = `
		SELECT ul.id, ul.user_id, ul.card_game_id, ul.name, ul.description, ul.color, ul.created_at, ul.updated_at
		FROM user_lists ul
		JOIN user_list_cards ulc ON ul.id = ulc.list_id
		WHERE ulc.card_id = ? AND ul.user_id = ?
	`
)

func (s *ListServiceImpl) CreateList(ctx context.Context, userID, cardGameID int64, name, description, color string) (*model.UserList, error) {
	result, err := db.ExecContext(ctx, s.db, insertListQuery, userID, cardGameID, name, description, color)
	if err != nil {
		slog.Error("failed to create list", "user_id", userID, "name", name, "error", err)
		return nil, &FailedToCreateListError{Err: err}
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, &FailedToCreateListError{Err: err}
	}
	return &model.UserList{
		ID:          id,
		UserID:      userID,
		CardGameID:  cardGameID,
		Name:        name,
		Description: description,
		Color:       color,
	}, nil
}

func (s *ListServiceImpl) GetListsByUserAndGame(userID, cardGameID int64) ([]model.UserList, error) {
	rows, err := db.Query(s.db, selectListsByUserAndGameQuery, userID, cardGameID)
	if err != nil {
		return nil, &FailedToQueryListsError{Err: err}
	}
	defer rows.Close()
	var lists []model.UserList
	for rows.Next() {
		var l model.UserList
		err := rows.Scan(&l.ID, &l.UserID, &l.CardGameID, &l.Name, &l.Description, &l.Color, &l.CreatedAt, &l.UpdatedAt)
		if err != nil {
			return nil, &FailedToScanListError{Err: err}
		}
		lists = append(lists, l)
	}
	if err := rows.Err(); err != nil {
		return nil, &FailedToQueryListsError{Err: err}
	}
	slog.Debug("retrieved lists", "user_id", userID, "game_id", cardGameID, "count", len(lists))
	return lists, nil
}

func (s *ListServiceImpl) GetListByID(listID int64) (*model.UserList, error) {
	var l model.UserList
	err := db.QueryRow(s.db, selectListByIDQuery, listID).Scan(&l.ID, &l.UserID, &l.CardGameID, &l.Name, &l.Description, &l.Color, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, &FailedToQueryListsError{Err: err}
	}
	return &l, nil
}

func (s *ListServiceImpl) UpdateList(ctx context.Context, listID int64, name, description, color string) error {
	_, err := db.ExecContext(ctx, s.db, updateListQuery, name, description, color, listID)
	if err != nil {
		slog.Error("failed to update list", "list_id", listID, "error", err)
		return &FailedToUpdateListError{Err: err}
	}
	return nil
}

func (s *ListServiceImpl) DeleteList(ctx context.Context, listID int64) error {
	_, err := db.ExecContext(ctx, s.db, deleteListQuery, listID)
	if err != nil {
		slog.Error("failed to delete list", "list_id", listID, "error", err)
		return &FailedToDeleteListError{Err: err}
	}
	return nil
}

func (s *ListServiceImpl) GetAllQuantitiesForList(listID int64) (map[int64]int, error) {
	rows, err := db.Query(s.db, selectAllQuantitiesForListQuery, listID)
	if err != nil {
		return nil, &FailedToGetListQuantitiesError{Err: err}
	}
	defer rows.Close()
	quantities := make(map[int64]int)
	for rows.Next() {
		var cardID int64
		var quantity int
		if err := rows.Scan(&cardID, &quantity); err != nil {
			slog.Error("failed to scan list quantity row", "error", err)
			continue
		}
		quantities[cardID] = quantity
	}
	if err := rows.Err(); err != nil {
		return nil, &FailedToGetListQuantitiesError{Err: err}
	}
	slog.Debug("retrieved list quantities", "list_id", listID, "count", len(quantities))
	return quantities, nil
}

func (s *ListServiceImpl) UpsertListCardBatch(ctx context.Context, listID int64, updates map[int64]int) error {
	return db.WithTransaction(ctx, s.db, func(tx *sql.Tx) error {
		for cardID, quantity := range updates {
			if quantity <= 0 {
				_, err := db.ExecContextTx(ctx, tx, deleteListCardQuery, listID, cardID)
				if err != nil {
					slog.Error("failed to delete list card in batch", "list_id", listID, "card_id", cardID, "error", err)
					return &FailedToUpsertListCardError{Err: err}
				}
				continue
			}
			_, err := db.ExecContextTx(ctx, tx, upsertListCardQuery, listID, cardID, quantity)
			if err != nil {
				slog.Error("failed to upsert list card in batch", "list_id", listID, "card_id", cardID, "error", err)
				return &FailedToUpsertListCardError{Err: err}
			}
		}
		slog.Debug("batch upserted list cards", "list_id", listID, "update_count", len(updates))
		return nil
	})
}

func (s *ListServiceImpl) GetListsContainingCard(userID, cardID int64) ([]model.UserList, error) {
	rows, err := db.Query(s.db, selectListsContainingCardQuery, cardID, userID)
	if err != nil {
		return nil, &FailedToQueryListsError{Err: err}
	}
	defer rows.Close()
	var lists []model.UserList
	for rows.Next() {
		var l model.UserList
		if err := rows.Scan(&l.ID, &l.UserID, &l.CardGameID, &l.Name, &l.Description, &l.Color, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, &FailedToScanListError{Err: err}
		}
		lists = append(lists, l)
	}
	if err := rows.Err(); err != nil {
		return nil, &FailedToQueryListsError{Err: err}
	}
	return lists, nil
}
