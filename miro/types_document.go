package miro

// =============================================================================
// Create Document
// =============================================================================

// CreateDocumentArgs contains parameters for creating a document.
type CreateDocumentArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"Board ID"`
	URL      string  `json:"url" jsonschema:"URL of the document (PDF, etc.) to add"`
	Title    string  `json:"title,omitempty" jsonschema:"Document title"`
	X        float64 `json:"x,omitempty" jsonschema:"X position"`
	Y        float64 `json:"y,omitempty" jsonschema:"Y position"`
	Width    float64 `json:"width,omitempty" jsonschema:"Document preview width"`
	ParentID string  `json:"parent_id,omitempty" jsonschema:"Frame ID to place document in"`
}

// CreateDocumentResult contains the created document.
type CreateDocumentResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

// =============================================================================
// Get Document
// =============================================================================

// GetDocumentArgs contains parameters for retrieving a document item.
type GetDocumentArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"Document item ID"`
}

// GetDocumentResult contains the document details.
type GetDocumentResult struct {
	ID          string  `json:"id"`
	Title       string  `json:"title,omitempty"`
	DocumentURL string  `json:"document_url,omitempty"`
	Width       float64 `json:"width,omitempty"`
	Height      float64 `json:"height,omitempty"`
	X           float64 `json:"x,omitempty"`
	Y           float64 `json:"y,omitempty"`
	ParentID    string  `json:"parent_id,omitempty"`
	Message     string  `json:"message"`
}

// =============================================================================
// Update Document
// =============================================================================

// UpdateDocumentArgs contains parameters for updating a document via dedicated endpoint.
type UpdateDocumentArgs struct {
	BoardID  string   `json:"board_id" jsonschema:"Board ID"`
	ItemID   string   `json:"item_id" jsonschema:"Document ID to update"`
	Title    *string  `json:"title,omitempty" jsonschema:"New document title"`
	URL      *string  `json:"url,omitempty" jsonschema:"New document URL"`
	X        *float64 `json:"x,omitempty" jsonschema:"New X position"`
	Y        *float64 `json:"y,omitempty" jsonschema:"New Y position"`
	Width    *float64 `json:"width,omitempty" jsonschema:"New preview width"`
	ParentID *string  `json:"parent_id,omitempty" jsonschema:"Move to frame (empty string removes from frame)"`
}

// UpdateDocumentResult contains the updated document details.
type UpdateDocumentResult struct {
	ID      string `json:"id"`
	Title   string `json:"title,omitempty"`
	Message string `json:"message"`
}
