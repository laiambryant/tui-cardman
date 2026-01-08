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
	DefaultPageSize = 250
	MaxPageSize     = 250
)

// Client is the Pokemon TCG API client (now backed by tcgdex)
type Client struct {
	sdk     *tcgdex.TCGDex
	limiter *rate.Limiter
}

// NewClient creates a new Pokemon TCG API client using tcgdex SDK
func NewClient(apiKey string) *Client {
	limiter := rate.NewLimiter(rate.Every(2*time.Second), 1)
	httpClient := &rateLimitedHTTPClient{
		client:  &http.Client{Timeout: 30 * time.Second},
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
	Count      int `json:"count"`
	TotalCount int `json:"totalCount"`
}

// Set represents a Pokemon TCG set
type Set struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	PrintedTotal int       `json:"printedTotal"`
	Total        int       `json:"total"`
	Legalities   Legality  `json:"legalities"`
	PtcgoCode    string    `json:"ptcgoCode"`
	UpdatedAt    string    `json:"updatedAt"`
	Images       SetImages `json:"images"`
}

// SetImages represents set image URLs
type SetImages struct {
	Symbol string `json:"symbol"`
	Logo   string `json:"logo"`
}

// Legality represents legality information
type Legality struct {
	Unlimited string `json:"unlimited,omitempty"`
	Standard  string `json:"standard,omitempty"`
	Expanded  string `json:"expanded,omitempty"`
}

// Card represents a Pokemon TCG card
type Card struct {
	ID                     string            `json:"id"`
	Name                   string            `json:"name"`
	Supertype              string            `json:"supertype"`
	Subtypes               []string          `json:"subtypes"`
	HP                     string            `json:"hp,omitempty"`
	Types                  []string          `json:"types,omitempty"`
	EvolvesFrom            string            `json:"evolvesFrom,omitempty"`
	EvolvesTo              []string          `json:"evolvesTo,omitempty"`
	Attacks                []Attack          `json:"attacks,omitempty"`
	Abilities              []Ability         `json:"abilities,omitempty"`
	Weaknesses             []Effect          `json:"weaknesses,omitempty"`
	Resistances            []Effect          `json:"resistances,omitempty"`
	RetreatCost            []string          `json:"retreatCost,omitempty"`
	ConvertedRetreatCost   int               `json:"convertedRetreatCost,omitempty"`
	Set                    Set               `json:"set"`
	Number                 string            `json:"number"`
	Artist                 string            `json:"artist,omitempty"`
	Rarity                 string            `json:"rarity,omitempty"`
	FlavorText             string            `json:"flavorText,omitempty"`
	NationalPokedexNumbers []int             `json:"nationalPokedexNumbers,omitempty"`
	Legalities             Legality          `json:"legalities"`
	RegulationMark         string            `json:"regulationMark,omitempty"`
	Images                 CardImages        `json:"images"`
	TCGPlayer              *TCGPlayerPrices  `json:"tcgplayer,omitempty"`
	CardMarket             *CardMarketPrices `json:"cardmarket,omitempty"`
}

type Attack struct {
	Name                string   `json:"name"`
	Cost                []string `json:"cost"`
	ConvertedEnergyCost int      `json:"convertedEnergyCost"`
	Damage              string   `json:"damage"`
	Text                string   `json:"text"`
}

type Ability struct {
	Name string `json:"name"`
	Text string `json:"text"`
	Type string `json:"type"`
}

type Effect struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type CardImages struct {
	Small string `json:"small"`
	Large string `json:"large"`
}

type TCGPlayerPrices struct {
	URL       string                    `json:"url"`
	UpdatedAt string                    `json:"updatedAt"`
	Prices    map[string]TCGPlayerPrice `json:"prices"`
}

type TCGPlayerPrice struct {
	Low       float64 `json:"low,omitempty"`
	Mid       float64 `json:"mid,omitempty"`
	High      float64 `json:"high,omitempty"`
	Market    float64 `json:"market,omitempty"`
	DirectLow float64 `json:"directLow,omitempty"`
}

type CardMarketPrices struct {
	URL       string                     `json:"url"`
	UpdatedAt string                     `json:"updatedAt"`
	Prices    map[string]CardMarketPrice `json:"prices"`
}

type CardMarketPrice struct {
	Avg      float64 `json:"avg,omitempty"`
	Low      float64 `json:"low,omitempty"`
	High     float64 `json:"high,omitempty"`
	Reversal float64 `json:"reversal,omitempty"`
	Trend    float64 `json:"trend,omitempty"`
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
			continue // skip
		}
		cards = append(cards, mapTCGDexCardToCard(fullCard))
	}

	// Build pagination response
	paginatedResp := &PaginatedResponse{
		Page:       page,
		PageSize:   MaxPageSize,
		Count:      len(cards),
		TotalCount: len(cards),
	}

	return paginatedResp, cards, nil
}

// GetCard fetches a single card by ID
func (c *Client) GetCard(ctx context.Context, cardID string) (*Card, error) {
	tcgdexCard, err := c.sdk.Card.Get(ctx, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch card: %w", err)
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
		Images: SetImages{
			Symbol: getStringOrEmpty(tcgdexSet.Symbol),
			Logo:   getStringOrEmpty(tcgdexSet.Logo),
		},
	}
}

// mapTCGDexCardToCard converts tcgdex Card model to our Card model
func mapTCGDexCardToCard(tcgdexCard tcgdexModels.Card) Card {
	// Build card images
	images := CardImages{}
	if tcgdexCard.Image != nil {
		images.Small = *tcgdexCard.Image + "/low.jpg"
		images.Large = *tcgdexCard.Image + "/high.jpg"
	}

	// Build attacks
	attacks := make([]Attack, 0, len(tcgdexCard.Attacks))
	for _, a := range tcgdexCard.Attacks {
		attacks = append(attacks, Attack{
			Name:   getStringOrEmpty(a.Name),
			Cost:   a.Cost,
			Damage: string(getStringOrEmpty((*string)(a.Damage))),
			Text:   getStringOrEmpty(a.Effect),
		})
	}

	// Build abilities
	abilities := make([]Ability, 0, len(tcgdexCard.Abilities))
	for _, a := range tcgdexCard.Abilities {
		abilities = append(abilities, Ability{
			Name: getStringOrEmpty(a.Name),
			Text: getStringOrEmpty(a.Effect),
			Type: a.Type,
		})
	}

	hp := ""
	if tcgdexCard.HP != nil {
		hp = fmt.Sprintf("%d", *tcgdexCard.HP)
	}

	return Card{
		ID:          tcgdexCard.ID,
		Name:        tcgdexCard.Name,
		Supertype:   tcgdexCard.Category,
		Subtypes:    []string{}, // Not directly mapped
		HP:          hp,
		Types:       tcgdexCard.Types,
		EvolvesFrom: getStringOrEmpty(tcgdexCard.EvolveFrom),
		EvolvesTo:   []string{}, // tcgdex doesn't provide this
		Attacks:     attacks,
		Abilities:   abilities,
		Weaknesses:  mapWeaknesses(tcgdexCard.Weaknesses),
		Resistances: mapResistances(tcgdexCard.Resistances),
		RetreatCost: []string{},
		Set: Set{
			ID:           tcgdexCard.Set.ID,
			Name:         tcgdexCard.Set.Name,
			PrintedTotal: tcgdexCard.Set.CardCount.Official,
			Total:        tcgdexCard.Set.CardCount.Total,
			Images: SetImages{
				Symbol: getStringOrEmpty(tcgdexCard.Set.Symbol),
				Logo:   getStringOrEmpty(tcgdexCard.Set.Logo),
			},
		},
		Number:         tcgdexCard.LocalID,
		Artist:         getStringOrEmpty(tcgdexCard.Illustrator),
		Rarity:         tcgdexCard.Rarity,
		FlavorText:     getStringOrEmpty(tcgdexCard.Description),
		RegulationMark: getStringOrEmpty(tcgdexCard.RegulationMark),
		Images:         images,
		// Price data not available in tcgdex
		TCGPlayer:  nil,
		CardMarket: nil,
	}
}

func mapWeaknesses(weaknesses []tcgdexModels.CardWeakRes) []Effect {
	effects := make([]Effect, 0, len(weaknesses))
	for _, w := range weaknesses {
		effects = append(effects, Effect{
			Type:  w.Type,
			Value: getStringOrEmpty(w.Value),
		})
	}
	return effects
}

func mapResistances(resistances []tcgdexModels.CardWeakRes) []Effect {
	effects := make([]Effect, 0, len(resistances))
	for _, r := range resistances {
		effects = append(effects, Effect{
			Type:  r.Type,
			Value: getStringOrEmpty(r.Value),
		})
	}
	return effects
}

func getStringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
