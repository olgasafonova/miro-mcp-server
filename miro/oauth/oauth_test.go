package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
	// Create mock token server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken:  "mock-access-token",
			RefreshToken: "mock-refresh-token",
			ExpiresIn:    3600,
			TokenType:    "bearer",
			Scope:        "boards:read",
		})
	}))
	defer server.Close()

	// This test would need to mock the token URL, which is hardcoded
	// For now, just verify the provider can be created
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost:8089/callback",
	}
	provider := NewProvider(config)

	if provider == nil {
		t.Fatal("provider should not be nil")
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
			expected:    ":8089",
		},
		{
			name:        "without port (http)",
			redirectURI: "http://localhost/callback",
			expected:    ":80",
		},
		{
			name:        "without port (https)",
			redirectURI: "https://localhost/callback",
			expected:    ":443",
		},
		{
			name:        "custom port",
			redirectURI: "http://127.0.0.1:9999/oauth",
			expected:    ":9999",
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
