package mtgcard

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

type MTGCardService interface {
	UpsertMTGCard(ctx context.Context, tx *sql.Tx, cardID int64, mc *model.MagicCard) error
	GetByCardID(cardID int64) (*model.MagicCard, error)
}

type MTGCardServiceImpl struct {
	db *sql.DB
}

func NewMTGCardService(database *sql.DB) MTGCardService {
	return &MTGCardServiceImpl{db: database}
}

const upsertQuery = `INSERT OR REPLACE INTO magic_cards
	(card_id, mana_cost, cmc, colors, color_identity, type_line,
	 types, supertypes, subtypes, text, flavor,
	 power, toughness, loyalty, layout, legalities, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

func (s *MTGCardServiceImpl) UpsertMTGCard(ctx context.Context, tx *sql.Tx, cardID int64, mc *model.MagicCard) error {
	colorsJSON, _ := json.Marshal(mc.Colors)
	colorIdentityJSON, _ := json.Marshal(mc.ColorIdentity)
	typesJSON, _ := json.Marshal(mc.Types)
	supertypesJSON, _ := json.Marshal(mc.Supertypes)
	subtypesJSON, _ := json.Marshal(mc.Subtypes)
	legalitiesJSON, _ := json.Marshal(mc.Legalities)
	if _, err := db.ExecContextTx(ctx, tx, upsertQuery,
		cardID, mc.ManaCost, mc.CMC, string(colorsJSON), string(colorIdentityJSON),
		mc.TypeLine, string(typesJSON), string(supertypesJSON), string(subtypesJSON),
		mc.Text, mc.Flavor, mc.Power, mc.Toughness, mc.Loyalty, mc.Layout,
		string(legalitiesJSON),
	); err != nil {
		slog.Error("failed to upsert magic card", "card_id", cardID, "error", err)
		return fmt.Errorf("failed to upsert magic card for card %d: %w", cardID, err)
	}
	return nil
}

const selectByCardIDQuery = `SELECT id, card_id, mana_cost, cmc, colors, color_identity,
	type_line, types, supertypes, subtypes, text, flavor,
	power, toughness, loyalty, layout, legalities
	FROM magic_cards WHERE card_id = ?`

func (s *MTGCardServiceImpl) GetByCardID(cardID int64) (*model.MagicCard, error) {
	row := s.db.QueryRow(selectByCardIDQuery, cardID)
	var mc model.MagicCard
	var manaCost, typeLine, text, flavor, power, toughness, loyalty, layout sql.NullString
	var cmc sql.NullFloat64
	var colorsJSON, colorIdentityJSON, typesJSON, supertypesJSON, subtypesJSON, legalitiesJSON sql.NullString
	err := row.Scan(&mc.ID, &mc.CardID,
		&manaCost, &cmc, &colorsJSON, &colorIdentityJSON,
		&typeLine, &typesJSON, &supertypesJSON, &subtypesJSON,
		&text, &flavor, &power, &toughness, &loyalty, &layout,
		&legalitiesJSON,
	)
	if err != nil {
		return nil, err
	}
	mc.ManaCost = manaCost.String
	mc.CMC = cmc.Float64
	mc.TypeLine = typeLine.String
	mc.Text = text.String
	mc.Flavor = flavor.String
	mc.Power = power.String
	mc.Toughness = toughness.String
	mc.Loyalty = loyalty.String
	mc.Layout = layout.String
	unmarshalJSONArray(colorsJSON, &mc.Colors)
	unmarshalJSONArray(colorIdentityJSON, &mc.ColorIdentity)
	unmarshalJSONArray(typesJSON, &mc.Types)
	unmarshalJSONArray(supertypesJSON, &mc.Supertypes)
	unmarshalJSONArray(subtypesJSON, &mc.Subtypes)
	if legalitiesJSON.Valid && legalitiesJSON.String != "" && legalitiesJSON.String != "null" {
		_ = json.Unmarshal([]byte(legalitiesJSON.String), &mc.Legalities)
	}
	return &mc, nil
}

func unmarshalJSONArray(ns sql.NullString, target *[]string) {
	if ns.Valid && ns.String != "" && ns.String != "null" {
		_ = json.Unmarshal([]byte(ns.String), target)
	}
}
