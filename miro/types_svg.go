package miro

// =============================================================================
// Canvas SVG Types
// =============================================================================

// ReadBoardSVGArgs contains parameters for rendering a board as SVG.
type ReadBoardSVGArgs struct {
	BoardID  string `json:"board_id" jsonschema:"Board ID to render"`
	FrameID  string `json:"frame_id,omitempty" jsonschema:"Render only this frame and its children. Child coordinates are relative to the frame's top-left corner. Omit for the whole board."`
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
	SVG     string  `json:"svg" jsonschema:"SVG document. Supported elements: rect (data-type=sticky|frame for those item types), circle, ellipse, text, polygon (3 points -> triangle), image (public href), line (data-start/data-end referencing element ids -> connector), and g with transform=translate(x,y). Everything else is reported as skipped."`
	OffsetX float64 `json:"offset_x,omitempty" jsonschema:"X offset added to every created item (default 0)"`
	OffsetY float64 `json:"offset_y,omitempty" jsonschema:"Y offset added to every created item (default 0)"`
}

// SVGCreatedItem records one item created from an SVG element.
type SVGCreatedItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`    // shape | text | sticky_note | frame | image | connector
	Element string `json:"element"` // source SVG element: rect | circle | ellipse | text | polygon | image | line
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

// UpdateFromSVGArgs contains parameters for applying an SVG diff to a board.
type UpdateFromSVGArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID to update items on"`
	SVG     string `json:"svg" jsonschema:"SVG document. Elements carrying data-miro-id are updated in place (geometry restated as a unit, fill -> color, text content -> content). Elements with data-miro-id and data-deleted='true' are deleted. Elements without data-miro-id are created, same dialect as miro_create_from_svg."`
}

// SVGUpdatedItem records one item updated from an SVG element.
type SVGUpdatedItem struct {
	ID      string `json:"id"`
	Element string `json:"element"`
}

// SVGFailedItem records one per-item semantic failure; the rest of the batch
// still applies.
type SVGFailedItem struct {
	ID      string `json:"id,omitempty"`
	Element string `json:"element"`
	Reason  string `json:"reason"`
}

// UpdateFromSVGResult reports the applied diff: updates, deletions, additive
// creations, per-item failures, and skipped elements.
type UpdateFromSVGResult struct {
	Updated []SVGUpdatedItem    `json:"updated"`
	Deleted []string            `json:"deleted,omitempty"`
	Created []SVGCreatedItem    `json:"created,omitempty"`
	Failed  []SVGFailedItem     `json:"failed,omitempty"`
	Skipped []SVGSkippedElement `json:"skipped,omitempty"`
	Message string              `json:"message"`
}
