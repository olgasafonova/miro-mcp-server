package miro

// =============================================================================
// Bulk Create Operations
// =============================================================================

// BulkCreateItem defines a single item in a bulk create request.
type BulkCreateItem struct {
	Type     string  `json:"type" jsonschema:"Item type: sticky_note, shape, text"`
	Content  string  `json:"content,omitempty" jsonschema:"Text content"`
	Shape    string  `json:"shape,omitempty" jsonschema:"Shape type (for shapes)"`
	X        float64 `json:"x,omitempty" jsonschema:"X position. On canvas: absolute. Inside a frame (parent_id set): relative to frame's TOP-LEFT, item's CENTER is placed at this x."`
	Y        float64 `json:"y,omitempty" jsonschema:"Y position. On canvas: absolute. Inside a frame (parent_id set): relative to frame's TOP-LEFT, item's CENTER is placed at this y."`
	Width    float64 `json:"width,omitempty" jsonschema:"Width"`
	Height   float64 `json:"height,omitempty" jsonschema:"Height"`
	Color    string  `json:"color,omitempty" jsonschema:"Color. For sticky_note items: Miro sticky names (yellow, light_green, dark_blue, etc.). For shape/text items: 6-char hex like #006400 or named (red, orange, yellow, green, blue, purple, pink, gray, white, black)."`
	ParentID string  `json:"parent_id,omitempty" jsonschema:"Frame ID to place item in. Coords (x, y) are then relative to the frame's TOP-LEFT corner; the item's CENTER is placed at (x, y)."`
}

// BulkCreateArgs contains parameters for bulk item creation.
type BulkCreateArgs struct {
	BoardID string           `json:"board_id" jsonschema:"Board ID"`
	Items   []BulkCreateItem `json:"items" jsonschema:"Items to create (max 20)"`
}

// BulkItemError represents a single item failure in a bulk operation.
type BulkItemError struct {
	Index       int    `json:"index"`                 // Position in the original request
	ItemID      string `json:"item_id,omitempty"`     // Item ID (for update/delete operations)
	ErrorType   string `json:"error_type"`            // Category: "rate_limit", "not_found", "validation", "server", "network"
	Message     string `json:"message"`               // Human-readable error description
	IsRetriable bool   `json:"is_retriable"`          // Whether this error can be retried
	StatusCode  int    `json:"status_code,omitempty"` // HTTP status code if applicable
}

// BulkCreateResult contains results of bulk item creation.
type BulkCreateResult struct {
	Created      int             `json:"created"`
	ItemIDs      []string        `json:"item_ids"`
	ItemURLs     []string        `json:"item_urls,omitempty"`
	Errors       []string        `json:"errors,omitempty"`
	FailedItems  []BulkItemError `json:"failed_items,omitempty"`  // Detailed failure info
	RetriableIDs []int           `json:"retriable_ids,omitempty"` // Indices that can be retried
	Message      string          `json:"message"`
}

// =============================================================================
// Bulk Update Operations
// =============================================================================

// BulkUpdateItem defines a single item update in a bulk update request.
type BulkUpdateItem struct {
	ItemID   string   `json:"item_id" jsonschema:"ID of the item to update"`
	Content  *string  `json:"content,omitempty" jsonschema:"New text content"`
	X        *float64 `json:"x,omitempty" jsonschema:"New X position"`
	Y        *float64 `json:"y,omitempty" jsonschema:"New Y position"`
	Width    *float64 `json:"width,omitempty" jsonschema:"New width"`
	Height   *float64 `json:"height,omitempty" jsonschema:"New height"`
	Color    *string  `json:"color,omitempty" jsonschema:"New color"`
	ParentID *string  `json:"parent_id,omitempty" jsonschema:"New frame ID (empty string to remove from frame)"`
}

// BulkUpdateArgs contains parameters for bulk item updates.
type BulkUpdateArgs struct {
	BoardID string           `json:"board_id" jsonschema:"Board ID"`
	Items   []BulkUpdateItem `json:"items" jsonschema:"Items to update (max 20)"`
}

// BulkUpdateResult contains results of bulk item updates.
type BulkUpdateResult struct {
	Updated      int             `json:"updated"`
	ItemIDs      []string        `json:"item_ids"`
	Errors       []string        `json:"errors,omitempty"`
	FailedItems  []BulkItemError `json:"failed_items,omitempty"`  // Detailed failure info
	RetriableIDs []string        `json:"retriable_ids,omitempty"` // Item IDs that can be retried
	Message      string          `json:"message"`
}

// =============================================================================
// Bulk Delete Operations
// =============================================================================

// BulkDeleteArgs contains parameters for bulk item deletion.
type BulkDeleteArgs struct {
	BoardID string   `json:"board_id" jsonschema:"Board ID"`
	ItemIDs []string `json:"item_ids" jsonschema:"IDs of items to delete (max 20)"`
	DryRun  bool     `json:"dry_run,omitempty" jsonschema:"If true, returns preview without deleting"`
}

// BulkDeleteResult contains results of bulk item deletion.
type BulkDeleteResult struct {
	Deleted      int             `json:"deleted"`
	ItemIDs      []string        `json:"item_ids"`
	Errors       []string        `json:"errors,omitempty"`
	FailedItems  []BulkItemError `json:"failed_items,omitempty"`  // Detailed failure info
	RetriableIDs []string        `json:"retriable_ids,omitempty"` // Item IDs that can be retried
	Message      string          `json:"message"`
}
