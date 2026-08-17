package miro

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// checkErrorsField fails the test when got differs from want.
func checkErrorsField[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

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
			checkErrorsField(t, "Error()", tt.err.Error(), tt.expect)
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

			checkErrorsField(t, "IsRateLimited()", err.IsRateLimited(), tt.isRateLimited)
			checkErrorsField(t, "IsUnauthorized()", err.IsUnauthorized(), tt.isUnauthorized)
			checkErrorsField(t, "IsForbidden()", err.IsForbidden(), tt.isForbidden)
			checkErrorsField(t, "IsNotFound()", err.IsNotFound(), tt.isNotFound)
			checkErrorsField(t, "IsServerError()", err.IsServerError(), tt.isServerError)
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

			if tt.contains == "" {
				checkErrorsField(t, "Suggestion()", suggestion, "")
				return
			}
			if suggestion == "" {
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
	checkErrorsField(t, "Suggestion() mentions '30 seconds'", strings.Contains(suggestion, "30 seconds"), true)
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

	checkErrorsField(t, "StatusCode", err.StatusCode, 401)
	checkErrorsField(t, "Code", err.Code, "unauthorized")
	checkErrorsField(t, "Message", err.Message, "Invalid access token")
}

func TestParseAPIError_RateLimitWithRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{},
	}
	resp.Header.Set("Retry-After", "30")
	body := []byte(`{"message":"Rate limit exceeded"}`)

	err := ParseAPIError(resp, body)

	checkErrorsField(t, "StatusCode", err.StatusCode, 429)
	checkErrorsField(t, "RetryAfter", err.RetryAfter, 30)
}

func TestParseAPIError_PlainText(t *testing.T) {
	// Non-JSON body must NOT flow through to apiErr.Message — the falls-back
	// to http.StatusText. Stable, no leak. (HG-2 regression test.)
	resp := &http.Response{
		StatusCode: 500,
		Header:     http.Header{},
	}
	body := []byte("Internal Server Error")

	err := ParseAPIError(resp, body)

	checkErrorsField(t, "StatusCode", err.StatusCode, 500)
	checkErrorsField(t, "Message (http.StatusText fallback)", err.Message, http.StatusText(500))
}

// TestParseAPIError_HTMLBodyDoesNotLeak is the HG-2 regression test. It asserts
// that an HTML response body (typical of CDN/edge errors during Miro outages,
// or corp MITM proxy responses) is NOT propagated into the caller-facing error
// string. Before the fix, ParseAPIError would assign Message = string(body)
// unconditionally and only override it when JSON decoding succeeded with a
// non-empty message field — so HTML bodies leaked verbatim into MCP errors.
func TestParseAPIError_HTMLBodyDoesNotLeak(t *testing.T) {
	resp := &http.Response{
		StatusCode: 502,
		Header:     http.Header{},
	}
	body := []byte(`<html><head><title>502 Bad Gateway</title></head>` +
		`<body><h1>nginx/1.21.6 internal-host.miro.local</h1>` +
		`<p>request_id: abc-secret-123</p></body></html>`)

	err := ParseAPIError(resp, body)

	checkErrorsField(t, "StatusCode", err.StatusCode, 502)
	leakSentinels := []string{
		"<html>",
		"nginx",
		"internal-host.miro.local",
		"abc-secret-123",
	}
	msg := err.Error()
	for _, leak := range leakSentinels {
		if strings.Contains(msg, leak) {
			t.Errorf("HG-2 regression: error message leaked %q into caller-facing string: %q", leak, msg)
		}
	}
	// And verify the fallback message is the stable status text.
	checkErrorsField(t, "Message", err.Message, http.StatusText(http.StatusBadGateway))
}

// TestParseAPIError_EmptyJSONMessageDoesNotLeak verifies that JSON bodies
// without a usable `message` field also fall back to StatusText rather than
// echoing the original body bytes.
func TestParseAPIError_EmptyJSONMessageDoesNotLeak(t *testing.T) {
	resp := &http.Response{
		StatusCode: 400,
		Header:     http.Header{},
	}
	// JSON parses fine but message is empty — pre-fix this would have left
	// Message = string(body) verbatim because the override was conditional on
	// jsonErr.Message != "".
	body := []byte(`{"code":"bad_request","message":""}`)

	err := ParseAPIError(resp, body)

	if strings.Contains(err.Error(), `"code"`) || strings.Contains(err.Error(), `"message"`) {
		t.Errorf("HG-2 regression: error message echoed raw JSON: %q", err.Error())
	}
	checkErrorsField(t, "Message", err.Message, http.StatusText(400))
}

// =============================================================================
// Error Helper Tests
// =============================================================================

// TestErrorPredicates covers the boolean error-classification helpers with
// the same case matrix each helper had in its own test.
func TestErrorPredicates(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(error) bool
		err    error
		expect bool
	}{
		{"IsRateLimitError nil error", IsRateLimitError, nil, false},
		{"IsRateLimitError rate limit error", IsRateLimitError, &APIError{StatusCode: 429}, true},
		{"IsRateLimitError not found error", IsRateLimitError, &APIError{StatusCode: 404}, false},
		{"IsRateLimitError wrapped 429", IsRateLimitError, errors.New("API error [429]: rate limited"), true},
		{"IsRateLimitError normal error", IsRateLimitError, errors.New("connection failed"), false},
		{"IsAuthError nil error", IsAuthError, nil, false},
		{"IsAuthError 401 unauthorized", IsAuthError, &APIError{StatusCode: 401}, true},
		{"IsAuthError 403 forbidden", IsAuthError, &APIError{StatusCode: 403}, true},
		{"IsAuthError 404 not found", IsAuthError, &APIError{StatusCode: 404}, false},
		{"IsAuthError normal error", IsAuthError, errors.New("connection failed"), false},
		{"IsNotFoundError nil error", IsNotFoundError, nil, false},
		{"IsNotFoundError 404 not found", IsNotFoundError, &APIError{StatusCode: 404}, true},
		{"IsNotFoundError 401 unauthorized", IsNotFoundError, &APIError{StatusCode: 401}, false},
		{"IsNotFoundError normal error", IsNotFoundError, errors.New("connection failed"), false},
		{"IsValidationError nil error", IsValidationError, nil, false},
		{"IsValidationError validation error", IsValidationError, NewValidationError("field", "message"), true},
		{"IsValidationError API error", IsValidationError, &APIError{StatusCode: 400}, false},
		{"IsValidationError normal error", IsValidationError, errors.New("something failed"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkErrorsField(t, tt.name, tt.fn(tt.err), tt.expect)
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
			checkErrorsField(t, "GetRetryAfter()", GetRetryAfter(tt.err), tt.expect)
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
		checkErrorsField(t, "error contains 'Suggestion:'", contains(err.Error(), "Suggestion:"), true)
	})

	t.Run("regular error", func(t *testing.T) {
		origErr := errors.New("connection failed")
		err := WrapError(origErr, "CreateSticky")

		if err == nil {
			t.Fatal("WrapError should return error")
		}
		checkErrorsField(t, "error contains 'CreateSticky failed:'", contains(err.Error(), "CreateSticky failed:"), true)
	})
}

// =============================================================================
// ValidationError Tests
// =============================================================================

func TestValidationError(t *testing.T) {
	err := NewValidationError("board_id", "is required")

	checkErrorsField(t, "Field", err.Field, "board_id")
	checkErrorsField(t, "Message", err.Message, "is required")
	checkErrorsField(t, "Error()", err.Error(), "validation error: board_id - is required")
}

// =============================================================================
// Integration with HTTP Response Tests
// =============================================================================

func TestParseAPIError_RealResponse(t *testing.T) {
	// Simulate a real HTTP response using httptest.ResponseRecorder
	tests := []struct {
		name        string
		statusCode  int
		headers     map[string]string
		body        string
		expectCode  string
		expectRetry int
	}{
		{
			name:       "401 with JSON",
			statusCode: 401,
			body:       `{"code":"unauthorized","message":"Invalid token"}`,
			expectCode: "unauthorized",
		},
		{
			name:        "429 with Retry-After",
			statusCode:  429,
			headers:     map[string]string{"Retry-After": "60"},
			body:        `{"message":"Rate limit exceeded"}`,
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

			checkErrorsField(t, "StatusCode", apiErr.StatusCode, tt.statusCode)
			if tt.expectCode != "" {
				checkErrorsField(t, "Code", apiErr.Code, tt.expectCode)
			}
			if tt.expectRetry != 0 {
				checkErrorsField(t, "RetryAfter", apiErr.RetryAfter, tt.expectRetry)
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

// TestRequireIDHelpers covers the board and item ID requirement helpers.
func TestRequireIDHelpers(t *testing.T) {
	tests := []struct {
		name    string
		got     error
		wantErr error
	}{
		{"RequireBoardID empty", RequireBoardID(""), ErrBoardIDRequired},
		{"RequireBoardID valid", RequireBoardID("abc123"), nil},
		{"RequireItemID empty", RequireItemID(""), ErrItemIDRequired},
		{"RequireItemID valid", RequireItemID("item123"), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkErrorsField(t, tt.name, tt.got, tt.wantErr)
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
			if err != nil {
				checkErrorsField(t, "RequireNonEmpty() error", err.Error(), tt.errMsg)
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
		checkErrorsField(t, "error", err.Error(), "at least one item is required")
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
		checkErrorsField(t, "error", err.Error(), "at least 2 item_ids required")
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
			checkErrorsField(t, "Error()", tt.err.Error(), tt.message)
		})
	}
}
