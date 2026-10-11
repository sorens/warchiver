package auth

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

const callbackPort = 8085

// InteractiveAuth performs the full interactive OAuth flow
func InteractiveAuth(ctx context.Context, clientID, clientSecret, region string) (*Token, error) {
	config := OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  fmt.Sprintf("http://localhost:%d/callback", callbackPort),
		Scopes:       []string{"wow.profile", "openid"},
		Region:       region,
	}

	oauth := NewOAuth(config)

	// Generate state for CSRF protection
	state, err := GenerateState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	// Start callback server
	server := NewCallbackServer(callbackPort)
	if err := server.Start(state); err != nil {
		return nil, fmt.Errorf("starting callback server: %w", err)
	}
	defer server.Shutdown()

	// Build authorization URL
	authURL := oauth.GetAuthorizationURL(state)

	// Open browser
	fmt.Println("Opening browser for Battle.net authorization")
	fmt.Println()
	fmt.Printf("If the browser doesn't open, visit:\n\n%s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser: %v\n", err)
	}

	// Wait for callback with timeout
	fmt.Println("Waiting for authorization (timeout: 5 minutes)")
	fmt.Println()

	authCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var code string
	err = server.Wait(authCtx, &code)
	if err != nil {
		return nil, fmt.Errorf("waiting for authorization: %w", err)
	}

	fmt.Println("Authorization received")

	token, err := oauth.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchanging code: %w", err)
	}

	fmt.Println("Authentication successful")
	fmt.Println()
	return token, nil
}

// openBrowser opens the default browser to the given URL
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
