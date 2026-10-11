package credentials

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
)

const service = "com.warchiver"

var ErrNotFound = keyring.ErrNotFound

type Credentials struct {
	ClientID     string     `json:"client_id,omitempty"`
	ClientSecret string     `json:"client_secret,omitempty"`
	AccessToken  string     `json:"access_token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	TokenExpiry  *time.Time `json:"token_expiry,omitempty"`
}

type Store struct {
	account string
}

func NewStore(configPath, userGUID string) *Store {
	account := userGUID
	if account == "" {
		path, err := filepath.Abs(configPath)
		if err == nil {
			path = filepath.Clean(path)
		}
		hash := sha256.Sum256([]byte(path))
		account = hex.EncodeToString(hash[:])
	}
	return &Store{account: account}
}

func (s *Store) Load() (*Credentials, error) {
	payload, err := keyring.Get(service, s.account)
	if err != nil {
		return nil, err
	}
	var creds Credentials
	if err := json.Unmarshal([]byte(payload), &creds); err != nil {
		return nil, fmt.Errorf("decoding keychain credentials: %w", err)
	}
	return &creds, nil
}

func (s *Store) Save(creds Credentials) error {
	payload, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	if err := keyring.Set(service, s.account, string(payload)+"\n"); err != nil {
		return err
	}
	return nil
}

func (s *Store) Delete() error {
	err := keyring.Delete(service, s.account)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
