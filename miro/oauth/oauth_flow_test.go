package oauth

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// AuthFlow Tests
// =============================================================================

// flowTestConfig builds the standard AuthFlow config with a temp token store.
func flowTestConfig(t *testing.T) *Config {
	t.Helper()
	return &Config{
		ClientID:       "test-id",
		ClientSecret:   "test-secret",
		RedirectURI:    "http://localhost:8089/callback",
		TokenStorePath: filepath.Join(t.TempDir(), "tokens.json"),
	}
}

// seedStoredTokens pre-saves tokens into the config's file token store.
func seedStoredTokens(t *testing.T, config *Config, tokens *TokenSet) *FileTokenStore {
	t.Helper()
	store := NewFileTokenStore(config.TokenStorePath)
	if err := store.Save(context.Background(), tokens); err != nil {
		t.Fatalf("failed to save tokens: %v", err)
	}
	return store
}

// newLoggedInFlow builds an AuthFlow whose store already holds valid tokens.
func newLoggedInFlow(t *testing.T, accessToken, refreshToken string) *AuthFlow {
	t.Helper()
	config := flowTestConfig(t)
	seedStoredTokens(t, config, &TokenSet{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(1 * time.Hour), // Valid for 1 hour
		UserID:       "user-123",
	})
	return NewAuthFlow(config, testLogger())
}

func TestNewAuthFlow(t *testing.T) {
	flow := NewAuthFlow(flowTestConfig(t), testLogger())
	if flow == nil {
		t.Fatal("NewAuthFlow returned nil")
	}
}

func TestAuthFlowStatus_NotLoggedIn(t *testing.T) {
	flow := NewAuthFlow(flowTestConfig(t), testLogger())
	ctx := context.Background()

	// Should fail when not logged in
	_, err := flow.Status(ctx)
	if err == nil {
		t.Error("expected error when not logged in")
	}
	if !strings.Contains(err.Error(), "not logged in") {
		t.Errorf("error should mention 'not logged in', got %q", err.Error())
	}
}

// mustSucceed fails fatally when a flow call errored.
func mustSucceed(t *testing.T, op string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s error = %v", op, err)
	}
}

func TestAuthFlowStatus_ValidToken(t *testing.T) {
	flow := newLoggedInFlow(t, "valid-access-token", "valid-refresh-token")

	// Should return valid tokens
	result, err := flow.Status(context.Background())
	mustSucceed(t, "Status()", err)
	if result.AccessToken != "valid-access-token" {
		t.Errorf("AccessToken = %q, want 'valid-access-token'", result.AccessToken)
	}
}

func TestAuthFlowStatus_ExpiredTokenNoRefresh(t *testing.T) {
	config := flowTestConfig(t)
	ctx := context.Background()

	// Pre-save expired tokens without refresh token
	seedStoredTokens(t, config, &TokenSet{
		AccessToken:  "expired-access-token",
		RefreshToken: "",                             // No refresh token
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		UserID:       "user-123",
	})

	flow := NewAuthFlow(config, testLogger())

	// Should fail because token expired and no refresh token
	_, err := flow.Status(ctx)
	if err == nil {
		t.Error("expected error for expired token without refresh")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("error should mention 'expired', got %q", err.Error())
	}
}

func TestAuthFlowLogout(t *testing.T) {
	config := flowTestConfig(t)
	ctx := context.Background()

	// Pre-save tokens
	store := seedStoredTokens(t, config, &TokenSet{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		UserID:       "user-123",
	})

	flow := NewAuthFlow(config, testLogger())

	// Note: Logout will try to revoke tokens via API which will fail,
	// but it should still delete local tokens
	err := flow.Logout(ctx)
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	// Tokens should be deleted
	if store.Exists(ctx) {
		t.Error("tokens should be deleted after logout")
	}
}

func TestAuthFlowLogout_NotLoggedIn(t *testing.T) {
	flow := NewAuthFlow(flowTestConfig(t), testLogger())
	ctx := context.Background()

	// Should succeed even when not logged in (nothing to delete)
	err := flow.Logout(ctx)
	if err != nil {
		t.Fatalf("Logout() should succeed even when not logged in, got error: %v", err)
	}
}

func TestAuthFlowGetAccessToken_NotLoggedIn(t *testing.T) {
	flow := NewAuthFlow(flowTestConfig(t), testLogger())
	ctx := context.Background()

	// Should fail when not logged in
	_, err := flow.GetAccessToken(ctx)
	if err == nil {
		t.Error("expected error when not logged in")
	}
}

func TestAuthFlowGetAccessToken_ValidToken(t *testing.T) {
	flow := newLoggedInFlow(t, "my-access-token", "my-refresh-token")

	// Should return the access token
	token, err := flow.GetAccessToken(context.Background())
	mustSucceed(t, "GetAccessToken()", err)
	if token != "my-access-token" {
		t.Errorf("AccessToken = %q, want 'my-access-token'", token)
	}
}

func TestOpenBrowser(t *testing.T) {
	// Just test that openBrowser doesn't panic on supported platforms
	// We won't actually open a browser in tests
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		// The function should return nil or an error, but not panic
		// We can't really test this without opening a browser
		t.Skip("skipping browser test in CI/automated environment")
	}
}
