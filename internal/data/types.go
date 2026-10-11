package data

import (
	"fmt"

	"github.com/sorens/warchiver/internal/slug"
)

type BasicLink = slug.BasicLink

type RefData struct {
	Key  BasicLink `json:"key"`
	Name string    `json:"name"`
	ID   int       `json:"id"`
}

func (r *RefData) String() string {
	return fmt.Sprintf("%s: [%d] (%s)", r.Key, r.ID, r.Name)
}

type RefSlugData struct {
	Key  BasicLink `json:"key"`
	Name string    `json:"name"`
	ID   int       `json:"id"`
	Slug string    `json:"slug"`
}

func (r *RefSlugData) String() string {
	return fmt.Sprintf("%s: [%d], slug: %s (%s)", r.Key, r.ID, r.Slug, r.Name)
}

type RefDisplayData struct {
	Key           BasicLink `json:"key"`
	Name          string    `json:"name"`
	ID            int       `json:"id"`
	DisplayString string    `json:"display"`
}

func (r *RefDisplayData) String() string {
	return fmt.Sprintf("%s: [%d], display: %s (%s)", r.Key, r.ID, r.DisplayString, r.Name)
}

type TypedData = slug.TypedData

// AccountProfile is the response from GET /profile/user/wow
// Used to enumerate all characters across realms
type AccountProfile struct {
	ID          int          `json:"id"`
	WowAccounts []WowAccount `json:"wow_accounts"`
}

// WowAccount represents a single WoW account under a Battle.net account
type WowAccount struct {
	ID         int         `json:"id"`
	Characters []Character `json:"characters"`
}

type GuildRef struct {
	Key     BasicLink   `json:"key"`
	Name    string      `json:"name"`
	ID      int         `json:"id"`
	Realm   RefSlugData `json:"realm"`
	Faction TypedData   `json:"faction"`
}

type CovenantProgress struct {
	ChosenCovenant RefData   `json:"chosen_covenant"`
	RenownLevel    int       `json:"renown_level"`
	Soulbinds      BasicLink `json:"soulbinds"`
}

// Character represents a WoW character summary from both
// /profile/wow/character/{realmSlug}/{characterName}
// /profile/user/wow
type Character struct {
	// "_links": {
	//    "self": {
	//      "href": "https://us.api.blizzard.com/profile/wow/character/rrrrtttttt/gggghffff?namespace=profile-us"
	//    }
	//  }, // /profile/wow/character/{realmSlug}/{characterName}
	// character // /profile/user/wow
	// protected_character // /profile/user/wow
	ID                      int              `json:"id"`                      // both
	Name                    string           `json:"name"`                    // both
	Gender                  TypedData        `json:"gender"`                  // both
	Faction                 TypedData        `json:"faction"`                 // both
	Race                    RefData          `json:"race"`                    // /profile/wow/character/{realmSlug}/{characterName}
	CharacterClass          RefData          `json:"character_class"`         // /profile/user/wow
	ActiveSpec              RefData          `json:"active_spec"`             // /profile/wow/character/{realmSlug}/{characterName}
	Realm                   RealmRef         `json:"realm"`                   // both
	Guild                   GuildRef         `json:"guild"`                   // /profile/wow/character/{realmSlug}/{characterName}
	Level                   int              `json:"level"`                   // both
	PlayableClass           RefData          `json:"playable_class"`          // /profile/user/wow
	PlayableRace            RefData          `json:"playable_race"`           // /profile/user/wow
	Experience              int              `json:"experience"`              // /profile/wow/character/{realmSlug}/{characterName}
	AchievementPoints       int              `json:"achievement_points"`      // /profile/wow/character/{realmSlug}/{characterName}
	Achievements            BasicLink        `json:"achievements"`            // /profile/wow/character/{realmSlug}/{characterName}
	Titles                  BasicLink        `json:"titles"`                  // /profile/wow/character/{realmSlug}/{characterName}
	PvpSummary              BasicLink        `json:"pvp_summary"`             // /profile/wow/character/{realmSlug}/{characterName}
	Encounters              BasicLink        `json:"encounters"`              // /profile/wow/character/{realmSlug}/{characterName}
	Media                   BasicLink        `json:"media"`                   // /profile/wow/character/{realmSlug}/{characterName}
	LastLoginTimestamp      int64            `json:"last_login_timestamp"`    // /profile/wow/character/{realmSlug}/{characterName}
	AverageItemLevel        int              `json:"average_item_level"`      // /profile/wow/character/{realmSlug}/{characterName}
	EquippedItemLevel       int              `json:"equipped_item_level"`     // /profile/wow/character/{realmSlug}/{characterName}
	Specializations         BasicLink        `json:"specializations"`         // /profile/wow/character/{realmSlug}/{characterName}
	Statistics              BasicLink        `json:"statistics"`              // /profile/wow/character/{realmSlug}/{characterName}
	MythicKeystoneProfile   BasicLink        `json:"mythic_keystone_profile"` // /profile/wow/character/{realmSlug}/{characterName}
	Equipment               BasicLink        `json:"equipment"`               // /profile/wow/character/{realmSlug}/{characterName}
	Appearance              BasicLink        `json:"appearance"`              // /profile/wow/character/{realmSlug}/{characterName}
	Collections             BasicLink        `json:"collections"`             // /profile/wow/character/{realmSlug}/{characterName}
	ActiveTitle             RefDisplayData   `json:"active_title"`            // /profile/wow/character/{realmSlug}/{characterName}
	Reputations             BasicLink        `json:"reputations"`             // /profile/wow/character/{realmSlug}/{characterName}
	Quests                  BasicLink        `json:"quests"`                  // /profile/wow/character/{realmSlug}/{characterName}
	Achievements_statistics BasicLink        `json:"achievements_statistics"` // /profile/wow/character/{realmSlug}/{characterName}
	Professions             BasicLink        `json:"professions"`             // /profile/wow/character/{realmSlug}/{characterName}
	CovenantProgress        CovenantProgress `json:"covenant_progress"`       // /profile/wow/character/{realmSlug}/{characterName}
	IsRemix                 bool             `json:"is_remix"`                // /profile/wow/character/{realmSlug}/{characterName}
	Houses                  BasicLink        `json:"houses"`                  // /profile/wow/character/{realmSlug}/{characterName}
	NameSearch              string           `json:"name_search"`             // /profile/wow/character/{realmSlug}/{characterName}
}

func (c *Character) String() string {
	return fmt.Sprintf("[%10d] %s", c.ID, c.Name)
}

// RealmRef is a reference to a realm in API responses
type RealmRef struct {
	Key  BasicLink `json:"key"`
	Name string    `json:"name"`
	ID   int       `json:"id"`
	Slug string    `json:"slug"`
}

// "titles": [
//
//	{
//	  "key": {
//	    "href": "https://us.api.blizzard.com/data/wow/title/23?namespace=static-12.0.1_65617-us"
//	  },
//	  "name": "Centurion",
//	  "id": 23
//	},
//
// ]
type Title struct {
	Key  BasicLink `json:"key"`
	Name string    `json:"name"`
	ID   string    `json:"id"`
}

type Data struct {
	// achievement_statistics
	// achievements
	// appearance
	// character_media
	// collections
	// collections_decor
	// collections_transmogs
	// dungeons
	// encounters
	// equipment
	// guild
	// guild_achievements
	// guild_activity
	// guild_roster
	// heirlooms
	// meta
	// mounts
	// mythic_keystone_profile
	// pets
	// professions
	// profile
	// pvp_summary
	// quests
	// quests_completed
	// raids
	// reputations
	// soulbinds
	// specializations
	Statistics Statistics `json:"statistics"`
	Status     Status     `json:"status"`
	Titles     []Title    `json:"titles"`
}

// // ClassRef is a reference to a playable class in API responses
// type ClassRef struct {
// 	Key  BasicLink `json:"key"`
// 	Name string    `json:"name"`
// 	ID   int       `json:"id"`
// }

// // RaceRef is a reference to a playable race in API responses
// type RaceRef struct {
// 	Key  BasicLink `json:"key"`
// 	Name string    `json:"name"`
// 	ID   int       `json:"id"`
// }

// // LocalizedString represents a localized enum value from the API
// type LocalizedString struct {
// 	Type string `json:"type"`
// 	Name string `json:"name"`
// }

type CharacterMedia struct {
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"_links"`
	Character struct {
		Key struct {
			Href string `json:"href"`
		} `json:"key"`
		Name  string `json:"name"`
		ID    int    `json:"id"`
		Realm struct {
			Key struct {
				Href string `json:"href"`
			} `json:"key"`
			Name string `json:"name"`
			ID   int    `json:"id"`
			Slug string `json:"slug"`
		} `json:"realm"`
	} `json:"character"`
	Assets []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"assets"`
}

type Self struct {
	Href string `json:"href,omitempty"`
}

type Links struct {
	Self Self `json:"self"`
}

type Status struct {
	Links   Links `json:"_links"`
	ID      int   `json:"id,omitempty"`
	IsValid bool  `json:"is_valid,omitempty"`
}

type RealmSlug = slug.RealmSlug
type GuildSlug = slug.GuildSlug
type CharacterSlug = slug.CharacterSlug

type Statistics struct {
	Links                       Links         `json:"_links"`
	Agility                     Agility       `json:"agility"`
	Armor                       Armor         `json:"armor"`
	AttackPower                 int           `json:"attack_power,omitempty"`
	Avoidance                   Avoidance     `json:"avoidance"`
	Block                       Block         `json:"block"`
	BonusArmor                  int           `json:"bonus_armor"`
	Character                   CharacterSlug `json:"character"`
	Dodge                       Dodge         `json:"dodge"`
	Health                      int           `json:"health,omitempty"`
	Intellect                   Intellect     `json:"intellect"`
	Lifesteal                   Lifesteal     `json:"lifesteal"`
	MainHandDamageMax           float64       `json:"main_hand_damage_max,omitempty"`
	MainHandDamageMin           float64       `json:"main_hand_damage_min,omitempty"`
	MainHandDps                 float64       `json:"main_hand_dps,omitempty"`
	MainHandSpeed               float64       `json:"main_hand_speed,omitempty"`
	ManaRegen                   int           `json:"mana_regen,omitempty"`
	ManaRegenCombat             int           `json:"mana_regen_combat,omitempty"`
	Mastery                     Mastery       `json:"mastery"`
	MeleeCrit                   MeleeCrit     `json:"melee_crit"`
	MeleeHaste                  MeleeHaste    `json:"melee_haste"`
	OffHandDamageMax            int           `json:"off_hand_damage_max,omitempty"`
	OffHandDamageMin            int           `json:"off_hand_damage_min,omitempty"`
	OffHandDps                  int           `json:"off_hand_dps,omitempty"`
	OffHandSpeed                float64       `json:"off_hand_speed,omitempty"`
	Parry                       Parry         `json:"parry"`
	Power                       int           `json:"power,omitempty"`
	PowerType                   PowerType     `json:"power_type"`
	RangedCrit                  RangedCrit    `json:"ranged_crit"`
	RangedHaste                 RangedHaste   `json:"ranged_haste"`
	Speed                       Speed         `json:"speed"`
	SpellCrit                   SpellCrit     `json:"spell_crit"`
	SpellHaste                  SpellHaste    `json:"spell_haste"`
	SpellPenetration            int           `json:"spell_penetration,omitempty"`
	SpellPower                  int           `json:"spell_power,omitempty"`
	Stamina                     Stamina       `json:"stamina"`
	Strength                    Strength      `json:"strength"`
	Versatility                 int           `json:"versatility,omitempty"`
	VersatilityDamageDoneBonus  float64       `json:"versatility_damage_done_bonus,omitempty"`
	VersatilityDamageTakenBonus float64       `json:"versatility_damage_taken_bonus,omitempty"`
	VersatilityHealingDoneBonus float64       `json:"versatility_healing_done_bonus,omitempty"`
}

type Agility struct {
	Base      int `json:"base,omitempty"`
	Effective int `json:"effective,omitempty"`
}

type Armor struct {
	Base      int `json:"base,omitempty"`
	Effective int `json:"effective,omitempty"`
}

type Avoidance struct {
	RatingBonus      int `json:"rating_bonus,omitempty"`
	RatingNormalized int `json:"rating_normalized,omitempty"`
}

type Block struct {
	RatingBonus      int `json:"rating_bonus,omitempty"`
	RatingNormalized int `json:"rating_normalized,omitempty"`
	Value            int `json:"value,omitempty"`
}

type Key struct {
	Href string `json:"href,omitempty"`
}

type Realm struct {
	ID   int    `json:"id,omitempty"`
	Key  Key    `json:"key"`
	Name string `json:"name,omitempty"`
	Slug string `json:"slug,omitempty"`
}

type Character2 struct {
	ID    int    `json:"id,omitempty"`
	Key   Key    `json:"key"`
	Name  string `json:"name,omitempty"`
	Realm Realm  `json:"realm"`
}

type Dodge struct {
	RatingBonus      int     `json:"rating_bonus,omitempty"`
	RatingNormalized int     `json:"rating_normalized,omitempty"`
	Value            float64 `json:"value,omitempty"`
}

type Intellect struct {
	Base      int `json:"base,omitempty"`
	Effective int `json:"effective,omitempty"`
}

type Lifesteal struct {
	RatingBonus      float64 `json:"rating_bonus,omitempty"`
	RatingNormalized int     `json:"rating_normalized,omitempty"`
	Value            float64 `json:"value,omitempty"`
}

type Mastery struct {
	RatingBonus      float64 `json:"rating_bonus,omitempty"`
	RatingNormalized int     `json:"rating_normalized,omitempty"`
	Value            float64 `json:"value,omitempty"`
}

type MeleeCrit struct {
	RatingBonus      float64 `json:"rating_bonus,omitempty"`
	RatingNormalized int     `json:"rating_normalized,omitempty"`
	Value            float64 `json:"value,omitempty"`
}

type MeleeHaste struct {
	RatingBonus      float64 `json:"rating_bonus,omitempty"`
	RatingNormalized int     `json:"rating_normalized,omitempty"`
	Value            float64 `json:"value,omitempty"`
}

type Parry struct {
	RatingBonus      int `json:"rating_bonus,omitempty"`
	RatingNormalized int `json:"rating_normalized,omitempty"`
	Value            int `json:"value,omitempty"`
}

type PowerType struct {
	ID   int    `json:"id,omitempty"`
	Key  Key    `json:"key"`
	Name string `json:"name,omitempty"`
}

type RangedCrit struct {
	RatingBonus      float64 `json:"rating_bonus,omitempty"`
	RatingNormalized int     `json:"rating_normalized,omitempty"`
	Value            float64 `json:"value,omitempty"`
}

type RangedHaste struct {
	RatingBonus      float64 `json:"rating_bonus,omitempty"`
	RatingNormalized int     `json:"rating_normalized,omitempty"`
	Value            float64 `json:"value,omitempty"`
}

type Speed struct {
	RatingBonus      int `json:"rating_bonus,omitempty"`
	RatingNormalized int `json:"rating_normalized,omitempty"`
}

type SpellCrit struct {
	RatingBonus      float64 `json:"rating_bonus,omitempty"`
	RatingNormalized int     `json:"rating_normalized,omitempty"`
	Value            float64 `json:"value,omitempty"`
}

type SpellHaste struct {
	RatingBonus      float64 `json:"rating_bonus,omitempty"`
	RatingNormalized int     `json:"rating_normalized,omitempty"`
	Value            float64 `json:"value,omitempty"`
}

type Stamina struct {
	Base      int `json:"base,omitempty"`
	Effective int `json:"effective,omitempty"`
}

type Strength struct {
	Base      int `json:"base,omitempty"`
	Effective int `json:"effective,omitempty"`
}
