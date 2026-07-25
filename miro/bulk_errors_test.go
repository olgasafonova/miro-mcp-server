package miro

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// =============================================================================
// Bulk Error Categorization Tests
// =============================================================================

func TestCategorizeAPIStatus(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		wantType      string
		wantRetriable bool
	}{
		{"400 is validation", 400, "validation", false},
		{"401 is auth", 401, "auth", false},
		{"403 is auth", 403, "auth", false},
		{"404 is not_found", 404, "not_found", false},
		{"429 is retriable rate_limit", 429, "rate_limit", true},
		{"500 is retriable server", 500, "server", true},
		{"503 is retriable server", 503, "server", true},
		{"418 falls through to api", 418, "api", false},
		{"302 falls through to api", 302, "api", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := categorizeAPIStatus(tt.status)
			if got.errorType != tt.wantType {
				t.Errorf("errorType = %q, want %q", got.errorType, tt.wantType)
			}
			if got.isRetriable != tt.wantRetriable {
				t.Errorf("isRetriable = %v, want %v", got.isRetriable, tt.wantRetriable)
			}
		})
	}
}

func TestHasNetworkErrorMarker(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"connection refused", errors.New("dial tcp: connection refused"), true},
		{"timeout", errors.New("request timeout after 30s"), true},
		{"network unreachable", errors.New("network is unreachable"), true},
		{"EOF", errors.New("unexpected EOF"), true},
		{"unrelated error", errors.New("board not found"), false},
		{"empty message", errors.New(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasNetworkErrorMarker(tt.err); got != tt.want {
				t.Errorf("hasNetworkErrorMarker(%q) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestCategorizeBulkError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantType       string
		wantRetriable  bool
		wantStatusCode int
	}{
		{
			name:           "api error carries status through",
			err:            &APIError{StatusCode: 429, Message: "slow down"},
			wantType:       "rate_limit",
			wantRetriable:  true,
			wantStatusCode: 429,
		},
		{
			name:           "server api error is retriable",
			err:            &APIError{StatusCode: 502, Message: "bad gateway"},
			wantType:       "server",
			wantRetriable:  true,
			wantStatusCode: 502,
		},
		{
			name:          "validation error",
			err:           &ValidationError{Field: "board_id", Message: "required"},
			wantType:      "validation",
			wantRetriable: false,
		},
		{
			name:          "deadline exceeded is a retriable timeout",
			err:           context.DeadlineExceeded,
			wantType:      "timeout",
			wantRetriable: true,
		},
		{
			name:          "canceled context is a retriable timeout",
			err:           context.Canceled,
			wantType:      "timeout",
			wantRetriable: true,
		},
		{
			name:          "wrapped deadline is still a timeout",
			err:           fmt.Errorf("calling API: %w", context.DeadlineExceeded),
			wantType:      "timeout",
			wantRetriable: true,
		},
		{
			name:          "network marker is retriable",
			err:           errors.New("dial tcp: connection refused"),
			wantType:      "network",
			wantRetriable: true,
		},
		{
			name:          "anything else is unknown and not retriable",
			err:           errors.New("something surprising happened"),
			wantType:      "unknown",
			wantRetriable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := categorizeBulkError(3, "item123", tt.err)

			if got.Index != 3 {
				t.Errorf("Index = %d, want 3", got.Index)
			}
			if got.ItemID != "item123" {
				t.Errorf("ItemID = %q, want item123", got.ItemID)
			}
			if got.Message != tt.err.Error() {
				t.Errorf("Message = %q, want %q", got.Message, tt.err.Error())
			}
			if got.ErrorType != tt.wantType {
				t.Errorf("ErrorType = %q, want %q", got.ErrorType, tt.wantType)
			}
			if got.IsRetriable != tt.wantRetriable {
				t.Errorf("IsRetriable = %v, want %v", got.IsRetriable, tt.wantRetriable)
			}
			if got.StatusCode != tt.wantStatusCode {
				t.Errorf("StatusCode = %d, want %d", got.StatusCode, tt.wantStatusCode)
			}
		})
	}
}
