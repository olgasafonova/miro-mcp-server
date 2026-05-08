package miro

// =============================================================================
// Create Sticky Grid (Composite)
// =============================================================================

// CreateStickyGridArgs contains parameters for creating multiple stickies in a grid.
type CreateStickyGridArgs struct {
	BoardID  string   `json:"board_id" jsonschema:"Board ID"`
	Contents []string `json:"contents" jsonschema:"Text for each sticky note"`
	Columns  int      `json:"columns,omitempty" jsonschema:"Number of columns in grid (default 3)"`
	Color    string   `json:"color,omitempty" jsonschema:"Color for all stickies: yellow, green, blue, pink, orange, etc."`
	StartX   float64  `json:"start_x,omitempty" jsonschema:"Starting X position (default 0)"`
	StartY   float64  `json:"start_y,omitempty" jsonschema:"Starting Y position (default 0)"`
	Spacing  float64  `json:"spacing,omitempty" jsonschema:"Space between stickies in pixels (default 220)"`
	ParentID string   `json:"parent_id,omitempty" jsonschema:"Frame ID to place stickies in"`
}

// CreateStickyGridResult contains the result of creating a sticky grid.
type CreateStickyGridResult struct {
	Created  int      `json:"created"`
	ItemIDs  []string `json:"item_ids"`
	ItemURLs []string `json:"item_urls,omitempty"`
	Rows     int      `json:"rows"`
	Columns  int      `json:"columns"`
	Message  string   `json:"message"`
}
