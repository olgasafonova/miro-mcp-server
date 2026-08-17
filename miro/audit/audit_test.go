package audit

import (
	"context"
	"errors"
	"testing"
	"time"
)

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
