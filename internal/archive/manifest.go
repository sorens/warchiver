package archive

import (
	"path/filepath"
	"strings"
	"time"
)

// Feature is a bitmask of what data types are included in a warchive.
type Feature uint32

const (
	FeatureAPIData       Feature = 1 << 0 // data collected from Battle.net API
	FeatureAddonData     Feature = 1 << 1 // data from WoW addon SavedVariables
	FeatureReputations   Feature = 1 << 2
	FeatureCurrencies    Feature = 1 << 3
	FeatureTransmog      Feature = 1 << 4
	FeatureDecor         Feature = 1 << 5 // Housing decor
	FeatureHousing       Feature = 1 << 6 // Player Housing
	FeatureGuild         Feature = 1 << 7
	FeatureProtected     Feature = 1 << 8 // used protected-character endpoint
	FeatureSimcHTML      Feature = 1 << 9
	FeatureSimcJSON      Feature = 1 << 10
	FeatureSimcText      Feature = 1 << 11
	FeatureGearSummary   Feature = 1 << 12
	FeatureHealerTopGear Feature = 1 << 13
	FeatureItemCache     Feature = 1 << 14
)

// Has reports whether flag is set.
func (f Feature) Has(flag Feature) bool { return f&flag != 0 }

// Set adds flag to f.
func (f *Feature) Set(flag Feature) { *f |= flag }

// String returns a comma-separated list of set feature names.
func (f Feature) String() string {
	type namedFlag struct {
		flag Feature
		name string
	}
	flags := []namedFlag{
		{FeatureAPIData, "api-data"},
		{FeatureAddonData, "addon-data"},
		{FeatureReputations, "reputations"},
		{FeatureCurrencies, "currencies"},
		{FeatureTransmog, "transmog"},
		{FeatureDecor, "decor"},
		{FeatureHousing, "housing"},
		{FeatureGuild, "guild"},
		{FeatureProtected, "protected"},
		{FeatureSimcHTML, "simc-html"},
		{FeatureSimcJSON, "simc-json"},
		{FeatureSimcText, "simc-text"},
		{FeatureGearSummary, "gear-summary"},
		{FeatureHealerTopGear, "healer-topgear"},
		{FeatureItemCache, "item-cache"},
	}
	var names []string
	for _, nf := range flags {
		if f.Has(nf.flag) {
			names = append(names, nf.name)
		}
	}
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ",")
}

// SimcRef records the simc file generated for a character.
type SimcRef struct {
	Path     string `json:"path"`                // e.g. "simc/simc-realm-character-...txt"
	Checksum string `json:"checksum"`            // full SHA256 hex of file bytes
	HTMLPath string `json:"html_path,omitempty"` // simc's html "html=" report
	JSONPath string `json:"json_path,omitempty"` // simc's json "json2=" report
	TextPath string `json:"text_path,omitempty"` // simc's plain-text "output=" report
}

// GearRef records the gear-summary artifacts written for a character.
type GearRef struct {
	SummaryPath     string `json:"summary_path,omitempty"`      // e.g. "gear/<name>.<ts>.<hash>_gear.json" (DPS/tank)
	ReportPath      string `json:"report_path,omitempty"`       // e.g. "gear/<name>.<ts>.<hash>_report.txt" (DPS/tank)
	TopGearTextPath string `json:"topgear_text_path,omitempty"` // e.g. "gear/<name>.<ts>.<hash>_topgear.txt" (heal)
	TopGearJSONPath string `json:"topgear_json_path,omitempty"` // e.g. "gear/<name>.<ts>.<hash>_topgear.json" (heal)
}

// AddonRef records the addon data bundled under a character directory.
type AddonRef struct {
	Dir            string   `json:"dir"` // e.g. "profile/wow/character/realm/character/addon"
	SchemaVersion  int      `json:"schema_version"`
	DataTypes      []string `json:"data_types"` // which of the 17 sections are present
	LatestScanTime string   `json:"latest_scan_time"`
}

// CharacterEntry describes a character included in the archive.
type CharacterEntry struct {
	Game     string    `json:"game,omitempty"`
	Region   string    `json:"region"`
	Realm    string    `json:"realm"`
	Name     string    `json:"name"`
	Filename string    `json:"filename"`
	Level    int       `json:"level,omitempty"`
	Class    string    `json:"class,omitempty"`
	RealmID  int       `json:"realm_id,omitempty"`
	CharID   int       `json:"char_id,omitempty"`
	CharHash string    `json:"char_hash,omitempty"`
	Simc     *SimcRef  `json:"simc,omitempty"`
	Addon    *AddonRef `json:"addon,omitempty"`
	Gear     *GearRef  `json:"gear,omitempty"`
}

func (c *CharacterEntry) Copy() *CharacterEntry {
	if c == nil {
		return nil
	}

	return &CharacterEntry{
		Region:   c.Region,
		Game:     c.Game,
		Realm:    c.Realm,
		Name:     c.Name,
		Filename: c.Filename,
		Level:    c.Level,
		Class:    c.Class,
		RealmID:  c.RealmID,
		CharID:   c.CharID,
		CharHash: c.CharHash,
		Simc:     c.Simc,
		Addon:    c.Addon,
		Gear:     c.Gear,
	}
}

func (c *CharacterEntry) DisplayName(base string) string {
	displayName, err := filepath.Rel(base, filepath.Dir(c.Filename))
	if err != nil {
		return c.Filename
	}

	return displayName
}

// ItemsRef records the global item cache snapshot bundled at the archive root.
// The snapshot is account-global (not per-character) — see plan
// .claude/plans/typed-pondering-aho.md.
type ItemsRef struct {
	Filename string `json:"filename"` // archive-relative path, e.g. "items.json"
	Count    int    `json:"count"`    // number of variants serialized
	Checksum string `json:"checksum,omitempty"`
}

type Context struct {
	Game      string `json:"game"`
	Region    string `json:"region"`
	Namespace string `json:"namespace,omitempty"`
	Status    string `json:"status,omitempty"`
}

// Manifest is the manifest.json in a warchive zip.
type Manifest struct {
	SchemaVersion int               `json:"schema_version,omitempty"`
	Version       string            `json:"version"`
	CreatedAt     time.Time         `json:"created_at"`
	Account       string            `json:"account,omitempty"`
	Regions       []string          `json:"regions"`
	Contexts      []Context         `json:"contexts,omitempty"`
	Features      Feature           `json:"features"`
	Source        string            `json:"source"` // "api", "addon", or "both"
	Characters    []*CharacterEntry `json:"characters"`
	Items         *ItemsRef         `json:"items,omitempty"`
}

func (m *Manifest) Copy() *Manifest {
	if m == nil {
		return nil
	}

	// copy the slice of character entries
	entries := []*CharacterEntry{}
	for _, c := range m.Characters {
		entries = append(entries, c.Copy())
	}
	// copy the regions
	regions := []string{}
	for _, r := range m.Regions {
		regions = append(regions, r)
	}

	return &Manifest{
		Version:       m.Version,
		SchemaVersion: m.SchemaVersion,
		CreatedAt:     m.CreatedAt,
		Account:       m.Account,
		Regions:       regions,
		Contexts:      append([]Context(nil), m.Contexts...),
		Features:      m.Features,
		Source:        m.Source,
		Characters:    entries,
		Items:         m.Items,
	}
}
