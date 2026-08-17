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

// bulkErrorExpectation captures the categorization fields expected from
// categorizeBulkError for one input error.
type bulkErrorExpectation struct {
	errorType  string
	retriable  bool
	statusCode int
}

func TestCategorizeBulkError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bulkErrorExpectation
	}{
		{
			name: "api error carries status through",
			err:  &APIError{StatusCode: 429, Message: "slow down"},
			want: bulkErrorExpectation{errorType: "rate_limit", retriable: true, statusCode: 429},
		},
		{
			name: "server api error is retriable",
			err:  &APIError{StatusCode: 502, Message: "bad gateway"},
			want: bulkErrorExpectation{errorType: "server", retriable: true, statusCode: 502},
		},
		{
			name: "validation error",
			err:  &ValidationError{Field: "board_id", Message: "required"},
			want: bulkErrorExpectation{errorType: "validation", retriable: false},
		},
		{
			name: "deadline exceeded is a retriable timeout",
			err:  context.DeadlineExceeded,
			want: bulkErrorExpectation{errorType: "timeout", retriable: true},
		},
		{
			name: "canceled context is a retriable timeout",
			err:  context.Canceled,
			want: bulkErrorExpectation{errorType: "timeout", retriable: true},
		},
		{
			name: "wrapped deadline is still a timeout",
			err:  fmt.Errorf("calling API: %w", context.DeadlineExceeded),
			want: bulkErrorExpectation{errorType: "timeout", retriable: true},
		},
		{
			name: "network marker is retriable",
			err:  errors.New("dial tcp: connection refused"),
			want: bulkErrorExpectation{errorType: "network", retriable: true},
		},
		{
			name: "anything else is unknown and not retriable",
			err:  errors.New("something surprising happened"),
			want: bulkErrorExpectation{errorType: "unknown", retriable: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := categorizeBulkError(3, "item123", tt.err)
			assertBulkErrorIdentity(t, got, tt.err)
			assertBulkErrorCategory(t, got, tt.want)
		})
	}
}

// assertBulkErrorIdentity checks the fields carried through unchanged from the input.
func assertBulkErrorIdentity(t *testing.T, got BulkItemError, err error) {
	t.Helper()
	if got.Index != 3 {
		t.Errorf("Index = %d, want 3", got.Index)
	}
	if got.ItemID != "item123" {
		t.Errorf("ItemID = %q, want item123", got.ItemID)
	}
	if got.Message != err.Error() {
		t.Errorf("Message = %q, want %q", got.Message, err.Error())
	}
}

// assertBulkErrorCategory checks the categorization fields derived from the error.
func assertBulkErrorCategory(t *testing.T, got BulkItemError, want bulkErrorExpectation) {
	t.Helper()
	if got.ErrorType != want.errorType {
		t.Errorf("ErrorType = %q, want %q", got.ErrorType, want.errorType)
	}
	if got.IsRetriable != want.retriable {
		t.Errorf("IsRetriable = %v, want %v", got.IsRetriable, want.retriable)
	}
	if got.StatusCode != want.statusCode {
		t.Errorf("StatusCode = %d, want %d", got.StatusCode, want.statusCode)
	}
}
