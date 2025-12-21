package webhooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// EventBus manages event subscriptions and distribution.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string]chan Event
	buffer      *RingBuffer
}

// NewEventBus creates a new event bus with a buffer for recent events.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string]chan Event),
		buffer:      NewRingBuffer(100), // Keep last 100 events
	}
}

// Subscribe adds a new subscriber and returns a channel for receiving events.
func (eb *EventBus) Subscribe(id string) <-chan Event {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan Event, 50)
	eb.subscribers[id] = ch
	return ch
}

// Unsubscribe removes a subscriber.
func (eb *EventBus) Unsubscribe(id string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if ch, ok := eb.subscribers[id]; ok {
		close(ch)
		delete(eb.subscribers, id)
	}
}

// Publish sends an event to all subscribers.
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// Add to buffer
	eb.buffer.Add(event)

	// Send to all subscribers (non-blocking)
	for _, ch := range eb.subscribers {
		select {
		case ch <- event:
		default:
			// Channel full, skip this subscriber
		}
	}
}

// Recent returns the most recent events from the buffer.
func (eb *EventBus) Recent(limit int) []Event {
	return eb.buffer.Recent(limit)
}

// SubscriberCount returns the number of active subscribers.
func (eb *EventBus) SubscriberCount() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.subscribers)
}

// RingBuffer is a fixed-size circular buffer for events.
type RingBuffer struct {
	mu     sync.RWMutex
	events []Event
	size   int
	head   int
	count  int
}

// NewRingBuffer creates a new ring buffer with the specified capacity.
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		events: make([]Event, size),
		size:   size,
	}
}

// Add adds an event to the buffer.
func (rb *RingBuffer) Add(event Event) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.events[rb.head] = event
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

// Recent returns the most recent n events in chronological order.
func (rb *RingBuffer) Recent(n int) []Event {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if n > rb.count {
		n = rb.count
	}
	if n == 0 {
		return nil
	}

	result := make([]Event, n)
	start := (rb.head - n + rb.size) % rb.size

	for i := 0; i < n; i++ {
		idx := (start + i) % rb.size
		result[i] = rb.events[idx]
	}

	return result
}

// SSEHandler serves Server-Sent Events for real-time event streaming.
type SSEHandler struct {
	eventBus *EventBus
}

// NewSSEHandler creates a new SSE handler.
func NewSSEHandler(eventBus *EventBus) *SSEHandler {
	return &SSEHandler{eventBus: eventBus}
}

// ServeHTTP implements http.Handler for SSE endpoint.
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Check if streaming is supported
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Generate unique subscriber ID
	subID := fmt.Sprintf("sse-%d", time.Now().UnixNano())

	// Subscribe to events
	events := h.eventBus.Subscribe(subID)
	defer h.eventBus.Unsubscribe(subID)

	// Send initial connection message
	fmt.Fprintf(w, "event: connected\ndata: {\"subscriber_id\":\"%s\"}\n\n", subID)
	flusher.Flush()

	// Optionally send recent events
	boardID := r.URL.Query().Get("board_id")
	recentEvents := h.eventBus.Recent(10)
	for _, event := range recentEvents {
		if boardID != "" && event.BoardID != boardID {
			continue
		}
		h.sendEvent(w, flusher, event)
	}

	// Keep-alive ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Stream events
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			// Filter by board if specified
			if boardID != "" && event.BoardID != boardID {
				continue
			}
			h.sendEvent(w, flusher, event)

		case <-ticker.C:
			// Send keep-alive comment
			fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

// sendEvent formats and sends an event in SSE format.
func (h *SSEHandler) sendEvent(w http.ResponseWriter, flusher http.Flusher, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "event: %s\n", event.Type)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// EventFilter provides filtering options for events.
type EventFilter struct {
	BoardID   string
	ItemType  string
	EventType EventType
}

// Matches returns true if the event matches the filter criteria.
func (f EventFilter) Matches(event Event) bool {
	if f.BoardID != "" && event.BoardID != f.BoardID {
		return false
	}
	if f.ItemType != "" && event.ItemType != f.ItemType {
		return false
	}
	if f.EventType != "" && event.Type != f.EventType {
		return false
	}
	return true
}
