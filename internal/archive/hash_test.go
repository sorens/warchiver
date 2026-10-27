package archive

import (
	"testing"
	"time"
)

func TestHashManifestIgnoresBuildMetadata(t *testing.T) {
	first := &Manifest{
		SchemaVersion: 2,
		Version:       "warchiver/2.0.0 (fdb17c5/2026-10-27T14:20:46Z)",
		CreatedAt:     time.Date(2026, 10, 27, 14, 21, 17, 0, time.UTC),
		Source:        "both",
		Characters:    []*CharacterEntry{{Game: "retail", Name: "Hero", CharHash: "same-player-data"}},
	}
	second := first.Copy()
	second.Version = "warchiver/2.0.0 (fdb17c5/2026-10-27T14:36:57Z)"
	second.CreatedAt = first.CreatedAt.Add(time.Hour)
	a, err := HashManifest(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := HashManifest(second)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("unchanged player data hashed differently after rebuild: %s != %s", a, b)
	}
	if first.Version == "" || second.Version == "" || first.CreatedAt.IsZero() || second.CreatedAt.IsZero() {
		t.Fatal("hashing mutated manifest provenance")
	}
	second.Characters[0].CharHash = "changed-player-data"
	b, err = HashManifest(second)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("changed player data did not change the hash")
	}
	second.Characters[0].CharHash = first.Characters[0].CharHash
	second.Source = "api"
	b, err = HashManifest(second)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("different data source did not change the hash")
	}
}
