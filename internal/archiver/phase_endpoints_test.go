package archiver

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sorens/warchiver/internal/api"
)

func TestQueueEndpointsBoundsWorkers(t *testing.T) {
	initial := make([]*api.Endpoint, 100)
	for i := range initial {
		initial[i] = &api.Endpoint{Filename: fmt.Sprintf("%d", i)}
	}

	var active, maximum, processed atomic.Int32
	queueEndpoints(context.Background(), 4, initial, func(*api.Endpoint) []*api.Endpoint {
		current := active.Add(1)
		for old := maximum.Load(); current > old && !maximum.CompareAndSwap(old, current); old = maximum.Load() {
		}
		time.Sleep(time.Millisecond)
		active.Add(-1)
		processed.Add(1)
		return nil
	}, func(*api.Endpoint) bool { return true })

	if got := maximum.Load(); got != 4 {
		t.Fatalf("maximum concurrent workers = %d, want 4", got)
	}
	if got := processed.Load(); got != int32(len(initial)) {
		t.Fatalf("processed endpoints = %d, want %d", got, len(initial))
	}
}

func TestQueueEndpointsProcessesDiscoveredAndDeduplicates(t *testing.T) {
	root := &api.Endpoint{Filename: "root"}
	child := &api.Endpoint{Filename: "child"}
	grandchild := &api.Endpoint{Filename: "grandchild"}
	var mu sync.Mutex
	seen := map[string]bool{"root": true}
	processed := map[string]int{}
	queueEndpoints(context.Background(), 4, []*api.Endpoint{root}, func(ep *api.Endpoint) []*api.Endpoint {
		mu.Lock()
		processed[ep.Filename]++
		mu.Unlock()
		if ep == root {
			return []*api.Endpoint{child, child}
		}
		if ep == child {
			return []*api.Endpoint{root, grandchild}
		}
		return nil
	}, func(ep *api.Endpoint) bool {
		mu.Lock()
		defer mu.Unlock()
		if seen[ep.Filename] {
			return false
		}
		seen[ep.Filename] = true
		return true
	})

	if len(processed) != 3 || processed["root"] != 1 || processed["child"] != 1 || processed["grandchild"] != 1 {
		t.Fatalf("processed endpoint counts = %#v", processed)
	}
}

func TestQueueEndpointsAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var processed atomic.Int32
	queueEndpoints(ctx, 4, []*api.Endpoint{{Filename: "queued"}}, func(*api.Endpoint) []*api.Endpoint {
		processed.Add(1)
		return nil
	}, func(*api.Endpoint) bool { return true })
	if got := processed.Load(); got != 0 {
		t.Fatalf("processed endpoints = %d, want 0", got)
	}
}

func TestQueueEndpointsCancellationStopsQueuedWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	initial := make([]*api.Endpoint, 100)
	for i := range initial {
		initial[i] = &api.Endpoint{Filename: fmt.Sprintf("%d", i)}
	}
	var processed atomic.Int32
	queueEndpoints(ctx, 4, initial, func(*api.Endpoint) []*api.Endpoint {
		processed.Add(1)
		cancel()
		<-ctx.Done()
		return nil
	}, func(*api.Endpoint) bool { return true })
	if got := processed.Load(); got > 4 {
		t.Fatalf("processed endpoints after cancellation = %d, want at most 4", got)
	}
}

func TestProcessAllContinuesAfterEndpointError(t *testing.T) {
	withTestAPITransport(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/data/wow/test" {
			t.Errorf("request path = %q", r.URL.Path)
		}
		w.Write([]byte(`{}`))
	}))
	client, err := api.NewClient(&api.ClientOptions{Region: api.RegionUS, AccessToken: "test", Game: api.GameRetail})
	if err != nil {
		t.Fatal(err)
	}
	a := NewArchiver(Options{Game: api.GameRetail})
	a.endpoints = []*api.Endpoint{
		{Kind: api.KindCharacter, Base: "/profile/wow/character", Filename: "profile.json"},
		{Kind: api.KindData, Base: "/data/wow/test", Filename: "test.json", Namespace: "static-%s"},
	}
	dir := t.TempDir()
	err = a.ProcessAll(context.Background(), client, dir, api.NewProfileSlug())
	if err != nil {
		t.Fatalf("endpoint failure stopped remaining work: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "retail", "test", "test.json")); err != nil {
		t.Fatalf("successful endpoint was not downloaded: %v", err)
	}
}

func TestProcessAllSkipsUnavailableGuildIndex(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/data/wow/guild/realm/guild" {
					t.Errorf("request path = %q", r.URL.Path)
				}
				w.WriteHeader(status)
			}))
			defer server.Close()

			transport := http.DefaultTransport.(*http.Transport).Clone()
			transport.Proxy = nil
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // test server certificate
			transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
			}
			original := http.DefaultTransport
			http.DefaultTransport = transport
			client, err := api.NewClient(&api.ClientOptions{Region: api.RegionUS, AccessToken: "test"})
			http.DefaultTransport = original
			if err != nil {
				t.Fatal(err)
			}

			a := NewArchiver(Options{Game: api.GameRetail})
			ep := api.GuildEndpoints[0].Copy()
			ep.RequestData = []string{"realm", "guild"}
			a.endpoints = []*api.Endpoint{ep}
			if err := a.ProcessAll(context.Background(), client, t.TempDir(), api.NewProfileSlug()); err != nil {
				t.Fatalf("unavailable guild prevented archive: %v", err)
			}
		})
	}
}
