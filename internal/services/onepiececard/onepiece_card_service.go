package onepiececard

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

type OnePieceCardService interface {
	UpsertOnePieceCard(ctx context.Context, tx *sql.Tx, cardID int64, oc *model.OnePieceCard) error
	GetByCardID(cardID int64) (*model.OnePieceCard, error)
}

type OnePieceCardServiceImpl struct {
	db *sql.DB
}

func NewOnePieceCardService(database *sql.DB) OnePieceCardService {
	return &OnePieceCardServiceImpl{db: database}
}

const upsertQuery = `INSERT OR REPLACE INTO onepiece_cards
	(card_id, card_color, card_type, card_text, sub_types,
	 attribute, life, card_cost, card_power, counter_amount, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

func (s *OnePieceCardServiceImpl) UpsertOnePieceCard(ctx context.Context, tx *sql.Tx, cardID int64, oc *model.OnePieceCard) error {
	if _, err := db.ExecContextTx(ctx, tx, upsertQuery,
		cardID, oc.CardColor, oc.CardType, oc.CardText, oc.SubTypes,
		oc.Attribute, oc.Life, oc.CardCost, oc.CardPower, oc.CounterAmount,
	); err != nil {
		slog.Error("failed to upsert onepiece card", "card_id", cardID, "error", err)
		return fmt.Errorf("failed to upsert onepiece card for card %d: %w", cardID, err)
	}
	return nil
}

const selectByCardIDQuery = `SELECT id, card_id, card_color, card_type, card_text,
	sub_types, attribute, life, card_cost, card_power, counter_amount
	FROM onepiece_cards WHERE card_id = ?`

func (s *OnePieceCardServiceImpl) GetByCardID(cardID int64) (*model.OnePieceCard, error) {
	row := s.db.QueryRow(selectByCardIDQuery, cardID)
	var oc model.OnePieceCard
	var cardColor, cardType, cardText, subTypes, attribute sql.NullString
	var life, cardCost, cardPower, counterAmount sql.NullString
	err := row.Scan(&oc.ID, &oc.CardID,
		&cardColor, &cardType, &cardText, &subTypes,
		&attribute, &life, &cardCost, &cardPower, &counterAmount,
	)
	if err != nil {
		return nil, err
	}
	oc.CardColor = cardColor.String
	oc.CardType = cardType.String
	oc.CardText = cardText.String
	oc.SubTypes = subTypes.String
	oc.Attribute = attribute.String
	oc.Life = life.String
	oc.CardCost = cardCost.String
	oc.CardPower = cardPower.String
	oc.CounterAmount = counterAmount.String
	return &oc, nil
}
