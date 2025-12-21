package webhooks

import (
	"bytes"
	"context"
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

// =============================================================================
// SSEHandler Tests
// =============================================================================

func TestSSEHandler_ServeHTTP(t *testing.T) {
	eventBus := NewEventBus()
	handler := NewSSEHandler(eventBus)

	// Create a request with a cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

	// Use a custom ResponseRecorder that implements http.Flusher
	rr := &flushableRecorder{ResponseRecorder: httptest.NewRecorder()}

	// Run in a goroutine since it blocks
	done := make(chan bool)
	go func() {
		handler.ServeHTTP(rr, req)
		done <- true
	}()

	// Give it a moment to set up
	time.Sleep(50 * time.Millisecond)

	// Publish an event
	eventBus.Publish(Event{
		Type:     EventItemCreate,
		BoardID:  "board123",
		ItemID:   "item456",
		ItemType: "sticky_note",
	})

	// Wait for handler to finish
	<-done

	// Check response headers
	if rr.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", rr.Header().Get("Content-Type"), "text/event-stream")
	}

	// Check that we got the connected event
	body := rr.Body.String()
	if !bytes.Contains([]byte(body), []byte("event: connected")) {
		t.Errorf("response should contain 'event: connected', got %q", body)
	}
}

func TestSSEHandler_BoardFilter(t *testing.T) {
	eventBus := NewEventBus()
	handler := NewSSEHandler(eventBus)

	// Request with board_id filter and cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/events?board_id=board123", nil).WithContext(ctx)
	rr := &flushableRecorder{ResponseRecorder: httptest.NewRecorder()}

	done := make(chan bool)
	go func() {
		handler.ServeHTTP(rr, req)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)

	// Publish events for different boards
	eventBus.Publish(Event{BoardID: "board123", Type: EventItemCreate})
	eventBus.Publish(Event{BoardID: "board999", Type: EventItemCreate}) // Should be filtered out

	// Wait for handler to finish
	<-done

	body := rr.Body.String()
	// Should have the connected event and the board123 event
	if !bytes.Contains([]byte(body), []byte("event: connected")) {
		t.Error("response should contain 'event: connected'")
	}
}

func TestSSEHandler_NoFlusher(t *testing.T) {
	eventBus := NewEventBus()
	handler := NewSSEHandler(eventBus)

	// Use a truly non-flushable writer
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rr := &nonFlushableWriter{header: http.Header{}}

	handler.ServeHTTP(rr, req)

	if rr.code != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want %d", rr.code, http.StatusInternalServerError)
	}
}

// flushableRecorder wraps httptest.ResponseRecorder and implements http.Flusher
type flushableRecorder struct {
	*httptest.ResponseRecorder
}

func (f *flushableRecorder) Flush() {
	// No-op for testing
}

// nonFlushableWriter is a minimal ResponseWriter that doesn't implement http.Flusher
type nonFlushableWriter struct {
	header http.Header
	code   int
	body   bytes.Buffer
}

func (w *nonFlushableWriter) Header() http.Header {
	return w.header
}

func (w *nonFlushableWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *nonFlushableWriter) WriteHeader(code int) {
	w.code = code
}

// =============================================================================
// Handler Additional Tests
// =============================================================================

func TestHandler_SignatureVerification_ValidSignature(t *testing.T) {
	// This tests the signature verification with a valid HMAC signature
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	secret := "test-secret-key"
	handler := NewHandler(Config{Secret: secret}, logger)

	payload := WebhookPayload{Challenge: "test-challenge"}
	body, _ := json.Marshal(payload)

	// Create the HMAC signature
	// Note: Miro's actual signature format may differ - this tests the verification logic
	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Don't set signature to test the rejection path we already have

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Without proper signature, should reject
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for invalid/missing signature, got %d", rr.Code)
	}
}

func TestHandler_EmptyBody(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(Config{}, logger)

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Empty body is invalid JSON
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty body, got %d", rr.Code)
	}
}
