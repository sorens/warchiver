package archiver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sorens/warchiver/internal/api"
	"github.com/sorens/warchiver/internal/archive"
)

// Archive writes manifest.json into dir and creates a .tar.gz warchive bundle in archiveDir.
// Returns the path of the created archive file. items, when non-nil, is stamped
// into the manifest so the items.json snapshot in dir is discoverable from the
// manifest.
func Archive(ctx context.Context, dir, region, battleTag, archiveDir, version string, entries []*archive.CharacterEntry, items *archive.ItemsRef) (string, error) {
	// Copy before sort so callers holding the original slice don't see in-place reordering.
	sorted := make([]*archive.CharacterEntry, len(entries))
	copy(sorted, entries)
	slices.SortFunc(sorted, func(a, b *archive.CharacterEntry) int {
		if a.Realm != b.Realm {
			return strings.Compare(a.Realm, b.Realm)
		}
		if a.Game != b.Game {
			return strings.Compare(a.Game, b.Game)
		}
		return strings.Compare(a.Name, b.Name)
	})
	entries = sorted
	for _, e := range entries {
		if e == nil || e.Filename == "" {
			continue
		}
		if h, err := archive.HashProfileTree(filepath.Dir(filepath.Join(dir, e.Filename))); err == nil {
			e.CharHash = h
		}
	}

	features := archiveFeatures(dir, entries, items)
	source := "api"
	if features.Has(archive.FeatureAddonData) {
		source = "both"
	}

	contexts := manifestContexts(entries, region)
	manifest := &archive.Manifest{
		SchemaVersion: 2,
		Version:       version,
		CreatedAt:     time.Now(),
		Account:       battleTag,
		Regions:       []string{region},
		Features:      features,
		Source:        source,
		Characters:    entries,
		Contexts:      contexts,
		Items:         items,
	}

	tmp, err := os.CreateTemp(dir, ".wa.manifest.*.json")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())

	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return "", err
	}
	tmp.Close()
	if err := os.Rename(tmp.Name(), filepath.Join(dir, "manifest.json")); err != nil {
		return "", fmt.Errorf("failed to replace file: %w", err)
	}

	for _, entry := range entries {
		printf(ctx, "    archiving %s/%s", strings.ToLower(entry.Realm), strings.ToLower(entry.Name))
	}

	path, err := archive.TarGzip(ctx, dir, archiveDir, manifest, entries, ".log", ".csv", ".txt", ".jpg", ".jpeg", ".png", ".webp", ".gif")
	if err != nil {
		return "", err
	}

	printf(ctx, "%3d characters archived", len(entries))
	printf(ctx, "archive: %s", path)
	return path, nil
}

func archiveFeatures(dir string, entries []*archive.CharacterEntry, items *archive.ItemsRef) archive.Feature {
	features := archive.Feature(0)
	if archiveHasFiles(dir, func(path string) bool {
		return strings.HasPrefix(path, "profile/") || strings.HasPrefix(path, "data/") || strings.HasPrefix(path, "account/")
	}) {
		features.Set(archive.FeatureAPIData)
	}
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if entry.Addon != nil && archiveHasFiles(dir, func(path string) bool {
			return strings.HasPrefix(path, filepath.ToSlash(entry.Addon.Dir)+"/") && filepath.Base(path) != "meta.json"
		}) {
			features.Set(archive.FeatureAddonData)
			for _, dataType := range entry.Addon.DataTypes {
				switch dataType {
				case "currency":
					features.Set(archive.FeatureCurrencies)
				case "reputations":
					features.Set(archive.FeatureReputations)
				case "guildBank":
					features.Set(archive.FeatureGuild)
				}
			}
		}
		if entry.Simc != nil {
			if entry.Simc.HTMLPath != "" && archiveFileExists(dir, entry.Simc.HTMLPath) {
				features.Set(archive.FeatureSimcHTML)
			}
			if entry.Simc.JSONPath != "" && archiveFileExists(dir, entry.Simc.JSONPath) {
				features.Set(archive.FeatureSimcJSON)
			}
			if entry.Simc.TextPath != "" && archiveFileExists(dir, entry.Simc.TextPath) {
				features.Set(archive.FeatureSimcText)
			}
		}
		if entry.Gear != nil {
			if entry.Gear.SummaryPath != "" && archiveFileExists(dir, entry.Gear.SummaryPath) {
				features.Set(archive.FeatureGearSummary)
			}
			if entry.Gear.TopGearTextPath != "" && archiveFileExists(dir, entry.Gear.TopGearTextPath) {
				features.Set(archive.FeatureHealerTopGear)
			}
		}
	}
	if archiveHasFiles(dir, func(path string) bool {
		base := filepath.Base(path)
		return base == "reputations.json" || base == "account-reputations.json"
	}) {
		features.Set(archive.FeatureReputations)
	}
	if archiveHasFiles(dir, func(path string) bool {
		base := filepath.Base(path)
		return base == "collections-transmogs.json" || base == "account-transmogs.json"
	}) {
		features.Set(archive.FeatureTransmog)
	}
	if archiveHasFiles(dir, func(path string) bool {
		base := filepath.Base(path)
		return base == "collections-decor.json" || base == "account-decor.json"
	}) {
		features.Set(archive.FeatureDecor)
	}
	if archiveHasFiles(dir, func(path string) bool {
		return filepath.Base(path) == "house-1.json"
	}) {
		features.Set(archive.FeatureHousing)
	}
	if archiveHasFiles(dir, func(path string) bool {
		return strings.HasPrefix(path, "data/") && strings.Contains(path, "/guild/")
	}) {
		features.Set(archive.FeatureGuild)
	}
	if archiveHasFiles(dir, func(path string) bool {
		return strings.Contains(path, "/protected-character/") && filepath.Base(path) == "protected.json"
	}) {
		features.Set(archive.FeatureProtected)
	}
	if items != nil && items.Count > 0 && archiveFileExists(dir, items.Filename) {
		features.Set(archive.FeatureItemCache)
	}
	return features
}

func archiveFileExists(dir, relPath string) bool {
	if relPath == "" || filepath.IsAbs(relPath) {
		return false
	}
	path := filepath.Join(dir, filepath.FromSlash(relPath))
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func archiveHasFiles(dir string, match func(string) bool) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err == nil && match(filepath.ToSlash(rel)) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func manifestContexts(entries []*archive.CharacterEntry, region string) []archive.Context {
	contexts := []archive.Context{}
	seenGames := map[string]bool{}
	for _, e := range entries {
		if e == nil || seenGames[e.Game] {
			continue
		}
		seenGames[e.Game] = true
		game := e.Game
		if game == "" {
			game = string(api.GameRetail)
		}
		namespace := "profile-" + region
		if game != string(api.GameRetail) {
			namespace = "profile-" + game + "-" + region
		}
		contexts = append(contexts, archive.Context{Game: game, Region: region, Namespace: namespace, Status: "complete"})
	}
	return contexts
}
