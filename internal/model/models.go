package model

import "time"

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
