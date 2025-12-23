package miro

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// =============================================================================
// APIError Tests
// =============================================================================

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name   string
		err    *APIError
		expect string
	}{
		{
			name: "with code",
			err: &APIError{
				StatusCode: 401,
				Code:       "unauthorized",
				Message:    "Invalid token",
			},
			expect: "Miro API error [401 unauthorized]: Invalid token",
		},
		{
			name: "without code",
			err: &APIError{
				StatusCode: 500,
				Message:    "Internal server error",
			},
			expect: "Miro API error [500]: Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expect {
				t.Errorf("Error() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestAPIError_StatusChecks(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		isRateLimited  bool
		isUnauthorized bool
		isForbidden    bool
		isNotFound     bool
		isServerError  bool
	}{
		{"429 rate limit", 429, true, false, false, false, false},
		{"401 unauthorized", 401, false, true, false, false, false},
		{"403 forbidden", 403, false, false, true, false, false},
		{"404 not found", 404, false, false, false, true, false},
		{"500 server error", 500, false, false, false, false, true},
		{"503 server error", 503, false, false, false, false, true},
		{"200 success", 200, false, false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode}

			if got := err.IsRateLimited(); got != tt.isRateLimited {
				t.Errorf("IsRateLimited() = %v, want %v", got, tt.isRateLimited)
			}
			if got := err.IsUnauthorized(); got != tt.isUnauthorized {
				t.Errorf("IsUnauthorized() = %v, want %v", got, tt.isUnauthorized)
			}
			if got := err.IsForbidden(); got != tt.isForbidden {
				t.Errorf("IsForbidden() = %v, want %v", got, tt.isForbidden)
			}
			if got := err.IsNotFound(); got != tt.isNotFound {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.isNotFound)
			}
			if got := err.IsServerError(); got != tt.isServerError {
				t.Errorf("IsServerError() = %v, want %v", got, tt.isServerError)
			}
		})
	}
}

func TestAPIError_Suggestion(t *testing.T) {
	tests := []struct {
		statusCode int
		contains   string
	}{
		{401, "MIRO_ACCESS_TOKEN"},
		{403, "token may lack"},
		{404, "deleted"},
		{429, "Rate limit"},
		{400, "required parameters"},
		{500, "server error"},
		{503, "temporarily unavailable"},
		{200, ""}, // No suggestion for success
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.statusCode), func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode}
			suggestion := err.Suggestion()

			if tt.contains == "" && suggestion != "" {
				t.Errorf("Suggestion() = %q, want empty", suggestion)
			}
			if tt.contains != "" && suggestion == "" {
				t.Errorf("Suggestion() is empty, want to contain %q", tt.contains)
			}
		})
	}
}

func TestAPIError_Suggestion_RetryAfter(t *testing.T) {
	err := &APIError{
		StatusCode: 429,
		RetryAfter: 30,
	}

	suggestion := err.Suggestion()
	if suggestion == "" {
		t.Error("Suggestion() should not be empty for rate limit")
	}
	if !contains(suggestion, "30 seconds") {
		t.Errorf("Suggestion() = %q, should mention 30 seconds", suggestion)
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// ParseAPIError Tests
// =============================================================================

func TestParseAPIError_JSONError(t *testing.T) {
	resp := &http.Response{
		StatusCode: 401,
		Header:     http.Header{},
	}
	body := []byte(`{"code":"unauthorized","message":"Invalid access token"}`)

	err := ParseAPIError(resp, body)

	if err.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", err.StatusCode)
	}
	if err.Code != "unauthorized" {
		t.Errorf("Code = %q, want 'unauthorized'", err.Code)
	}
	if err.Message != "Invalid access token" {
		t.Errorf("Message = %q, want 'Invalid access token'", err.Message)
	}
}

func TestParseAPIError_RateLimitWithRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{},
	}
	resp.Header.Set("Retry-After", "30")
	body := []byte(`{"message":"Rate limit exceeded"}`)

	err := ParseAPIError(resp, body)

	if err.StatusCode != 429 {
		t.Errorf("StatusCode = %d, want 429", err.StatusCode)
	}
	if err.RetryAfter != 30 {
		t.Errorf("RetryAfter = %d, want 30", err.RetryAfter)
	}
}

func TestParseAPIError_PlainText(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Header:     http.Header{},
	}
	body := []byte("Internal Server Error")

	err := ParseAPIError(resp, body)

	if err.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", err.StatusCode)
	}
	if err.Message != "Internal Server Error" {
		t.Errorf("Message = %q, want 'Internal Server Error'", err.Message)
	}
}

// =============================================================================
// Error Helper Tests
// =============================================================================

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"nil error", nil, false},
		{"rate limit error", &APIError{StatusCode: 429}, true},
		{"not found error", &APIError{StatusCode: 404}, false},
		{"wrapped 429", errors.New("API error [429]: rate limited"), true},
		{"normal error", errors.New("connection failed"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRateLimitError(tt.err)
			if got != tt.expect {
				t.Errorf("IsRateLimitError() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"nil error", nil, false},
		{"401 unauthorized", &APIError{StatusCode: 401}, true},
		{"403 forbidden", &APIError{StatusCode: 403}, true},
		{"404 not found", &APIError{StatusCode: 404}, false},
		{"normal error", errors.New("connection failed"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAuthError(tt.err)
			if got != tt.expect {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"nil error", nil, false},
		{"404 not found", &APIError{StatusCode: 404}, true},
		{"401 unauthorized", &APIError{StatusCode: 401}, false},
		{"normal error", errors.New("connection failed"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFoundError(tt.err)
			if got != tt.expect {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestGetRetryAfter(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect time.Duration
	}{
		{"nil error", nil, 0},
		{"no retry-after", &APIError{StatusCode: 429}, 0},
		{"with retry-after", &APIError{StatusCode: 429, RetryAfter: 30}, 30 * time.Second},
		{"normal error", errors.New("connection failed"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRetryAfter(tt.err)
			if got != tt.expect {
				t.Errorf("GetRetryAfter() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		err := WrapError(nil, "test")
		if err != nil {
			t.Error("WrapError(nil) should return nil")
		}
	})

	t.Run("API error with suggestion", func(t *testing.T) {
		apiErr := &APIError{StatusCode: 401, Message: "Unauthorized"}
		err := WrapError(apiErr, "ListBoards")

		if err == nil {
			t.Fatal("WrapError should return error")
		}
		if !contains(err.Error(), "Suggestion:") {
			t.Errorf("Error should contain suggestion: %s", err.Error())
		}
	})

	t.Run("regular error", func(t *testing.T) {
		origErr := errors.New("connection failed")
		err := WrapError(origErr, "CreateSticky")

		if err == nil {
			t.Fatal("WrapError should return error")
		}
		if !contains(err.Error(), "CreateSticky failed:") {
			t.Errorf("Error should contain operation: %s", err.Error())
		}
	})
}

// =============================================================================
// ValidationError Tests
// =============================================================================

func TestValidationError(t *testing.T) {
	err := NewValidationError("board_id", "is required")

	if err.Field != "board_id" {
		t.Errorf("Field = %q, want 'board_id'", err.Field)
	}
	if err.Message != "is required" {
		t.Errorf("Message = %q, want 'is required'", err.Message)
	}

	expected := "validation error: board_id - is required"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"nil error", nil, false},
		{"validation error", NewValidationError("field", "message"), true},
		{"API error", &APIError{StatusCode: 400}, false},
		{"normal error", errors.New("something failed"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidationError(tt.err)
			if got != tt.expect {
				t.Errorf("IsValidationError() = %v, want %v", got, tt.expect)
			}
		})
	}
}

// =============================================================================
// Integration with HTTP Response Tests
// =============================================================================

func TestParseAPIError_RealResponse(t *testing.T) {
	// Simulate a real HTTP response using httptest.ResponseRecorder
	tests := []struct {
		name         string
		statusCode   int
		headers      map[string]string
		body         string
		expectCode   string
		expectRetry  int
	}{
		{
			name:       "401 with JSON",
			statusCode: 401,
			body:       `{"code":"unauthorized","message":"Invalid token"}`,
			expectCode: "unauthorized",
		},
		{
			name:       "429 with Retry-After",
			statusCode: 429,
			headers:    map[string]string{"Retry-After": "60"},
			body:       `{"message":"Rate limit exceeded"}`,
			expectRetry: 60,
		},
		{
			name:       "500 plain text",
			statusCode: 500,
			body:       "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rec.Code = tt.statusCode
			for k, v := range tt.headers {
				rec.Header().Set(k, v)
			}
			rec.Body.WriteString(tt.body)

			resp := rec.Result()
			apiErr := ParseAPIError(resp, rec.Body.Bytes())

			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.statusCode)
			}
			if tt.expectCode != "" && apiErr.Code != tt.expectCode {
				t.Errorf("Code = %q, want %q", apiErr.Code, tt.expectCode)
			}
			if tt.expectRetry != 0 && apiErr.RetryAfter != tt.expectRetry {
				t.Errorf("RetryAfter = %d, want %d", apiErr.RetryAfter, tt.expectRetry)
			}
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkParseAPIError_JSON(b *testing.B) {
	resp := &http.Response{
		StatusCode: 401,
		Header:     http.Header{},
	}
	body := []byte(`{"code":"unauthorized","message":"Invalid access token"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseAPIError(resp, body)
	}
}

func BenchmarkIsRateLimitError(b *testing.B) {
	err := &APIError{StatusCode: 429}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsRateLimitError(err)
	}
}

// =============================================================================
// Validation Helper Tests
// =============================================================================

func TestRequireBoardID(t *testing.T) {
	tests := []struct {
		name    string
		boardID string
		wantErr error
	}{
		{"empty", "", ErrBoardIDRequired},
		{"valid", "abc123", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RequireBoardID(tt.boardID)
			if err != tt.wantErr {
				t.Errorf("RequireBoardID() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequireItemID(t *testing.T) {
	tests := []struct {
		name   string
		itemID string
		wantErr error
	}{
		{"empty", "", ErrItemIDRequired},
		{"valid", "item123", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RequireItemID(tt.itemID)
			if err != tt.wantErr {
				t.Errorf("RequireItemID() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequireNonEmpty(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
		errMsg  string
	}{
		{"empty value", "board_id", "", true, "board_id is required"},
		{"non-empty value", "board_id", "abc123", false, ""},
		{"whitespace only", "name", "   ", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RequireNonEmpty(tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireNonEmpty() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("RequireNonEmpty() error = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestRequireNonEmptySlice(t *testing.T) {
	t.Run("empty string slice", func(t *testing.T) {
		err := RequireNonEmptySlice("item", []string{})
		if err == nil {
			t.Error("RequireNonEmptySlice() expected error for empty slice")
		}
		if err.Error() != "at least one item is required" {
			t.Errorf("error = %q, want %q", err.Error(), "at least one item is required")
		}
	})

	t.Run("non-empty string slice", func(t *testing.T) {
		err := RequireNonEmptySlice("item", []string{"a", "b"})
		if err != nil {
			t.Errorf("RequireNonEmptySlice() unexpected error: %v", err)
		}
	})

	t.Run("empty int slice", func(t *testing.T) {
		err := RequireNonEmptySlice("number", []int{})
		if err == nil {
			t.Error("RequireNonEmptySlice() expected error for empty slice")
		}
	})
}

func TestRequireMinItems(t *testing.T) {
	t.Run("below minimum", func(t *testing.T) {
		err := RequireMinItems("item_ids", []string{"a"}, 2)
		if err == nil {
			t.Error("RequireMinItems() expected error")
		}
		if err.Error() != "at least 2 item_ids required" {
			t.Errorf("error = %q, want %q", err.Error(), "at least 2 item_ids required")
		}
	})

	t.Run("at minimum", func(t *testing.T) {
		err := RequireMinItems("item_ids", []string{"a", "b"}, 2)
		if err != nil {
			t.Errorf("RequireMinItems() unexpected error: %v", err)
		}
	})

	t.Run("above minimum", func(t *testing.T) {
		err := RequireMinItems("item_ids", []string{"a", "b", "c"}, 2)
		if err != nil {
			t.Errorf("RequireMinItems() unexpected error: %v", err)
		}
	})
}

func TestPredefinedErrors(t *testing.T) {
	// Verify all predefined errors have correct messages
	tests := []struct {
		err     error
		message string
	}{
		{ErrBoardIDRequired, "board_id is required"},
		{ErrItemIDRequired, "item_id is required"},
		{ErrNameRequired, "name is required"},
		{ErrTitleRequired, "title is required"},
		{ErrContentRequired, "content is required"},
		{ErrQueryRequired, "query is required"},
		{ErrTagIDRequired, "tag_id is required"},
		{ErrFrameIDRequired, "frame_id is required"},
		{ErrGroupIDRequired, "group_id is required"},
		{ErrConnectorRequired, "connector_id is required"},
		{ErrNodeIDRequired, "node_id is required"},
		{ErrMemberIDRequired, "member_id is required"},
		{ErrEmailRequired, "email is required"},
		{ErrURLRequired, "url is required"},
		{ErrShapeRequired, "shape is required"},
		{ErrDiagramRequired, "diagram is required"},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			if tt.err.Error() != tt.message {
				t.Errorf("Error() = %q, want %q", tt.err.Error(), tt.message)
			}
		})
	}
}
