// Package yugioh provides the Yu-Gi-Oh! API client and import service.
package yugioh

// YGOSet represents a Yu-Gi-Oh! card set fetched from the API.
type YGOSet struct {
	SetName  string
	SetCode  string
	NumCards int
}

// YGOCard is an internal representation of a Yu-Gi-Oh! card for a specific
// set printing, derived from the SDK models.
type YGOCard struct {
	ID          int
	Name        string
	Type        string
	FrameType   string
	Desc        string
	ATK         *int
	DEF         *int
	Level       *int
	Attribute   *string
	Race        string
	Scale       *int
	LinkVal     *int
	LinkMarkers []string
	// Set-specific fields
	SetCode    string // set code, e.g. "LOB"
	Rarity     string // e.g. "Ultra Rare"
	CardNumber string // card's code within the set, e.g. "LOB-005"
}
