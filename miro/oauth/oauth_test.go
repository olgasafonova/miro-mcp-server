package oauth

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// mockRoundTripper allows mocking HTTP responses in tests
type mockRoundTripper struct {
	handler func(req *http.Request) *http.Response
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.handler(req), nil
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

// =============================================================================
// Provider Tests
// =============================================================================

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
		if !containsSubstring(url, expected) {
			t.Errorf("URL should contain %q, got %q", expected, url)
		}
	}
}

func TestProviderExchangeCode(t *testing.T) {
	// Create mock HTTP client using RoundTripper
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost:8089/callback",
	}
	provider := NewProvider(config)

	// Replace HTTP client with mock
	provider.httpClient = &http.Client{
		Transport: &mockRoundTripper{
			handler: func(req *http.Request) *http.Response {
				// Verify request
				if req.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", req.Method)
				}
				if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
					t.Errorf("wrong content type: %s", req.Header.Get("Content-Type"))
				}

				// Return success response
				body := `{
					"access_token": "mock-access-token",
					"refresh_token": "mock-refresh-token",
					"expires_in": 3600,
					"token_type": "bearer",
					"scope": "boards:read"
				}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}
			},
		},
	}

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
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost:8089/callback",
	}
	provider := NewProvider(config)

	// Mock error response
	provider.httpClient = &http.Client{
		Transport: &mockRoundTripper{
			handler: func(req *http.Request) *http.Response {
				body := `{"error": "invalid_grant", "error_description": "Code expired"}`
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}
			},
		},
	}

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
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost:8089/callback",
	}
	provider := NewProvider(config)

	// Mock invalid JSON response
	provider.httpClient = &http.Client{
		Transport: &mockRoundTripper{
			handler: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("not json")),
					Header:     make(http.Header),
				}
			},
		},
	}

	ctx := context.Background()
	_, err := provider.ExchangeCode(ctx, "test-code", "test-verifier")

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestProviderRefreshToken(t *testing.T) {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
	}
	provider := NewProvider(config)

	provider.httpClient = &http.Client{
		Transport: &mockRoundTripper{
			handler: func(req *http.Request) *http.Response {
				body := `{
					"access_token": "new-access-token",
					"refresh_token": "new-refresh-token",
					"expires_in": 3600,
					"token_type": "bearer"
				}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}
			},
		},
	}

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
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
	}
	provider := NewProvider(config)

	provider.httpClient = &http.Client{
		Transport: &mockRoundTripper{
			handler: func(req *http.Request) *http.Response {
				body := `{"error": "invalid_grant", "error_description": "Refresh token expired"}`
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}
			},
		},
	}

	ctx := context.Background()
	_, err := provider.RefreshToken(ctx, "expired-refresh-token")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProviderRevokeToken(t *testing.T) {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
	}
	provider := NewProvider(config)

	provider.httpClient = &http.Client{
		Transport: &mockRoundTripper{
			handler: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}
			},
		},
	}

	ctx := context.Background()
	err := provider.RevokeToken(ctx, "token-to-revoke")

	if err != nil {
		t.Fatalf("RevokeToken() error = %v", err)
	}
}

func TestProviderRevokeToken_Error(t *testing.T) {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
	}
	provider := NewProvider(config)

	provider.httpClient = &http.Client{
		Transport: &mockRoundTripper{
			handler: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("server error")),
					Header:     make(http.Header),
				}
			},
		},
	}

	ctx := context.Background()
	err := provider.RevokeToken(ctx, "token-to-revoke")

	if err == nil {
		t.Fatal("expected error")
	}
}

// =============================================================================
// TokenStore Tests
// =============================================================================

func TestFileTokenStore(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, ".miro", "tokens.json")

	store := NewFileTokenStore(tokenPath)
	ctx := context.Background()

	// Initially should not exist
	if store.Exists(ctx) {
		t.Error("store should not exist initially")
	}

	// Save tokens
	tokens := &TokenSet{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		UserID:       "test-user",
	}

	if err := store.Save(ctx, tokens); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Should exist now
	if !store.Exists(ctx) {
		t.Error("store should exist after save")
	}

	// Load tokens
	loaded, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.AccessToken != tokens.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, tokens.AccessToken)
	}
	if loaded.RefreshToken != tokens.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, tokens.RefreshToken)
	}
	if loaded.UserID != tokens.UserID {
		t.Errorf("UserID = %q, want %q", loaded.UserID, tokens.UserID)
	}

	// Delete tokens
	if err := store.Delete(ctx); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if store.Exists(ctx) {
		t.Error("store should not exist after delete")
	}

	// Load should fail after delete
	_, err = store.Load(ctx)
	if err == nil {
		t.Error("Load() should fail after delete")
	}
}

func TestMemoryTokenStore(t *testing.T) {
	store := NewMemoryTokenStore()
	ctx := context.Background()

	// Initially should not exist
	if store.Exists(ctx) {
		t.Error("store should not exist initially")
	}

	// Save tokens
	tokens := &TokenSet{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	if err := store.Save(ctx, tokens); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Should exist now
	if !store.Exists(ctx) {
		t.Error("store should exist after save")
	}

	// Load tokens
	loaded, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.AccessToken != tokens.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, tokens.AccessToken)
	}

	// Modify original should not affect stored copy
	tokens.AccessToken = "modified"
	loaded2, _ := store.Load(ctx)
	if loaded2.AccessToken == "modified" {
		t.Error("stored tokens should be a copy, not reference")
	}

	// Delete tokens
	if err := store.Delete(ctx); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if store.Exists(ctx) {
		t.Error("store should not exist after delete")
	}
}

// =============================================================================
// Server Tests
// =============================================================================

func TestGetCallbackPort(t *testing.T) {
	tests := []struct {
		name        string
		redirectURI string
		expected    string
		shouldError bool
	}{
		{
			name:        "with explicit port",
			redirectURI: "http://localhost:8089/callback",
			expected:    "127.0.0.1:8089",
		},
		{
			name:        "without port (http)",
			redirectURI: "http://localhost/callback",
			expected:    "127.0.0.1:80",
		},
		{
			name:        "without port (https)",
			redirectURI: "https://localhost/callback",
			expected:    "127.0.0.1:443",
		},
		{
			name:        "custom port",
			redirectURI: "http://127.0.0.1:9999/oauth",
			expected:    "127.0.0.1:9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := GetCallbackPort(tt.redirectURI)
			if tt.shouldError {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetCallbackPort() error = %v", err)
			}
			if port != tt.expected {
				t.Errorf("GetCallbackPort() = %q, want %q", port, tt.expected)
			}
		})
	}
}

// =============================================================================
// CallbackServer Tests
// =============================================================================

func TestNewCallbackServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test successful creation
	server, err := NewCallbackServer(":0", logger) // :0 picks a random available port
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}
	defer server.Stop(context.Background())

	if server.Addr() == "" {
		t.Error("server address should not be empty")
	}
}

func TestNewCallbackServer_InvalidAddress(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Try to bind to an invalid address
	_, err := NewCallbackServer("invalid:address:format", logger)
	if err == nil {
		t.Error("expected error for invalid address")
	}
}

func TestCallbackServer_HandleCallback_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	server, err := NewCallbackServer(":0", logger)
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}
	defer server.Stop(context.Background())

	server.Start()

	// Make a request to the callback endpoint with code and state
	addr := "http://" + server.Addr()
	resp, err := http.Get(addr + "/callback?code=test-auth-code&state=test-state")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Wait for the result
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := server.WaitForCallback(ctx, 2*time.Second)
	if err != nil {
		t.Fatalf("WaitForCallback() error = %v", err)
	}

	if result.Code != "test-auth-code" {
		t.Errorf("Code = %q, want %q", result.Code, "test-auth-code")
	}
	if result.State != "test-state" {
		t.Errorf("State = %q, want %q", result.State, "test-state")
	}
	if result.Error != nil {
		t.Errorf("Error should be nil, got %v", result.Error)
	}
}

func TestCallbackServer_HandleCallback_Error(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	server, err := NewCallbackServer(":0", logger)
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}
	defer server.Stop(context.Background())

	server.Start()

	// Make a request with an error
	addr := "http://" + server.Addr()
	resp, err := http.Get(addr + "/callback?error=access_denied&error_description=User+denied+access")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	// Wait for the result
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := server.WaitForCallback(ctx, 2*time.Second)
	if err != nil {
		t.Fatalf("WaitForCallback() error = %v", err)
	}

	if result.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if result.Error.Code != "access_denied" {
		t.Errorf("Error.Code = %q, want %q", result.Error.Code, "access_denied")
	}
	if result.Error.Description != "User denied access" {
		t.Errorf("Error.Description = %q, want %q", result.Error.Description, "User denied access")
	}
}

func TestCallbackServer_HandleCallback_MissingCode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	server, err := NewCallbackServer(":0", logger)
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}
	defer server.Stop(context.Background())

	server.Start()

	// Make a request without a code
	addr := "http://" + server.Addr()
	resp, err := http.Get(addr + "/callback?state=test-state")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	// Wait for the result
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := server.WaitForCallback(ctx, 2*time.Second)
	if err != nil {
		t.Fatalf("WaitForCallback() error = %v", err)
	}

	if result.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if result.Error.Code != "missing_code" {
		t.Errorf("Error.Code = %q, want %q", result.Error.Code, "missing_code")
	}
}

func TestCallbackServer_HandleRoot(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	server, err := NewCallbackServer(":0", logger)
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}
	defer server.Stop(context.Background())

	server.Start()

	// Make a request to root
	addr := "http://" + server.Addr()
	resp, err := http.Get(addr + "/")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/html; charset=utf-8")
	}
}

func TestCallbackServer_WaitForCallback_Timeout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	server, err := NewCallbackServer(":0", logger)
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}
	defer server.Stop(context.Background())

	server.Start()

	// Wait with a very short timeout (no callback will come)
	ctx := context.Background()
	_, err = server.WaitForCallback(ctx, 10*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error")
	}
	if !containsSubstring(err.Error(), "timed out") {
		t.Errorf("error should mention timeout, got %q", err.Error())
	}
}

// =============================================================================
// AuthFlow Tests
// =============================================================================

func TestNewAuthFlow(t *testing.T) {
	config := &Config{
		ClientID:       "test-id",
		ClientSecret:   "test-secret",
		RedirectURI:    "http://localhost:8089/callback",
		TokenStorePath: filepath.Join(t.TempDir(), "tokens.json"),
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	flow := NewAuthFlow(config, logger)
	if flow == nil {
		t.Fatal("NewAuthFlow returned nil")
	}
}

func TestAuthFlowStatus_NotLoggedIn(t *testing.T) {
	config := &Config{
		ClientID:       "test-id",
		ClientSecret:   "test-secret",
		RedirectURI:    "http://localhost:8089/callback",
		TokenStorePath: filepath.Join(t.TempDir(), "tokens.json"),
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	flow := NewAuthFlow(config, logger)
	ctx := context.Background()

	// Should fail when not logged in
	_, err := flow.Status(ctx)
	if err == nil {
		t.Error("expected error when not logged in")
	}
	if !containsSubstring(err.Error(), "not logged in") {
		t.Errorf("error should mention 'not logged in', got %q", err.Error())
	}
}

func TestAuthFlowStatus_ValidToken(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "tokens.json")

	config := &Config{
		ClientID:       "test-id",
		ClientSecret:   "test-secret",
		RedirectURI:    "http://localhost:8089/callback",
		TokenStorePath: tokenPath,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Pre-save valid tokens
	store := NewFileTokenStore(tokenPath)
	ctx := context.Background()
	tokens := &TokenSet{
		AccessToken:  "valid-access-token",
		RefreshToken: "valid-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour), // Valid for 1 hour
		UserID:       "user-123",
	}
	if err := store.Save(ctx, tokens); err != nil {
		t.Fatalf("failed to save tokens: %v", err)
	}

	flow := NewAuthFlow(config, logger)

	// Should return valid tokens
	result, err := flow.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if result.AccessToken != "valid-access-token" {
		t.Errorf("AccessToken = %q, want 'valid-access-token'", result.AccessToken)
	}
}

func TestAuthFlowStatus_ExpiredTokenNoRefresh(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "tokens.json")

	config := &Config{
		ClientID:       "test-id",
		ClientSecret:   "test-secret",
		RedirectURI:    "http://localhost:8089/callback",
		TokenStorePath: tokenPath,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Pre-save expired tokens without refresh token
	store := NewFileTokenStore(tokenPath)
	ctx := context.Background()
	tokens := &TokenSet{
		AccessToken:  "expired-access-token",
		RefreshToken: "",                             // No refresh token
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		UserID:       "user-123",
	}
	if err := store.Save(ctx, tokens); err != nil {
		t.Fatalf("failed to save tokens: %v", err)
	}

	flow := NewAuthFlow(config, logger)

	// Should fail because token expired and no refresh token
	_, err := flow.Status(ctx)
	if err == nil {
		t.Error("expected error for expired token without refresh")
	}
	if !containsSubstring(err.Error(), "expired") {
		t.Errorf("error should mention 'expired', got %q", err.Error())
	}
}

func TestAuthFlowLogout(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "tokens.json")

	config := &Config{
		ClientID:       "test-id",
		ClientSecret:   "test-secret",
		RedirectURI:    "http://localhost:8089/callback",
		TokenStorePath: tokenPath,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Pre-save tokens
	store := NewFileTokenStore(tokenPath)
	ctx := context.Background()
	tokens := &TokenSet{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		UserID:       "user-123",
	}
	if err := store.Save(ctx, tokens); err != nil {
		t.Fatalf("failed to save tokens: %v", err)
	}

	flow := NewAuthFlow(config, logger)

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
	config := &Config{
		ClientID:       "test-id",
		ClientSecret:   "test-secret",
		RedirectURI:    "http://localhost:8089/callback",
		TokenStorePath: filepath.Join(t.TempDir(), "tokens.json"),
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	flow := NewAuthFlow(config, logger)
	ctx := context.Background()

	// Should succeed even when not logged in (nothing to delete)
	err := flow.Logout(ctx)
	if err != nil {
		t.Fatalf("Logout() should succeed even when not logged in, got error: %v", err)
	}
}

func TestAuthFlowGetAccessToken_NotLoggedIn(t *testing.T) {
	config := &Config{
		ClientID:       "test-id",
		ClientSecret:   "test-secret",
		RedirectURI:    "http://localhost:8089/callback",
		TokenStorePath: filepath.Join(t.TempDir(), "tokens.json"),
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	flow := NewAuthFlow(config, logger)
	ctx := context.Background()

	// Should fail when not logged in
	_, err := flow.GetAccessToken(ctx)
	if err == nil {
		t.Error("expected error when not logged in")
	}
}

func TestAuthFlowGetAccessToken_ValidToken(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "tokens.json")

	config := &Config{
		ClientID:       "test-id",
		ClientSecret:   "test-secret",
		RedirectURI:    "http://localhost:8089/callback",
		TokenStorePath: tokenPath,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Pre-save valid tokens
	store := NewFileTokenStore(tokenPath)
	ctx := context.Background()
	tokens := &TokenSet{
		AccessToken:  "my-access-token",
		RefreshToken: "my-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		UserID:       "user-123",
	}
	if err := store.Save(ctx, tokens); err != nil {
		t.Fatalf("failed to save tokens: %v", err)
	}

	flow := NewAuthFlow(config, logger)

	// Should return the access token
	token, err := flow.GetAccessToken(ctx)
	if err != nil {
		t.Fatalf("GetAccessToken() error = %v", err)
	}
	if token != "my-access-token" {
		t.Errorf("AccessToken = %q, want 'my-access-token'", token)
	}
}

func TestOpenBrowser(t *testing.T) {
	// Just test that openBrowser doesn't panic on supported platforms
	// We won't actually open a browser in tests
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		// The function should return nil or an error, but not panic
		// We can't really test this without opening a browser
		t.Skip("skipping browser test in CI/automated environment")
	}
}

// =============================================================================
// XSS Prevention Tests
// =============================================================================

func TestWriteErrorPage_XSSPrevention(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	server, err := NewCallbackServer("127.0.0.1:0", logger)
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}
	defer server.Stop(context.Background())

	server.Start()

	// Send a callback with XSS payload in error params
	addr := "http://" + server.Addr()
	xssPayload := `<script>alert('xss')</script>`
	resp, err := http.Get(addr + "/callback?error=" + xssPayload + "&error_description=" + xssPayload)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// The raw script tag should NOT appear in the response
	if strings.Contains(bodyStr, "<script>") {
		t.Error("response contains unescaped <script> tag; XSS vulnerability present")
	}

	// The escaped version should appear
	if !strings.Contains(bodyStr, "&lt;script&gt;") {
		t.Error("response should contain HTML-escaped script tag")
	}
}

func TestGetCallbackPort_BindsToLocalhost(t *testing.T) {
	tests := []struct {
		name        string
		redirectURI string
	}{
		{"http with port", "http://localhost:8089/callback"},
		{"https without port", "https://example.com/callback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := GetCallbackPort(tt.redirectURI)
			if err != nil {
				t.Fatalf("GetCallbackPort() error = %v", err)
			}
			if !strings.HasPrefix(addr, "127.0.0.1:") {
				t.Errorf("GetCallbackPort() = %q, should start with '127.0.0.1:'", addr)
			}
		})
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
