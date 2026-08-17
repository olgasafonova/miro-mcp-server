package oauth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// =============================================================================
// Provider Tests
// =============================================================================

// mockRoundTripper allows mocking HTTP responses in tests
type mockRoundTripper struct {
	handler func(req *http.Request) *http.Response
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.handler(req), nil
}

// httpResponse builds a bare HTTP response with the given status and body.
func httpResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// exchangeTestConfig is the standard config for code-exchange tests.
func exchangeTestConfig() *Config {
	return &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost:8089/callback",
	}
}

// newMockedProvider builds a Provider whose HTTP client is served by handler.
func newMockedProvider(config *Config, handler func(req *http.Request) *http.Response) *Provider {
	provider := NewProvider(config)
	provider.httpClient = &http.Client{
		Transport: &mockRoundTripper{handler: handler},
	}
	return provider
}

// newStubProvider builds a Provider that always answers with one canned response.
func newStubProvider(status int, body string) *Provider {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
	}
	return newMockedProvider(config, func(req *http.Request) *http.Response {
		return httpResponse(status, body)
	})
}

func TestProviderGenerateAuthorizationState(t *testing.T) {
	config := &Config{
		ClientID:    "test-id",
		RedirectURI: "http://localhost:8089/callback",
		Scopes:      DefaultScopes,
	}
	provider := NewProvider(config)

	state, err := provider.GenerateAuthorizationState()
	if err != nil {
		t.Fatalf("GenerateAuthorizationState() error = %v", err)
	}

	if state.State == "" {
		t.Error("State should not be empty")
	}
	if state.CodeVerifier == "" {
		t.Error("CodeVerifier should not be empty")
	}
	if state.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	// State and code verifier should be unique
	state2, _ := provider.GenerateAuthorizationState()
	if state.State == state2.State {
		t.Error("State should be unique")
	}
	if state.CodeVerifier == state2.CodeVerifier {
		t.Error("CodeVerifier should be unique")
	}
}

func TestProviderGetAuthorizationURL(t *testing.T) {
	config := &Config{
		ClientID:    "test-client-id",
		RedirectURI: "http://localhost:8089/callback",
		Scopes:      []string{"boards:read", "boards:write"},
	}
	provider := NewProvider(config)

	state := &AuthorizationState{
		State:        "test-state",
		CodeVerifier: "test-verifier",
	}

	url := provider.GetAuthorizationURL(state)

	// Check URL contains required parameters
	if url == "" {
		t.Fatal("URL should not be empty")
	}

	expectedContains := []string{
		"https://miro.com/oauth/authorize",
		"response_type=code",
		"client_id=test-client-id",
		"redirect_uri=http",
		"state=test-state",
		"code_challenge=",
		"code_challenge_method=S256",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(url, expected) {
			t.Errorf("URL should contain %q, got %q", expected, url)
		}
	}
}

func TestProviderExchangeCode(t *testing.T) {
	provider := newMockedProvider(exchangeTestConfig(), func(req *http.Request) *http.Response {
		// Verify request
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("wrong content type: %s", req.Header.Get("Content-Type"))
		}

		// Return success response
		return httpResponse(http.StatusOK, `{
			"access_token": "mock-access-token",
			"refresh_token": "mock-refresh-token",
			"expires_in": 3600,
			"token_type": "bearer",
			"scope": "boards:read"
		}`)
	})

	ctx := context.Background()
	tokens, err := provider.ExchangeCode(ctx, "test-code", "test-verifier")

	if err != nil {
		t.Fatalf("ExchangeCode() error = %v", err)
	}
	if tokens.AccessToken != "mock-access-token" {
		t.Errorf("AccessToken = %q, want 'mock-access-token'", tokens.AccessToken)
	}
	if tokens.RefreshToken != "mock-refresh-token" {
		t.Errorf("RefreshToken = %q, want 'mock-refresh-token'", tokens.RefreshToken)
	}
}

func TestProviderExchangeCode_Error(t *testing.T) {
	provider := newMockedProvider(exchangeTestConfig(), func(req *http.Request) *http.Response {
		return httpResponse(http.StatusBadRequest, `{"error": "invalid_grant", "error_description": "Code expired"}`)
	})

	ctx := context.Background()
	_, err := provider.ExchangeCode(ctx, "expired-code", "test-verifier")

	if err == nil {
		t.Fatal("expected error")
	}
	authErr, ok := err.(*AuthError)
	if !ok {
		t.Fatalf("expected AuthError, got %T", err)
	}
	if authErr.Code != "invalid_grant" {
		t.Errorf("Error.Code = %q, want 'invalid_grant'", authErr.Code)
	}
}

func TestProviderExchangeCode_InvalidJSON(t *testing.T) {
	provider := newMockedProvider(exchangeTestConfig(), func(req *http.Request) *http.Response {
		return httpResponse(http.StatusOK, "not json")
	})

	ctx := context.Background()
	_, err := provider.ExchangeCode(ctx, "test-code", "test-verifier")

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestProviderRefreshToken(t *testing.T) {
	provider := newStubProvider(http.StatusOK, `{
		"access_token": "new-access-token",
		"refresh_token": "new-refresh-token",
		"expires_in": 3600,
		"token_type": "bearer"
	}`)

	ctx := context.Background()
	tokens, err := provider.RefreshToken(ctx, "old-refresh-token")

	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if tokens.AccessToken != "new-access-token" {
		t.Errorf("AccessToken = %q, want 'new-access-token'", tokens.AccessToken)
	}
}

func TestProviderRefreshToken_Error(t *testing.T) {
	provider := newStubProvider(http.StatusBadRequest, `{"error": "invalid_grant", "error_description": "Refresh token expired"}`)

	ctx := context.Background()
	_, err := provider.RefreshToken(ctx, "expired-refresh-token")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProviderRevokeToken(t *testing.T) {
	provider := newStubProvider(http.StatusOK, "")

	ctx := context.Background()
	err := provider.RevokeToken(ctx, "token-to-revoke")

	if err != nil {
		t.Fatalf("RevokeToken() error = %v", err)
	}
}

func TestProviderRevokeToken_Error(t *testing.T) {
	provider := newStubProvider(http.StatusInternalServerError, "server error")

	ctx := context.Background()
	err := provider.RevokeToken(ctx, "token-to-revoke")

	if err == nil {
		t.Fatal("expected error")
	}
}
