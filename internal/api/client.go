package api

import (
	"errors"
	"fmt"
	"net/http"
)

// APIError represents an HTTP error response from the Battle.net API.
type APIError struct {
	StatusCode int
	Path       string
	Filename   string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d [%s] %s", e.StatusCode, e.Body, e.Path)
}

// IsNotFound reports whether err is an APIError with status 404.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// Region represents a Battle.net region
type Region string

const (
	RegionUS Region = "us"
	RegionEU Region = "eu"
	RegionKR Region = "kr"
	RegionTW Region = "tw"
	RegionCN Region = "cn"
)

type ClientOptions struct {
	AccessToken string
	Region      Region
	Game        GameProperty
	Limiter     *RateLimiter
}

// BaseURL returns only a fixed, trusted API origin for a supported region.
func (r Region) BaseURL() (string, error) {
	switch r {
	case RegionUS:
		return "https://us.api.blizzard.com", nil
	case RegionEU:
		return "https://eu.api.blizzard.com", nil
	case RegionKR:
		return "https://kr.api.blizzard.com", nil
	case RegionTW:
		return "https://tw.api.blizzard.com", nil
	case RegionCN:
		return "https://gateway.battlenet.com.cn", nil
	default:
		return "", fmt.Errorf("unsupported region %q", r)
	}
}
