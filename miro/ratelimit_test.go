package miro

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestAdaptiveRateLimiter_NoDelayWhenFresh(t *testing.T) {
	rl := NewAdaptiveRateLimiter(DefaultRateLimiterConfig())

	// Fresh limiter should not apply any delay
	ctx := context.Background()
	delay, err := rl.Wait(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delay != 0 {
		t.Errorf("expected no delay for fresh limiter, got %v", delay)
	}
}

func TestAdaptiveRateLimiter_UpdateFromResponse(t *testing.T) {
	rl := NewAdaptiveRateLimiter(DefaultRateLimiterConfig())

	// Create a mock response with rate limit headers
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header.Set("X-RateLimit-Limit", "100")
	resp.Header.Set("X-RateLimit-Remaining", "50")
	resp.Header.Set("X-RateLimit-Reset", "60") // 60 seconds

	rl.UpdateFromResponse(resp)

	state := rl.State()
	if state.Limit != 100 {
		t.Errorf("expected limit 100, got %d", state.Limit)
	}
	if state.Remaining != 50 {
		t.Errorf("expected remaining 50, got %d", state.Remaining)
	}
	if state.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestAdaptiveRateLimiter_SlowsDownAtThreshold(t *testing.T) {
	config := RateLimiterConfig{
		SlowdownThreshold: 0.2,
		MinDelay:          10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		DefaultLimit:      100,
		ProactiveBuffer:   5,
	}
	rl := NewAdaptiveRateLimiter(config)

	// Simulate low remaining (10% which is below 20% threshold)
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("X-RateLimit-Limit", "100")
	resp.Header.Set("X-RateLimit-Remaining", "10")
	rl.UpdateFromResponse(resp)

	ctx := context.Background()
	start := time.Now()
	delay, err := rl.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delay == 0 {
		t.Error("expected delay when below threshold")
	}
	if elapsed < delay {
		t.Errorf("wait should have taken at least %v, took %v", delay, elapsed)
	}
}

func TestAdaptiveRateLimiter_NoDelayAboveThreshold(t *testing.T) {
	config := RateLimiterConfig{
		SlowdownThreshold: 0.2,
		MinDelay:          10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		DefaultLimit:      100,
		ProactiveBuffer:   5,
	}
	rl := NewAdaptiveRateLimiter(config)

	// Simulate plenty of remaining requests (50% remaining)
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("X-RateLimit-Limit", "100")
	resp.Header.Set("X-RateLimit-Remaining", "50")
	rl.UpdateFromResponse(resp)

	ctx := context.Background()
	delay, err := rl.Wait(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delay != 0 {
		t.Errorf("expected no delay above threshold, got %v", delay)
	}
}

func TestAdaptiveRateLimiter_WaitsUntilReset(t *testing.T) {
	config := RateLimiterConfig{
		SlowdownThreshold: 0.2,
		MinDelay:          10 * time.Millisecond,
		MaxDelay:          50 * time.Millisecond,
		DefaultLimit:      100,
		ProactiveBuffer:   5,
	}
	rl := NewAdaptiveRateLimiter(config)

	// Simulate exhausted rate limit with reset in 30ms
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("X-RateLimit-Limit", "100")
	resp.Header.Set("X-RateLimit-Remaining", "3") // At buffer threshold
	resetTime := time.Now().Add(30 * time.Millisecond).Unix()
	resp.Header.Set("X-RateLimit-Reset", formatUnixTimestamp(resetTime))
	rl.UpdateFromResponse(resp)

	ctx := context.Background()
	delay, err := rl.Wait(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should wait up to MaxDelay (50ms) since we're at buffer
	if delay == 0 {
		t.Error("expected delay when at buffer threshold")
	}
}

func formatUnixTimestamp(ts int64) string {
	return string(rune('0'+ts/1000000000)) + string(rune('0'+(ts/100000000)%10)) +
		string(rune('0'+(ts/10000000)%10)) + string(rune('0'+(ts/1000000)%10)) +
		string(rune('0'+(ts/100000)%10)) + string(rune('0'+(ts/10000)%10)) +
		string(rune('0'+(ts/1000)%10)) + string(rune('0'+(ts/100)%10)) +
		string(rune('0'+(ts/10)%10)) + string(rune('0'+ts%10))
}

func TestAdaptiveRateLimiter_ContextCancellation(t *testing.T) {
	config := RateLimiterConfig{
		SlowdownThreshold: 0.2,
		MinDelay:          10 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		DefaultLimit:      100,
		ProactiveBuffer:   5,
	}
	rl := NewAdaptiveRateLimiter(config)

	// Simulate low remaining to trigger delay
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("X-RateLimit-Limit", "100")
	resp.Header.Set("X-RateLimit-Remaining", "5") // At buffer
	rl.UpdateFromResponse(resp)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := rl.Wait(ctx)
	if err == nil {
		t.Error("expected context cancelled error")
	}
}

func TestAdaptiveRateLimiter_Stats(t *testing.T) {
	rl := NewAdaptiveRateLimiter(DefaultRateLimiterConfig())

	// Make a few requests
	ctx := context.Background()
	rl.Wait(ctx)
	rl.Wait(ctx)
	rl.Wait(ctx)

	stats := rl.Stats()
	if stats.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", stats.TotalRequests)
	}
}

func TestAdaptiveRateLimiter_Reset(t *testing.T) {
	config := RateLimiterConfig{
		SlowdownThreshold: 0.2,
		MinDelay:          10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		DefaultLimit:      100,
		ProactiveBuffer:   5,
	}
	rl := NewAdaptiveRateLimiter(config)

	// Update state
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("X-RateLimit-Limit", "50")
	resp.Header.Set("X-RateLimit-Remaining", "5")
	rl.UpdateFromResponse(resp)

	// Reset
	rl.Reset()

	state := rl.State()
	if state.Limit != 100 {
		t.Errorf("expected default limit 100 after reset, got %d", state.Limit)
	}
	if state.Remaining != 100 {
		t.Errorf("expected default remaining 100 after reset, got %d", state.Remaining)
	}

	stats := rl.Stats()
	if stats.TotalRequests != 0 {
		t.Errorf("expected 0 total requests after reset, got %d", stats.TotalRequests)
	}
}

func TestAdaptiveRateLimiter_NilResponse(t *testing.T) {
	rl := NewAdaptiveRateLimiter(DefaultRateLimiterConfig())

	// Should not panic on nil response
	rl.UpdateFromResponse(nil)

	state := rl.State()
	if state.Limit != 100 {
		t.Errorf("expected default limit after nil response, got %d", state.Limit)
	}
}

func TestRateLimitState_IsStale(t *testing.T) {
	state := RateLimitState{
		UpdatedAt: time.Now().Add(-2 * time.Minute),
	}
	if !state.IsStale() {
		t.Error("state older than 1 minute should be stale")
	}

	state.UpdatedAt = time.Now()
	if state.IsStale() {
		t.Error("fresh state should not be stale")
	}
}

func TestRateLimitState_PercentRemaining(t *testing.T) {
	tests := []struct {
		limit     int
		remaining int
		expected  float64
	}{
		{100, 50, 0.5},
		{100, 0, 0.0},
		{100, 100, 1.0},
		{0, 50, 1.0}, // Edge case: zero limit
	}

	for _, tt := range tests {
		state := RateLimitState{Limit: tt.limit, Remaining: tt.remaining}
		got := state.PercentRemaining()
		if got != tt.expected {
			t.Errorf("PercentRemaining(%d/%d) = %v, want %v", tt.remaining, tt.limit, got, tt.expected)
		}
	}
}

func TestAdaptiveRateLimiter_UnixTimestampReset(t *testing.T) {
	rl := NewAdaptiveRateLimiter(DefaultRateLimiterConfig())

	// Create a mock response with Unix timestamp reset
	resetTime := time.Now().Add(time.Minute).Unix()
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("X-RateLimit-Limit", "100")
	resp.Header.Set("X-RateLimit-Remaining", "50")
	resp.Header.Set("X-RateLimit-Reset", formatInt64(resetTime))
	rl.UpdateFromResponse(resp)

	state := rl.State()
	// Reset should be approximately 1 minute from now
	if state.ResetAt.Before(time.Now().Add(50 * time.Second)) {
		t.Error("reset time should be about 1 minute in the future")
	}
}

func formatInt64(n int64) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
