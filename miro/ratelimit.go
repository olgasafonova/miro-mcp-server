package miro

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// =============================================================================
// Rate Limiter Configuration
// =============================================================================

// RateLimiterConfig holds configuration for adaptive rate limiting.
type RateLimiterConfig struct {
	// SlowdownThreshold is the percentage of remaining requests at which
	// to start slowing down (e.g., 0.2 = slow down when 20% remaining).
	SlowdownThreshold float64

	// MinDelay is the minimum delay between requests when slowing down.
	MinDelay time.Duration

	// MaxDelay is the maximum delay between requests when slowing down.
	MaxDelay time.Duration

	// DefaultLimit is the assumed limit if we haven't seen headers yet.
	DefaultLimit int

	// ProactiveBuffer is how many requests to keep in reserve.
	ProactiveBuffer int
}

// DefaultRateLimiterConfig returns sensible defaults for Miro's rate limits.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		SlowdownThreshold: 0.2, // Slow down at 20% remaining
		MinDelay:          100 * time.Millisecond,
		MaxDelay:          2 * time.Second,
		DefaultLimit:      100, // Miro's default is ~100/min
		ProactiveBuffer:   5,   // Keep 5 requests in reserve
	}
}

// =============================================================================
// Rate Limit State
// =============================================================================

// RateLimitState tracks the current rate limit state from API responses.
type RateLimitState struct {
	Limit     int       // Max requests per window
	Remaining int       // Requests remaining in current window
	ResetAt   time.Time // When the window resets
	UpdatedAt time.Time // When this state was last updated
}

// IsStale returns true if the state is older than 1 minute.
func (s *RateLimitState) IsStale() bool {
	return time.Since(s.UpdatedAt) > time.Minute
}

// PercentRemaining returns the percentage of remaining requests.
func (s *RateLimitState) PercentRemaining() float64 {
	if s.Limit == 0 {
		return 1.0
	}
	return float64(s.Remaining) / float64(s.Limit)
}

// =============================================================================
// Adaptive Rate Limiter
// =============================================================================

// AdaptiveRateLimiter implements rate limiting that adapts based on
// rate limit headers from API responses.
type AdaptiveRateLimiter struct {
	mu     sync.RWMutex
	config RateLimiterConfig
	state  RateLimitState

	// Stats
	totalRequests int64
	totalDelays   int64
	totalDelayMs  int64
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter.
func NewAdaptiveRateLimiter(config RateLimiterConfig) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		config: config,
		state: RateLimitState{
			Limit:     config.DefaultLimit,
			Remaining: config.DefaultLimit,
		},
	}
}

// parseIntHeader returns the parsed int when the header is a valid integer
// at or above min, and (0, false) otherwise.
func parseIntHeader(value string, min int) (int, bool) {
	if value == "" {
		return 0, false
	}
	v, err := strconv.Atoi(value)
	if err != nil || v < min {
		return 0, false
	}
	return v, true
}

// parseResetAt converts an X-RateLimit-Reset header into an absolute time.
// Values greater than 1e9 are treated as Unix timestamps; smaller values are
// treated as seconds-until-reset.
func parseResetAt(value string, now time.Time) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	if v > 1000000000 {
		return time.Unix(v, 0), true
	}
	return now.Add(time.Duration(v) * time.Second), true
}

// UpdateFromResponse updates the rate limit state from response headers.
// Standard headers: X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset
func (r *AdaptiveRateLimiter) UpdateFromResponse(resp *http.Response) {
	if resp == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if v, ok := parseIntHeader(resp.Header.Get("X-RateLimit-Limit"), 1); ok {
		r.state.Limit = v
	}
	if v, ok := parseIntHeader(resp.Header.Get("X-RateLimit-Remaining"), 0); ok {
		r.state.Remaining = v
	}
	now := time.Now()
	if t, ok := parseResetAt(resp.Header.Get("X-RateLimit-Reset"), now); ok {
		r.state.ResetAt = t
	}
	r.state.UpdatedAt = now
}

// Wait blocks until it's safe to make a request, or ctx is cancelled.
// Returns the delay that was applied.
func (r *AdaptiveRateLimiter) Wait(ctx context.Context) (time.Duration, error) {
	r.mu.Lock()
	r.totalRequests++
	state := r.state
	config := r.config
	r.mu.Unlock()

	// If state is stale, don't apply delays
	if state.IsStale() {
		return 0, nil
	}

	// Calculate appropriate delay
	delay := r.calculateDelay(state, config)
	if delay == 0 {
		return 0, nil
	}

	// Track delay stats
	r.mu.Lock()
	r.totalDelays++
	r.totalDelayMs += delay.Milliseconds()
	r.mu.Unlock()

	// Wait with context cancellation support
	select {
	case <-time.After(delay):
		return delay, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// proportionalSlowdownDelay computes a delay that grows as the remaining
// budget approaches the slowdown threshold.
func proportionalSlowdownDelay(state RateLimitState, config RateLimiterConfig) time.Duration {
	percentRemaining := state.PercentRemaining()
	if percentRemaining >= config.SlowdownThreshold {
		return 0
	}
	ratio := 1.0 - (percentRemaining / config.SlowdownThreshold)
	return time.Duration(float64(config.MaxDelay-config.MinDelay)*ratio) + config.MinDelay
}

// waitUntilReset returns the time remaining until ResetAt, capped at MaxDelay.
// Returns 0 if ResetAt is unset or already in the past.
func waitUntilReset(state RateLimitState, config RateLimiterConfig) time.Duration {
	if state.ResetAt.IsZero() || !state.ResetAt.After(time.Now()) {
		return 0
	}
	waitTime := time.Until(state.ResetAt)
	if waitTime > config.MaxDelay {
		return config.MaxDelay
	}
	return waitTime
}

// calculateDelay determines the appropriate delay based on current state.
func (r *AdaptiveRateLimiter) calculateDelay(state RateLimitState, config RateLimiterConfig) time.Duration {
	if state.Remaining > config.ProactiveBuffer {
		return proportionalSlowdownDelay(state, config)
	}
	if wait := waitUntilReset(state, config); wait > 0 {
		return wait
	}
	return config.MaxDelay
}

// State returns a copy of the current rate limit state.
func (r *AdaptiveRateLimiter) State() RateLimitState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state
}

// RateLimiterStats holds statistics for the rate limiter.
type RateLimiterStats struct {
	TotalRequests int64
	TotalDelays   int64
	AvgDelayMs    float64
	CurrentState  RateLimitState
}

// Stats returns statistics for the rate limiter.
func (r *AdaptiveRateLimiter) Stats() RateLimiterStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	avgDelay := float64(0)
	if r.totalDelays > 0 {
		avgDelay = float64(r.totalDelayMs) / float64(r.totalDelays)
	}

	return RateLimiterStats{
		TotalRequests: r.totalRequests,
		TotalDelays:   r.totalDelays,
		AvgDelayMs:    avgDelay,
		CurrentState:  r.state,
	}
}

// Reset resets the rate limiter to its initial state.
func (r *AdaptiveRateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.state = RateLimitState{
		Limit:     r.config.DefaultLimit,
		Remaining: r.config.DefaultLimit,
	}
	r.totalRequests = 0
	r.totalDelays = 0
	r.totalDelayMs = 0
}
