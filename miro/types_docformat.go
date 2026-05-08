package miro

// =============================================================================
// Doc Format (Markdown Documents)
// =============================================================================

// CreateDocFormatArgs contains parameters for creating a doc format item from Markdown.
type CreateDocFormatArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"Board ID"`
	Content  string  `json:"content" jsonschema:"Markdown content for the document"`
	X        float64 `json:"x,omitempty" jsonschema:"X position"`
	Y        float64 `json:"y,omitempty" jsonschema:"Y position"`
	ParentID string  `json:"parent_id,omitempty" jsonschema:"Frame ID to place document in"`
}

// CreateDocFormatResult contains the created doc format item.
type CreateDocFormatResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Message string `json:"message"`
}

// GetDocFormatArgs contains parameters for getting a doc format item.
type GetDocFormatArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"Doc format item ID"`
}

// GetDocFormatResult contains the doc format item details.
type GetDocFormatResult struct {
	ID         string  `json:"id"`
	Content    string  `json:"content,omitempty"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	CreatedAt  string  `json:"created_at,omitempty"`
	ModifiedAt string  `json:"modified_at,omitempty"`
	CreatedBy  string  `json:"created_by,omitempty"`
	ModifiedBy string  `json:"modified_by,omitempty"`
	Message    string  `json:"message"`
}

// DeleteDocFormatArgs contains parameters for deleting a doc format item.
type DeleteDocFormatArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"Doc format item ID to delete"`
	DryRun  bool   `json:"dry_run,omitempty" jsonschema:"If true, returns preview without deleting"`
}

// DeleteDocFormatResult confirms doc format deletion.
type DeleteDocFormatResult struct {
	Success bool   `json:"success"`
	ItemID  string `json:"item_id"`
	Message string `json:"message"`
}

// =============================================================================
// Update Doc Format
// =============================================================================

// UpdateDocFormatArgs contains parameters for updating a doc format item's content.
// The Miro REST API does not support PATCH on docs, so this operation deletes the
// original and recreates it with the new content at the same position.
type UpdateDocFormatArgs struct {
	BoardID    string `json:"board_id" jsonschema:"Board ID"`
	ItemID     string `json:"item_id" jsonschema:"Doc format item ID to update"`
	Content    string `json:"content" jsonschema:"New Markdown content for the document"`
	OldContent string `json:"old_content,omitempty" jsonschema:"Text to find (for find-and-replace mode). If empty, replaces entire content."`
	NewContent string `json:"new_content,omitempty" jsonschema:"Replacement text (for find-and-replace mode)"`
	ReplaceAll bool   `json:"replace_all,omitempty" jsonschema:"Replace all occurrences (default: first only)"`
}

// UpdateDocFormatResult contains the updated doc format item.
type UpdateDocFormatResult struct {
	ID       string `json:"id"`
	OldID    string `json:"old_id,omitempty"`
	Content  string `json:"content,omitempty"`
	ItemURL  string `json:"item_url,omitempty"`
	Replaced int    `json:"replaced,omitempty"`
	Message  string `json:"message"`
}
