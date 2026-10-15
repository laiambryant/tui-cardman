package mtg

import (
	"context"
	"time"

	mtgsdk "github.com/laiambryant/mtg-sdk-go"
	"github.com/laiambryant/mtg-sdk-go/models"
	"github.com/laiambryant/mtg-sdk-go/query"
	"golang.org/x/time/rate"
)

const pageSize = 100

type Client struct {
	limiter *rate.Limiter
	sdk     *mtgsdk.MTG
}

func NewClient() *Client {
	return &Client{
		limiter: rate.NewLimiter(rate.Every(100*time.Millisecond), 1),
		sdk:     mtgsdk.New(),
	}
}

func (c *Client) GetSets(ctx context.Context) ([]MTGSet, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	sdkSets, err := c.sdk.Sets.List(ctx, query.New())
	if err != nil {
		return nil, err
	}
	sets := make([]MTGSet, 0, len(sdkSets))
	for _, s := range sdkSets {
		sets = append(sets, MTGSet{
			SetCode: s.Code,
			Name:    s.Name,
			Block:   derefString(s.Block),
		})
	}
	return sets, nil
}

// GetCardsForSet fetches one page of cards for a set and reports whether more pages exist.
func (c *Client) GetCardsForSet(ctx context.Context, setCode string, page int) ([]MTGCard, bool, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, false, err
	}
	q := query.New().SetCode(setCode).Page(page).PageSize(pageSize)
	sdkCards, err := c.sdk.Cards.List(ctx, q)
	if err != nil {
		return nil, false, err
	}
	cards := make([]MTGCard, 0, len(sdkCards))
	for _, sc := range sdkCards {
		cards = append(cards, mapSDKCard(sc, setCode))
	}
	// The SDK does not expose the Total-Count header, so a full page means "maybe more".
	return cards, resolveHasMore(page, pageSize, len(cards), 0), nil
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

func mapSDKCard(sc models.Card, setCode string) MTGCard {
	legalities := make([]MTGLegality, 0, len(sc.Legalities))
	for _, l := range sc.Legalities {
		legalities = append(legalities, MTGLegality{
			Format:       l.Format,
			LegalityName: l.Legality,
		})
	}
	return MTGCard{
		ID:            sc.ID,
		Name:          sc.Name,
		ManaCost:      derefString(sc.ManaCost),
		CMC:           derefFloat64(sc.CMC),
		Colors:        sc.Colors,
		ColorIdentity: sc.ColorIdentity,
		Type:          sc.Type,
		Types:         sc.Types,
		Supertypes:    sc.Supertypes,
		Subtypes:      sc.Subtypes,
		Rarity:        sc.Rarity,
		SetCode:       setCode,
		SetName:       sc.SetName,
		Text:          derefString(sc.Text),
		Flavor:        derefString(sc.Flavor),
		Artist:        sc.Artist,
		Number:        derefString(sc.Number),
		Power:         derefString(sc.Power),
		Toughness:     derefString(sc.Toughness),
		Loyalty:       derefString(sc.Loyalty),
		Layout:        sc.Layout,
		MultiverseID:  uint32(derefInt(sc.MultiverseID)),
		Legalities:    legalities,
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
