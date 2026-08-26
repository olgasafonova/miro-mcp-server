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
	// Deliberate discard: this is teardown of a short-lived local callback
	// server on a path that already has its own result to return. A shutdown
	// failure cannot change the outcome of the flow, and surfacing it would
	// mask the real error. Logged rather than silently dropped.
	defer func() {
		if err := server.Stop(ctx); err != nil {
			f.logger.Warn("callback server shutdown failed", "error", err)
		}
	}()

	// Run the browser round trip, then trade the code for stored tokens
	code, err := f.authorize(ctx, server, state)
	if err != nil {
		return nil, err
	}

	tokens, err := f.exchangeAndStore(ctx, code, state)
	if err != nil {
		return nil, err
	}

	f.logger.Info("Authorization successful!",
		"user_id", tokens.UserID,
		"team_id", tokens.TeamID,
		"expires_at", tokens.ExpiresAt.Format(time.RFC3339),
	)

	return tokens, nil
}

// authorize opens the browser for the consent screen and waits for the
// verified callback, returning the authorization code.
func (f *AuthFlow) authorize(ctx context.Context, server *CallbackServer, state *AuthorizationState) (string, error) {
	authURL := f.provider.GetAuthorizationURL(state)

	f.logger.Info("Opening browser for authorization...", "url", authURL)
	if err := openBrowser(authURL); err != nil {
		f.logger.Warn("Failed to open browser automatically", "error", err)
		fmt.Printf("\nPlease open this URL in your browser:\n%s\n\n", authURL)
	}

	f.logger.Info("Waiting for authorization...", "timeout", "5m")
	result, err := server.WaitForCallback(ctx, 5*time.Minute)
	if err != nil {
		return "", fmt.Errorf("authorization failed: %w", err)
	}
	if result.Error != nil {
		return "", result.Error
	}
	if result.State != state.State {
		return "", fmt.Errorf("state mismatch: possible CSRF attack")
	}
	return result.Code, nil
}

// exchangeAndStore trades the authorization code for tokens and persists
// them in the token store.
func (f *AuthFlow) exchangeAndStore(ctx context.Context, code string, state *AuthorizationState) (*TokenSet, error) {
	f.logger.Info("Exchanging code for tokens...")
	tokens, err := f.provider.ExchangeCode(ctx, code, state.CodeVerifier)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	if err := f.tokenStore.Save(ctx, tokens); err != nil {
		return nil, fmt.Errorf("failed to save tokens: %w", err)
	}
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

// revokeIfPresent revokes a non-empty token, logging (but not propagating)
// failures. Empty tokens are a no-op.
func (f *AuthFlow) revokeIfPresent(ctx context.Context, token, label string) {
	if token == "" {
		return
	}
	if err := f.provider.RevokeToken(ctx, token); err != nil {
		f.logger.Warn("Failed to revoke "+label, "error", err)
	}
}

// Logout revokes tokens and clears storage.
func (f *AuthFlow) Logout(ctx context.Context) error {
	if tokens, err := f.tokenStore.Load(ctx); err == nil && tokens != nil {
		f.revokeIfPresent(ctx, tokens.AccessToken, "access token")
		f.revokeIfPresent(ctx, tokens.RefreshToken, "refresh token")
	}
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
