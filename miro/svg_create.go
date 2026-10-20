package miro

import (
	"context"
	"fmt"
	"strings"
)

// =============================================================================
// SVG -> Board (local parsing, creates items via the existing typed methods)
// =============================================================================
//
// Parses a constrained SVG subset and creates matching board items:
//   rect            -> shape "rectangle"   (rx>0 -> "round_rectangle")
//   rect data-type="sticky" -> sticky note (data-content -> text)
//   rect data-type="frame"  -> frame (data-title -> title)
//   circle/ellipse  -> shape "circle"
//   text            -> text item
//   polygon (3 pts) -> shape "triangle" (bounding box)
//   image href=URL  -> image item
//   line data-start/data-end -> connector between referenced element ids
//   g transform=translate(x,y) -> offset applied to children (nesting ok)
// Everything else (path, multi-point polygon, ...) is reported per element
// in Skipped rather than silently dropped. SVG y-down matches Miro y-down;
// SVG rect x/y are top-left while Miro positions are centers, so the
// converter recenters.

// maxSVGDocumentBytes bounds the accepted SVG source.
const maxSVGDocumentBytes = 1 << 20 // 1 MiB

// maxSVGCreateElements bounds how many items one call may create.
const maxSVGCreateElements = 200

// validateCreateFromSVGArgs checks the request bounds before any parsing.
func validateCreateFromSVGArgs(args CreateFromSVGArgs) error {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return err
	}
	if strings.TrimSpace(args.SVG) == "" {
		return fmt.Errorf("svg is required")
	}
	if len(args.SVG) > maxSVGDocumentBytes {
		return fmt.Errorf("svg exceeds %d bytes (got %d)", maxSVGDocumentBytes, len(args.SVG))
	}
	return nil
}

// svgShapeArgs maps a parsed element to CreateShapeArgs.
func svgShapeArgs(boardID string, el svgElement, off svgOffset) CreateShapeArgs {
	shape := "rectangle"
	switch {
	case el.name == "circle" || el.name == "ellipse":
		shape = "circle"
	case el.name == "polygon":
		shape = "triangle"
	case el.rounded:
		shape = "round_rectangle"
	}
	args := CreateShapeArgs{
		BoardID: boardID,
		Shape:   shape,
		X:       el.x + off.dx,
		Y:       el.y + off.dy,
		Width:   el.w,
		Height:  el.h,
	}
	if el.fill != "" && el.fill != "none" {
		args.Color = el.fill
	}
	return args
}

// svgStickyColor passes only named sticky colors through; the sticky API
// rejects hex fills.
func svgStickyColor(fill string) string {
	unnamed := fill == "" || fill == "none"
	if unnamed || strings.HasPrefix(fill, "#") {
		return ""
	}
	return fill
}

// createSVGSticky creates a sticky note from a data-type="sticky" rect.
func (c *Client) createSVGSticky(ctx context.Context, boardID string, el svgElement, off svgOffset) (string, error) {
	res, err := c.CreateSticky(ctx, CreateStickyArgs{
		BoardID: boardID, Content: el.text, Color: svgStickyColor(el.fill),
		X: el.x + off.dx, Y: el.y + off.dy, Width: el.w,
	})
	return res.ID, err
}

// createSVGFrame creates a frame from a data-type="frame" rect.
func (c *Client) createSVGFrame(ctx context.Context, boardID string, el svgElement, off svgOffset) (string, error) {
	res, err := c.CreateFrame(ctx, CreateFrameArgs{
		BoardID: boardID, Title: el.title,
		X: el.x + off.dx, Y: el.y + off.dy, Width: el.w, Height: el.h,
	})
	return res.ID, err
}

// createSVGImage creates an image item from an <image> element.
func (c *Client) createSVGImage(ctx context.Context, boardID string, el svgElement, off svgOffset) (string, error) {
	res, err := c.CreateImage(ctx, CreateImageArgs{
		BoardID: boardID, URL: el.href, Title: el.title,
		X: el.x + off.dx, Y: el.y + off.dy, Width: el.w,
	})
	return res.ID, err
}

// createSVGText creates a text item from a <text> element.
func (c *Client) createSVGText(ctx context.Context, boardID string, el svgElement, off svgOffset) (string, error) {
	res, err := c.CreateText(ctx, CreateTextArgs{
		BoardID: boardID, Content: el.text,
		X: el.x + off.dx, Y: el.y + off.dy,
	})
	return res.ID, err
}

// createSVGShape creates a shape item for a rect, circle, ellipse or polygon.
func (c *Client) createSVGShape(ctx context.Context, boardID string, el svgElement, off svgOffset) (string, error) {
	res, err := c.CreateShape(ctx, svgShapeArgs(boardID, el, off))
	return res.ID, err
}

// svgItemType names the Miro item type a parsed non-line element maps to.
func svgItemType(el svgElement) string {
	switch {
	case el.name == "text":
		return "text"
	case el.dataType == "sticky":
		return "sticky_note"
	case el.dataType == "frame":
		return "frame"
	case el.name == "image":
		return "image"
	}
	return "shape"
}

// svgCreators maps each Miro item type to its creation call.
var svgCreators = map[string]func(*Client, context.Context, string, svgElement, svgOffset) (string, error){
	"text":        (*Client).createSVGText,
	"sticky_note": (*Client).createSVGSticky,
	"frame":       (*Client).createSVGFrame,
	"image":       (*Client).createSVGImage,
	"shape":       (*Client).createSVGShape,
}

// createSVGElement creates one board item for a parsed non-line element.
func (c *Client) createSVGElement(ctx context.Context, boardID string, el svgElement, off svgOffset) (SVGCreatedItem, error) {
	itemType := svgItemType(el)
	id, err := svgCreators[itemType](c, ctx, boardID, el, off)
	if err != nil {
		return SVGCreatedItem{}, err
	}
	return SVGCreatedItem{ID: id, Type: itemType, Element: el.name}, nil
}

// createSVGConnector creates a connector for a <line>, resolving its
// data-start/data-end references against the ids created this call. An
// unresolvable reference is a skip, not an error: the referenced element may
// itself have been skipped.
func (c *Client) createSVGConnector(ctx context.Context, boardID string, el svgElement, ids map[string]string) (SVGCreatedItem, bool, error) {
	startID, okS := ids[el.start]
	endID, okE := ids[el.end]
	if !okS || !okE {
		return SVGCreatedItem{}, false, nil
	}
	res, err := c.CreateConnector(ctx, CreateConnectorArgs{
		BoardID:     boardID,
		StartItemID: startID,
		EndItemID:   endID,
		Caption:     el.text,
	})
	if err != nil {
		return SVGCreatedItem{}, false, err
	}
	return SVGCreatedItem{ID: res.ID, Type: "connector", Element: "line"}, true, nil
}

// svgCreateRun carries the state of one create pass: what landed, what was
// skipped, and the authored-id map connectors resolve against.
type svgCreateRun struct {
	boardID string
	off     svgOffset
	created []SVGCreatedItem
	skipped []SVGSkippedElement
	ids     map[string]string
}

// createItem creates one non-line element and records its authored id.
func (r *svgCreateRun) createItem(ctx context.Context, c *Client, el svgElement) error {
	item, err := c.createSVGElement(ctx, r.boardID, el, r.off)
	if err != nil {
		return err
	}
	r.created = append(r.created, item)
	if el.authoredID != "" {
		r.ids[el.authoredID] = item.ID
	}
	return nil
}

// createConnector creates one line connector, or records a skip when its
// references don't resolve.
func (r *svgCreateRun) createConnector(ctx context.Context, c *Client, el svgElement) error {
	item, ok, err := c.createSVGConnector(ctx, r.boardID, el, r.ids)
	if err != nil {
		return err
	}
	if !ok {
		r.skipped = append(r.skipped, SVGSkippedElement{Element: "line", Reason: fmt.Sprintf("unresolved reference (data-start=%q, data-end=%q must match created element ids)", el.start, el.end)})
		return nil
	}
	r.created = append(r.created, item)
	return nil
}

// pass creates every element matching the line filter, in document order.
func (r *svgCreateRun) pass(ctx context.Context, c *Client, elements []svgElement, lines bool) error {
	for _, el := range elements {
		if (el.name == "line") != lines {
			continue
		}
		var err error
		if lines {
			err = r.createConnector(ctx, c, el)
		} else {
			err = r.createItem(ctx, c, el)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// createSVGElements runs the two creation passes: items first (building the
// authored-id map connectors resolve against), then line connectors. The
// returned skips (unresolved connector references) are additional to any the
// parse already reported.
func (c *Client) createSVGElements(ctx context.Context, boardID string, elements []svgElement, off svgOffset) ([]SVGCreatedItem, []SVGSkippedElement, error) {
	run := &svgCreateRun{
		boardID: boardID,
		off:     off,
		created: make([]SVGCreatedItem, 0, len(elements)),
		ids:     make(map[string]string),
	}
	for _, lines := range []bool{false, true} {
		if err := run.pass(ctx, c, elements, lines); err != nil {
			return run.created, run.skipped, err
		}
	}
	return run.created, run.skipped, nil
}

// CreateFromSVG parses an SVG document and creates matching items on the
// board. Unsupported elements are reported, not silently dropped.
func (c *Client) CreateFromSVG(ctx context.Context, args CreateFromSVGArgs) (CreateFromSVGResult, error) {
	if err := validateCreateFromSVGArgs(args); err != nil {
		return CreateFromSVGResult{}, err
	}

	elements, skipped, err := parseSVGElements(args.SVG)
	if err != nil {
		return CreateFromSVGResult{}, err
	}
	if len(elements) > maxSVGCreateElements {
		return CreateFromSVGResult{}, fmt.Errorf("svg contains %d drawable elements; the cap is %d per call", len(elements), maxSVGCreateElements)
	}
	if len(elements) == 0 {
		return CreateFromSVGResult{
			Skipped: skipped,
			Message: "No supported elements found in the SVG (supported: rect, circle, ellipse, text, polygon, image, line)",
		}, nil
	}

	off := svgOffset{dx: args.OffsetX, dy: args.OffsetY}
	created, createSkips, err := c.createSVGElements(ctx, args.BoardID, elements, off)
	skipped = append(skipped, createSkips...)
	if err != nil {
		return partialSVGResult(created, skipped, err)
	}

	return CreateFromSVGResult{
		Created: created,
		Skipped: skipped,
		Count:   len(created),
		Message: fmt.Sprintf("Created %d item(s) from SVG (%d element(s) skipped)", len(created), len(skipped)),
	}, nil
}

// partialSVGResult reports a mid-batch failure without discarding what
// already landed: the caller needs the created IDs to verify or clean up.
func partialSVGResult(created []SVGCreatedItem, skipped []SVGSkippedElement, err error) (CreateFromSVGResult, error) {
	return CreateFromSVGResult{
		Created: created,
		Skipped: skipped,
		Count:   len(created),
		Message: fmt.Sprintf("Partial failure after %d item(s); see error", len(created)),
	}, fmt.Errorf("create from svg failed after %d item(s): %w", len(created), err)
}
