package miro

// =============================================================================
// Board Summary Types
// =============================================================================

// BoardSummary is a compact board representation for listings.
type BoardSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ViewLink    string `json:"view_link"`
	TeamName    string `json:"team_name,omitempty"`
}

// =============================================================================
// List Boards
// =============================================================================

// ListBoardsArgs contains parameters for listing boards.
type ListBoardsArgs struct {
	TeamID string `json:"team_id,omitempty" jsonschema_description:"Filter by team ID"`
	Query  string `json:"query,omitempty" jsonschema_description:"Search boards by name"`
	Limit  int    `json:"limit,omitempty" jsonschema_description:"Max boards to return (default 20, max 50)"`
	Offset string `json:"offset,omitempty" jsonschema_description:"Pagination cursor"`
}

// ListBoardsResult contains the list of boards.
type ListBoardsResult struct {
	Boards  []BoardSummary `json:"boards"`
	Count   int            `json:"count"`
	HasMore bool           `json:"has_more"`
	Offset  string         `json:"offset,omitempty"`
}

// =============================================================================
// Get Board
// =============================================================================

// GetBoardArgs contains parameters for getting a board.
type GetBoardArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID to retrieve"`
}

// GetBoardResult contains the board details.
type GetBoardResult struct {
	Board
	ItemCount int `json:"item_count,omitempty"`
}

// =============================================================================
// Create Board
// =============================================================================

// CreateBoardArgs contains parameters for creating a new board.
type CreateBoardArgs struct {
	Name        string `json:"name" jsonschema:"required" jsonschema_description:"Name for the new board"`
	Description string `json:"description,omitempty" jsonschema_description:"Board description"`
	TeamID      string `json:"team_id,omitempty" jsonschema_description:"Team ID to create board in"`
}

// CreateBoardResult contains the created board details.
type CreateBoardResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ViewLink string `json:"view_link"`
	Message  string `json:"message"`
}

// =============================================================================
// Copy Board
// =============================================================================

// CopyBoardArgs contains parameters for copying a board.
type CopyBoardArgs struct {
	BoardID     string `json:"board_id" jsonschema:"required" jsonschema_description:"ID of the board to copy"`
	Name        string `json:"name,omitempty" jsonschema_description:"Name for the copy (defaults to 'Copy of {original}')"`
	Description string `json:"description,omitempty" jsonschema_description:"Description for the copy"`
	TeamID      string `json:"team_id,omitempty" jsonschema_description:"Team ID to copy board to"`
}

// CopyBoardResult contains the copied board details.
type CopyBoardResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ViewLink string `json:"view_link"`
	Message  string `json:"message"`
}

// =============================================================================
// Delete Board
// =============================================================================

// DeleteBoardArgs contains parameters for deleting a board.
type DeleteBoardArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"ID of the board to delete"`
	DryRun  bool   `json:"dry_run,omitempty" jsonschema_description:"If true, returns preview without deleting"`
}

// DeleteBoardResult confirms board deletion.
type DeleteBoardResult struct {
	Success bool   `json:"success"`
	BoardID string `json:"board_id"`
	Message string `json:"message"`
}

// =============================================================================
// Find Board by Name
// =============================================================================

// FindBoardByNameArgs contains parameters for finding a board by name.
type FindBoardByNameArgs struct {
	Name string `json:"name" jsonschema:"required" jsonschema_description:"Board name to search for (case-insensitive, supports partial matching)"`
}

// FindBoardByNameResult contains the found board.
type FindBoardByNameResult struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ViewLink    string `json:"view_link"`
	Message     string `json:"message"`
}

// =============================================================================
// Board Summary (Composite)
// =============================================================================

// GetBoardSummaryArgs contains parameters for getting a board summary.
type GetBoardSummaryArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID to summarize"`
}

// GetBoardSummaryResult contains the board summary with item counts.
type GetBoardSummaryResult struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	ViewLink    string         `json:"view_link"`
	ItemCounts  map[string]int `json:"item_counts"` // {"sticky_note": 15, "shape": 8, ...}
	TotalItems  int            `json:"total_items"`
	RecentItems []ItemSummary  `json:"recent_items,omitempty"` // Last 5 modified
	Message     string         `json:"message"`
}

// =============================================================================
// Update Board
// =============================================================================

// UpdateBoardArgs contains parameters for updating a board.
type UpdateBoardArgs struct {
	BoardID     string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID to update"`
	Name        string `json:"name,omitempty" jsonschema_description:"New name for the board"`
	Description string `json:"description,omitempty" jsonschema_description:"New description for the board"`
}

// UpdateBoardResult contains the updated board details.
type UpdateBoardResult struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ViewLink    string `json:"view_link"`
	Message     string `json:"message"`
}

// =============================================================================
// Board Content (Rich Data Export for AI Analysis)
// =============================================================================

// GetBoardContentArgs contains parameters for getting comprehensive board content.
type GetBoardContentArgs struct {
	BoardID           string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID to analyze"`
	IncludeConnectors bool   `json:"include_connectors,omitempty" jsonschema_description:"Include connector relationships (default true)"`
	IncludeTags       bool   `json:"include_tags,omitempty" jsonschema_description:"Include tag data and usage (default true)"`
	MaxItems          int    `json:"max_items,omitempty" jsonschema_description:"Maximum items to fetch (default 500, max 2000)"`
}

// FrameContext represents a frame with its child items.
type FrameContext struct {
	ID       string        `json:"id"`
	Title    string        `json:"title,omitempty"`
	X        float64       `json:"x"`
	Y        float64       `json:"y"`
	Width    float64       `json:"width"`
	Height   float64       `json:"height"`
	Children []ItemSummary `json:"children,omitempty"`
}

// ConnectorContext represents a connection between items.
type ConnectorContext struct {
	ID            string `json:"id"`
	StartItemID   string `json:"start_item_id"`
	StartItemType string `json:"start_item_type,omitempty"`
	EndItemID     string `json:"end_item_id"`
	EndItemType   string `json:"end_item_type,omitempty"`
	Caption       string `json:"caption,omitempty"`
}

// TagContext represents a tag with usage information.
type TagContext struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Color      string   `json:"color,omitempty"`
	UsageCount int      `json:"usage_count"`
	ItemIDs    []string `json:"item_ids,omitempty"`
}

// ItemsByType groups items by their type for easier analysis.
type ItemsByType struct {
	StickyNotes []ItemSummary `json:"sticky_notes,omitempty"`
	Shapes      []ItemSummary `json:"shapes,omitempty"`
	Text        []ItemSummary `json:"text,omitempty"`
	Cards       []ItemSummary `json:"cards,omitempty"`
	Images      []ItemSummary `json:"images,omitempty"`
	Documents   []ItemSummary `json:"documents,omitempty"`
	Embeds      []ItemSummary `json:"embeds,omitempty"`
	Other       []ItemSummary `json:"other,omitempty"`
}

// ContentSummary provides extracted text content for analysis.
type ContentSummary struct {
	AllText       []string `json:"all_text"`
	UniqueEntries int      `json:"unique_entries"`
	TotalChars    int      `json:"total_chars"`
}

// GetBoardContentResult contains comprehensive board data for AI analysis.
type GetBoardContentResult struct {
	// Board metadata
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ViewLink    string `json:"view_link"`
	CreatedAt   string `json:"created_at,omitempty"`
	ModifiedAt  string `json:"modified_at,omitempty"`

	// Item statistics
	ItemCounts map[string]int `json:"item_counts"`
	TotalItems int            `json:"total_items"`

	// Structured content
	ItemsByType ItemsByType        `json:"items_by_type"`
	Frames      []FrameContext     `json:"frames,omitempty"`
	Connectors  []ConnectorContext `json:"connectors,omitempty"`
	Tags        []TagContext       `json:"tags,omitempty"`

	// Content for text analysis
	ContentSummary ContentSummary `json:"content_summary"`

	// Status
	Truncated bool   `json:"truncated"`
	Message   string `json:"message"`
}
