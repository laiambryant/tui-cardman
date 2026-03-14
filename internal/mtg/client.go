package mtg

import (
	"context"

	sdk "github.com/MagicTheGathering/mtg-sdk-go"
	"golang.org/x/time/rate"
)

const pageSize = 100

type Client struct {
	limiter *rate.Limiter
}

func NewClient() *Client {
	return &Client{limiter: rate.NewLimiter(rate.Every(100_000_000), 1)} // 100ms
}

func (c *Client) GetSets(_ context.Context) ([]MTGSet, error) {
	c.limiter.Wait(context.Background())
	sdkSets, err := sdk.NewSetQuery().All()
	if err != nil {
		return nil, err
	}
	sets := make([]MTGSet, 0, len(sdkSets))
	for _, s := range sdkSets {
		sets = append(sets, MTGSet{
			SetCode: string(s.SetCode),
			Name:    s.Name,
			Block:   s.Block,
		})
	}
	return sets, nil
}

func (c *Client) GetCardsForSet(_ context.Context, setCode string, page int) (cards []MTGCard, hasMore bool, totalCount int, err error) {
	c.limiter.Wait(context.Background())
	sdkCards, totalCount, err := sdk.NewQuery().Where(sdk.CardSet, setCode).PageS(page, pageSize)
	if err != nil {
		return nil, false, 0, err
	}
	cards = make([]MTGCard, 0, len(sdkCards))
	for _, sc := range sdkCards {
		cards = append(cards, mapSDKCard(sc, setCode))
	}
	hasMore = resolveHasMore(page, pageSize, len(cards), totalCount)
	return cards, hasMore, totalCount, nil
}

// resolveHasMore determines whether more pages exist.
// When the API returns a non-zero Total-Count header it is authoritative.
// When it is absent (totalCount == 0) a full page implies more pages may exist.
func resolveHasMore(page, ps, fetched, totalCount int) bool {
	if totalCount > 0 {
		return page*ps < totalCount
	}
	return fetched == ps
}

func mapSDKCard(sc *sdk.Card, setCode string) MTGCard {
	legalities := make([]MTGLegality, 0, len(sc.Legalities))
	for _, l := range sc.Legalities {
		legalities = append(legalities, MTGLegality{
			Format:       l.Format,
			LegalityName: l.Legality,
		})
	}
	return MTGCard{
		ID:            string(sc.Id),
		Name:          sc.Name,
		ManaCost:      sc.ManaCost,
		CMC:           sc.CMC,
		Colors:        sc.Colors,
		ColorIdentity: sc.ColorIdentity,
		Type:          sc.Type,
		Types:         sc.Types,
		Supertypes:    sc.Supertypes,
		Subtypes:      sc.Subtypes,
		Rarity:        sc.Rarity,
		SetCode:       setCode,
		SetName:       sc.SetName,
		Text:          sc.Text,
		Flavor:        sc.Flavor,
		Artist:        sc.Artist,
		Number:        sc.Number,
		Power:         sc.Power,
		Toughness:     sc.Toughness,
		Loyalty:       sc.Loyalty,
		Layout:        sc.Layout,
		MultiverseId:  uint32(sc.MultiverseId),
		Legalities:    legalities,
	}
}
