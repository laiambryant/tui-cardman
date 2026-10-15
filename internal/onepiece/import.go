package onepiece

import (
	"context"
	"database/sql"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/model"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/onepiececard"
)

// GameName is the name of the One Piece Card Game in the card_games table.
const GameName = "One Piece"

// NewImporter returns the importer for One Piece Card Game data. Booster sets
// and starter decks are both treated as sets.
func NewImporter(database *sql.DB) (*gameimporter.Importer, error) {
	client := NewClient()
	cards := card.NewCardService(database)
	opCards := onepiececard.NewOnePieceCardService(database)
	return gameimporter.New(database, GameName, gameimporter.Source[OPCard]{
		FetchSets: func(ctx context.Context) ([]gameimporter.GameSet, error) {
			opSets, err := client.GetSets(ctx)
			if err != nil {
				return nil, err
			}
			result := make([]gameimporter.GameSet, 0, len(opSets))
			for _, s := range opSets {
				result = append(result, gameimporter.GameSet{APIID: s.SetID, Name: s.SetName, Code: s.SetID})
			}
			return result, nil
		},
		FetchCards: func(ctx context.Context, set gameimporter.GameSet) ([]OPCard, error) {
			return client.GetCardsForSet(ctx, OPSet{SetID: set.APIID, SetName: set.Name})
		},
		StoreCard: func(ctx context.Context, tx *sql.Tx, gameID, setID int64, c OPCard) error {
			cardID, err := cards.UpsertCard(ctx, tx, c.CardSetID, setID, c.CardSetID, c.CardName, c.Rarity, "", gameID)
			if err != nil {
				return err
			}
			return opCards.UpsertOnePieceCard(ctx, tx, cardID, &model.OnePieceCard{
				CardID:        cardID,
				CardColor:     c.CardColor,
				CardType:      c.CardType,
				CardText:      c.CardText,
				SubTypes:      c.SubTypes,
				Attribute:     c.Attribute,
				Life:          c.Life,
				CardCost:      c.CardCost,
				CardPower:     c.CardPower,
				CounterAmount: c.CounterAmount,
			})
		},
	})
}
