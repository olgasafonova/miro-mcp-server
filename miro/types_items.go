package miro

import "time"

// =============================================================================
// Item Summary (for listings and search results)
// =============================================================================

// ItemSummary is a compact item representation.
type ItemSummary struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`
	Content  string  `json:"content,omitempty"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	ParentID string  `json:"parent_id,omitempty"`
}

// ItemMatch represents a search result with context.
type ItemMatch struct {
	ID      string  `json:"id"`
	Type    string  `json:"type"`
	Content string  `json:"content"`
	Snippet string  `json:"snippet,omitempty"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
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
	FillColor         string `json:"fillColor,omitempty"`         // "gray", "light_yellow", "yellow", "orange", "light_green", "green", "dark_green", "cyan", "light_pink", "pink", "violet", "red", "light_blue", "blue", "dark_blue", "black"
	TextAlign         string `json:"textAlign,omitempty"`         // "left", "center", "right"
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
	FillColor         string `json:"fillColor,omitempty"`
	FillOpacity       string `json:"fillOpacity,omitempty"` // "0.0" to "1.0"
	BorderColor       string `json:"borderColor,omitempty"`
	BorderWidth       string `json:"borderWidth,omitempty"`
	BorderStyle       string `json:"borderStyle,omitempty"` // "normal", "dashed", "dotted"
	FontFamily        string `json:"fontFamily,omitempty"`
	FontSize          string `json:"fontSize,omitempty"`
	TextAlign         string `json:"textAlign,omitempty"`
	TextAlignVertical string `json:"textAlignVertical,omitempty"`
	Color             string `json:"color,omitempty"` // Text color
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
	ItemID   string    `json:"item,omitempty"`     // ID of connected item
	Position *Position `json:"position,omitempty"` // Position if not connected to item
	SnapTo   string    `json:"snapTo,omitempty"`   // "auto", "top", "right", "bottom", "left"
}

// ConnectorStyle defines connector appearance.
type ConnectorStyle struct {
	StartStrokeCap  string `json:"startStrokeCap,omitempty"` // "none", "stealth", "arrow", "filled_arrow", "diamond", "filled_diamond", "oval", "filled_oval", "erd_one", "erd_many", "erd_one_or_many", "erd_zero_or_one", "erd_zero_or_many"
	EndStrokeCap    string `json:"endStrokeCap,omitempty"`
	StrokeStyle     string `json:"strokeStyle,omitempty"` // "normal", "dashed", "dotted"
	StrokeColor     string `json:"strokeColor,omitempty"`
	StrokeWidth     string `json:"strokeWidth,omitempty"`
	Color           string `json:"color,omitempty"` // Text color
	FontFamily      string `json:"fontFamily,omitempty"`
	FontSize        string `json:"fontSize,omitempty"`
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
	Data     FrameData  `json:"data"`
	Style    FrameStyle `json:"style,omitempty"`
	Children []string   `json:"children,omitempty"` // Child item IDs
}

// =============================================================================
// Card Types
// =============================================================================

// CardData contains card specific data.
type CardData struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	DueDate     string `json:"dueDate,omitempty"` // ISO 8601 format
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

// =============================================================================
// Image Types
// =============================================================================

// ImageData contains image specific data.
type ImageData struct {
	Title    string `json:"title,omitempty"`
	URL      string `json:"url,omitempty"`      // Source URL for create
	ImageURL string `json:"imageUrl,omitempty"` // Miro-hosted URL after create
}

// Image represents an image item.
type Image struct {
	ItemBase
	Data ImageData `json:"data"`
}

// =============================================================================
// Document Types
// =============================================================================

// DocumentData contains document specific data.
type DocumentData struct {
	Title       string `json:"title,omitempty"`
	URL         string `json:"url,omitempty"`         // Source URL for create
	DocumentURL string `json:"documentUrl,omitempty"` // Miro-hosted URL after create
}

// Document represents a document item.
type Document struct {
	ItemBase
	Data DocumentData `json:"data"`
}

// =============================================================================
// Embed Types
// =============================================================================

// EmbedData contains embed specific data.
type EmbedData struct {
	URL          string `json:"url,omitempty"`
	Mode         string `json:"mode,omitempty"`         // "inline" or "modal"
	PreviewURL   string `json:"previewUrl,omitempty"`
	ProviderName string `json:"providerName,omitempty"` // YouTube, Vimeo, etc.
}

// Embed represents an embedded content item.
type Embed struct {
	ItemBase
	Data EmbedData `json:"data"`
}
