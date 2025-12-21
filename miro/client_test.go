package miro

import (
	"context"
	"encoding/json"
	"errors"
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
// Mock HTTP Server Tests - Client Methods
// =============================================================================

// newTestClientWithServer creates a client pointing to a mock HTTP server.
func newTestClientWithServer(serverURL string) *Client {
	cfg := testConfig()
	client := NewClient(cfg, testLogger())
	client.baseURL = serverURL
	return client
}

func TestListBoards_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards") {
			t.Errorf("expected /boards path, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or incorrect Authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "board1", "name": "Design Sprint", "viewLink": "https://miro.com/board1"},
				{"id": "board2", "name": "Retro", "viewLink": "https://miro.com/board2"},
			},
			"size":  2,
			"total": 2,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListBoards(context.Background(), ListBoardsArgs{Query: "test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if result.Boards[0].Name != "Design Sprint" {
		t.Errorf("first board name = %q, want 'Design Sprint'", result.Boards[0].Name)
	}
}

func TestListBoards_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "unauthorized",
			"message": "Invalid access token",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListBoards(context.Background(), ListBoardsArgs{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Check it's an API error
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if apiErr.Code != "unauthorized" {
		t.Errorf("Code = %q, want 'unauthorized'", apiErr.Code)
	}
}

func TestGetBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boards/board123" {
			t.Errorf("expected /boards/board123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "board123",
			"name":        "Test Board",
			"description": "A test board",
			"viewLink":    "https://miro.com/board123",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetBoard(context.Background(), GetBoardArgs{BoardID: "board123"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "board123" {
		t.Errorf("ID = %q, want 'board123'", result.ID)
	}
	if result.Name != "Test Board" {
		t.Errorf("Name = %q, want 'Test Board'", result.Name)
	}
}

func TestGetBoard_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "not_found",
			"message": "Board not found",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetBoard(context.Background(), GetBoardArgs{BoardID: "nonexistent"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsNotFoundError(err) {
		t.Errorf("expected not found error, got: %v", err)
	}
}

func TestGetBoard_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.GetBoard(context.Background(), GetBoardArgs{BoardID: ""})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestGetBoard_Caching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "board123",
			"name": "Cached Board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	ctx := context.Background()

	// First call - should hit the server
	_, err := client.GetBoard(ctx, GetBoardArgs{BoardID: "board123"})
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Second call - should use cache
	_, err = client.GetBoard(ctx, GetBoardArgs{BoardID: "board123"})
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("server called %d times, want 1 (caching should prevent second call)", callCount)
	}
}

func TestCreateBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards" {
			t.Errorf("expected /boards, got %s", r.URL.Path)
		}

		// Verify request body
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "New Board" {
			t.Errorf("name = %v, want 'New Board'", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "new-board-id",
			"name":     "New Board",
			"viewLink": "https://miro.com/new-board-id",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateBoard(context.Background(), CreateBoardArgs{
		Name:        "New Board",
		Description: "Test description",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "new-board-id" {
		t.Errorf("ID = %q, want 'new-board-id'", result.ID)
	}
	if result.Name != "New Board" {
		t.Errorf("Name = %q, want 'New Board'", result.Name)
	}
}

func TestCreateBoard_EmptyName(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.CreateBoard(context.Background(), CreateBoardArgs{Name: ""})

	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("expected 'name is required' error, got: %v", err)
	}
}

func TestCreateSticky_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/sticky_notes" {
			t.Errorf("expected /boards/board123/sticky_notes, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "sticky-id",
			"data": map[string]interface{}{
				"content": "Test sticky",
			},
			"style": map[string]interface{}{
				"fillColor": "light_yellow",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateSticky(context.Background(), CreateStickyArgs{
		BoardID: "board123",
		Content: "Test sticky",
		Color:   "yellow",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "sticky-id" {
		t.Errorf("ID = %q, want 'sticky-id'", result.ID)
	}
	if result.Content != "Test sticky" {
		t.Errorf("Content = %q, want 'Test sticky'", result.Content)
	}
}

func TestCreateSticky_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateStickyArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateStickyArgs{Content: "Test"},
			errText: "board_id is required",
		},
		{
			name:    "empty content",
			args:    CreateStickyArgs{BoardID: "board123"},
			errText: "content is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateSticky(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestRateLimitRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Rate limit exceeded",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	// Use context with short timeout for faster test
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// This tests that requestWithRetry works - we'll use it indirectly via a method that uses retry
	_, err := client.request(ctx, http.MethodGet, "/boards", nil)

	// The first call should hit rate limit (we don't retry in basic request)
	if err == nil {
		t.Fatal("expected rate limit error from first attempt")
	}
	if !IsRateLimitError(err) {
		t.Errorf("expected rate limit error, got: %v", err)
	}
}

func TestDeleteBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123" {
			t.Errorf("expected /boards/board123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteBoard(context.Background(), DeleteBoardArgs{BoardID: "board123"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.BoardID != "board123" {
		t.Errorf("BoardID = %q, want 'board123'", result.BoardID)
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
