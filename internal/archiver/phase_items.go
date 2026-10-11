package archiver

import (
	"context"
	"github.com/sorens/warchiver/internal/heirloom"
)

// HeirloomFiles calculates the retail heirloom report from downloaded API data.
func (a *Archiver) HeirloomFiles(ctx context.Context, outDir, dir string) error {
	return (&heirloom.Archiver{DataDir: dir}).Run(ctx, outDir)
}
