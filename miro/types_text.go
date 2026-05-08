package miro

// =============================================================================
// Create Text
// =============================================================================

// CreateTextArgs contains parameters for creating a text item.
type CreateTextArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"Board ID"`
	Content  string  `json:"content" jsonschema:"Text content"`
	X        float64 `json:"x,omitempty" jsonschema:"X position"`
	Y        float64 `json:"y,omitempty" jsonschema:"Y position"`
	Width    float64 `json:"width,omitempty" jsonschema:"Text box width"`
	FontSize int     `json:"font_size,omitempty" jsonschema:"Font size (default 14)"`
	Color    string  `json:"color,omitempty" jsonschema:"Text color: 6-char hex like #1a1a1a or named (red, orange, yellow, green, blue, purple, pink, gray, white, black)"`
	ParentID string  `json:"parent_id,omitempty" jsonschema:"Frame ID"`
}

// CreateTextResult contains the created text item.
type CreateTextResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Content string `json:"content"`
	Message string `json:"message"`
}

// =============================================================================
// Update Text
// =============================================================================

// UpdateTextArgs contains parameters for updating a text item via dedicated endpoint.
type UpdateTextArgs struct {
	BoardID   string   `json:"board_id" jsonschema:"Board ID"`
	ItemID    string   `json:"item_id" jsonschema:"Text item ID to update"`
	Content   *string  `json:"content,omitempty" jsonschema:"New text content (supports basic HTML: <p>, <a>, <b>, <strong>, <i>, <em>, <u>, <s>)"`
	FontSize  *int     `json:"font_size,omitempty" jsonschema:"New font size (10-288, default 14)"`
	TextAlign *string  `json:"text_align,omitempty" jsonschema:"Text alignment: left, center, right"`
	Color     *string  `json:"color,omitempty" jsonschema:"New text color: 6-char hex like #1a1a1a or named (red, orange, yellow, green, blue, purple, pink, gray, white, black)"`
	X         *float64 `json:"x,omitempty" jsonschema:"New X position"`
	Y         *float64 `json:"y,omitempty" jsonschema:"New Y position"`
	Width     *float64 `json:"width,omitempty" jsonschema:"New width"`
	ParentID  *string  `json:"parent_id,omitempty" jsonschema:"Move to frame (empty string removes from frame)"`
}

// UpdateTextResult contains the updated text item details.
type UpdateTextResult struct {
	ID       string `json:"id"`
	Content  string `json:"content,omitempty"`
	FontSize int    `json:"font_size,omitempty"`
	Message  string `json:"message"`
}
