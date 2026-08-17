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

// testLimiterConfig returns a limiter config tuned for fast tests.
func testLimiterConfig(maxDelay time.Duration) RateLimiterConfig {
	return RateLimiterConfig{
		SlowdownThreshold: 0.2,
		MinDelay:          10 * time.Millisecond,
		MaxDelay:          maxDelay,
		DefaultLimit:      100,
		ProactiveBuffer:   5,
	}
}

// applyRateHeaders feeds the limiter a response carrying the given rate headers.
func applyRateHeaders(rl *AdaptiveRateLimiter, headers map[string]string) {
	resp := &http.Response{Header: make(http.Header)}
	for k, v := range headers {
		resp.Header.Set(k, v)
	}
	rl.UpdateFromResponse(resp)
}

func TestAdaptiveRateLimiter_ThresholdBehavior(t *testing.T) {
	tests := []struct {
		name      string
		remaining string
		wantDelay bool
	}{
		// 10% remaining is below the 20% threshold, so a delay applies.
		{name: "SlowsDownAtThreshold", remaining: "10", wantDelay: true},
		// 50% remaining is plenty, so no delay applies.
		{name: "NoDelayAboveThreshold", remaining: "50", wantDelay: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewAdaptiveRateLimiter(testLimiterConfig(100 * time.Millisecond))
			applyRateHeaders(rl, map[string]string{
				"X-RateLimit-Limit":     "100",
				"X-RateLimit-Remaining": tt.remaining,
			})

			start := time.Now()
			delay, err := rl.Wait(context.Background())
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantDelay && delay == 0 {
				t.Error("expected delay when below threshold")
			}
			if !tt.wantDelay && delay != 0 {
				t.Errorf("expected no delay above threshold, got %v", delay)
			}
			if elapsed < delay {
				t.Errorf("wait should have taken at least %v, took %v", delay, elapsed)
			}
		})
	}
}

func TestAdaptiveRateLimiter_WaitsUntilReset(t *testing.T) {
	rl := NewAdaptiveRateLimiter(testLimiterConfig(50 * time.Millisecond))

	// Simulate exhausted rate limit (at buffer threshold) with reset in 30ms
	resetTime := time.Now().Add(30 * time.Millisecond).Unix()
	applyRateHeaders(rl, map[string]string{
		"X-RateLimit-Limit":     "100",
		"X-RateLimit-Remaining": "3",
		"X-RateLimit-Reset":     formatUnixTimestamp(resetTime),
	})

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
	rl := NewAdaptiveRateLimiter(testLimiterConfig(1 * time.Second))

	// Simulate low remaining (at buffer) to trigger delay
	applyRateHeaders(rl, map[string]string{
		"X-RateLimit-Limit":     "100",
		"X-RateLimit-Remaining": "5",
	})

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
	rl := NewAdaptiveRateLimiter(testLimiterConfig(100 * time.Millisecond))

	// Update state
	applyRateHeaders(rl, map[string]string{
		"X-RateLimit-Limit":     "50",
		"X-RateLimit-Remaining": "5",
	})

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
