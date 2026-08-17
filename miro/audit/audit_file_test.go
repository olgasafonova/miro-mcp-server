package audit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fileTestConfig builds a standard unbuffered file-logger config rooted at dir.
func fileTestConfig(dir string) Config {
	return Config{
		Enabled:      true,
		Path:         dir,
		MaxSizeBytes: 10 * 1024 * 1024,
		BufferSize:   0, // No buffering for test
	}
}

// newFileTestLogger creates a FileLogger, failing the test on error.
func newFileTestLogger(t *testing.T, config Config) *FileLogger {
	t.Helper()
	logger, err := NewFileLogger(config)
	if err != nil {
		t.Fatalf("failed to create file logger: %v", err)
	}
	return logger
}

// writeOldAuditFile drops a 60-day-old audit file into dir and returns its path.
func writeOldAuditFile(t *testing.T, dir string) string {
	t.Helper()
	oldFile := filepath.Join(dir, "audit-2020-01-01T00-00-00.jsonl")
	os.WriteFile(oldFile, []byte("{}"), 0600)
	// Set modification time to 60 days ago
	oldTime := time.Now().AddDate(0, 0, -60)
	os.Chtimes(oldFile, oldTime, oldTime)
	return oldFile
}

// runCleanupCycle logs one event to trigger cleanup, closes the logger, and
// gives the cleanup goroutine time to run.
func runCleanupCycle(t *testing.T, config Config, eventID string) {
	t.Helper()
	logger := newFileTestLogger(t, config)
	logger.Log(context.Background(), Event{ID: eventID, Tool: "test"})
	logger.Close()
	time.Sleep(100 * time.Millisecond)
}

func TestFileLogger(t *testing.T) {
	tmpDir := t.TempDir()

	config := fileTestConfig(tmpDir)
	config.RetentionDays = 30
	config.SanitizeInput = true

	logger := newFileTestLogger(t, config)
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

	err := logger.Log(context.Background(), event)
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
	config := fileTestConfig(t.TempDir())
	config.BufferSize = 3 // Buffer 3 events before flushing

	logger := newFileTestLogger(t, config)
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

// cleanupLeavesOldFile seeds an old audit file, runs a cleanup cycle under the
// given retention, and reports whether the old file survived it.
func cleanupLeavesOldFile(t *testing.T, retentionDays int, eventID string) bool {
	t.Helper()
	tmpDir := t.TempDir()
	oldFile := writeOldAuditFile(t, tmpDir)

	config := fileTestConfig(tmpDir)
	config.RetentionDays = retentionDays

	runCleanupCycle(t, config, eventID)

	_, err := os.Stat(oldFile)
	return !os.IsNotExist(err)
}

func TestFileLogger_Cleanup(t *testing.T) {
	// With a 30 day retention, the 60-day-old file should be deleted
	if cleanupLeavesOldFile(t, 30, "cleanup-test") {
		t.Error("old file should have been deleted by cleanup")
	}
}

func TestFileLogger_CleanupRetentionZero(t *testing.T) {
	// With retention disabled, the old file should still exist
	if !cleanupLeavesOldFile(t, 0, "retention-zero-test") {
		t.Error("old file should not have been deleted when retention is 0")
	}
}

func TestFileLogger_CleanupWithDirectoryEntry(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a subdirectory (should be skipped by cleanup)
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)

	config := fileTestConfig(tmpDir)
	config.RetentionDays = 30

	runCleanupCycle(t, config, "cleanup-dir-test")

	// Subdirectory should still exist (not deleted by cleanup)
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Error("subdirectory should not have been deleted by cleanup")
	}
}

func TestFileLogger_CurrentFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	logger := newFileTestLogger(t, fileTestConfig(tmpDir))
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

func TestFileLogger_QueryReadDirError(t *testing.T) {
	tmpDir := t.TempDir()

	logger := newFileTestLogger(t, fileTestConfig(tmpDir))

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
	logger := newFileTestLogger(t, fileTestConfig(t.TempDir()))
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
	logger := newFileTestLogger(t, fileTestConfig(t.TempDir()))
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

func TestFileLogger_SanitizeInputOnLog(t *testing.T) {
	config := fileTestConfig(t.TempDir())
	config.SanitizeInput = true

	logger := newFileTestLogger(t, config)
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
	config := fileTestConfig(t.TempDir())
	config.Enabled = false // Disabled

	logger := newFileTestLogger(t, config)
	defer logger.Close()

	// Log should be a no-op when disabled
	err := logger.Log(context.Background(), Event{ID: "disabled-test", Tool: "test"})
	if err != nil {
		t.Errorf("Log should not error when disabled: %v", err)
	}
}

func TestFileLogger_CloseWithNoFile(t *testing.T) {
	logger := newFileTestLogger(t, fileTestConfig(t.TempDir()))

	// Close immediately without logging
	err := logger.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}

	// Close again should also not error
	_ = logger.Close()
	// May error or not depending on implementation, just check no panic
}

func TestFileLogger_FlushLockedWithError(t *testing.T) {
	config := fileTestConfig(t.TempDir())
	config.BufferSize = 5 // Buffer events

	logger := newFileTestLogger(t, config)

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
	_ = logger.Flush(context.Background())
	// May or may not error depending on buffering state
	// Just verify no panic occurs
}
