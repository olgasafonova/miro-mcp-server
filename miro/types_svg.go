package miro

// =============================================================================
// Canvas SVG Types
// =============================================================================

// ReadBoardSVGArgs contains parameters for rendering a board as SVG.
type ReadBoardSVGArgs struct {
	BoardID  string `json:"board_id" jsonschema:"Board ID to render"`
	MaxItems int    `json:"max_items,omitempty" jsonschema:"Maximum items to include (default 500, max 2000)"`
}

// ReadBoardSVGResult contains the rendered SVG document.
type ReadBoardSVGResult struct {
	SVG       string `json:"svg"`
	ItemCount int    `json:"item_count"`
	Skipped   int    `json:"skipped"` // items with no visual mapping (embeds, images without geometry, ...)
	Truncated bool   `json:"truncated"`
	Message   string `json:"message"`
}

// CreateFromSVGArgs contains parameters for creating board items from SVG.
type CreateFromSVGArgs struct {
	BoardID string  `json:"board_id" jsonschema:"Board ID to create items on"`
	SVG     string  `json:"svg" jsonschema:"SVG document. Supported elements: rect, circle, ellipse, text, and g with transform=translate(x,y). Everything else is reported as skipped."`
	OffsetX float64 `json:"offset_x,omitempty" jsonschema:"X offset added to every created item (default 0)"`
	OffsetY float64 `json:"offset_y,omitempty" jsonschema:"Y offset added to every created item (default 0)"`
}

// SVGCreatedItem records one item created from an SVG element.
type SVGCreatedItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`    // shape | text
	Element string `json:"element"` // source SVG element: rect | circle | ellipse | text
}

// SVGSkippedElement records one SVG element that was not converted.
type SVGSkippedElement struct {
	Element string `json:"element"`
	Reason  string `json:"reason"`
}

// CreateFromSVGResult reports what was created and what was skipped.
type CreateFromSVGResult struct {
	Created []SVGCreatedItem    `json:"created"`
	Skipped []SVGSkippedElement `json:"skipped,omitempty"`
	Count   int                 `json:"count"`
	Message string              `json:"message"`
}
