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

// UpdateFromResponse updates the rate limit state from response headers.
// Standard headers: X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset
func (r *AdaptiveRateLimiter) UpdateFromResponse(resp *http.Response) {
	if resp == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Parse rate limit headers
	if limit := resp.Header.Get("X-RateLimit-Limit"); limit != "" {
		if v, err := strconv.Atoi(limit); err == nil && v > 0 {
			r.state.Limit = v
		}
	}

	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		if v, err := strconv.Atoi(remaining); err == nil && v >= 0 {
			r.state.Remaining = v
		}
	}

	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		// Reset can be a Unix timestamp or seconds until reset
		if v, err := strconv.ParseInt(reset, 10, 64); err == nil {
			if v > 1000000000 { // Looks like a Unix timestamp
				r.state.ResetAt = time.Unix(v, 0)
			} else { // Seconds until reset
				r.state.ResetAt = time.Now().Add(time.Duration(v) * time.Second)
			}
		}
	}

	r.state.UpdatedAt = time.Now()
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

// calculateDelay determines the appropriate delay based on current state.
func (r *AdaptiveRateLimiter) calculateDelay(state RateLimitState, config RateLimiterConfig) time.Duration {
	// If we have remaining requests above buffer, no delay needed
	if state.Remaining > config.ProactiveBuffer {
		percentRemaining := state.PercentRemaining()

		// Only slow down if below threshold
		if percentRemaining >= config.SlowdownThreshold {
			return 0
		}

		// Calculate delay proportional to how close we are to the limit
		// As remaining approaches 0, delay approaches MaxDelay
		ratio := 1.0 - (percentRemaining / config.SlowdownThreshold)
		delay := time.Duration(float64(config.MaxDelay-config.MinDelay)*ratio) + config.MinDelay
		return delay
	}

	// If remaining is at or below buffer, wait until reset
	if !state.ResetAt.IsZero() && state.ResetAt.After(time.Now()) {
		waitTime := time.Until(state.ResetAt)
		if waitTime > config.MaxDelay {
			return config.MaxDelay
		}
		return waitTime
	}

	// Fallback: apply max delay when at buffer threshold
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
