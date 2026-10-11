package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	walog "github.com/sorens/warchiver/internal/log"
)

func runVersion(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	var verbose bool
	fs.BoolVar(&verbose, "verbose", false, "enable verbose output")
	fs.Parse(args)

	level := walog.LevelInfo
	if verbose {
		level = walog.LevelVerbose
	}
	logger := walog.New(os.Stderr, level)
	ctx = logger.WithContext(ctx)

	walog.Ctx(ctx).Verbose("ua: %s", useragent())
	walog.Ctx(ctx).Verbose("go: %s", runtime.Version())

	fmt.Println(useragent())
	fmt.Println("copyright © 2026 Stephen Orens")
	return nil
}
