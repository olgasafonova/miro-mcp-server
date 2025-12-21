// Package oauth provides OAuth 2.1 authentication for the Miro API.
package oauth

import (
	"os"
	"time"
)

// Config holds OAuth configuration.
type Config struct {
	// ClientID is the OAuth client ID from Miro Developer Console
	ClientID string

	// ClientSecret is the OAuth client secret from Miro Developer Console
	ClientSecret string

	// RedirectURI is the callback URL for OAuth flow (default: http://localhost:8089/callback)
	RedirectURI string

	// Scopes are the OAuth scopes to request
	Scopes []string

	// TokenStorePath is the path to store tokens (default: ~/.miro/tokens.json)
	TokenStorePath string
}

// DefaultScopes are the recommended scopes for full Miro access.
var DefaultScopes = []string{
	"boards:read",
	"boards:write",
	"team:read",
	"identity:read",
}

// TokenSet represents OAuth tokens returned by Miro.
type TokenSet struct {
	// AccessToken is the bearer token for API requests
	AccessToken string `json:"access_token"`

	// RefreshToken is used to obtain new access tokens
	RefreshToken string `json:"refresh_token"`

	// ExpiresAt is when the access token expires
	ExpiresAt time.Time `json:"expires_at"`

	// TokenType is typically "bearer"
	TokenType string `json:"token_type"`

	// Scope is the granted scopes (space-separated)
	Scope string `json:"scope"`

	// UserID is the Miro user ID associated with the token
	UserID string `json:"user_id,omitempty"`

	// TeamID is the Miro team ID
	TeamID string `json:"team_id,omitempty"`
}

// IsExpired returns true if the access token has expired.
func (t *TokenSet) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// NeedsRefresh returns true if the token should be refreshed.
// Returns true if token expires within 5 minutes.
func (t *TokenSet) NeedsRefresh() bool {
	return time.Now().Add(5 * time.Minute).After(t.ExpiresAt)
}

// TokenResponse is the raw response from Miro's token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds until expiry
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	UserID       string `json:"user_id"`
	TeamID       string `json:"team_id"`
}

// ToTokenSet converts a TokenResponse to a TokenSet with calculated expiry time.
func (r *TokenResponse) ToTokenSet() *TokenSet {
	return &TokenSet{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(r.ExpiresIn) * time.Second),
		TokenType:    r.TokenType,
		Scope:        r.Scope,
		UserID:       r.UserID,
		TeamID:       r.TeamID,
	}
}

// AuthorizationState holds state for the OAuth flow.
type AuthorizationState struct {
	// State is the CSRF protection token
	State string

	// CodeVerifier is for PKCE (Proof Key for Code Exchange)
	CodeVerifier string

	// CreatedAt tracks when the state was created for expiry
	CreatedAt time.Time
}

// LoadConfigFromEnv loads OAuth configuration from environment variables.
func LoadConfigFromEnv() *Config {
	config := &Config{
		ClientID:     os.Getenv("MIRO_CLIENT_ID"),
		ClientSecret: os.Getenv("MIRO_CLIENT_SECRET"),
		RedirectURI:  os.Getenv("MIRO_REDIRECT_URI"),
		Scopes:       DefaultScopes,
	}

	if config.RedirectURI == "" {
		config.RedirectURI = "http://localhost:8089/callback"
	}

	// Token store path
	config.TokenStorePath = os.Getenv("MIRO_TOKEN_PATH")
	if config.TokenStorePath == "" {
		home, _ := os.UserHomeDir()
		config.TokenStorePath = home + "/.miro/tokens.json"
	}

	return config
}

// IsConfigured returns true if OAuth credentials are set.
func (c *Config) IsConfigured() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}

// AuthError represents an OAuth-related error.
type AuthError struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

func (e *AuthError) Error() string {
	if e.Description != "" {
		return e.Code + ": " + e.Description
	}
	return e.Code
}
