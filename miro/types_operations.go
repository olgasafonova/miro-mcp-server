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
	Title   string `json:"title"`
	URL     string `json:"url"`
	Message string `json:"message"`
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
	Title   string `json:"title"`
	Message string `json:"message"`
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
	Created int      `json:"created"`
	ItemIDs []string `json:"item_ids"`
	Rows    int      `json:"rows"`
	Columns int      `json:"columns"`
	Message string   `json:"message"`
}

// =============================================================================
// List Items
// =============================================================================

// ListItemsArgs contains parameters for listing board items.
type ListItemsArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Type    string `json:"type,omitempty" jsonschema_description:"Filter by item type: sticky_note, shape, text, connector, frame"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"Max items to return (default 50, max 100)"`
	Cursor  string `json:"cursor,omitempty" jsonschema_description:"Pagination cursor"`
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
	BoardID  string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Type     string `json:"type,omitempty" jsonschema_description:"Filter by item type: sticky_note, shape, text, connector, frame, card, image, document, embed"`
	MaxItems int    `json:"max_items,omitempty" jsonschema_description:"Maximum total items to fetch across all pages (default 500, max 10000)"`
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

// BulkCreateResult contains results of bulk item creation.
type BulkCreateResult struct {
	Created int      `json:"created"`
	ItemIDs []string `json:"item_ids"`
	Errors  []string `json:"errors,omitempty"`
	Message string   `json:"message"`
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
	Updated int      `json:"updated"`
	ItemIDs []string `json:"item_ids"`
	Errors  []string `json:"errors,omitempty"`
	Message string   `json:"message"`
}

// =============================================================================
// Bulk Delete Operations
// =============================================================================

// BulkDeleteArgs contains parameters for bulk item deletion.
type BulkDeleteArgs struct {
	BoardID string   `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemIDs []string `json:"item_ids" jsonschema:"required" jsonschema_description:"IDs of items to delete (max 20)"`
}

// BulkDeleteResult contains results of bulk item deletion.
type BulkDeleteResult struct {
	Deleted int      `json:"deleted"`
	ItemIDs []string `json:"item_ids"`
	Errors  []string `json:"errors,omitempty"`
	Message string   `json:"message"`
}
