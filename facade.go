package warchiver

import (
	"context"
	"log/slog"
	"time"

	"github.com/sorens/warchiver/internal/api"
	"github.com/sorens/warchiver/internal/archiver"
	"github.com/sorens/warchiver/internal/auth"
	walog "github.com/sorens/warchiver/internal/log"
	"github.com/sorens/warchiver/pkg/features"
)

type Game string

const (
	GameRetail             Game = "retail"
	GameClassic            Game = "classic"
	GameClassic1x          Game = "classic1x"
	GameClassicAnniversary Game = "classicann"
)

type ArchiveRequest struct {
	Game        Game
	Realm       string
	Character   string
	Region      string
	AccessToken string
	BattleTag   string
	WorkDir     string
	ArchiveDir  string
	ConfigDir   string
	WowDir      string
	SIMCBinPath string
	SimcJobs    int
	SimcThreads int
	EnrichItems bool
	Features    features.Provider
	MinLevel    int
	Version     string
	Logger      *slog.Logger
}

type ArchiveResult struct {
	Path string
}

func Archive(ctx context.Context, req ArchiveRequest) (ArchiveResult, error) {
	if req.Logger != nil {
		ctx = walog.FromSlog(ctx, req.Logger).WithContext(ctx)
	}
	path, err := archiver.Run(ctx, archiver.Request{
		Game:        api.GameProperty(req.Game),
		Realm:       req.Realm,
		Character:   req.Character,
		Region:      req.Region,
		AccessToken: req.AccessToken,
		BattleTag:   req.BattleTag,
		WorkDir:     req.WorkDir,
		ArchiveDir:  req.ArchiveDir,
		ConfigDir:   req.ConfigDir,
		WowDir:      req.WowDir,
		SIMCBinPath: req.SIMCBinPath,
		SimcJobs:    req.SimcJobs,
		SimcThreads: req.SimcThreads,
		EnrichItems: req.EnrichItems,
		Features:    req.Features,
		MinLevel:    req.MinLevel,
		Version:     req.Version,
	})
	if err != nil {
		return ArchiveResult{}, err
	}
	return ArchiveResult{Path: path}, nil
}

type AuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	Region       string
	BaseURL      string
}

type AuthClient struct {
	client *auth.OAuth
}

type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	Scope        string    `json:"scope"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ObtainedAt   time.Time `json:"obtained_at"`
}

func (t *Token) IsExpired() bool {
	ttl := time.Duration(t.ExpiresIn)*time.Second - time.Minute
	return ttl < 0 || time.Since(t.ObtainedAt) > ttl
}

func (t *Token) ExpiresAt() time.Time {
	return t.ObtainedAt.Add(time.Duration(t.ExpiresIn) * time.Second)
}

type UserInfo struct {
	Sub       string `json:"sub"`
	ID        int    `json:"id"`
	BattleTag string `json:"battletag"`
}

func Auth(config AuthConfig) *AuthClient {
	return &AuthClient{client: auth.NewOAuth(auth.OAuthConfig{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURI:  config.RedirectURI,
		Scopes:       config.Scopes,
		Region:       config.Region,
		BaseURL:      config.BaseURL,
	})}
}

func (a *AuthClient) AuthorizationURL(state string) string {
	return a.client.GetAuthorizationURL(state)
}

func (a *AuthClient) ExchangeCode(ctx context.Context, code string) (*Token, error) {
	token, err := a.client.ExchangeCode(ctx, code)
	return tokenFromInternal(token), err
}

func (a *AuthClient) UserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	info, err := a.client.GetUserInfo(ctx, accessToken)
	if err != nil || info == nil {
		return nil, err
	}
	return &UserInfo{Sub: info.Sub, ID: info.ID, BattleTag: info.BattleTag}, nil
}

func (a *AuthClient) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	token, err := a.client.RefreshToken(ctx, refreshToken)
	return tokenFromInternal(token), err
}

func tokenFromInternal(token *auth.Token) *Token {
	if token == nil {
		return nil
	}
	return &Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		ExpiresIn:    token.ExpiresIn,
		Scope:        token.Scope,
		RefreshToken: token.RefreshToken,
		ObtainedAt:   token.ObtainedAt,
	}
}

func NewAuthState() (string, error) {
	return auth.GenerateState()
}

func InteractiveAuth(ctx context.Context, clientID, clientSecret, region string) (*Token, error) {
	token, err := auth.InteractiveAuth(ctx, clientID, clientSecret, region)
	return tokenFromInternal(token), err
}

func ValidateRegion(region string) error {
	_, err := api.Region(region).BaseURL()
	return err
}

type AccountRequest struct {
	AccessToken string
	Region      string
	Game        Game
	WorkDir     string
	MinLevel    int
}

type AccountProfile struct {
	ID       int
	Accounts []WoWAccount
}

type WoWAccount struct {
	ID         int
	Characters []Character
}

type Character struct {
	ProfileID     int
	AccountID     int
	Name          string
	Level         int
	Realm         Realm
	PlayableClass NamedValue
	Faction       NamedValue
}

type Realm struct {
	Name string
	ID   int
	Slug string
}

type NamedValue struct {
	Name string
}

func Account(ctx context.Context, req AccountRequest) (*AccountProfile, error) {
	game := req.Game
	if game == "" {
		game = GameRetail
	}
	arch := archiver.NewArchiver(archiver.Options{Game: api.GameProperty(game)})
	client, err := api.NewClient(&api.ClientOptions{
		AccessToken: req.AccessToken,
		Region:      api.Region(req.Region),
		Game:        api.GameProperty(game),
	})
	if err != nil {
		return nil, err
	}
	profile, err := arch.Account(ctx, client, req.WorkDir, req.MinLevel)
	if err != nil || profile == nil {
		return nil, err
	}
	result := &AccountProfile{ID: profile.ID, Accounts: make([]WoWAccount, 0, len(profile.Accounts))}
	for _, account := range profile.Accounts {
		if account == nil {
			continue
		}
		resultAccount := WoWAccount{ID: account.ID, Characters: make([]Character, 0, len(account.Characters))}
		for _, character := range account.Characters {
			if character == nil {
				continue
			}
			resultCharacter := Character{
				ProfileID:     profile.ID,
				AccountID:     account.ID,
				Name:          character.Name,
				Level:         character.Level,
				PlayableClass: NamedValue{Name: character.PlayableClass.Name},
				Faction:       NamedValue{Name: character.Faction.Name},
			}
			if character.Realm != nil {
				resultCharacter.Realm = Realm{Name: character.Realm.Name, ID: character.Realm.ID, Slug: character.Realm.Slug}
			}
			resultAccount.Characters = append(resultAccount.Characters, resultCharacter)
		}
		result.Accounts = append(result.Accounts, resultAccount)
	}
	return result, nil
}
