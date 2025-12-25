// Package audit provides audit logging for MCP tool executions.
// It tracks all operations for debugging, compliance, and analytics.
package audit

import (
	"context"
	"time"
)

// =============================================================================
// Event Types
// =============================================================================

// Action represents the type of operation performed.
type Action string

const (
	ActionCreate Action = "create"
	ActionRead   Action = "read"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionExport Action = "export"
	ActionAuth   Action = "auth"
)

// Event represents a single auditable operation.
type Event struct {
	// ID is a unique identifier for this event (UUID v4).
	ID string `json:"id"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// Tool is the MCP tool name (e.g., "miro_create_sticky").
	Tool string `json:"tool"`

	// Method is the client method called (e.g., "CreateSticky").
	Method string `json:"method"`

	// UserID is the Miro user ID who performed the action.
	UserID string `json:"user_id,omitempty"`

	// UserEmail is the email of the user who performed the action.
	UserEmail string `json:"user_email,omitempty"`

	// BoardID is the target board ID, if applicable.
	BoardID string `json:"board_id,omitempty"`

	// ItemID is the target item ID, if applicable.
	ItemID string `json:"item_id,omitempty"`

	// Action categorizes the operation type.
	Action Action `json:"action"`

	// Input contains sanitized input arguments (sensitive data redacted).
	Input map[string]interface{} `json:"input,omitempty"`

	// Success indicates whether the operation completed successfully.
	Success bool `json:"success"`

	// Error contains the error message if the operation failed.
	Error string `json:"error,omitempty"`

	// DurationMs is how long the operation took in milliseconds.
	DurationMs int64 `json:"duration_ms"`

	// ItemType is the type of item affected (e.g., "sticky_note", "shape").
	ItemType string `json:"item_type,omitempty"`

	// ItemCount is the number of items affected (for bulk operations).
	ItemCount int `json:"item_count,omitempty"`
}

// =============================================================================
// Query Options
// =============================================================================

// QueryOptions specifies filters for querying audit events.
type QueryOptions struct {
	// Since returns events after this time.
	Since time.Time

	// Until returns events before this time.
	Until time.Time

	// Tool filters by tool name.
	Tool string

	// Method filters by method name.
	Method string

	// UserID filters by Miro user ID.
	UserID string

	// BoardID filters by board ID.
	BoardID string

	// Action filters by action type.
	Action Action

	// Success filters by success status (nil = both).
	Success *bool

	// Limit is the maximum number of events to return.
	Limit int

	// Offset skips this many events (for pagination).
	Offset int
}

// QueryResult contains the results of an audit log query.
type QueryResult struct {
	// Events is the list of matching events.
	Events []Event `json:"events"`

	// Total is the total number of matching events (before limit/offset).
	Total int `json:"total"`

	// HasMore indicates if there are more events after this page.
	HasMore bool `json:"has_more"`
}

// =============================================================================
// Configuration
// =============================================================================

// Config holds audit logger configuration.
type Config struct {
	// Enabled determines whether audit logging is active.
	Enabled bool

	// Path is the directory for audit log files.
	// Only used by FileLogger.
	Path string

	// RetentionDays is how long to keep audit logs.
	RetentionDays int

	// MaxSizeBytes is the maximum size of a single log file.
	// When exceeded, the logger rotates to a new file.
	MaxSizeBytes int64

	// BufferSize is the number of events to buffer before flushing.
	// 0 means flush immediately (synchronous).
	BufferSize int

	// SanitizeInput determines whether to redact sensitive input fields.
	SanitizeInput bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:       true,
		Path:          "", // In-memory if empty
		RetentionDays: 30,
		MaxSizeBytes:  100 * 1024 * 1024, // 100 MB
		BufferSize:    100,
		SanitizeInput: true,
	}
}

// =============================================================================
// Logger Interface
// =============================================================================

// Logger is the interface for audit logging implementations.
type Logger interface {
	// Log records an audit event.
	Log(ctx context.Context, event Event) error

	// Query retrieves audit events matching the specified criteria.
	Query(ctx context.Context, opts QueryOptions) (*QueryResult, error)

	// Flush ensures all buffered events are written.
	Flush(ctx context.Context) error

	// Close releases any resources held by the logger.
	Close() error
}

// =============================================================================
// Sensitive Fields
// =============================================================================

// sensitiveFields lists input field names that should be redacted.
var sensitiveFields = map[string]bool{
	"access_token":  true,
	"refresh_token": true,
	"api_key":       true,
	"password":      true,
	"secret":        true,
	"authorization": true,
}

// SanitizeInput redacts sensitive fields from input arguments.
func SanitizeInput(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return nil
	}

	sanitized := make(map[string]interface{}, len(input))
	for key, value := range input {
		if sensitiveFields[key] {
			sanitized[key] = "[REDACTED]"
		} else {
			sanitized[key] = value
		}
	}
	return sanitized
}
