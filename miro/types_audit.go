package miro

import "time"

// =============================================================================
// Audit Log Query Types
// =============================================================================

// GetAuditLogArgs specifies the filter options for querying audit logs.
type GetAuditLogArgs struct {
	// Since returns events after this time (ISO 8601 format)
	Since string `json:"since,omitempty" jsonschema_description:"Return events after this time (ISO 8601, e.g., 2024-01-01T00:00:00Z)"`

	// Until returns events before this time (ISO 8601 format)
	Until string `json:"until,omitempty" jsonschema_description:"Return events before this time (ISO 8601, e.g., 2024-01-02T00:00:00Z)"`

	// Tool filters by tool name (e.g., "miro_create_sticky")
	Tool string `json:"tool,omitempty" jsonschema_description:"Filter by tool name (e.g., miro_create_sticky)"`

	// BoardID filters by board ID
	BoardID string `json:"board_id,omitempty" jsonschema_description:"Filter by board ID"`

	// Action filters by action type: create, read, update, delete, export, auth
	Action string `json:"action,omitempty" jsonschema_description:"Filter by action type: create, read, update, delete, export, auth"`

	// Success filters by success status
	Success *bool `json:"success,omitempty" jsonschema_description:"Filter by success status (true/false)"`

	// Limit is the maximum number of events to return (default 50, max 500)
	Limit int `json:"limit,omitempty" jsonschema_description:"Maximum events to return (default 50, max 500)"`
}

// AuditLogEvent represents a single audit event in the query result.
type AuditLogEvent struct {
	// ID is the unique event identifier
	ID string `json:"id"`

	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Tool is the MCP tool name
	Tool string `json:"tool"`

	// Action is the operation type
	Action string `json:"action"`

	// BoardID is the affected board
	BoardID string `json:"board_id,omitempty"`

	// ItemID is the affected item
	ItemID string `json:"item_id,omitempty"`

	// Success indicates if the operation succeeded
	Success bool `json:"success"`

	// Error is the error message if failed
	Error string `json:"error,omitempty"`

	// DurationMs is the operation duration in milliseconds
	DurationMs int64 `json:"duration_ms"`
}

// GetAuditLogResult contains the audit log query results.
type GetAuditLogResult struct {
	// Events is the list of matching audit events
	Events []AuditLogEvent `json:"events"`

	// Total is the total count of matching events
	Total int `json:"total"`

	// HasMore indicates if there are more events beyond the limit
	HasMore bool `json:"has_more"`

	// Message is a summary message
	Message string `json:"message"`
}

// AuditStatsResult contains audit log statistics.
type AuditStatsResult struct {
	// TotalEvents is the total number of logged events
	TotalEvents int `json:"total_events"`

	// SuccessCount is the number of successful operations
	SuccessCount int `json:"success_count"`

	// ErrorCount is the number of failed operations
	ErrorCount int `json:"error_count"`

	// ByTool shows counts per tool
	ByTool map[string]int `json:"by_tool"`

	// ByAction shows counts per action type
	ByAction map[string]int `json:"by_action"`

	// AvgDurationMs is the average operation duration
	AvgDurationMs float64 `json:"avg_duration_ms"`

	// Message is a summary message
	Message string `json:"message"`
}
