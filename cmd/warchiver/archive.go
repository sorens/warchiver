package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sorens/warchiver"
	"github.com/sorens/warchiver/internal/config"
	walog "github.com/sorens/warchiver/internal/log"
)

func runArchive(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("archive", flag.ExitOnError)
	fs.Usage = func() {
		FlagSetUsage(fs)
		fmt.Fprintf(fs.Output(), "subcommands:\n")
		fmt.Fprintf(fs.Output(), "  account, acct, a\n    archive all account data and all characters on all realms\n")
		fmt.Fprintf(fs.Output(), "  character, char, c\n    archive single realm/character\n")
	}
	fs.Parse(args)

	if len(args) == 0 {
		fs.Usage()
		return nil
	}

	switch args[0] {
	case "account", "acct", "a":
		return runArchiveAccount(ctx, args[1:])
	case "character", "char", "c":
		return runArchiveCharacter(ctx, args[1:])
	default:
		fs.Usage()
		return nil
	}
}

func runArchiveAccount(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("account", flag.ExitOnError)
	fs.Usage = func() { FlagSetUsage(fs) }
	var configPath, region, game string
	var minlvl int
	var verbose bool
	fs.StringVar(&configPath, "config", DefaultConfigDir, "path to config file")
	fs.IntVar(&minlvl, "minlvl", 1, "minimum level of character to archive")
	fs.StringVar(&region, "region", "us", "Battle.net region [us|eu|kr|tw|cn]")
	fs.StringVar(&game, "game", "retail", "WoW game [retail|classic|classic1x|classicann]")
	fs.BoolVar(&verbose, "verbose", false, "enable verbose output")
	fs.Parse(args)

	ctx, req, err := buildArchiveRequest(ctx, configPath, region, game, verbose)
	if err != nil {
		return err
	}
	req.MinLevel = minlvl

	_, err = warchiver.Archive(ctx, req)
	return err
}

func runArchiveCharacter(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("character", flag.ExitOnError)
	fs.Usage = func() { FlagSetUsage(fs) }
	var configPath, region, game string
	var verbose bool
	fs.StringVar(&configPath, "config", DefaultConfigDir, "path to config file")
	fs.StringVar(&region, "region", "us", "Battle.net region [us|eu|kr|tw|cn]")
	fs.StringVar(&game, "game", "retail", "WoW game [retail|classic|classic1x|classicann]")
	fs.BoolVar(&verbose, "verbose", false, "enable verbose output")
	fs.Parse(args)

	if fs.NArg() < 2 {
		return fmt.Errorf("missing required parameter")
	}
	realm, char := fs.Arg(0), fs.Arg(1)
	if realm == "" || char == "" {
		return fmt.Errorf("missing required parameter")
	}

	ctx, req, err := buildArchiveRequest(ctx, configPath, region, game, verbose)
	if err != nil {
		return err
	}
	req.Realm = realm
	req.Character = char

	_, err = warchiver.Archive(ctx, req)
	return err
}

// buildArchiveRequest installs the verbose-gated logger on ctx, loads config,
// refreshes the token, and creates the per-run working directory. It returns
// the augmented ctx alongside a partially-filled Request — callers set
// Realm/Character/MinLevel afterwards.
func buildArchiveRequest(ctx context.Context, configPath, region, game string, verbose bool) (context.Context, warchiver.ArchiveRequest, error) {
	level := walog.LevelInfo
	if verbose {
		level = walog.LevelVerbose
	}
	ctx = walog.New(os.Stderr, level).WithContext(ctx)

	cfg, cfgPath, err := loadOrCreateConfig(configPath)
	if err != nil {
		return ctx, warchiver.ArchiveRequest{}, err
	}
	if err := requireToken(ctx, cfg, cfgPath); err != nil {
		return ctx, warchiver.ArchiveRequest{}, err
	}

	workDir, err := makeWorkDir(ctx, cfg)
	if err != nil {
		return ctx, warchiver.ArchiveRequest{}, err
	}

	effectiveRegion := region
	if effectiveRegion == "" {
		effectiveRegion = cfg.Region
	}
	if effectiveRegion == "" {
		effectiveRegion = "us"
	}

	archiveDir := cfg.ArchiveDir
	if archiveDir == "" {
		archiveDir, _ = config.ArchiveDir()
	}

	return ctx, warchiver.ArchiveRequest{
		Game:        warchiver.Game(strings.ToLower(game)),
		Region:      effectiveRegion,
		AccessToken: cfg.AccessToken,
		BattleTag:   cfg.BattleTag,
		WorkDir:     workDir,
		ArchiveDir:  archiveDir,
		ConfigDir:   filepath.Dir(cfgPath),
		WowDir:      os.Getenv("WARCHIVER_WOW_DIR"),
		SIMCBinPath: os.Getenv("SIMC"),
		Features:    archiveFeatureProvider(),
		Version:     useragent(),
	}, nil
}

func makeWorkDir(ctx context.Context, cfg *config.Config) (string, error) {
	base := os.TempDir()
	if v, ok := ctx.Value(ctxKeyDebugDir).(string); ok && v != "" {
		base = v
	}
	tmp, err := os.MkdirTemp(base, "wa.debug.*")
	if err != nil {
		return "", err
	}
	dir := filepath.Join(tmp, fmt.Sprintf("%s.%s", cfg.BattleTag, cfg.AccountID))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	walog.Ctx(ctx).Debug("archive working dir: %s", dir)
	return dir, nil
}
