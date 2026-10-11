package slug

import "fmt"

// BasicLink is a hypermedia link in API responses
type BasicLink struct {
	Href string `json:"href"`
}

func (r *BasicLink) String() string {
	return r.Href
}

type TypedData struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func (r *TypedData) String() string {
	return fmt.Sprintf("%s: %s", r.Type, r.Name)
}

type RealmSlug struct {
	Key  BasicLink `json:"key"`
	Name string    `json:"name"`
	ID   int       `json:"id"`
	Slug string    `json:"slug"`
}

type GuildSlug struct {
	Key     BasicLink  `json:"key"`
	Name    string     `json:"name"`
	ID      int        `json:"id"`
	Realm   *RealmSlug `json:"realm"`
	Faction TypedData  `json:"faction"`
}

type CharacterSlug struct {
	ProfileID int
	AccountID int

	Key struct {
		Href string `json:"href"`
	} `json:"key"`
	Name          string     `json:"name"`
	ID            int        `json:"id"`
	Realm         *RealmSlug `json:"realm"`
	Guild         *GuildSlug `json:"guild,omitempty"`
	PlayableClass struct {
		Name string `json:"name"`
	} `json:"playable_class"`
	PlayableRace struct {
		Name string `json:"name"`
	} `json:"playable_race"`
	Faction struct {
		Name string `json:"name"`
	} `json:"faction"`
	Level int  `json:"level"`
	Addon bool `json:"-"`
}
