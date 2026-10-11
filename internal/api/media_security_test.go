package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestValidateMediaURL(t *testing.T) {
	for _, tc := range []struct {
		raw, host string
		ok        bool
	}{
		{"https://render-eu.worldofwarcraft.com/image.jpg", "render-eu.worldofwarcraft.com", true},
		{"https://media.blizzard.com/image.jpg", "media.blizzard.com", true},
		{"http://render-eu.worldofwarcraft.com/image.jpg", "", false},
		{"https://evil.example/image.jpg", "", false},
		{"https://127.0.0.1/image.jpg", "", false},
		{"https://169.254.169.254/image.jpg", "", false},
		{"https://user:pass@media.blizzard.com/image.jpg", "", false},
		{"https://media.blizzard.com:444/image.jpg", "", false},
	} {
		u, err := validateMediaURL(tc.raw)
		if (err == nil) != tc.ok {
			t.Errorf("validateMediaURL(%q): url=%v err=%v", tc.raw, u, err)
		}
		if tc.ok && u.Hostname() != tc.host {
			t.Errorf("host=%q, want %q", u.Hostname(), tc.host)
		}
	}
}

type mediaTransport func(*http.Request) (*http.Response, error)

func (f mediaTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRequestMediaOmitsAuthorization(t *testing.T) {
	c, err := NewClient(&ClientOptions{Region: RegionUS, AccessToken: "secret-token"})
	if err != nil {
		t.Fatal(err)
	}
	var gotAuth string
	c.httpClient.Transport = mediaTransport(func(r *http.Request) (*http.Response, error) {
		gotAuth = r.Header.Get("Authorization")
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("image"))}, nil
	})
	_, err = c.RequestMedia(context.Background(), t.TempDir(), &Endpoint{Kind: KindMedia, Base: "media", Path: "image", Url: "https://media.blizzard.com/image.jpg"})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "" {
		t.Fatalf("media request sent authorization header %q", gotAuth)
	}
}

func TestRequestMediaBlocksRedirect(t *testing.T) {
	c, err := NewClient(&ClientOptions{Region: RegionUS, AccessToken: "secret-token"})
	if err != nil {
		t.Fatal(err)
	}
	c.httpClient.Transport = mediaTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"https://evil.example/image.jpg"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})
	_, err = c.RequestMedia(context.Background(), t.TempDir(), &Endpoint{Kind: KindMedia, Base: "media", Path: "image", Url: "https://media.blizzard.com/image.jpg"})
	if err == nil || !strings.Contains(err.Error(), "blocked media redirect") {
		t.Fatalf("redirect was not blocked: %v", err)
	}
}
