package miro

import (
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	// Should start in closed state
	if cb.State() != CircuitClosed {
		t.Errorf("expected closed state, got %s", cb.State())
	}

	// Should allow requests
	if err := cb.Allow(); err != nil {
		t.Errorf("expected request to be allowed, got error: %v", err)
	}

	// Record success - should stay closed
	cb.RecordSuccess()
	if cb.State() != CircuitClosed {
		t.Errorf("expected closed state after success, got %s", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    3,
		SuccessThreshold:    2,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Record failures up to threshold
	for i := 0; i < 3; i++ {
		cb.Allow()
		cb.RecordFailure()
	}

	// Should now be open
	if cb.State() != CircuitOpen {
		t.Errorf("expected open state after %d failures, got %s", 3, cb.State())
	}

	// Should reject requests
	if err := cb.Allow(); err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    1,
		Timeout:             50 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Trip the circuit
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Fatalf("expected open state, got %s", cb.State())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Should be half-open now
	if cb.State() != CircuitHalfOpen {
		t.Errorf("expected half-open state after timeout, got %s", cb.State())
	}

	// Should allow one test request
	if err := cb.Allow(); err != nil {
		t.Errorf("expected request to be allowed in half-open state, got: %v", err)
	}
}

func TestCircuitBreaker_ClosesAfterSuccessInHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    1,
		Timeout:             50 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Trip the circuit
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	// Wait for timeout to get to half-open
	time.Sleep(60 * time.Millisecond)

	// Allow a test request
	cb.Allow()

	// Record success - should close circuit
	cb.RecordSuccess()

	if cb.State() != CircuitClosed {
		t.Errorf("expected closed state after success in half-open, got %s", cb.State())
	}
}

func TestCircuitBreaker_ReopensAfterFailureInHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    1,
		Timeout:             50 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Trip the circuit
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	// Wait for timeout to get to half-open
	time.Sleep(60 * time.Millisecond)

	// Allow a test request
	cb.Allow()

	// Record failure - should reopen circuit
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("expected open state after failure in half-open, got %s", cb.State())
	}
}

func TestCircuitBreaker_TooManyHalfOpenRequests(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    2,
		Timeout:             50 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Trip the circuit
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	// Wait for timeout to get to half-open
	time.Sleep(60 * time.Millisecond)

	// First request should be allowed
	if err := cb.Allow(); err != nil {
		t.Errorf("first half-open request should be allowed, got: %v", err)
	}

	// Second request should be rejected
	if err := cb.Allow(); err != ErrTooManyRequests {
		t.Errorf("expected ErrTooManyRequests, got %v", err)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    2,
		Timeout:             time.Hour, // Long timeout
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Trip the circuit
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Fatalf("expected open state, got %s", cb.State())
	}

	// Reset should return to closed
	cb.Reset()

	if cb.State() != CircuitClosed {
		t.Errorf("expected closed state after reset, got %s", cb.State())
	}

	// Should allow requests again
	if err := cb.Allow(); err != nil {
		t.Errorf("expected request to be allowed after reset, got: %v", err)
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	// Generate some activity
	cb.Allow()
	cb.RecordSuccess()
	cb.Allow()
	cb.RecordSuccess()
	cb.Allow()
	cb.RecordFailure()

	stats := cb.Stats()

	if stats.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", stats.TotalRequests)
	}
	if stats.TotalSuccesses != 2 {
		t.Errorf("expected 2 successes, got %d", stats.TotalSuccesses)
	}
	if stats.TotalFailures != 1 {
		t.Errorf("expected 1 failure, got %d", stats.TotalFailures)
	}
	if stats.State != "closed" {
		t.Errorf("expected closed state, got %s", stats.State)
	}
}

func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    3,
		SuccessThreshold:    1,
		Timeout:             time.Hour,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Record 2 failures (just below threshold)
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	// Record a success - should reset failure count
	cb.Allow()
	cb.RecordSuccess()

	// Record 2 more failures - should not trip circuit
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	// Should still be closed (failure count was reset)
	if cb.State() != CircuitClosed {
		t.Errorf("expected closed state, got %s", cb.State())
	}
}

func TestCircuitBreakerRegistry_GetCreatesNew(t *testing.T) {
	registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())

	cb1 := registry.Get("/boards")
	cb2 := registry.Get("/boards")
	cb3 := registry.Get("/items")

	// Same endpoint should return same circuit breaker
	if cb1 != cb2 {
		t.Error("expected same circuit breaker for same endpoint")
	}

	// Different endpoint should return different circuit breaker
	if cb1 == cb3 {
		t.Error("expected different circuit breaker for different endpoint")
	}
}

func TestCircuitBreakerRegistry_AllStats(t *testing.T) {
	registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())

	// Access a few endpoints
	registry.Get("/boards")
	registry.Get("/items")
	registry.Get("/tags")

	stats := registry.AllStats()

	if len(stats) != 3 {
		t.Errorf("expected 3 circuit breakers, got %d", len(stats))
	}

	if _, ok := stats["/boards"]; !ok {
		t.Error("expected /boards in stats")
	}
	if _, ok := stats["/items"]; !ok {
		t.Error("expected /items in stats")
	}
	if _, ok := stats["/tags"]; !ok {
		t.Error("expected /tags in stats")
	}
}

func TestCircuitBreakerRegistry_Reset(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    1,
		Timeout:             time.Hour,
		MaxHalfOpenRequests: 1,
	}
	registry := NewCircuitBreakerRegistry(config)

	// Trip a circuit breaker
	cb := registry.Get("/boards")
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Fatalf("expected open state, got %s", cb.State())
	}

	// Reset all
	registry.Reset()

	if cb.State() != CircuitClosed {
		t.Errorf("expected closed state after registry reset, got %s", cb.State())
	}
}

func TestExtractEndpoint(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/boards", "/boards"},
		{"/boards/abc123def456", "/boards/{id}"},
		{"/boards/uXjVOXQCe5c=/items", "/boards/{id}/items"},
		{"/boards/abc123/items/xyz789", "/boards/{id}/items/{id}"},
		{"/boards/abc123/sticky_notes", "/boards/{id}/sticky_notes"},
		{"/boards?limit=10", "/boards"},
	}

	for _, tt := range tests {
		got := extractEndpoint(tt.path)
		if got != tt.expected {
			t.Errorf("extractEndpoint(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.state.String()
		if got != tt.expected {
			t.Errorf("CircuitState(%d).String() = %q, want %q", tt.state, got, tt.expected)
		}
	}
}
