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

// ptrString returns a pointer to the given string.
func ptrString(s string) *string { return &s }

// ptrFloat64 returns a pointer to the given float64.
func ptrFloat64(f float64) *float64 { return &f }

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

func TestCreateBoard_WithAllOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify all optional fields
		if body["description"] != "Full description" {
			t.Errorf("description = %v, want 'Full description'", body["description"])
		}
		if body["teamId"] != "team123" {
			t.Errorf("teamId = %v, want 'team123'", body["teamId"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "full-board",
			"name":     "Full Board",
			"viewLink": "https://miro.com/full-board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateBoard(context.Background(), CreateBoardArgs{
		Name:        "Full Board",
		Description: "Full description",
		TeamID:      "team123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "full-board" {
		t.Errorf("ID = %q, want 'full-board'", result.ID)
	}
}

func TestCopyBoard_WithAllOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "copy_from=source-board") {
			t.Errorf("expected copy_from query param, got %s", r.URL.RawQuery)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify optional fields
		if body["name"] != "Copy Name" {
			t.Errorf("name = %v, want 'Copy Name'", body["name"])
		}
		if body["description"] != "Copy Desc" {
			t.Errorf("description = %v, want 'Copy Desc'", body["description"])
		}
		if body["teamId"] != "team456" {
			t.Errorf("teamId = %v, want 'team456'", body["teamId"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "copied-board",
			"name":     "Copy Name",
			"viewLink": "https://miro.com/copied-board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CopyBoard(context.Background(), CopyBoardArgs{
		BoardID:     "source-board",
		Name:        "Copy Name",
		Description: "Copy Desc",
		TeamID:      "team456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "copied-board" {
		t.Errorf("ID = %q, want 'copied-board'", result.ID)
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

func TestCreateSticky_WithWidthAndParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify geometry
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else if geom["width"] != float64(250) {
			t.Errorf("width = %v, want 250", geom["width"])
		}
		// Verify parent
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected parent in request body")
		} else if parent["id"] != "frame-abc" {
			t.Errorf("parent.id = %v, want 'frame-abc'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "sticky-geo",
			"data": map[string]interface{}{
				"content": "Wide sticky",
			},
			"style": map[string]interface{}{
				"fillColor": "light_yellow",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateSticky(context.Background(), CreateStickyArgs{
		BoardID:  "board123",
		Content:  "Wide sticky",
		Width:    250,
		ParentID: "frame-abc",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "sticky-geo" {
		t.Errorf("ID = %q, want 'sticky-geo'", result.ID)
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

func TestDeleteBoard_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.DeleteBoard(context.Background(), DeleteBoardArgs{BoardID: ""})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestDeleteBoard_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  403,
			"message": "Access denied",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteBoard(context.Background(), DeleteBoardArgs{BoardID: "board123"})

	if err == nil {
		t.Fatal("expected error for API failure")
	}
	if result.Success {
		t.Error("Success should be false for API error")
	}
}

func TestGetBoardSummary_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.GetBoardSummary(context.Background(), GetBoardSummaryArgs{BoardID: ""})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestGetBoardSummary_GetBoardError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  404,
			"message": "Board not found",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetBoardSummary(context.Background(), GetBoardSummaryArgs{BoardID: "board123"})

	if err == nil {
		t.Fatal("expected error when board not found")
	}
	if !strings.Contains(err.Error(), "failed to get board") {
		t.Errorf("expected 'failed to get board' error, got: %v", err)
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

func TestUpdateItem_WithYPosition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		pos, ok := body["position"].(map[string]interface{})
		if !ok {
			t.Error("expected 'position' field in request body")
		}
		if pos["y"] != float64(300) {
			t.Errorf("y = %v, want 300", pos["y"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "item456"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	y := float64(300)
	_, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
		Y:       &y,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateItem_WithGeometry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		geom, ok := body["geometry"].(map[string]interface{})
		if !ok {
			t.Error("expected 'geometry' field in request body")
		}
		if geom["width"] != float64(400) {
			t.Errorf("width = %v, want 400", geom["width"])
		}
		if geom["height"] != float64(250) {
			t.Errorf("height = %v, want 250", geom["height"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "item456"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	width := float64(400)
	height := float64(250)
	_, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
		Width:   &width,
		Height:  &height,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateItem_WithColorAndParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify style
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected 'style' field in request body")
		} else if style["fillColor"] != "green" {
			t.Errorf("fillColor = %v, want 'green'", style["fillColor"])
		}

		// Verify parent
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected 'parent' field in request body")
		} else if parent["id"] != "frame-123" {
			t.Errorf("parent.id = %v, want 'frame-123'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "item456"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	color := "green"
	parentID := "frame-123"
	_, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID:  "board123",
		ItemID:   "item456",
		Color:    &color,
		ParentID: &parentID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateItem_RemoveFromFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify parent is null
		if parent, exists := body["parent"]; !exists {
			t.Error("expected 'parent' field in request body")
		} else if parent != nil {
			t.Errorf("parent = %v, want nil", parent)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "item456"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	emptyParent := ""
	_, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID:  "board123",
		ItemID:   "item456",
		ParentID: &emptyParent,
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

func TestUpdateTag_TitleOnly(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		// First request: GET to fetch existing tag
		if requestCount == 1 && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "tag456",
				"title":     "Old Title",
				"fillColor": "red",
			})
			return
		}

		// Second request: PATCH to update tag
		if r.Method == http.MethodPatch {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "tag456",
				"title":     "New Title",
				"fillColor": "red",
			})
			return
		}
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateTag(context.Background(), UpdateTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
		Title:   "New Title", // Only title, no color - should fetch existing
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Title != "New Title" {
		t.Errorf("Title = %q, want 'New Title'", result.Title)
	}
}

func TestUpdateTag_ColorOnly(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		// First request: GET to fetch existing tag
		if requestCount == 1 && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "tag456",
				"title":     "Existing Title",
				"fillColor": "red",
			})
			return
		}

		// Second request: PATCH to update tag
		if r.Method == http.MethodPatch {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "tag456",
				"title":     "Existing Title",
				"fillColor": "blue",
			})
			return
		}
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateTag(context.Background(), UpdateTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
		Color:   "blue", // Only color, no title - should fetch existing
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Color != "blue" {
		t.Errorf("Color = %q, want 'blue'", result.Color)
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

func TestDeleteTag_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    DeleteTagArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    DeleteTagArgs{TagID: "tag123"},
			errText: "board_id is required",
		},
		{
			name:    "empty tag_id",
			args:    DeleteTagArgs{BoardID: "board123"},
			errText: "tag_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.DeleteTag(context.Background(), tt.args)
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

func TestCreateText_WithStyle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify style is included
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected style in request body")
		} else {
			if style["fontSize"] != "24" {
				t.Errorf("fontSize = %v, want '24'", style["fontSize"])
			}
			if style["color"] != "#ff0000" {
				t.Errorf("color = %v, want '#ff0000'", style["color"])
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "text-styled",
			"data": map[string]interface{}{
				"content": "Styled text",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateText(context.Background(), CreateTextArgs{
		BoardID:  "board123",
		Content:  "Styled text",
		FontSize: 24,
		Color:    "#ff0000",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "text-styled" {
		t.Errorf("ID = %q, want 'text-styled'", result.ID)
	}
}

func TestCreateText_WithGeometryAndParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify geometry
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else if geom["width"] != float64(300) {
			t.Errorf("width = %v, want 300", geom["width"])
		}
		// Verify parent
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected parent in request body")
		} else if parent["id"] != "frame123" {
			t.Errorf("parent.id = %v, want 'frame123'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "text-geom",
			"data": map[string]interface{}{
				"content": "Text with width",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateText(context.Background(), CreateTextArgs{
		BoardID:  "board123",
		Content:  "Text with width",
		Width:    300,
		ParentID: "frame123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "text-geom" {
		t.Errorf("ID = %q, want 'text-geom'", result.ID)
	}
}

func TestListConnectors_LimitBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		inputLimit  int
		expectLimit string
	}{
		{"zero limit defaults to 50", 0, "50"},
		{"limit below 10 becomes 10", 5, "10"},
		{"limit above 100 becomes 100", 200, "100"},
		{"valid limit passes through", 30, "30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				limit := r.URL.Query().Get("limit")
				if limit != tt.expectLimit {
					t.Errorf("limit = %q, want %q", limit, tt.expectLimit)
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": []interface{}{},
				})
			}))
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			_, err := client.ListConnectors(context.Background(), ListConnectorsArgs{
				BoardID: "board123",
				Limit:   tt.inputLimit,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDeleteConnector_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    DeleteConnectorArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    DeleteConnectorArgs{ConnectorID: "conn123"},
			errText: "board_id is required",
		},
		{
			name:    "empty connector_id",
			args:    DeleteConnectorArgs{BoardID: "board123"},
			errText: "connector_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.DeleteConnector(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestDeleteConnector_SuccessPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/connectors/conn456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteConnector(context.Background(), DeleteConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success to be true")
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

func TestCreateFrame_DefaultDimensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify default dimensions
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else {
			if geom["width"] != float64(800) {
				t.Errorf("default width = %v, want 800", geom["width"])
			}
			if geom["height"] != float64(600) {
				t.Errorf("default height = %v, want 600", geom["height"])
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "frame-defaults",
			"data": map[string]interface{}{"title": "Default Frame"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	// Width and Height are 0, should get defaults
	result, err := client.CreateFrame(context.Background(), CreateFrameArgs{
		BoardID: "board123",
		Title:   "Default Frame",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "frame-defaults" {
		t.Errorf("ID = %q, want 'frame-defaults'", result.ID)
	}
}

func TestCreateFrame_WithColor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify style with fillColor
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected style in request body")
		} else if style["fillColor"] != "#ffcc00" {
			t.Errorf("fillColor = %v, want '#ffcc00'", style["fillColor"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "frame-color",
			"data": map[string]interface{}{"title": "Colored Frame"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateFrame(context.Background(), CreateFrameArgs{
		BoardID: "board123",
		Title:   "Colored Frame",
		Color:   "#ffcc00",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "frame-color" {
		t.Errorf("ID = %q, want 'frame-color'", result.ID)
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

func TestGetConnector_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.GetConnector(context.Background(), GetConnectorArgs{
		BoardID:     "",
		ConnectorID: "conn456",
	})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestGetConnector_EmptyConnectorID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.GetConnector(context.Background(), GetConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "",
	})

	if err == nil {
		t.Fatal("expected error for empty connector_id")
	}
	if !strings.Contains(err.Error(), "connector_id is required") {
		t.Errorf("expected 'connector_id is required' error, got: %v", err)
	}
}

func TestGetConnector_WithAllDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "conn456",
			"shape": "elbowed",
			"startItem": map[string]interface{}{
				"item": "start123",
			},
			"endItem": map[string]interface{}{
				"item": "end456",
			},
			"style": map[string]interface{}{
				"startStrokeCap": "arrow",
				"endStrokeCap":   "stealth",
				"color":          "#FF0000",
			},
			"captions": []map[string]interface{}{
				{"content": "Label text"},
			},
			"createdAt":  "2024-01-15T10:00:00Z",
			"modifiedAt": "2024-01-16T15:30:00Z",
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
	if result.StartCap != "arrow" {
		t.Errorf("StartCap = %q, want 'arrow'", result.StartCap)
	}
	if result.EndCap != "stealth" {
		t.Errorf("EndCap = %q, want 'stealth'", result.EndCap)
	}
	if result.Color != "#FF0000" {
		t.Errorf("Color = %q, want '#FF0000'", result.Color)
	}
	if result.Caption != "Label text" {
		t.Errorf("Caption = %q, want 'Label text'", result.Caption)
	}
	if result.CreatedAt == "" {
		t.Error("CreatedAt should be set")
	}
	if result.ModifiedAt == "" {
		t.Error("ModifiedAt should be set")
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

func TestFindBoardByName_EmptyName(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{
		Name: "",
	})

	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected 'required' error, got: %v", err)
	}
}

func TestFindBoardByName_StartsWithMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return boards where none is an exact match but one starts with the query
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":       "board2",
					"name":     "Something else",
					"viewLink": "https://miro.com/board2",
				},
				{
					"id":       "board1",
					"name":     "Sprint Planning Q1",
					"viewLink": "https://miro.com/board1",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{
		Name: "Sprint",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "board1" {
		t.Errorf("ID = %q, want 'board1'", result.ID)
	}
}

func TestFindBoardByName_ContainsMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return boards where none is an exact match or starts with, but one contains the query
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":       "board2",
					"name":     "Other board",
					"viewLink": "https://miro.com/board2",
				},
				{
					"id":       "board1",
					"name":     "Q1 Sprint Review",
					"viewLink": "https://miro.com/board1",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{
		Name: "Sprint",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "board1" {
		t.Errorf("ID = %q, want 'board1' (contains match)", result.ID)
	}
}

func TestFindBoardByName_Fallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return boards where none matches any criteria, should return first
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":       "board1",
					"name":     "Random Board ABC",
					"viewLink": "https://miro.com/board1",
				},
				{
					"id":       "board2",
					"name":     "Another Board XYZ",
					"viewLink": "https://miro.com/board2",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{
		Name: "Something completely different",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return first board as fallback
	if result.ID != "board1" {
		t.Errorf("ID = %q, want 'board1' (fallback)", result.ID)
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
// Mindmap Get/List/Delete Tests
// =============================================================================

func TestGetMindmapNode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/mindmap_nodes/") {
			t.Errorf("expected mindmap_nodes path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "node123",
			"position": map[string]interface{}{
				"x": 100.0,
				"y": 200.0,
			},
			"data": map[string]interface{}{
				"isRoot": true,
				"nodeView": map[string]interface{}{
					"type": "text",
					"data": map[string]interface{}{
						"content": "Root Node",
					},
				},
			},
			"children": []map[string]interface{}{
				{"id": "child1"},
				{"id": "child2"},
			},
			"createdAt":  "2024-01-01T00:00:00Z",
			"modifiedAt": "2024-01-02T00:00:00Z",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetMindmapNode(context.Background(), GetMindmapNodeArgs{
		BoardID: "board123",
		NodeID:  "node123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "node123" {
		t.Errorf("ID = %q, want 'node123'", result.ID)
	}
	if result.Content != "Root Node" {
		t.Errorf("Content = %q, want 'Root Node'", result.Content)
	}
	if !result.IsRoot {
		t.Error("IsRoot = false, want true")
	}
	if len(result.ChildIDs) != 2 {
		t.Errorf("ChildIDs count = %d, want 2", len(result.ChildIDs))
	}
	if result.X != 100.0 {
		t.Errorf("X = %f, want 100.0", result.X)
	}
}

func TestGetMindmapNode_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    GetMindmapNodeArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    GetMindmapNodeArgs{NodeID: "node123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty node ID",
			args:    GetMindmapNodeArgs{BoardID: "board123"},
			wantErr: "node_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetMindmapNode(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestListMindmapNodes_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/mindmap_nodes") {
			t.Errorf("expected mindmap_nodes path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id": "node1",
					"data": map[string]interface{}{
						"isRoot": true,
						"nodeView": map[string]interface{}{
							"data": map[string]interface{}{
								"content": "Root",
							},
						},
					},
				},
				{
					"id": "node2",
					"data": map[string]interface{}{
						"isRoot": false,
						"nodeView": map[string]interface{}{
							"data": map[string]interface{}{
								"content": "Child",
							},
						},
					},
					"parent": map[string]interface{}{
						"id": "node1",
					},
				},
			},
			"cursor": "",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if len(result.Nodes) != 2 {
		t.Errorf("Nodes count = %d, want 2", len(result.Nodes))
	}
	if result.Nodes[0].ID != "node1" {
		t.Errorf("Nodes[0].ID = %q, want 'node1'", result.Nodes[0].ID)
	}
	if !result.Nodes[0].IsRoot {
		t.Error("Nodes[0].IsRoot = false, want true")
	}
	if result.Nodes[1].ParentID != "node1" {
		t.Errorf("Nodes[1].ParentID = %q, want 'node1'", result.Nodes[1].ParentID)
	}
}

func TestListMindmapNodes_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	_, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("error = %q, want containing 'board_id is required'", err.Error())
	}
}

func TestListMindmapNodes_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id": "node1",
					"data": map[string]interface{}{
						"isRoot": true,
						"nodeView": map[string]interface{}{
							"data": map[string]interface{}{
								"content": "Node",
							},
						},
					},
				},
			},
			"cursor": "next_page",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasMore {
		t.Error("HasMore = false, want true")
	}
}

func TestDeleteMindmapNode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/mindmap_nodes/") {
			t.Errorf("expected mindmap_nodes path, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteMindmapNode(context.Background(), DeleteMindmapNodeArgs{
		BoardID: "board123",
		NodeID:  "node123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success = false, want true")
	}
	if result.ID != "node123" {
		t.Errorf("ID = %q, want 'node123'", result.ID)
	}
}

func TestDeleteMindmapNode_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    DeleteMindmapNodeArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    DeleteMindmapNodeArgs{NodeID: "node123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty node ID",
			args:    DeleteMindmapNodeArgs{BoardID: "board123"},
			wantErr: "node_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.DeleteMindmapNode(context.Background(), tt.args)
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
		{
			name:    "no updates provided",
			args:    UpdateConnectorArgs{BoardID: "board123", ConnectorID: "conn123"},
			wantErr: "at least one update field is required",
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

func TestUpdateConnector_WithStyle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify shape (style) field
		if body["shape"] != "curved" {
			t.Errorf("shape = %v, want 'curved'", body["shape"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "conn123"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateConnector(context.Background(), UpdateConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn123",
		Style:       "curved",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success to be true")
	}
}

func TestUpdateConnector_WithCapsAndColor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify style object with caps and color
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected style object in request body")
		} else {
			if style["startStrokeCap"] != "arrow" {
				t.Errorf("startStrokeCap = %v, want 'arrow'", style["startStrokeCap"])
			}
			if style["endStrokeCap"] != "stealth" {
				t.Errorf("endStrokeCap = %v, want 'stealth'", style["endStrokeCap"])
			}
			if style["strokeColor"] != "#ff0000" {
				t.Errorf("strokeColor = %v, want '#ff0000'", style["strokeColor"])
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "conn123"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateConnector(context.Background(), UpdateConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn123",
		StartCap:    "arrow",
		EndCap:      "stealth",
		Color:       "#ff0000",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success to be true")
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

// =============================================================================
// GetTag Tests
// =============================================================================

func TestGetTag_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/tags/tag456" {
			t.Errorf("expected /boards/board123/tags/tag456, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        "tag456",
			"title":     "Urgent",
			"fillColor": "red",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetTag(context.Background(), GetTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "tag456" {
		t.Errorf("ID = %q, want 'tag456'", result.ID)
	}
	if result.Title != "Urgent" {
		t.Errorf("Title = %q, want 'Urgent'", result.Title)
	}
	if result.Color != "red" {
		t.Errorf("Color = %q, want 'red'", result.Color)
	}
}

func TestGetTag_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    GetTagArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    GetTagArgs{TagID: "tag123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty tag ID",
			args:    GetTagArgs{BoardID: "board123"},
			wantErr: "tag_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetTag(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestGetTag_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "not_found",
			"message": "Tag not found",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetTag(context.Background(), GetTagArgs{
		BoardID: "board123",
		TagID:   "nonexistent",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected not found error, got: %v", err)
	}
}

// =============================================================================
// Frame Operations Tests
// =============================================================================

func TestGetFrame_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/frames/frame456" {
			t.Errorf("expected /boards/board123/frames/frame456, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "frame456",
			"type": "frame",
			"data": map[string]interface{}{
				"title": "Sprint Planning",
			},
			"position": map[string]interface{}{
				"x": 100.0,
				"y": 200.0,
			},
			"geometry": map[string]interface{}{
				"width":  800.0,
				"height": 600.0,
			},
			"style": map[string]interface{}{
				"fillColor": "#FFFFFF",
			},
			"children": []string{"child1", "child2"},
			"createdAt":  "2024-01-01T10:00:00Z",
			"modifiedAt": "2024-01-02T15:30:00Z",
			"createdBy":  map[string]interface{}{"id": "user1"},
			"modifiedBy": map[string]interface{}{"id": "user2"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetFrame(context.Background(), GetFrameArgs{
		BoardID: "board123",
		FrameID: "frame456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "frame456" {
		t.Errorf("ID = %q, want 'frame456'", result.ID)
	}
	if result.Title != "Sprint Planning" {
		t.Errorf("Title = %q, want 'Sprint Planning'", result.Title)
	}
	if result.X != 100 {
		t.Errorf("X = %f, want 100", result.X)
	}
	if result.Y != 200 {
		t.Errorf("Y = %f, want 200", result.Y)
	}
	if result.Width != 800 {
		t.Errorf("Width = %f, want 800", result.Width)
	}
	if result.Height != 600 {
		t.Errorf("Height = %f, want 600", result.Height)
	}
	if result.ChildCount != 2 {
		t.Errorf("ChildCount = %d, want 2", result.ChildCount)
	}
}

func TestGetFrame_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    GetFrameArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    GetFrameArgs{FrameID: "frame123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty frame ID",
			args:    GetFrameArgs{BoardID: "board123"},
			wantErr: "frame_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetFrame(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestUpdateFrame_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/frames/frame456" {
			t.Errorf("expected /boards/board123/frames/frame456, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if data, ok := body["data"].(map[string]interface{}); ok {
			if data["title"] != "Updated Title" {
				t.Errorf("title = %v, want 'Updated Title'", data["title"])
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "frame456",
			"data": map[string]interface{}{
				"title": "Updated Title",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	title := "Updated Title"
	result, err := client.UpdateFrame(context.Background(), UpdateFrameArgs{
		BoardID: "board123",
		FrameID: "frame456",
		Title:   &title,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.ID != "frame456" {
		t.Errorf("ID = %q, want 'frame456'", result.ID)
	}
}

func TestUpdateFrame_NoUpdates(t *testing.T) {
	client := newTestClientWithServer("http://localhost")
	_, err := client.UpdateFrame(context.Background(), UpdateFrameArgs{
		BoardID: "board123",
		FrameID: "frame456",
		// No update fields provided
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "at least one update field is required") {
		t.Errorf("expected 'at least one update field is required', got: %v", err)
	}
}

func TestUpdateFrame_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    UpdateFrameArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    UpdateFrameArgs{FrameID: "frame123", Title: ptrString("Test")},
			errText: "board_id is required",
		},
		{
			name:    "empty frame_id",
			args:    UpdateFrameArgs{BoardID: "board123", Title: ptrString("Test")},
			errText: "frame_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UpdateFrame(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestUpdateFrame_WithPositionAndGeometry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		// Verify request body contains position and geometry
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["position"] == nil {
			t.Error("expected position in body")
		}
		if body["geometry"] == nil {
			t.Error("expected geometry in body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "frame456",
			"position": map[string]interface{}{
				"x": 100,
				"y": 200,
			},
			"geometry": map[string]interface{}{
				"width":  800,
				"height": 600,
			},
		})
	}))
	defer server.Close()

	x := float64(100)
	y := float64(200)
	width := float64(800)
	height := float64(600)

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateFrame(context.Background(), UpdateFrameArgs{
		BoardID: "board123",
		FrameID: "frame456",
		X:       &x,
		Y:       &y,
		Width:   &width,
		Height:  &height,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestUpdateFrame_WithColor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["style"] == nil {
			t.Error("expected style in body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "frame456",
			"style": map[string]interface{}{
				"fillColor": "#FF0000",
			},
		})
	}))
	defer server.Close()

	color := "#FF0000"
	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateFrame(context.Background(), UpdateFrameArgs{
		BoardID: "board123",
		FrameID: "frame456",
		Color:   &color,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestDeleteFrame_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/frames/frame456" {
			t.Errorf("expected /boards/board123/frames/frame456, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteFrame(context.Background(), DeleteFrameArgs{
		BoardID: "board123",
		FrameID: "frame456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.ID != "frame456" {
		t.Errorf("ID = %q, want 'frame456'", result.ID)
	}
}

func TestDeleteFrame_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    DeleteFrameArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    DeleteFrameArgs{FrameID: "frame123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty frame ID",
			args:    DeleteFrameArgs{BoardID: "board123"},
			wantErr: "frame_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.DeleteFrame(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestGetFrameItems_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards/board123/frames/frame456/items") {
			t.Errorf("expected /boards/board123/frames/frame456/items, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "item1",
					"type": "sticky_note",
					"data": map[string]interface{}{
						"content": "Sticky content",
					},
				},
				{
					"id":   "item2",
					"type": "shape",
					"data": map[string]interface{}{
						"content": "Shape content",
					},
				},
			},
			"cursor": "next-cursor",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetFrameItems(context.Background(), GetFrameItemsArgs{
		BoardID: "board123",
		FrameID: "frame456",
		Limit:   50,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if !result.HasMore {
		t.Error("HasMore should be true when cursor is present")
	}
	if result.Items[0].ID != "item1" {
		t.Errorf("first item ID = %q, want 'item1'", result.Items[0].ID)
	}
}

func TestGetFrameItems_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    GetFrameItemsArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    GetFrameItemsArgs{FrameID: "frame123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty frame ID",
			args:    GetFrameItemsArgs{BoardID: "board123"},
			wantErr: "frame_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetFrameItems(context.Background(), tt.args)
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
// Group Operations Tests
// =============================================================================

func TestListGroups_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards/board123/groups") {
			t.Errorf("expected /boards/board123/groups, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":    "group1",
					"items": []string{"item1", "item2"},
				},
				{
					"id":    "group2",
					"items": []string{"item3"},
				},
			},
			"cursor": "",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListGroups(context.Background(), ListGroupsArgs{
		BoardID: "board123",
		Limit:   50,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if result.Groups[0].ID != "group1" {
		t.Errorf("first group ID = %q, want 'group1'", result.Groups[0].ID)
	}
}

func TestGetGroup_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/groups/group456" {
			t.Errorf("expected /boards/board123/groups/group456, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "group456",
			"items": []string{"item1", "item2", "item3"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetGroup(context.Background(), GetGroupArgs{
		BoardID: "board123",
		GroupID: "group456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "group456" {
		t.Errorf("ID = %q, want 'group456'", result.ID)
	}
	if len(result.Items) != 3 {
		t.Errorf("Items count = %d, want 3", len(result.Items))
	}
}

func TestGetGroup_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    GetGroupArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    GetGroupArgs{GroupID: "group123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty group ID",
			args:    GetGroupArgs{BoardID: "board123"},
			wantErr: "invalid group_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetGroup(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestGetGroupItems_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards/board123/groups/group456/items") {
			t.Errorf("expected /boards/board123/groups/group456/items, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "item1",
					"type": "sticky_note",
					"data": map[string]interface{}{
						"content": "First item",
					},
				},
				{
					"id":   "item2",
					"type": "shape",
					"data": map[string]interface{}{
						"content": "Second item",
					},
				},
			},
			"cursor": "",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetGroupItems(context.Background(), GetGroupItemsArgs{
		BoardID: "board123",
		GroupID: "group456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if len(result.Items) != 2 {
		t.Errorf("Items count = %d, want 2", len(result.Items))
	}
	if result.Items[0].ID != "item1" {
		t.Errorf("Items[0].ID = %q, want 'item1'", result.Items[0].ID)
	}
	if result.Items[0].Type != "sticky_note" {
		t.Errorf("Items[0].Type = %q, want 'sticky_note'", result.Items[0].Type)
	}
	if result.HasMore {
		t.Error("HasMore = true, want false")
	}
}

func TestGetGroupItems_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    GetGroupItemsArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    GetGroupItemsArgs{GroupID: "group123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty group ID",
			args:    GetGroupItemsArgs{BoardID: "board123"},
			wantErr: "invalid group_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetGroupItems(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestGetGroupItems_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "item1",
					"type": "sticky_note",
					"data": map[string]interface{}{
						"content": "Item",
					},
				},
			},
			"cursor": "next_page_token",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetGroupItems(context.Background(), GetGroupItemsArgs{
		BoardID: "board123",
		GroupID: "group456",
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasMore {
		t.Error("HasMore = false, want true")
	}
}

func TestDeleteGroup_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/groups/group456" {
			t.Errorf("expected /boards/board123/groups/group456, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteGroup(context.Background(), DeleteGroupArgs{
		BoardID: "board123",
		GroupID: "group456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.GroupID != "group456" {
		t.Errorf("GroupID = %q, want 'group456'", result.GroupID)
	}
}

func TestDeleteGroup_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    DeleteGroupArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    DeleteGroupArgs{GroupID: "group123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty group ID",
			args:    DeleteGroupArgs{BoardID: "board123"},
			wantErr: "invalid group_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.DeleteGroup(context.Background(), tt.args)
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
// Bulk Operations Tests
// =============================================================================

func TestBulkUpdate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:],
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	content1 := "Updated content 1"
	content2 := "Updated content 2"
	result, err := client.BulkUpdate(context.Background(), BulkUpdateArgs{
		BoardID: "board123",
		Items: []BulkUpdateItem{
			{ItemID: "item1", Content: &content1},
			{ItemID: "item2", Content: &content2},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Updated != 2 {
		t.Errorf("Updated = %d, want 2", result.Updated)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors count = %d, want 0", len(result.Errors))
	}
}

func TestBulkUpdate_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    BulkUpdateArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    BulkUpdateArgs{Items: []BulkUpdateItem{{ItemID: "item1"}}},
			wantErr: "board_id is required",
		},
		{
			name:    "empty items",
			args:    BulkUpdateArgs{BoardID: "board123", Items: []BulkUpdateItem{}},
			wantErr: "at least one item is required",
		},
		{
			name: "too many items",
			args: BulkUpdateArgs{
				BoardID: "board123",
				Items: func() []BulkUpdateItem {
					items := make([]BulkUpdateItem, 21)
					for i := range items {
						items[i] = BulkUpdateItem{ItemID: fmt.Sprintf("item%d", i)}
					}
					return items
				}(),
			},
			wantErr: "maximum 20 items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.BulkUpdate(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestBulkDelete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.BulkDelete(context.Background(), BulkDeleteArgs{
		BoardID: "board123",
		ItemIDs: []string{"item1", "item2", "item3"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Deleted != 3 {
		t.Errorf("Deleted = %d, want 3", result.Deleted)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors count = %d, want 0", len(result.Errors))
	}
}

func TestBulkDelete_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    BulkDeleteArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    BulkDeleteArgs{ItemIDs: []string{"item1"}},
			wantErr: "board_id is required",
		},
		{
			name:    "empty items",
			args:    BulkDeleteArgs{BoardID: "board123", ItemIDs: []string{}},
			wantErr: "at least one item_id is required",
		},
		{
			name: "too many items",
			args: BulkDeleteArgs{
				BoardID: "board123",
				ItemIDs: func() []string {
					ids := make([]string, 21)
					for i := range ids {
						ids[i] = fmt.Sprintf("item%d", i)
					}
					return ids
				}(),
			},
			wantErr: "maximum 20 items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.BulkDelete(context.Background(), tt.args)
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
// Member Operations Tests
// =============================================================================

func TestGetBoardMember_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/members/member456" {
			t.Errorf("expected /boards/board123/members/member456, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "member456",
			"name":  "John Doe",
			"email": "john@example.com",
			"role":  "editor",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetBoardMember(context.Background(), GetBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "member456" {
		t.Errorf("ID = %q, want 'member456'", result.ID)
	}
	if result.Name != "John Doe" {
		t.Errorf("Name = %q, want 'John Doe'", result.Name)
	}
	if result.Role != "editor" {
		t.Errorf("Role = %q, want 'editor'", result.Role)
	}
}

func TestRemoveBoardMember_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/members/member456" {
			t.Errorf("expected /boards/board123/members/member456, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.RemoveBoardMember(context.Background(), RemoveBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestUpdateBoardMember_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/members/member456" {
			t.Errorf("expected /boards/board123/members/member456, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["role"] != "editor" {
			t.Errorf("role = %v, want 'editor'", body["role"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "member456",
			"name": "John Doe",
			"role": "editor",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
		Role:     "editor",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "member456" {
		t.Errorf("ID = %q, want 'member456'", result.ID)
	}
	if result.Role != "editor" {
		t.Errorf("Role = %q, want 'editor'", result.Role)
	}
}

// =============================================================================
// UpdateBoard Tests
// =============================================================================

func TestUpdateBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123" {
			t.Errorf("expected /boards/board123, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "Updated Board Name" {
			t.Errorf("name = %v, want 'Updated Board Name'", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "board123",
			"name":        "Updated Board Name",
			"description": "Updated description",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateBoard(context.Background(), UpdateBoardArgs{
		BoardID:     "board123",
		Name:        "Updated Board Name",
		Description: "Updated description",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "board123" {
		t.Errorf("ID = %q, want 'board123'", result.ID)
	}
	if result.Name != "Updated Board Name" {
		t.Errorf("Name = %q, want 'Updated Board Name'", result.Name)
	}
}

func TestUpdateBoard_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    UpdateBoardArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    UpdateBoardArgs{Name: "New Name"},
			wantErr: "board_id is required",
		},
		{
			name:    "no updates",
			args:    UpdateBoardArgs{BoardID: "board123"},
			wantErr: "at least one of name or description is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UpdateBoard(context.Background(), tt.args)
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
// Client Utility Method Tests
// =============================================================================

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

// =============================================================================
// Path Utility Function Tests (additional cases not in circuitbreaker_test.go)
// =============================================================================

func TestSplitPath_AdditionalCases(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"", nil},
		{"/", nil},
		{"/boards", []string{"boards"}},
		{"boards/abc123", []string{"boards", "abc123"}}, // without leading slash
		{"/a/b/c", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := splitPath(tt.path)
			if len(got) != len(tt.want) {
				t.Errorf("splitPath(%q) = %v, want %v", tt.path, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitPath(%q)[%d] = %q, want %q", tt.path, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIndexOf_AdditionalCases(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   int
	}{
		{"hello world", "world", 6},
		{"hello world", "hello", 0},
		{"hello world", "x", -1},
		{"", "x", -1},
		{"abc", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.s+"/"+tt.substr, func(t *testing.T) {
			got := indexOf(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("indexOf(%q, %q) = %d, want %d", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestJoinPath_AdditionalCases(t *testing.T) {
	tests := []struct {
		parts []string
		want  string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"boards"}, "boards"},
		{[]string{"a", "b", "c", "d"}, "a/b/c/d"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := joinPath(tt.parts)
			if got != tt.want {
				t.Errorf("joinPath(%v) = %q, want %q", tt.parts, got, tt.want)
			}
		})
	}
}

// =============================================================================
// Type-Specific Update Tests
// =============================================================================

func strPtr(s string) *string {
	return &s
}

func floatPtr(f float64) *float64 {
	return &f
}

func TestUpdateSticky_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/sticky_notes/") {
			t.Errorf("expected /sticky_notes/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "sticky123",
			"data": map[string]interface{}{
				"content": "Updated content",
				"shape":   "square",
			},
			"style": map[string]interface{}{
				"fillColor": "light_blue",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
		BoardID: "board123",
		ItemID:  "sticky123",
		Content: strPtr("Updated content"),
		Color:   strPtr("blue"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "sticky123" {
		t.Errorf("ID = %q, want 'sticky123'", result.ID)
	}
	if result.Content != "Updated content" {
		t.Errorf("Content = %q, want 'Updated content'", result.Content)
	}
}

func TestUpdateSticky_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
		BoardID: "board123",
		ItemID:  "sticky123",
		// No fields to update
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateSticky_InvalidBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
		BoardID: "",
		ItemID:  "sticky123",
		Content: strPtr("test"),
	})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
}

func TestUpdateSticky_InvalidItemID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
		BoardID: "board123",
		ItemID:  "",
		Content: strPtr("test"),
	})

	if err == nil {
		t.Fatal("expected error for empty item_id")
	}
}

func TestUpdateSticky_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		if data, ok := body["data"].(map[string]interface{}); !ok {
			t.Error("expected data in request body")
		} else {
			if data["content"] != "Updated content" {
				t.Errorf("content = %v, want 'Updated content'", data["content"])
			}
			if data["shape"] != "circle" {
				t.Errorf("shape = %v, want 'circle'", data["shape"])
			}
		}

		// Verify style section
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected style in request body")
		} else if style["fillColor"] != "blue" {
			t.Errorf("fillColor = %v, want 'blue'", style["fillColor"])
		}

		// Verify position section
		if pos, ok := body["position"].(map[string]interface{}); !ok {
			t.Error("expected position in request body")
		} else {
			if pos["x"] != float64(100) {
				t.Errorf("x = %v, want 100", pos["x"])
			}
			if pos["y"] != float64(200) {
				t.Errorf("y = %v, want 200", pos["y"])
			}
		}

		// Verify geometry section
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else if geom["width"] != float64(300) {
			t.Errorf("width = %v, want 300", geom["width"])
		}

		// Verify parent section
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected parent in request body")
		} else if parent["id"] != "frame123" {
			t.Errorf("parent.id = %v, want 'frame123'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "sticky123",
			"data":  map[string]interface{}{"content": "Updated content", "shape": "circle"},
			"style": map[string]interface{}{"fillColor": "blue"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	x := float64(100)
	y := float64(200)
	width := float64(300)
	parentID := "frame123"
	content := "Updated content"
	shape := "circle"
	color := "blue"

	result, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
		BoardID:  "board123",
		ItemID:   "sticky123",
		Content:  &content,
		Shape:    &shape,
		Color:    &color,
		X:        &x,
		Y:        &y,
		Width:    &width,
		ParentID: &parentID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "sticky123" {
		t.Errorf("ID = %v, want 'sticky123'", result.ID)
	}
}

func TestUpdateSticky_RemoveFromFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify parent is null when empty string provided
		if parent, exists := body["parent"]; !exists {
			t.Error("expected parent in request body")
		} else if parent != nil {
			t.Errorf("parent = %v, want nil", parent)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "sticky123",
			"data":  map[string]interface{}{"content": "test"},
			"style": map[string]interface{}{"fillColor": "yellow"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	emptyParent := ""

	result, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
		BoardID:  "board123",
		ItemID:   "sticky123",
		ParentID: &emptyParent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "sticky123" {
		t.Errorf("ID = %v, want 'sticky123'", result.ID)
	}
}

func TestUpdateShape_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/shapes/") {
			t.Errorf("expected /shapes/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "shape123",
			"data": map[string]interface{}{
				"content": "Updated shape",
				"shape":   "circle",
			},
			"style": map[string]interface{}{
				"fillColor": "#FF0000",
				"fontColor": "#FFFFFF",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateShape(context.Background(), UpdateShapeArgs{
		BoardID:   "board123",
		ItemID:    "shape123",
		Content:   strPtr("Updated shape"),
		ShapeType: strPtr("circle"),
		Color:     strPtr("#FF0000"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "shape123" {
		t.Errorf("ID = %q, want 'shape123'", result.ID)
	}
	if result.ShapeType != "circle" {
		t.Errorf("ShapeType = %q, want 'circle'", result.ShapeType)
	}
}

func TestUpdateShape_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateShape(context.Background(), UpdateShapeArgs{
		BoardID: "board123",
		ItemID:  "shape123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateShape_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		if data, ok := body["data"].(map[string]interface{}); !ok {
			t.Error("expected data in request body")
		} else {
			if data["content"] != "New shape content" {
				t.Errorf("content = %v, want 'New shape content'", data["content"])
			}
			if data["shape"] != "circle" {
				t.Errorf("shape = %v, want 'circle'", data["shape"])
			}
		}

		// Verify style section
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected style in request body")
		} else {
			if style["fillColor"] != "#FF0000" {
				t.Errorf("fillColor = %v, want '#FF0000'", style["fillColor"])
			}
			if style["fontColor"] != "#FFFFFF" {
				t.Errorf("fontColor = %v, want '#FFFFFF'", style["fontColor"])
			}
		}

		// Verify position section
		if pos, ok := body["position"].(map[string]interface{}); !ok {
			t.Error("expected position in request body")
		} else {
			if pos["x"] != float64(50) {
				t.Errorf("x = %v, want 50", pos["x"])
			}
			if pos["y"] != float64(75) {
				t.Errorf("y = %v, want 75", pos["y"])
			}
		}

		// Verify geometry section
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else {
			if geom["width"] != float64(200) {
				t.Errorf("width = %v, want 200", geom["width"])
			}
			if geom["height"] != float64(150) {
				t.Errorf("height = %v, want 150", geom["height"])
			}
		}

		// Verify parent section
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected parent in request body")
		} else if parent["id"] != "frame-xyz" {
			t.Errorf("parent.id = %v, want 'frame-xyz'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "shape123",
			"data": map[string]interface{}{"content": "New shape content", "shape": "circle"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	content := "New shape content"
	shapeType := "circle"
	color := "#FF0000"
	textColor := "#FFFFFF"
	x := float64(50)
	y := float64(75)
	width := float64(200)
	height := float64(150)
	parentID := "frame-xyz"

	result, err := client.UpdateShape(context.Background(), UpdateShapeArgs{
		BoardID:   "board123",
		ItemID:    "shape123",
		Content:   &content,
		ShapeType: &shapeType,
		Color:     &color,
		TextColor: &textColor,
		X:         &x,
		Y:         &y,
		Width:     &width,
		Height:    &height,
		ParentID:  &parentID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "shape123" {
		t.Errorf("ID = %v, want 'shape123'", result.ID)
	}
}

func TestUpdateShape_RemoveFromFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify parent is null
		if parent, exists := body["parent"]; !exists {
			t.Error("expected parent in request body")
		} else if parent != nil {
			t.Errorf("parent = %v, want nil", parent)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "shape123",
			"data": map[string]interface{}{"content": "test", "shape": "rectangle"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	emptyParent := ""

	result, err := client.UpdateShape(context.Background(), UpdateShapeArgs{
		BoardID:  "board123",
		ItemID:   "shape123",
		ParentID: &emptyParent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "shape123" {
		t.Errorf("ID = %v, want 'shape123'", result.ID)
	}
}

func TestUpdateText_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/texts/") {
			t.Errorf("expected /texts/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "text123",
			"data": map[string]interface{}{
				"content": "Updated text",
			},
			"style": map[string]interface{}{
				"fontSize":  "24",
				"fontColor": "#000000",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	fontSize := 24
	result, err := client.UpdateText(context.Background(), UpdateTextArgs{
		BoardID:  "board123",
		ItemID:   "text123",
		Content:  strPtr("Updated text"),
		FontSize: &fontSize,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "text123" {
		t.Errorf("ID = %q, want 'text123'", result.ID)
	}
	if result.Content != "Updated text" {
		t.Errorf("Content = %q, want 'Updated text'", result.Content)
	}
}

func TestUpdateText_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateText(context.Background(), UpdateTextArgs{
		BoardID: "board123",
		ItemID:  "text123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateText_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		if data, ok := body["data"].(map[string]interface{}); !ok {
			t.Error("expected data in request body")
		} else if data["content"] != "Updated text content" {
			t.Errorf("content = %v, want 'Updated text content'", data["content"])
		}

		// Verify style section
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected style in request body")
		} else {
			if style["fontSize"] != "18" {
				t.Errorf("fontSize = %v, want '18'", style["fontSize"])
			}
			if style["textAlign"] != "center" {
				t.Errorf("textAlign = %v, want 'center'", style["textAlign"])
			}
			if style["color"] != "#333333" {
				t.Errorf("color = %v, want '#333333'", style["color"])
			}
		}

		// Verify position section
		if pos, ok := body["position"].(map[string]interface{}); !ok {
			t.Error("expected position in request body")
		} else {
			if pos["x"] != float64(100) {
				t.Errorf("x = %v, want 100", pos["x"])
			}
			if pos["y"] != float64(200) {
				t.Errorf("y = %v, want 200", pos["y"])
			}
		}

		// Verify geometry section
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else if geom["width"] != float64(400) {
			t.Errorf("width = %v, want 400", geom["width"])
		}

		// Verify parent section
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected parent in request body")
		} else if parent["id"] != "frame-abc" {
			t.Errorf("parent.id = %v, want 'frame-abc'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "text123",
			"data":  map[string]interface{}{"content": "Updated text content"},
			"style": map[string]interface{}{"fontSize": "18"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	content := "Updated text content"
	fontSize := 18
	textAlign := "center"
	color := "#333333"
	x := float64(100)
	y := float64(200)
	width := float64(400)
	parentID := "frame-abc"

	result, err := client.UpdateText(context.Background(), UpdateTextArgs{
		BoardID:   "board123",
		ItemID:    "text123",
		Content:   &content,
		FontSize:  &fontSize,
		TextAlign: &textAlign,
		Color:     &color,
		X:         &x,
		Y:         &y,
		Width:     &width,
		ParentID:  &parentID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "text123" {
		t.Errorf("ID = %v, want 'text123'", result.ID)
	}
}

func TestUpdateText_RemoveFromFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify parent is null
		if parent, exists := body["parent"]; !exists {
			t.Error("expected parent in request body")
		} else if parent != nil {
			t.Errorf("parent = %v, want nil", parent)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "text123",
			"data":  map[string]interface{}{"content": "test"},
			"style": map[string]interface{}{"fontSize": "14"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	emptyParent := ""

	result, err := client.UpdateText(context.Background(), UpdateTextArgs{
		BoardID:  "board123",
		ItemID:   "text123",
		ParentID: &emptyParent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "text123" {
		t.Errorf("ID = %v, want 'text123'", result.ID)
	}
}

func TestUpdateCard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/cards/") {
			t.Errorf("expected /cards/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "card123",
			"data": map[string]interface{}{
				"title":       "Updated card",
				"description": "New description",
			},
			"dueDate": "2025-01-01",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateCard(context.Background(), UpdateCardArgs{
		BoardID:     "board123",
		ItemID:      "card123",
		Title:       strPtr("Updated card"),
		Description: strPtr("New description"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "card123" {
		t.Errorf("ID = %q, want 'card123'", result.ID)
	}
	if result.Title != "Updated card" {
		t.Errorf("Title = %q, want 'Updated card'", result.Title)
	}
}

func TestUpdateCard_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateCard(context.Background(), UpdateCardArgs{
		BoardID: "board123",
		ItemID:  "card123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateCard_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		if data, ok := body["data"].(map[string]interface{}); !ok {
			t.Error("expected data in request body")
		} else {
			if data["title"] != "Updated Title" {
				t.Errorf("title = %v, want 'Updated Title'", data["title"])
			}
			if data["description"] != "New description" {
				t.Errorf("description = %v, want 'New description'", data["description"])
			}
			if data["dueDate"] != "2025-12-31" {
				t.Errorf("dueDate = %v, want '2025-12-31'", data["dueDate"])
			}
		}

		// Verify position section
		if pos, ok := body["position"].(map[string]interface{}); !ok {
			t.Error("expected position in request body")
		} else {
			if pos["x"] != float64(150) {
				t.Errorf("x = %v, want 150", pos["x"])
			}
			if pos["y"] != float64(250) {
				t.Errorf("y = %v, want 250", pos["y"])
			}
		}

		// Verify geometry section
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else if geom["width"] != float64(350) {
			t.Errorf("width = %v, want 350", geom["width"])
		}

		// Verify parent section
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected parent in request body")
		} else if parent["id"] != "frame-card" {
			t.Errorf("parent.id = %v, want 'frame-card'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "card123",
			"data": map[string]interface{}{
				"title":       "Updated Title",
				"description": "New description",
				"dueDate":     "2025-12-31",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	title := "Updated Title"
	description := "New description"
	dueDate := "2025-12-31"
	x := float64(150)
	y := float64(250)
	width := float64(350)
	parentID := "frame-card"

	result, err := client.UpdateCard(context.Background(), UpdateCardArgs{
		BoardID:     "board123",
		ItemID:      "card123",
		Title:       &title,
		Description: &description,
		DueDate:     &dueDate,
		X:           &x,
		Y:           &y,
		Width:       &width,
		ParentID:    &parentID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "card123" {
		t.Errorf("ID = %v, want 'card123'", result.ID)
	}
}

func TestUpdateCard_ClearDueDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify dueDate is null when empty string provided
		if data, ok := body["data"].(map[string]interface{}); !ok {
			t.Error("expected data in request body")
		} else if data["dueDate"] != nil {
			t.Errorf("dueDate = %v, want nil", data["dueDate"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "card123",
			"data": map[string]interface{}{"title": "Test", "description": "", "dueDate": ""},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	emptyDueDate := ""

	result, err := client.UpdateCard(context.Background(), UpdateCardArgs{
		BoardID: "board123",
		ItemID:  "card123",
		DueDate: &emptyDueDate,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "card123" {
		t.Errorf("ID = %v, want 'card123'", result.ID)
	}
}

func TestUpdateCard_RemoveFromFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify parent is null
		if parent, exists := body["parent"]; !exists {
			t.Error("expected parent in request body")
		} else if parent != nil {
			t.Errorf("parent = %v, want nil", parent)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "card123",
			"data": map[string]interface{}{"title": "Test", "description": "", "dueDate": ""},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	emptyParent := ""

	result, err := client.UpdateCard(context.Background(), UpdateCardArgs{
		BoardID:  "board123",
		ItemID:   "card123",
		ParentID: &emptyParent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "card123" {
		t.Errorf("ID = %v, want 'card123'", result.ID)
	}
}

func TestUpdateGroup_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/groups/") {
			t.Errorf("expected /groups/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "group123",
			"itemIds": []string{"item1", "item2", "item3"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateGroup(context.Background(), UpdateGroupArgs{
		BoardID: "board123",
		GroupID: "group123",
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

func TestUpdateGroup_NoItemIDs(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.UpdateGroup(context.Background(), UpdateGroupArgs{
		BoardID: "board123",
		GroupID: "group123",
		ItemIDs: []string{}, // Empty item list
	})

	if err == nil {
		t.Fatal("expected error for empty item_ids")
	}
}

func TestUpdateGroup_InvalidBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.UpdateGroup(context.Background(), UpdateGroupArgs{
		BoardID: "",
		GroupID: "group123",
		ItemIDs: []string{"item1"},
	})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
}

func TestUpdateGroup_InvalidGroupID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.UpdateGroup(context.Background(), UpdateGroupArgs{
		BoardID: "board123",
		GroupID: "",
		ItemIDs: []string{"item1"},
	})

	if err == nil {
		t.Fatal("expected error for empty group_id")
	}
}

// =============================================================================
// ValidateToken Tests
// =============================================================================

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

func TestValidateToken_NoBoards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
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
	if result.ID != "validated" {
		t.Errorf("ID = %q, want 'validated' (default)", result.ID)
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

// =============================================================================
// Board Member Tests (coverage for members.go)
// =============================================================================

func TestGetBoardMember_WithEmptyName(t *testing.T) {
	// Tests the branch where member.Name == "" and email is used as fallback
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "member123",
			"name":  "", // Empty name
			"email": "test@example.com",
			"role":  "editor",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetBoardMember(context.Background(), GetBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Message should use email as fallback when name is empty
	if !strings.Contains(result.Message, "test@example.com") {
		t.Errorf("message should contain email when name is empty, got: %s", result.Message)
	}
}

func TestRemoveBoardMember_APIError(t *testing.T) {
	// Tests the error branch where API returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  403,
			"message": "Access denied",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.RemoveBoardMember(context.Background(), RemoveBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member123",
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if result.Success {
		t.Error("expected Success to be false")
	}
	if !strings.Contains(result.Message, "Failed to remove member") {
		t.Errorf("expected failure message, got: %s", result.Message)
	}
}

func TestUpdateBoardMember_WithEmptyName(t *testing.T) {
	// Tests the branch where member.Name == "" and email is used as fallback
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "member123",
			"name":  "", // Empty name
			"email": "user@example.com",
			"role":  "viewer",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member123",
		Role:     "viewer",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Message should use email as fallback when name is empty
	if !strings.Contains(result.Message, "user@example.com") {
		t.Errorf("message should contain email when name is empty, got: %s", result.Message)
	}
}

func TestListBoardMembers_EmptyResult(t *testing.T) {
	// Tests the branch for empty member list message
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":   []interface{}{},
			"total":  0,
			"offset": 0,
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
	if result.Message != "No members found on this board" {
		t.Errorf("expected 'No members found on this board', got: %s", result.Message)
	}
}

func TestListBoardMembers_WithOffset(t *testing.T) {
	// Tests the HasMore calculation when offset > 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return exactly limit items to trigger HasMore = true
		members := make([]map[string]interface{}, 50)
		for i := 0; i < 50; i++ {
			members[i] = map[string]interface{}{
				"id":    fmt.Sprintf("member%d", i),
				"name":  fmt.Sprintf("User %d", i),
				"email": fmt.Sprintf("user%d@example.com", i),
				"role":  "viewer",
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":   members,
			"total":  100,
			"offset": 50, // Non-zero offset indicates more pages
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListBoardMembers(context.Background(), ListBoardMembersArgs{
		BoardID: "board123",
		Limit:   50,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasMore {
		t.Error("expected HasMore to be true when offset > 0 and count >= limit")
	}
}

// =============================================================================
// Tag Tests (coverage for tags.go)
// =============================================================================

func TestDetachTag_APIError(t *testing.T) {
	// Tests the error branch for DetachTag
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  404,
			"message": "Tag not found",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "board123",
		ItemID:  "item123",
		TagID:   "tag123",
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if result.Success {
		t.Error("expected Success to be false")
	}
	if !strings.Contains(result.Message, "Failed to detach tag") {
		t.Errorf("expected failure message, got: %s", result.Message)
	}
}

func TestGetItemTags_EmptyTags(t *testing.T) {
	// Tests the branch where no tags are returned and message is "No tags on this item"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "board123",
		ItemID:  "item123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No tags on this item" {
		t.Errorf("expected 'No tags on this item', got: %s", result.Message)
	}
	if result.Count != 0 {
		t.Errorf("expected Count = 0, got %d", result.Count)
	}
}

func TestGetItemTags_NilData(t *testing.T) {
	// Tests the branch where data is null/nil (line 183-184)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return null data
		w.Write([]byte(`{"data": null}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "board123",
		ItemID:  "item123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Tags should be empty slice, not nil
	if result.Tags == nil {
		t.Error("expected Tags to be empty slice, not nil")
	}
	if len(result.Tags) != 0 {
		t.Errorf("expected empty Tags, got %d items", len(result.Tags))
	}
}

func TestListTags_EmptyBoard(t *testing.T) {
	// Tests the empty board message branch
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
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
	if result.Message != "No tags on this board" {
		t.Errorf("expected 'No tags on this board', got: %s", result.Message)
	}
}

// =============================================================================
// Group Tests (coverage for groups.go)
// =============================================================================

func TestListGroups_WithCursor(t *testing.T) {
	// Tests pagination with cursor
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "cursor=next123") {
			t.Error("expected cursor parameter in request")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "group1", "items": []string{"item1", "item2"}},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListGroups(context.Background(), ListGroupsArgs{
		BoardID: "board123",
		Cursor:  "next123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 1 {
		t.Errorf("expected 1 group, got %d", result.Count)
	}
}

func TestGetGroupItems_WithCursor(t *testing.T) {
	// Tests pagination with cursor
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "cursor=page2") {
			t.Error("expected cursor parameter in request")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "item1", "type": "sticky_note"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetGroupItems(context.Background(), GetGroupItemsArgs{
		BoardID: "board123",
		GroupID: "group123",
		Cursor:  "page2",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteGroup_APIError(t *testing.T) {
	// Tests the error branch for DeleteGroup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  403,
			"message": "Access denied",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteGroup(context.Background(), DeleteGroupArgs{
		BoardID: "board123",
		GroupID: "group123",
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if result.Success {
		t.Error("expected Success to be false")
	}
}

// =============================================================================
// Frame Tests (coverage for frames.go)
// =============================================================================

func TestGetFrameItems_WithCursor(t *testing.T) {
	// Tests pagination with cursor
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "cursor=nextpage") {
			t.Error("expected cursor parameter in request")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "item1", "type": "sticky_note"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetFrameItems(context.Background(), GetFrameItemsArgs{
		BoardID: "board123",
		FrameID: "frame123",
		Cursor:  "nextpage",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetFrameItems_WithTypeFilter(t *testing.T) {
	// Tests type filtering
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "type=sticky_note") {
			t.Error("expected type parameter in request")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "sticky1", "type": "sticky_note"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetFrameItems(context.Background(), GetFrameItemsArgs{
		BoardID: "board123",
		FrameID: "frame123",
		Type:    "sticky_note",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Create Tests (coverage for create.go)
// =============================================================================

func TestCreateCard_WithAllFields(t *testing.T) {
	// Tests CreateCard with all optional fields to improve coverage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Error("expected 'data' field")
		}
		if data["title"] != "Test Card" {
			t.Errorf("title = %v, want 'Test Card'", data["title"])
		}
		if data["description"] != "Card description" {
			t.Errorf("description = %v, want 'Card description'", data["description"])
		}
		if data["dueDate"] != "2024-12-31" {
			t.Errorf("dueDate = %v, want '2024-12-31'", data["dueDate"])
		}

		// Verify position
		pos, ok := body["position"].(map[string]interface{})
		if !ok {
			t.Error("expected 'position' field")
		}
		if pos["x"] != float64(100) || pos["y"] != float64(200) {
			t.Errorf("position = %v, want x=100, y=200", pos)
		}

		// Verify geometry
		geom, ok := body["geometry"].(map[string]interface{})
		if !ok {
			t.Error("expected 'geometry' field")
		}
		if geom["width"] != float64(300) {
			t.Errorf("width = %v, want 300", geom["width"])
		}

		// Verify parent
		parent, ok := body["parent"].(map[string]interface{})
		if !ok {
			t.Error("expected 'parent' field")
		}
		if parent["id"] != "frame123" {
			t.Errorf("parent.id = %v, want 'frame123'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "card123",
			"type": "card",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateCard(context.Background(), CreateCardArgs{
		BoardID:     "board123",
		Title:       "Test Card",
		Description: "Card description",
		DueDate:     "2024-12-31",
		X:           100,
		Y:           200,
		Width:       300,
		ParentID:    "frame123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateImage_WithAllFields(t *testing.T) {
	// Tests CreateImage with all optional fields
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Error("expected 'data' field")
		}
		if data["url"] != "https://example.com/image.png" {
			t.Errorf("url = %v, want 'https://example.com/image.png'", data["url"])
		}
		if data["title"] != "Test Image" {
			t.Errorf("title = %v, want 'Test Image'", data["title"])
		}

		// Verify position and geometry
		if _, ok := body["position"]; !ok {
			t.Error("expected 'position' field")
		}
		if _, ok := body["geometry"]; !ok {
			t.Error("expected 'geometry' field")
		}
		if _, ok := body["parent"]; !ok {
			t.Error("expected 'parent' field")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "image123",
			"type": "image",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateImage(context.Background(), CreateImageArgs{
		BoardID:  "board123",
		URL:      "https://example.com/image.png",
		Title:    "Test Image",
		X:        100,
		Y:        200,
		Width:    400,
		ParentID: "frame123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateDocument_WithAllFields(t *testing.T) {
	// Tests CreateDocument with all optional fields
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Error("expected 'data' field")
		}
		if data["url"] != "https://example.com/doc.pdf" {
			t.Errorf("url = %v, want 'https://example.com/doc.pdf'", data["url"])
		}
		if data["title"] != "Test Document" {
			t.Errorf("title = %v, want 'Test Document'", data["title"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "doc123",
			"type": "document",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateDocument(context.Background(), CreateDocumentArgs{
		BoardID:  "board123",
		URL:      "https://example.com/doc.pdf",
		Title:    "Test Document",
		X:        100,
		Y:        200,
		Width:    500,
		ParentID: "frame123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateEmbed_WithAllFields(t *testing.T) {
	// Tests CreateEmbed with all optional fields
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Error("expected 'data' field")
		}
		if data["url"] != "https://youtube.com/watch?v=abc123" {
			t.Errorf("url = %v, want 'https://youtube.com/watch?v=abc123'", data["url"])
		}
		if data["mode"] != "modal" {
			t.Errorf("mode = %v, want 'modal'", data["mode"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "embed123",
			"type": "embed",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateEmbed(context.Background(), CreateEmbedArgs{
		BoardID:  "board123",
		URL:      "https://youtube.com/watch?v=abc123",
		Mode:     "modal",
		X:        100,
		Y:        200,
		Width:    640,
		Height:   480,
		ParentID: "frame123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Rate Limiter Tests
// =============================================================================

func TestCalculateDelay_SlowdownThreshold(t *testing.T) {
	// Test calculateDelay when below slowdown threshold
	config := RateLimiterConfig{
		MinDelay:          10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		ProactiveBuffer:   5,
		SlowdownThreshold: 0.3, // 30%
	}
	rl := NewAdaptiveRateLimiter(config)

	// State with 15% remaining (below 30% threshold) should cause delay
	state := RateLimitState{
		Limit:     100,
		Remaining: 15,
	}

	delay := rl.calculateDelay(state, config)
	if delay <= 0 {
		t.Errorf("expected delay > 0 when below threshold, got %v", delay)
	}
}

func TestCalculateDelay_AtBufferThreshold(t *testing.T) {
	// Test calculateDelay when at/below proactive buffer
	config := RateLimiterConfig{
		MinDelay:          10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		ProactiveBuffer:   10,
		SlowdownThreshold: 0.3,
	}
	rl := NewAdaptiveRateLimiter(config)

	// State at buffer (remaining <= ProactiveBuffer)
	state := RateLimitState{
		Limit:     100,
		Remaining: 5, // Below buffer of 10
		ResetAt:   time.Time{}, // Zero time - fallback case
	}

	delay := rl.calculateDelay(state, config)
	if delay != config.MaxDelay {
		t.Errorf("expected max delay %v at buffer threshold, got %v", config.MaxDelay, delay)
	}
}

func TestCalculateDelay_WaitUntilReset(t *testing.T) {
	// Test calculateDelay waiting until reset time
	config := RateLimiterConfig{
		MinDelay:          10 * time.Millisecond,
		MaxDelay:          500 * time.Millisecond,
		ProactiveBuffer:   10,
		SlowdownThreshold: 0.3,
	}
	rl := NewAdaptiveRateLimiter(config)

	// State at buffer with reset time in future
	resetTime := time.Now().Add(200 * time.Millisecond)
	state := RateLimitState{
		Limit:     100,
		Remaining: 5, // At buffer
		ResetAt:   resetTime,
	}

	delay := rl.calculateDelay(state, config)
	if delay <= 0 || delay > 250*time.Millisecond {
		t.Errorf("expected delay around 200ms, got %v", delay)
	}
}

func TestCalculateDelay_CapAtMaxDelay(t *testing.T) {
	// Test that delay is capped at MaxDelay when reset time is far away
	config := RateLimiterConfig{
		MinDelay:          10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		ProactiveBuffer:   10,
		SlowdownThreshold: 0.3,
	}
	rl := NewAdaptiveRateLimiter(config)

	// Reset time is 10 seconds in future, should cap at MaxDelay
	state := RateLimitState{
		Limit:     100,
		Remaining: 5,
		ResetAt:   time.Now().Add(10 * time.Second),
	}

	delay := rl.calculateDelay(state, config)
	if delay != config.MaxDelay {
		t.Errorf("expected delay capped at %v, got %v", config.MaxDelay, delay)
	}
}

// =============================================================================
// Circuit Breaker Tests
// =============================================================================

func TestCircuitBreaker_RecordFailureInHalfOpen(t *testing.T) {
	// Test RecordFailure when circuit is half-open
	config := CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    2,
		Timeout:             1 * time.Millisecond, // Very short for testing
		MaxHalfOpenRequests: 5,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit by recording failures
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for circuit to transition to half-open
	time.Sleep(5 * time.Millisecond)

	// Allow a request to set half-open state
	if err := cb.Allow(); err != nil {
		t.Errorf("expected circuit to allow request in half-open state: %v", err)
	}

	// Record failure in half-open state
	cb.RecordFailure()

	// Circuit should be open again
	if err := cb.Allow(); err == nil {
		t.Error("expected circuit to be open after failure in half-open state")
	}
}

// =============================================================================
// AppCard Tests
// =============================================================================

func TestGetAppCard_WithNilFields(t *testing.T) {
	// Tests GetAppCard when some fields are nil
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard123",
			"type": "app_card",
			"data": map[string]interface{}{
				"title":       "Test App Card",
				"description": "",
				"fields":      nil,
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetAppCard(context.Background(), GetAppCardArgs{
		BoardID: "board123",
		ItemID:  "appcard123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "appcard123" {
		t.Errorf("ID = %v, want 'appcard123'", result.ID)
	}
}

func TestUpdateAppCard_WithStatusOnly(t *testing.T) {
	// Tests UpdateAppCard with only status field
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify status is in the data section
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Error("expected 'data' field")
		}
		if data["status"] != "connected" {
			t.Errorf("status = %v, want 'connected'", data["status"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard123",
			"type": "app_card",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.UpdateAppCard(context.Background(), UpdateAppCardArgs{
		BoardID: "board123",
		ItemID:  "appcard123",
		Status:  "connected",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Board Tests
// =============================================================================

func TestListBoards_WithQueryAndTeamID(t *testing.T) {
	// Tests ListBoards with both query and team_id filters
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") != "design" {
			t.Errorf("query = %v, want 'design'", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("team_id") != "team123" {
			t.Errorf("team_id = %v, want 'team123'", r.URL.Query().Get("team_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "board1", "name": "Design Board"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListBoards(context.Background(), ListBoardsArgs{
		Query:  "design",
		TeamID: "team123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateBoard_WithDescription(t *testing.T) {
	// Tests CreateBoard with optional description
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "New Board" {
			t.Errorf("name = %v, want 'New Board'", body["name"])
		}
		if body["description"] != "Board description" {
			t.Errorf("description = %v, want 'Board description'", body["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "newboard123",
			"name": "New Board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateBoard(context.Background(), CreateBoardArgs{
		Name:        "New Board",
		Description: "Board description",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCopyBoard_WithNameAndDescription(t *testing.T) {
	// Tests CopyBoard with custom name and description
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "Copy of Board" {
			t.Errorf("name = %v, want 'Copy of Board'", body["name"])
		}
		if body["description"] != "Copied board" {
			t.Errorf("description = %v, want 'Copied board'", body["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "copyboard123",
			"name": "Copy of Board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CopyBoard(context.Background(), CopyBoardArgs{
		BoardID:     "board123",
		Name:        "Copy of Board",
		Description: "Copied board",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Mindmap Tests
// =============================================================================

func TestCreateMindmapNode_WithPosition(t *testing.T) {
	// Tests CreateMindmapNode with x, y position (root node)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Root node should have position
		if _, ok := body["position"]; !ok {
			t.Error("expected 'position' field for root node")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "node123",
			"type": "mindmap_node",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateMindmapNode(context.Background(), CreateMindmapNodeArgs{
		BoardID: "board123",
		Content: "Root Node",
		X:       100,
		Y:       200,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListMindmapNodes_WithCursor(t *testing.T) {
	// Tests ListMindmapNodes with cursor pagination
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") != "abc123" {
			t.Errorf("cursor = %v, want 'abc123'", r.URL.Query().Get("cursor"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "node1", "type": "mindmap_node"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
		Cursor:  "abc123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Tag Tests - Additional Branches
// =============================================================================

func TestCreateTag_WithDefaultColor(t *testing.T) {
	// Tests CreateTag when color is not specified (defaults to blue)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["fillColor"] != "blue" {
			t.Errorf("fillColor = %v, want 'blue' (default)", body["fillColor"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        "tag123",
			"title":     "My Tag",
			"fillColor": "blue",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateTag(context.Background(), CreateTagArgs{
		BoardID: "board123",
		Title:   "My Tag",
		// Color not specified - should default to blue
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Items Tests - Additional Branches
// =============================================================================

func TestGetItem_WithLinks(t *testing.T) {
	// Tests GetItem when item has links
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "item123",
			"type": "sticky_note",
			"links": map[string]interface{}{
				"self":   "https://api.miro.com/v2/boards/board123/items/item123",
				"related": "https://api.miro.com/v2/boards/board123",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetItem(context.Background(), GetItemArgs{
		BoardID: "board123",
		ItemID:  "item123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "item123" {
		t.Errorf("ID = %v, want 'item123'", result.ID)
	}
}

// =============================================================================
// Server Error Tests (5xx) - Circuit Breaker Coverage
// =============================================================================

func TestRequest_ServerError500_TripsCircuitBreaker(t *testing.T) {
	// Tests that 5xx errors trip the circuit breaker
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  500,
			"message": "Internal Server Error",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	// Make several requests to trigger circuit breaker
	for i := 0; i < 5; i++ {
		_, err := client.GetBoard(context.Background(), GetBoardArgs{BoardID: "board123"})
		if err == nil {
			t.Error("expected error for 500 response")
		}
	}

	// Verify requests were made
	if requestCount < 3 {
		t.Errorf("expected at least 3 requests, got %d", requestCount)
	}
}

func TestRequest_ClientError4xx_DoesNotTripCircuitBreaker(t *testing.T) {
	// Tests that 4xx errors do NOT trip the circuit breaker
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  404,
			"message": "Not Found",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	// Make requests - circuit breaker should NOT trip for 4xx
	for i := 0; i < 10; i++ {
		_, _ = client.GetBoard(context.Background(), GetBoardArgs{BoardID: "board123"})
	}

	// All 10 requests should have been made (circuit not tripped)
	if requestCount != 10 {
		t.Errorf("expected 10 requests (circuit should not trip for 4xx), got %d", requestCount)
	}
}

// =============================================================================
// ListConnectors Tests - Additional Coverage
// =============================================================================

func TestListConnectors_WithCursor(t *testing.T) {
	// Tests ListConnectors with cursor pagination
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") != "next-page-cursor" {
			t.Errorf("cursor = %v, want 'next-page-cursor'", r.URL.Query().Get("cursor"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "conn1", "type": "connector"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListConnectors(context.Background(), ListConnectorsArgs{
		BoardID: "board123",
		Cursor:  "next-page-cursor",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListConnectors_WithLimitParam(t *testing.T) {
	// Tests ListConnectors with limit parameter
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "25" {
			t.Errorf("limit = %v, want '25'", r.URL.Query().Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListConnectors(context.Background(), ListConnectorsArgs{
		BoardID: "board123",
		Limit:   25,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// CreateSticky Tests - Error Branch Coverage
// =============================================================================

func TestCreateSticky_EmptyBoardID(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	_, err := client.CreateSticky(context.Background(), CreateStickyArgs{
		BoardID: "",
		Content: "test",
	})

	if err == nil {
		t.Error("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id") {
		t.Errorf("error should mention board_id: %v", err)
	}
}

func TestCreateSticky_EmptyContent(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	_, err := client.CreateSticky(context.Background(), CreateStickyArgs{
		BoardID: "board123",
		Content: "",
	})

	if err == nil {
		t.Error("expected error for empty content")
	}
	if !strings.Contains(err.Error(), "content") {
		t.Errorf("error should mention content: %v", err)
	}
}

// =============================================================================
// CreateBoard Tests - Coverage for Optional Fields
// =============================================================================

func TestCreateBoard_WithTeamID(t *testing.T) {
	// Tests CreateBoard with teamId in request body (Miro uses camelCase)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "New Board" {
			t.Errorf("name = %v, want 'New Board'", body["name"])
		}
		if body["teamId"] != "team456" {
			t.Errorf("teamId = %v, want 'team456'", body["teamId"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "newboard123",
			"name": "New Board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateBoard(context.Background(), CreateBoardArgs{
		Name:   "New Board",
		TeamID: "team456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "newboard123" {
		t.Errorf("ID = %v, want 'newboard123'", result.ID)
	}
}

// =============================================================================
// GetBoardSummary Tests - Additional Coverage
// =============================================================================

func TestGetBoardSummary_WithManyItemTypes(t *testing.T) {
	// Tests GetBoardSummary with various item types
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/items") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "1", "type": "sticky_note"},
					{"id": "2", "type": "shape"},
					{"id": "3", "type": "text"},
					{"id": "4", "type": "connector"},
					{"id": "5", "type": "frame"},
					{"id": "6", "type": "card"},
					{"id": "7", "type": "image"},
					{"id": "8", "type": "document"},
					{"id": "9", "type": "embed"},
					{"id": "10", "type": "app_card"},
				},
			})
		} else {
			w.Header().Set("Content-Type", "application/json")
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
	if result.TotalItems != 10 {
		t.Errorf("TotalItems = %v, want 10", result.TotalItems)
	}
}

// =============================================================================
// CopyBoard Tests - Additional Coverage
// =============================================================================

func TestCopyBoard_WithDescription(t *testing.T) {
	// Tests CopyBoard with optional description
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["description"] != "Copied board description" {
			t.Errorf("description = %v, want 'Copied board description'", body["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "copied123",
			"name": "Board Copy",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CopyBoard(context.Background(), CopyBoardArgs{
		BoardID:     "original123",
		Name:        "Board Copy",
		Description: "Copied board description",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "copied123" {
		t.Errorf("ID = %v, want 'copied123'", result.ID)
	}
}

// =============================================================================
// UpdateBoard Tests - Additional Coverage
// =============================================================================

func TestUpdateBoard_WithBothNameAndDescription(t *testing.T) {
	// Tests UpdateBoard with both name and description
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "Updated Name" {
			t.Errorf("name = %v, want 'Updated Name'", body["name"])
		}
		if body["description"] != "Updated Description" {
			t.Errorf("description = %v, want 'Updated Description'", body["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "board123",
			"name":        "Updated Name",
			"description": "Updated Description",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateBoard(context.Background(), UpdateBoardArgs{
		BoardID:     "board123",
		Name:        "Updated Name",
		Description: "Updated Description",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated Name" {
		t.Errorf("Name = %v, want 'Updated Name'", result.Name)
	}
}

// =============================================================================
// ValidateItemID Tests - Edge Cases
// =============================================================================

func TestValidateItemID_TooLong(t *testing.T) {
	// Create an ID that exceeds max length (256 chars)
	longID := strings.Repeat("a", 300)
	err := ValidateItemID(longID)

	if err == nil {
		t.Error("expected error for ID that's too long")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("error should mention 'too long': %v", err)
	}
}

func TestValidateItemID_InvalidCharacters(t *testing.T) {
	// Test ID with invalid characters
	err := ValidateItemID("item<>123")

	if err == nil {
		t.Error("expected error for ID with invalid characters")
	}
	if !strings.Contains(err.Error(), "invalid characters") {
		t.Errorf("error should mention 'invalid characters': %v", err)
	}
}

// =============================================================================
// ListBoards Tests - Offset Coverage
// =============================================================================

func TestListBoards_WithOffset(t *testing.T) {
	// Tests ListBoards with offset for pagination
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("offset") != "20" {
			t.Errorf("offset = %v, want '20'", r.URL.Query().Get("offset"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListBoards(context.Background(), ListBoardsArgs{
		Offset: "20",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// CreateBoard Tests - Error Handling
// =============================================================================

func TestCreateBoard_EmptyNameValidation(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	_, err := client.CreateBoard(context.Background(), CreateBoardArgs{
		Name: "",
	})

	if err == nil {
		t.Error("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should mention 'name': %v", err)
	}
}

// =============================================================================
// CopyBoard Tests - Error Handling
// =============================================================================

func TestCopyBoard_EmptyBoardID(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	_, err := client.CopyBoard(context.Background(), CopyBoardArgs{
		BoardID: "",
	})

	if err == nil {
		t.Error("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id") {
		t.Errorf("error should mention 'board_id': %v", err)
	}
}

func TestCopyBoard_WithTeamID(t *testing.T) {
	// Tests CopyBoard with teamId (Miro uses camelCase)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["teamId"] != "team789" {
			t.Errorf("teamId = %v, want 'team789'", body["teamId"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "copied456",
			"name": "Copied Board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CopyBoard(context.Background(), CopyBoardArgs{
		BoardID: "original123",
		TeamID:  "team789",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// UpdateBoard Tests - Error Handling
// =============================================================================

func TestUpdateBoard_EmptyBoardID(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	_, err := client.UpdateBoard(context.Background(), UpdateBoardArgs{
		BoardID: "",
		Name:    "New Name",
	})

	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

func TestUpdateBoard_NoChanges(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	_, err := client.UpdateBoard(context.Background(), UpdateBoardArgs{
		BoardID: "board123",
		// No name or description provided
	})

	if err == nil {
		t.Error("expected error when no changes specified")
	}
}

// =============================================================================
// UpdateAppCard Tests - Field Coverage
// =============================================================================

func TestUpdateAppCard_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		data, _ := body["data"].(map[string]interface{})
		if data["title"] != "Updated Title" {
			t.Errorf("title = %v, want 'Updated Title'", data["title"])
		}
		if data["description"] != "Updated Desc" {
			t.Errorf("description = %v, want 'Updated Desc'", data["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard123",
			"type": "app_card",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.UpdateAppCard(context.Background(), UpdateAppCardArgs{
		BoardID:     "board123",
		ItemID:      "appcard123",
		Title:       "Updated Title",
		Description: "Updated Desc",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// RecordSuccess Tests - Circuit Breaker
// =============================================================================

func TestCircuitBreaker_RecordSuccessInHalfOpen(t *testing.T) {
	// Test RecordSuccess when circuit is half-open to transition to closed
	config := CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    2,
		Timeout:             1 * time.Millisecond,
		MaxHalfOpenRequests: 5,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for circuit to transition to half-open
	time.Sleep(5 * time.Millisecond)

	// Allow request in half-open
	if err := cb.Allow(); err != nil {
		t.Errorf("expected circuit to allow in half-open: %v", err)
	}

	// Record success
	cb.RecordSuccess()

	// Record another success to reach threshold
	if err := cb.Allow(); err != nil {
		t.Errorf("expected circuit to still allow: %v", err)
	}
	cb.RecordSuccess()

	// Circuit should now be closed
	state := cb.State()
	if state != CircuitClosed {
		t.Errorf("state = %v, want CircuitClosed", state)
	}
}

// =============================================================================
// GetAppCard Tests - Field Parsing
// =============================================================================

func TestGetAppCard_WithFields(t *testing.T) {
	// Tests GetAppCard with custom fields in response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard123",
			"type": "app_card",
			"data": map[string]interface{}{
				"title":       "Test Card",
				"description": "A test description",
				"status":      "connected",
				"fields": []map[string]interface{}{
					{"value": "Field 1", "fillColor": "#FF0000"},
					{"value": "Field 2", "fillColor": "#00FF00"},
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetAppCard(context.Background(), GetAppCardArgs{
		BoardID: "board123",
		ItemID:  "appcard123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "Test Card" {
		t.Errorf("Title = %v, want 'Test Card'", result.Title)
	}
}

// =============================================================================
// CreateAppCard Tests - Field Coverage
// =============================================================================

func TestCreateAppCard_WithMultipleFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		data, _ := body["data"].(map[string]interface{})
		fields, _ := data["fields"].([]interface{})
		if len(fields) != 2 {
			t.Errorf("fields count = %d, want 2", len(fields))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard456",
			"type": "app_card",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateAppCard(context.Background(), CreateAppCardArgs{
		BoardID: "board123",
		Title:   "New App Card",
		Fields: []AppCardField{
			{Value: "Field 1", FillColor: "#FF0000"},
			{Value: "Field 2", FillColor: "#00FF00"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Tests for RemoveBoardMember error paths
func TestRemoveBoardMember_EmptyMemberID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.RemoveBoardMember(context.Background(), RemoveBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "",
	})
	if err == nil {
		t.Error("expected error for empty member_id")
	}
	if !strings.Contains(err.Error(), "member_id is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRemoveBoardMember_APIError_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Member not found"}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.RemoveBoardMember(context.Background(), RemoveBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
	})
	if err == nil {
		t.Error("expected error for 404 response")
	}
	if result.Success {
		t.Error("expected Success=false on error")
	}
}

// Tests for UpdateBoardMember error paths
func TestUpdateBoardMember_EmptyMemberID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "",
		Role:     "editor",
	})
	if err == nil {
		t.Error("expected error for empty member_id")
	}
}

func TestUpdateBoardMember_EmptyRole(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
		Role:     "",
	})
	if err == nil {
		t.Error("expected error for empty role")
	}
}

func TestUpdateBoardMember_InvalidRole(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
		Role:     "admin",
	})
	if err == nil {
		t.Error("expected error for invalid role")
	}
	if !strings.Contains(err.Error(), "invalid role") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateBoardMember_JSONParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
		Role:     "editor",
	})
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestUpdateBoardMember_EmptyNameFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "member456",
			"name":  "",
			"email": "user@example.com",
			"role":  "editor",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
		Role:     "editor",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Message, "user@example.com") {
		t.Errorf("expected email in message when name is empty, got: %s", result.Message)
	}
}

// Tests for DetachTag error paths
func TestDetachTag_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "",
		ItemID:  "item123",
		TagID:   "tag456",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

func TestDetachTag_EmptyItemID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "board123",
		ItemID:  "",
		TagID:   "tag456",
	})
	if err == nil {
		t.Error("expected error for empty item_id")
	}
}

func TestDetachTag_EmptyTagID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "board123",
		ItemID:  "item123",
		TagID:   "",
	})
	if err == nil {
		t.Error("expected error for empty tag_id")
	}
}

func TestDetachTag_APIError_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Tag not found"}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "board123",
		ItemID:  "item123",
		TagID:   "tag456",
	})
	if err == nil {
		t.Error("expected error for 404 response")
	}
	if result.Success {
		t.Error("expected Success=false on error")
	}
}

// Tests for GetItemTags error paths
func TestGetItemTags_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "",
		ItemID:  "item123",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

func TestGetItemTags_EmptyItemID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "board123",
		ItemID:  "",
	})
	if err == nil {
		t.Error("expected error for empty item_id")
	}
}

func TestGetItemTags_JSONParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "board123",
		ItemID:  "item123",
	})
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

// Tests for ListTags error paths
func TestListTags_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

// Tests for ListMindmapNodes error paths
func TestListMindmapNodes_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

func TestListMindmapNodes_WithCursorPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "cursor=abc123") {
			t.Errorf("expected cursor in query string, got: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
		Cursor:  "abc123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListMindmapNodes_LimitClampTo100(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "limit=100") {
			t.Errorf("expected limit=100, got: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
		Limit:   500, // Should be clamped to 100
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListMindmapNodes_JSONParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
	})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestListMindmapNodes_WithParentNode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{
					"id": "node1",
					"data": map[string]interface{}{
						"isRoot": true,
						"nodeView": map[string]interface{}{
							"data": map[string]interface{}{
								"content": "Root Node",
							},
						},
					},
				},
				map[string]interface{}{
					"id": "node2",
					"data": map[string]interface{}{
						"isRoot": false,
						"nodeView": map[string]interface{}{
							"data": map[string]interface{}{
								"content": "Child Node",
							},
						},
					},
					"parent": map[string]interface{}{
						"id": "node1",
					},
				},
			},
			"cursor": "next_cursor",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("expected 2 nodes, got: %d", result.Count)
	}
	if !result.HasMore {
		t.Error("expected HasMore=true when cursor is present")
	}
	if result.Nodes[1].ParentID != "node1" {
		t.Errorf("expected parent_id=node1, got: %s", result.Nodes[1].ParentID)
	}
}

// Tests for CreateMindmapNode error paths
func TestCreateMindmapNode_EmptyContent(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.CreateMindmapNode(context.Background(), CreateMindmapNodeArgs{
		BoardID: "board123",
		Content: "",
	})
	if err == nil {
		t.Error("expected error for empty content")
	}
}

// Tests for DeleteMindmapNode error paths
func TestDeleteMindmapNode_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.DeleteMindmapNode(context.Background(), DeleteMindmapNodeArgs{
		BoardID: "",
		NodeID:  "node123",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

func TestDeleteMindmapNode_EmptyNodeID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.DeleteMindmapNode(context.Background(), DeleteMindmapNodeArgs{
		BoardID: "board123",
		NodeID:  "",
	})
	if err == nil {
		t.Error("expected error for empty node_id")
	}
}

// Tests for ListTags
func TestListTags_WithLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "limit=25") {
			t.Errorf("expected limit=25, got: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "board123",
		Limit:   25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No tags on this board" {
		t.Errorf("expected no tags message, got: %s", result.Message)
	}
}

func TestListTags_JSONParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "board123",
	})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// Tests for GetMindmapNode error paths
func TestGetMindmapNode_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.GetMindmapNode(context.Background(), GetMindmapNodeArgs{
		BoardID: "",
		NodeID:  "node123",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

func TestGetMindmapNode_EmptyNodeID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.GetMindmapNode(context.Background(), GetMindmapNodeArgs{
		BoardID: "board123",
		NodeID:  "",
	})
	if err == nil {
		t.Error("expected error for empty node_id")
	}
}

// =============================================================================
// UpdateImage Tests
// =============================================================================

func TestUpdateImage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/images/") {
			t.Errorf("expected /images/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "image123",
			"data": map[string]interface{}{
				"title":    "Updated image",
				"imageUrl": "https://example.com/new.png",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateImage(context.Background(), UpdateImageArgs{
		BoardID: "board123",
		ItemID:  "image123",
		Title:   strPtr("Updated image"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "image123" {
		t.Errorf("ID = %q, want 'image123'", result.ID)
	}
	if result.Title != "Updated image" {
		t.Errorf("Title = %q, want 'Updated image'", result.Title)
	}
}

func TestUpdateImage_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateImage(context.Background(), UpdateImageArgs{
		BoardID: "board123",
		ItemID:  "image123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateImage_Validation(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Empty board_id
	_, err := client.UpdateImage(context.Background(), UpdateImageArgs{
		BoardID: "",
		ItemID:  "image123",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}

	// Empty item_id
	_, err = client.UpdateImage(context.Background(), UpdateImageArgs{
		BoardID: "board123",
		ItemID:  "",
	})
	if err == nil {
		t.Error("expected error for empty item_id")
	}
}

// =============================================================================
// UpdateDocument Tests
// =============================================================================

func TestUpdateDocument_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/documents/") {
			t.Errorf("expected /documents/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "doc123",
			"data": map[string]interface{}{
				"title": "Updated document",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateDocument(context.Background(), UpdateDocumentArgs{
		BoardID: "board123",
		ItemID:  "doc123",
		Title:   strPtr("Updated document"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "doc123" {
		t.Errorf("ID = %q, want 'doc123'", result.ID)
	}
	if result.Title != "Updated document" {
		t.Errorf("Title = %q, want 'Updated document'", result.Title)
	}
}

func TestUpdateDocument_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateDocument(context.Background(), UpdateDocumentArgs{
		BoardID: "board123",
		ItemID:  "doc123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateDocument_Validation(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Empty board_id
	_, err := client.UpdateDocument(context.Background(), UpdateDocumentArgs{
		BoardID: "",
		ItemID:  "doc123",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}

	// Empty item_id
	_, err = client.UpdateDocument(context.Background(), UpdateDocumentArgs{
		BoardID: "board123",
		ItemID:  "",
	})
	if err == nil {
		t.Error("expected error for empty item_id")
	}
}

// =============================================================================
// UpdateEmbed Tests
// =============================================================================

func TestUpdateEmbed_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/embeds/") {
			t.Errorf("expected /embeds/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "embed123",
			"data": map[string]interface{}{
				"url":         "https://youtube.com/watch?v=123",
				"providerUrl": "youtube.com",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateEmbed(context.Background(), UpdateEmbedArgs{
		BoardID: "board123",
		ItemID:  "embed123",
		URL:     strPtr("https://youtube.com/watch?v=123"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "embed123" {
		t.Errorf("ID = %q, want 'embed123'", result.ID)
	}
	if result.URL != "https://youtube.com/watch?v=123" {
		t.Errorf("URL = %q, want 'https://youtube.com/watch?v=123'", result.URL)
	}
}

func TestUpdateEmbed_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateEmbed(context.Background(), UpdateEmbedArgs{
		BoardID: "board123",
		ItemID:  "embed123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateEmbed_Validation(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Empty board_id
	_, err := client.UpdateEmbed(context.Background(), UpdateEmbedArgs{
		BoardID: "",
		ItemID:  "embed123",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}

	// Empty item_id
	_, err = client.UpdateEmbed(context.Background(), UpdateEmbedArgs{
		BoardID: "board123",
		ItemID:  "",
	})
	if err == nil {
		t.Error("expected error for empty item_id")
	}
}

func TestUpdateEmbed_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		if data, ok := body["data"].(map[string]interface{}); !ok {
			t.Error("expected data in request body")
		} else {
			if data["url"] != "https://youtube.com/watch?v=456" {
				t.Errorf("url = %v, want 'https://youtube.com/watch?v=456'", data["url"])
			}
			if data["mode"] != "modal" {
				t.Errorf("mode = %v, want 'modal'", data["mode"])
			}
		}

		// Verify geometry section
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else {
			if geom["width"] != float64(800) {
				t.Errorf("width = %v, want 800", geom["width"])
			}
			if geom["height"] != float64(600) {
				t.Errorf("height = %v, want 600", geom["height"])
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "embed123",
			"data": map[string]interface{}{
				"url":         "https://youtube.com/watch?v=456",
				"providerUrl": "youtube.com",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	width := float64(800)
	height := float64(600)
	result, err := client.UpdateEmbed(context.Background(), UpdateEmbedArgs{
		BoardID: "board123",
		ItemID:  "embed123",
		URL:     strPtr("https://youtube.com/watch?v=456"),
		Mode:    strPtr("modal"),
		Width:   &width,
		Height:  &height,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "embed123" {
		t.Errorf("ID = %q, want 'embed123'", result.ID)
	}
}

