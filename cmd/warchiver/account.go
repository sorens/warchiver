package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sorens/warchiver"
	walog "github.com/sorens/warchiver/internal/log"
)

func runAccount(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("account", flag.ExitOnError)
	fs.Usage = func() {
		FlagSetUsage(fs)
		fmt.Fprintf(fs.Output(), "subcommands:\n")
		fmt.Fprintf(fs.Output(), "  list, ls\n    list characters for specified account\n")
	}
	fs.Parse(args)

	// check for subcommands
	if len(args) == 0 {
		fs.Usage()
		return nil
	}

	switch args[0] {
	case "list", "ls":
		return runAccountList(ctx, args[1:])
	default:
		fs.Usage()
		return nil
	}
}

type accountListCharacter struct {
	game warchiver.Game
	char warchiver.Character
}

type accountListSlug struct {
	game warchiver.Game
	slug *warchiver.AccountProfile
}

func runAccountList(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	fs.Usage = func() {
		FlagSetUsage(fs)
	}
	var configPath string
	var region string
	var verbose bool
	var minlvl int
	fs.StringVar(&configPath, "config", DefaultConfigDir, "path to config file")
	fs.StringVar(&region, "region", "", "Battle.net region [us|eu|kr|tw|cn]")
	fs.IntVar(&minlvl, "minlvl", 1, "minimum level of character to list")
	fs.BoolVar(&verbose, "verbose", false, "enable verbose output")
	fs.Parse(args)

	level := walog.LevelInfo
	if verbose {
		level = walog.LevelVerbose
	}
	logger := walog.New(os.Stderr, level)
	ctx = logger.WithContext(ctx)

	cfg, cfgPath, err := loadOrCreateConfig(configPath)
	if err != nil {
		return err
	}
	if err := requireToken(ctx, cfg, cfgPath); err != nil {
		return err
	}

	dirFlag := filepath.Join(os.TempDir(), fmt.Sprintf("%s.%s", cfg.BattleTag, cfg.AccountID))

	// Determine effective region
	effectiveRegion := region
	if effectiveRegion == "" {
		effectiveRegion = cfg.Region
	}
	if effectiveRegion == "" {
		effectiveRegion = "us"
	}

	var slugs []accountListSlug
	games := []warchiver.Game{warchiver.GameRetail, warchiver.GameClassic, warchiver.GameClassic1x, warchiver.GameClassicAnniversary}
	for _, gameProperty := range games {
		slug, clientErr := warchiver.Account(ctx, warchiver.AccountRequest{
			AccessToken: cfg.AccessToken,
			Region:      strings.ToLower(effectiveRegion),
			Game:        gameProperty,
			WorkDir:     dirFlag,
			MinLevel:    minlvl,
		})
		if clientErr != nil {
			continue
		}
		if slug != nil {
			slugs = append(slugs, accountListSlug{game: gameProperty, slug: slug})
		}
	}

	var total int
	var chars []accountListCharacter
	for _, resultSlug := range slugs {
		game := resultSlug.game
		for _, a := range resultSlug.slug.Accounts {
			for _, c := range a.Characters {
				total++
				if c.Level < minlvl {
					continue
				}
				c.ProfileID = resultSlug.slug.ID
				c.AccountID = a.ID

				chars = append(chars, accountListCharacter{game: game, char: c})
			}
		}
	}

	// sort by level, realm then name
	sort.Slice(chars, func(i, j int) bool {
		li := chars[i].char.Level
		lj := chars[j].char.Level
		if li != lj {
			return li > lj
		}
		ri := strings.ToLower(chars[i].char.Realm.Name)
		rj := strings.ToLower(chars[j].char.Realm.Name)
		if ri != rj {
			return ri < rj
		}
		return strings.ToLower(chars[i].char.Name) < strings.ToLower(chars[j].char.Name)
	})

	if len(chars) == 0 {
		fmt.Println("No characters found.")
		return nil
	}

	rgnWidth := len("Region")
	gameWidth := len("Game")
	realmWidth := len("Realm")
	nameWidth := len("Name")
	classWidth := len("Class")

	if len(effectiveRegion) > rgnWidth {
		rgnWidth = len(effectiveRegion)
	}

	for _, row := range chars {
		c := row.char
		if len(string(row.game)) > gameWidth {
			gameWidth = len(string(row.game))
		}
		if len(c.Realm.Name) > realmWidth {
			realmWidth = len(c.Realm.Name)
		}
		if len(c.Name) > nameWidth {
			nameWidth = len(c.Name)
		}
		if len(c.PlayableClass.Name) > classWidth {
			classWidth = len(c.PlayableClass.Name)
		}
	}

	// header
	fmtStr := fmt.Sprintf("%%-8s  %%-8s  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%5s  %%-%ds  %%-8s\n",
		gameWidth, rgnWidth, realmWidth, nameWidth, classWidth)
	fmt.Printf(fmtStr, "Blizz ID", "WOW ID", "Game", "Region", "Realm", "Name", "Level", "Class", "Faction")
	fmt.Printf(fmtStr,
		"--------",
		"--------",
		strings.Repeat("-", gameWidth),
		strings.Repeat("-", rgnWidth),
		strings.Repeat("-", realmWidth),
		strings.Repeat("-", nameWidth),
		"-----",
		strings.Repeat("-", classWidth),
		"--------",
	)

	for _, row := range chars {
		c := row.char
		fmt.Printf(fmtStr,
			strconv.Itoa(c.ProfileID),
			strconv.Itoa(c.AccountID),
			row.game,
			strings.ToUpper(effectiveRegion),
			c.Realm.Name,
			c.Name,
			fmt.Sprintf("%d", c.Level),
			c.PlayableClass.Name,
			c.Faction.Name,
		)
	}

	fmt.Println()
	fmt.Printf("%3d characters found\n", total)
	fmt.Printf("%3d characters displayed", len(chars))

	fmt.Println()
	return nil
}
