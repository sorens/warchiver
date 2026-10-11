package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sorens/warchiver"
	"github.com/sorens/warchiver/internal/config"
	"github.com/sorens/warchiver/internal/credentials"
	walog "github.com/sorens/warchiver/internal/log"
)

var (
	buildTime        = "unknown"
	buildVersion     = ""
	marketingVersion = "dev"

	DefaultConfigDir = func() string {
		p, err := config.ConfigFilePath()
		if err != nil {
			return ""
		}
		return p
	}()
)

// ctxKey is a private context.Value key type to avoid cross-package collisions.
type ctxKey int

const ctxKeyDebugDir ctxKey = iota

func useragent() string {
	return fmt.Sprintf("warchiver/%s (%s/%s)", marketingVersion, buildVersion, buildTime)
}

func AppName() string {
	return "warchiver"
}

func usage() {
	fmt.Fprintf(os.Stderr, `%s — World of Warcraft™ account & character data archiver
copyright © 2026 Stephen Orens

usage:
  %s <command> <subcommand> [flags]

commands/subcommands:
  auth:
    info                      account info
    login                     authenticate with Battle.net
    logout                    remove authentication token
  account:
    list                      list characters for specified account
  archive:
    account                   archives all account data and all eligible characters on all realms
    character <realm> <char>  archive single realm/character

common flags:
  -config   path to config file
  -region   Battle.net region: us, eu, kr, tw, cn
  -verbose  enable verbose output

environment variables:
  WARCHIVER_DIR          working directory for debugging
  WARCHIVER_TOKEN        override stored access token (development use)
  WARCHIVER_TOKEN_EXPIRY expiry for WARCHIVER_TOKEN (RFC3339)

examples:
  %s auth login
    authenticate a Battle.net™ account; required for list & archive commands
  %s account ls
    list all characters on all WoW accounts
  %s account ls -minlvl 10
    list all characters level 10 and above on all WoW accounts
  %s archive char <realm> <character>
    archive <realm>/<character>
  %s archive account
    archives all account data and all eligible characters on all realms (Blizzard excludes data from characters below level 10)
`, AppName(), AppName(), AppName(), AppName(), AppName(), AppName(), AppName())
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	debugDir := os.Getenv("WARCHIVER_DIR") // specify a working directory for debugging

	ctx := context.WithValue(context.Background(), ctxKeyDebugDir, debugDir)

	if debugDir != "" {
		if err := os.MkdirAll(debugDir, 0755); err != nil {
			errorf(ctx, err, "failed to create WARCHIVER_DIR")
		}
	}

	var err error
	switch cmd {
	case "auth":
		err = runAuth(ctx, args)
	case "account":
		err = runAccount(ctx, args)
	case "archive":
		err = runArchive(ctx, args)
	case "version":
		err = runVersion(ctx, args)
	case "--help", "-help", "-h", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(1)
	}

	if err != nil {
		errorf(ctx, err, "cmd error %q", cmd)
	}
}

// loadOrCreateConfig loads config from the given path, or the platform default if path is empty.
// If no config file exists, a default config is created and saved.
// WARCHIVER_TOKEN and WARCHIVER_TOKEN_EXPIRY env vars override the stored token for dev use.
func loadOrCreateConfig(configPath string) (*config.Config, string, error) {
	path := configPath
	if path == "" {
		p, err := config.ConfigFilePath()
		if err != nil {
			return nil, "", fmt.Errorf("determining config path: %w", err)
		}
		path = p
	}

	cfg, err := config.Load(path)
	if err != nil {
		return nil, "", fmt.Errorf("loading config: %w", err)
	}
	store := credentials.NewStore(path, cfg.UserGUID)
	stored, err := store.Load()
	switch {
	case err == nil:
		cfg.ClientID = stored.ClientID
		cfg.ClientSecret = stored.ClientSecret
		cfg.AccessToken = stored.AccessToken
		cfg.RefreshToken = stored.RefreshToken
		cfg.TokenExpiry = stored.TokenExpiry
	case errors.Is(err, credentials.ErrNotFound):
		if cfg.ClientID != "" || cfg.ClientSecret != "" || cfg.AccessToken != "" || cfg.RefreshToken != "" || cfg.TokenExpiry != nil {
			cfg.ClientID = ""
			cfg.ClientSecret = ""
			cfg.AccessToken = ""
			cfg.RefreshToken = ""
			cfg.TokenExpiry = nil
			if err := config.Save(cfg, path); err != nil {
				return nil, "", fmt.Errorf("removing credentials from config file: %w", err)
			}
		}
	default:
		return nil, "", fmt.Errorf("loading credentials from platform keychain: %w", err)
	}

	if tok := os.Getenv("WARCHIVER_TOKEN"); tok != "" {
		cfg.AccessToken = tok
		if expiry := os.Getenv("WARCHIVER_TOKEN_EXPIRY"); expiry != "" {
			t, err := time.Parse(time.RFC3339Nano, expiry)
			if err == nil {
				cfg.TokenExpiry = &t
			}
		}
		if cfg.TokenExpiry == nil {
			t := time.Now().Add(time.Hour)
			cfg.TokenExpiry = &t
		}
	}

	return cfg, path, nil
}

func saveCredentials(cfg *config.Config, configPath string) error {
	return credentials.NewStore(configPath, cfg.UserGUID).Save(credentials.Credentials{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		AccessToken:  cfg.AccessToken,
		RefreshToken: cfg.RefreshToken,
		TokenExpiry:  cfg.TokenExpiry,
	})
}

// tokenRefreshLeeway proactively refreshes tokens that expire within this
// window so an in-flight archive run does not cross the boundary.
const tokenRefreshLeeway = 60 * time.Second

// requireToken checks that the config has a non-expired access token.
// If TokenExpiry is within tokenRefreshLeeway, it attempts a refresh using
// cfg.RefreshToken and persists the new token to cfgPath. On refresh failure
// (or when no refresh token is available), it returns a clear error directing
// the user to re-run `auth login`.
func requireToken(ctx context.Context, cfg *config.Config, cfgPath string) error {
	if cfg.AccessToken == "" || cfg.TokenExpiry == nil {
		return fmt.Errorf("not authenticated — run 'warchiver auth login' first")
	}
	if time.Until(*cfg.TokenExpiry) > tokenRefreshLeeway {
		return nil
	}
	if cfg.RefreshToken == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return fmt.Errorf("token expired at %s — run 'warchiver auth login' to re-authenticate",
			cfg.TokenExpiry.Format(time.RFC3339))
	}
	oa := warchiver.Auth(warchiver.AuthConfig{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Region:       cfg.Region,
	})
	tok, err := oa.RefreshToken(ctx, cfg.RefreshToken)
	if err != nil {
		return fmt.Errorf("token expired and refresh failed (%v) — run 'warchiver auth login' to re-authenticate", err)
	}
	cfg.AccessToken = tok.AccessToken
	exp := tok.ExpiresAt()
	cfg.TokenExpiry = &exp
	if tok.RefreshToken != "" {
		cfg.RefreshToken = tok.RefreshToken
	}
	if err := saveCredentials(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving refreshed credentials to platform keychain: %w", err)
	}
	if cfgPath != "" {
		if err := config.Save(cfg, cfgPath); err != nil {
			return fmt.Errorf("saving refreshed token: %w", err)
		}
	}
	walog.Ctx(ctx).Verbose("auth: refreshed access token (valid until %s)", exp.Format(time.RFC3339))
	return nil
}

// print the error
func errorf(ctx context.Context, err error, format string, args ...any) {
	_ = ctx
	var b strings.Builder
	fmt.Fprintf(&b, "[ERRR] ")
	if format != "" {
		fmt.Fprintf(&b, "%s", fmt.Sprintf(format, args...))
	}
	if err != nil {
		fmt.Fprintf(&b, ": %v", err)
	}

	fmt.Fprintln(os.Stderr, b.String())
}

// print the error and exit
func fatalf(ctx context.Context, err error, format string, args ...any) {
	_ = ctx
	var b strings.Builder
	fmt.Fprintf(&b, "[FATL] ")
	if err != nil {
		fmt.Fprintf(&b, "%v", err)
	}
	if format != "" {
		fmt.Fprintf(&b, ": %s", fmt.Sprintf(format, args...))
	}

	fmt.Fprintln(os.Stderr, b.String())
	os.Exit(1)
}

// printf writes user-facing text to stdout (two-channel I/O policy).
func printf(ctx context.Context, format string, args ...any) {
	_ = ctx
	fmt.Fprintf(os.Stdout, format, args...)
	fmt.Fprintln(os.Stdout)
}

// verbose diagnostic message; requires -verbose
func verbosef(ctx context.Context, format string, args ...any) {
	walog.Ctx(ctx).Verbose(format, args...)
}

func warnf(ctx context.Context, err error, format string, args ...any) {
	_ = ctx
	f := format
	if err != nil {
		f = fmt.Sprintf("WARN [%v]: %s", err, format)
	}

	fmt.Fprintf(os.Stderr, f, args...)
}
