package api

import (
	"encoding/json"
	"strings"

	slugtypes "github.com/sorens/warchiver/internal/slug"
)

type AccountSlug struct {
	ID         int                        `json:"id"`
	Characters []*slugtypes.CharacterSlug `json:"characters"`
}

type ProfileSlug struct {
	ID       int            `json:"id"`
	Accounts []*AccountSlug `json:"wow_accounts"`

	cmap map[int]map[int]*slugtypes.CharacterSlug
}

func NewProfileSlug() *ProfileSlug {
	return &ProfileSlug{
		Accounts: []*AccountSlug{},
		cmap:     map[int]map[int]*slugtypes.CharacterSlug{},
	}
}

func (s *ProfileSlug) Map() map[int]map[int]*slugtypes.CharacterSlug {
	return s.cmap
}

func (s *ProfileSlug) Characters() []*slugtypes.CharacterSlug {
	c := []*slugtypes.CharacterSlug{}
	for _, a := range s.cmap {
		for _, char := range a {
			c = append(c, char)
		}
	}
	return c
}

func NewProfileFromData(b []byte) (*ProfileSlug, error) {
	slug := NewProfileSlug()
	if err := json.Unmarshal(b, &slug); err != nil {
		return nil, err
	}
	if slug == nil {
		return nil, nil
	}

	for _, account := range slug.Accounts {
		var amap map[int]*slugtypes.CharacterSlug
		var ok bool
		amap, ok = slug.cmap[account.ID]
		if !ok {
			amap = map[int]*slugtypes.CharacterSlug{}
		}

		for _, char := range account.Characters {
			c := &slugtypes.CharacterSlug{
				Name: char.Name,
				ID:   char.ID,
				Realm: &slugtypes.RealmSlug{
					Name: char.Realm.Name,
					ID:   char.Realm.ID,
					Slug: char.Realm.Slug,
				},
			}
			amap[char.ID] = c
		}
		slug.cmap[account.ID] = amap
	}

	return slug, nil
}

func (s *ProfileSlug) Find(r, c string) *slugtypes.CharacterSlug {
	for _, a := range s.cmap {
		for _, char := range a {
			if strings.EqualFold(char.Realm.Slug, r) && strings.EqualFold(char.Name, c) {
				return char
			}
		}
	}
	return nil
}
