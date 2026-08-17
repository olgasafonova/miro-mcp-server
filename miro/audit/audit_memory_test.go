package audit

import (
	"context"
	"testing"
	"time"
)

// newMemoryTestLogger builds an enabled in-memory logger for tests.
func newMemoryTestLogger(maxSize int) *MemoryLogger {
	return NewMemoryLogger(maxSize, Config{Enabled: true})
}

// logMemoryEvents logs count sequential events with the given ID prefix.
func logMemoryEvents(logger *MemoryLogger, prefix string, count int) {
	for i := 0; i < count; i++ {
		logger.Log(context.Background(), Event{
			ID:        prefix + string(rune('0'+i)),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Tool:      "miro_test",
		})
	}
}

// assertQueryCount runs a query and checks the number of returned events.
func assertQueryCount(t *testing.T, logger Logger, opts QueryOptions, want int) {
	t.Helper()
	result, err := logger.Query(context.Background(), opts)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result.Events) != want {
		t.Errorf("expected %d events, got %d", want, len(result.Events))
	}
}

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
	logger := newMemoryTestLogger(3) // Small buffer

	// Log 5 events
	logMemoryEvents(logger, "test-", 5)

	// Should only have last 3
	events := logger.GetAllEvents()
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestMemoryLogger_Query(t *testing.T) {
	logger := newMemoryTestLogger(100)

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
	assertQueryCount(t, logger, QueryOptions{Tool: "miro_create_sticky"}, 1)

	// Query by board
	assertQueryCount(t, logger, QueryOptions{BoardID: "board-1"}, 2)

	// Query by action
	assertQueryCount(t, logger, QueryOptions{Action: ActionDelete}, 1)

	// Query by success status
	success := true
	assertQueryCount(t, logger, QueryOptions{Success: &success}, 2)
}

func TestMemoryLogger_QueryPagination(t *testing.T) {
	logger := newMemoryTestLogger(100)

	// Log 10 events
	logMemoryEvents(logger, "test-", 10)

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
	logger := newMemoryTestLogger(100)

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

func TestMemoryLogger_Flush(t *testing.T) {
	logger := newMemoryTestLogger(100)

	// Log some events
	logMemoryEvents(logger, "flush-", 5)

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
	logger := newMemoryTestLogger(100)

	// Log some events
	logMemoryEvents(logger, "clear-", 5)

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
	logger := newMemoryTestLogger(100)

	// Log some events
	logger.Log(context.Background(), Event{ID: "close-1", Tool: "test"})

	// Close should be a no-op but should not error
	err := logger.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}
}

func TestMemoryLogger_QueryTimeRange(t *testing.T) {
	logger := newMemoryTestLogger(100)

	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)

	// Log events at different times
	logger.Log(context.Background(), Event{ID: "1", Tool: "test", Timestamp: earlier})
	logger.Log(context.Background(), Event{ID: "2", Tool: "test", Timestamp: now})
	logger.Log(context.Background(), Event{ID: "3", Tool: "test", Timestamp: later})

	// Query since now should exclude earlier event
	assertQueryCount(t, logger, QueryOptions{Since: now}, 2)

	// Query until now should exclude later event
	assertQueryCount(t, logger, QueryOptions{Until: now}, 2)
}

func TestMemoryLogger_QueryMethodAndUserID(t *testing.T) {
	logger := newMemoryTestLogger(100)

	// Log events with different methods and users
	logger.Log(context.Background(), Event{ID: "1", Method: "CreateSticky", UserID: "user-1", Timestamp: time.Now()})
	logger.Log(context.Background(), Event{ID: "2", Method: "CreateSticky", UserID: "user-2", Timestamp: time.Now()})
	logger.Log(context.Background(), Event{ID: "3", Method: "DeleteItem", UserID: "user-1", Timestamp: time.Now()})

	// Query by method
	assertQueryCount(t, logger, QueryOptions{Method: "CreateSticky"}, 2)

	// Query by user ID
	assertQueryCount(t, logger, QueryOptions{UserID: "user-1"}, 2)
}

func TestMemoryLogger_QueryOffsetExceedsTotal(t *testing.T) {
	logger := newMemoryTestLogger(100)

	// Log a few events
	logMemoryEvents(logger, "off-", 3)

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
	// Pass 0 or negative maxSize, should default to 1000
	logger := newMemoryTestLogger(0)

	// Verify we can log at least 1000 events without error
	for i := 0; i < 1000; i++ {
		logger.Log(context.Background(), Event{ID: "default-" + string(rune(i)), Tool: "test"})
	}

	events := logger.GetAllEvents()
	if len(events) != 1000 {
		t.Errorf("expected 1000 events with default maxSize, got %d", len(events))
	}
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
