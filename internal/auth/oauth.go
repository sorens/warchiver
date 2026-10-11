package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuthConfig holds OAuth configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	Region       string // "us", "eu", "kr", "tw", "cn"
	BaseURL      string // overrides region-derived OAuth host (test/dev only)
}

// OAuth handles the OAuth flow
type OAuth struct {
	config       OAuthConfig
	httpClient   *http.Client
	oauthBaseURL string
}

// NewOAuth creates a new OAuth handler; panics on an unparseable BaseURL override.
func NewOAuth(config OAuthConfig) *OAuth {
	oauthBaseURL := config.BaseURL
	if oauthBaseURL == "" {
		if config.Region == "cn" {
			oauthBaseURL = "https://oauth.battlenet.com.cn"
		} else {
			oauthBaseURL = "https://oauth.battle.net"
		}
	}
	if _, err := url.Parse(oauthBaseURL); err != nil {
		panic(fmt.Sprintf("invalid OAuth base URL %q: %v", oauthBaseURL, err))
	}

	if config.Scopes == nil {
		config.Scopes = []string{"wow.profile", "openid"}
	}

	return &OAuth{
		config:       config,
		oauthBaseURL: oauthBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateState creates a random state string for CSRF protection
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GetAuthorizationURL builds the authorization URL.
func (o *OAuth) GetAuthorizationURL(state string) string {
	u, err := url.Parse(o.oauthBaseURL + "/authorize")
	if err != nil {
		panic(fmt.Sprintf("invalid OAuth base URL %q: %v", o.oauthBaseURL, err))
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", o.config.ClientID)
	q.Set("redirect_uri", o.config.RedirectURI)
	q.Set("scope", strings.Join(o.config.Scopes, " "))
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String()
}

// ExchangeCode exchanges an authorization code for an access token
func (o *OAuth) ExchangeCode(ctx context.Context, code string) (*Token, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", o.config.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.oauthBaseURL+"/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(o.config.ClientID, o.config.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	token.ObtainedAt = time.Now()

	return &token, nil
}

// RefreshToken exchanges a refresh token for a fresh access token. Battle.net
// only issues refresh tokens to clients that opted into them; if this client
// did not, the call returns an error and the caller should fall back to a
// full re-authentication.
func (o *OAuth) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is empty")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.oauthBaseURL+"/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(o.config.ClientID, o.config.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed (%d): %s", resp.StatusCode, string(body))
	}

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	token.ObtainedAt = time.Now()
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}
	return &token, nil
}

// ValidateToken checks if a token is still valid
func (o *OAuth) ValidateToken(ctx context.Context, token string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.oauthBaseURL+"/check_token",
		strings.NewReader("token="+token))
	if err != nil {
		return false, err
	}

	req.SetBasicAuth(o.config.ClientID, o.config.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// UserInfo represents the response from the /userinfo endpoint
type UserInfo struct {
	Sub       string `json:"sub"`
	ID        int    `json:"id"`
	BattleTag string `json:"battletag"`
}

// GetUserInfo fetches user information using an access token
func (o *OAuth) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.oauthBaseURL+"/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("creating userinfo request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo request failed %d: %s", resp.StatusCode, string(body))
	}

	var info UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decoding userinfo: %w", err)
	}

	return &info, nil
}
