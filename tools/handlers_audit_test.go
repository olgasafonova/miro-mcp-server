package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/olgasafonova/miro-mcp-server/miro"
	"github.com/olgasafonova/miro-mcp-server/miro/audit"
	"github.com/olgasafonova/miro-mcp-server/miro/desirepath"
)

// =============================================================================
// Mock Audit Logger
// =============================================================================

// MockAuditLogger is a mock implementation of audit.Logger for testing.
type MockAuditLogger struct {
	LogFn   func(ctx context.Context, event audit.Event) error
	QueryFn func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error)
	events  []audit.Event
}

func (m *MockAuditLogger) Log(ctx context.Context, event audit.Event) error {
	m.events = append(m.events, event)
	if m.LogFn != nil {
		return m.LogFn(ctx, event)
	}
	return nil
}

func (m *MockAuditLogger) Query(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
	if m.QueryFn != nil {
		return m.QueryFn(ctx, opts)
	}
	// Default: return all events
	return &audit.QueryResult{
		Events:  m.events,
		Total:   len(m.events),
		HasMore: false,
	}, nil
}

func (m *MockAuditLogger) Flush(ctx context.Context) error {
	return nil
}

func (m *MockAuditLogger) Close() error {
	return nil
}

// newAuditTestRegistry wires a registry to a MockAuditLogger whose Query is
// served by queryFn.
func newAuditTestRegistry(queryFn func(context.Context, audit.QueryOptions) (*audit.QueryResult, error)) *HandlerRegistry {
	registry := newTestRegistry(&MockClient{})
	registry.WithAuditLogger(&MockAuditLogger{QueryFn: queryFn})
	return registry
}

// newOptsCapturingRegistry wires a registry to an audit logger that records
// the QueryOptions it receives into captured and returns an empty result.
func newOptsCapturingRegistry(captured *audit.QueryOptions) *HandlerRegistry {
	return newAuditTestRegistry(func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
		*captured = opts
		return &audit.QueryResult{Events: []audit.Event{}, Total: 0}, nil
	})
}

// newFixedAuditRegistry wires a registry to an audit logger that always
// answers queries with the given fixed result.
func newFixedAuditRegistry(events []audit.Event, total int, hasMore bool) *HandlerRegistry {
	return newAuditTestRegistry(func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
		return &audit.QueryResult{Events: events, Total: total, HasMore: hasMore}, nil
	})
}

// =============================================================================
// GetAuditLog Tests
// =============================================================================

// successAuditEvents is the two-event fixture behind TestGetAuditLog_Success:
// one successful create followed by a failed delete.
func successAuditEvents(now time.Time) []audit.Event {
	return []audit.Event{
		{
			ID:         "event1",
			Timestamp:  now,
			Tool:       "miro_create_sticky",
			Action:     audit.ActionCreate,
			BoardID:    "board123",
			ItemID:     "item456",
			Success:    true,
			DurationMs: 150,
		},
		{
			ID:         "event2",
			Timestamp:  now.Add(-time.Hour),
			Tool:       "miro_delete_item",
			Action:     audit.ActionDelete,
			BoardID:    "board123",
			ItemID:     "item789",
			Success:    false,
			Error:      "item not found",
			DurationMs: 50,
		},
	}
}

// assertSuccessAuditEvents verifies the returned events mirror the
// successAuditEvents fixture: identifiers on the first, failure on the second.
func assertSuccessAuditEvents(t *testing.T, result miro.GetAuditLogResult) {
	t.Helper()
	if result.Events[0].ID != "event1" {
		t.Errorf("Events[0].ID = %q, want 'event1'", result.Events[0].ID)
	}
	if result.Events[0].Tool != "miro_create_sticky" {
		t.Errorf("Events[0].Tool = %q, want 'miro_create_sticky'", result.Events[0].Tool)
	}
	if !result.Events[0].Success {
		t.Error("Events[0].Success should be true")
	}
	if result.Events[1].Success {
		t.Error("Events[1].Success should be false")
	}
	if result.Events[1].Error != "item not found" {
		t.Errorf("Events[1].Error = %q, want 'item not found'", result.Events[1].Error)
	}
}

func TestGetAuditLog_Success(t *testing.T) {
	registry := newFixedAuditRegistry(successAuditEvents(time.Now()), 2, false)

	result, err := registry.GetAuditLog(context.Background(), miro.GetAuditLogArgs{BoardID: "board123"})
	mustSucceed(t, err)
	if len(result.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(result.Events))
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if result.Message == "" {
		t.Error("Message should not be empty")
	}
	assertSuccessAuditEvents(t, result)
}

func TestGetAuditLog_EmptyResult(t *testing.T) {
	registry := newFixedAuditRegistry([]audit.Event{}, 0, false)

	result, err := registry.GetAuditLog(context.Background(), miro.GetAuditLogArgs{})
	mustSucceed(t, err)
	if len(result.Events) != 0 {
		t.Errorf("expected 0 events, got %d", len(result.Events))
	}
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
}

func TestGetAuditLog_InvalidSinceTime(t *testing.T) {
	registry := newTestRegistry(&MockClient{})
	registry.WithAuditLogger(&MockAuditLogger{})

	_, err := registry.GetAuditLog(context.Background(), miro.GetAuditLogArgs{
		Since: "not-a-valid-time",
	})

	if err == nil {
		t.Fatal("expected error for invalid 'since' time")
	}
	if !errors.Is(err, err) || err.Error() == "" {
		t.Errorf("error should mention invalid time format")
	}
}

func TestGetAuditLog_InvalidUntilTime(t *testing.T) {
	registry := newTestRegistry(&MockClient{})
	registry.WithAuditLogger(&MockAuditLogger{})

	_, err := registry.GetAuditLog(context.Background(), miro.GetAuditLogArgs{
		Until: "invalid-time-format",
	})

	if err == nil {
		t.Fatal("expected error for invalid 'until' time")
	}
}

func TestGetAuditLog_ValidTimeRange(t *testing.T) {
	var capturedOpts audit.QueryOptions
	registry := newOptsCapturingRegistry(&capturedOpts)

	_, err := registry.GetAuditLog(context.Background(), miro.GetAuditLogArgs{
		Since: "2024-01-01T00:00:00Z",
		Until: "2024-01-31T23:59:59Z",
	})
	mustSucceed(t, err)

	// Verify the times were parsed correctly
	if capturedOpts.Since.IsZero() {
		t.Error("Since time should be set")
	}
	if capturedOpts.Until.IsZero() {
		t.Error("Until time should be set")
	}
	if got := capturedOpts.Since.Format("2006-01-02"); got != "2024-01-01" {
		t.Errorf("Since = %v, want 2024-01-01", capturedOpts.Since)
	}
}

func TestGetAuditLog_LimitNormalization(t *testing.T) {
	tests := []struct {
		name        string
		give, want  int
		explanation string
	}{
		{name: "default limit", give: 0, want: 50, explanation: "50 (default)"},
		{name: "limit capped", give: 1000, want: 500, explanation: "500 (max cap)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedOpts audit.QueryOptions
			registry := newOptsCapturingRegistry(&capturedOpts)

			_, err := registry.GetAuditLog(context.Background(), miro.GetAuditLogArgs{Limit: tt.give})
			mustSucceed(t, err)

			if capturedOpts.Limit != tt.want {
				t.Errorf("Limit = %d, want %s", capturedOpts.Limit, tt.explanation)
			}
		})
	}
}

func TestGetAuditLog_QueryError(t *testing.T) {
	registry := newAuditTestRegistry(func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
		return nil, errors.New("database error")
	})

	_, err := registry.GetAuditLog(context.Background(), miro.GetAuditLogArgs{})

	if err == nil {
		t.Fatal("expected error from query")
	}
	if err.Error() != "audit query failed: database error" {
		t.Errorf("error = %q, want 'audit query failed: database error'", err.Error())
	}
}

func TestGetAuditLog_FiltersByToolAndAction(t *testing.T) {
	var capturedOpts audit.QueryOptions
	registry := newOptsCapturingRegistry(&capturedOpts)

	_, err := registry.GetAuditLog(context.Background(), miro.GetAuditLogArgs{
		Tool:    "miro_create_sticky",
		Action:  "create",
		BoardID: "board123",
	})
	mustSucceed(t, err)

	if capturedOpts.Tool != "miro_create_sticky" {
		t.Errorf("Tool = %q, want 'miro_create_sticky'", capturedOpts.Tool)
	}
	if capturedOpts.Action != "create" {
		t.Errorf("Action = %q, want 'create'", capturedOpts.Action)
	}
	if capturedOpts.BoardID != "board123" {
		t.Errorf("BoardID = %q, want 'board123'", capturedOpts.BoardID)
	}
}

func TestGetAuditLog_FiltersBySuccess(t *testing.T) {
	var capturedOpts audit.QueryOptions
	registry := newOptsCapturingRegistry(&capturedOpts)

	success := true
	_, err := registry.GetAuditLog(context.Background(), miro.GetAuditLogArgs{
		Success: &success,
	})
	mustSucceed(t, err)

	if capturedOpts.Success == nil {
		t.Error("Success filter should be set")
	} else if *capturedOpts.Success != true {
		t.Errorf("Success = %v, want true", *capturedOpts.Success)
	}
}

func TestGetAuditLog_HasMore(t *testing.T) {
	registry := newFixedAuditRegistry([]audit.Event{{ID: "event1"}}, 100, true)

	result, err := registry.GetAuditLog(context.Background(), miro.GetAuditLogArgs{Limit: 1})
	mustSucceed(t, err)
	if !result.HasMore {
		t.Error("HasMore should be true")
	}
	if result.Total != 100 {
		t.Errorf("Total = %d, want 100", result.Total)
	}
}

// =============================================================================
// Desire Path Tests
// =============================================================================

func TestWithDesirePathLogger(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	dpLogger := desirepath.NewLogger(desirepath.Config{Enabled: true, MaxEvents: 10}, testLogger())
	normalizers := []desirepath.Normalizer{
		&desirepath.WhitespaceNormalizer{},
	}

	result := registry.WithDesirePathLogger(dpLogger, normalizers)
	if result != registry {
		t.Error("WithDesirePathLogger should return the same registry for chaining")
	}
	if registry.desireLogger == nil {
		t.Error("desireLogger should be set")
	}
	if len(registry.normalizers) != 1 {
		t.Errorf("normalizers len = %d, want 1", len(registry.normalizers))
	}
}

func TestGetDesirePathReport_NoLogger(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	result, err := registry.GetDesirePathReport(context.Background(), miro.GetDesirePathReportArgs{})
	mustSucceed(t, err)
	if result.Message != "Desire path logging is not enabled" {
		t.Errorf("message = %q, want disabled message", result.Message)
	}
}

func TestGetDesirePathReport_WithEvents(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	dpLogger := desirepath.NewLogger(desirepath.Config{Enabled: true, MaxEvents: 100}, testLogger())
	registry.WithDesirePathLogger(dpLogger, nil)

	// Log some events directly
	dpLogger.Log(desirepath.Event{
		Tool:         "miro_get_board",
		Parameter:    "board_id",
		Rule:         "url_to_id",
		RawValue:     "https://miro.com/app/board/uXjVN123=/",
		NormalizedTo: "uXjVN123=",
	})
	dpLogger.Log(desirepath.Event{
		Tool:         "miro_list_items",
		Parameter:    "limit",
		Rule:         "string_to_numeric",
		RawValue:     `"10"`,
		NormalizedTo: "10",
	})

	result, err := registry.GetDesirePathReport(context.Background(), miro.GetDesirePathReportArgs{})
	mustSucceed(t, err)
	if result.TotalNormalizations != 2 {
		t.Errorf("total = %d, want 2", result.TotalNormalizations)
	}
	if len(result.RecentEvents) != 2 {
		t.Errorf("recent events = %d, want 2", len(result.RecentEvents))
	}
	if result.ByRule["url_to_id"] != 1 {
		t.Errorf("by_rule[url_to_id] = %d, want 1", result.ByRule["url_to_id"])
	}
}

func TestGetDesirePathReport_FilterByTool(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	dpLogger := desirepath.NewLogger(desirepath.Config{Enabled: true, MaxEvents: 100}, testLogger())
	registry.WithDesirePathLogger(dpLogger, nil)

	dpLogger.Log(desirepath.Event{Tool: "miro_get_board", Rule: "url_to_id"})
	dpLogger.Log(desirepath.Event{Tool: "miro_list_items", Rule: "string_to_numeric"})

	result, err := registry.GetDesirePathReport(context.Background(), miro.GetDesirePathReportArgs{
		Tool: "miro_get_board",
	})
	mustSucceed(t, err)
	for _, e := range result.RecentEvents {
		if e.Tool != "miro_get_board" {
			t.Errorf("filtered event has wrong tool: %q", e.Tool)
		}
	}
}
