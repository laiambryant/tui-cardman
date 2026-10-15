package pokemontcg

import (
	"context"
	"database/sql"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/model"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/pokemoncard"
	"github.com/laiambryant/tui-cardman/internal/services/prices"
)

// GameName is the name of the Pokemon TCG in the card_games table.
const GameName = "Pokemon"

type importClient interface {
	GetSets(ctx context.Context) ([]Set, error)
	GetCardsForSet(ctx context.Context, setID string, page int) (*PaginatedResponse, []Card, error)
}

// NewImporter returns the importer for Pokemon TCG data from the TCGDex API.
func NewImporter(database *sql.DB, apiKey string) (*gameimporter.Importer, error) {
	return newImporter(database, NewClient(apiKey))
}

func newImporter(database *sql.DB, client importClient) (*gameimporter.Importer, error) {
	store := newCardStore(database)
	return gameimporter.New(database, GameName, gameimporter.Source[Card]{
		FetchSets: func(ctx context.Context) ([]gameimporter.GameSet, error) {
			ptcgSets, err := client.GetSets(ctx)
			if err != nil {
				return nil, err
			}
			result := make([]gameimporter.GameSet, 0, len(ptcgSets))
			for _, s := range ptcgSets {
				result = append(result, gameimporter.GameSet{
					APIID:        s.ID,
					Name:         s.Name,
					Code:         s.PtcgoCode,
					PrintedTotal: s.PrintedTotal,
					Total:        s.Total,
				})
			}
			return result, nil
		},
		FetchCards: func(ctx context.Context, set gameimporter.GameSet) ([]Card, error) {
			var all []Card
			for page := 1; ; page++ {
				resp, cards, err := client.GetCardsForSet(ctx, set.APIID, page)
				if err != nil {
					return nil, err
				}
				all = append(all, cards...)
				if !resp.HasMore {
					return all, nil
				}
			}
		},
		StoreCard: store.storeCard,
	})
}

// cardStore writes a Pokemon card, its prices and its Pokemon-specific data.
type cardStore struct {
	cards      card.CardService
	tcgPrices  prices.TCGPlayerPriceService
	cmPrices   prices.CardMarketPriceService
	pokemonSvc pokemoncard.PokemonCardService
}

func newCardStore(database *sql.DB) *cardStore {
	return &cardStore{
		cards:      card.NewCardService(database),
		tcgPrices:  prices.NewTCGPlayerPriceService(database),
		cmPrices:   prices.NewCardMarketPriceService(database),
		pokemonSvc: pokemoncard.NewPokemonCardService(database),
	}
}

func (s *cardStore) storeCard(ctx context.Context, tx *sql.Tx, gameID, setID int64, c Card) error {
	cardID, err := s.cards.UpsertCard(ctx, tx, c.ID, setID, c.Number, c.Name, c.Rarity, c.Artist, gameID)
	if err != nil {
		return err
	}
	if err := s.replacePrices(ctx, tx, cardID, c); err != nil {
		return err
	}
	return s.pokemonSvc.UpsertPokemonCard(ctx, tx, cardID, toPokemonCard(cardID, c))
}

func (s *cardStore) replacePrices(ctx context.Context, tx *sql.Tx, cardID int64, c Card) error {
	if err := s.tcgPrices.DeletePrices(ctx, tx, cardID); err != nil {
		return err
	}
	if c.TCGPlayer != nil {
		for priceType, p := range c.TCGPlayer.Prices {
			if err := s.tcgPrices.InsertPrice(ctx, tx, cardID, priceType,
				p.Low, p.Mid, p.High, p.Market, p.DirectLow, c.TCGPlayer.URL, c.TCGPlayer.UpdatedAt); err != nil {
				return err
			}
		}
	}
	if err := s.cmPrices.DeletePrices(ctx, tx, cardID); err != nil {
		return err
	}
	if p, ok := primaryCardMarketPrice(c.CardMarket); ok {
		return s.cmPrices.InsertPrice(ctx, tx, cardID, p.Avg, p.Trend, c.CardMarket.URL)
	}
	return nil
}

// primaryCardMarketPrice picks the CardMarket price shown for a card. The
// prices table has no variant column, so only one variant is stored, chosen
// deterministically: normal, then holo, then reverse holo.
func primaryCardMarketPrice(cm *CardMarketPrices) (CardMarketPrice, bool) {
	if cm == nil {
		return CardMarketPrice{}, false
	}
	for _, variant := range []string{"normal", "holo", "reverseHolo"} {
		if p, ok := cm.Prices[variant]; ok {
			return p, true
		}
	}
	return CardMarketPrice{}, false
}

func toPokemonCard(cardID int64, c Card) *model.PokemonCard {
	pc := &model.PokemonCard{
		CardID:         cardID,
		HP:             c.HP,
		Retreat:        c.Retreat,
		Category:       c.Category,
		Stage:          c.Stage,
		EvolveFrom:     c.EvolveFrom,
		Description:    c.Description,
		Level:          c.Level,
		RegulationMark: c.RegulationMark,
		LegalStandard:  c.LegalStandard,
		LegalExpanded:  c.LegalExpanded,
		Types:          c.Types,
	}
	for _, a := range c.Attacks {
		pc.Attacks = append(pc.Attacks, model.PokemonCardAttack{Name: a.Name, Cost: a.Cost, Effect: a.Effect, Damage: a.Damage})
	}
	for _, a := range c.Abilities {
		pc.Abilities = append(pc.Abilities, model.PokemonCardAbility{Type: a.Type, Name: a.Name, Effect: a.Effect})
	}
	for _, w := range c.Weaknesses {
		pc.Weaknesses = append(pc.Weaknesses, model.PokemonCardWeakRes{Type: w.Type, Value: w.Value})
	}
	for _, r := range c.Resistances {
		pc.Resistances = append(pc.Resistances, model.PokemonCardWeakRes{Type: r.Type, Value: r.Value})
	}
	return pc
}
