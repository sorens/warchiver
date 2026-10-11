package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/sorens/warchiver"
	"github.com/sorens/warchiver/internal/auth"
	"github.com/sorens/warchiver/internal/config"
	"github.com/sorens/warchiver/internal/credentials"
	walog "github.com/sorens/warchiver/internal/log"
)

func runAuth(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("auth", flag.ExitOnError)
	fs.Usage = func() {
		FlagSetUsage(fs)
		fmt.Fprintf(fs.Output(), "subcommands:\n")
		fmt.Fprintf(fs.Output(), "  info\n    account info\n")
		fmt.Fprintf(fs.Output(), "  login\n    authenticate with Battle.net\n")
		fmt.Fprintf(fs.Output(), "  logout,clear\n    remove authentication token\n")
	}
	fs.Parse(args)

	if len(args) == 0 {
		fs.Usage()
		return nil
	}

	switch args[0] {
	case "debug":
		return runDebug(ctx, args[1:])
	case "info":
		return runAuthInfo(ctx, args[1:])
	case "login":
		return runAuthLogin(ctx, args[1:])
	case "logout", "clear":
		return runAuthClear(ctx, args[1:])
	default:
		fs.Usage()
		return nil
	}
}

func runAuthLogin(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	fs.Usage = func() {
		FlagSetUsage(fs)
	}
	var configPath string
	var region string
	var verbose bool
	fs.StringVar(&configPath, "config", DefaultConfigDir, "path to config file")
	fs.StringVar(&region, "region", "us", "Battle.net region (us, eu, kr, tw, cn)")
	fs.BoolVar(&verbose, "verbose", false, "enable verbose output")

	var clientID, clientSecret string
	fs.StringVar(&clientID, "client-id", "", "Battle.net API client ID (WARCHIVER_BNET_CLIENT_ID)")
	fs.StringVar(&clientSecret, "client-secret", "", "Battle.net API client secret (WARCHIVER_BNET_CLIENT_SECRET)")

	fs.Parse(args)

	level := walog.LevelInfo
	if verbose {
		level = walog.LevelVerbose
	}
	ctx = walog.New(os.Stderr, level).WithContext(ctx)

	cfg, cfgPath, err := loadOrCreateConfig(configPath)
	if err != nil {
		return err
	}

	var userInfo *warchiver.UserInfo
	var oauth *warchiver.AuthClient
	var token *warchiver.Token
	var authRegion string
	var tv time.Time

	if cfg.AccessToken != "" && cfg.TokenExpiry != nil && cfg.TokenExpiry.After(time.Now()) {
		goto authenticated
	}

	// resolve client ID: flag > env > config
	if clientID == "" {
		clientID = os.Getenv("WARCHIVER_BNET_CLIENT_ID")
	}
	if clientID == "" {
		clientID = cfg.ClientID
	}
	if clientID == "" {
		return fmt.Errorf("client ID required: use -client-id, WARCHIVER_BNET_CLIENT_ID env var, or set in config")
	}

	// resolve client secret: flag > env > config
	if clientSecret == "" {
		clientSecret = os.Getenv("WARCHIVER_BNET_CLIENT_SECRET")
	}
	if clientSecret == "" {
		clientSecret = cfg.ClientSecret
	}
	if clientSecret == "" {
		return fmt.Errorf("client secret required: use -client-secret, WARCHIVER_BNET_CLIENT_SECRET env var, or set in config")
	}

	authRegion = region
	if authRegion == "" {
		authRegion = cfg.Region
	}
	if authRegion == "" {
		authRegion = "us"
	}

	// interactive OAuth flow
	token, err = warchiver.InteractiveAuth(ctx, clientID, clientSecret, authRegion)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// fetch user info for BattleTag + AccountID
	oauth = warchiver.Auth(warchiver.AuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Region:       authRegion,
	})

	userInfo, err = oauth.UserInfo(ctx, token.AccessToken)
	if err != nil {
		return fmt.Errorf("fetching user info: %w", err)
	}

	cfg.ClientID = clientID
	cfg.ClientSecret = clientSecret
	cfg.Region = authRegion
	cfg.AccessToken = token.AccessToken
	cfg.RefreshToken = token.RefreshToken
	tv = token.ExpiresAt()
	cfg.TokenExpiry = &tv
	cfg.AccountID = userInfo.Sub
	cfg.BattleTag = userInfo.BattleTag

	if err := saveCredentials(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving credentials to platform keychain: %w", err)
	}
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

authenticated:
	printf(ctx, "%s Authenticated", cfg.BattleTag)
	if cfg.TokenExpiry != nil {
		printf(ctx, "Token valid until %s", cfg.TokenExpiry.Format("Jan 2, 2006 at 3:04PM MST"))
	}
	printf(ctx, "use `auth logout` to clear authentication token")

	return nil
}

func runAuthClear(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("clear", flag.ExitOnError)

	var configPath string
	var verbose bool
	fs.StringVar(&configPath, "config", DefaultConfigDir, "path to config file")
	fs.BoolVar(&verbose, "verbose", false, "enable verbose output")
	fs.Parse(args)

	level := walog.LevelInfo
	if verbose {
		level = walog.LevelVerbose
	}
	ctx = walog.New(os.Stderr, level).WithContext(ctx)

	cfg, cfgPath, err := loadOrCreateConfig(configPath)
	if err != nil {
		return err
	}

	if err := credentials.NewStore(cfgPath, cfg.UserGUID).Delete(); err != nil {
		return fmt.Errorf("removing credentials from platform keychain: %w", err)
	}
	cfg.AccountID = ""
	cfg.AccessToken = ""
	cfg.RefreshToken = ""
	cfg.BattleTag = ""
	cfg.TokenExpiry = nil
	cfg.UserGUID = ""
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	printf(ctx, "Authentication cleared")
	return nil
}

func runAuthInfo(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	fs.Usage = func() {
		FlagSetUsage(fs)
	}
	var clientID string
	var clientSecret string
	var configPath string
	var region string
	var verbose bool
	fs.StringVar(&clientID, "client-id", "", "Battle.net API client ID (WARCHIVER_BNET_CLIENT_ID)")
	fs.StringVar(&clientSecret, "client-secret", "", "Battle.net API client secret (WARCHIVER_BNET_CLIENT_SECRET)")
	fs.StringVar(&configPath, "config", DefaultConfigDir, "path to config file")
	fs.StringVar(&region, "region", "us", "Battle.net region (us, eu, kr, tw, cn)")
	fs.BoolVar(&verbose, "verbose", false, "enable verbose output")

	fs.Parse(args)

	level := walog.LevelInfo
	if verbose {
		level = walog.LevelVerbose
	}
	ctx = walog.New(os.Stderr, level).WithContext(ctx)
	cfg, _, err := loadOrCreateConfig(configPath)
	if err != nil {
		return err
	}

	if cfg.AccessToken != "" && cfg.TokenExpiry != nil && cfg.TokenExpiry.After(time.Now()) {
		printf(ctx, "%s Authenticated", cfg.BattleTag)
		printf(ctx, "Token valid until %s", cfg.TokenExpiry.Format("Jan 2, 2006 at 3:04PM MST"))
		return nil
	}

	return fmt.Errorf("not authenticated")
}

func runDebug(ctx context.Context, args []string) error {
	_ = args

	state, err := warchiver.NewAuthState()
	if err != nil {
		return fmt.Errorf("generating state: %w", err)
	}

	server := auth.NewCallbackServer(8085)
	if err := server.Start(state); err != nil {
		return fmt.Errorf("starting callback server: %w", err)
	}
	defer server.Shutdown()

	cmd := exec.Command("open", "http://localhost:8085/debug")
	if err := cmd.Start(); err != nil {
		return err
	}

	dctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	var code string
	err = server.Wait(dctx, &code)
	_ = code
	if !errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	return nil
}
