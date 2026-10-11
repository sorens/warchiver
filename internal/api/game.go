package api

// GameProperty identifies a World of Warcraft game property represented by an API namespace family.
type GameProperty string

type APIContext struct {
	Game   GameProperty
	Region Region
}

func (g GameProperty) String() string { return string(g) }

const (
	GameRetail             GameProperty = "retail"
	GameClassic1x          GameProperty = "classic1x"
	GameClassic            GameProperty = "classic"
	GameClassicAnniversary GameProperty = "classicann"
)

type CharacterKey struct {
	Game   GameProperty `json:"game,omitempty"`
	Region string       `json:"region"`
	Realm  string       `json:"realm"`
	Name   string       `json:"name"`
}

func (k CharacterKey) String() string {
	return string(k.Game) + "|" + k.Region + "|" + k.Realm + "|" + k.Name
}
