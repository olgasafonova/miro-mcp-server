package oauth

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"time"
)

// AuthFlow orchestrates the complete OAuth authorization flow.
type AuthFlow struct {
	provider   *Provider
	tokenStore TokenStore
	logger     *slog.Logger
	config     *Config
}

// NewAuthFlow creates a new auth flow handler.
func NewAuthFlow(config *Config, logger *slog.Logger) *AuthFlow {
	return &AuthFlow{
		provider:   NewProvider(config),
		tokenStore: NewFileTokenStore(config.TokenStorePath),
		logger:     logger,
		config:     config,
	}
}

// Login initiates the OAuth flow and waits for authorization.
func (f *AuthFlow) Login(ctx context.Context) (*TokenSet, error) {
	// Generate state and code verifier
	state, err := f.provider.GenerateAuthorizationState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Start callback server
	port, err := GetCallbackPort(f.config.RedirectURI)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect URI: %w", err)
	}

	server, err := NewCallbackServer(port, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	server.Start()
	defer server.Stop(ctx)

	// Get authorization URL
	authURL := f.provider.GetAuthorizationURL(state)

	// Open browser
	f.logger.Info("Opening browser for authorization...", "url", authURL)
	if err := openBrowser(authURL); err != nil {
		f.logger.Warn("Failed to open browser automatically", "error", err)
		fmt.Printf("\nPlease open this URL in your browser:\n%s\n\n", authURL)
	}

	// Wait for callback
	f.logger.Info("Waiting for authorization...", "timeout", "5m")
	result, err := server.WaitForCallback(ctx, 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	if result.Error != nil {
		return nil, result.Error
	}

	// Verify state
	if result.State != state.State {
		return nil, fmt.Errorf("state mismatch: possible CSRF attack")
	}

	// Exchange code for tokens
	f.logger.Info("Exchanging code for tokens...")
	tokens, err := f.provider.ExchangeCode(ctx, result.Code, state.CodeVerifier)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	// Store tokens
	if err := f.tokenStore.Save(ctx, tokens); err != nil {
		return nil, fmt.Errorf("failed to save tokens: %w", err)
	}

	f.logger.Info("Authorization successful!",
		"user_id", tokens.UserID,
		"team_id", tokens.TeamID,
		"expires_at", tokens.ExpiresAt.Format(time.RFC3339),
	)

	return tokens, nil
}

// Status returns the current authentication status.
func (f *AuthFlow) Status(ctx context.Context) (*TokenSet, error) {
	if !f.tokenStore.Exists(ctx) {
		return nil, fmt.Errorf("not logged in")
	}

	tokens, err := f.tokenStore.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load tokens: %w", err)
	}

	// Check if token needs refresh
	if tokens.NeedsRefresh() {
		if tokens.RefreshToken != "" {
			f.logger.Info("Refreshing expired token...")
			newTokens, err := f.provider.RefreshToken(ctx, tokens.RefreshToken)
			if err != nil {
				return nil, fmt.Errorf("token refresh failed: %w", err)
			}
			if err := f.tokenStore.Save(ctx, newTokens); err != nil {
				f.logger.Warn("Failed to save refreshed tokens", "error", err)
			}
			tokens = newTokens
		} else {
			return nil, fmt.Errorf("token expired and no refresh token available")
		}
	}

	return tokens, nil
}

// Logout revokes tokens and clears storage.
func (f *AuthFlow) Logout(ctx context.Context) error {
	tokens, err := f.tokenStore.Load(ctx)
	if err == nil && tokens != nil {
		// Revoke access token
		if tokens.AccessToken != "" {
			if err := f.provider.RevokeToken(ctx, tokens.AccessToken); err != nil {
				f.logger.Warn("Failed to revoke access token", "error", err)
			}
		}
		// Revoke refresh token
		if tokens.RefreshToken != "" {
			if err := f.provider.RevokeToken(ctx, tokens.RefreshToken); err != nil {
				f.logger.Warn("Failed to revoke refresh token", "error", err)
			}
		}
	}

	// Delete stored tokens
	if err := f.tokenStore.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete tokens: %w", err)
	}

	f.logger.Info("Logged out successfully")
	return nil
}

// GetAccessToken returns a valid access token, refreshing if needed.
func (f *AuthFlow) GetAccessToken(ctx context.Context) (string, error) {
	tokens, err := f.Status(ctx)
	if err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
}

// openBrowser opens the URL in the default browser.
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
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
