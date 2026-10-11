package data

type SpellTooltip struct {
	Spell       RefData `json:"spell"`
	Description string  `json:"description"`
	CastTime    string  `json:"cast_time"`
}

type TalentTooltip struct {
	Talent       RefData      `json:"talent"`
	SpellTooltip SpellTooltip `json:"spell_tooltip"`
}

type TalentEntry struct {
	ID      int           `json:"id"`
	Rank    int           `json:"rank"`
	Tooltip TalentTooltip `json:"tooltip"`
}

type PvpTalentSlot struct {
	SelectedTalent RefData `json:"selected_talent"`
	SlotNumber     int     `json:"slot_number"`
}

// SpecializationsResponse is the top-level shape of specializations/specializations.json.
type SpecializationsResponse struct {
	ActiveSpecialization RefData     `json:"active_specialization"`
	ActiveHeroTalentTree RefData     `json:"active_hero_talent_tree"`
	Specializations      []SpecEntry `json:"specializations"`
}

// SpecEntry is one specialization with its talent loadouts.
type SpecEntry struct {
	Specialization RefData         `json:"specialization"`
	Glyphs         []RefData       `json:"glyphs"`
	PvpTalentSlots []PvpTalentSlot `json:"pvp_talent_slots"`
	Loadouts       []Loadout       `json:"loadouts"`
}

// Loadout is a single talent loadout (active or saved).
type Loadout struct {
	IsActive                bool          `json:"is_active"`
	TalentLoadoutCode       string        `json:"talent_loadout_code"`
	SelectedClassTalentTree RefData       `json:"selected_class_talent_tree"`
	SelectedClassTalents    []TalentEntry `json:"selected_class_talents"`
	SelectedSpecTalentTree  RefData       `json:"selected_spec_talent_tree"`
	SelectedSpecTalents     []TalentEntry `json:"selected_spec_talents"`
	SelectedHeroTalentTree  RefData       `json:"selected_hero_talent_tree"`
	SelectedHeroTalents     []TalentEntry `json:"selected_hero_talents"`
}
