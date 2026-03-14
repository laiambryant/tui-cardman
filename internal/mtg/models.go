package mtg

type MTGSet struct {
	SetCode     string
	Name        string
	Block       string
	ReleaseDate string
	NumCards    int
}

type MTGCard struct {
	ID            string
	Name          string
	ManaCost      string
	CMC           float64
	Colors        []string
	ColorIdentity []string
	Type          string
	Types         []string
	Supertypes    []string
	Subtypes      []string
	Rarity        string
	SetCode       string
	SetName       string
	Text          string
	Flavor        string
	Artist        string
	Number        string
	Power         string
	Toughness     string
	Loyalty       string
	Layout        string
	MultiverseId  uint32
	Legalities    []MTGLegality
}

type MTGLegality struct {
	Format       string
	LegalityName string
}
