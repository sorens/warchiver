package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestReformatJsonToBufferRejectsIncompleteOrTrailingData(t *testing.T) {
	for _, input := range []string{
		`{"character":`,
		`{"character":"A"}{"extra":true}`,
		`{"character":"A"} trailing`,
	} {
		if _, err := ReformatJsonToBuffer(context.Background(), bytes.NewBufferString(input)); err == nil {
			t.Errorf("ReformatJsonToBuffer(%q) succeeded", input)
		}
	}
}

func TestReformatJsonToBufferAcceptsTrailingWhitespace(t *testing.T) {
	got, err := ReformatJsonToBuffer(context.Background(), bytes.NewBufferString("{\"character\":\"A\"} \n\t"))
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "{\n  \"character\": \"A\"\n}\n" {
		t.Fatalf("unexpected reformatted JSON: %q", got.String())
	}
}

func TestTarGzipPreservesOriginalInvalidJSON(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	original := []byte(`{"character":`)
	if err := os.WriteFile(filepath.Join(src, "character.json"), original, 0644); err != nil {
		t.Fatal(err)
	}

	path, err := TarGzip(context.Background(), src, dst, &Manifest{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Name != "character.json" {
			continue
		}
		got, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, original) {
			t.Fatalf("archived contents = %q, want original %q", got, original)
		}
		return
	}
	t.Fatal("character.json missing from archive")
}
