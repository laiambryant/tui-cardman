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
	DefaultRateLimit = 2 * time.Second
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
	}
}

// mapTCGDexCardToCard converts tcgdex Card model to our Card model
func mapTCGDexCardToCard(tcgdexCard tcgdexModels.Card) Card {
	return Card{
		ID:   tcgdexCard.ID,
		Name: tcgdexCard.Name,
		Set: Set{
			ID:           tcgdexCard.Set.ID,
			Name:         tcgdexCard.Set.Name,
			PrintedTotal: tcgdexCard.Set.CardCount.Official,
			Total:        tcgdexCard.Set.CardCount.Total,
		},
		Number:     tcgdexCard.LocalID,
		Artist:     getStringOrEmpty(tcgdexCard.Illustrator),
		Rarity:     tcgdexCard.Rarity,
		TCGPlayer:  nil,
		CardMarket: nil,
	}
}

func getStringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
