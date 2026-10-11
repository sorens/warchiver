package auth

import "time"

// tokenExpirySafetyWindow buffers against clock skew and in-flight latency.
const tokenExpirySafetyWindow = 60 * time.Second

// Token represents an OAuth token response
type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	Scope        string    `json:"scope"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ObtainedAt   time.Time `json:"obtained_at"`
}

// IsExpired reports whether the token is within tokenExpirySafetyWindow of expiry.
func (t *Token) IsExpired() bool {
	ttl := time.Duration(t.ExpiresIn)*time.Second - tokenExpirySafetyWindow
	if ttl < 0 {
		return true
	}
	return time.Since(t.ObtainedAt) > ttl
}

// ExpiresAt returns the absolute expiry time
func (t *Token) ExpiresAt() time.Time {
	return t.ObtainedAt.Add(time.Duration(t.ExpiresIn) * time.Second)
}
