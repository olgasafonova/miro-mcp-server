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

// ptrString returns a pointer to the given string.
func ptrString(s string) *string { return &s }

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
		name       string
		content    string
		query      string
		contextLen int
		expect     string
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

// newTestClientWithServer creates a client pointing to a mock HTTP server.
func newTestClientWithServer(serverURL string) *Client {
	cfg := testConfig()
	client := NewClient(cfg, testLogger())
	client.baseURL = serverURL
	return client
}

func TestRateLimitRetry(t *testing.T) {
	t.Run("succeeds_after_retry", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.Header().Set("Retry-After", "0") // Use 0 for faster tests
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

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// With retry logic, this should succeed after 2 rate limits + 1 success
		_, err := client.request(ctx, http.MethodGet, "/boards", nil)
		if err != nil {
			t.Fatalf("expected success after retry, got error: %v", err)
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("fails_after_max_retries", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			// Always return rate limit error
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Rate limit exceeded",
			})
		}))
		defer server.Close()

		client := newTestClientWithServer(server.URL)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Should fail after 1 initial + MaxRetries (3) retry attempts = 4 total
		_, err := client.request(ctx, http.MethodGet, "/boards", nil)
		if err == nil {
			t.Fatal("expected error after max retries")
		}
		if !IsRateLimitError(err) {
			t.Errorf("expected rate limit error, got: %v", err)
		}
		// 1 initial attempt + MaxRetries retries
		expectedAttempts := 1 + MaxRetries
		if attempts != expectedAttempts {
			t.Errorf("expected %d attempts (1 initial + %d retries), got %d", expectedAttempts, MaxRetries, attempts)
		}
	})

	t.Run("retries_transient_errors", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts == 1 {
				// First attempt: 503 Service Unavailable
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message": "Service temporarily unavailable",
				})
				return
			}
			if attempts == 2 {
				// Second attempt: 502 Bad Gateway
				w.WriteHeader(http.StatusBadGateway)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message": "Bad gateway",
				})
				return
			}
			// Third attempt: success
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{},
			})
		}))
		defer server.Close()

		client := newTestClientWithServer(server.URL)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.request(ctx, http.MethodGet, "/boards", nil)
		if err != nil {
			t.Fatalf("expected success after retrying transient errors, got: %v", err)
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})
}

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

func strPtr(s string) *string {
	return &s
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
		Remaining: 5,           // Below buffer of 10
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
