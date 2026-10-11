package archiver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sorens/warchiver/internal/archive"
)

func TestManifestContextsPerGame(t *testing.T) {
	contexts := manifestContexts([]*archive.CharacterEntry{
		{Game: "retail"},
		{Game: "classic"},
		{Game: "retail"},
		{Game: "classic1x"},
		{Game: ""},
		nil,
	}, "eu")
	if len(contexts) != 4 {
		t.Fatalf("contexts = %d, want 4", len(contexts))
	}
	if contexts[0].Namespace != "profile-eu" {
		t.Errorf("retail namespace = %q", contexts[0].Namespace)
	}
	if contexts[1].Namespace != "profile-classic-eu" {
		t.Errorf("classic namespace = %q", contexts[1].Namespace)
	}
	if contexts[3].Game != "retail" {
		t.Errorf("empty game = %q, want retail", contexts[3].Game)
	}
}

func TestArchiveFeaturesAreSetByMatchingArtifacts(t *testing.T) {
	const addonDir = "profile/wow/retail/character/realm/char/addon"
	const simcHTML = "simc/char.html"
	const simcJSON = "simc/char.json"
	const simcText = "simc/char.txt"
	const gearSummary = "gear/char.json"
	const healerTopGear = "gear/char_topgear.txt"

	cases := []struct {
		name    string
		files   []string
		entries []*archive.CharacterEntry
		items   *archive.ItemsRef
		want    archive.Feature
	}{
		{
			name:  "api data",
			files: []string{"profile/wow/retail/character/realm/char/summary.json"},
			want:  archive.FeatureAPIData,
		},
		{
			name:  "addon data",
			files: []string{addonDir + "/inventory.json"},
			entries: []*archive.CharacterEntry{{Addon: &archive.AddonRef{
				Dir: addonDir, DataTypes: []string{"inventory"},
			}}},
			want: archive.FeatureAPIData | archive.FeatureAddonData,
		},
		{
			name:  "reputations",
			files: []string{"profile/wow/retail/character/realm/char/reputations.json"},
			want:  archive.FeatureAPIData | archive.FeatureReputations,
		},
		{
			name:  "currencies",
			files: []string{addonDir + "/currency.json"},
			entries: []*archive.CharacterEntry{{Addon: &archive.AddonRef{
				Dir: addonDir, DataTypes: []string{"currency"},
			}}},
			want: archive.FeatureAPIData | archive.FeatureAddonData | archive.FeatureCurrencies,
		},
		{
			name:  "transmog",
			files: []string{"profile/user/wow/collections-transmogs/account-transmogs.json"},
			want:  archive.FeatureAPIData | archive.FeatureTransmog,
		},
		{
			name:  "decor",
			files: []string{"profile/user/wow/collections-decor/account-decor.json"},
			want:  archive.FeatureAPIData | archive.FeatureDecor,
		},
		{
			name:  "housing",
			files: []string{"profile/wow/retail/character/realm/char/house-1.json"},
			want:  archive.FeatureAPIData | archive.FeatureHousing,
		},
		{
			name:  "guild",
			files: []string{"data/retail/guild/realm/guild/guild.json"},
			want:  archive.FeatureAPIData | archive.FeatureGuild,
		},
		{
			name:  "protected character",
			files: []string{"account/retail/protected-character/123/protected.json"},
			want:  archive.FeatureAPIData | archive.FeatureProtected,
		},
		{
			name:    "simc html",
			files:   []string{simcHTML},
			entries: []*archive.CharacterEntry{{Simc: &archive.SimcRef{HTMLPath: simcHTML}}},
			want:    archive.FeatureSimcHTML,
		},
		{
			name:    "simc json",
			files:   []string{simcJSON},
			entries: []*archive.CharacterEntry{{Simc: &archive.SimcRef{JSONPath: simcJSON}}},
			want:    archive.FeatureSimcJSON,
		},
		{
			name:    "simc text",
			files:   []string{simcText},
			entries: []*archive.CharacterEntry{{Simc: &archive.SimcRef{TextPath: simcText}}},
			want:    archive.FeatureSimcText,
		},
		{
			name:    "gear summary",
			files:   []string{gearSummary},
			entries: []*archive.CharacterEntry{{Gear: &archive.GearRef{SummaryPath: gearSummary}}},
			want:    archive.FeatureGearSummary,
		},
		{
			name:    "healer top gear",
			files:   []string{healerTopGear},
			entries: []*archive.CharacterEntry{{Gear: &archive.GearRef{TopGearTextPath: healerTopGear}}},
			want:    archive.FeatureHealerTopGear,
		},
		{
			name:  "item cache",
			files: []string{"items/retail/items.json"},
			items: &archive.ItemsRef{Filename: "items/retail/items.json", Count: 1},
			want:  archive.FeatureItemCache,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, path := range tc.files {
				writeArchiveTestFile(t, dir, path)
			}

			got := archiveFeatures(dir, tc.entries, tc.items)
			if got != tc.want {
				t.Errorf("archiveFeatures() = %s, want %s", got, tc.want)
			}
		})
	}
}

func writeArchiveTestFile(t *testing.T, dir, rel string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
}
