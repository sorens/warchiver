package heirloom

import (
	"context"
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	walog "github.com/sorens/warchiver/internal/log"
)

//go:embed data/heirlooms_index.json
var heirloomIndexData []byte

// Archiver reads downloaded API data and writes heirloom CSV files.
type Archiver struct {
	DataDir string // root of the downloaded archive working directory
}

// Run calculates heirloom costs and writes CSV files to outDir.
func (a *Archiver) Run(ctx context.Context, outDir string) error {
	index, err := a.readIndex(ctx)
	if err != nil {
		return err
	}

	owned, err := a.readOwned(ctx)
	if err != nil {
		return err
	}
	if owned == nil {
		walog.Ctx(ctx).Debug("heirloom: ownership data unavailable, skipping report")
		return os.RemoveAll(outDir)
	}

	details := a.readDetails(ctx, index)
	goldCopper := a.sumGold(ctx)

	report := Calculate(index, details, owned, goldCopper, nil)

	walog.Ctx(ctx).Debug("heirloom: %d missing, %d owned-below-max, %d maxed",
		len(report.Missing), len(report.OwnedBelow), len(report.OwnedMax))

	return Write(outDir, report)
}

func (a *Archiver) readIndex(ctx context.Context) (*HeirloomIndex, error) {
	var idx HeirloomIndex
	if err := json.Unmarshal(heirloomIndexData, &idx); err != nil {
		return nil, err
	}
	walog.Ctx(ctx).Debug("heirloom: index loaded (%d items)", len(idx.Heirlooms))
	return &idx, nil
}

func (a *Archiver) readOwned(ctx context.Context) (map[int]int, error) {
	path := filepath.Join(a.DataDir, "account", "retail", "collections", "heirlooms", "account-heirlooms.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			walog.Ctx(ctx).Debug("heirloom: account-heirlooms.json not found")
			return nil, nil
		}
		return nil, err
	}
	var col AccountHeirlooms
	if err := json.Unmarshal(b, &col); err != nil {
		return nil, err
	}
	owned := make(map[int]int, len(col.Heirlooms))
	for _, h := range col.Heirlooms {
		owned[h.Heirloom.ID] = h.Upgrade.Level
	}
	walog.Ctx(ctx).Debug("heirloom: %d owned heirlooms", len(owned))
	return owned, nil
}

func (a *Archiver) readDetails(ctx context.Context, index *HeirloomIndex) map[int]*HeirloomDetail {
	details := make(map[int]*HeirloomDetail, len(index.Heirlooms))
	for _, ref := range index.Heirlooms {
		path := filepath.Join(a.DataDir, "data", "retail", "heirloom", strconv.Itoa(ref.ID), "heirloom.json")
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var d HeirloomDetail
		if err := json.Unmarshal(b, &d); err != nil {
			walog.Ctx(ctx).Debug("heirloom: failed to parse detail %d: %v", ref.ID, err)
			continue
		}
		details[ref.ID] = &d
	}
	walog.Ctx(ctx).Debug("heirloom: %d detail files loaded", len(details))
	return details
}

// sumGold walks protected-character directories and sums the money field (in copper).
// Returns -1 if no protected files are found.
func (a *Archiver) sumGold(ctx context.Context) int64 {
	protDir := filepath.Join(a.DataDir, "account", "retail", "protected-character")
	entries, err := os.ReadDir(protDir)
	if err != nil {
		walog.Ctx(ctx).Debug("heirloom: protected-character dir not found: %v", err)
		return -1
	}

	var total int64
	found := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(protDir, entry.Name(), "protected.json")
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var p ProtectedCharacter
		if err := json.Unmarshal(b, &p); err != nil {
			continue
		}
		total += p.Money
		found = true
	}

	if !found {
		return -1
	}
	walog.Ctx(ctx).Debug("heirloom: total gold across characters: %dg", total/10000)
	return total
}
