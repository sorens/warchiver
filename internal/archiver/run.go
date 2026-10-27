package archiver

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sorens/warchiver/internal/api"
	"github.com/sorens/warchiver/internal/archive"
	walog "github.com/sorens/warchiver/internal/log"
	"github.com/sorens/warchiver/pkg/features"
)

// Request describes a single archive run. Realm+Character drive single-character
// mode; both empty selects account-mode. WorkDir is the per-run working
// directory (caller creates it; Run does not clean it up). ArchiveDir is where
// the final .tar.gz is written. ConfigDir, when non-empty, points at the
// directory under which configuration is read.
type Request struct {
	Game      api.GameProperty
	Realm     string
	Character string

	Region      string
	AccessToken string
	BattleTag   string

	WorkDir     string
	ArchiveDir  string
	ConfigDir   string
	WowDir      string
	SIMCBinPath string
	SimcJobs    int
	SimcThreads int
	EnrichItems bool
	Features    features.Provider

	MinLevel int
	Version  string
}

// Run executes a full archive pipeline and returns the path to the resulting
// .tar.gz bundle. It is the single entry point shared by the CLI and the
// hosted service so per-phase wiring stays in one place.
func Run(ctx context.Context, req Request) (string, error) {
	if req.Realm != "" || req.Character != "" {
		return runOne(ctx, req)
	}

	games := []api.GameProperty{api.GameRetail, api.GameClassic, api.GameClassic1x, api.GameClassicAnniversary}
	pipelines := make([]*accountPipeline, 0, len(games))
	for _, game := range games {
		pipeline, err := newAccountPipeline(ctx, req, game)
		if err != nil {
			walog.Ctx(ctx).Error(err, "%s archive, skipping", game)
			continue
		}
		pipelines = append(pipelines, pipeline)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	for _, pipeline := range pipelines {
		if err := pipeline.arch.ProcessAll(ctx, pipeline.client, req.WorkDir, pipeline.slug); err != nil {
			return "", err
		}
	}
	var entries []*archive.CharacterEntry
	for _, pipeline := range pipelines {
		if pipeline.game == api.GameRetail {
			if err := pipeline.arch.HeirloomFiles(ctx, filepath.Join(req.WorkDir, "heirloom"), req.WorkDir); err != nil {
				walog.Ctx(ctx).Debug("heirloom: %v", err)
			}
			break
		}
	}
	for _, pipeline := range pipelines {
		entries = append(entries, pipeline.arch.Entries()...)
	}
	if !downloadedDataExists(req.WorkDir) {
		return "", fmt.Errorf("no data downloaded")
	}
	archiveDir := req.ArchiveDir
	if archiveDir == "" {
		archiveDir = req.WorkDir
	}
	items := runOptionalFeatures(ctx, req, entries, archiveDir)
	return Archive(ctx, req.WorkDir, defaultRegion(req.Region), req.BattleTag, archiveDir, req.Version, entries, items)
}

type accountPipeline struct {
	game   api.GameProperty
	arch   *Archiver
	client *api.Client
	slug   *api.ProfileSlug
}

func newAccountPipeline(ctx context.Context, req Request, game api.GameProperty) (*accountPipeline, error) {
	region := defaultRegion(req.Region)
	arch := NewArchiver(Options{Game: game})
	client, err := api.NewClient(&api.ClientOptions{AccessToken: req.AccessToken, Region: api.Region(strings.ToLower(region)), Game: game})
	if err != nil {
		return nil, err
	}
	slug, err := arch.AccountWithEndpoints(ctx, client, req.WorkDir, req.MinLevel)
	if err != nil {
		return nil, err
	}
	return &accountPipeline{game: game, arch: arch, client: client, slug: slug}, nil
}

func defaultRegion(region string) string {
	if region == "" {
		return "us"
	}
	return region
}

func runOne(ctx context.Context, req Request) (string, error) {
	region := req.Region
	if region == "" {
		region = "us"
	}
	game := req.Game
	if game == "" {
		game = api.GameRetail
	}

	arch := NewArchiver(Options{Game: game})

	client, err := api.NewClient(&api.ClientOptions{
		AccessToken: req.AccessToken,
		Region:      api.Region(strings.ToLower(region)),
		Game:        game,
	})
	if err != nil {
		return "", err
	}

	var slug *api.ProfileSlug
	if req.Realm != "" && req.Character != "" {
		slug, err = arch.Character(ctx, client, req.WorkDir, req.Realm, req.Character)
	} else {
		slug, err = arch.AccountWithEndpoints(ctx, client, req.WorkDir, req.MinLevel)
	}
	if err != nil {
		return "", err
	}

	if err := arch.ProcessAll(ctx, client, req.WorkDir, slug); err != nil {
		return "", err
	}

	heirloomDir := filepath.Join(req.WorkDir, "heirloom")
	if err := arch.HeirloomFiles(ctx, heirloomDir, req.WorkDir); err != nil {
		walog.Ctx(ctx).Debug("heirloom: %v", err)
	}

	archiveDir := req.ArchiveDir
	if archiveDir == "" {
		archiveDir = req.WorkDir
	}
	entries := arch.Entries()
	if !downloadedDataExists(req.WorkDir) {
		return "", fmt.Errorf("no data downloaded")
	}
	items := runOptionalFeatures(ctx, req, entries, archiveDir)
	return Archive(ctx, req.WorkDir, region, req.BattleTag, archiveDir, req.Version, entries, items)
}

func downloadedDataExists(dir string) bool {
	return archiveHasFiles(dir, func(path string) bool {
		if !strings.HasPrefix(path, "profile/") && !strings.HasPrefix(path, "account/") && !strings.HasPrefix(path, "data/") {
			return false
		}
		for _, ext := range []string{".json", ".jsonl", ".jpg", ".jpeg", ".png", ".webp", ".gif"} {
			if strings.HasSuffix(path, ext) {
				return true
			}
		}
		return false
	})
}

func runOptionalFeatures(ctx context.Context, req Request, entries []*archive.CharacterEntry, archiveDir string) *archive.ItemsRef {
	provider := req.Features
	if provider == nil {
		provider = features.Disabled{}
	}
	input := features.Request{
		WorkDir:     req.WorkDir,
		Region:      defaultRegion(req.Region),
		ArchiveDir:  archiveDir,
		ConfigDir:   req.ConfigDir,
		WowDir:      req.WowDir,
		SimcBinPath: req.SIMCBinPath,
		SimcJobs:    req.SimcJobs,
		SimcThreads: req.SimcThreads,
		EnrichItems: req.EnrichItems,
	}
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		input.Characters = append(input.Characters, features.Character{
			Game:      entry.Game,
			Region:    entry.Region,
			Realm:     entry.Realm,
			Name:      entry.Name,
			Directory: filepath.Join(req.WorkDir, filepath.FromSlash(filepath.Dir(entry.Filename))),
		})
	}
	result, err := provider.Run(ctx, input)
	if err != nil {
		walog.Ctx(ctx).Debug("optional features: %v", err)
		return nil
	}
	for _, ref := range result.Characters {
		for _, entry := range entries {
			if entry == nil || !strings.EqualFold(entry.Game, ref.Game) ||
				!strings.EqualFold(entry.Region, ref.Region) ||
				!strings.EqualFold(entry.Realm, ref.Realm) ||
				!strings.EqualFold(entry.Name, ref.Name) {
				continue
			}
			if ref.AddonDir != "" {
				entry.Addon = &archive.AddonRef{
					Dir:            ref.AddonDir,
					SchemaVersion:  ref.AddonSchemaVersion,
					DataTypes:      ref.AddonDataTypes,
					LatestScanTime: ref.AddonLatestScanTime,
				}
			}
			if ref.SimcPath != "" || ref.SimcHTMLPath != "" || ref.SimcJSONPath != "" || ref.SimcTextPath != "" {
				entry.Simc = &archive.SimcRef{
					Path: ref.SimcPath, Checksum: ref.SimcChecksum,
					HTMLPath: ref.SimcHTMLPath, JSONPath: ref.SimcJSONPath, TextPath: ref.SimcTextPath,
				}
			}
			if ref.GearSummaryPath != "" || ref.GearReportPath != "" || ref.HealerTopGearTextPath != "" || ref.HealerTopGearJSONPath != "" {
				entry.Gear = &archive.GearRef{
					SummaryPath: ref.GearSummaryPath, ReportPath: ref.GearReportPath,
					TopGearTextPath: ref.HealerTopGearTextPath, TopGearJSONPath: ref.HealerTopGearJSONPath,
				}
			}
			break
		}
	}
	if result.Items == nil {
		return nil
	}
	return &archive.ItemsRef{Filename: result.Items.Filename, Count: result.Items.Count, Checksum: result.Items.Checksum}
}
