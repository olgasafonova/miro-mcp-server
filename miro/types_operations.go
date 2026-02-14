package miro

// =============================================================================
// Create Sticky Note
// =============================================================================

// CreateStickyArgs contains parameters for creating a sticky note.
type CreateStickyArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Content  string  `json:"content" jsonschema:"required" jsonschema_description:"Text content of the sticky note"`
	X        float64 `json:"x,omitempty" jsonschema_description:"X position (default 0)"`
	Y        float64 `json:"y,omitempty" jsonschema_description:"Y position (default 0)"`
	Color    string  `json:"color,omitempty" jsonschema_description:"Sticky color: yellow, green, blue, pink, orange, etc."`
	Width    float64 `json:"width,omitempty" jsonschema_description:"Width in pixels"`
	ParentID string  `json:"parent_id,omitempty" jsonschema_description:"Frame ID to place sticky in"`
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
// Create Shape
// =============================================================================

// CreateShapeArgs contains parameters for creating a shape.
type CreateShapeArgs struct {
	BoardID   string  `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Shape     string  `json:"shape" jsonschema:"required" jsonschema_description:"Shape type: rectangle, circle, triangle, rhombus, round_rectangle, etc."`
	Content   string  `json:"content,omitempty" jsonschema_description:"Text inside the shape"`
	X         float64 `json:"x,omitempty" jsonschema_description:"X position"`
	Y         float64 `json:"y,omitempty" jsonschema_description:"Y position"`
	Width     float64 `json:"width,omitempty" jsonschema_description:"Width in pixels (default 200)"`
	Height    float64 `json:"height,omitempty" jsonschema_description:"Height in pixels (default 200)"`
	Color     string  `json:"color,omitempty" jsonschema_description:"Fill/background color (hex like #006400)"`
	TextColor string  `json:"text_color,omitempty" jsonschema_description:"Text color (hex like #ffffff for white)"`
	ParentID  string  `json:"parent_id,omitempty" jsonschema_description:"Frame ID"`
}

// CreateShapeResult contains the created shape.
type CreateShapeResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Shape   string `json:"shape"`
	Content string `json:"content,omitempty"`
	Message string `json:"message"`
}

// =============================================================================
// Create Text
// =============================================================================

// CreateTextArgs contains parameters for creating a text item.
type CreateTextArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Content  string  `json:"content" jsonschema:"required" jsonschema_description:"Text content"`
	X        float64 `json:"x,omitempty" jsonschema_description:"X position"`
	Y        float64 `json:"y,omitempty" jsonschema_description:"Y position"`
	Width    float64 `json:"width,omitempty" jsonschema_description:"Text box width"`
	FontSize int     `json:"font_size,omitempty" jsonschema_description:"Font size (default 14)"`
	Color    string  `json:"color,omitempty" jsonschema_description:"Text color"`
	ParentID string  `json:"parent_id,omitempty" jsonschema_description:"Frame ID"`
}

// CreateTextResult contains the created text item.
type CreateTextResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Content string `json:"content"`
	Message string `json:"message"`
}

// =============================================================================
// List Connectors
// =============================================================================

// ListConnectorsArgs contains parameters for listing connectors on a board.
type ListConnectorsArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"Max connectors to return (default 50, max 100)"`
	Cursor  string `json:"cursor,omitempty" jsonschema_description:"Pagination cursor"`
}

// ConnectorSummary represents a connector in list results.
type ConnectorSummary struct {
	ID          string `json:"id"`
	StartItemID string `json:"start_item_id"`
	EndItemID   string `json:"end_item_id"`
	Style       string `json:"style,omitempty"`
	Caption     string `json:"caption,omitempty"`
}

// ListConnectorsResult contains the list of connectors.
type ListConnectorsResult struct {
	Connectors []ConnectorSummary `json:"connectors"`
	Count      int                `json:"count"`
	HasMore    bool               `json:"has_more"`
	Cursor     string             `json:"cursor,omitempty"`
	Message    string             `json:"message"`
}

// =============================================================================
// Get Connector
// =============================================================================

// GetConnectorArgs contains parameters for getting a specific connector.
type GetConnectorArgs struct {
	BoardID     string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ConnectorID string `json:"connector_id" jsonschema:"required" jsonschema_description:"Connector ID to retrieve"`
}

// GetConnectorResult contains the full connector details.
type GetConnectorResult struct {
	ID          string `json:"id"`
	StartItemID string `json:"start_item_id"`
	EndItemID   string `json:"end_item_id"`
	Style       string `json:"style,omitempty"`
	StartCap    string `json:"start_cap,omitempty"`
	EndCap      string `json:"end_cap,omitempty"`
	Caption     string `json:"caption,omitempty"`
	Color       string `json:"color,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	ModifiedAt  string `json:"modified_at,omitempty"`
	CreatedBy   string `json:"created_by,omitempty"`
	ModifiedBy  string `json:"modified_by,omitempty"`
	Message     string `json:"message"`
}

// =============================================================================
// Create Connector
// =============================================================================

// CreateConnectorArgs contains parameters for creating a connector.
type CreateConnectorArgs struct {
	BoardID     string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	StartItemID string `json:"start_item_id" jsonschema:"required" jsonschema_description:"ID of the item to connect from"`
	EndItemID   string `json:"end_item_id" jsonschema:"required" jsonschema_description:"ID of the item to connect to"`
	Style       string `json:"style,omitempty" jsonschema_description:"Connector style: straight, elbowed, curved (default elbowed)"`
	StartCap    string `json:"start_cap,omitempty" jsonschema_description:"Start arrow: none, arrow, filled_arrow, diamond, etc."`
	EndCap      string `json:"end_cap,omitempty" jsonschema_description:"End arrow: none, arrow, filled_arrow, diamond, etc."`
	Caption     string `json:"caption,omitempty" jsonschema_description:"Text label on the connector"`
}

// CreateConnectorResult contains the created connector.
type CreateConnectorResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Message string `json:"message"`
}

// =============================================================================
// Update Connector
// =============================================================================

// UpdateConnectorArgs contains parameters for updating a connector.
type UpdateConnectorArgs struct {
	BoardID     string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ConnectorID string `json:"connector_id" jsonschema:"required" jsonschema_description:"ID of the connector to update"`
	Style       string `json:"style,omitempty" jsonschema_description:"Connector style: straight, elbowed, curved"`
	StartCap    string `json:"start_cap,omitempty" jsonschema_description:"Start arrow: none, arrow, filled_arrow, diamond, etc."`
	EndCap      string `json:"end_cap,omitempty" jsonschema_description:"End arrow: none, arrow, filled_arrow, diamond, etc."`
	Caption     string `json:"caption,omitempty" jsonschema_description:"Text label on the connector"`
	Color       string `json:"color,omitempty" jsonschema_description:"Connector line color (hex)"`
}

// UpdateConnectorResult confirms connector update.
type UpdateConnectorResult struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

// =============================================================================
// Delete Connector
// =============================================================================

// DeleteConnectorArgs contains parameters for deleting a connector.
type DeleteConnectorArgs struct {
	BoardID     string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ConnectorID string `json:"connector_id" jsonschema:"required" jsonschema_description:"ID of the connector to delete"`
	DryRun      bool   `json:"dry_run,omitempty" jsonschema_description:"If true, returns preview without deleting"`
}

// DeleteConnectorResult confirms connector deletion.
type DeleteConnectorResult struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

// =============================================================================
// Create Frame
// =============================================================================

// CreateFrameArgs contains parameters for creating a frame.
type CreateFrameArgs struct {
	BoardID string  `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Title   string  `json:"title,omitempty" jsonschema_description:"Frame title"`
	X       float64 `json:"x,omitempty" jsonschema_description:"X position"`
	Y       float64 `json:"y,omitempty" jsonschema_description:"Y position"`
	Width   float64 `json:"width,omitempty" jsonschema_description:"Width (default 800)"`
	Height  float64 `json:"height,omitempty" jsonschema_description:"Height (default 600)"`
	Color   string  `json:"color,omitempty" jsonschema_description:"Background color"`
}

// CreateFrameResult contains the created frame.
type CreateFrameResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

// =============================================================================
// Create Card
// =============================================================================

// CreateCardArgs contains parameters for creating a card.
type CreateCardArgs struct {
	BoardID     string  `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Title       string  `json:"title" jsonschema:"required" jsonschema_description:"Card title"`
	Description string  `json:"description,omitempty" jsonschema_description:"Card description/body text"`
	DueDate     string  `json:"due_date,omitempty" jsonschema_description:"Due date in ISO 8601 format (e.g., 2024-12-31)"`
	X           float64 `json:"x,omitempty" jsonschema_description:"X position"`
	Y           float64 `json:"y,omitempty" jsonschema_description:"Y position"`
	Width       float64 `json:"width,omitempty" jsonschema_description:"Card width (default 320)"`
	ParentID    string  `json:"parent_id,omitempty" jsonschema_description:"Frame ID to place card in"`
}

// CreateCardResult contains the created card.
type CreateCardResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

// =============================================================================
// Create Image
// =============================================================================

// CreateImageArgs contains parameters for creating an image.
type CreateImageArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	URL      string  `json:"url" jsonschema:"required" jsonschema_description:"URL of the image to add (must be publicly accessible)"`
	Title    string  `json:"title,omitempty" jsonschema_description:"Image title/alt text"`
	X        float64 `json:"x,omitempty" jsonschema_description:"X position"`
	Y        float64 `json:"y,omitempty" jsonschema_description:"Y position"`
	Width    float64 `json:"width,omitempty" jsonschema_description:"Image width (preserves aspect ratio)"`
	ParentID string  `json:"parent_id,omitempty" jsonschema_description:"Frame ID to place image in"`
}

// CreateImageResult contains the created image.
type CreateImageResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Message string `json:"message"`
}

// =============================================================================
// Get Image
// =============================================================================

// GetImageArgs contains parameters for retrieving an image item.
type GetImageArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"required" jsonschema_description:"Image item ID"`
}

// GetImageResult contains the image details.
type GetImageResult struct {
	ID       string  `json:"id"`
	Title    string  `json:"title,omitempty"`
	ImageURL string  `json:"image_url"`
	Width    float64 `json:"width,omitempty"`
	Height   float64 `json:"height,omitempty"`
	X        float64 `json:"x,omitempty"`
	Y        float64 `json:"y,omitempty"`
	ParentID string  `json:"parent_id,omitempty"`
	Message  string  `json:"message"`
}

// =============================================================================
// Create Document
// =============================================================================

// CreateDocumentArgs contains parameters for creating a document.
type CreateDocumentArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	URL      string  `json:"url" jsonschema:"required" jsonschema_description:"URL of the document (PDF, etc.) to add"`
	Title    string  `json:"title,omitempty" jsonschema_description:"Document title"`
	X        float64 `json:"x,omitempty" jsonschema_description:"X position"`
	Y        float64 `json:"y,omitempty" jsonschema_description:"Y position"`
	Width    float64 `json:"width,omitempty" jsonschema_description:"Document preview width"`
	ParentID string  `json:"parent_id,omitempty" jsonschema_description:"Frame ID to place document in"`
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
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"required" jsonschema_description:"Document item ID"`
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
// Create Embed
// =============================================================================

// CreateEmbedArgs contains parameters for creating an embed.
type CreateEmbedArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	URL      string  `json:"url" jsonschema:"required" jsonschema_description:"URL to embed (YouTube, Vimeo, Figma, Google Docs, etc.)"`
	Mode     string  `json:"mode,omitempty" jsonschema_description:"Display mode: inline (default) or modal"`
	X        float64 `json:"x,omitempty" jsonschema_description:"X position"`
	Y        float64 `json:"y,omitempty" jsonschema_description:"Y position"`
	Width    float64 `json:"width,omitempty" jsonschema_description:"Embed width (default 400)"`
	Height   float64 `json:"height,omitempty" jsonschema_description:"Embed height (default 300)"`
	ParentID string  `json:"parent_id,omitempty" jsonschema_description:"Frame ID to place embed in"`
}

// CreateEmbedResult contains the created embed.
type CreateEmbedResult struct {
	ID       string `json:"id"`
	ItemURL  string `json:"item_url,omitempty"`
	URL      string `json:"url"`
	Provider string `json:"provider,omitempty"`
	Message  string `json:"message"`
}

// =============================================================================
// Create Sticky Grid (Composite)
// =============================================================================

// CreateStickyGridArgs contains parameters for creating multiple stickies in a grid.
type CreateStickyGridArgs struct {
	BoardID  string   `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Contents []string `json:"contents" jsonschema:"required" jsonschema_description:"Text for each sticky note"`
	Columns  int      `json:"columns,omitempty" jsonschema_description:"Number of columns in grid (default 3)"`
	Color    string   `json:"color,omitempty" jsonschema_description:"Color for all stickies: yellow, green, blue, pink, orange, etc."`
	StartX   float64  `json:"start_x,omitempty" jsonschema_description:"Starting X position (default 0)"`
	StartY   float64  `json:"start_y,omitempty" jsonschema_description:"Starting Y position (default 0)"`
	Spacing  float64  `json:"spacing,omitempty" jsonschema_description:"Space between stickies in pixels (default 220)"`
	ParentID string   `json:"parent_id,omitempty" jsonschema_description:"Frame ID to place stickies in"`
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

// =============================================================================
// List Items
// =============================================================================

// ListItemsArgs contains parameters for listing board items.
type ListItemsArgs struct {
	BoardID     string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Type        string `json:"type,omitempty" jsonschema_description:"Filter by item type: sticky_note, shape, text, connector, frame"`
	Limit       int    `json:"limit,omitempty" jsonschema_description:"Max items to return (default 50, max 100)"`
	Cursor      string `json:"cursor,omitempty" jsonschema_description:"Pagination cursor"`
	DetailLevel string `json:"detail_level,omitempty" jsonschema_description:"Response detail level: 'minimal' (default) returns basic fields, 'full' includes style, geometry, timestamps, and creator info"`
}

// ListItemsResult contains board items.
type ListItemsResult struct {
	Items   []ItemSummary `json:"items"`
	Count   int           `json:"count"`
	HasMore bool          `json:"has_more"`
	Cursor  string        `json:"cursor,omitempty"`
}

// =============================================================================
// List All Items (Paginated)
// =============================================================================

// ListAllItemsArgs extends ListItemsArgs for full pagination.
type ListAllItemsArgs struct {
	BoardID     string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Type        string `json:"type,omitempty" jsonschema_description:"Filter by item type: sticky_note, shape, text, connector, frame, card, image, document, embed"`
	MaxItems    int    `json:"max_items,omitempty" jsonschema_description:"Maximum total items to fetch across all pages (default 500, max 10000)"`
	DetailLevel string `json:"detail_level,omitempty" jsonschema_description:"Response detail level: 'minimal' (default) returns basic fields, 'full' includes style, geometry, timestamps, and creator info"`
}

// ListAllItemsResult contains all items from a board.
type ListAllItemsResult struct {
	Items      []ItemSummary `json:"items"`
	Count      int           `json:"count"`
	TotalPages int           `json:"total_pages"`
	Truncated  bool          `json:"truncated"` // True if max_items limit was reached
	Message    string        `json:"message"`
}

// =============================================================================
// Get Item
// =============================================================================

// GetItemArgs contains parameters for getting a specific item.
type GetItemArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"required" jsonschema_description:"Item ID to retrieve"`
}

// GetItemResult contains the full item details.
type GetItemResult struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Content    string  `json:"content,omitempty"`
	Title      string  `json:"title,omitempty"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width,omitempty"`
	Height     float64 `json:"height,omitempty"`
	Color      string  `json:"color,omitempty"`
	Shape      string  `json:"shape,omitempty"`
	ParentID   string  `json:"parent_id,omitempty"`
	CreatedAt  string  `json:"created_at,omitempty"`
	ModifiedAt string  `json:"modified_at,omitempty"`
	CreatedBy  string  `json:"created_by,omitempty"`
	ModifiedBy string  `json:"modified_by,omitempty"`
}

// =============================================================================
// Update Item
// =============================================================================

// UpdateItemArgs contains parameters for updating an item.
type UpdateItemArgs struct {
	BoardID  string   `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID   string   `json:"item_id" jsonschema:"required" jsonschema_description:"Item ID to update"`
	Content  *string  `json:"content,omitempty" jsonschema_description:"New content text"`
	X        *float64 `json:"x,omitempty" jsonschema_description:"New X position"`
	Y        *float64 `json:"y,omitempty" jsonschema_description:"New Y position"`
	Width    *float64 `json:"width,omitempty" jsonschema_description:"New width"`
	Height   *float64 `json:"height,omitempty" jsonschema_description:"New height"`
	Color    *string  `json:"color,omitempty" jsonschema_description:"New color"`
	ParentID *string  `json:"parent_id,omitempty" jsonschema_description:"Move to new frame"`
}

// UpdateItemResult confirms item update.
type UpdateItemResult struct {
	Success bool   `json:"success"`
	ItemID  string `json:"item_id"`
	Message string `json:"message"`
}

// =============================================================================
// Delete Item
// =============================================================================

// DeleteItemArgs contains parameters for deleting an item.
type DeleteItemArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"required" jsonschema_description:"Item ID to delete"`
	DryRun  bool   `json:"dry_run,omitempty" jsonschema_description:"If true, returns preview without deleting"`
}

// DeleteItemResult confirms item deletion.
type DeleteItemResult struct {
	Success bool   `json:"success"`
	ItemID  string `json:"item_id"`
	Message string `json:"message"`
}

// =============================================================================
// Search Board
// =============================================================================

// SearchBoardArgs contains parameters for searching items on a board.
type SearchBoardArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID to search"`
	Query   string `json:"query" jsonschema:"required" jsonschema_description:"Text to search for in item content"`
	Type    string `json:"type,omitempty" jsonschema_description:"Filter by item type: sticky_note, shape, text, frame"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"Max results (default 20, max 50)"`
}

// SearchBoardResult contains matching items.
type SearchBoardResult struct {
	Matches []ItemMatch `json:"matches"`
	Count   int         `json:"count"`
	Query   string      `json:"query"`
	Message string      `json:"message"`
}

// =============================================================================
// Bulk Operations
// =============================================================================

// BulkCreateItem defines a single item in a bulk create request.
type BulkCreateItem struct {
	Type     string  `json:"type" jsonschema:"required" jsonschema_description:"Item type: sticky_note, shape, text"`
	Content  string  `json:"content,omitempty" jsonschema_description:"Text content"`
	Shape    string  `json:"shape,omitempty" jsonschema_description:"Shape type (for shapes)"`
	X        float64 `json:"x,omitempty" jsonschema_description:"X position"`
	Y        float64 `json:"y,omitempty" jsonschema_description:"Y position"`
	Width    float64 `json:"width,omitempty" jsonschema_description:"Width"`
	Height   float64 `json:"height,omitempty" jsonschema_description:"Height"`
	Color    string  `json:"color,omitempty" jsonschema_description:"Color"`
	ParentID string  `json:"parent_id,omitempty" jsonschema_description:"Frame ID"`
}

// BulkCreateArgs contains parameters for bulk item creation.
type BulkCreateArgs struct {
	BoardID string           `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Items   []BulkCreateItem `json:"items" jsonschema:"required" jsonschema_description:"Items to create (max 20)"`
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
	ItemID   string   `json:"item_id" jsonschema:"required" jsonschema_description:"ID of the item to update"`
	Content  *string  `json:"content,omitempty" jsonschema_description:"New text content"`
	X        *float64 `json:"x,omitempty" jsonschema_description:"New X position"`
	Y        *float64 `json:"y,omitempty" jsonschema_description:"New Y position"`
	Width    *float64 `json:"width,omitempty" jsonschema_description:"New width"`
	Height   *float64 `json:"height,omitempty" jsonschema_description:"New height"`
	Color    *string  `json:"color,omitempty" jsonschema_description:"New color"`
	ParentID *string  `json:"parent_id,omitempty" jsonschema_description:"New frame ID (empty string to remove from frame)"`
}

// BulkUpdateArgs contains parameters for bulk item updates.
type BulkUpdateArgs struct {
	BoardID string           `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Items   []BulkUpdateItem `json:"items" jsonschema:"required" jsonschema_description:"Items to update (max 20)"`
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
	BoardID string   `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemIDs []string `json:"item_ids" jsonschema:"required" jsonschema_description:"IDs of items to delete (max 20)"`
	DryRun  bool     `json:"dry_run,omitempty" jsonschema_description:"If true, returns preview without deleting"`
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

// =============================================================================
// Type-Specific Update Operations
// =============================================================================

// UpdateStickyArgs contains parameters for updating a sticky note via dedicated endpoint.
type UpdateStickyArgs struct {
	BoardID  string   `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID   string   `json:"item_id" jsonschema:"required" jsonschema_description:"Sticky note ID to update"`
	Content  *string  `json:"content,omitempty" jsonschema_description:"New text content"`
	Shape    *string  `json:"shape,omitempty" jsonschema_description:"Sticky shape: square or rectangle"`
	Color    *string  `json:"color,omitempty" jsonschema_description:"Sticky color: gray, light_yellow, yellow, orange, light_green, green, dark_green, cyan, light_pink, pink, violet, red, light_blue, blue, dark_blue, black"`
	X        *float64 `json:"x,omitempty" jsonschema_description:"New X position"`
	Y        *float64 `json:"y,omitempty" jsonschema_description:"New Y position"`
	Width    *float64 `json:"width,omitempty" jsonschema_description:"New width"`
	ParentID *string  `json:"parent_id,omitempty" jsonschema_description:"Move to frame (empty string removes from frame)"`
}

// UpdateStickyResult contains the updated sticky note details.
type UpdateStickyResult struct {
	ID      string `json:"id"`
	Content string `json:"content,omitempty"`
	Shape   string `json:"shape,omitempty"`
	Color   string `json:"color,omitempty"`
	Message string `json:"message"`
}

// UpdateShapeArgs contains parameters for updating a shape via dedicated endpoint.
type UpdateShapeArgs struct {
	BoardID   string   `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID    string   `json:"item_id" jsonschema:"required" jsonschema_description:"Shape ID to update"`
	Content   *string  `json:"content,omitempty" jsonschema_description:"New text inside shape"`
	ShapeType *string  `json:"shape_type,omitempty" jsonschema_description:"New shape type: rectangle, circle, triangle, rhombus, round_rectangle, parallelogram, trapezoid, pentagon, hexagon, star, flow_chart_predefined_process, etc."`
	Color     *string  `json:"color,omitempty" jsonschema_description:"New fill color (hex like #006400)"`
	TextColor *string  `json:"text_color,omitempty" jsonschema_description:"New text color (hex like #ffffff)"`
	X         *float64 `json:"x,omitempty" jsonschema_description:"New X position"`
	Y         *float64 `json:"y,omitempty" jsonschema_description:"New Y position"`
	Width     *float64 `json:"width,omitempty" jsonschema_description:"New width"`
	Height    *float64 `json:"height,omitempty" jsonschema_description:"New height"`
	ParentID  *string  `json:"parent_id,omitempty" jsonschema_description:"Move to frame (empty string removes from frame)"`
}

// UpdateShapeResult contains the updated shape details.
type UpdateShapeResult struct {
	ID        string `json:"id"`
	ShapeType string `json:"shape_type,omitempty"`
	Content   string `json:"content,omitempty"`
	Message   string `json:"message"`
}

// UpdateTextArgs contains parameters for updating a text item via dedicated endpoint.
type UpdateTextArgs struct {
	BoardID   string   `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID    string   `json:"item_id" jsonschema:"required" jsonschema_description:"Text item ID to update"`
	Content   *string  `json:"content,omitempty" jsonschema_description:"New text content (supports basic HTML: <p>, <a>, <b>, <strong>, <i>, <em>, <u>, <s>)"`
	FontSize  *int     `json:"font_size,omitempty" jsonschema_description:"New font size (10-288, default 14)"`
	TextAlign *string  `json:"text_align,omitempty" jsonschema_description:"Text alignment: left, center, right"`
	Color     *string  `json:"color,omitempty" jsonschema_description:"New text color (hex like #1a1a1a)"`
	X         *float64 `json:"x,omitempty" jsonschema_description:"New X position"`
	Y         *float64 `json:"y,omitempty" jsonschema_description:"New Y position"`
	Width     *float64 `json:"width,omitempty" jsonschema_description:"New width"`
	ParentID  *string  `json:"parent_id,omitempty" jsonschema_description:"Move to frame (empty string removes from frame)"`
}

// UpdateTextResult contains the updated text item details.
type UpdateTextResult struct {
	ID       string `json:"id"`
	Content  string `json:"content,omitempty"`
	FontSize int    `json:"font_size,omitempty"`
	Message  string `json:"message"`
}

// UpdateCardArgs contains parameters for updating a card via dedicated endpoint.
type UpdateCardArgs struct {
	BoardID     string   `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID      string   `json:"item_id" jsonschema:"required" jsonschema_description:"Card ID to update"`
	Title       *string  `json:"title,omitempty" jsonschema_description:"New card title"`
	Description *string  `json:"description,omitempty" jsonschema_description:"New card description/body"`
	DueDate     *string  `json:"due_date,omitempty" jsonschema_description:"New due date (ISO 8601) or empty to remove"`
	X           *float64 `json:"x,omitempty" jsonschema_description:"New X position"`
	Y           *float64 `json:"y,omitempty" jsonschema_description:"New Y position"`
	Width       *float64 `json:"width,omitempty" jsonschema_description:"New width"`
	ParentID    *string  `json:"parent_id,omitempty" jsonschema_description:"Move to frame (empty string removes from frame)"`
}

// UpdateCardResult contains the updated card details.
type UpdateCardResult struct {
	ID          string `json:"id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
	Message     string `json:"message"`
}

// =============================================================================
// Update Image
// =============================================================================

// UpdateImageArgs contains parameters for updating an image via dedicated endpoint.
type UpdateImageArgs struct {
	BoardID  string   `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID   string   `json:"item_id" jsonschema:"required" jsonschema_description:"Image ID to update"`
	Title    *string  `json:"title,omitempty" jsonschema_description:"New image title/alt text"`
	URL      *string  `json:"url,omitempty" jsonschema_description:"New image URL"`
	X        *float64 `json:"x,omitempty" jsonschema_description:"New X position"`
	Y        *float64 `json:"y,omitempty" jsonschema_description:"New Y position"`
	Width    *float64 `json:"width,omitempty" jsonschema_description:"New width (preserves aspect ratio)"`
	ParentID *string  `json:"parent_id,omitempty" jsonschema_description:"Move to frame (empty string removes from frame)"`
}

// UpdateImageResult contains the updated image details.
type UpdateImageResult struct {
	ID      string `json:"id"`
	Title   string `json:"title,omitempty"`
	URL     string `json:"url,omitempty"`
	Message string `json:"message"`
}

// =============================================================================
// Update Document
// =============================================================================

// UpdateDocumentArgs contains parameters for updating a document via dedicated endpoint.
type UpdateDocumentArgs struct {
	BoardID  string   `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID   string   `json:"item_id" jsonschema:"required" jsonschema_description:"Document ID to update"`
	Title    *string  `json:"title,omitempty" jsonschema_description:"New document title"`
	URL      *string  `json:"url,omitempty" jsonschema_description:"New document URL"`
	X        *float64 `json:"x,omitempty" jsonschema_description:"New X position"`
	Y        *float64 `json:"y,omitempty" jsonschema_description:"New Y position"`
	Width    *float64 `json:"width,omitempty" jsonschema_description:"New preview width"`
	ParentID *string  `json:"parent_id,omitempty" jsonschema_description:"Move to frame (empty string removes from frame)"`
}

// UpdateDocumentResult contains the updated document details.
type UpdateDocumentResult struct {
	ID      string `json:"id"`
	Title   string `json:"title,omitempty"`
	Message string `json:"message"`
}

// =============================================================================
// Update Embed
// =============================================================================

// UpdateEmbedArgs contains parameters for updating an embed via dedicated endpoint.
type UpdateEmbedArgs struct {
	BoardID  string   `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID   string   `json:"item_id" jsonschema:"required" jsonschema_description:"Embed ID to update"`
	URL      *string  `json:"url,omitempty" jsonschema_description:"New embed URL"`
	Mode     *string  `json:"mode,omitempty" jsonschema_description:"Display mode: inline or modal"`
	X        *float64 `json:"x,omitempty" jsonschema_description:"New X position"`
	Y        *float64 `json:"y,omitempty" jsonschema_description:"New Y position"`
	Width    *float64 `json:"width,omitempty" jsonschema_description:"New embed width"`
	Height   *float64 `json:"height,omitempty" jsonschema_description:"New embed height"`
	ParentID *string  `json:"parent_id,omitempty" jsonschema_description:"Move to frame (empty string removes from frame)"`
}

// UpdateEmbedResult contains the updated embed details.
type UpdateEmbedResult struct {
	ID       string `json:"id"`
	URL      string `json:"url,omitempty"`
	Provider string `json:"provider,omitempty"`
	Message  string `json:"message"`
}
