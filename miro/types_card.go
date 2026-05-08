package miro

// =============================================================================
// Create Card
// =============================================================================

// CreateCardArgs contains parameters for creating a card.
type CreateCardArgs struct {
	BoardID     string  `json:"board_id" jsonschema:"Board ID"`
	Title       string  `json:"title" jsonschema:"Card title"`
	Description string  `json:"description,omitempty" jsonschema:"Card description/body text"`
	DueDate     string  `json:"due_date,omitempty" jsonschema:"Due date in ISO 8601 format (e.g., 2024-12-31)"`
	X           float64 `json:"x,omitempty" jsonschema:"X position"`
	Y           float64 `json:"y,omitempty" jsonschema:"Y position"`
	Width       float64 `json:"width,omitempty" jsonschema:"Card width (default 320)"`
	ParentID    string  `json:"parent_id,omitempty" jsonschema:"Frame ID to place card in"`
}

// CreateCardResult contains the created card.
type CreateCardResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

// =============================================================================
// Update Card
// =============================================================================

// UpdateCardArgs contains parameters for updating a card via dedicated endpoint.
type UpdateCardArgs struct {
	BoardID     string   `json:"board_id" jsonschema:"Board ID"`
	ItemID      string   `json:"item_id" jsonschema:"Card ID to update"`
	Title       *string  `json:"title,omitempty" jsonschema:"New card title"`
	Description *string  `json:"description,omitempty" jsonschema:"New card description/body"`
	DueDate     *string  `json:"due_date,omitempty" jsonschema:"New due date (ISO 8601) or empty to remove"`
	X           *float64 `json:"x,omitempty" jsonschema:"New X position"`
	Y           *float64 `json:"y,omitempty" jsonschema:"New Y position"`
	Width       *float64 `json:"width,omitempty" jsonschema:"New width"`
	ParentID    *string  `json:"parent_id,omitempty" jsonschema:"Move to frame (empty string removes from frame)"`
}

// UpdateCardResult contains the updated card details.
type UpdateCardResult struct {
	ID          string `json:"id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
	Message     string `json:"message"`
}
