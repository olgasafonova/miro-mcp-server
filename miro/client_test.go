package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// testLogger creates a silent logger for tests.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// testConfig creates a test configuration.
func testConfig() *Config {
	return &Config{
		AccessToken: "test-token",
		Timeout:     5 * time.Second,
		UserAgent:   "test-agent",
	}
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}

// ptrString returns a pointer to the given string.
func ptrString(s string) *string { return strPtr(s) }

// newTestClientWithServer creates a client pointing to a mock HTTP server.
func newTestClientWithServer(serverURL string) *Client {
	cfg := testConfig()
	client := NewClient(cfg, testLogger())
	client.baseURL = serverURL
	return client
}

// mockTokenRefresher is a test helper for token refresh tests.
type mockTokenRefresher struct {
	token string
	err   error
}

func (m *mockTokenRefresher) GetAccessToken(ctx context.Context) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.token, nil
}

func TestNewClient(t *testing.T) {
	cfg := testConfig()
	logger := testLogger()

	client := NewClient(cfg, logger)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.config != cfg {
		t.Error("config not set correctly")
	}
	if client.cache == nil {
		t.Error("cache not initialized")
	}
	if cap(client.semaphore) != MaxConcurrentRequests {
		t.Errorf("semaphore capacity = %d, want %d", cap(client.semaphore), MaxConcurrentRequests)
	}
}

func TestClientCache(t *testing.T) {
	cfg := testConfig()
	client := NewClient(cfg, testLogger())

	// Test setCache and getCached
	key := "test-key"
	data := "test-data"

	client.setCache(key, data)

	cached, ok := client.getCached(key)
	if !ok {
		t.Error("getCached returned false for existing key")
	}
	if cached != data {
		t.Errorf("getCached = %v, want %v", cached, data)
	}

	// Test missing key
	_, ok = client.getCached("missing")
	if ok {
		t.Error("getCached returned true for missing key")
	}
}

func TestValidateToken(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Now uses /boards?limit=1 instead of /users/me due to Miro API bug
		if r.URL.Path != "/boards" {
			t.Errorf("unexpected path: %s, want /boards", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing or incorrect Authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		// Return a boards response with owner info
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "board123",
					"name": "Test Board",
					"owner": map[string]string{
						"id":   "user123",
						"name": "Test User",
					},
				},
			},
		})
	}))
	defer server.Close()

	// Create client pointing to mock server
	cfg := testConfig()
	client := NewClient(cfg, testLogger())

	// Override base URL for testing (requires modifying the request method)
	// For now, we'll test the response parsing logic
	t.Run("parses valid response", func(t *testing.T) {
		// This test verifies the UserInfo struct can be parsed correctly
		jsonData := `{"id":"user123","name":"Test User","email":"test@example.com"}`
		var user UserInfo
		if err := json.Unmarshal([]byte(jsonData), &user); err != nil {
			t.Fatalf("failed to unmarshal UserInfo: %v", err)
		}
		if user.ID != "user123" {
			t.Errorf("ID = %q, want %q", user.ID, "user123")
		}
		if user.Name != "Test User" {
			t.Errorf("Name = %q, want %q", user.Name, "Test User")
		}
		if user.Email != "test@example.com" {
			t.Errorf("Email = %q, want %q", user.Email, "test@example.com")
		}
	})

	_ = client // Use client to avoid unused variable error
}

func TestListBoardsArgs(t *testing.T) {
	args := ListBoardsArgs{
		TeamID: "team123",
		Query:  "test",
		Limit:  10,
		Offset: "cursor123",
	}

	if args.TeamID != "team123" {
		t.Errorf("TeamID = %q, want %q", args.TeamID, "team123")
	}
	if args.Limit != 10 {
		t.Errorf("Limit = %d, want %d", args.Limit, 10)
	}
}

func TestRequestContextCancellation(t *testing.T) {
	cfg := testConfig()
	client := NewClient(cfg, testLogger())

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Attempt to make a request with cancelled context
	_, err := client.request(ctx, http.MethodGet, "/test", nil)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestCacheStats(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Initial stats should be zero
	stats := client.CacheStats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Errorf("initial stats should be zero, got hits=%d, misses=%d", stats.Hits, stats.Misses)
	}

	// Add item and verify stats update
	client.setCache("key1", "value1")
	client.getCached("key1") // hit
	client.getCached("key2") // miss

	stats = client.CacheStats()
	if stats.Hits != 1 {
		t.Errorf("CacheStats().Hits = %d, want 1", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("CacheStats().Misses = %d, want 1", stats.Misses)
	}
}

func TestInvalidateCache(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Add items to cache
	client.setCache("key1", "value1")
	client.setCache("key2", "value2")

	// Verify items exist
	if _, ok := client.getCached("key1"); !ok {
		t.Error("key1 should exist before invalidation")
	}

	// Invalidate cache
	client.InvalidateCache()

	// Verify items are gone
	if _, ok := client.getCached("key1"); ok {
		t.Error("key1 should not exist after invalidation")
	}
	if _, ok := client.getCached("key2"); ok {
		t.Error("key2 should not exist after invalidation")
	}
}

func TestCircuitBreakerStats(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Get stats before any requests
	stats := client.CircuitBreakerStats()
	if stats == nil {
		t.Fatal("CircuitBreakerStats should not return nil")
	}

	// Stats map should be empty initially (no circuit breakers created yet)
	if len(stats) != 0 {
		t.Errorf("initial stats should be empty, got %d entries", len(stats))
	}
}

func TestResetCircuitBreakers(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// This should not panic even when no circuit breakers exist
	client.ResetCircuitBreakers()

	// Verify stats are still accessible after reset
	stats := client.CircuitBreakerStats()
	if stats == nil {
		t.Error("CircuitBreakerStats should not return nil after reset")
	}
}

func TestRateLimiterStats(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	stats := client.RateLimiterStats()

	// Initial state should have default limit set
	if stats.CurrentState.Limit <= 0 {
		t.Errorf("initial CurrentState.Limit = %d, want > 0", stats.CurrentState.Limit)
	}
	if stats.TotalDelays != 0 {
		t.Errorf("initial TotalDelays = %d, want 0", stats.TotalDelays)
	}
	if stats.TotalRequests != 0 {
		t.Errorf("initial TotalRequests = %d, want 0", stats.TotalRequests)
	}
}

func TestResetRateLimiter(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Reset should not panic
	client.ResetRateLimiter()

	// After reset, stats should be zeroed
	stats := client.RateLimiterStats()
	if stats.TotalRequests != 0 {
		t.Errorf("after reset, TotalRequests = %d, want 0", stats.TotalRequests)
	}
	if stats.TotalDelays != 0 {
		t.Errorf("after reset, TotalDelays = %d, want 0", stats.TotalDelays)
	}
}

func TestWithTokenRefresher(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Create a mock token refresher
	mockRefresher := &mockTokenRefresher{token: "refreshed-token"}

	// Chain should work
	result := client.WithTokenRefresher(mockRefresher)
	if result != client {
		t.Error("WithTokenRefresher should return the same client for chaining")
	}

	// Token should now come from refresher
	token, err := client.getAccessToken(context.Background())
	if err != nil {
		t.Fatalf("getAccessToken failed: %v", err)
	}
	if token != "refreshed-token" {
		t.Errorf("token = %q, want 'refreshed-token'", token)
	}
}

func TestWithTokenRefresher_Error(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Create a failing mock token refresher
	mockRefresher := &mockTokenRefresher{err: fmt.Errorf("refresh failed")}
	client.WithTokenRefresher(mockRefresher)

	// Token retrieval should fail
	_, err := client.getAccessToken(context.Background())
	if err == nil {
		t.Error("expected error from failing refresher")
	}
	if !strings.Contains(err.Error(), "refresh failed") {
		t.Errorf("error = %q, want containing 'refresh failed'", err.Error())
	}
}

func TestSetCacheWithTTL(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Set with custom TTL
	client.setCacheWithTTL("ttl-key", "ttl-value", 100*time.Millisecond)

	// Should be retrievable immediately
	val, ok := client.getCached("ttl-key")
	if !ok {
		t.Error("setCacheWithTTL value should be retrievable")
	}
	if val != "ttl-value" {
		t.Errorf("cached value = %v, want 'ttl-value'", val)
	}

	// After TTL expires, should be gone
	time.Sleep(150 * time.Millisecond)
	if _, ok := client.getCached("ttl-key"); ok {
		t.Error("cached value should expire after TTL")
	}
}

func TestValidateToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/boards") {
			t.Errorf("expected /boards in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "board123",
					"name": "Test Board",
					"owner": map[string]interface{}{
						"id":   "owner123",
						"name": "John Doe",
					},
					"team": map[string]interface{}{
						"id":   "team123",
						"name": "Test Team",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ValidateToken(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "owner123" {
		t.Errorf("ID = %q, want 'owner123'", result.ID)
	}
	if result.Name != "John Doe" {
		t.Errorf("Name = %q, want 'John Doe'", result.Name)
	}
}

func TestValidateToken_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  401,
			"code":    "unauthorized",
			"message": "Invalid token",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ValidateToken(context.Background())

	if err == nil {
		t.Fatal("expected error for unauthorized response")
	}
	if !strings.Contains(err.Error(), "token validation failed") {
		t.Errorf("expected error to contain 'token validation failed', got: %v", err)
	}
}

func TestValidateToken_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ValidateToken(context.Background())

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse boards response") {
		t.Errorf("expected error about parsing, got: %v", err)
	}
}
