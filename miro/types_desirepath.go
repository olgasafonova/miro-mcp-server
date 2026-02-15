package miro

// =============================================================================
// Desire Path Report Types
// =============================================================================

// GetDesirePathReportArgs specifies options for querying the desire path report.
type GetDesirePathReportArgs struct {
	// Tool filters events by tool name (e.g., "miro_get_board")
	Tool string `json:"tool,omitempty" jsonschema_description:"Filter by tool name (e.g., miro_get_board)"`

	// Rule filters events by normalizer rule (e.g., "url_to_id")
	Rule string `json:"rule,omitempty" jsonschema_description:"Filter by normalizer rule: url_to_id, camel_to_snake, string_to_numeric, whitespace, boolean_coercion"`

	// Limit is the maximum number of recent events to include (default 20, max 100)
	Limit int `json:"limit,omitempty" jsonschema_description:"Maximum recent events to return (default 20, max 100)"`
}

// DesirePathEvent represents a single normalization event in the response.
type DesirePathEvent struct {
	// Timestamp is when the normalization occurred
	Timestamp string `json:"timestamp"`

	// Tool is the MCP tool name
	Tool string `json:"tool"`

	// Parameter is the argument name that was normalized
	Parameter string `json:"parameter"`

	// Rule is the normalizer that fired
	Rule string `json:"rule"`

	// RawValue is what the agent sent
	RawValue string `json:"raw_value"`

	// NormalizedTo is what we corrected it to
	NormalizedTo string `json:"normalized_to"`
}

// DesirePathPattern groups identical normalization patterns.
type DesirePathPattern struct {
	// Rule is the normalizer that fired
	Rule string `json:"rule"`

	// Tool is the MCP tool name
	Tool string `json:"tool"`

	// Parameter is the argument name
	Parameter string `json:"parameter"`

	// Example shows a sample transformation
	Example string `json:"example"`

	// Count is how many times this pattern occurred
	Count int `json:"count"`
}

// GetDesirePathReportResult contains the desire path report.
type GetDesirePathReportResult struct {
	// TotalNormalizations is the total number of normalizations recorded
	TotalNormalizations int `json:"total_normalizations"`

	// ByRule shows counts per normalizer rule
	ByRule map[string]int `json:"by_rule"`

	// ByTool shows counts per tool
	ByTool map[string]int `json:"by_tool"`

	// ByParam shows counts per parameter
	ByParam map[string]int `json:"by_param"`

	// TopPatterns shows the most common normalization patterns
	TopPatterns []DesirePathPattern `json:"top_patterns"`

	// RecentEvents shows the most recent normalizations
	RecentEvents []DesirePathEvent `json:"recent_events"`

	// Message is a summary
	Message string `json:"message"`
}
