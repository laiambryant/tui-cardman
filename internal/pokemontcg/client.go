// Package pokemontcg provides a client and importer for the Pokemon TCG API.
package pokemontcg

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/laiambryant/tcgdex"
	"github.com/laiambryant/tcgdex/client"
	tcgdexModels "github.com/laiambryant/tcgdex/models"
	"github.com/laiambryant/tcgdex/query"
	"golang.org/x/time/rate"
)

const (
	DefaultPageSize  = 250
	MaxPageSize      = 250
	DefaultTimeout   = 30 * time.Second
	DefaultRateLimit = 500 * time.Millisecond
)

// Client is the Pokemon TCG API client
type Client struct {
	sdk     *tcgdex.TCGDex
	limiter *rate.Limiter
}

// NewClient creates a new Pokemon TCG API client using tcgdex SDK
func NewClient(apiKey string) *Client {
	limiter := rate.NewLimiter(rate.Every(DefaultRateLimit), 1)
	httpClient := &rateLimitedHTTPClient{
		client:  &http.Client{Timeout: DefaultTimeout},
		limiter: limiter,
		apiKey:  apiKey,
	}
	sdk := tcgdex.New(
		client.WithHTTPClient(httpClient),
	)
	return &Client{
		sdk:     sdk,
		limiter: limiter,
	}
}

// rateLimitedHTTPClient wraps http.Client with rate limiting and API key support
type rateLimitedHTTPClient struct {
	client  *http.Client
	limiter *rate.Limiter
	apiKey  string
}

func (c *rateLimitedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if err := c.limiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	return c.client.Do(req)
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalCount int `json:"totalCount"`
}

// GetSets fetches all Pokemon TCG sets
func (c *Client) GetSets(ctx context.Context) ([]Set, error) {
	tcgdexSets, err := c.sdk.Set.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sets: %w", err)
	}
	sets := make([]Set, 0, len(tcgdexSets))
	for _, tcgdexSet := range tcgdexSets {
		sets = append(sets, mapTCGDexSetToSet(tcgdexSet))
	}
	return sets, nil
}

// GetCardsForSet fetches all cards for a specific set with pagination
func (c *Client) GetCardsForSet(ctx context.Context, setID string, page int) (*PaginatedResponse, []Card, error) {
	q := query.New().
		Equal("set.id", setID).
		Paginate(page, MaxPageSize)
	tcgdexCards, err := c.sdk.Card.List(ctx, q)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch cards: %w", err)
	}
	cards := make([]Card, 0, len(tcgdexCards))
	for _, tcgdexCard := range tcgdexCards {
		fullCard, err := c.sdk.Card.Get(ctx, tcgdexCard.ID)
		if err != nil {
			continue
		}
		cards = append(cards, mapTCGDexCardToCard(fullCard))
	}
	// Build pagination response
	paginatedResp := &PaginatedResponse{
		Page:       page,
		PageSize:   MaxPageSize,
		TotalCount: len(cards),
	}
	return paginatedResp, cards, nil
}

// GetCard fetches a single card by ID
func (c *Client) GetCard(ctx context.Context, cardID string) (*Card, error) {
	tcgdexCard, err := c.sdk.Card.Get(ctx, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch card %s: %w", cardID, err)
	}
	card := mapTCGDexCardToCard(tcgdexCard)
	return &card, nil
}

// mapTCGDexSetToSet converts tcgdex Set model to our Set model
func mapTCGDexSetToSet(tcgdexSet tcgdexModels.SetResume) Set {
	return Set{
		ID:           tcgdexSet.ID,
		Name:         tcgdexSet.Name,
		PrintedTotal: tcgdexSet.CardCount.Official,
		Total:        tcgdexSet.CardCount.Total,
		PtcgoCode:    tcgdexSet.ID, // Use ID as code
		UpdatedAt:    "",
	}
}

// mapTCGDexCardToCard converts tcgdex Card model to our Card model
func mapTCGDexCardToCard(tcgdexCard tcgdexModels.Card) Card {
	c := Card{
		ID:   tcgdexCard.ID,
		Name: tcgdexCard.Name,
		Set: Set{
			ID:           tcgdexCard.Set.ID,
			Name:         tcgdexCard.Set.Name,
			PrintedTotal: tcgdexCard.Set.CardCount.Official,
			Total:        tcgdexCard.Set.CardCount.Total,
		},
		Number:         tcgdexCard.LocalID,
		Artist:         getStringOrEmpty(tcgdexCard.Illustrator),
		Rarity:         tcgdexCard.Rarity,
		TCGPlayer:      mapTCGPlayerPrices(tcgdexCard.Pricing),
		CardMarket:     mapCardMarketPrices(tcgdexCard.Pricing),
		HP:             getIntOrZero(tcgdexCard.HP),
		Retreat:        getIntOrZero(tcgdexCard.Retreat),
		Category:       tcgdexCard.Category,
		Stage:          getStringOrEmpty(tcgdexCard.Stage),
		EvolveFrom:     getStringOrEmpty(tcgdexCard.EvolveFrom),
		Description:    getStringOrEmpty(tcgdexCard.Description),
		Level:          getStringOrEmpty(tcgdexCard.Level),
		RegulationMark: getStringOrEmpty(tcgdexCard.RegulationMark),
		LegalStandard:  tcgdexCard.Legal.Standard,
		LegalExpanded:  tcgdexCard.Legal.Expanded,
		Types:          tcgdexCard.Types,
	}

	for _, a := range tcgdexCard.Attacks {
		var dmg string
		if a.Damage != nil {
			dmg = string(*a.Damage)
		}
		c.Attacks = append(c.Attacks, CardAttack{
			Name:   getStringOrEmpty(a.Name),
			Cost:   a.Cost,
			Effect: getStringOrEmpty(a.Effect),
			Damage: dmg,
		})
	}
	for _, a := range tcgdexCard.Abilities {
		c.Abilities = append(c.Abilities, CardAbility{
			Type:   a.Type,
			Name:   getStringOrEmpty(a.Name),
			Effect: getStringOrEmpty(a.Effect),
		})
	}
	for _, w := range tcgdexCard.Weaknesses {
		c.Weaknesses = append(c.Weaknesses, CardWeakRes{
			Type:  w.Type,
			Value: getStringOrEmpty(w.Value),
		})
	}
	for _, r := range tcgdexCard.Resistances {
		c.Resistances = append(c.Resistances, CardWeakRes{
			Type:  r.Type,
			Value: getStringOrEmpty(r.Value),
		})
	}

	return c
}

func getFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func mapTCGPlayerPrices(pricing *tcgdexModels.Pricing) *TCGPlayerPrices {
	if pricing == nil || pricing.TCGPlayer == nil {
		return nil
	}
	tp := pricing.TCGPlayer
	prices := make(map[string]TCGPlayerPrice)
	if tp.Normal != nil {
		prices["normal"] = TCGPlayerPrice{
			Low:       getFloat(tp.Normal.LowPrice),
			Mid:       getFloat(tp.Normal.MidPrice),
			High:      getFloat(tp.Normal.HighPrice),
			Market:    getFloat(tp.Normal.MarketPrice),
			DirectLow: getFloat(tp.Normal.DirectLowPrice),
		}
	}
	if tp.Reverse != nil {
		prices["reverse"] = TCGPlayerPrice{
			Low:       getFloat(tp.Reverse.LowPrice),
			Mid:       getFloat(tp.Reverse.MidPrice),
			High:      getFloat(tp.Reverse.HighPrice),
			Market:    getFloat(tp.Reverse.MarketPrice),
			DirectLow: getFloat(tp.Reverse.DirectLowPrice),
		}
	}
	if len(prices) == 0 {
		return nil
	}
	updatedAt := ""
	if tp.Updated != nil {
		updatedAt = tp.Updated.Format("2006/01/02")
	}
	return &TCGPlayerPrices{
		UpdatedAt: updatedAt,
		Prices:    prices,
	}
}

func mapCardMarketPrices(pricing *tcgdexModels.Pricing) *CardMarketPrices {
	if pricing == nil || pricing.Cardmarket == nil {
		return nil
	}
	cm := pricing.Cardmarket
	prices := make(map[string]CardMarketPrice)
	if cm.Avg != nil || cm.Low != nil || cm.Trend != nil {
		prices["normal"] = CardMarketPrice{
			Avg:   getFloat(cm.Avg),
			Low:   getFloat(cm.Low),
			Trend: getFloat(cm.Trend),
		}
	}
	if cm.AvgHolo != nil || cm.LowHolo != nil || cm.TrendHolo != nil {
		prices["holo"] = CardMarketPrice{
			Avg:   getFloat(cm.AvgHolo),
			Low:   getFloat(cm.LowHolo),
			Trend: getFloat(cm.TrendHolo),
		}
	}
	if cm.AvgReverseHolo != nil || cm.LowReverseHolo != nil || cm.TrendReverseHolo != nil {
		prices["reverseHolo"] = CardMarketPrice{
			Avg:   getFloat(cm.AvgReverseHolo),
			Low:   getFloat(cm.LowReverseHolo),
			Trend: getFloat(cm.TrendReverseHolo),
		}
	}
	if len(prices) == 0 {
		return nil
	}
	updatedAt := ""
	if cm.Updated != nil {
		updatedAt = cm.Updated.Format("2006/01/02")
	}
	return &CardMarketPrices{
		UpdatedAt: updatedAt,
		Prices:    prices,
	}
}

func getIntOrZero(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func getStringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
