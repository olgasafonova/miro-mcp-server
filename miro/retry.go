package miro

import (
	"net/http"
	"time"
)

// =============================================================================
// Retry Logic
// =============================================================================

// isRetriableError returns true if the error/status code should be retried.
func isRetriableError(statusCode int, err error) bool {
	// Retry on transient server errors
	if statusCode == http.StatusBadGateway || // 502
		statusCode == http.StatusServiceUnavailable || // 503
		statusCode == http.StatusGatewayTimeout || // 504
		statusCode == http.StatusTooManyRequests { // 429
		return true
	}
	// Retry on network errors (connection reset, timeout, etc.)
	if err != nil {
		errStr := err.Error()
		return contains(errStr, "connection reset") ||
			contains(errStr, "connection refused") ||
			contains(errStr, "i/o timeout") ||
			contains(errStr, "no such host") ||
			contains(errStr, "EOF")
	}
	return false
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
