package miro

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// Error Types
// =============================================================================

// APIError represents a structured error from the Miro API.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	Type       string `json:"type,omitempty"`
	Status     int    `json:"status,omitempty"`
	Context    string `json:"context,omitempty"`
	RetryAfter int    `json:"-"` // Seconds until retry (for rate limits)
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("Miro API error [%d %s]: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("Miro API error [%d]: %s", e.StatusCode, e.Message)
}

// IsRateLimited returns true if this is a rate limit error.
func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// IsUnauthorized returns true if this is an authentication error.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsForbidden returns true if this is a permission error.
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == http.StatusForbidden
}

// IsNotFound returns true if the resource was not found.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsServerError returns true if this is a server-side error.
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500
}

// feedbackURL is the pre-filled issue link shown on setup/auth errors.
const feedbackURL = "https://github.com/olgasafonova/miro-mcp-server/issues/new?template=bug_report.yml"

// Suggestion returns actionable guidance for resolving the error.
func (e *APIError) Suggestion() string {
	switch e.StatusCode {
	case http.StatusUnauthorized:
		return "Check that MIRO_ACCESS_TOKEN is set and valid. Get a token at https://miro.com/app/settings/user-profile/apps — Still stuck? " + feedbackURL
	case http.StatusForbidden:
		return "Your token may lack the required scopes. Check board sharing permissions or regenerate your token with correct scopes. Still stuck? " + feedbackURL
	case http.StatusNotFound:
		return "The board or item may have been deleted, or the ID may be incorrect. Verify the ID exists."
	case http.StatusTooManyRequests:
		if e.RetryAfter > 0 {
			return fmt.Sprintf("Rate limit exceeded. Wait %d seconds before retrying.", e.RetryAfter)
		}
		return "Rate limit exceeded. Wait a moment before retrying, or reduce request frequency."
	case http.StatusBadRequest:
		return "Check that all required parameters are provided and valid."
	case http.StatusConflict:
		return "The operation conflicts with the current state. The resource may already exist."
	case 413: // Request Entity Too Large
		return "Request payload is too large. Reduce content size or split into multiple requests."
	case http.StatusInternalServerError:
		return "Miro API server error. Try again later."
	case http.StatusServiceUnavailable:
		return "Miro API is temporarily unavailable. Try again in a few minutes."
	default:
		return ""
	}
}

// =============================================================================
// Error Helpers
// =============================================================================

// ParseAPIError parses an HTTP response into a structured APIError.
func ParseAPIError(resp *http.Response, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Message:    string(body),
	}

	// Try to parse as JSON error
	var jsonErr struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
		Status  int    `json:"status"`
		Context string `json:"context"`
	}
	if err := json.Unmarshal(body, &jsonErr); err == nil {
		if jsonErr.Message != "" {
			apiErr.Message = jsonErr.Message
		}
		apiErr.Code = jsonErr.Code
		apiErr.Type = jsonErr.Type
		apiErr.Context = jsonErr.Context
	}

	// Parse Retry-After header for rate limits
	if resp.StatusCode == http.StatusTooManyRequests {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				apiErr.RetryAfter = seconds
			}
		}
	}

	return apiErr
}

// WrapError wraps an error with additional context.
func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		// Add suggestion for API errors
		if suggestion := apiErr.Suggestion(); suggestion != "" {
			return fmt.Errorf("%s failed: %w. Suggestion: %s", operation, err, suggestion)
		}
	}

	return fmt.Errorf("%s failed: %w", operation, err)
}

// IsRateLimitError checks if an error is a rate limit error.
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsRateLimited()
	}
	// Fallback to string matching for wrapped errors
	return strings.Contains(err.Error(), "429") || strings.Contains(strings.ToLower(err.Error()), "rate limit")
}

// IsAuthError checks if an error is an authentication/authorization error.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsUnauthorized() || apiErr.IsForbidden()
	}
	return false
}

// IsNotFoundError checks if an error is a not-found error.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsNotFound()
	}
	return false
}

// GetRetryAfter returns the retry-after duration if this is a rate limit error.
func GetRetryAfter(err error) time.Duration {
	if err == nil {
		return 0
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
		return time.Duration(apiErr.RetryAfter) * time.Second
	}
	return 0
}

// =============================================================================
// Validation Errors
// =============================================================================

// ValidationError represents an input validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// IsValidationError checks if an error is a validation error.
func IsValidationError(err error) bool {
	var validErr *ValidationError
	return errors.As(err, &validErr)
}

// =============================================================================
// Predefined Validation Errors
// =============================================================================

// Common validation errors as variables for consistent error messages.
// These use simple messages matching existing error strings for compatibility.
var (
	ErrBoardIDRequired   = errors.New("board_id is required")
	ErrItemIDRequired    = errors.New("item_id is required")
	ErrNameRequired      = errors.New("name is required")
	ErrTitleRequired     = errors.New("title is required")
	ErrContentRequired   = errors.New("content is required")
	ErrQueryRequired     = errors.New("query is required")
	ErrTagIDRequired     = errors.New("tag_id is required")
	ErrFrameIDRequired   = errors.New("frame_id is required")
	ErrGroupIDRequired   = errors.New("group_id is required")
	ErrConnectorRequired = errors.New("connector_id is required")
	ErrNodeIDRequired    = errors.New("node_id is required")
	ErrMemberIDRequired  = errors.New("member_id is required")
	ErrEmailRequired     = errors.New("email is required")
	ErrURLRequired       = errors.New("url is required")
	ErrShapeRequired     = errors.New("shape is required")
	ErrDiagramRequired   = errors.New("diagram is required")
)

// =============================================================================
// Validation Helpers
// =============================================================================

// RequireBoardID validates that a board ID is not empty.
func RequireBoardID(boardID string) error {
	if boardID == "" {
		return ErrBoardIDRequired
	}
	return nil
}

// RequireItemID validates that an item ID is not empty.
func RequireItemID(itemID string) error {
	if itemID == "" {
		return ErrItemIDRequired
	}
	return nil
}

// RequireNonEmpty validates that a string field is not empty.
// Returns an error with the format "{field} is required".
func RequireNonEmpty(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

// RequireNonEmptySlice validates that a slice has at least one element.
func RequireNonEmptySlice[T any](field string, slice []T) error {
	if len(slice) == 0 {
		return fmt.Errorf("at least one %s is required", field)
	}
	return nil
}

// RequireMinItems validates that a slice has at least n elements.
func RequireMinItems[T any](field string, slice []T, min int) error {
	if len(slice) < min {
		return fmt.Errorf("at least %d %s required", min, field)
	}
	return nil
}
