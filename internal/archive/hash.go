package archive

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// HashManifest computes a short 8-char hex hash of the manifest content.
// Used in the warchive filename for uniqueness.
// removes the timestamp
func HashManifest(m *Manifest) (string, error) {
	hashableManifest := m.Copy()
	hashableManifest.CreatedAt = time.Time{}
	// Item-cache metadata is operational state: entries gain last_seen times
	// and new API variants may be discovered during any run. It must not make
	// the user-data archive hash unstable.
	hashableManifest.Items = nil
	for _, c := range hashableManifest.Characters {
		c.Simc = nil
		c.Gear = nil
	}
	data, err := json.Marshal(hashableManifest)
	if err != nil {
		return "", fmt.Errorf("marshaling manifest for hash: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:8], nil
}

// HashProfileTree returns a stable hash of all regular files below dir.
func HashProfileTree(dir string) (string, error) {
	type item struct {
		name string
		data []byte
	}
	var items []item
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.HasSuffix(p, ".json") {
			var v any
			if json.Unmarshal(b, &v) == nil {
				b, _ = json.Marshal(v)
			}
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		items = append(items, item{filepath.ToSlash(rel), b})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	h := sha256.New()
	for _, it := range items {
		fmt.Fprintf(h, "%s\x00%d\x00", it.name, len(it.data))
		_, _ = h.Write(it.data)
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}
