// Package model defines the core data models used across the application.
package model

import (
	"time"

	"github.com/laiambryant/tui-cardman/internal/auth"
)

// CardGame represents a card game in the database
type CardGame struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

// Card represents a card in the database
type Card struct {
	ID            int64     `json:"id"`
	CardGameID    int64     `json:"card_game_id"`
	Name          string    `json:"name"`
	Rarity        string    `json:"rarity"`
	IsPlaceholder bool      `json:"is_placeholder"`
	CreatedAt     time.Time `json:"created_at"`
	APIID         string    `json:"api_id"`
	SetID         int64     `json:"set_id"`
	Number        string    `json:"number"`
	Artist        string    `json:"artist"`
	UpdatedAt     time.Time `json:"updated_at"`
	CardGame      *CardGame `json:"card_game,omitempty"`
	Set           *Set      `json:"set,omitempty"`
}

// UserCollection represents a user's card collection entry
type UserCollection struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	CardID       int64     `json:"card_id"`
	Quantity     int       `json:"quantity"`
	Condition    string    `json:"condition"`
	AcquiredDate time.Time `json:"acquired_date"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Card         *Card     `json:"card,omitempty"`
}

type ButtonConfiguration struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"user_id"`
	Configuration string     `json:"configuration"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	User          *auth.User `json:"user,omitempty"`
}

type UserList struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	CardGameID  int64     `json:"card_game_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserListCard struct {
	ID        int64     `json:"id"`
	ListID    int64     `json:"list_id"`
	CardID    int64     `json:"card_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Card      *Card     `json:"card,omitempty"`
}

type TCGPlayerPriceRow struct {
	PriceType  string
	Low        float64
	Mid        float64
	High       float64
	Market     float64
	DirectLow  float64
	URL        string
	SnapshotAt time.Time
}

type CardMarketPriceRow struct {
	AvgPrice   float64
	TrendPrice float64
	URL        string
	SnapshotAt time.Time
}

type Deck struct {
	ID         int64
	UserID     int64
	CardGameID int64
	Name       string
	Format     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type DeckCard struct {
	ID        int64
	DeckID    int64
	CardID    int64
	Quantity  int
	CreatedAt time.Time
	UpdatedAt time.Time
	Card      *Card
}

// PokemonCard represents Pokemon-specific card data
type PokemonCard struct {
	ID, CardID                                                      int64
	HP, Retreat                                                     int
	Category, Stage, EvolveFrom, Description, Level, RegulationMark string
	LegalStandard, LegalExpanded                                    bool
	Types                                                           []string
	Attacks                                                         []PokemonCardAttack
	Abilities                                                       []PokemonCardAbility
	Weaknesses, Resistances                                         []PokemonCardWeakRes
}

type PokemonCardAttack struct {
	Name   string
	Cost   []string
	Effect string
	Damage string
}

type PokemonCardAbility struct {
	Type, Name, Effect string
}

type PokemonCardWeakRes struct {
	Type, Value string
}

// YuGiOhCard represents Yu-Gi-Oh!-specific card data
type YuGiOhCard struct {
	ID, CardID      int64
	CardType        string
	FrameType       string
	Description     string
	ATK, DEF, Level *int
	Attribute, Race *string
	Scale, LinkVal  *int
	LinkMarkers     []string
}

// MagicCard represents Magic: The Gathering-specific card data
type MagicCard struct {
	ID, CardID    int64
	ManaCost      string
	CMC           float64
	Colors        []string
	ColorIdentity []string
	TypeLine      string
	Types         []string
	Supertypes    []string
	Subtypes      []string
	Text          string
	Flavor        string
	Power         string
	Toughness     string
	Loyalty       string
	Layout        string
	Legalities    []MTGLegality
}

type MTGLegality struct {
	Format       string
	LegalityName string
}

type OnePieceCard struct {
	ID, CardID    int64
	CardColor     string
	CardType      string
	CardText      string
	SubTypes      string
	Attribute     string
	Life          string
	CardCost      string
	CardPower     string
	CounterAmount string
}

type Set struct {
	ID           int64     `json:"id"`
	APIID        string    `json:"api_id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	PrintedTotal int       `json:"printed_total"`
	Total        int       `json:"total"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
