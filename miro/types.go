// Package miro provides a client for the Miro REST API.
package miro

import "time"

// =============================================================================
// Board Types
// =============================================================================

// Board represents a Miro board.
type Board struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	ModifiedAt  time.Time `json:"modifiedAt,omitempty"`
	ViewLink    string    `json:"viewLink,omitempty"`
	Owner       *User     `json:"owner,omitempty"`
	Team        *Team     `json:"team,omitempty"`
}

// User represents a Miro user.
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Team represents a Miro team.
type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// =============================================================================
// Item Types - Base
// =============================================================================

// Position defines x,y coordinates on the board.
type Position struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Origin string  `json:"origin,omitempty"` // "center" (default)
}

// Geometry defines width and height of an item.
type Geometry struct {
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
}

// ItemBase contains common fields for all board items.
type ItemBase struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Position   *Position  `json:"position,omitempty"`
	Geometry   *Geometry  `json:"geometry,omitempty"`
	CreatedAt  time.Time  `json:"createdAt,omitempty"`
	ModifiedAt time.Time  `json:"modifiedAt,omitempty"`
	CreatedBy  *User      `json:"createdBy,omitempty"`
	ModifiedBy *User      `json:"modifiedBy,omitempty"`
	ParentID   string     `json:"parentId,omitempty"` // Frame or group ID
}

// =============================================================================
// Sticky Note Types
// =============================================================================

// StickyNoteData contains sticky note specific data.
type StickyNoteData struct {
	Content string `json:"content"`
	Shape   string `json:"shape,omitempty"` // "square", "rectangle"
}

// StickyNoteStyle defines sticky note appearance.
type StickyNoteStyle struct {
	FillColor       string `json:"fillColor,omitempty"`       // "gray", "light_yellow", "yellow", "orange", "light_green", "green", "dark_green", "cyan", "light_pink", "pink", "violet", "red", "light_blue", "blue", "dark_blue", "black"
	TextAlign       string `json:"textAlign,omitempty"`       // "left", "center", "right"
	TextAlignVertical string `json:"textAlignVertical,omitempty"` // "top", "middle", "bottom"
}

// StickyNote represents a sticky note item.
type StickyNote struct {
	ItemBase
	Data  StickyNoteData  `json:"data"`
	Style StickyNoteStyle `json:"style,omitempty"`
}

// =============================================================================
// Shape Types
// =============================================================================

// ShapeData contains shape specific data.
type ShapeData struct {
	Content string `json:"content,omitempty"`
	Shape   string `json:"shape"` // "rectangle", "round_rectangle", "circle", "triangle", "rhombus", "parallelogram", "trapezoid", "pentagon", "hexagon", "octagon", "wedge_round_rectangle_callout", "star", "flow_chart_predefined_process", etc.
}

// ShapeStyle defines shape appearance.
type ShapeStyle struct {
	FillColor   string `json:"fillColor,omitempty"`
	FillOpacity string `json:"fillOpacity,omitempty"` // "0.0" to "1.0"
	BorderColor string `json:"borderColor,omitempty"`
	BorderWidth string `json:"borderWidth,omitempty"`
	BorderStyle string `json:"borderStyle,omitempty"` // "normal", "dashed", "dotted"
	FontFamily  string `json:"fontFamily,omitempty"`
	FontSize    string `json:"fontSize,omitempty"`
	TextAlign   string `json:"textAlign,omitempty"`
	TextAlignVertical string `json:"textAlignVertical,omitempty"`
	Color       string `json:"color,omitempty"` // Text color
}

// Shape represents a shape item.
type Shape struct {
	ItemBase
	Data  ShapeData  `json:"data"`
	Style ShapeStyle `json:"style,omitempty"`
}

// =============================================================================
// Text Types
// =============================================================================

// TextData contains text item data.
type TextData struct {
	Content string `json:"content"`
}

// TextStyle defines text appearance.
type TextStyle struct {
	FillColor   string `json:"fillColor,omitempty"`
	FillOpacity string `json:"fillOpacity,omitempty"`
	FontFamily  string `json:"fontFamily,omitempty"`
	FontSize    string `json:"fontSize,omitempty"`
	TextAlign   string `json:"textAlign,omitempty"`
	Color       string `json:"color,omitempty"`
}

// TextItem represents a text item.
type TextItem struct {
	ItemBase
	Data  TextData  `json:"data"`
	Style TextStyle `json:"style,omitempty"`
}

// =============================================================================
// Connector Types
// =============================================================================

// ConnectorEndpoint defines one end of a connector.
type ConnectorEndpoint struct {
	ItemID   string   `json:"item,omitempty"`     // ID of connected item
	Position *Position `json:"position,omitempty"` // Position if not connected to item
	SnapTo   string   `json:"snapTo,omitempty"`   // "auto", "top", "right", "bottom", "left"
}

// ConnectorStyle defines connector appearance.
type ConnectorStyle struct {
	StartStrokeCap string `json:"startStrokeCap,omitempty"` // "none", "stealth", "arrow", "filled_arrow", "diamond", "filled_diamond", "oval", "filled_oval", "erd_one", "erd_many", "erd_one_or_many", "erd_zero_or_one", "erd_zero_or_many"
	EndStrokeCap   string `json:"endStrokeCap,omitempty"`
	StrokeStyle    string `json:"strokeStyle,omitempty"`    // "normal", "dashed", "dotted"
	StrokeColor    string `json:"strokeColor,omitempty"`
	StrokeWidth    string `json:"strokeWidth,omitempty"`
	Color          string `json:"color,omitempty"` // Text color
	FontFamily     string `json:"fontFamily,omitempty"`
	FontSize       string `json:"fontSize,omitempty"`
	TextOrientation string `json:"textOrientation,omitempty"` // "horizontal", "aligned"
}

// Connector represents a connector between items.
type Connector struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"` // Always "connector"
	StartItem  ConnectorEndpoint `json:"startItem"`
	EndItem    ConnectorEndpoint `json:"endItem"`
	Shape      string            `json:"shape,omitempty"` // "straight", "elbowed", "curved"
	Style      ConnectorStyle    `json:"style,omitempty"`
	Captions   []Caption         `json:"captions,omitempty"`
	CreatedAt  time.Time         `json:"createdAt,omitempty"`
	ModifiedAt time.Time         `json:"modifiedAt,omitempty"`
}

// Caption is text attached to a connector.
type Caption struct {
	Content  string `json:"content"`
	Position string `json:"position,omitempty"` // "0.0" to "1.0" (position along connector)
}

// =============================================================================
// Frame Types
// =============================================================================

// FrameData contains frame specific data.
type FrameData struct {
	Title  string `json:"title,omitempty"`
	Format string `json:"format,omitempty"` // "custom", "letter", "a4", etc.
	Type   string `json:"type,omitempty"`   // "freeform", "heap", "grid", "flow_chart", "kanban", "timeline"
}

// FrameStyle defines frame appearance.
type FrameStyle struct {
	FillColor string `json:"fillColor,omitempty"`
}

// Frame represents a frame container.
type Frame struct {
	ItemBase
	Data     FrameData   `json:"data"`
	Style    FrameStyle  `json:"style,omitempty"`
	Children []string    `json:"children,omitempty"` // Child item IDs
}

// =============================================================================
// API Request/Response Types
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
	Boards []BoardSummary `json:"boards"`
	Count  int            `json:"count"`
	HasMore bool          `json:"has_more"`
	Offset string         `json:"offset,omitempty"`
}

// BoardSummary is a compact board representation for listings.
type BoardSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ViewLink    string `json:"view_link"`
	TeamName    string `json:"team_name,omitempty"`
}

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
// Item CRUD Args/Results
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

// CreateShapeArgs contains parameters for creating a shape.
type CreateShapeArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Shape    string  `json:"shape" jsonschema:"required" jsonschema_description:"Shape type: rectangle, circle, triangle, rhombus, round_rectangle, etc."`
	Content  string  `json:"content,omitempty" jsonschema_description:"Text inside the shape"`
	X        float64 `json:"x,omitempty" jsonschema_description:"X position"`
	Y        float64 `json:"y,omitempty" jsonschema_description:"Y position"`
	Width    float64 `json:"width,omitempty" jsonschema_description:"Width in pixels (default 200)"`
	Height   float64 `json:"height,omitempty" jsonschema_description:"Height in pixels (default 200)"`
	Color    string  `json:"color,omitempty" jsonschema_description:"Fill color (hex or named)"`
	ParentID string  `json:"parent_id,omitempty" jsonschema_description:"Frame ID"`
}

// CreateShapeResult contains the created shape.
type CreateShapeResult struct {
	ID      string `json:"id"`
	Shape   string `json:"shape"`
	Content string `json:"content,omitempty"`
	Message string `json:"message"`
}

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

// ItemSummary is a compact item representation.
type ItemSummary struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`
	Content  string  `json:"content,omitempty"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	ParentID string  `json:"parent_id,omitempty"`
}

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
// Frame Operations
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
// Get Item Operations
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
// Search Operations
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

// ItemMatch represents a search result with context.
type ItemMatch struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Snippet string `json:"snippet,omitempty"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
}

// =============================================================================
// Card Types
// =============================================================================

// CardData contains card specific data.
type CardData struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	DueDate     string `json:"dueDate,omitempty"`  // ISO 8601 format
	Assignee    *User  `json:"assignee,omitempty"`
}

// CardStyle defines card appearance.
type CardStyle struct {
	CardTheme string `json:"cardTheme,omitempty"` // "#1a1a2e", "#2d3748", etc.
}

// Card represents a card item.
type Card struct {
	ItemBase
	Data  CardData  `json:"data"`
	Style CardStyle `json:"style,omitempty"`
}

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
// Image Types
// =============================================================================

// ImageData contains image specific data.
type ImageData struct {
	Title    string `json:"title,omitempty"`
	URL      string `json:"url,omitempty"` // Source URL for create
	ImageURL string `json:"imageUrl,omitempty"` // Miro-hosted URL after create
}

// Image represents an image item.
type Image struct {
	ItemBase
	Data ImageData `json:"data"`
}

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
// Document Types
// =============================================================================

// DocumentData contains document specific data.
type DocumentData struct {
	Title       string `json:"title,omitempty"`
	URL         string `json:"url,omitempty"` // Source URL for create
	DocumentURL string `json:"documentUrl,omitempty"` // Miro-hosted URL after create
}

// Document represents a document item.
type Document struct {
	ItemBase
	Data DocumentData `json:"data"`
}

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
// Embed Types
// =============================================================================

// EmbedData contains embed specific data.
type EmbedData struct {
	URL         string `json:"url,omitempty"`
	Mode        string `json:"mode,omitempty"`        // "inline" or "modal"
	PreviewURL  string `json:"previewUrl,omitempty"`
	ProviderName string `json:"providerName,omitempty"` // YouTube, Vimeo, etc.
}

// Embed represents an embedded content item.
type Embed struct {
	ItemBase
	Data EmbedData `json:"data"`
}

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
// Tag Types
// =============================================================================

// Tag represents a tag that can be attached to items.
type Tag struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	FillColor string `json:"fillColor,omitempty"`
}

// CreateTagArgs contains parameters for creating a tag.
type CreateTagArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Title   string `json:"title" jsonschema:"required" jsonschema_description:"Tag text (e.g., 'Urgent', 'Done', 'Review')"`
	Color   string `json:"color,omitempty" jsonschema_description:"Tag color: red, magenta, violet, blue, cyan, green, yellow, orange, gray"`
}

// CreateTagResult contains the created tag.
type CreateTagResult struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Color   string `json:"color"`
	Message string `json:"message"`
}

// ListTagsArgs contains parameters for listing tags on a board.
type ListTagsArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"Max tags to return (default 50)"`
}

// ListTagsResult contains the list of tags.
type ListTagsResult struct {
	Tags    []Tag  `json:"tags"`
	Count   int    `json:"count"`
	Message string `json:"message"`
}

// AttachTagArgs contains parameters for attaching a tag to an item.
type AttachTagArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"required" jsonschema_description:"ID of the item to tag (sticky note only)"`
	TagID   string `json:"tag_id" jsonschema:"required" jsonschema_description:"ID of the tag to attach"`
}

// AttachTagResult confirms tag attachment.
type AttachTagResult struct {
	Success bool   `json:"success"`
	ItemID  string `json:"item_id"`
	TagID   string `json:"tag_id"`
	Message string `json:"message"`
}

// DetachTagArgs contains parameters for removing a tag from an item.
type DetachTagArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"required" jsonschema_description:"ID of the item to untag"`
	TagID   string `json:"tag_id" jsonschema:"required" jsonschema_description:"ID of the tag to remove"`
}

// DetachTagResult confirms tag removal.
type DetachTagResult struct {
	Success bool   `json:"success"`
	ItemID  string `json:"item_id"`
	TagID   string `json:"tag_id"`
	Message string `json:"message"`
}

// GetItemTagsArgs contains parameters for listing tags on an item.
type GetItemTagsArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"required" jsonschema_description:"ID of the item"`
}

// GetItemTagsResult contains tags attached to an item.
type GetItemTagsResult struct {
	Tags    []Tag  `json:"tags"`
	Count   int    `json:"count"`
	ItemID  string `json:"item_id"`
	Message string `json:"message"`
}

// =============================================================================
// Pagination Types
// =============================================================================

// ListAllItemsArgs extends ListItemsArgs for full pagination.
type ListAllItemsArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Type    string `json:"type,omitempty" jsonschema_description:"Filter by item type: sticky_note, shape, text, connector, frame, card, image, document, embed"`
	MaxItems int   `json:"max_items,omitempty" jsonschema_description:"Maximum total items to fetch across all pages (default 500, max 10000)"`
}

// ListAllItemsResult contains all items from a board.
type ListAllItemsResult struct {
	Items       []ItemSummary `json:"items"`
	Count       int           `json:"count"`
	TotalPages  int           `json:"total_pages"`
	Truncated   bool          `json:"truncated"` // True if max_items limit was reached
	Message     string        `json:"message"`
}

// =============================================================================
// API Response Wrappers
// =============================================================================

// APIError represents a Miro API error response.
type APIError struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// PaginatedResponse wraps paginated API responses.
type PaginatedResponse struct {
	Data   []interface{} `json:"data"`
	Total  int           `json:"total,omitempty"`
	Size   int           `json:"size,omitempty"`
	Offset string        `json:"offset,omitempty"`
	Limit  int           `json:"limit,omitempty"`
	Cursor string        `json:"cursor,omitempty"`
}
