package archiver

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func withTestAPITransport(t *testing.T, handler http.Handler) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // test server certificate
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	original := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original })
}

func TestRunArchivesAvailableAccountDataDespiteAPIErrors(t *testing.T) {
	withTestAPITransport(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/data/wow/playable-race/index":
			w.Write([]byte(`{"races":[]}`))
		case "/profile/user/wow/collections/mounts":
			w.Write([]byte(`{"mounts":[]}`))
		default:
			status := http.StatusForbidden
			if strings.HasPrefix(r.URL.Path, "/profile/user/wow/collections/") {
				status = http.StatusNotFound
			}
			http.Error(w, "unavailable", status)
		}
	}))
	path, err := Run(context.Background(), Request{
		Region: "us", AccessToken: "test", WorkDir: t.TempDir(), ArchiveDir: t.TempDir(), Version: "test",
	})
	if err != nil {
		t.Fatalf("available API data was not archived: %v", err)
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
	want := map[string]bool{
		"data/retail/playable-race/index/races.json":            false,
		"account/retail/collections/mounts/account-mounts.json": false,
	}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := want[hdr.Name]; ok {
			want[hdr.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("archive missing %s", name)
		}
	}
}

func TestRunRejectsArchiveWithoutDownloadedData(t *testing.T) {
	withTestAPITransport(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusForbidden)
	}))
	path, err := Run(context.Background(), Request{
		Region: "us", AccessToken: "test", WorkDir: t.TempDir(), ArchiveDir: t.TempDir(), Version: "test",
	})
	if err == nil || path != "" || !strings.Contains(err.Error(), "no data downloaded") {
		t.Fatalf("Run() = (%q, %v), want no-data error", path, err)
	}
}

func TestRunArchivesExplicitCharacterWhenAccountProfileFails(t *testing.T) {
	withTestAPITransport(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/profile/wow/character/realm/hero" {
			w.Write([]byte(`{"id":7,"name":"Hero","level":70,"realm":{"id":1,"name":"Realm","slug":"realm"}}`))
			return
		}
		http.Error(w, "unavailable", http.StatusForbidden)
	}))
	path, err := Run(context.Background(), Request{
		Game: "retail", Realm: "Realm", Character: "Hero", Region: "us",
		AccessToken: "test", WorkDir: t.TempDir(), ArchiveDir: t.TempDir(), Version: "test",
	})
	if err != nil {
		t.Fatalf("character profile was not archived: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("archive missing: %v", err)
	}
}
