package miro

// =============================================================================
// Get Frame
// =============================================================================

// GetFrameArgs contains parameters for getting a specific frame.
type GetFrameArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	FrameID string `json:"frame_id" jsonschema:"Frame ID to retrieve"`
}

// GetFrameResult contains the full frame details.
type GetFrameResult struct {
	ID         string  `json:"id"`
	Title      string  `json:"title,omitempty"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	Color      string  `json:"color,omitempty"`
	ChildCount int     `json:"child_count"`
	CreatedAt  string  `json:"created_at,omitempty"`
	ModifiedAt string  `json:"modified_at,omitempty"`
	CreatedBy  string  `json:"created_by,omitempty"`
	ModifiedBy string  `json:"modified_by,omitempty"`
	Message    string  `json:"message"`
}

// =============================================================================
// Update Frame
// =============================================================================

// UpdateFrameArgs contains parameters for updating a frame.
type UpdateFrameArgs struct {
	BoardID string   `json:"board_id" jsonschema:"Board ID"`
	FrameID string   `json:"frame_id" jsonschema:"Frame ID to update"`
	Title   *string  `json:"title,omitempty" jsonschema:"New frame title"`
	X       *float64 `json:"x,omitempty" jsonschema:"New X position"`
	Y       *float64 `json:"y,omitempty" jsonschema:"New Y position"`
	Width   *float64 `json:"width,omitempty" jsonschema:"New width"`
	Height  *float64 `json:"height,omitempty" jsonschema:"New height"`
	Color   *string  `json:"color,omitempty" jsonschema:"New background color"`
}

// UpdateFrameResult confirms frame update.
type UpdateFrameResult struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

// =============================================================================
// Delete Frame
// =============================================================================

// DeleteFrameArgs contains parameters for deleting a frame.
type DeleteFrameArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	FrameID string `json:"frame_id" jsonschema:"Frame ID to delete"`
	DryRun  bool   `json:"dry_run,omitempty" jsonschema:"If true, returns preview without deleting"`
}

// DeleteFrameResult confirms frame deletion.
type DeleteFrameResult struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

// =============================================================================
// Get Frame Items
// =============================================================================

// GetFrameItemsArgs contains parameters for getting items within a frame.
type GetFrameItemsArgs struct {
	BoardID     string `json:"board_id" jsonschema:"Board ID"`
	FrameID     string `json:"frame_id" jsonschema:"Frame ID to get items from"`
	Type        string `json:"type,omitempty" jsonschema:"Filter by item type: sticky_note, shape, text, card, image"`
	Limit       int    `json:"limit,omitempty" jsonschema:"Max items to return (default 50, max 100)"`
	Cursor      string `json:"cursor,omitempty" jsonschema:"Pagination cursor"`
	DetailLevel string `json:"detail_level,omitempty" jsonschema:"Response detail level: 'minimal' (default) returns basic fields, 'full' includes style, geometry, timestamps, and creator info"`
}

// GetFrameItemsResult contains items within a frame.
type GetFrameItemsResult struct {
	Items   []ItemSummary `json:"items"`
	Count   int           `json:"count"`
	HasMore bool          `json:"has_more"`
	Cursor  string        `json:"cursor,omitempty"`
	Message string        `json:"message"`
}
