package archiver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/sorens/warchiver/internal/api"
	"github.com/sorens/warchiver/internal/archive"
	"github.com/sorens/warchiver/internal/data"
	"github.com/sorens/warchiver/internal/heirloom"
	walog "github.com/sorens/warchiver/internal/log"
)

// ProcessAll dispatches queued and discovered endpoints with a fixed worker pool.
func (a *Archiver) ProcessAll(ctx context.Context, client *api.Client, dir string, slug *api.ProfileSlug) error {
	queueEndpoints(ctx, 4, a.endpoints, func(ep *api.Endpoint) []*api.Endpoint {
		if ep.IsIndex && ep.Kind == api.KindCharacter {
			printf(ctx, "  downloading %s/%s", ep.RequestData[0], ep.RequestData[1])
		}
		discovered, err := a.process(ctx, client, ep, dir, slug)
		if err != nil {
			if ep.Optional {
				walog.Ctx(ctx).Verbose("skipping error %q (optional): %v", ep.Filename, err)
			} else {
				errorf(ctx, err, "account visit error")
			}
		}
		return discovered
	}, func(newEp *api.Endpoint) bool {
		key := string(newEp.Game) + ":" + newEp.String()
		if newEp.Kind == api.KindMedia {
			key += ":" + newEp.Url
		}
		a.m.Lock()
		skip := a.seen[key]
		if !skip {
			a.seen[key] = true
		}
		a.m.Unlock()
		return !skip
	})
	a.endpoints = nil

	visited := make([]*api.Endpoint, 0, len(a.visited))
	for _, ep := range a.visited {
		if ep.Kind == api.KindCharacter && !ep.Optional {
			visited = append(visited, ep)
		}
	}
	slices.SortFunc(visited, func(a, b *api.Endpoint) int {
		if a.RequestData[0] != b.RequestData[0] {
			return strings.Compare(a.RequestData[0], b.RequestData[0])
		}
		return strings.Compare(a.RequestData[1], b.RequestData[1])
	})
	return ctx.Err()
}

func queueEndpoints(ctx context.Context, workers int, initial []*api.Endpoint, process func(*api.Endpoint) []*api.Endpoint, accept func(*api.Endpoint) bool) {
	type result struct {
		discovered []*api.Endpoint
	}
	type job struct {
		endpoint *api.Endpoint
	}
	jobs := make(chan job)
	results := make(chan result, workers)
	queue := append([]*api.Endpoint(nil), initial...)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range jobs {
				results <- result{discovered: process(task.endpoint)}
			}
		}()
	}

	active := 0
	for len(queue) > 0 || active > 0 {
		if ctx.Err() != nil {
			queue = nil
		}
		var send chan job
		var next job
		if len(queue) > 0 && ctx.Err() == nil {
			send = jobs
			next = job{endpoint: queue[0]}
		}
		select {
		case send <- next:
			queue = queue[1:]
			active++
		case completed := <-results:
			active--
			for _, newEp := range completed.discovered {
				if accept(newEp) {
					queue = append(queue, newEp)
				}
			}
		case <-ctx.Done():
			queue = nil
		}
	}
	close(jobs)
	wg.Wait()
}

// Account fetches the account profile and race data (no character endpoints queued).
func (a *Archiver) Account(ctx context.Context, client *api.Client, dir string, minlvl int) (*api.ProfileSlug, error) {
	start := api.StartEndpoints[0].Copy()
	slug := api.NewProfileSlug()
	if _, err := a.process(ctx, client, start, dir, slug); err != nil {
		return nil, err
	}
	races := api.DataEndpoints[0].Copy()
	if _, err := a.process(ctx, client, races, dir, slug); err != nil {
		return nil, err
	}
	return slug, nil
}

// AccountWithEndpoints fetches the account profile and race data, then queues all character endpoints.
func (a *Archiver) AccountWithEndpoints(ctx context.Context, client *api.Client, dir string, minlvl int) (*api.ProfileSlug, error) {
	start := api.StartEndpoints[0].Copy()
	slug := api.NewProfileSlug()
	if _, err := a.process(ctx, client, start, dir, slug); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		errorf(ctx, err, "account profile unavailable")
	}

	races := api.DataEndpoints[0].Copy()
	if _, err := a.process(ctx, client, races, dir, slug); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		errorf(ctx, err, "race index unavailable")
	}

	a.endpoints = append(a.endpoints, api.AccountEndpoints...)

	for _, acct := range slug.Accounts {
		for _, char := range acct.Characters {
			if char.Level < 10 {
				printf(ctx, "m %2d skipping %s/%s", char.Level, char.Realm.Slug, strings.ToLower(char.Name))
				continue
			}
			if char.Level < minlvl {
				printf(ctx, "f %2d skipping %s/%s", char.Level, char.Realm.Slug, strings.ToLower(char.Name))
				continue
			}
			protected := api.ProtectedEndpoints[0].Copy()
			protected.RequestData = append(protected.RequestData, fmt.Sprintf("%d", char.Realm.ID), fmt.Sprintf("%d", char.ID))
			a.endpoints = append(a.endpoints, protected)
			for _, ep := range api.CharacterEndpoints {
				c := ep.Copy()
				c.RequestData = append(c.RequestData, char.Realm.Slug, strings.ToLower(char.Name))
				a.endpoints = append(a.endpoints, c)
			}
		}
	}

	SortEndpoints(a.endpoints)

	return slug, nil
}

// Character archives one character; also queues account-level collection endpoints.
func (a *Archiver) Character(ctx context.Context, client *api.Client, dir string, realmName, charName string) (*api.ProfileSlug, error) {
	if realmName == "" || charName == "" {
		return nil, fmt.Errorf("missing required parameters")
	}

	start := api.StartEndpoints[0].Copy()
	slug := api.NewProfileSlug()
	profileAvailable := true
	if _, err := a.process(ctx, client, start, dir, slug); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		errorf(ctx, err, "account profile unavailable")
		profileAvailable = false
	}

	races := api.DataEndpoints[0].Copy()
	if _, err := a.process(ctx, client, races, dir, slug); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		errorf(ctx, err, "race index unavailable")
	}

	a.endpoints = append(a.endpoints, api.AccountEndpoints...)

	found := false
	for _, acct := range slug.Accounts {
		for _, char := range acct.Characters {
			if !strings.EqualFold(realmName, char.Realm.Name) {
				continue
			}
			if !strings.EqualFold(charName, char.Name) {
				continue
			}

			if char.Level < 10 {
				printf(ctx, "m %2d skipping %s/%s", char.Level, char.Realm.Slug, strings.ToLower(char.Name))
				return nil, ErrLevelBelowMinimum
			}

			found = true
			protected := api.ProtectedEndpoints[0].Copy()
			protected.RequestData = append(protected.RequestData, fmt.Sprintf("%d", char.Realm.ID), fmt.Sprintf("%d", char.ID))
			a.endpoints = append(a.endpoints, protected)
			for _, ep := range api.CharacterEndpoints {
				c := ep.Copy()
				c.RequestData = append(c.RequestData, char.Realm.Slug, strings.ToLower(char.Name))
				a.endpoints = append(a.endpoints, c)
			}
			break
		}

		if found {
			break
		}
	}

	if !found {
		if profileAvailable {
			return nil, fmt.Errorf("character not found")
		}
		realmSlug := api.NewSlugFromName(realmName).Slug()
		for _, ep := range api.CharacterEndpoints {
			c := ep.Copy()
			c.RequestData = append(c.RequestData, realmSlug, strings.ToLower(charName))
			a.endpoints = append(a.endpoints, c)
		}
	}

	return slug, nil
}

// responseError surfaces a non-nil error from a Request*-style return.
func responseError(_ *api.EndpointResponse, err error) error {
	return err
}

func (a *Archiver) process(ctx context.Context, client *api.Client, ep *api.Endpoint, dir string, slug *api.ProfileSlug) ([]*api.Endpoint, error) {
	var resp *api.EndpointResponse
	var err error
	switch ep.Kind {
	case api.KindMedia:
		resp, err = client.RequestMedia(ctx, dir, ep)
	default:
		resp, err = client.RequestEndpoint(ctx, dir, ep)
	}

	if e := responseError(resp, err); e != nil {
		if ep.Optional {
			walog.Ctx(ctx).Verbose("optional endpoint %s skipped: %v", ep.String(), e)
			return nil, nil
		}
		if ep.Kind == api.KindCharacter {
			displayName, _ := filepath.Rel(`profile/wow/character/`, ep.String())
			return nil, fmt.Errorf("required character endpoint %s: %w", displayName, e)
		}
		return nil, fmt.Errorf("endpoint request failed: %w", e)
	}

	// Parse outside the lock; the apply step below holds a short critical section.
	var (
		discovered  []*api.Endpoint
		profileData *api.ProfileSlug
		charData    *characterIndexData
	)
	switch {
	case ep.IsIndex && ep.IsProfile:
		profileData, err = parseProfileIndex(resp)
	case ep.IsIndex && ep.Kind == api.KindCharacter:
		charData, err = parseCharacterIndex(client, ep, resp)
	case filepath.Base(resp.ID) == "character-media" && ep.Kind != api.KindMedia:
		discovered, err = parseCharacterMedia(resp)
	case ep.IsIndex && ep.Kind == api.KindData && strings.HasSuffix(ep.Base, "heirloom/index"):
		discovered, err = parseHeirloomIndex(resp)
	}
	if err != nil {
		return nil, err
	}

	a.m.Lock()
	defer a.m.Unlock()
	a.visited = append(a.visited, ep)

	switch {
	case profileData != nil:
		*slug = *profileData
		walog.Ctx(ctx).Debug("auth id: %d, # accounts: %d, # characters: %d",
			profileData.ID, len(profileData.Accounts), len(profileData.Characters()))
	case charData != nil:
		discovered = a.applyCharacterIndex(slug, charData)
	}
	return discovered, nil
}

// characterIndexData carries a parsed character-index between parse and apply phases.
type characterIndexData struct {
	region   string
	realm    string
	char     string
	cs       *data.CharacterSlug
	respPath string
	epPath   string
	epFile   string
}

func parseProfileIndex(resp *api.EndpointResponse) (*api.ProfileSlug, error) {
	b, err := os.ReadFile(resp.Path)
	if err != nil {
		return nil, err
	}
	s, err := api.NewProfileFromData(b)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("profile data nil")
	}
	return s, nil
}

func parseCharacterIndex(client *api.Client, ep *api.Endpoint, resp *api.EndpointResponse) (*characterIndexData, error) {
	check, found := strings.CutPrefix(resp.ID, "/profile/wow/character")
	if !found {
		return nil, nil
	}
	split := strings.Split(check, "/")
	if len(split) != 3 {
		return nil, nil
	}
	r, c := split[1], split[2]

	b, err := os.ReadFile(resp.Path)
	if err != nil {
		return nil, err
	}
	cs := &data.CharacterSlug{}
	if err := json.Unmarshal(b, cs); err != nil {
		return nil, err
	}
	return &characterIndexData{
		region:   string(client.Region()),
		realm:    r,
		char:     c,
		cs:       cs,
		respPath: resp.Path,
		epPath:   ep.String(),
		epFile:   ep.Filename,
	}, nil
}

func parseCharacterMedia(resp *api.EndpointResponse) ([]*api.Endpoint, error) {
	b, err := os.ReadFile(resp.Path)
	if err != nil {
		return nil, err
	}
	media := data.CharacterMedia{}
	if err := json.Unmarshal(b, &media); err != nil {
		return nil, err
	}
	var discovered []*api.Endpoint
	for _, asset := range media.Assets {
		discovered = append(discovered, &api.Endpoint{
			Namespace: "dynamic-%s",
			Kind:      api.KindMedia,
			Url:       asset.Value,
			Path:      resp.ID,
		})
	}
	return discovered, nil
}

func parseHeirloomIndex(resp *api.EndpointResponse) ([]*api.Endpoint, error) {
	b, err := os.ReadFile(resp.Path)
	if err != nil {
		return nil, err
	}
	var idx heirloom.HeirloomIndex
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, err
	}
	discovered := make([]*api.Endpoint, 0, len(idx.Heirlooms))
	for _, ref := range idx.Heirlooms {
		discovered = append(discovered, api.HeirloomDetailEndpoint(ref.ID))
	}
	return discovered, nil
}

// applyCharacterIndex updates slug and a.entries; caller must hold a.m.
func (a *Archiver) applyCharacterIndex(slug *api.ProfileSlug, d *characterIndexData) []*api.Endpoint {
	cs := slug.Find(d.realm, d.char)
	if cs == nil {
		cs = &data.CharacterSlug{}
	}
	*cs = *d.cs

	var discovered []*api.Endpoint
	if cs.Guild != nil {
		rSlug := strings.ToLower(cs.Realm.Slug)
		gSlug := strings.ToLower(strings.ReplaceAll(cs.Guild.Name, " ", "-"))
		for _, ep := range api.GuildEndpoints {
			g := ep.Copy()
			g.RequestData = append(g.RequestData, rSlug, gSlug)
			discovered = append(discovered, g)
		}
	}

	entry := &archive.CharacterEntry{
		Game:     string(a.game),
		Region:   d.region,
		Realm:    cs.Realm.Name,
		Name:     cs.Name,
		Filename: filepath.Join("profile", "wow", string(a.game), "character", d.realm, d.char, d.epFile),
		Level:    cs.Level,
		Class:    cs.PlayableClass.Name,
		RealmID:  cs.Realm.ID,
		CharID:   cs.ID,
	}
	a.entries = append(a.entries, entry)

	return discovered
}
