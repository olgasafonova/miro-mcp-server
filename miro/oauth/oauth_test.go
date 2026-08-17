package oauth

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

// testLogger returns a quiet logger for tests.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// =============================================================================
// Config Tests
// =============================================================================

func TestLoadConfigFromEnv(t *testing.T) {
	// Clear any existing env vars
	os.Unsetenv("MIRO_CLIENT_ID")
	os.Unsetenv("MIRO_CLIENT_SECRET")
	os.Unsetenv("MIRO_REDIRECT_URI")
	os.Unsetenv("MIRO_TOKEN_PATH")

	config := LoadConfigFromEnv()

	if config.ClientID != "" {
		t.Errorf("expected empty ClientID, got %q", config.ClientID)
	}

	// Check defaults
	if config.RedirectURI != "http://localhost:8089/callback" {
		t.Errorf("expected default redirect URI, got %q", config.RedirectURI)
	}

	if len(config.Scopes) == 0 {
		t.Error("expected default scopes")
	}
}

func TestLoadConfigFromEnvWithValues(t *testing.T) {
	os.Setenv("MIRO_CLIENT_ID", "test-client-id")
	os.Setenv("MIRO_CLIENT_SECRET", "test-client-secret")
	os.Setenv("MIRO_REDIRECT_URI", "http://custom:9999/cb")
	defer func() {
		os.Unsetenv("MIRO_CLIENT_ID")
		os.Unsetenv("MIRO_CLIENT_SECRET")
		os.Unsetenv("MIRO_REDIRECT_URI")
	}()

	config := LoadConfigFromEnv()

	if config.ClientID != "test-client-id" {
		t.Errorf("expected test-client-id, got %q", config.ClientID)
	}
	if config.ClientSecret != "test-client-secret" {
		t.Errorf("expected test-client-secret, got %q", config.ClientSecret)
	}
	if config.RedirectURI != "http://custom:9999/cb" {
		t.Errorf("expected custom redirect URI, got %q", config.RedirectURI)
	}
}

func TestConfigIsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "empty config",
			config:   &Config{},
			expected: false,
		},
		{
			name:     "only client id",
			config:   &Config{ClientID: "id"},
			expected: false,
		},
		{
			name:     "only client secret",
			config:   &Config{ClientSecret: "secret"},
			expected: false,
		},
		{
			name:     "fully configured",
			config:   &Config{ClientID: "id", ClientSecret: "secret"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsConfigured(); got != tt.expected {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// TokenSet Tests
// =============================================================================

func TestTokenSetIsExpired(t *testing.T) {
	tests := []struct {
		name     string
		tokens   *TokenSet
		expected bool
	}{
		{
			name:     "expired token",
			tokens:   &TokenSet{ExpiresAt: time.Now().Add(-1 * time.Hour)},
			expected: true,
		},
		{
			name:     "valid token",
			tokens:   &TokenSet{ExpiresAt: time.Now().Add(1 * time.Hour)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tokens.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTokenSetNeedsRefresh(t *testing.T) {
	tests := []struct {
		name     string
		tokens   *TokenSet
		expected bool
	}{
		{
			name:     "expires in 1 minute (needs refresh)",
			tokens:   &TokenSet{ExpiresAt: time.Now().Add(1 * time.Minute)},
			expected: true,
		},
		{
			name:     "expires in 4 minutes (needs refresh)",
			tokens:   &TokenSet{ExpiresAt: time.Now().Add(4 * time.Minute)},
			expected: true,
		},
		{
			name:     "expires in 10 minutes (no refresh needed)",
			tokens:   &TokenSet{ExpiresAt: time.Now().Add(10 * time.Minute)},
			expected: false,
		},
		{
			name:     "expires in 1 hour (no refresh needed)",
			tokens:   &TokenSet{ExpiresAt: time.Now().Add(1 * time.Hour)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tokens.NeedsRefresh(); got != tt.expected {
				t.Errorf("NeedsRefresh() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTokenResponseToTokenSet(t *testing.T) {
	resp := &TokenResponse{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		ExpiresIn:    3600, // 1 hour
		TokenType:    "bearer",
		Scope:        "boards:read boards:write",
		UserID:       "user-789",
		TeamID:       "team-abc",
	}

	tokens := resp.ToTokenSet()

	if tokens.AccessToken != "access-123" {
		t.Errorf("AccessToken = %q, want %q", tokens.AccessToken, "access-123")
	}
	if tokens.RefreshToken != "refresh-456" {
		t.Errorf("RefreshToken = %q, want %q", tokens.RefreshToken, "refresh-456")
	}
	if tokens.UserID != "user-789" {
		t.Errorf("UserID = %q, want %q", tokens.UserID, "user-789")
	}

	// Check expiry is approximately 1 hour from now
	expectedExpiry := time.Now().Add(1 * time.Hour)
	if tokens.ExpiresAt.Before(expectedExpiry.Add(-1*time.Minute)) ||
		tokens.ExpiresAt.After(expectedExpiry.Add(1*time.Minute)) {
		t.Errorf("ExpiresAt = %v, expected around %v", tokens.ExpiresAt, expectedExpiry)
	}
}

// =============================================================================
// AuthError Tests
// =============================================================================

func TestAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      *AuthError
		expected string
	}{
		{
			name:     "with description",
			err:      &AuthError{Code: "invalid_grant", Description: "Token has expired"},
			expected: "invalid_grant: Token has expired",
		},
		{
			name:     "without description",
			err:      &AuthError{Code: "access_denied"},
			expected: "access_denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}
