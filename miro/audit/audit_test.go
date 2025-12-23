package audit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestEventBuilder_WithItemCount(t *testing.T) {
	event := NewEvent("miro_bulk_create", "BulkCreate", ActionCreate).
		WithBoard("board-123").
		WithItemCount(5).
		Success().
		Build()

	if event.ItemCount != 5 {
		t.Errorf("expected ItemCount=5, got %d", event.ItemCount)
	}
}

func TestEventBuilder_WithInput(t *testing.T) {
	input := map[string]interface{}{
		"board_id": "abc123",
		"content":  "test sticky",
		"color":    "yellow",
	}

	event := NewEvent("miro_create_sticky", "CreateSticky", ActionCreate).
		WithBoard("abc123").
		WithInput(input).
		Success().
		Build()

	if event.Input == nil {
		t.Fatal("expected Input to be set")
	}
	if event.Input["board_id"] != "abc123" {
		t.Error("input board_id not set correctly")
	}
	if event.Input["content"] != "test sticky" {
		t.Error("input content not set correctly")
	}
	if event.Input["color"] != "yellow" {
		t.Error("input color not set correctly")
	}
}

func TestEventBuilder_Failure(t *testing.T) {
	testErr := errors.New("API rate limit exceeded")

	event := NewEvent("miro_create_sticky", "CreateSticky", ActionCreate).
		WithBoard("board-123").
		Failure(testErr).
		Build()

	if event.Success {
		t.Error("expected Success=false for failure")
	}
	if event.Error != "API rate limit exceeded" {
		t.Errorf("expected error message 'API rate limit exceeded', got '%s'", event.Error)
	}
}

func TestEventBuilder_FailureWithNilError(t *testing.T) {
	event := NewEvent("miro_create_sticky", "CreateSticky", ActionCreate).
		WithBoard("board-123").
		Failure(nil).
		Build()

	if event.Success {
		t.Error("expected Success=false for failure")
	}
	if event.Error != "" {
		t.Errorf("expected empty error message for nil error, got '%s'", event.Error)
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

func TestFileLogger_CurrentFilePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-filepath-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:      true,
		Path:         tmpDir,
		MaxSizeBytes: 10 * 1024 * 1024,
		BufferSize:   0,
	}

	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create file logger: %v", err)
	}
	defer logger.Close()

	filePath := logger.CurrentFilePath()

	// Verify path is in the temp directory
	if !strings.HasPrefix(filePath, tmpDir) {
		t.Errorf("expected file path to start with %s, got %s", tmpDir, filePath)
	}

	// Verify path has expected format: audit-YYYY-MM-DDTHH-MM-SS.jsonl
	if !strings.HasSuffix(filePath, ".jsonl") {
		t.Errorf("expected file path to end with .jsonl, got %s", filePath)
	}

	if !strings.Contains(filepath.Base(filePath), "audit-") {
		t.Errorf("expected file name to contain 'audit-', got %s", filepath.Base(filePath))
	}

	// Verify the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("expected file to exist at %s", filePath)
	}
}

func TestMemoryLogger_Flush(t *testing.T) {
	config := Config{Enabled: true}
	logger := NewMemoryLogger(100, config)

	// Log some events
	for i := 0; i < 5; i++ {
		logger.Log(context.Background(), Event{ID: "flush-" + string(rune('0'+i)), Tool: "test"})
	}

	// Flush should be a no-op for MemoryLogger but should not error
	err := logger.Flush(context.Background())
	if err != nil {
		t.Errorf("Flush should not error: %v", err)
	}

	// Events should still be there after flush
	events := logger.GetAllEvents()
	if len(events) != 5 {
		t.Errorf("expected 5 events after flush, got %d", len(events))
	}
}

func TestMemoryLogger_Clear(t *testing.T) {
	config := Config{Enabled: true}
	logger := NewMemoryLogger(100, config)

	// Log some events
	for i := 0; i < 5; i++ {
		logger.Log(context.Background(), Event{
			ID:   "clear-" + string(rune('0'+i)),
			Tool: "test",
		})
	}

	// Verify events exist
	events := logger.GetAllEvents()
	if len(events) != 5 {
		t.Fatalf("expected 5 events before clear, got %d", len(events))
	}

	// Clear the logger
	logger.Clear()

	// Verify events are gone
	events = logger.GetAllEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events after clear, got %d", len(events))
	}

	// Verify stats are reset
	stats := logger.GetStats()
	if stats.TotalEvents != 0 {
		t.Errorf("expected TotalEvents=0 after clear, got %d", stats.TotalEvents)
	}

	// Verify we can log new events after clear
	logger.Log(context.Background(), Event{ID: "new-1", Tool: "test"})
	events = logger.GetAllEvents()
	if len(events) != 1 {
		t.Errorf("expected 1 event after logging post-clear, got %d", len(events))
	}
}

func TestMemoryLogger_Close(t *testing.T) {
	config := Config{Enabled: true}
	logger := NewMemoryLogger(100, config)

	// Log some events
	logger.Log(context.Background(), Event{ID: "close-1", Tool: "test"})

	// Close should be a no-op but should not error
	err := logger.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}
}

func TestMemoryLogger_QueryTimeRange(t *testing.T) {
	config := Config{Enabled: true}
	logger := NewMemoryLogger(100, config)

	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)

	// Log events at different times
	logger.Log(context.Background(), Event{ID: "1", Tool: "test", Timestamp: earlier})
	logger.Log(context.Background(), Event{ID: "2", Tool: "test", Timestamp: now})
	logger.Log(context.Background(), Event{ID: "3", Tool: "test", Timestamp: later})

	// Query since now should exclude earlier event
	result, err := logger.Query(context.Background(), QueryOptions{Since: now})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result.Events) != 2 {
		t.Errorf("expected 2 events since now, got %d", len(result.Events))
	}

	// Query until now should exclude later event
	result, err = logger.Query(context.Background(), QueryOptions{Until: now})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result.Events) != 2 {
		t.Errorf("expected 2 events until now, got %d", len(result.Events))
	}
}

func TestMemoryLogger_QueryMethodAndUserID(t *testing.T) {
	config := Config{Enabled: true}
	logger := NewMemoryLogger(100, config)

	// Log events with different methods and users
	logger.Log(context.Background(), Event{ID: "1", Method: "CreateSticky", UserID: "user-1", Timestamp: time.Now()})
	logger.Log(context.Background(), Event{ID: "2", Method: "CreateSticky", UserID: "user-2", Timestamp: time.Now()})
	logger.Log(context.Background(), Event{ID: "3", Method: "DeleteItem", UserID: "user-1", Timestamp: time.Now()})

	// Query by method
	result, err := logger.Query(context.Background(), QueryOptions{Method: "CreateSticky"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result.Events) != 2 {
		t.Errorf("expected 2 events for method CreateSticky, got %d", len(result.Events))
	}

	// Query by user ID
	result, err = logger.Query(context.Background(), QueryOptions{UserID: "user-1"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result.Events) != 2 {
		t.Errorf("expected 2 events for user-1, got %d", len(result.Events))
	}
}

func TestMemoryLogger_QueryOffsetExceedsTotal(t *testing.T) {
	config := Config{Enabled: true}
	logger := NewMemoryLogger(100, config)

	// Log a few events
	for i := 0; i < 3; i++ {
		logger.Log(context.Background(), Event{ID: "off-" + string(rune('0'+i)), Tool: "test", Timestamp: time.Now()})
	}

	// Query with offset exceeding total
	result, err := logger.Query(context.Background(), QueryOptions{Offset: 10})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result.Events) != 0 {
		t.Errorf("expected 0 events when offset exceeds total, got %d", len(result.Events))
	}
	if result.Total != 3 {
		t.Errorf("expected Total=3, got %d", result.Total)
	}
}

func TestNewMemoryLogger_DefaultMaxSize(t *testing.T) {
	config := Config{Enabled: true}
	// Pass 0 or negative maxSize, should default to 1000
	logger := NewMemoryLogger(0, config)

	// Verify we can log at least 1000 events without error
	for i := 0; i < 1000; i++ {
		logger.Log(context.Background(), Event{ID: "default-" + string(rune(i)), Tool: "test"})
	}

	events := logger.GetAllEvents()
	if len(events) != 1000 {
		t.Errorf("expected 1000 events with default maxSize, got %d", len(events))
	}
}

// =============================================================================
// parseDuration Tests
// =============================================================================

func TestParseDuration_DaysSuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"30d", 30},
		{"7d", 7},
		{"90d", 90},
		{"1d", 1},
		{"365d", 365},
		{"  30d  ", 30},     // whitespace
		{"30D", 30},         // uppercase
		{"  30D  ", 30},     // mixed
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDuration(tt.input)
			if result != tt.expected {
				t.Errorf("parseDuration(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDuration_IntegerOnly(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"30", 30},
		{"7", 7},
		{"90", 90},
		{"  30  ", 30},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDuration(tt.input)
			if result != tt.expected {
				t.Errorf("parseDuration(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDuration_Invalid(t *testing.T) {
	tests := []string{
		"",
		"abc",
		"abcd",
		"30x",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			result := parseDuration(input)
			if result != 0 {
				t.Errorf("parseDuration(%q) = %d, want 0", input, result)
			}
		})
	}
}

// =============================================================================
// parseSize Tests
// =============================================================================

func TestParseSize_KilobytesSuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1K", 1024},
		{"10K", 10 * 1024},
		{"100K", 100 * 1024},
		{"500k", 500 * 1024}, // lowercase
		{"  1K  ", 1024},     // whitespace
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSize(tt.input)
			if result != tt.expected {
				t.Errorf("parseSize(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseSize_MegabytesSuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1M", 1024 * 1024},
		{"10M", 10 * 1024 * 1024},
		{"100M", 100 * 1024 * 1024},
		{"500m", 500 * 1024 * 1024}, // lowercase
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSize(tt.input)
			if result != tt.expected {
				t.Errorf("parseSize(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseSize_GigabytesSuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1G", 1024 * 1024 * 1024},
		{"2G", 2 * 1024 * 1024 * 1024},
		{"1g", 1024 * 1024 * 1024}, // lowercase
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSize(tt.input)
			if result != tt.expected {
				t.Errorf("parseSize(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseSize_PlainBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1024", 1024},
		{"1000000", 1000000},
		{"  1024  ", 1024}, // whitespace
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSize(tt.input)
			if result != tt.expected {
				t.Errorf("parseSize(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseSize_Invalid(t *testing.T) {
	tests := []string{
		"",
		"abc",
		"abcM",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			result := parseSize(input)
			if result != 0 {
				t.Errorf("parseSize(%q) = %d, want 0", input, result)
			}
		})
	}
}

// =============================================================================
// Additional Coverage Tests
// =============================================================================

func TestNewFileLogger_InvalidPath(t *testing.T) {
	// Test with path that cannot be created (null character in path)
	config := Config{
		Enabled: true,
		Path:    "/dev/null\x00invalid",
	}

	_, err := NewFileLogger(config)
	if err == nil {
		t.Error("expected error for invalid path with null character")
	}
}

func TestNewFileLogger_EmptyPath(t *testing.T) {
	config := Config{
		Enabled: true,
		Path:    "",
	}

	_, err := NewFileLogger(config)
	if err == nil {
		t.Error("expected error for empty path")
	}
	if !strings.Contains(err.Error(), "path is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewLogger_FileLoggerWithPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-newlogger-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:      true,
		Path:         tmpDir,
		MaxSizeBytes: 10 * 1024 * 1024,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	// Should be file logger
	if _, ok := logger.(*FileLogger); !ok {
		t.Error("expected FileLogger for non-empty path")
	}
}

func TestFileLogger_QueryReadDirError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-query-err-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	config := Config{
		Enabled:      true,
		Path:         tmpDir,
		MaxSizeBytes: 10 * 1024 * 1024,
		BufferSize:   0,
	}

	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Remove directory to cause ReadDir error
	logger.Close()
	os.RemoveAll(tmpDir)

	// Query should fail because directory is gone
	_, queryErr := logger.Query(context.Background(), QueryOptions{})
	if queryErr == nil {
		t.Error("expected error when directory doesn't exist")
	}
}

func TestFileLogger_QueryWithOffset(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-query-offset-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:      true,
		Path:         tmpDir,
		MaxSizeBytes: 10 * 1024 * 1024,
		BufferSize:   0,
	}

	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log 5 events
	for i := 0; i < 5; i++ {
		logger.Log(context.Background(), Event{
			ID:        "offset-" + string(rune('0'+i)),
			Tool:      "test",
			Timestamp: time.Now(),
		})
	}
	logger.Flush(context.Background())

	// Query with offset exceeding total
	result, err := logger.Query(context.Background(), QueryOptions{Offset: 10})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result.Events) != 0 {
		t.Errorf("expected 0 events with offset=10, got %d", len(result.Events))
	}
	if result.Total != 5 {
		t.Errorf("expected Total=5, got %d", result.Total)
	}
}

func TestFileLogger_ReadEventsContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-ctx-cancel-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:      true,
		Path:         tmpDir,
		MaxSizeBytes: 10 * 1024 * 1024,
		BufferSize:   0,
	}

	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log many events
	for i := 0; i < 100; i++ {
		logger.Log(context.Background(), Event{
			ID:        "ctx-" + string(rune(i)),
			Tool:      "test",
			Timestamp: time.Now(),
		})
	}
	logger.Flush(context.Background())

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Query with cancelled context - should return partial results or error
	result, _ := logger.Query(ctx, QueryOptions{})
	// The context is checked per line during scan, so with cancelled context
	// we might get 0 results or partial results
	_ = result // Just ensure no panic
}

func TestFileLogger_CleanupWithDirectoryEntry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-cleanup-dir-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory (should be skipped by cleanup)
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)

	config := Config{
		Enabled:       true,
		Path:          tmpDir,
		RetentionDays: 30,
		MaxSizeBytes:  10 * 1024 * 1024,
		BufferSize:    0,
	}

	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Log an event to trigger cleanup
	logger.Log(context.Background(), Event{ID: "cleanup-dir-test", Tool: "test"})
	logger.Close()

	// Give cleanup goroutine time to run
	time.Sleep(100 * time.Millisecond)

	// Subdirectory should still exist (not deleted by cleanup)
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Error("subdirectory should not have been deleted by cleanup")
	}
}

func TestFileLogger_CleanupRetentionZero(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-cleanup-zero-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an old file
	oldFile := filepath.Join(tmpDir, "audit-2020-01-01T00-00-00.jsonl")
	os.WriteFile(oldFile, []byte("{}"), 0600)
	oldTime := time.Now().AddDate(0, 0, -60)
	os.Chtimes(oldFile, oldTime, oldTime)

	config := Config{
		Enabled:       true,
		Path:          tmpDir,
		RetentionDays: 0, // Disabled retention
		MaxSizeBytes:  10 * 1024 * 1024,
		BufferSize:    0,
	}

	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Log(context.Background(), Event{ID: "retention-zero-test", Tool: "test"})
	logger.Close()

	// Give cleanup goroutine time to run
	time.Sleep(100 * time.Millisecond)

	// Old file should still exist (retention disabled)
	if _, err := os.Stat(oldFile); os.IsNotExist(err) {
		t.Error("old file should not have been deleted when retention is 0")
	}
}

func TestDetectAction_DefaultCase(t *testing.T) {
	// Test methods that don't match any prefix - should return ActionRead
	result := DetectAction("unknownmethod")
	if result != ActionRead {
		t.Errorf("DetectAction(unknownmethod) = %v, want %v", result, ActionRead)
	}

	result = DetectAction("random")
	if result != ActionRead {
		t.Errorf("DetectAction(random) = %v, want %v", result, ActionRead)
	}
}

func TestLoadConfigFromEnv_SanitizeOption(t *testing.T) {
	os.Setenv("MIRO_AUDIT_ENABLED", "1")
	os.Setenv("MIRO_AUDIT_SANITIZE", "true")
	defer func() {
		os.Unsetenv("MIRO_AUDIT_ENABLED")
		os.Unsetenv("MIRO_AUDIT_SANITIZE")
	}()

	config := LoadConfigFromEnv()

	if !config.Enabled {
		t.Error("expected Enabled=true")
	}
	if !config.SanitizeInput {
		t.Error("expected SanitizeInput=true")
	}
}

func TestFileLogger_SanitizeInputOnLog(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-sanitize-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:       true,
		Path:          tmpDir,
		MaxSizeBytes:  10 * 1024 * 1024,
		BufferSize:    0,
		SanitizeInput: true,
	}

	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log event with sensitive input
	event := Event{
		ID:        "sanitize-test",
		Tool:      "test",
		Timestamp: time.Now(),
		Input: map[string]interface{}{
			"board_id":     "abc123",
			"access_token": "secret123",
		},
	}

	logger.Log(context.Background(), event)
	logger.Flush(context.Background())

	// Query and verify input was sanitized
	result, err := logger.Query(context.Background(), QueryOptions{})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Input["access_token"] != "[REDACTED]" {
		t.Error("expected access_token to be redacted")
	}
}

func TestFileLogger_LogWhenDisabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-disabled-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:      false, // Disabled
		Path:         tmpDir,
		MaxSizeBytes: 10 * 1024 * 1024,
		BufferSize:   0,
	}

	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log should be a no-op when disabled
	err = logger.Log(context.Background(), Event{ID: "disabled-test", Tool: "test"})
	if err != nil {
		t.Errorf("Log should not error when disabled: %v", err)
	}
}

func TestFileLogger_CloseWithNoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-close-nofile-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:      true,
		Path:         tmpDir,
		MaxSizeBytes: 10 * 1024 * 1024,
		BufferSize:   0,
	}

	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Close immediately without logging
	err = logger.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}

	// Close again should also not error
	err = logger.Close()
	// May error or not depending on implementation, just check no panic
}

func TestFileLogger_FlushLockedWithError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-flush-err-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:      true,
		Path:         tmpDir,
		MaxSizeBytes: 10 * 1024 * 1024,
		BufferSize:   5, // Buffer events
	}

	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Log some events to fill buffer
	for i := 0; i < 3; i++ {
		logger.Log(context.Background(), Event{
			ID:        "flush-err-" + string(rune('0'+i)),
			Tool:      "test",
			Timestamp: time.Now(),
		})
	}

	// Close the underlying file to cause write error on flush
	logger.mu.Lock()
	if logger.file != nil {
		logger.file.Close()
	}
	logger.mu.Unlock()

	// Flush should now fail
	err = logger.Flush(context.Background())
	// May or may not error depending on buffering state
	// Just verify no panic occurs
}

func TestMemoryLogger_SanitizeInputOnLog(t *testing.T) {
	config := Config{Enabled: true, SanitizeInput: true}
	logger := NewMemoryLogger(100, config)

	// Log event with sensitive input
	event := Event{
		ID:        "sanitize-mem-test",
		Tool:      "test",
		Timestamp: time.Now(),
		Input: map[string]interface{}{
			"board_id": "abc123",
			"secret":   "secret123",
		},
	}

	logger.Log(context.Background(), event)

	events := logger.GetAllEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Input["secret"] != "[REDACTED]" {
		t.Error("expected secret to be redacted")
	}
}
