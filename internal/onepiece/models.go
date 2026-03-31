// Package onepiece provides types, client, and import logic for One Piece Card Game cards.
package onepiece

type OPSet struct {
	SetID   string
	SetName string
}

type OPCard struct {
	CardName      string
	CardSetID     string
	SetID         string
	SetName       string
	CardText      string
	Rarity        string
	CardColor     string
	CardType      string
	SubTypes      string
	Attribute     string
	Life          string
	CardCost      string
	CardPower     string
	CounterAmount string
}
