package mtg

import (
	"context"
	"database/sql"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/model"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/mtgcard"
)

// GameName is the name of Magic: The Gathering in the card_games table.
const GameName = "Magic: The Gathering"

// NewImporter returns the importer for Magic: The Gathering data.
func NewImporter(database *sql.DB) (*gameimporter.Importer, error) {
	client := NewClient()
	cards := card.NewCardService(database)
	mtgCards := mtgcard.NewMTGCardService(database)
	return gameimporter.New(database, GameName, gameimporter.Source[MTGCard]{
		FetchSets: func(ctx context.Context) ([]gameimporter.GameSet, error) {
			mtgSets, err := client.GetSets(ctx)
			if err != nil {
				return nil, err
			}
			result := make([]gameimporter.GameSet, 0, len(mtgSets))
			for _, s := range mtgSets {
				result = append(result, gameimporter.GameSet{APIID: s.SetCode, Name: s.Name, Code: s.SetCode, Total: s.NumCards})
			}
			return result, nil
		},
		FetchCards: func(ctx context.Context, set gameimporter.GameSet) ([]MTGCard, error) {
			var all []MTGCard
			for page := 1; ; page++ {
				cards, hasMore, err := client.GetCardsForSet(ctx, set.APIID, page)
				if err != nil {
					return nil, err
				}
				all = append(all, cards...)
				if !hasMore {
					return all, nil
				}
			}
		},
		StoreCard: func(ctx context.Context, tx *sql.Tx, gameID, setID int64, c MTGCard) error {
			cardID, err := cards.UpsertCard(ctx, tx, c.ID, setID, c.Number, c.Name, c.Rarity, c.Artist, gameID)
			if err != nil {
				return err
			}
			return mtgCards.UpsertMTGCard(ctx, tx, cardID, toMagicCard(cardID, c))
		},
	})
}

func toMagicCard(cardID int64, c MTGCard) *model.MagicCard {
	legalities := make([]model.MTGLegality, 0, len(c.Legalities))
	for _, l := range c.Legalities {
		legalities = append(legalities, model.MTGLegality{Format: l.Format, LegalityName: l.LegalityName})
	}
	return &model.MagicCard{
		CardID:        cardID,
		ManaCost:      c.ManaCost,
		CMC:           c.CMC,
		Colors:        c.Colors,
		ColorIdentity: c.ColorIdentity,
		TypeLine:      c.Type,
		Types:         c.Types,
		Supertypes:    c.Supertypes,
		Subtypes:      c.Subtypes,
		Text:          c.Text,
		Flavor:        c.Flavor,
		Power:         c.Power,
		Toughness:     c.Toughness,
		Loyalty:       c.Loyalty,
		Layout:        c.Layout,
		Legalities:    legalities,
	}
}
