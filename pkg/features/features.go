// Package features defines the extension boundary for optional local archive
// features. The public CLI does not install a provider; a local executable may
// supply one from a separate module.
package features

import "context"

// Character identifies one API-collected character and its profile directory.
type Character struct {
	Game      string
	Region    string
	Realm     string
	Name      string
	Directory string
}

// Request gives a provider the archive inputs and API-collected character data.
type Request struct {
	WorkDir     string
	ArchiveDir  string
	ConfigDir   string
	WowDir      string
	SimcBinPath string
	SimcJobs    int
	SimcThreads int
	EnrichItems bool
	Characters  []Character
}

// CharacterArtifacts associates optional feature outputs with one character.
type CharacterArtifacts struct {
	Game, Region, Realm, Name string

	AddonDir            string
	AddonSchemaVersion  int
	AddonDataTypes      []string
	AddonLatestScanTime string

	SimcPath, SimcChecksum, SimcHTMLPath, SimcJSONPath, SimcTextPath string

	GearSummaryPath, GearReportPath              string
	HealerTopGearTextPath, HealerTopGearJSONPath string
}

// ItemSnapshot describes an optional item-cache snapshot bundled in the archive.
type ItemSnapshot struct {
	Filename string
	Count    int
	Checksum string
}

// Result reports optional artifacts created by a provider.
type Result struct {
	Characters []CharacterArtifacts
	Items      *ItemSnapshot
}

// Provider runs optional local archive features and returns their manifest refs.
type Provider interface {
	Run(context.Context, Request) (Result, error)
}

// Disabled is the default provider. It leaves the API archive unchanged.
type Disabled struct{}

func (Disabled) Run(context.Context, Request) (Result, error) {
	return Result{}, nil
}
