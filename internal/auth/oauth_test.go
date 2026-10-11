package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestRefreshToken_Success(t *testing.T) {
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotForm, _ = url.ParseQuery(string(body))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access",
			"token_type":    "bearer",
			"expires_in":    3600,
			"scope":         "wow.profile openid",
			"refresh_token": "new-refresh",
		})
	}))
	defer srv.Close()

	oa := NewOAuth(OAuthConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		BaseURL:      srv.URL,
	})

	tok, err := oa.RefreshToken(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if tok.AccessToken != "new-access" {
		t.Errorf("AccessToken = %q, want new-access", tok.AccessToken)
	}
	if tok.RefreshToken != "new-refresh" {
		t.Errorf("RefreshToken = %q, want new-refresh", tok.RefreshToken)
	}
	if tok.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want 3600", tok.ExpiresIn)
	}
	if tok.IsExpired() {
		t.Errorf("token reported expired immediately after refresh")
	}
	if got := gotForm.Get("grant_type"); got != "refresh_token" {
		t.Errorf("grant_type = %q, want refresh_token", got)
	}
	if got := gotForm.Get("refresh_token"); got != "old-refresh" {
		t.Errorf("refresh_token = %q, want old-refresh", got)
	}
}

func TestRefreshToken_PreservesRefreshOnEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	oa := NewOAuth(OAuthConfig{ClientID: "id", ClientSecret: "secret", BaseURL: srv.URL})
	tok, err := oa.RefreshToken(context.Background(), "preserve-me")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if tok.RefreshToken != "preserve-me" {
		t.Errorf("RefreshToken = %q, want preserve-me", tok.RefreshToken)
	}
}

func TestRefreshToken_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	oa := NewOAuth(OAuthConfig{ClientID: "id", ClientSecret: "secret", BaseURL: srv.URL})
	_, err := oa.RefreshToken(context.Background(), "bad")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error %q does not contain status code 401", err.Error())
	}
}

func TestRefreshToken_EmptyRefreshToken(t *testing.T) {
	oa := NewOAuth(OAuthConfig{ClientID: "id", ClientSecret: "secret", BaseURL: "http://unused"})
	_, err := oa.RefreshToken(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty refresh token")
	}
}

// TestTokenExpiresAt is a sanity check used by the CLI's requireToken path:
// when the cached token's ExpiresAt is in the past, callers should attempt
// refresh rather than returning the access token as-is.
func TestTokenExpiresAt(t *testing.T) {
	tok := &Token{
		ExpiresIn:  3600,
		ObtainedAt: time.Now().Add(-2 * time.Hour),
	}
	if !tok.IsExpired() {
		t.Errorf("IsExpired = false for token obtained 2h ago with 1h lifetime")
	}
	if tok.ExpiresAt().After(time.Now()) {
		t.Errorf("ExpiresAt %v is in the future for an already-expired token", tok.ExpiresAt())
	}
}
