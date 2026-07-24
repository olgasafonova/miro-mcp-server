package oauth

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// AuthFlow refresh / exchange tests
//
// Covers the branches TestAuthFlowStatus_* leaves open: an expired token that
// DOES carry a refresh token (success and failure), and exchangeAndStore.
// Login, authorize, and openBrowser are deliberately not covered here — they
// launch a real browser and bind a real callback port.
// =============================================================================

// newTestAuthFlow builds an AuthFlow whose provider talks to handler instead
// of the network, backed by an in-memory token store seeded with tokens.
func newTestAuthFlow(t *testing.T, tokens *TokenSet, handler func(*http.Request) *http.Response) (*AuthFlow, *MemoryTokenStore) {
	t.Helper()

	config := &Config{
		ClientID:       "test-id",
		ClientSecret:   "test-secret",
		RedirectURI:    "http://localhost:8089/callback",
		TokenStorePath: filepath.Join(t.TempDir(), "tokens.json"),
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	provider := NewProvider(config)
	provider.httpClient = &http.Client{Transport: &mockRoundTripper{handler: handler}}

	store := NewMemoryTokenStore()
	if tokens != nil {
		if err := store.Save(context.Background(), tokens); err != nil {
			t.Fatalf("seeding token store: %v", err)
		}
	}

	return &AuthFlow{
		provider:   provider,
		tokenStore: store,
		logger:     logger,
		config:     config,
	}, store
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func TestAuthFlowStatus_RefreshesExpiredToken(t *testing.T) {
	expired := &TokenSet{
		AccessToken:  "stale-access-token",
		RefreshToken: "good-refresh-token",
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
		UserID:       "user-123",
	}

	var refreshCalled bool
	flow, store := newTestAuthFlow(t, expired, func(_ *http.Request) *http.Response {
		refreshCalled = true
		return jsonResponse(http.StatusOK, `{
			"access_token": "fresh-access-token",
			"refresh_token": "fresh-refresh-token",
			"expires_in": 3600,
			"token_type": "bearer",
			"user_id": "user-123"
		}`)
	})

	got, err := flow.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !refreshCalled {
		t.Error("expected the provider refresh endpoint to be called")
	}
	if got.AccessToken != "fresh-access-token" {
		t.Errorf("AccessToken = %q, want fresh-access-token", got.AccessToken)
	}

	// The refreshed set must be persisted, not just returned.
	saved, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("loading saved tokens: %v", err)
	}
	if saved.AccessToken != "fresh-access-token" {
		t.Errorf("persisted AccessToken = %q, want fresh-access-token", saved.AccessToken)
	}
}

func TestAuthFlowStatus_RefreshFailureIsReported(t *testing.T) {
	expired := &TokenSet{
		AccessToken:  "stale-access-token",
		RefreshToken: "revoked-refresh-token",
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
	}

	flow, _ := newTestAuthFlow(t, expired, func(_ *http.Request) *http.Response {
		return jsonResponse(http.StatusBadRequest, `{"error":"invalid_grant"}`)
	})

	_, err := flow.Status(context.Background())
	if err == nil {
		t.Fatal("expected an error when refresh fails")
	}
	if !strings.Contains(err.Error(), "token refresh failed") {
		t.Errorf("error = %v, want token refresh failed", err)
	}
}

func TestAuthFlowExchangeAndStore_Success(t *testing.T) {
	flow, store := newTestAuthFlow(t, nil, func(_ *http.Request) *http.Response {
		return jsonResponse(http.StatusOK, `{
			"access_token": "exchanged-access-token",
			"refresh_token": "exchanged-refresh-token",
			"expires_in": 3600,
			"token_type": "bearer",
			"user_id": "user-456",
			"team_id": "team-789"
		}`)
	})

	state := &AuthorizationState{State: "state-abc", CodeVerifier: "verifier-xyz"}
	tokens, err := flow.exchangeAndStore(context.Background(), "auth-code", state)
	if err != nil {
		t.Fatalf("exchangeAndStore() error = %v", err)
	}
	if tokens.AccessToken != "exchanged-access-token" {
		t.Errorf("AccessToken = %q, want exchanged-access-token", tokens.AccessToken)
	}
	if tokens.UserID != "user-456" {
		t.Errorf("UserID = %q, want user-456", tokens.UserID)
	}

	saved, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("loading saved tokens: %v", err)
	}
	if saved.AccessToken != "exchanged-access-token" {
		t.Errorf("persisted AccessToken = %q, want exchanged-access-token", saved.AccessToken)
	}
}

func TestAuthFlowExchangeAndStore_ExchangeFails(t *testing.T) {
	flow, _ := newTestAuthFlow(t, nil, func(_ *http.Request) *http.Response {
		return jsonResponse(http.StatusUnauthorized, `{"error":"invalid_client"}`)
	})

	state := &AuthorizationState{State: "state-abc", CodeVerifier: "verifier-xyz"}
	_, err := flow.exchangeAndStore(context.Background(), "auth-code", state)
	if err == nil {
		t.Fatal("expected an error when the exchange fails")
	}
	if !strings.Contains(err.Error(), "token exchange failed") {
		t.Errorf("error = %v, want token exchange failed", err)
	}
}

func TestAuthFlowRevokeIfPresent(t *testing.T) {
	t.Run("empty token is a no-op", func(t *testing.T) {
		var called bool
		flow, _ := newTestAuthFlow(t, nil, func(_ *http.Request) *http.Response {
			called = true
			return jsonResponse(http.StatusOK, `{}`)
		})

		flow.revokeIfPresent(context.Background(), "", "access token")
		if called {
			t.Error("an empty token must not reach the revoke endpoint")
		}
	})

	t.Run("revoke failure is logged, not propagated", func(t *testing.T) {
		flow, _ := newTestAuthFlow(t, nil, func(_ *http.Request) *http.Response {
			return jsonResponse(http.StatusInternalServerError, `{"error":"server_error"}`)
		})

		// No return value to assert on; the contract is that this does not panic
		// and does not propagate the failure.
		flow.revokeIfPresent(context.Background(), "some-token", "access token")
	})
}
