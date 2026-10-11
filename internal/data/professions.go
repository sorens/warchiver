package data

type ProfessionSpec struct {
	Name string `json:"name"`
}

// ProfessionsResponse is the top-level shape of professions/professions.json.
type ProfessionsResponse struct {
	Primaries   []ProfessionEntry `json:"primaries"`
	Secondaries []ProfessionEntry `json:"secondaries"`
}

// ProfessionEntry is a single profession with all expansion tiers.
type ProfessionEntry struct {
	Profession     RefData          `json:"profession"`
	Specialization ProfessionSpec   `json:"specialization"`
	Tiers          []ProfessionTier `json:"tiers"`
}

// ProfessionTier is one expansion's worth of skill for a profession.
// Tiers are ordered oldest-first; the last entry is the current expansion.
type ProfessionTier struct {
	SkillPoints    int       `json:"skill_points"`
	MaxSkillPoints int       `json:"max_skill_points"`
	Tier           RefData   `json:"tier"`
	KnownRecipes   []RefData `json:"known_recipes"`
}
