package yugioh

import (
	"context"
	"fmt"

	ygoprodeck "github.com/laiambryant/sdkygopro"
	sdkmodels "github.com/laiambryant/sdkygopro/models"
	"github.com/laiambryant/sdkygopro/query"
)

const pageSize = 100

// Client wraps the ygoprodeck SDK for Yu-Gi-Oh! card data access.
type Client struct {
	sdk *ygoprodeck.YGOProDeck
}

// NewClient creates a new YGO API client.
func NewClient() *Client {
	return &Client{sdk: ygoprodeck.New()}
}

// GetSets returns all available Yu-Gi-Oh! card sets.
func (c *Client) GetSets(ctx context.Context) ([]YGOSet, error) {
	sdkSets, err := c.sdk.GetCardSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch YGO card sets: %w", err)
	}
	sets := make([]YGOSet, 0, len(sdkSets))
	for _, s := range sdkSets {
		sets = append(sets, YGOSet{
			SetName:  s.SetName,
			SetCode:  s.SetCode,
			NumCards: s.NumOfCards,
		})
	}
	return sets, nil
}

// GetCardsForSet fetches a page of Yu-Gi-Oh! cards for the given set.
// offset is zero-based. Returns cards and whether more pages remain.
func (c *Client) GetCardsForSet(ctx context.Context, setName, setCode string, offset int) ([]YGOCard, bool, error) {
	q := query.New().CardSet(setName).Num(pageSize).Offset(offset)
	resp, err := c.sdk.GetCards(ctx, q)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch YGO cards for set %q: %w", setName, err)
	}

	cards := make([]YGOCard, 0, len(resp.Cards))
	for _, sdkCard := range resp.Cards {
		cards = append(cards, mapSDKCard(sdkCard, setName, setCode))
	}

	morePages := false
	if resp.Meta != nil {
		morePages = resp.Meta.RowsRemaining > 0
	}

	return cards, morePages, nil
}

func mapSDKCard(sdkCard sdkmodels.Card, setName, setCode string) YGOCard {
	// Find the matching CardSetEntry for the set being imported
	var rarity, cardNumber string
	for _, entry := range sdkCard.CardSets {
		if entry.SetName == setName {
			rarity = entry.SetRarity
			cardNumber = entry.SetCode // e.g. "LOB-005"
			break
		}
	}

	ygo := YGOCard{
		ID:          sdkCard.ID,
		Name:        sdkCard.Name,
		Type:        sdkCard.Type,
		FrameType:   sdkCard.FrameType,
		Desc:        sdkCard.Desc,
		Race:        sdkCard.Race,
		LinkMarkers: sdkCard.LinkMarkers,
		SetCode:     setCode,
		Rarity:      rarity,
		CardNumber:  cardNumber,
	}

	// Pointer fields — copy only if present in SDK card
	if sdkCard.ATK != nil {
		v := *sdkCard.ATK
		ygo.ATK = &v
	}
	if sdkCard.DEF != nil {
		v := *sdkCard.DEF
		ygo.DEF = &v
	}
	if sdkCard.Level != nil {
		v := *sdkCard.Level
		ygo.Level = &v
	}
	if sdkCard.Attribute != nil {
		s := *sdkCard.Attribute
		ygo.Attribute = &s
	}
	if sdkCard.Scale != nil {
		v := *sdkCard.Scale
		ygo.Scale = &v
	}
	if sdkCard.LinkVal != nil {
		v := *sdkCard.LinkVal
		ygo.LinkVal = &v
	}

	return ygo
}
