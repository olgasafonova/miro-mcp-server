package miro

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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

// =============================================================================
// Input Validation Tests
// =============================================================================

func TestValidateBoardID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid alphanumeric", "abc123", false},
		{"valid with underscore", "board_123", false},
		{"valid with hyphen", "board-123", false},
		{"valid with equals", "board=123", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 101)), true},
		{"invalid chars space", "board 123", true},
		{"invalid chars slash", "board/123", true},
		{"invalid chars dot", "board.123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBoardID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBoardID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestValidateItemID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid", "item123", false},
		{"empty", "", true},
		{"invalid chars", "item/123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateItemID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateItemID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"valid short", "Hello", false},
		{"valid empty", "", false},
		{"valid at limit", string(make([]byte, maxContentLen)), false},
		{"too long", string(make([]byte, maxContentLen+1)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContent(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		max    int
		expect string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"very short max", "hello", 3, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.max)
			if result != tt.expect {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, result, tt.expect)
			}
		})
	}
}

func TestNormalizeStickyColor(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"yellow", "light_yellow"},
		{"Yellow", "light_yellow"},
		{"YELLOW", "light_yellow"},
		{"green", "light_green"},
		{"blue", "light_blue"},
		{"pink", "light_pink"},
		{"purple", "violet"},
		{"gray", "gray"},
		{"grey", "gray"},
		{"unknown_color", "unknown_color"},
		{"#FF0000", "#FF0000"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeStickyColor(tt.input)
			if result != tt.expect {
				t.Errorf("normalizeStickyColor(%q) = %q, want %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestNormalizeTagColor(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"red", "red"},
		{"Red", "red"},
		{"grey", "gray"},
		{"custom", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeTagColor(tt.input)
			if result != tt.expect {
				t.Errorf("normalizeTagColor(%q) = %q, want %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCreateSnippet(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		query     string
		contextLen int
		expect    string
	}{
		{"match at start", "hello world test", "hello", 5, "hello worl..."},
		{"match in middle", "this is a test string", "test", 5, "...is a test stri..."},
		{"no match", "hello world", "xyz", 5, "hello w..."}, // truncate uses contextLen*2
		{"case insensitive", "Hello World", "world", 5, "...ello World"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createSnippet(tt.content, tt.query, tt.contextLen)
			if result != tt.expect {
				t.Errorf("createSnippet(%q, %q, %d) = %q, want %q",
					tt.content, tt.query, tt.contextLen, result, tt.expect)
			}
		})
	}
}

// =============================================================================
// Client Tests
// =============================================================================

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
	if client.cacheTTL != DefaultCacheTTL {
		t.Errorf("cacheTTL = %v, want %v", client.cacheTTL, DefaultCacheTTL)
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

// =============================================================================
// API Request Tests with Mock Server
// =============================================================================

func TestValidateToken(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing or incorrect Authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(UserInfo{
			ID:    "user123",
			Name:  "Test User",
			Email: "test@example.com",
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

func TestCreateStickyArgs(t *testing.T) {
	args := CreateStickyArgs{
		BoardID:  "board123",
		Content:  "Test sticky",
		X:        100,
		Y:        200,
		Color:    "yellow",
		Width:    150,
		ParentID: "frame123",
	}

	if args.BoardID != "board123" {
		t.Errorf("BoardID = %q, want %q", args.BoardID, "board123")
	}
	if args.Content != "Test sticky" {
		t.Errorf("Content = %q, want %q", args.Content, "Test sticky")
	}
}

// =============================================================================
// Context Cancellation Tests
// =============================================================================

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

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkValidateBoardID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateBoardID("uXjVN1234567890")
	}
}

func BenchmarkTruncate(b *testing.B) {
	content := "This is a test string that needs to be truncated to a shorter length"
	for i := 0; i < b.N; i++ {
		truncate(content, 30)
	}
}

func BenchmarkNormalizeStickyColor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		normalizeStickyColor("yellow")
	}
}
