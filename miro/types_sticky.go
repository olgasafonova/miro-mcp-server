package miro

// =============================================================================
// Create Sticky Note
// =============================================================================

// CreateStickyArgs contains parameters for creating a sticky note.
type CreateStickyArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"Board ID"`
	Content  string  `json:"content" jsonschema:"Text content of the sticky note"`
	X        float64 `json:"x,omitempty" jsonschema:"X position. On canvas: absolute (0 = canvas left). Inside a frame (parent_id set): relative to frame's TOP-LEFT (0 = frame's left edge), and the sticky's center is placed at this x. To center horizontally in a W-wide frame, use x = W/2."`
	Y        float64 `json:"y,omitempty" jsonschema:"Y position. On canvas: absolute. Inside a frame (parent_id set): relative to frame's TOP-LEFT, sticky center is placed at this y. Y increases downward."`
	Color    string  `json:"color,omitempty" jsonschema:"Sticky color (named only): yellow, light_yellow, light_green, green, dark_green, cyan, light_pink, pink, violet, red, light_blue, blue, dark_blue, gray, orange, black."`
	Width    float64 `json:"width,omitempty" jsonschema:"Width in pixels (default ~199; height auto-scales). Set width=160 to fit 3 stickies in a 600-tall frame."`
	ParentID string  `json:"parent_id,omitempty" jsonschema:"Frame ID to place sticky in. Coords (x, y) are then relative to the frame's TOP-LEFT corner; the sticky's CENTER is placed at (x, y). Default sticky is 199x228, so to keep it fully inside an 800x600 frame use x in [100, 700], y in [114, 486]."`
}

// CreateStickyResult contains the created sticky note.
type CreateStickyResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Content string `json:"content"`
	Color   string `json:"color"`
	Message string `json:"message"`
}

// =============================================================================
// Update Sticky Note
// =============================================================================

// UpdateStickyArgs contains parameters for updating a sticky note via dedicated endpoint.
type UpdateStickyArgs struct {
	BoardID  string   `json:"board_id" jsonschema:"Board ID"`
	ItemID   string   `json:"item_id" jsonschema:"Sticky note ID to update"`
	Content  *string  `json:"content,omitempty" jsonschema:"New text content"`
	Shape    *string  `json:"shape,omitempty" jsonschema:"Sticky shape: square or rectangle"`
	Color    *string  `json:"color,omitempty" jsonschema:"Sticky color: gray, light_yellow, yellow, orange, light_green, green, dark_green, cyan, light_pink, pink, violet, red, light_blue, blue, dark_blue, black"`
	X        *float64 `json:"x,omitempty" jsonschema:"New X position"`
	Y        *float64 `json:"y,omitempty" jsonschema:"New Y position"`
	Width    *float64 `json:"width,omitempty" jsonschema:"New width"`
	ParentID *string  `json:"parent_id,omitempty" jsonschema:"Move to frame (empty string removes from frame)"`
}

// UpdateStickyResult contains the updated sticky note details.
type UpdateStickyResult struct {
	ID      string `json:"id"`
	Content string `json:"content,omitempty"`
	Shape   string `json:"shape,omitempty"`
	Color   string `json:"color,omitempty"`
	Message string `json:"message"`
}
