package onepiece

import (
	"context"
	"errors"
	"fmt"

	optcgapi "github.com/laiambryant/optcgapi-go-sdk"
	optcgclient "github.com/laiambryant/optcgapi-go-sdk/client"
	"github.com/laiambryant/optcgapi-go-sdk/models"
	"github.com/laiambryant/optcgapi-go-sdk/query"
)

type Client struct {
	sdk *optcgapi.OPTCGAPI
}

func NewClient() *Client {
	return &Client{sdk: optcgapi.New()}
}

func (c *Client) GetSets(ctx context.Context) ([]OPSet, error) {
	boosterSets, err := c.sdk.GetAllSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch booster sets: %w", err)
	}
	starterDecks, err := c.sdk.GetAllDecks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch starter decks: %w", err)
	}
	sets := make([]OPSet, 0, len(boosterSets)+len(starterDecks))
	for _, s := range boosterSets {
		sets = append(sets, OPSet{SetID: s.SetID, SetName: s.SetName})
	}
	for _, d := range starterDecks {
		sets = append(sets, OPSet{SetID: d.StructureDeckID, SetName: d.StructureDeckName})
	}
	return sets, nil
}

func (c *Client) GetCardsForSet(ctx context.Context, set OPSet) ([]OPCard, error) {
	sdkCards, err := c.fetchCardsForSet(ctx, set.SetID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cards for set %s: %w", set.SetID, err)
	}
	cards := make([]OPCard, 0, len(sdkCards))
	for _, sc := range sdkCards {
		cards = append(cards, mapSDKCard(sc))
	}
	return cards, nil
}

func (c *Client) fetchCardsForSet(ctx context.Context, setID string) ([]models.Card, error) {
	setCards, err := c.sdk.GetSetCards(ctx, setID)
	if err == nil && len(setCards) > 0 {
		return setCards, nil
	}
	if err != nil && !errors.Is(err, optcgclient.ErrNotFound) {
		return nil, err
	}
	deckCards, err := c.sdk.GetDeckCards(ctx, setID)
	if err == nil && len(deckCards) > 0 {
		return deckCards, nil
	}
	if err != nil && !errors.Is(err, optcgclient.ErrNotFound) {
		return nil, err
	}
	filtered, err := c.sdk.GetFilteredSetCards(ctx, query.New().SetID(setID))
	if err != nil {
		return nil, fmt.Errorf("set %s not found via any endpoint: %w", setID, err)
	}
	return filtered, nil
}

func mapSDKCard(sc models.Card) OPCard {
	return OPCard{
		CardName:      sc.CardName,
		CardSetID:     sc.CardSetID,
		SetID:         sc.SetID,
		SetName:       sc.SetName,
		CardText:      sc.CardText,
		Rarity:        sc.Rarity,
		CardColor:     sc.CardColor,
		CardType:      sc.CardType,
		SubTypes:      sc.SubTypes,
		Attribute:     sc.Attribute,
		Life:          derefString(sc.Life),
		CardCost:      derefString(sc.CardCost),
		CardPower:     derefString(sc.CardPower),
		CounterAmount: derefString(sc.CounterAmount),
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
