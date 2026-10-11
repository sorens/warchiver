package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTarGzipFallsBackToMalformedJSONAndIncludesMedia(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	malformed := []byte(`{"valid":true} {"trailing":true}`)
	if err := os.WriteFile(filepath.Join(src, "bad.json"), malformed, 0600); err != nil {
		t.Fatal(err)
	}
	media := []byte("image bytes")
	if err := os.WriteFile(filepath.Join(src, "portrait.jpg"), media, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "skip.bin"), []byte("skip"), 0600); err != nil {
		t.Fatal(err)
	}

	path, err := TarGzip(context.Background(), src, dst, &Manifest{}, nil, ".jpg")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	seenJSON, seenMedia, seenSkip := false, false, false
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		switch h.Name {
		case "bad.json":
			seenJSON = string(b) == string(malformed)
		case "portrait.jpg":
			seenMedia = string(b) == string(media)
		case "skip.bin":
			seenSkip = true
		}
	}
	if !seenJSON || !seenMedia || seenSkip {
		t.Fatalf("bad JSON preserved=%v, media included=%v, unselected file included=%v", seenJSON, seenMedia, seenSkip)
	}
}

func TestTarGzipRejectsCanceledFinalizationAndRemovesOutput(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "profile.json"), []byte(`{"ok":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	path, err := TarGzip(ctx, src, dst, &Manifest{}, nil)
	if err == nil || path != "" {
		t.Fatalf("TarGzip() = (%q, %v), want error and empty path", path, err)
	}
	entries, err := os.ReadDir(dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("destination contains partial output: %s", strings.Join([]string{entries[0].Name()}, ","))
	}
}

func TestTarGzipFailedRetryPreservesExistingArchive(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "profile.json"), []byte(`{"ok":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	manifest := &Manifest{Version: "retry-test"}

	for attempt := 0; attempt < 5; attempt++ {
		first, err := TarGzip(context.Background(), src, dst, manifest, nil)
		if err != nil {
			t.Fatal(err)
		}
		stamp := time.Now().UTC().Format("20060102T150405Z")
		if !strings.Contains(first, stamp) {
			continue
		}
		original, err := os.ReadFile(first)
		if err != nil {
			t.Fatal(err)
		}
		_, err = TarGzip(context.Background(), filepath.Join(src, "missing"), dst, manifest, nil)
		if err == nil {
			t.Fatal("expected retry to fail on missing source")
		}
		if time.Now().UTC().Format("20060102T150405Z") != stamp {
			continue
		}
		retained, err := os.ReadFile(first)
		if err != nil {
			t.Fatalf("failed retry removed successful archive: %v", err)
		}
		if !bytes.Equal(retained, original) {
			t.Fatal("failed retry changed successful archive")
		}
		return
	}
	t.Fatal("could not run both archive attempts within the same second")
}
