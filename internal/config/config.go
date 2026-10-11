package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Schedule string

const (
	ScheduleDaily   Schedule = "daily"
	ScheduleWeekly  Schedule = "weekly"
	ScheduleMonthly Schedule = "monthly"
)

type Config struct {
	UserGUID     string     `json:"user_guid,omitempty"`
	ClientID     string     `json:"client_id,omitempty"`
	ClientSecret string     `json:"client_secret,omitempty"`
	Region       string     `json:"region,omitempty"`
	AccessToken  string     `json:"access_token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	TokenExpiry  *time.Time `json:"token_expiry,omitempty"`
	QueryEnabled bool       `json:"query_enabled"`
	Schedule     Schedule   `json:"schedule"`
	ScheduleHour int        `json:"schedule_hour"`
	ScheduleMin  int        `json:"schedule_min"`
	ArchiveDir   string     `json:"archive_dir,omitempty"`
	AccountID    string     `json:"account_id,omitempty"` // from OpenID /userinfo sub
	BattleTag    string     `json:"battle_tag,omitempty"` // from OpenID /userinfo battletag
}

// DefaultConfig creates a new Config with a generated GUID and derived schedule.
// Returns an error only if GUID generation fails.
func DefaultConfig() (*Config, error) {
	guid, err := GenerateGUID()
	if err != nil {
		return nil, err
	}
	hour, min := DeriveSchedule(guid)
	archiveDir, _ := ArchiveDir() // best-effort
	return &Config{
		UserGUID:     guid,
		Region:       "us",
		QueryEnabled: false,
		Schedule:     ScheduleDaily,
		ScheduleHour: hour,
		ScheduleMin:  min,
		ArchiveDir:   archiveDir,
	}, nil
}

// Load reads config from path. Returns empty default config if file doesn't exist.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig()
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes config to path with 0600 permissions.
func Save(cfg *Config, path string) error {
	stored := *cfg
	stored.ClientID = ""
	stored.ClientSecret = ""
	stored.AccessToken = ""
	stored.RefreshToken = ""
	stored.TokenExpiry = nil
	data, err := json.MarshalIndent(&stored, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// LoadDefault loads config from the platform default path.
func LoadDefault() (*Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}
	return Load(path)
}

// SaveDefault saves config to the platform default path.
func SaveDefault(cfg *Config) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}
	return Save(cfg, path)
}
