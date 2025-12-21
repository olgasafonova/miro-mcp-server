package webhooks

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// =============================================================================
// Config Tests
// =============================================================================

func TestLoadConfigFromEnv(t *testing.T) {
	// Test default config (without env vars)
	config := LoadConfigFromEnv()
	if config.Enabled {
		t.Error("expected Enabled to be false by default")
	}
}

func TestConfigIsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name:     "empty config",
			config:   Config{},
			expected: false,
		},
		{
			name: "with callback URL",
			config: Config{
				CallbackURL: "https://example.com/webhook",
			},
			expected: true,
		},
		{
			name: "fully configured",
			config: Config{
				Enabled:     true,
				CallbackURL: "https://example.com/webhook",
				Secret:      "test-secret",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsConfigured(); got != tt.expected {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// Handler Tests
// =============================================================================

func TestHandler_ChallengeValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(Config{}, logger)

	// Create challenge request
	payload := WebhookPayload{
		Challenge: "test-challenge-123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp ChallengeResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Challenge != "test-challenge-123" {
		t.Errorf("expected challenge 'test-challenge-123', got '%s'", resp.Challenge)
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(Config{}, logger)

	req := httptest.NewRequest(http.MethodGet, "/webhooks", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(Config{}, logger)

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandler_EventProcessing(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(Config{}, logger)

	// Subscribe to events
	eventCh := handler.EventBus().Subscribe("test")
	defer handler.EventBus().Unsubscribe("test")

	// Create event payload
	payload := WebhookPayload{
		Event: &EventPayload{
			Type: "board.item.create",
			Board: &EventBoard{
				ID: "board123",
			},
			Item: &EventItem{
				ID:   "item456",
				Type: "sticky_note",
			},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Check event was published
	select {
	case event := <-eventCh:
		if event.Type != EventItemCreate {
			t.Errorf("expected event type %s, got %s", EventItemCreate, event.Type)
		}
		if event.BoardID != "board123" {
			t.Errorf("expected board ID 'board123', got '%s'", event.BoardID)
		}
		if event.ItemID != "item456" {
			t.Errorf("expected item ID 'item456', got '%s'", event.ItemID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event")
	}

	// Check event count
	if handler.EventCount() != 1 {
		t.Errorf("expected event count 1, got %d", handler.EventCount())
	}
}

func TestHandler_SignatureVerification(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(Config{Secret: "test-secret"}, logger)

	payload := WebhookPayload{Challenge: "test"}
	body, _ := json.Marshal(payload)

	// Request without signature should fail
	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for missing signature, got %d", rr.Code)
	}
}

// =============================================================================
// EventBus Tests
// =============================================================================

func TestEventBus_SubscribePublish(t *testing.T) {
	bus := NewEventBus()

	// Subscribe
	ch1 := bus.Subscribe("sub1")
	ch2 := bus.Subscribe("sub2")

	// Publish event
	event := Event{
		Type:    EventItemCreate,
		BoardID: "board123",
	}
	bus.Publish(event)

	// Both subscribers should receive the event
	select {
	case received := <-ch1:
		if received.BoardID != "board123" {
			t.Error("sub1: wrong board ID")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("sub1: timeout")
	}

	select {
	case received := <-ch2:
		if received.BoardID != "board123" {
			t.Error("sub2: wrong board ID")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("sub2: timeout")
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus()

	ch := bus.Subscribe("sub1")
	if bus.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount())
	}

	bus.Unsubscribe("sub1")
	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCount())
	}

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed")
	}
}

func TestEventBus_Recent(t *testing.T) {
	bus := NewEventBus()

	// Publish multiple events
	for i := 0; i < 5; i++ {
		bus.Publish(Event{
			Type:    EventItemCreate,
			BoardID: string(rune('A' + i)),
		})
	}

	// Get recent events
	recent := bus.Recent(3)
	if len(recent) != 3 {
		t.Errorf("expected 3 recent events, got %d", len(recent))
	}

	// Should be in chronological order (oldest first)
	if recent[0].BoardID != "C" {
		t.Errorf("expected first event board ID 'C', got '%s'", recent[0].BoardID)
	}
	if recent[2].BoardID != "E" {
		t.Errorf("expected last event board ID 'E', got '%s'", recent[2].BoardID)
	}
}

// =============================================================================
// RingBuffer Tests
// =============================================================================

func TestRingBuffer_Add(t *testing.T) {
	rb := NewRingBuffer(3)

	rb.Add(Event{BoardID: "A"})
	rb.Add(Event{BoardID: "B"})
	rb.Add(Event{BoardID: "C"})

	recent := rb.Recent(3)
	if len(recent) != 3 {
		t.Errorf("expected 3 events, got %d", len(recent))
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer(3)

	// Add 5 events to a buffer of size 3
	rb.Add(Event{BoardID: "A"})
	rb.Add(Event{BoardID: "B"})
	rb.Add(Event{BoardID: "C"})
	rb.Add(Event{BoardID: "D"})
	rb.Add(Event{BoardID: "E"})

	// Should only keep the last 3
	recent := rb.Recent(5)
	if len(recent) != 3 {
		t.Errorf("expected 3 events, got %d", len(recent))
	}

	// Should be C, D, E (A, B were overwritten)
	if recent[0].BoardID != "C" || recent[1].BoardID != "D" || recent[2].BoardID != "E" {
		t.Errorf("expected C, D, E; got %s, %s, %s", recent[0].BoardID, recent[1].BoardID, recent[2].BoardID)
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := NewRingBuffer(3)

	recent := rb.Recent(5)
	if recent != nil {
		t.Errorf("expected nil for empty buffer, got %v", recent)
	}
}

// =============================================================================
// EventFilter Tests
// =============================================================================

func TestEventFilter_Matches(t *testing.T) {
	tests := []struct {
		name    string
		filter  EventFilter
		event   Event
		matches bool
	}{
		{
			name:    "empty filter matches all",
			filter:  EventFilter{},
			event:   Event{BoardID: "board1", ItemType: "sticky_note", Type: EventItemCreate},
			matches: true,
		},
		{
			name:    "board filter matches",
			filter:  EventFilter{BoardID: "board1"},
			event:   Event{BoardID: "board1"},
			matches: true,
		},
		{
			name:    "board filter doesn't match",
			filter:  EventFilter{BoardID: "board1"},
			event:   Event{BoardID: "board2"},
			matches: false,
		},
		{
			name:    "item type filter matches",
			filter:  EventFilter{ItemType: "sticky_note"},
			event:   Event{ItemType: "sticky_note"},
			matches: true,
		},
		{
			name:    "event type filter matches",
			filter:  EventFilter{EventType: EventItemCreate},
			event:   Event{Type: EventItemCreate},
			matches: true,
		},
		{
			name:    "combined filter matches",
			filter:  EventFilter{BoardID: "board1", EventType: EventItemUpdate},
			event:   Event{BoardID: "board1", Type: EventItemUpdate},
			matches: true,
		},
		{
			name:    "combined filter partial match fails",
			filter:  EventFilter{BoardID: "board1", EventType: EventItemUpdate},
			event:   Event{BoardID: "board1", Type: EventItemDelete},
			matches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.Matches(tt.event); got != tt.matches {
				t.Errorf("Matches() = %v, want %v", got, tt.matches)
			}
		})
	}
}

// =============================================================================
// Event Types Tests
// =============================================================================

func TestAllEventTypes(t *testing.T) {
	types := AllEventTypes()
	if len(types) != 3 {
		t.Errorf("expected 3 event types, got %d", len(types))
	}

	expected := []EventType{EventItemCreate, EventItemUpdate, EventItemDelete}
	for i, typ := range expected {
		if types[i] != typ {
			t.Errorf("expected %s at position %d, got %s", typ, i, types[i])
		}
	}
}
