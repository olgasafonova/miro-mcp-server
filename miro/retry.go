package miro

import (
	"net/http"
	"time"
)

// =============================================================================
// Retry Logic
// =============================================================================

// retriableStatusCodes lists HTTP status codes considered transient.
var retriableStatusCodes = map[int]bool{
	http.StatusBadGateway:         true, // 502
	http.StatusServiceUnavailable: true, // 503
	http.StatusGatewayTimeout:     true, // 504
	http.StatusTooManyRequests:    true, // 429
}

// retriableNetworkErrorSubstrings names lower-level network-error markers.
var retriableNetworkErrorSubstrings = []string{
	"connection reset",
	"connection refused",
	"i/o timeout",
	"no such host",
	"EOF",
}

// isRetriableNetworkError reports whether err is one of the transient
// network failures we retry.
func isRetriableNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	for _, sub := range retriableNetworkErrorSubstrings {
		if contains(errStr, sub) {
			return true
		}
	}
	return false
}

// isRetriableError returns true if the error/status code should be retried.
func isRetriableError(statusCode int, err error) bool {
	if retriableStatusCodes[statusCode] {
		return true
	}
	return isRetriableNetworkError(err)
}

// contains is a simple string contains helper.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// calculateRetryDelay calculates the delay for a retry attempt using exponential backoff.
func calculateRetryDelay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	// Exponential backoff: 1s, 2s, 4s, capped at MaxRetryDelay
	delay := BaseRetryDelay * time.Duration(1<<uint(attempt))
	if delay > MaxRetryDelay {
		delay = MaxRetryDelay
	}
	return delay
}
