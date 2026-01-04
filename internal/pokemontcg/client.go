package pokemontcg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

const (
	BaseURL         = "https://api.pokemontcg.io/v2"
	DefaultPageSize = 250
	MaxPageSize     = 250

	// Rate limits
	DailyRateLimitWithKey    = 20000
	DailyRateLimitWithoutKey = 1000
	HourlyRateLimitPerMinute = 30
)

// Client is the Pokemon TCG API client
type Client struct {
	httpClient *http.Client
	apiKey     string
	limiter    *rate.Limiter
}

// NewClient creates a new Pokemon TCG API client
func NewClient(apiKey string) *Client {
	limiter := rate.NewLimiter(rate.Every(2*time.Second), 1)
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey:  apiKey,
		limiter: limiter,
	}
}

// doRequest executes an HTTP request with rate limiting and API key
func (c *Client) doRequest(ctx context.Context, url string) ([]byte, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	Count      int             `json:"count"`
	TotalCount int             `json:"totalCount"`
	Data       json.RawMessage `json:"data"`
}

// Set represents a Pokemon TCG set
type Set struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Series       string    `json:"series"`
	PrintedTotal int       `json:"printedTotal"`
	Total        int       `json:"total"`
	Legalities   Legality  `json:"legalities"`
	PtcgoCode    string    `json:"ptcgoCode"`
	ReleaseDate  string    `json:"releaseDate"`
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
	url := fmt.Sprintf("%s/sets", BaseURL)
	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}
	var response struct {
		Data []Set `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse sets response: %w", err)
	}
	return response.Data, nil
}

// GetCardsForSet fetches all cards for a specific set with pagination
func (c *Client) GetCardsForSet(ctx context.Context, setID string, page int) (*PaginatedResponse, []Card, error) {
	url := fmt.Sprintf("%s/cards?q=set.id:%s&pageSize=%d&page=%d", BaseURL, setID, MaxPageSize, page)
	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, nil, err
	}
	var paginatedResp PaginatedResponse
	if err := json.Unmarshal(body, &paginatedResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse paginated response: %w", err)
	}
	var cards []Card
	if err := json.Unmarshal(paginatedResp.Data, &cards); err != nil {
		return nil, nil, fmt.Errorf("failed to parse cards data: %w", err)
	}
	return &paginatedResp, cards, nil
}

// GetCard fetches a single card by ID
func (c *Client) GetCard(ctx context.Context, cardID string) (*Card, error) {
	url := fmt.Sprintf("%s/cards/%s", BaseURL, cardID)
	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}
	var response struct {
		Data Card `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse card response: %w", err)
	}
	return &response.Data, nil
}
