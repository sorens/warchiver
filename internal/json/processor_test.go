package json_processor

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchAndProcessStreamsAndExtracts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.json")
	p := NewJSONProcessor([]string{"name"})
	if err := p.FetchAndProcess(io.NopCloser(strings.NewReader(`{"name":"a","nested":[{"name":"b"}]}`)), path); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(p.Values(), ","); got != "a,b" {
		t.Fatalf("values=%q", got)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) == "" {
		t.Fatal("output file empty")
	}
}

func TestFetchAndProcessFromURLRejectsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "upstream failed", http.StatusBadGateway) }))
	defer srv.Close()
	p := NewJSONProcessor([]string{"name"})
	if err := p.FetchAndProcessFromURL(srv.URL, filepath.Join(t.TempDir(), "x")); err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("error=%v", err)
	}
}

func TestFetchAndProcessRejectsInvalidJSON(t *testing.T) {
	p := NewJSONProcessor([]string{"name"})
	if err := p.FetchAndProcess(io.NopCloser(strings.NewReader(`{"name":`)), filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatal("expected parse error")
	}
}
