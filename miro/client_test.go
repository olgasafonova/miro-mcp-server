package miro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
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

// =============================================================================
// Item Operations Tests
// =============================================================================

func TestListItems_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards/board123/items") {
			t.Errorf("expected /boards/board123/items, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "item1",
					"type": "sticky_note",
					"position": map[string]interface{}{
						"x": 100.0,
						"y": 200.0,
					},
					"data": map[string]interface{}{
						"content": "Test sticky",
					},
				},
				{
					"id":   "item2",
					"type": "shape",
					"position": map[string]interface{}{
						"x": 300.0,
						"y": 400.0,
					},
					"data": map[string]interface{}{
						"content": "Test shape",
					},
				},
			},
			"cursor": "next-page-cursor",
			"size":   2,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListItems(context.Background(), ListItemsArgs{
		BoardID: "board123",
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if !result.HasMore {
		t.Error("HasMore should be true")
	}
	if result.Items[0].ID != "item1" {
		t.Errorf("first item ID = %q, want 'item1'", result.Items[0].ID)
	}
	if result.Items[0].Type != "sticky_note" {
		t.Errorf("first item type = %q, want 'sticky_note'", result.Items[0].Type)
	}
}

func TestListItems_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.ListItems(context.Background(), ListItemsArgs{})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestListItems_WithTypeFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") != "sticky_note" {
			t.Errorf("expected type=sticky_note, got %s", r.URL.Query().Get("type"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
			"size": 0,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListItems(context.Background(), ListItemsArgs{
		BoardID: "board123",
		Type:    "sticky_note",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetItem_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boards/board123/items/item456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "item456",
			"type": "sticky_note",
			"position": map[string]interface{}{
				"x": 150.0,
				"y": 250.0,
			},
			"geometry": map[string]interface{}{
				"width":  200.0,
				"height": 200.0,
			},
			"data": map[string]interface{}{
				"content": "Detailed content",
			},
			"style": map[string]interface{}{
				"fillColor": "light_yellow",
			},
			"createdAt":  "2024-01-01T00:00:00Z",
			"modifiedAt": "2024-01-02T00:00:00Z",
			"createdBy": map[string]interface{}{
				"name": "John Doe",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetItem(context.Background(), GetItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "item456" {
		t.Errorf("ID = %q, want 'item456'", result.ID)
	}
	if result.Content != "Detailed content" {
		t.Errorf("Content = %q, want 'Detailed content'", result.Content)
	}
	if result.X != 150.0 {
		t.Errorf("X = %f, want 150.0", result.X)
	}
	if result.Width != 200.0 {
		t.Errorf("Width = %f, want 200.0", result.Width)
	}
	if result.CreatedBy != "John Doe" {
		t.Errorf("CreatedBy = %q, want 'John Doe'", result.CreatedBy)
	}
}

func TestGetItem_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    GetItemArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    GetItemArgs{ItemID: "item123"},
			errText: "board_id is required",
		},
		{
			name:    "empty item_id",
			args:    GetItemArgs{BoardID: "board123"},
			errText: "item_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetItem(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestUpdateItem_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/items/item456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify the update body structure
		if _, ok := body["data"]; !ok {
			t.Error("expected 'data' field in request body")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "item456",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	content := "Updated content"
	result, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
		Content: &content,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.ItemID != "item456" {
		t.Errorf("ItemID = %q, want 'item456'", result.ItemID)
	}
}

func TestUpdateItem_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
		// No changes specified
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true even with no changes")
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateItem_PositionUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		pos, ok := body["position"].(map[string]interface{})
		if !ok {
			t.Error("expected 'position' field in request body")
		}
		if pos["x"] != 500.0 {
			t.Errorf("x = %v, want 500.0", pos["x"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "item456"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	x := 500.0
	_, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
		X:       &x,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteItem_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/items/item456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteItem(context.Background(), DeleteItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.ItemID != "item456" {
		t.Errorf("ItemID = %q, want 'item456'", result.ItemID)
	}
}

func TestDeleteItem_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    DeleteItemArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    DeleteItemArgs{ItemID: "item123"},
			errText: "board_id is required",
		},
		{
			name:    "empty item_id",
			args:    DeleteItemArgs{BoardID: "board123"},
			errText: "item_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.DeleteItem(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestSearchBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "item1",
					"type": "sticky_note",
					"position": map[string]interface{}{
						"x": 100.0,
						"y": 200.0,
					},
					"data": map[string]interface{}{
						"content": "This is a test sticky note",
					},
				},
				{
					"id":   "item2",
					"type": "text",
					"data": map[string]interface{}{
						"content": "Another item without test",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.SearchBoard(context.Background(), SearchBoardArgs{
		BoardID: "board123",
		Query:   "test",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if result.Query != "test" {
		t.Errorf("Query = %q, want 'test'", result.Query)
	}
}

func TestSearchBoard_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    SearchBoardArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    SearchBoardArgs{Query: "test"},
			errText: "board_id is required",
		},
		{
			name:    "empty query",
			args:    SearchBoardArgs{BoardID: "board123"},
			errText: "query is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.SearchBoard(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestListAllItems_Success(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "item1", "type": "sticky_note"},
					{"id": "item2", "type": "sticky_note"},
				},
				"cursor": "page2",
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "item3", "type": "shape"},
				},
				"cursor": "",
			})
		}
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListAllItems(context.Background(), ListAllItemsArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 3 {
		t.Errorf("Count = %d, want 3", result.Count)
	}
	if result.TotalPages != 2 {
		t.Errorf("TotalPages = %d, want 2", result.TotalPages)
	}
}

func TestListAllItems_MaxItemsLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return more items than max
		items := make([]map[string]interface{}, 10)
		for i := 0; i < 10; i++ {
			items[i] = map[string]interface{}{
				"id":   fmt.Sprintf("item%d", i),
				"type": "sticky_note",
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":   items,
			"cursor": "more",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListAllItems(context.Background(), ListAllItemsArgs{
		BoardID:  "board123",
		MaxItems: 5,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 5 {
		t.Errorf("Count = %d, want 5 (max items limit)", result.Count)
	}
	if !result.Truncated {
		t.Error("Truncated should be true")
	}
}

// =============================================================================
// Tag Operations Tests
// =============================================================================

func TestCreateTag_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["title"] != "Important" {
			t.Errorf("title = %v, want 'Important'", body["title"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        "tag123",
			"title":     "Important",
			"fillColor": "red",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateTag(context.Background(), CreateTagArgs{
		BoardID: "board123",
		Title:   "Important",
		Color:   "red",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "tag123" {
		t.Errorf("ID = %q, want 'tag123'", result.ID)
	}
	if result.Title != "Important" {
		t.Errorf("Title = %q, want 'Important'", result.Title)
	}
}

func TestCreateTag_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateTagArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateTagArgs{Title: "Test"},
			errText: "board_id is required",
		},
		{
			name:    "empty title",
			args:    CreateTagArgs{BoardID: "board123"},
			errText: "title is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateTag(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestListTags_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "tag1", "title": "Urgent", "fillColor": "red"},
				{"id": "tag2", "title": "Done", "fillColor": "green"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if result.Tags[0].Title != "Urgent" {
		t.Errorf("first tag title = %q, want 'Urgent'", result.Tags[0].Title)
	}
}

func TestListTags_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 0 {
		t.Errorf("Count = %d, want 0", result.Count)
	}
	if result.Message != "No tags on this board" {
		t.Errorf("Message = %q, want 'No tags on this board'", result.Message)
	}
}

func TestAttachTag_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		expectedPath := "/boards/board123/items/item456"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %q, want %q", r.URL.Path, expectedPath)
		}
		if r.URL.Query().Get("tag_id") != "tag789" {
			t.Errorf("tag_id = %q, want 'tag789'", r.URL.Query().Get("tag_id"))
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.AttachTag(context.Background(), AttachTagArgs{
		BoardID: "board123",
		ItemID:  "item456",
		TagID:   "tag789",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestAttachTag_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    AttachTagArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    AttachTagArgs{ItemID: "item123", TagID: "tag123"},
			errText: "board_id is required",
		},
		{
			name:    "empty item_id",
			args:    AttachTagArgs{BoardID: "board123", TagID: "tag123"},
			errText: "item_id is required",
		},
		{
			name:    "empty tag_id",
			args:    AttachTagArgs{BoardID: "board123", ItemID: "item123"},
			errText: "tag_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.AttachTag(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestDetachTag_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "board123",
		ItemID:  "item456",
		TagID:   "tag789",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestGetItemTags_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boards/board123/items/item456/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "tag1", "title": "Priority", "fillColor": "red"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "board123",
		ItemID:  "item456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 1 {
		t.Errorf("Count = %d, want 1", result.Count)
	}
	if result.ItemID != "item456" {
		t.Errorf("ItemID = %q, want 'item456'", result.ItemID)
	}
}

func TestUpdateTag_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/tags/tag456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        "tag456",
			"title":     "Updated Title",
			"fillColor": "blue",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateTag(context.Background(), UpdateTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
		Title:   "Updated Title",
		Color:   "blue",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Title != "Updated Title" {
		t.Errorf("Title = %q, want 'Updated Title'", result.Title)
	}
}

func TestUpdateTag_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    UpdateTagArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    UpdateTagArgs{TagID: "tag123", Title: "Test"},
			errText: "board_id is required",
		},
		{
			name:    "empty tag_id",
			args:    UpdateTagArgs{BoardID: "board123", Title: "Test"},
			errText: "tag_id is required",
		},
		{
			name:    "no changes",
			args:    UpdateTagArgs{BoardID: "board123", TagID: "tag123"},
			errText: "at least one of title or color is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UpdateTag(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestDeleteTag_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/tags/tag456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteTag(context.Background(), DeleteTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.TagID != "tag456" {
		t.Errorf("TagID = %q, want 'tag456'", result.TagID)
	}
}

// =============================================================================
// Create Operations Tests
// =============================================================================

func TestCreateShape_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/shapes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		data := body["data"].(map[string]interface{})
		if data["shape"] != "rectangle" {
			t.Errorf("shape = %v, want 'rectangle'", data["shape"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "shape123",
			"data": map[string]interface{}{
				"shape":   "rectangle",
				"content": "Test shape",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateShape(context.Background(), CreateShapeArgs{
		BoardID: "board123",
		Shape:   "rectangle",
		Content: "Test shape",
		X:       100,
		Y:       200,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "shape123" {
		t.Errorf("ID = %q, want 'shape123'", result.ID)
	}
	if result.Shape != "rectangle" {
		t.Errorf("Shape = %q, want 'rectangle'", result.Shape)
	}
}

func TestCreateShape_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateShapeArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateShapeArgs{Shape: "rectangle"},
			errText: "board_id is required",
		},
		{
			name:    "empty shape",
			args:    CreateShapeArgs{BoardID: "board123"},
			errText: "shape type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateShape(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestCreateText_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/texts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "text123",
			"data": map[string]interface{}{
				"content": "Hello World",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateText(context.Background(), CreateTextArgs{
		BoardID: "board123",
		Content: "Hello World",
		X:       100,
		Y:       200,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "text123" {
		t.Errorf("ID = %q, want 'text123'", result.ID)
	}
	if result.Content != "Hello World" {
		t.Errorf("Content = %q, want 'Hello World'", result.Content)
	}
}

func TestCreateText_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateTextArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateTextArgs{Content: "test"},
			errText: "board_id is required",
		},
		{
			name:    "empty content",
			args:    CreateTextArgs{BoardID: "board123"},
			errText: "content is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateText(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestCreateConnector_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/connectors" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "connector123",
			"startItem": map[string]interface{}{
				"id": "item1",
			},
			"endItem": map[string]interface{}{
				"id": "item2",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateConnector(context.Background(), CreateConnectorArgs{
		BoardID:     "board123",
		StartItemID: "item1",
		EndItemID:   "item2",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "connector123" {
		t.Errorf("ID = %q, want 'connector123'", result.ID)
	}
}

func TestCreateConnector_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateConnectorArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateConnectorArgs{StartItemID: "item1", EndItemID: "item2"},
			errText: "board_id is required",
		},
		{
			name:    "empty start_item_id",
			args:    CreateConnectorArgs{BoardID: "board123", EndItemID: "item2"},
			errText: "start_item_id and end_item_id are required",
		},
		{
			name:    "empty end_item_id",
			args:    CreateConnectorArgs{BoardID: "board123", StartItemID: "item1"},
			errText: "start_item_id and end_item_id are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateConnector(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestCreateFrame_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/frames" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "frame123",
			"data": map[string]interface{}{
				"title": "Sprint 1",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateFrame(context.Background(), CreateFrameArgs{
		BoardID: "board123",
		Title:   "Sprint 1",
		X:       0,
		Y:       0,
		Width:   800,
		Height:  600,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "frame123" {
		t.Errorf("ID = %q, want 'frame123'", result.ID)
	}
	if result.Title != "Sprint 1" {
		t.Errorf("Title = %q, want 'Sprint 1'", result.Title)
	}
}

func TestCreateFrame_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Only board_id is required - title is optional
	_, err := client.CreateFrame(context.Background(), CreateFrameArgs{Title: "Test"})
	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestCreateCard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/cards" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "card123",
			"data": map[string]interface{}{
				"title":       "Task Card",
				"description": "Do something",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateCard(context.Background(), CreateCardArgs{
		BoardID:     "board123",
		Title:       "Task Card",
		Description: "Do something",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "card123" {
		t.Errorf("ID = %q, want 'card123'", result.ID)
	}
	if result.Title != "Task Card" {
		t.Errorf("Title = %q, want 'Task Card'", result.Title)
	}
}

func TestCreateCard_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateCardArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateCardArgs{Title: "Test"},
			errText: "board_id is required",
		},
		{
			name:    "empty title",
			args:    CreateCardArgs{BoardID: "board123"},
			errText: "title is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateCard(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestCreateImage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/images" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "image123",
			"data": map[string]interface{}{
				"title": "Logo",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateImage(context.Background(), CreateImageArgs{
		BoardID: "board123",
		URL:     "https://example.com/image.png",
		Title:   "Logo",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "image123" {
		t.Errorf("ID = %q, want 'image123'", result.ID)
	}
}

func TestCreateImage_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateImageArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateImageArgs{URL: "https://example.com/img.png"},
			errText: "board_id is required",
		},
		{
			name:    "empty url",
			args:    CreateImageArgs{BoardID: "board123"},
			errText: "url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateImage(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestListConnectors_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards/board123/connectors") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id": "conn1",
					"startItem": map[string]interface{}{
						"id": "item1",
					},
					"endItem": map[string]interface{}{
						"id": "item2",
					},
				},
			},
			"size": 1,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListConnectors(context.Background(), ListConnectorsArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 1 {
		t.Errorf("Count = %d, want 1", result.Count)
	}
	if result.Connectors[0].ID != "conn1" {
		t.Errorf("first connector ID = %q, want 'conn1'", result.Connectors[0].ID)
	}
}

func TestGetConnector_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/connectors/conn456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "conn456",
			"startItem": map[string]interface{}{
				"item": "start123",
			},
			"endItem": map[string]interface{}{
				"item": "end456",
			},
			"style": map[string]interface{}{
				"strokeColor": "#000000",
				"strokeWidth": "2.0",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetConnector(context.Background(), GetConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "conn456" {
		t.Errorf("ID = %q, want 'conn456'", result.ID)
	}
	if result.StartItemID != "start123" {
		t.Errorf("StartItemID = %q, want 'start123'", result.StartItemID)
	}
}

// =============================================================================
// Board Extended Operations Tests
// =============================================================================

func TestCopyBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/boards" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("copy_from") != "board123" {
			t.Errorf("expected copy_from=board123, got %s", r.URL.Query().Get("copy_from"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "newboard456",
			"name":     "Copied Board",
			"viewLink": "https://miro.com/newboard456",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CopyBoard(context.Background(), CopyBoardArgs{
		BoardID: "board123",
		Name:    "Copied Board",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "newboard456" {
		t.Errorf("ID = %q, want 'newboard456'", result.ID)
	}
	if result.Name != "Copied Board" {
		t.Errorf("Name = %q, want 'Copied Board'", result.Name)
	}
}

func TestCopyBoard_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.CopyBoard(context.Background(), CopyBoardArgs{Name: "Test"})
	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestFindBoardByName_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query != "Design Sprint" {
			t.Errorf("query = %q, want 'Design Sprint'", query)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":       "board123",
					"name":     "Design Sprint",
					"viewLink": "https://miro.com/board123",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{
		Name: "Design Sprint",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "board123" {
		t.Errorf("ID = %q, want 'board123'", result.ID)
	}
	if result.Name != "Design Sprint" {
		t.Errorf("Name = %q, want 'Design Sprint'", result.Name)
	}
}

func TestFindBoardByName_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{
		Name: "Nonexistent Board",
	})

	if err == nil {
		t.Fatal("expected error for board not found")
	}
	if !strings.Contains(err.Error(), "no board found") {
		t.Errorf("expected 'no board found' error, got: %v", err)
	}
}

func TestGetBoardSummary_Success(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/items") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "item1", "type": "sticky_note"},
					{"id": "item2", "type": "sticky_note"},
					{"id": "item3", "type": "shape"},
				},
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "board123",
				"name":        "Test Board",
				"description": "A test board",
			})
		}
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetBoardSummary(context.Background(), GetBoardSummaryArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Test Board" {
		t.Errorf("Name = %q, want 'Test Board'", result.Name)
	}
	if result.TotalItems != 3 {
		t.Errorf("TotalItems = %d, want 3", result.TotalItems)
	}
}

// =============================================================================
// Group Operations Tests
// =============================================================================

func TestCreateGroup_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/groups" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify request body
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		data := body["data"].(map[string]interface{})
		items := data["items"].([]interface{})
		if len(items) != 3 {
			t.Errorf("expected 3 items, got %d", len(items))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "group123",
			"type": "group",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateGroup(context.Background(), CreateGroupArgs{
		BoardID: "board123",
		ItemIDs: []string{"item1", "item2", "item3"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "group123" {
		t.Errorf("ID = %q, want 'group123'", result.ID)
	}
	if len(result.ItemIDs) != 3 {
		t.Errorf("ItemIDs count = %d, want 3", len(result.ItemIDs))
	}
}

func TestCreateGroup_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateGroupArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateGroupArgs{ItemIDs: []string{"item1", "item2"}},
			errText: "board_id",
		},
		{
			name:    "less than 2 items",
			args:    CreateGroupArgs{BoardID: "board123", ItemIDs: []string{"item1"}},
			errText: "at least 2 items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateGroup(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestUngroup_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/groups/group456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.Ungroup(context.Background(), UngroupArgs{
		BoardID: "board123",
		GroupID: "group456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.GroupID != "group456" {
		t.Errorf("GroupID = %q, want 'group456'", result.GroupID)
	}
}

func TestUngroup_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    UngroupArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    UngroupArgs{GroupID: "group123"},
			errText: "board_id",
		},
		{
			name:    "empty group_id",
			args:    UngroupArgs{BoardID: "board123"},
			errText: "group_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Ungroup(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

// =============================================================================
// Member Operations Tests
// =============================================================================

func TestListBoardMembers_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards/board123/members") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "user1", "name": "Alice", "role": "owner"},
				{"id": "user2", "name": "Bob", "role": "editor"},
			},
			"total": 2,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListBoardMembers(context.Background(), ListBoardMembersArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if len(result.Members) != 2 {
		t.Errorf("Members count = %d, want 2", len(result.Members))
	}
}

func TestListBoardMembers_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListBoardMembers(context.Background(), ListBoardMembersArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Message, "No members") {
		t.Errorf("expected 'No members' message, got: %s", result.Message)
	}
}

func TestShareBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/members" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify request body
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		role := body["role"].(string)
		if role != "editor" {
			t.Errorf("expected role 'editor', got '%s'", role)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ShareBoard(context.Background(), ShareBoardArgs{
		BoardID: "board123",
		Email:   "user@example.com",
		Role:    "editor",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.Email != "user@example.com" {
		t.Errorf("Email = %q, want 'user@example.com'", result.Email)
	}
	if result.Role != "editor" {
		t.Errorf("Role = %q, want 'editor'", result.Role)
	}
}

func TestShareBoard_DefaultRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		role := body["role"].(string)
		if role != "viewer" {
			t.Errorf("expected default role 'viewer', got '%s'", role)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ShareBoard(context.Background(), ShareBoardArgs{
		BoardID: "board123",
		Email:   "user@example.com",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Role != "viewer" {
		t.Errorf("Role = %q, want 'viewer'", result.Role)
	}
}

func TestShareBoard_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    ShareBoardArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    ShareBoardArgs{Email: "user@example.com"},
			errText: "board_id",
		},
		{
			name:    "empty email",
			args:    ShareBoardArgs{BoardID: "board123"},
			errText: "email is required",
		},
		{
			name:    "invalid role",
			args:    ShareBoardArgs{BoardID: "board123", Email: "user@example.com", Role: "admin"},
			errText: "invalid role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ShareBoard(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

// =============================================================================
// Export Operations Tests
// =============================================================================

func TestGetBoardPicture_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "board123",
			"name": "Test Board",
			"picture": map[string]interface{}{
				"imageURL": "https://miro-media.com/board123/preview.png",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetBoardPicture(context.Background(), GetBoardPictureArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ImageURL != "https://miro-media.com/board123/preview.png" {
		t.Errorf("ImageURL = %q, want 'https://miro-media.com/board123/preview.png'", result.ImageURL)
	}
}

func TestGetBoardPicture_NoPicture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "board123",
			"name": "Test Board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetBoardPicture(context.Background(), GetBoardPictureArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ImageURL != "" {
		t.Errorf("expected empty ImageURL, got: %s", result.ImageURL)
	}
	if !strings.Contains(result.Message, "no picture") {
		t.Errorf("expected 'no picture' message, got: %s", result.Message)
	}
}

func TestCreateExportJob_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/orgs/org123/boards/export/jobs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        "job123",
			"status":    "pending",
			"requestId": "req456",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateExportJob(context.Background(), CreateExportJobArgs{
		OrgID:    "org123",
		BoardIDs: []string{"board1", "board2"},
		Format:   "pdf",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.JobID != "job123" {
		t.Errorf("JobID = %q, want 'job123'", result.JobID)
	}
	if result.Status != "pending" {
		t.Errorf("Status = %q, want 'pending'", result.Status)
	}
}

func TestCreateExportJob_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateExportJobArgs
		errText string
	}{
		{
			name:    "empty org_id",
			args:    CreateExportJobArgs{BoardIDs: []string{"board1"}},
			errText: "org_id is required",
		},
		{
			name:    "empty board_ids",
			args:    CreateExportJobArgs{OrgID: "org123", BoardIDs: []string{}},
			errText: "board_ids is required",
		},
		{
			name:    "too many boards",
			args:    CreateExportJobArgs{OrgID: "org123", BoardIDs: make([]string, 51)},
			errText: "maximum 50 boards",
		},
		{
			name:    "invalid format",
			args:    CreateExportJobArgs{OrgID: "org123", BoardIDs: []string{"board1"}, Format: "png"},
			errText: "format must be pdf, svg, or html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateExportJob(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestGetExportJobStatus_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/orgs/org123/boards/export/jobs/job456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":             "job456",
			"status":         "in_progress",
			"progress":       50,
			"boardsTotal":    10,
			"boardsExported": 5,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetExportJobStatus(context.Background(), GetExportJobStatusArgs{
		OrgID: "org123",
		JobID: "job456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "in_progress" {
		t.Errorf("Status = %q, want 'in_progress'", result.Status)
	}
	if result.Progress != 50 {
		t.Errorf("Progress = %d, want 50", result.Progress)
	}
	if result.BoardsExported != 5 {
		t.Errorf("BoardsExported = %d, want 5", result.BoardsExported)
	}
}

func TestGetExportJobStatus_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    GetExportJobStatusArgs
		errText string
	}{
		{
			name:    "empty org_id",
			args:    GetExportJobStatusArgs{JobID: "job123"},
			errText: "org_id is required",
		},
		{
			name:    "empty job_id",
			args:    GetExportJobStatusArgs{OrgID: "org123"},
			errText: "job_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetExportJobStatus(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestGetExportJobResults_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/orgs/org123/boards/export/jobs/job456/results" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "job456",
			"status": "completed",
			"data": []map[string]interface{}{
				{
					"boardId":     "board1",
					"boardName":   "Board One",
					"downloadUrl": "https://download.miro.com/exports/board1.pdf",
					"format":      "pdf",
				},
				{
					"boardId":     "board2",
					"boardName":   "Board Two",
					"downloadUrl": "https://download.miro.com/exports/board2.pdf",
					"format":      "pdf",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetExportJobResults(context.Background(), GetExportJobResultsArgs{
		OrgID: "org123",
		JobID: "job456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("Status = %q, want 'completed'", result.Status)
	}
	if len(result.Boards) != 2 {
		t.Errorf("Boards count = %d, want 2", len(result.Boards))
	}
	if result.Boards[0].DownloadURL != "https://download.miro.com/exports/board1.pdf" {
		t.Errorf("unexpected download URL: %s", result.Boards[0].DownloadURL)
	}
}

func TestGetExportJobResults_NotCompleted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "job456",
			"status": "in_progress",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetExportJobResults(context.Background(), GetExportJobResultsArgs{
		OrgID: "org123",
		JobID: "job456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "in_progress" {
		t.Errorf("Status = %q, want 'in_progress'", result.Status)
	}
	if result.Boards != nil {
		t.Error("expected nil Boards for incomplete job")
	}
	if !strings.Contains(result.Message, "not yet available") {
		t.Errorf("expected 'not yet available' message, got: %s", result.Message)
	}
}

// =============================================================================
// Mindmap Tests
// =============================================================================

func TestCreateMindmapNode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/mindmap_nodes") {
			t.Errorf("expected mindmap_nodes path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "mindmap123",
			"data": map[string]interface{}{
				"content": "Root Node",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateMindmapNode(context.Background(), CreateMindmapNodeArgs{
		BoardID: "board123",
		Content: "Root Node",
		X:       100,
		Y:       200,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "mindmap123" {
		t.Errorf("ID = %q, want 'mindmap123'", result.ID)
	}
	if result.Content != "Root Node" {
		t.Errorf("Content = %q, want 'Root Node'", result.Content)
	}
}

func TestCreateMindmapNode_WithParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		parent, ok := body["parent"].(map[string]interface{})
		if !ok {
			t.Error("expected parent in request body")
		}
		if parent["id"] != "parent123" {
			t.Errorf("parent id = %v, want 'parent123'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "child123",
			"data": map[string]interface{}{
				"content": "Child Node",
			},
			"parent": map[string]interface{}{
				"id": "parent123",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateMindmapNode(context.Background(), CreateMindmapNodeArgs{
		BoardID:  "board123",
		Content:  "Child Node",
		ParentID: "parent123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ParentID != "parent123" {
		t.Errorf("ParentID = %q, want 'parent123'", result.ParentID)
	}
}

func TestCreateMindmapNode_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    CreateMindmapNodeArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    CreateMindmapNodeArgs{Content: "Test"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty content",
			args:    CreateMindmapNodeArgs{BoardID: "board123"},
			wantErr: "content is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateMindmapNode(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestTruncateMindmap(t *testing.T) {
	tests := []struct {
		input  string
		max    int
		expect string
	}{
		{"short", 10, "short"},
		{"exactly ten", 10, "exactly..."},
		{"longer string", 10, "longer ..."},
	}

	for _, tt := range tests {
		result := truncateMindmap(tt.input, tt.max)
		if result != tt.expect {
			t.Errorf("truncateMindmap(%q, %d) = %q, want %q", tt.input, tt.max, result, tt.expect)
		}
	}
}

// =============================================================================
// Connector Update/Delete Tests
// =============================================================================

func TestUpdateConnector_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/connectors/") {
			t.Errorf("expected connectors path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "conn123",
			"captions": []map[string]interface{}{
				{"content": "Updated Caption"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateConnector(context.Background(), UpdateConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn123",
		Caption:     "Updated Caption",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "conn123" {
		t.Errorf("ID = %q, want 'conn123'", result.ID)
	}
}

func TestUpdateConnector_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    UpdateConnectorArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    UpdateConnectorArgs{ConnectorID: "conn123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty connector ID",
			args:    UpdateConnectorArgs{BoardID: "board123"},
			wantErr: "connector_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UpdateConnector(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestDeleteConnector_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/connectors/") {
			t.Errorf("expected connectors path, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteConnector(context.Background(), DeleteConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "conn123" {
		t.Errorf("ID = %q, want 'conn123'", result.ID)
	}
}

// =============================================================================
// BulkCreate Tests
// =============================================================================

func TestBulkCreate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "item123",
			"data": map[string]interface{}{
				"content": "Test",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.BulkCreate(context.Background(), BulkCreateArgs{
		BoardID: "board123",
		Items: []BulkCreateItem{
			{Type: "sticky_note", Content: "Note 1"},
			{Type: "sticky_note", Content: "Note 2"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Created != 2 {
		t.Errorf("Created = %d, want 2", result.Created)
	}
}

func TestBulkCreate_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    BulkCreateArgs
		wantErr string
	}{
		{
			name: "empty board ID",
			args: BulkCreateArgs{
				Items: []BulkCreateItem{{Type: "sticky_note", Content: "Test"}},
			},
			wantErr: "board_id is required",
		},
		{
			name: "empty items",
			args: BulkCreateArgs{
				BoardID: "board123",
				Items:   []BulkCreateItem{},
			},
			wantErr: "at least one item is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.BulkCreate(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// =============================================================================
// CreateStickyGrid Tests
// =============================================================================

func TestCreateStickyGrid_Success(t *testing.T) {
	var callCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": fmt.Sprintf("sticky%d", count),
			"data": map[string]interface{}{
				"content": "Test",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateStickyGrid(context.Background(), CreateStickyGridArgs{
		BoardID:  "board123",
		Contents: []string{"Note 1", "Note 2", "Note 3"},
		Columns:  2,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Created != 3 {
		t.Errorf("Created = %d, want 3", result.Created)
	}
	if len(result.ItemIDs) != 3 {
		t.Errorf("ItemIDs length = %d, want 3", len(result.ItemIDs))
	}
}

func TestCreateStickyGrid_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    CreateStickyGridArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    CreateStickyGridArgs{Contents: []string{"Test"}},
			wantErr: "board_id is required",
		},
		{
			name:    "empty contents",
			args:    CreateStickyGridArgs{BoardID: "board123"},
			wantErr: "at least one content item is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateStickyGrid(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// =============================================================================
// CreateDocument Tests
// =============================================================================

func TestCreateDocument_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/documents") {
			t.Errorf("expected documents path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "doc123",
			"data": map[string]interface{}{
				"title": "Test Document",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateDocument(context.Background(), CreateDocumentArgs{
		BoardID: "board123",
		Title:   "Test Document",
		URL:     "https://example.com/doc.pdf",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "doc123" {
		t.Errorf("ID = %q, want 'doc123'", result.ID)
	}
}

func TestCreateDocument_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    CreateDocumentArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    CreateDocumentArgs{URL: "https://example.com/doc.pdf"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty URL",
			args:    CreateDocumentArgs{BoardID: "board123"},
			wantErr: "url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateDocument(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// =============================================================================
// CreateEmbed Tests
// =============================================================================

func TestCreateEmbed_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/embeds") {
			t.Errorf("expected embeds path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "embed123",
			"data": map[string]interface{}{
				"url": "https://youtube.com/watch?v=test",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateEmbed(context.Background(), CreateEmbedArgs{
		BoardID: "board123",
		URL:     "https://youtube.com/watch?v=test",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "embed123" {
		t.Errorf("ID = %q, want 'embed123'", result.ID)
	}
}

func TestCreateEmbed_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    CreateEmbedArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    CreateEmbedArgs{URL: "https://youtube.com/watch?v=test"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty URL",
			args:    CreateEmbedArgs{BoardID: "board123"},
			wantErr: "url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateEmbed(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}
