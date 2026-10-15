package yugioh

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/model"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/yugiohcard"
)

// GameName is the name of Yu-Gi-Oh! in the card_games table.
const GameName = "Yu-Gi-Oh!"

// NewImporter returns the importer for Yu-Gi-Oh! data. Sets are identified by
// their set code.
func NewImporter(database *sql.DB) (*gameimporter.Importer, error) {
	client := NewClient()
	cards := card.NewCardService(database)
	ygoCards := yugiohcard.NewYuGiOhCardService(database)
	return gameimporter.New(database, GameName, gameimporter.Source[YGOCard]{
		FetchSets: func(ctx context.Context) ([]gameimporter.GameSet, error) {
			ygoSets, err := client.GetSets(ctx)
			if err != nil {
				return nil, err
			}
			result := make([]gameimporter.GameSet, 0, len(ygoSets))
			for _, s := range ygoSets {
				result = append(result, gameimporter.GameSet{APIID: s.SetCode, Name: s.SetName, Code: s.SetCode, Total: s.NumCards})
			}
			return result, nil
		},
		FetchCards: func(ctx context.Context, set gameimporter.GameSet) ([]YGOCard, error) {
			var all []YGOCard
			for offset := 0; ; offset += pageSize {
				cards, morePages, err := client.GetCardsForSet(ctx, set.Name, set.APIID, offset)
				if err != nil {
					return nil, err
				}
				all = append(all, cards...)
				if !morePages {
					return all, nil
				}
			}
		},
		StoreCard: func(ctx context.Context, tx *sql.Tx, gameID, setID int64, c YGOCard) error {
			apiID := fmt.Sprintf("ygo-%d-%s", c.ID, c.SetCode)
			cardID, err := cards.UpsertCard(ctx, tx, apiID, setID, c.CardNumber, c.Name, c.Rarity, "", gameID)
			if err != nil {
				return err
			}
			return ygoCards.UpsertYuGiOhCard(ctx, tx, cardID, toYuGiOhCard(cardID, c))
		},
	})
}

func toYuGiOhCard(cardID int64, c YGOCard) *model.YuGiOhCard {
	yc := &model.YuGiOhCard{
		CardID:      cardID,
		CardType:    c.Type,
		FrameType:   c.FrameType,
		Description: c.Desc,
		ATK:         c.ATK,
		DEF:         c.DEF,
		Level:       c.Level,
		Attribute:   c.Attribute,
		Scale:       c.Scale,
		LinkVal:     c.LinkVal,
		LinkMarkers: c.LinkMarkers,
	}
	if c.Race != "" {
		race := c.Race
		yc.Race = &race
	}
	return yc
}
