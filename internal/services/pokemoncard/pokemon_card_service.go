package pokemoncard

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

type PokemonCardService interface {
	UpsertPokemonCard(ctx context.Context, tx *sql.Tx, cardID int64, pc *model.PokemonCard) error
	GetByCardID(cardID int64) (*model.PokemonCard, error)
}

type PokemonCardServiceImpl struct {
	db *sql.DB
}

func NewPokemonCardService(database *sql.DB) PokemonCardService {
	return &PokemonCardServiceImpl{db: database}
}

const upsertQuery = `INSERT OR REPLACE INTO pokemon_cards
	(card_id, hp, category, stage, evolve_from, description, level, retreat,
	 regulation_mark, legal_standard, legal_expanded, types, attacks, abilities,
	 weaknesses, resistances, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

func (s *PokemonCardServiceImpl) UpsertPokemonCard(ctx context.Context, tx *sql.Tx, cardID int64, pc *model.PokemonCard) error {
	typesJSON, _ := json.Marshal(pc.Types)
	attacksJSON, _ := json.Marshal(pc.Attacks)
	abilitiesJSON, _ := json.Marshal(pc.Abilities)
	weaknessesJSON, _ := json.Marshal(pc.Weaknesses)
	resistancesJSON, _ := json.Marshal(pc.Resistances)

	if _, err := db.ExecContextTx(ctx, tx, upsertQuery,
		cardID, pc.HP, pc.Category, pc.Stage, pc.EvolveFrom, pc.Description,
		pc.Level, pc.Retreat, pc.RegulationMark, pc.LegalStandard, pc.LegalExpanded,
		string(typesJSON), string(attacksJSON), string(abilitiesJSON),
		string(weaknessesJSON), string(resistancesJSON),
	); err != nil {
		slog.Error("failed to upsert pokemon card", "card_id", cardID, "error", err)
		return fmt.Errorf("failed to upsert pokemon card for card %d: %w", cardID, err)
	}
	return nil
}

const selectByCardIDQuery = `SELECT id, card_id, hp, category, stage, evolve_from,
	description, level, retreat, regulation_mark, legal_standard, legal_expanded,
	types, attacks, abilities, weaknesses, resistances
	FROM pokemon_cards WHERE card_id = ?`

func (s *PokemonCardServiceImpl) GetByCardID(cardID int64) (*model.PokemonCard, error) {
	row := s.db.QueryRow(selectByCardIDQuery, cardID)

	var pc model.PokemonCard
	var typesJSON, attacksJSON, abilitiesJSON, weaknessesJSON, resistancesJSON sql.NullString
	var category, stage, evolveFrom, description, level, regulationMark sql.NullString
	var hp, retreat sql.NullInt64

	err := row.Scan(&pc.ID, &pc.CardID, &hp, &category, &stage, &evolveFrom,
		&description, &level, &retreat, &regulationMark,
		&pc.LegalStandard, &pc.LegalExpanded,
		&typesJSON, &attacksJSON, &abilitiesJSON, &weaknessesJSON, &resistancesJSON)
	if err != nil {
		return nil, err
	}

	pc.HP = int(hp.Int64)
	pc.Retreat = int(retreat.Int64)
	pc.Category = category.String
	pc.Stage = stage.String
	pc.EvolveFrom = evolveFrom.String
	pc.Description = description.String
	pc.Level = level.String
	pc.RegulationMark = regulationMark.String

	if typesJSON.Valid {
		_ = json.Unmarshal([]byte(typesJSON.String), &pc.Types)
	}
	if attacksJSON.Valid {
		_ = json.Unmarshal([]byte(attacksJSON.String), &pc.Attacks)
	}
	if abilitiesJSON.Valid {
		_ = json.Unmarshal([]byte(abilitiesJSON.String), &pc.Abilities)
	}
	if weaknessesJSON.Valid {
		_ = json.Unmarshal([]byte(weaknessesJSON.String), &pc.Weaknesses)
	}
	if resistancesJSON.Valid {
		_ = json.Unmarshal([]byte(resistancesJSON.String), &pc.Resistances)
	}

	return &pc, nil
}
