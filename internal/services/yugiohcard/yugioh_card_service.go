package yugiohcard

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

// YuGiOhCardService manages persistence of Yu-Gi-Oh!-specific card data.
type YuGiOhCardService interface {
	UpsertYuGiOhCard(ctx context.Context, tx *sql.Tx, cardID int64, yc *model.YuGiOhCard) error
	GetByCardID(cardID int64) (*model.YuGiOhCard, error)
}

type YuGiOhCardServiceImpl struct {
	db *sql.DB
}

func NewYuGiOhCardService(database *sql.DB) YuGiOhCardService {
	return &YuGiOhCardServiceImpl{db: database}
}

const upsertQuery = `INSERT OR REPLACE INTO yugioh_cards
	(card_id, card_type, frame_type, description, atk, def, level,
	 attribute, race, scale, link_val, link_markers, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

func (s *YuGiOhCardServiceImpl) UpsertYuGiOhCard(ctx context.Context, tx *sql.Tx, cardID int64, yc *model.YuGiOhCard) error {
	markersJSON, _ := json.Marshal(yc.LinkMarkers)

	var attribute, race *string
	if yc.Attribute != nil {
		attribute = yc.Attribute
	}
	if yc.Race != nil {
		race = yc.Race
	}

	if _, err := db.ExecContextTx(ctx, tx, upsertQuery,
		cardID, yc.CardType, yc.FrameType, yc.Description,
		yc.ATK, yc.DEF, yc.Level,
		attribute, race, yc.Scale, yc.LinkVal,
		string(markersJSON),
	); err != nil {
		slog.Error("failed to upsert yugioh card", "card_id", cardID, "error", err)
		return fmt.Errorf("failed to upsert yugioh card for card %d: %w", cardID, err)
	}
	return nil
}

const selectByCardIDQuery = `SELECT id, card_id, card_type, frame_type, description,
	atk, def, level, attribute, race, scale, link_val, link_markers
	FROM yugioh_cards WHERE card_id = ?`

func (s *YuGiOhCardServiceImpl) GetByCardID(cardID int64) (*model.YuGiOhCard, error) {
	row := s.db.QueryRow(selectByCardIDQuery, cardID)

	var yc model.YuGiOhCard
	var cardType, frameType, description sql.NullString
	var atk, def, level, scale, linkVal sql.NullInt64
	var attribute, race sql.NullString
	var markersJSON sql.NullString

	err := row.Scan(&yc.ID, &yc.CardID,
		&cardType, &frameType, &description,
		&atk, &def, &level,
		&attribute, &race, &scale, &linkVal,
		&markersJSON,
	)
	if err != nil {
		return nil, err
	}

	yc.CardType = cardType.String
	yc.FrameType = frameType.String
	yc.Description = description.String

	if atk.Valid {
		v := int(atk.Int64)
		yc.ATK = &v
	}
	if def.Valid {
		v := int(def.Int64)
		yc.DEF = &v
	}
	if level.Valid {
		v := int(level.Int64)
		yc.Level = &v
	}
	if attribute.Valid {
		s := attribute.String
		yc.Attribute = &s
	}
	if race.Valid {
		s := race.String
		yc.Race = &s
	}
	if scale.Valid {
		v := int(scale.Int64)
		yc.Scale = &v
	}
	if linkVal.Valid {
		v := int(linkVal.Int64)
		yc.LinkVal = &v
	}
	if markersJSON.Valid && markersJSON.String != "" && markersJSON.String != "null" {
		_ = json.Unmarshal([]byte(markersJSON.String), &yc.LinkMarkers)
	}

	return &yc, nil
}
