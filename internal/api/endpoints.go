package api

import (
	"fmt"
	"path/filepath"
	"strings"
)

// EndpointKind categorizes API endpoints
type EndpointKind int

const (
	KindCharacter EndpointKind = iota // /profile/wow/character/{realm}/{char}/...
	KindAccount                       // /profile/user/wow/...  requires user auth
	KindProtected                     // /profile/user/wow/protected-character/...
	KindGuild                         // /data/wow/guild/{realm}/{name}/...
	KindData                          // /data/wow/playable-race/index
	KindMedia                         // media files e.g. character renders
)

// Endpoint defines a downloadable API endpoint
type Endpoint struct {
	Game             GameProperty
	id               string       //
	IsIndex          bool         // is index API?
	IsProfile        bool         // the index data that guides all future connections?
	Base             string       // base prefix e.g. /profile/wow/character, /profile/user/wow, /data/wow/guild
	Path             string       // path fmt string
	RequestData      []string     // slice of ordered request data as strings
	Namespace        string       // namespace fmt string e.g. profile-%s, static-%s, dynamic-%s
	Filename         string       // output filename
	Kind             EndpointKind // endpoint category
	Optional         bool         // skip 404s silently
	RequiresUserAuth bool         // needs Authorization Code token (not client credentials)
	Url              string       // url
}

func (e *Endpoint) Copy() *Endpoint {
	if e == nil {
		return nil
	}

	// Copy the slice of strings
	var requestDataCopy []string
	if e.RequestData != nil {
		requestDataCopy = make([]string, len(e.RequestData))
		copy(requestDataCopy, e.RequestData)
	}

	return &Endpoint{
		Game:             e.Game,
		id:               e.id,
		IsIndex:          e.IsIndex,
		IsProfile:        e.IsProfile,
		Base:             e.Base,
		Path:             e.Path,
		RequestData:      requestDataCopy,
		Namespace:        e.Namespace,
		Filename:         e.Filename,
		Kind:             e.Kind,
		Optional:         e.Optional,
		RequiresUserAuth: e.RequiresUserAuth,
		Url:              e.Url,
	}
}

func (e *Endpoint) ID() (string, error) {
	id := e.Base
	switch e.Kind {
	case KindCharacter, KindGuild:
		if len(e.RequestData) != 2 {
			return "", fmt.Errorf("missing request data for character api")
		}

		for _, s := range e.RequestData {
			id = filepath.Join(id, s)
		}
		id = filepath.Join(id, e.Path)
	case KindProtected:
		if len(e.RequestData) != 2 {
			return "", fmt.Errorf("missing request data for protected api %v", e.RequestData)
		}
		id = filepath.Join(id, e.Path)
		id = filepath.Join(id, fmt.Sprintf("%s-%s", e.RequestData[0], e.RequestData[1]))
	default:
		id = filepath.Join(id, e.Path)
	}

	return id, nil
}

func (e *Endpoint) Name() string {
	if e.Path == "" && e.IsIndex {
		return fmt.Sprintf("root for %s", e.Filename)
	}

	if e.Path != "" {
		return fmt.Sprintf("%s for %s", e.Path, e.Filename)
	}

	return fmt.Sprintf("non-index empty path for %s", e.Filename)
}

func (e *Endpoint) String() string {
	base := strings.TrimPrefix(e.Base, "/")
	if len(e.RequestData) > 0 {
		return filepath.Join(base, strings.Join(e.RequestData, "/"), e.Path)
	}

	return filepath.Join(base, e.Path)
}

// AccountEndpoints contains account-level endpoints requiring user authorization
var AccountEndpoints = []*Endpoint{
	{Namespace: "profile-%s", Base: "/profile/user/wow", Kind: KindAccount, Path: "collections", Filename: "account-collections.json", RequiresUserAuth: true, Optional: true},
	{Namespace: "profile-%s", Base: "/profile/user/wow", Kind: KindAccount, Path: "collections/mounts", Filename: "account-mounts.json", RequiresUserAuth: true},
	{Namespace: "profile-%s", Base: "/profile/user/wow", Kind: KindAccount, Path: "collections/pets", Filename: "account-pets.json", RequiresUserAuth: true},
	{Namespace: "profile-%s", Base: "/profile/user/wow", Kind: KindAccount, Path: "collections/toys", Filename: "account-toys.json", RequiresUserAuth: true},
	{Namespace: "profile-%s", Base: "/profile/user/wow", Kind: KindAccount, Path: "collections/heirlooms", Filename: "account-heirlooms.json", RequiresUserAuth: true, Optional: true},
	{Namespace: "profile-%s", Base: "/profile/user/wow", Kind: KindAccount, Path: "collections/decor", Filename: "account-decor.json", RequiresUserAuth: true, Optional: true},
	{Namespace: "profile-%s", Base: "/profile/user/wow", Kind: KindAccount, Path: "collections/transmogs", Filename: "account-transmogs.json", RequiresUserAuth: true},
}

// CharacterEndpoints contains all character profile endpoints
var CharacterEndpoints = []*Endpoint{
	// character endpoints; requires the slug of the realm and the The lowercase name of the character
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "", IsIndex: true, Filename: "profile.json", Optional: false},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "achievements", Filename: "achievements.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "achievements/statistics", Filename: "achievement-statistics.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "appearance", Filename: "appearance.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "character-media", Filename: "character-media.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "collections", Filename: "collections.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "collections/mounts", Filename: "mounts.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "collections/pets", Filename: "pets.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "collections/toys", Filename: "toys.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "collections/heirlooms", Filename: "heirlooms.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "collections/decor", Filename: "collections-decor.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "collections/transmogs", Filename: "collections-transmogs.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "encounters", Filename: "encounters.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "encounters/dungeons", Filename: "dungeons.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "encounters/raids", Filename: "raids.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "equipment", Filename: "equipment.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "hunter-pets", Filename: "hunter-pets.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "mythic-keystone-profile", Filename: "mythic-keystone-profile.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "professions", Filename: "professions.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "pvp-summary", Filename: "pvp-summary.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "pvp-bracket/2v2", Filename: "pvp-2v2.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "pvp-bracket/3v3", Filename: "pvp-3v3.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "pvp-bracket/rbg", Filename: "pvp-rbg.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "quests", Filename: "quests.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "quests/completed", Filename: "quests-completed.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "reputations", Filename: "reputations.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "soulbinds", Filename: "soulbinds.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "specializations", Filename: "specializations.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "statistics", Filename: "statistics.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "titles", Filename: "titles.json", Optional: true},
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "status", Filename: "status.json", Optional: true},
	// houses must be discovered
	// "houses": [
	//    {
	//      "href": "https://us.api.blizzard.com/profile/wow/character/<realm>/<character>/house/house-1?namespace=profile-us"
	//    }
	//  ],
	{Namespace: "profile-%s", Base: "/profile/wow/character", Kind: KindCharacter, Path: "house/house-1", Filename: "house-1.json", Optional: true},
}

// GuildEndpoints contains guild-related endpoints
var GuildEndpoints = []*Endpoint{
	// guild endpoints; requires the slug of the realm and the slug of the guild
	{Namespace: "profile-%s", Base: "/data/wow/guild", Kind: KindGuild, Path: "", IsIndex: true, Filename: "guild.json"},
	{Namespace: "profile-%s", Base: "/data/wow/guild", Kind: KindGuild, Path: "activity", Filename: "guild-activity.json", Optional: true},
	{Namespace: "profile-%s", Base: "/data/wow/guild", Kind: KindGuild, Path: "achievements", Filename: "guild-achievements.json", Optional: true},
	{Namespace: "profile-%s", Base: "/data/wow/guild", Kind: KindGuild, Path: "roster", Filename: "guild-roster.json", Optional: true},
}

var DataEndpoints = []*Endpoint{
	{Namespace: "static-%s", Base: "/data/wow/playable-race/index", Kind: KindData, Path: "", IsIndex: true, Filename: "races.json"},
	{Namespace: "static-%s", Base: "/data/wow/heirloom/index", Kind: KindData, Path: "", IsIndex: true, Filename: "heirlooms.json"},
}

// HeirloomDetailEndpoint creates an endpoint for a single heirloom by ID.
// Files land at data/wow/heirloom/{id}/heirloom.json.
func HeirloomDetailEndpoint(id int) *Endpoint {
	return &Endpoint{
		Namespace: "static-%s",
		Base:      fmt.Sprintf("/data/wow/heirloom/%d", id),
		Kind:      KindData,
		Path:      "",
		Filename:  "heirloom.json",
		Optional:  true,
	}
}

// ItemDetailEndpoint creates an endpoint for a single item by ID. Used by the
// item cache enrichment path to fetch base item details (name, slot, base
// stats) for IDs we have not seen via equipment or addon data. Files land at
// data/wow/item/{id}/item.json.
func ItemDetailEndpoint(id int) *Endpoint {
	return &Endpoint{
		Namespace: "static-%s",
		Base:      fmt.Sprintf("/data/wow/item/%d", id),
		Kind:      KindData,
		Path:      "",
		Filename:  "item.json",
		Optional:  true,
	}
}

// protected is a fixed endpoint
var ProtectedEndpoints = []*Endpoint{
	{Namespace: "profile-%s", Base: "/profile/user/wow", Kind: KindProtected, Path: "protected-character", IsIndex: true, Filename: "protected.json", Optional: true},
}

var StartEndpoints = []*Endpoint{
	// account endpoints contains account-level endpoints requiring user authorization
	{Namespace: "profile-%s", Base: "/profile/user/wow", Kind: KindAccount, Path: "", IsIndex: true, IsProfile: true, Filename: "account-profile.json", RequiresUserAuth: true},
}
