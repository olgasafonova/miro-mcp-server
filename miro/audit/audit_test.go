package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryLogger_Log(t *testing.T) {
	config := Config{Enabled: true, SanitizeInput: true}
	logger := NewMemoryLogger(10, config)

	event := Event{
		ID:        "test-1",
		Timestamp: time.Now().UTC(),
		Tool:      "miro_create_sticky",
		Method:    "CreateSticky",
		Action:    ActionCreate,
		BoardID:   "board-123",
		Success:   true,
	}

	err := logger.Log(context.Background(), event)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	events := logger.GetAllEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Tool != "miro_create_sticky" {
		t.Errorf("expected tool miro_create_sticky, got %s", events[0].Tool)
	}
}

func TestMemoryLogger_RingBuffer(t *testing.T) {
	config := Config{Enabled: true}
	logger := NewMemoryLogger(3, config) // Small buffer

	// Log 5 events
	for i := 0; i < 5; i++ {
		event := Event{
			ID:        "test-" + string(rune('0'+i)),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Tool:      "tool-" + string(rune('0'+i)),
		}
		logger.Log(context.Background(), event)
	}

	// Should only have last 3
	events := logger.GetAllEvents()
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestMemoryLogger_Query(t *testing.T) {
	config := Config{Enabled: true}
	logger := NewMemoryLogger(100, config)

	// Log events
	events := []Event{
		{ID: "1", Tool: "miro_create_sticky", Action: ActionCreate, BoardID: "board-1", Success: true, Timestamp: time.Now()},
		{ID: "2", Tool: "miro_list_boards", Action: ActionRead, Success: true, Timestamp: time.Now()},
		{ID: "3", Tool: "miro_delete_item", Action: ActionDelete, BoardID: "board-1", Success: false, Timestamp: time.Now()},
	}
	for _, e := range events {
		logger.Log(context.Background(), e)
	}

	// Query by tool
	result, err := logger.Query(context.Background(), QueryOptions{Tool: "miro_create_sticky"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(result.Events))
	}

	// Query by board
	result, err = logger.Query(context.Background(), QueryOptions{BoardID: "board-1"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(result.Events))
	}

	// Query by action
	result, err = logger.Query(context.Background(), QueryOptions{Action: ActionDelete})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(result.Events))
	}

	// Query by success status
	success := true
	result, err = logger.Query(context.Background(), QueryOptions{Success: &success})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(result.Events))
	}
}

func TestMemoryLogger_QueryPagination(t *testing.T) {
	config := Config{Enabled: true}
	logger := NewMemoryLogger(100, config)

	// Log 10 events
	for i := 0; i < 10; i++ {
		event := Event{
			ID:        "test-" + string(rune('0'+i)),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Tool:      "miro_test",
		}
		logger.Log(context.Background(), event)
	}

	// Query with limit
	result, _ := logger.Query(context.Background(), QueryOptions{Limit: 3})
	if len(result.Events) != 3 {
		t.Errorf("expected 3 events, got %d", len(result.Events))
	}
	if !result.HasMore {
		t.Error("expected HasMore to be true")
	}

	// Query with offset
	result, _ = logger.Query(context.Background(), QueryOptions{Offset: 5, Limit: 10})
	if len(result.Events) != 5 {
		t.Errorf("expected 5 events, got %d", len(result.Events))
	}
}

func TestMemoryLogger_Stats(t *testing.T) {
	config := Config{Enabled: true}
	logger := NewMemoryLogger(100, config)

	// Log events
	events := []Event{
		{Tool: "miro_create_sticky", Action: ActionCreate, Success: true, DurationMs: 100, Timestamp: time.Now()},
		{Tool: "miro_create_sticky", Action: ActionCreate, Success: true, DurationMs: 200, Timestamp: time.Now()},
		{Tool: "miro_list_boards", Action: ActionRead, Success: false, DurationMs: 50, Timestamp: time.Now()},
	}
	for _, e := range events {
		logger.Log(context.Background(), e)
	}

	stats := logger.GetStats()
	if stats.TotalEvents != 3 {
		t.Errorf("expected 3 total events, got %d", stats.TotalEvents)
	}
	if stats.SuccessCount != 2 {
		t.Errorf("expected 2 success, got %d", stats.SuccessCount)
	}
	if stats.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", stats.ErrorCount)
	}
	if stats.ByTool["miro_create_sticky"] != 2 {
		t.Errorf("expected 2 create_sticky, got %d", stats.ByTool["miro_create_sticky"])
	}
}

func TestMemoryLogger_Disabled(t *testing.T) {
	config := Config{Enabled: false}
	logger := NewMemoryLogger(10, config)

	event := Event{ID: "1", Tool: "test"}
	logger.Log(context.Background(), event)

	events := logger.GetAllEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events when disabled, got %d", len(events))
	}
}

func TestSanitizeInput(t *testing.T) {
	input := map[string]interface{}{
		"board_id":     "abc123",
		"access_token": "secret123",
		"content":      "hello",
		"password":     "hunter2",
	}

	sanitized := SanitizeInput(input)

	if sanitized["board_id"] != "abc123" {
		t.Error("board_id should not be redacted")
	}
	if sanitized["content"] != "hello" {
		t.Error("content should not be redacted")
	}
	if sanitized["access_token"] != "[REDACTED]" {
		t.Error("access_token should be redacted")
	}
	if sanitized["password"] != "[REDACTED]" {
		t.Error("password should be redacted")
	}
}

func TestEventBuilder(t *testing.T) {
	event := NewEvent("miro_create_sticky", "CreateSticky", ActionCreate).
		WithUser("user-1", "test@example.com").
		WithBoard("board-123").
		WithItem("item-456", "sticky_note").
		WithDuration(150 * time.Millisecond).
		Success().
		Build()

	if event.Tool != "miro_create_sticky" {
		t.Error("tool not set")
	}
	if event.UserID != "user-1" {
		t.Error("user_id not set")
	}
	if event.BoardID != "board-123" {
		t.Error("board_id not set")
	}
	if event.ItemID != "item-456" {
		t.Error("item_id not set")
	}
	if event.DurationMs != 150 {
		t.Error("duration not set")
	}
	if !event.Success {
		t.Error("success not set")
	}
	if event.ID == "" {
		t.Error("ID not generated")
	}
}

func TestDetectAction(t *testing.T) {
	tests := []struct {
		method string
		want   Action
	}{
		{"CreateSticky", ActionCreate},
		{"BulkCreate", ActionCreate},
		{"ListBoards", ActionRead},
		{"GetBoard", ActionRead},
		{"SearchBoard", ActionRead},
		{"FindBoardByName", ActionRead},
		{"UpdateItem", ActionUpdate},
		{"DeleteItem", ActionDelete},
		{"Ungroup", ActionDelete},
		{"DetachTag", ActionDelete},
		{"ExportBoard", ActionExport},
		{"ValidateToken", ActionAuth},
		{"ShareBoard", ActionAuth},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got := DetectAction(tt.method)
			if got != tt.want {
				t.Errorf("DetectAction(%s) = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}

func TestNoopLogger(t *testing.T) {
	logger := NewNoopLogger()

	// All operations should succeed silently
	err := logger.Log(context.Background(), Event{})
	if err != nil {
		t.Errorf("Log should not error: %v", err)
	}

	result, err := logger.Query(context.Background(), QueryOptions{})
	if err != nil {
		t.Errorf("Query should not error: %v", err)
	}
	if len(result.Events) != 0 {
		t.Error("Query should return empty events")
	}

	if logger.Flush(context.Background()) != nil {
		t.Error("Flush should not error")
	}
	if logger.Close() != nil {
		t.Error("Close should not error")
	}
}

func TestFileLogger(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "audit-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:       true,
		Path:          tmpDir,
		RetentionDays: 30,
		MaxSizeBytes:  10 * 1024 * 1024,
		BufferSize:    0, // No buffering for test
		SanitizeInput: true,
	}

	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create file logger: %v", err)
	}
	defer logger.Close()

	// Log an event
	event := Event{
		ID:        "file-test-1",
		Timestamp: time.Now().UTC(),
		Tool:      "miro_create_sticky",
		Method:    "CreateSticky",
		Action:    ActionCreate,
		BoardID:   "board-123",
		Success:   true,
	}

	err = logger.Log(context.Background(), event)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Force flush
	logger.Flush(context.Background())

	// Verify file was created
	files, _ := os.ReadDir(tmpDir)
	if len(files) == 0 {
		t.Fatal("no audit log file created")
	}

	// Query the event
	result, err := logger.Query(context.Background(), QueryOptions{})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Tool != "miro_create_sticky" {
		t.Errorf("expected tool miro_create_sticky, got %s", result.Events[0].Tool)
	}
}

func TestFileLogger_WithBuffer(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-buffer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:      true,
		Path:         tmpDir,
		MaxSizeBytes: 10 * 1024 * 1024,
		BufferSize:   3, // Buffer 3 events before flushing
	}

	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create file logger: %v", err)
	}
	defer logger.Close()

	// Log 2 events (less than buffer size)
	for i := 0; i < 2; i++ {
		logger.Log(context.Background(), Event{ID: "buf-" + string(rune('0'+i)), Tool: "test"})
	}

	// File should be empty or very small (buffered)
	logger.Flush(context.Background())

	// Log more events to trigger buffer flush
	for i := 2; i < 5; i++ {
		logger.Log(context.Background(), Event{ID: "buf-" + string(rune('0'+i)), Tool: "test"})
	}
	logger.Flush(context.Background())

	// Query should find all events
	result, _ := logger.Query(context.Background(), QueryOptions{})
	if len(result.Events) != 5 {
		t.Errorf("expected 5 events, got %d", len(result.Events))
	}
}

func TestNewLogger_Memory(t *testing.T) {
	config := Config{Enabled: true, Path: ""} // Empty path = memory logger
	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	// Should be memory logger
	if _, ok := logger.(*MemoryLogger); !ok {
		t.Error("expected MemoryLogger for empty path")
	}
}

func TestNewLogger_Disabled(t *testing.T) {
	config := Config{Enabled: false}
	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	// Should be noop logger
	if _, ok := logger.(*NoopLogger); !ok {
		t.Error("expected NoopLogger when disabled")
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Set test env vars
	os.Setenv("MIRO_AUDIT_ENABLED", "true")
	os.Setenv("MIRO_AUDIT_PATH", "/tmp/audit")
	os.Setenv("MIRO_AUDIT_RETENTION", "7d")
	os.Setenv("MIRO_AUDIT_MAX_SIZE", "50M")
	os.Setenv("MIRO_AUDIT_BUFFER_SIZE", "50")
	defer func() {
		os.Unsetenv("MIRO_AUDIT_ENABLED")
		os.Unsetenv("MIRO_AUDIT_PATH")
		os.Unsetenv("MIRO_AUDIT_RETENTION")
		os.Unsetenv("MIRO_AUDIT_MAX_SIZE")
		os.Unsetenv("MIRO_AUDIT_BUFFER_SIZE")
	}()

	config := LoadConfigFromEnv()

	if !config.Enabled {
		t.Error("expected Enabled=true")
	}
	if config.Path != "/tmp/audit" {
		t.Errorf("expected Path=/tmp/audit, got %s", config.Path)
	}
	if config.RetentionDays != 7 {
		t.Errorf("expected RetentionDays=7, got %d", config.RetentionDays)
	}
	if config.MaxSizeBytes != 50*1024*1024 {
		t.Errorf("expected MaxSizeBytes=50MB, got %d", config.MaxSizeBytes)
	}
	if config.BufferSize != 50 {
		t.Errorf("expected BufferSize=50, got %d", config.BufferSize)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{50, "50ms"},
		{999, "999ms"},
		{1000, "1.0s"},
		{2500, "2.5s"},
		{60000, "1.0m"},
		{90000, "1.5m"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatDuration(tt.ms)
			if got != tt.want {
				t.Errorf("FormatDuration(%d) = %s, want %s", tt.ms, got, tt.want)
			}
		})
	}
}

func TestFileLogger_Cleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-cleanup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an old file
	oldFile := filepath.Join(tmpDir, "audit-2020-01-01T00-00-00.jsonl")
	os.WriteFile(oldFile, []byte("{}"), 0600)
	// Set modification time to 60 days ago
	oldTime := time.Now().AddDate(0, 0, -60)
	os.Chtimes(oldFile, oldTime, oldTime)

	config := Config{
		Enabled:       true,
		Path:          tmpDir,
		RetentionDays: 30, // 30 day retention
		MaxSizeBytes:  10 * 1024 * 1024,
		BufferSize:    0,
	}

	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create file logger: %v", err)
	}

	// Log an event to trigger cleanup
	logger.Log(context.Background(), Event{ID: "cleanup-test", Tool: "test"})
	logger.Close()

	// Give cleanup goroutine time to run
	time.Sleep(100 * time.Millisecond)

	// Old file should be deleted
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("old file should have been deleted by cleanup")
	}
}
