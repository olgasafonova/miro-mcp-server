package miro

// Tests for retry, rate limiting, and circuit breaker behavior, split out of
// client_test.go.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newResilienceClient starts a test server with the given handler and returns
// a client pointed at it. The server is closed automatically at test end.
func newResilienceClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

// newRetrySequenceClient serves the given HTTP statuses in order, one per
// request. Once the sequence is exhausted it answers with an empty success
// payload, unless repeatLast keeps replaying the final status forever.
// Rate-limit statuses carry a zero Retry-After header for fast tests.
func newRetrySequenceClient(t *testing.T, statuses []int, repeatLast bool, attempts *int) *Client {
	t.Helper()
	return newResilienceClient(t, func(w http.ResponseWriter, r *http.Request) {
		*attempts++
		idx := *attempts - 1
		if idx >= len(statuses) {
			if !repeatLast {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]interface{}{}})
				return
			}
			idx = len(statuses) - 1
		}
		if statuses[idx] == http.StatusTooManyRequests {
			w.Header().Set("Retry-After", "0") // Use 0 for faster tests
		}
		w.WriteHeader(statuses[idx])
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "upstream error"})
	})
}

// newStatusCountingClient always answers with the given status and counts
// requests through the provided counter.
func newStatusCountingClient(t *testing.T, status int, requestCount *int) *Client {
	t.Helper()
	return newResilienceClient(t, func(w http.ResponseWriter, r *http.Request) {
		*requestCount++
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  status,
			"message": http.StatusText(status),
		})
	})
}

func TestRateLimitRetry(t *testing.T) {
	t.Run("succeeds_after_retry", func(t *testing.T) {
		attempts := 0
		client := newRetrySequenceClient(t, []int{http.StatusTooManyRequests, http.StatusTooManyRequests}, false, &attempts)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// With retry logic, this should succeed after 2 rate limits + 1 success
		_, err := client.request(ctx, http.MethodGet, "/boards", nil)
		if err != nil {
			t.Fatalf("expected success after retry, got error: %v", err)
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("fails_after_max_retries", func(t *testing.T) {
		attempts := 0
		client := newRetrySequenceClient(t, []int{http.StatusTooManyRequests}, true, &attempts)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Should fail after 1 initial + MaxRetries (3) retry attempts = 4 total
		_, err := client.request(ctx, http.MethodGet, "/boards", nil)
		if err == nil {
			t.Fatal("expected error after max retries")
		}
		if !IsRateLimitError(err) {
			t.Errorf("expected rate limit error, got: %v", err)
		}
		// 1 initial attempt + MaxRetries retries
		expectedAttempts := 1 + MaxRetries
		if attempts != expectedAttempts {
			t.Errorf("expected %d attempts (1 initial + %d retries), got %d", expectedAttempts, MaxRetries, attempts)
		}
	})

	t.Run("retries_transient_errors", func(t *testing.T) {
		attempts := 0
		client := newRetrySequenceClient(t, []int{http.StatusServiceUnavailable, http.StatusBadGateway}, false, &attempts)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.request(ctx, http.MethodGet, "/boards", nil)
		if err != nil {
			t.Fatalf("expected success after retrying transient errors, got: %v", err)
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})
}

// newDelayConfig builds a rate limiter config for calculateDelay tests.
func newDelayConfig(maxDelay time.Duration, proactiveBuffer int) RateLimiterConfig {
	return RateLimiterConfig{
		MinDelay:          10 * time.Millisecond,
		MaxDelay:          maxDelay,
		ProactiveBuffer:   proactiveBuffer,
		SlowdownThreshold: 0.3, // 30%
	}
}

// rateLimiterDelay computes the delay a fresh rate limiter applies for state.
func rateLimiterDelay(config RateLimiterConfig, state RateLimitState) time.Duration {
	return NewAdaptiveRateLimiter(config).calculateDelay(state, config)
}

func TestCalculateDelay_SlowdownThreshold(t *testing.T) {
	// Test calculateDelay when below slowdown threshold
	config := newDelayConfig(100*time.Millisecond, 5)

	// State with 15% remaining (below 30% threshold) should cause delay
	state := RateLimitState{
		Limit:     100,
		Remaining: 15,
	}

	delay := rateLimiterDelay(config, state)
	if delay <= 0 {
		t.Errorf("expected delay > 0 when below threshold, got %v", delay)
	}
}

func TestCalculateDelay_AtBufferThreshold(t *testing.T) {
	// Test calculateDelay when at/below proactive buffer
	config := newDelayConfig(100*time.Millisecond, 10)

	// State at buffer (remaining <= ProactiveBuffer)
	state := RateLimitState{
		Limit:     100,
		Remaining: 5,           // Below buffer of 10
		ResetAt:   time.Time{}, // Zero time - fallback case
	}

	delay := rateLimiterDelay(config, state)
	if delay != config.MaxDelay {
		t.Errorf("expected max delay %v at buffer threshold, got %v", config.MaxDelay, delay)
	}
}

func TestCalculateDelay_WaitUntilReset(t *testing.T) {
	// Test calculateDelay waiting until reset time
	config := newDelayConfig(500*time.Millisecond, 10)

	// State at buffer with reset time in future
	state := RateLimitState{
		Limit:     100,
		Remaining: 5, // At buffer
		ResetAt:   time.Now().Add(200 * time.Millisecond),
	}

	delay := rateLimiterDelay(config, state)
	if delay <= 0 || delay > 250*time.Millisecond {
		t.Errorf("expected delay around 200ms, got %v", delay)
	}
}

func TestCalculateDelay_CapAtMaxDelay(t *testing.T) {
	// Test that delay is capped at MaxDelay when reset time is far away
	config := newDelayConfig(100*time.Millisecond, 10)

	// Reset time is 10 seconds in future, should cap at MaxDelay
	state := RateLimitState{
		Limit:     100,
		Remaining: 5,
		ResetAt:   time.Now().Add(10 * time.Second),
	}

	delay := rateLimiterDelay(config, state)
	if delay != config.MaxDelay {
		t.Errorf("expected delay capped at %v, got %v", config.MaxDelay, delay)
	}
}

// newHalfOpenBreaker opens a fresh circuit breaker, waits out its short
// timeout, and lets one request through so the breaker sits in half-open.
func newHalfOpenBreaker(t *testing.T) *CircuitBreaker {
	t.Helper()
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    2,
		Timeout:             1 * time.Millisecond, // Very short for testing
		MaxHalfOpenRequests: 5,
	})

	// Open the circuit by recording failures
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for circuit to transition to half-open
	time.Sleep(5 * time.Millisecond)

	// Allow a request to set half-open state
	if err := cb.Allow(); err != nil {
		t.Errorf("expected circuit to allow request in half-open state: %v", err)
	}
	return cb
}

func TestCircuitBreaker_RecordFailureInHalfOpen(t *testing.T) {
	cb := newHalfOpenBreaker(t)

	// Record failure in half-open state
	cb.RecordFailure()

	// Circuit should be open again
	if err := cb.Allow(); err == nil {
		t.Error("expected circuit to be open after failure in half-open state")
	}
}

func TestCircuitBreaker_RecordSuccessInHalfOpen(t *testing.T) {
	cb := newHalfOpenBreaker(t)

	// Record success
	cb.RecordSuccess()

	// Record another success to reach threshold
	if err := cb.Allow(); err != nil {
		t.Errorf("expected circuit to still allow: %v", err)
	}
	cb.RecordSuccess()

	// Circuit should now be closed
	state := cb.State()
	if state != CircuitClosed {
		t.Errorf("state = %v, want CircuitClosed", state)
	}
}

func TestRequest_ServerError500_TripsCircuitBreaker(t *testing.T) {
	// Tests that 5xx errors trip the circuit breaker
	requestCount := 0
	client := newStatusCountingClient(t, http.StatusInternalServerError, &requestCount)

	// Make several requests to trigger circuit breaker
	for i := 0; i < 5; i++ {
		_, err := client.GetBoard(context.Background(), GetBoardArgs{BoardID: "board123"})
		if err == nil {
			t.Error("expected error for 500 response")
		}
	}

	// Verify requests were made
	if requestCount < 3 {
		t.Errorf("expected at least 3 requests, got %d", requestCount)
	}
}

func TestRequest_ClientError4xx_DoesNotTripCircuitBreaker(t *testing.T) {
	// Tests that 4xx errors do NOT trip the circuit breaker
	requestCount := 0
	client := newStatusCountingClient(t, http.StatusNotFound, &requestCount)

	// Make requests - circuit breaker should NOT trip for 4xx
	for i := 0; i < 10; i++ {
		_, _ = client.GetBoard(context.Background(), GetBoardArgs{BoardID: "board123"})
	}

	// All 10 requests should have been made (circuit not tripped)
	if requestCount != 10 {
		t.Errorf("expected 10 requests (circuit should not trip for 4xx), got %d", requestCount)
	}
}
