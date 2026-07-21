package miro

// =============================================================================
// Code Widget Types (v2-experimental)
// =============================================================================

// =============================================================================
// Create Code Widget
// =============================================================================

// CreateCodeWidgetArgs contains parameters for creating a code widget.
type CreateCodeWidgetArgs struct {
	BoardID            string   `json:"board_id" jsonschema:"Board ID"`
	Code               string   `json:"code" jsonschema:"The code content of the widget (max 6000 characters)"`
	Language           string   `json:"language,omitempty" jsonschema:"Programming language for syntax highlighting (e.g. javascript, go, python)"`
	Title              string   `json:"title,omitempty" jsonschema:"Title of the code widget (max 100 characters)"`
	LineNumbersVisible *bool    `json:"line_numbers_visible,omitempty" jsonschema:"Show line numbers (API default applies when omitted)"`
	X                  float64  `json:"x,omitempty" jsonschema:"X position on the board (center origin)"`
	Y                  float64  `json:"y,omitempty" jsonschema:"Y position on the board (center origin)"`
	Width              *float64 `json:"width,omitempty" jsonschema:"Width in pixels"`
	Height             *float64 `json:"height,omitempty" jsonschema:"Height in pixels"`
	ParentID           string   `json:"parent_id,omitempty" jsonschema:"ID of a parent frame to attach the widget to"`
}

// CreateCodeWidgetResult contains the created code widget.
type CreateCodeWidgetResult struct {
	ID       string `json:"id"`
	ItemURL  string `json:"item_url,omitempty"`
	Title    string `json:"title,omitempty"`
	Language string `json:"language,omitempty"`
	Message  string `json:"message"`
}

// =============================================================================
// Get Code Widget
// =============================================================================

// GetCodeWidgetArgs contains parameters for getting a code widget.
type GetCodeWidgetArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"Code widget item ID to retrieve"`
}

// GetCodeWidgetResult contains the code widget details.
type GetCodeWidgetResult struct {
	ID                 string  `json:"id"`
	Code               string  `json:"code"`
	Language           string  `json:"language,omitempty"`
	Title              string  `json:"title,omitempty"`
	LineNumbersVisible bool    `json:"line_numbers_visible"`
	X                  float64 `json:"x"`
	Y                  float64 `json:"y"`
	Width              float64 `json:"width,omitempty"`
	Height             float64 `json:"height,omitempty"`
	ParentID           string  `json:"parent_id,omitempty"`
	CreatedAt          string  `json:"created_at,omitempty"`
	ModifiedAt         string  `json:"modified_at,omitempty"`
	Message            string  `json:"message"`
}

// =============================================================================
// List Code Widgets
// =============================================================================

// ListCodeWidgetsArgs contains parameters for listing code widgets.
type ListCodeWidgetsArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Max widgets to return (default 50, max 100)"`
	Cursor  string `json:"cursor,omitempty" jsonschema:"Pagination cursor"`
}

// CodeWidgetSummary is a brief summary of a code widget. Code content is
// truncated to a preview; use miro_get_code_widget for the full source.
type CodeWidgetSummary struct {
	ID          string `json:"id"`
	Title       string `json:"title,omitempty"`
	Language    string `json:"language,omitempty"`
	CodePreview string `json:"code_preview,omitempty"`
	ParentID    string `json:"parent_id,omitempty"`
}

// ListCodeWidgetsResult contains the list of code widgets.
type ListCodeWidgetsResult struct {
	Widgets []CodeWidgetSummary `json:"widgets"`
	Count   int                 `json:"count"`
	HasMore bool                `json:"has_more"`
	Cursor  string              `json:"cursor,omitempty"`
	Message string              `json:"message"`
}

// =============================================================================
// Update Code Widget
// =============================================================================

// UpdateCodeWidgetArgs contains parameters for updating a code widget's
// content, appearance, or parent frame. Position moves use MoveCodeWidget.
type UpdateCodeWidgetArgs struct {
	BoardID            string   `json:"board_id" jsonschema:"Board ID"`
	ItemID             string   `json:"item_id" jsonschema:"Code widget item ID to update"`
	Code               string   `json:"code,omitempty" jsonschema:"New code content (max 6000 characters; omit to keep current)"`
	Language           string   `json:"language,omitempty" jsonschema:"New programming language (omit to keep current)"`
	Title              string   `json:"title,omitempty" jsonschema:"New title (max 100 characters; omit to keep current)"`
	LineNumbersVisible *bool    `json:"line_numbers_visible,omitempty" jsonschema:"Show line numbers (omit to keep current)"`
	Width              *float64 `json:"width,omitempty" jsonschema:"New width in pixels (omit to keep current)"`
	Height             *float64 `json:"height,omitempty" jsonschema:"New height in pixels (omit to keep current)"`
	ParentID           string   `json:"parent_id,omitempty" jsonschema:"ID of a parent frame to attach the widget to (omit to keep current)"`
}

// UpdateCodeWidgetResult confirms the update.
type UpdateCodeWidgetResult struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// =============================================================================
// Move Code Widget
// =============================================================================

// MoveCodeWidgetArgs contains parameters for moving a code widget to a new
// position via the dedicated position endpoint.
type MoveCodeWidgetArgs struct {
	BoardID string  `json:"board_id" jsonschema:"Board ID"`
	ItemID  string  `json:"item_id" jsonschema:"Code widget item ID to move"`
	X       float64 `json:"x" jsonschema:"New X position on the board (center origin)"`
	Y       float64 `json:"y" jsonschema:"New Y position on the board (center origin)"`
}

// MoveCodeWidgetResult confirms the move.
type MoveCodeWidgetResult struct {
	ID      string  `json:"id"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Message string  `json:"message"`
}

// =============================================================================
// Delete Code Widget
// =============================================================================

// DeleteCodeWidgetArgs contains parameters for deleting a code widget.
type DeleteCodeWidgetArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"Code widget item ID to delete"`
	DryRun  bool   `json:"dry_run,omitempty" jsonschema:"If true, returns preview without deleting"`
}

// DeleteCodeWidgetResult confirms the deletion.
type DeleteCodeWidgetResult struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Message string `json:"message"`
}
