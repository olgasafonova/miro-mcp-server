package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/olgasafonova/miro-mcp-server/miro/oauth"
)

// =============================================================================
// auth subcommand tests
//
// Only the paths that stay local are covered: they read or delete the token
// file and print. Paths that call Miro (a real login, a token refresh, revoking
// a stored token) or end in os.Exit are left out; the oauth package tests the
// flow behind them with a fake browser and a fake token endpoint.
// =============================================================================

// authTestConfig points the OAuth config at a temp token file.
func authTestConfig(t *testing.T) *oauth.Config {
	t.Helper()
	return &oauth.Config{
		ClientID:       "test-client",
		ClientSecret:   "test-client-credential",
		RedirectURI:    "http://localhost:8089/callback",
		TokenStorePath: filepath.Join(t.TempDir(), "tokens.json"),
	}
}

func saveTokens(t *testing.T, cfg *oauth.Config, tokens *oauth.TokenSet) {
	t.Helper()
	if err := oauth.NewFileTokenStore(cfg.TokenStorePath).Save(context.Background(), tokens); err != nil {
		t.Fatalf("saving tokens: %v", err)
	}
}

func TestAuthCommands_RegistersAllSubcommands(t *testing.T) {
	for _, name := range []string{"login", "status", "logout"} {
		if authCommands[name] == nil {
			t.Errorf("auth subcommand %q is not registered", name)
		}
	}
}

func TestRunAuthStatus_LoggedIn(t *testing.T) {
	cfg := authTestConfig(t)
	expires := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	saveTokens(t, cfg, &oauth.TokenSet{UserID: "user-42", TeamID: "team-7", ExpiresAt: expires})

	var err error
	out := captureStdout(t, func() {
		err = runAuthStatus(context.Background(), oauth.NewAuthFlow(cfg, quietLogger()), cfg)
	})
	if err != nil {
		t.Fatalf("runAuthStatus: %v", err)
	}
	for _, want := range []string{
		"Authentication status: Logged in",
		"User ID: user-42",
		"Team ID: team-7",
		"Token expires: " + expires.Format(time.RFC3339),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRunAuthLogout_DeletesTokenFile(t *testing.T) {
	cfg := authTestConfig(t)
	// No access or refresh token, so Logout has nothing to revoke and makes
	// no network call; it only deletes the file.
	saveTokens(t, cfg, &oauth.TokenSet{UserID: "user-42", ExpiresAt: time.Now().Add(time.Hour)})

	var err error
	out := captureStdout(t, func() {
		err = runAuthLogout(context.Background(), oauth.NewAuthFlow(cfg, quietLogger()), cfg)
	})
	if err != nil {
		t.Fatalf("runAuthLogout: %v", err)
	}
	if !strings.Contains(out, "Logged out successfully.") {
		t.Errorf("unexpected output: %q", out)
	}
	if _, statErr := os.Stat(cfg.TokenStorePath); !os.IsNotExist(statErr) {
		t.Errorf("token file still exists after logout (stat err: %v)", statErr)
	}
}

func TestRunAuthLogout_WhenNotLoggedIn(t *testing.T) {
	cfg := authTestConfig(t)

	var err error
	captureStdout(t, func() {
		err = runAuthLogout(context.Background(), oauth.NewAuthFlow(cfg, quietLogger()), cfg)
	})
	if err != nil {
		t.Fatalf("logout with no saved tokens should succeed, got %v", err)
	}
}

func TestRunAuthLogin_ReportsFlowError(t *testing.T) {
	cfg := authTestConfig(t)
	cfg.RedirectURI = "://not a url"

	var err error
	captureStdout(t, func() {
		err = runAuthLogin(context.Background(), oauth.NewAuthFlow(cfg, quietLogger()), cfg)
	})
	if err == nil || !strings.HasPrefix(err.Error(), "login failed:") {
		t.Fatalf("expected a wrapped login error, got %v", err)
	}
}

func TestRunAuthCommand_NoArgsPrintsUsage(t *testing.T) {
	t.Setenv("MIRO_CLIENT_ID", "test-client")
	t.Setenv("MIRO_CLIENT_SECRET", "test-client-credential")
	t.Setenv("MIRO_TOKEN_PATH", filepath.Join(t.TempDir(), "tokens.json"))

	out := captureStdout(t, func() { runAuthCommand(nil) })
	if !strings.Contains(out, "Usage: miro-mcp-server auth <command>") {
		t.Errorf("expected usage text, got:\n%s", out)
	}
}

func TestRunAuthCommand_DispatchesSubcommand(t *testing.T) {
	t.Setenv("MIRO_CLIENT_ID", "test-client")
	t.Setenv("MIRO_CLIENT_SECRET", "test-client-credential")
	t.Setenv("MIRO_TOKEN_PATH", filepath.Join(t.TempDir(), "tokens.json"))

	out := captureStdout(t, func() { runAuthCommand([]string{"logout"}) })
	if !strings.Contains(out, "Logged out successfully.") {
		t.Errorf("expected logout to run, got:\n%s", out)
	}
}
