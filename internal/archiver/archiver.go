package archiver

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/sorens/warchiver/internal/api"
	"github.com/sorens/warchiver/internal/archive"
	walog "github.com/sorens/warchiver/internal/log"
)

var ErrLevelBelowMinimum = fmt.Errorf("level too low")

// Options configure per-run behavior of an Archiver.
type Options struct {
	Game api.GameProperty
}

type Archiver struct {
	opts      Options
	endpoints []*api.Endpoint
	visited   []*api.Endpoint
	entries   []*archive.CharacterEntry
	seen      map[string]bool
	game      api.GameProperty
	m         sync.RWMutex
}

func NewArchiver(opts Options) *Archiver {
	return &Archiver{
		opts:      opts,
		endpoints: []*api.Endpoint{},
		visited:   []*api.Endpoint{},
		entries:   []*archive.CharacterEntry{},
		seen:      map[string]bool{},
		game:      opts.Game,
	}
}

// Entries returns a snapshot of the per-character entries collected during the
// archive run. The slice header is freshly allocated so callers iterating it
// after ProcessAll cannot race against writers; the *CharacterEntry pointers
// are shared, so concurrent mutation of an individual entry is still the
// caller's responsibility.
func (a *Archiver) Entries() []*archive.CharacterEntry {
	return a.entriesSnapshot()
}

func (a *Archiver) entriesSnapshot() []*archive.CharacterEntry {
	a.m.RLock()
	defer a.m.RUnlock()
	out := make([]*archive.CharacterEntry, len(a.entries))
	copy(out, a.entries)
	return out
}

// appendEntryLocked appends e to a.entries under a.m write lock.
func (a *Archiver) appendEntryLocked(e *archive.CharacterEntry) {
	a.m.Lock()
	defer a.m.Unlock()
	a.entries = append(a.entries, e)
}

func SortEndpoints(endpoints []*api.Endpoint) {
	sort.SliceStable(endpoints, func(i, j int) bool {
		a, b := endpoints[i], endpoints[j]

		kindOrder := func(kind api.EndpointKind) int {
			switch kind {
			case api.KindAccount:
				return 0
			case api.KindCharacter:
				return 1
			case api.KindProtected:
				return 2
			case api.KindGuild:
				return 3
			case api.KindData:
				return 4
			case api.KindMedia:
				return 5
			default:
				return 6
			}
		}

		if kindOrder(a.Kind) != kindOrder(b.Kind) {
			return kindOrder(a.Kind) < kindOrder(b.Kind)
		}

		getRequestData := func(e *api.Endpoint, index int) string {
			if len(e.RequestData) > index {
				return e.RequestData[index]
			}
			return ""
		}

		aData0, bData0 := getRequestData(a, 0), getRequestData(b, 0)
		if aData0 != bData0 {
			return aData0 < bData0
		}

		aData1, bData1 := getRequestData(a, 1), getRequestData(b, 1)
		if aData1 != bData1 {
			return aData1 < bData1
		}

		if a.Optional != b.Optional {
			return !a.Optional
		}

		return false
	})
}

func printf(ctx context.Context, format string, args ...any) {
	walog.Ctx(ctx).Info(format, args...)
}

func errorf(ctx context.Context, err error, format string, args ...any) {
	walog.Ctx(ctx).Error(err, format, args...)
}
