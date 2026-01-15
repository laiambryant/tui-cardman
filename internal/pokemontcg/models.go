package pokemontcg

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
	Set                    Set               `json:"set"`
	Number                 string            `json:"number"`
	Artist                 string            `json:"artist,omitempty"`
	Rarity                 string            `json:"rarity,omitempty"`
	TCGPlayer              *TCGPlayerPrices  `json:"tcgplayer,omitempty"`
	CardMarket             *CardMarketPrices `json:"cardmarket,omitempty"`
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
