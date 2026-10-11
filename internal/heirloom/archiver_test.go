package heirloom

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunSkipsWhenOwnershipDataIsMissing(t *testing.T) {
	root, out := t.TempDir(), filepath.Join(t.TempDir(), "heirloom")
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "stale.csv"), []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := (&Archiver{DataDir: root}).Run(context.Background(), out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("output directory still exists: err=%v", err)
	}
}

func TestWriteReportsCSVFlushFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink("/dev/full", filepath.Join(dir, "summary.csv")); err != nil {
		t.Skipf("cannot create /dev/full symlink: %v", err)
	}
	err := Write(dir, &Report{})
	if err == nil {
		t.Fatal("Write() succeeded with a CSV output that cannot be flushed")
	}
}
