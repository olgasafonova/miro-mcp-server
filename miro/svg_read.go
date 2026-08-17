package miro

import (
	"context"
	"fmt"
	"html"
	"strings"
)

// =============================================================================
// Board -> SVG (local rendering, no API beyond the item listing)
// =============================================================================
//
// Renders board items to a plain SVG document from the same ListAllItems data
// the content tools already fetch. The mapping is deliberately flat: frames
// and cards become outlined rects, stickies and shapes become filled rects or
// ellipses, text becomes <text>, connectors become lines between item
// centers. Items with no useful geometry are counted as skipped rather than
// guessed at. This mirrors the local-Mermaid-parser philosophy: a useful
// approximation computed locally beats a round trip to a rendering service.

// maxSVGItems caps the item fetch for a render.
const maxSVGItems = 2000

// defaultSVGItems is the fetch size when the caller doesn't specify one.
const defaultSVGItems = 500

// svgRenderable reports whether an item carries enough geometry to draw.
func svgRenderable(it ItemSummary) bool {
	return it.Width > 0 && it.Height > 0 || it.Type == "text"
}

// svgFill picks a fill color for an item, defaulting per type.
func svgFill(it ItemSummary) string {
	if it.Style != nil && it.Style.FillColor != "" {
		return it.Style.FillColor
	}
	switch it.Type {
	case "sticky_note":
		return "#fff9b1" // Miro's default yellow
	case "frame":
		return "none"
	default:
		return "#e6e6e6"
	}
}

// svgLabel extracts a plain-text label from item content (which may carry
// HTML fragments like <p>...</p>).
func svgLabel(content string) string {
	replacer := strings.NewReplacer("<p>", "", "</p>", " ", "<br>", " ", "<br/>", " ")
	s := strings.TrimSpace(replacer.Replace(content))
	const maxLabel = 60
	if len(s) > maxLabel {
		s = s[:maxLabel-3] + "..."
	}
	return s
}

// svgBounds tracks the bounding box of rendered items so the viewBox can be
// fitted after the fact.
type svgBounds struct {
	minX, minY, maxX, maxY float64
	set                    bool
}

func (b *svgBounds) add(x, y, w, h float64) {
	if !b.set {
		b.minX, b.minY, b.maxX, b.maxY = x, y, x+w, y+h
		b.set = true
		return
	}
	b.minX = min(b.minX, x)
	b.minY = min(b.minY, y)
	b.maxX = max(b.maxX, x+w)
	b.maxY = max(b.maxY, y+h)
}

// svgCenteredLabel appends a centered <text> label when there is one.
func svgCenteredLabel(sb *strings.Builder, x, y float64, label string) {
	if label == "" {
		return
	}
	fmt.Fprintf(sb, `<text x="%.1f" y="%.1f" font-size="12" text-anchor="middle">%s</text>`+"\n",
		x, y, html.EscapeString(label))
}

// renderSVGText draws a standalone text item.
func renderSVGText(sb *strings.Builder, it ItemSummary, bounds *svgBounds, label string) {
	w := it.Width
	if w == 0 {
		w = float64(len(label)) * 8
	}
	bounds.add(it.X-w/2, it.Y-10, w, 20)
	fmt.Fprintf(sb, `<text x="%.1f" y="%.1f" font-size="14" text-anchor="middle" data-miro-id=%q data-miro-type="text">%s</text>`+"\n",
		it.X, it.Y, it.ID, html.EscapeString(label))
}

// renderSVGShape draws a shape item as an ellipse (circle kind) or rect.
func renderSVGShape(sb *strings.Builder, it ItemSummary, bounds *svgBounds, label string) {
	x, y := it.X-it.Width/2, it.Y-it.Height/2
	bounds.add(x, y, it.Width, it.Height)
	if it.Style != nil && it.Style.Shape == "circle" {
		fmt.Fprintf(sb, `<ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill=%q stroke="#333" data-miro-id=%q data-miro-type="shape"/>`+"\n",
			it.X, it.Y, it.Width/2, it.Height/2, svgFill(it), it.ID)
	} else {
		fmt.Fprintf(sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill=%q stroke="#333" data-miro-id=%q data-miro-type="shape"/>`+"\n",
			x, y, it.Width, it.Height, svgFill(it), it.ID)
	}
	svgCenteredLabel(sb, it.X, it.Y, label)
}

// renderSVGFrame draws a frame as a dashed outline with its title above.
func renderSVGFrame(sb *strings.Builder, it ItemSummary, bounds *svgBounds, label string) {
	x, y := it.X-it.Width/2, it.Y-it.Height/2
	bounds.add(x, y, it.Width, it.Height)
	fmt.Fprintf(sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="none" stroke="#999" stroke-dasharray="6,3" data-miro-id=%q data-miro-type="frame"/>`+"\n",
		x, y, it.Width, it.Height, it.ID)
	if label != "" {
		fmt.Fprintf(sb, `<text x="%.1f" y="%.1f" font-size="12" fill="#666">%s</text>`+"\n",
			x+4, y-6, html.EscapeString(label))
	}
}

// renderSVGBox draws sticky notes, cards, images and documents as rounded
// rects with a centered label.
func renderSVGBox(sb *strings.Builder, it ItemSummary, bounds *svgBounds, label string) {
	x, y := it.X-it.Width/2, it.Y-it.Height/2
	bounds.add(x, y, it.Width, it.Height)
	fmt.Fprintf(sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="4" fill=%q stroke="#666" data-miro-id=%q data-miro-type=%q/>`+"\n",
		x, y, it.Width, it.Height, svgFill(it), it.ID, it.Type)
	svgCenteredLabel(sb, it.X, it.Y, label)
}

// renderSVGItem dispatches one item to its type renderer. Returns false when
// the item has no visual mapping.
func renderSVGItem(sb *strings.Builder, it ItemSummary, bounds *svgBounds) bool {
	if !svgRenderable(it) {
		return false
	}
	label := svgLabel(it.Content)

	switch it.Type {
	case "text":
		renderSVGText(sb, it, bounds, label)
	case "shape":
		renderSVGShape(sb, it, bounds, label)
	case "frame":
		renderSVGFrame(sb, it, bounds, label)
	case "sticky_note", "card", "app_card", "image", "document":
		renderSVGBox(sb, it, bounds, label)
	default:
		return false
	}
	return true
}

// renderSVGConnectors appends a line per connector whose endpoints are known.
func renderSVGConnectors(sb *strings.Builder, connectors []ConnectorSummary, byID map[string]ItemSummary) {
	for _, conn := range connectors {
		from, okF := byID[conn.StartItemID]
		to, okT := byID[conn.EndItemID]
		if !okF || !okT {
			continue
		}
		fmt.Fprintf(sb, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#555" data-miro-id=%q data-miro-type="connector"/>`+"\n",
			from.X, from.Y, to.X, to.Y, conn.ID)
		if conn.Caption != "" {
			fmt.Fprintf(sb, `<text x="%.1f" y="%.1f" font-size="10" fill="#555" text-anchor="middle">%s</text>`+"\n",
				(from.X+to.X)/2, (from.Y+to.Y)/2-4, html.EscapeString(svgLabel(conn.Caption)))
		}
	}
}

// clampSVGMaxItems normalizes the requested item cap.
func clampSVGMaxItems(n int) int {
	if n > 0 && n <= maxSVGItems {
		return n
	}
	return defaultSVGItems
}

// renderSVGItems draws all items in two passes (frames first, so they sit
// under their children) and reports how many rendered vs skipped.
func renderSVGItems(sb *strings.Builder, items []ItemSummary, bounds *svgBounds) (rendered, skipped int) {
	for _, framesPass := range []bool{true, false} {
		for _, it := range items {
			if (it.Type == "frame") != framesPass {
				continue
			}
			if renderSVGItem(sb, it, bounds) {
				rendered++
			} else {
				skipped++
			}
		}
	}
	return rendered, skipped
}

// wrapSVGDocument fits the viewBox to the rendered bounds and wraps the body.
func wrapSVGDocument(body string, bounds *svgBounds) string {
	if !bounds.set {
		bounds.minX, bounds.minY, bounds.maxX, bounds.maxY = 0, 0, 100, 100
	}
	const pad = 20.0
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="%.1f %.1f %.1f %.1f">`+"\n%s</svg>",
		bounds.minX-pad, bounds.minY-pad,
		bounds.maxX-bounds.minX+2*pad, bounds.maxY-bounds.minY+2*pad,
		body)
}

// collectFrameItems pages through a frame's children up to maxItems. The
// returned truncated flag reports whether more children remained.
func (c *Client) collectFrameItems(ctx context.Context, boardID, frameID string, maxItems int) ([]ItemSummary, bool, error) {
	var items []ItemSummary
	cursor := ""
	for {
		page, err := c.GetFrameItems(ctx, GetFrameItemsArgs{
			BoardID: boardID, FrameID: frameID,
			Limit: 100, Cursor: cursor, DetailLevel: "full",
		})
		if err != nil {
			return nil, false, fmt.Errorf("failed to list frame items: %w", err)
		}
		items = append(items, page.Items...)
		if len(items) >= maxItems {
			return items[:maxItems], page.HasMore || len(items) > maxItems, nil
		}
		if !page.HasMore {
			return items, false, nil
		}
		cursor = page.Cursor
	}
}

// readFrameSVG renders one frame and its children. Child coordinates come
// back frame-relative (center-anchored, origin at the frame's top-left), so
// the frame outline is drawn at (0,0) and children render as returned. This
// is the scoped read the official Miro MCP composer lacks: its
// canvas_read_as_svg always returns the whole board.
func (c *Client) readFrameSVG(ctx context.Context, args ReadBoardSVGArgs) (ReadBoardSVGResult, error) {
	frame, err := c.GetFrame(ctx, GetFrameArgs{BoardID: args.BoardID, FrameID: args.FrameID})
	if err != nil {
		return ReadBoardSVGResult{}, fmt.Errorf("failed to get frame: %w", err)
	}

	items, truncated, err := c.collectFrameItems(ctx, args.BoardID, args.FrameID, clampSVGMaxItems(args.MaxItems))
	if err != nil {
		return ReadBoardSVGResult{}, err
	}

	var body strings.Builder
	bounds := &svgBounds{}
	bounds.add(0, 0, frame.Width, frame.Height)
	fmt.Fprintf(&body, `<rect x="0" y="0" width="%.1f" height="%.1f" fill="none" stroke="#999" stroke-dasharray="6,3" data-miro-id=%q data-miro-type="frame"/>`+"\n",
		frame.Width, frame.Height, frame.ID)
	if frame.Title != "" {
		fmt.Fprintf(&body, `<text x="4" y="-6" font-size="12" fill="#666">%s</text>`+"\n", html.EscapeString(frame.Title))
	}
	rendered, skipped := renderSVGItems(&body, items, bounds)

	return ReadBoardSVGResult{
		SVG:       wrapSVGDocument(body.String(), bounds),
		ItemCount: rendered,
		Skipped:   skipped,
		Truncated: truncated,
		Message:   fmt.Sprintf("Rendered frame %q with %d item(s) as SVG (%d skipped); child coordinates are relative to the frame's top-left", frame.ID, rendered, skipped),
	}, nil
}

// ReadBoardSVG renders a board's items as an SVG document, computed locally
// from the item listing. Connector endpoints resolve to item centers. With
// frame_id set, the render is scoped to that frame and its children.
func (c *Client) ReadBoardSVG(ctx context.Context, args ReadBoardSVGArgs) (ReadBoardSVGResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ReadBoardSVGResult{}, err
	}
	if args.FrameID != "" {
		return c.readFrameSVG(ctx, args)
	}

	items, err := c.ListAllItems(ctx, ListAllItemsArgs{
		BoardID:     args.BoardID,
		MaxItems:    clampSVGMaxItems(args.MaxItems),
		DetailLevel: "full",
	})
	if err != nil {
		return ReadBoardSVGResult{}, fmt.Errorf("failed to list items: %w", err)
	}

	// Connectors are best-effort decoration; a failure there shouldn't sink
	// the render.
	var connectors []ConnectorSummary
	if connResult, connErr := c.ListConnectors(ctx, ListConnectorsArgs{BoardID: args.BoardID, Limit: 100}); connErr == nil {
		connectors = connResult.Connectors
	}

	byID := make(map[string]ItemSummary, len(items.Items))
	for _, it := range items.Items {
		byID[it.ID] = it
	}

	var body strings.Builder
	bounds := &svgBounds{}
	rendered, skipped := renderSVGItems(&body, items.Items, bounds)
	renderSVGConnectors(&body, connectors, byID)

	return ReadBoardSVGResult{
		SVG:       wrapSVGDocument(body.String(), bounds),
		ItemCount: rendered,
		Skipped:   skipped,
		Truncated: items.Truncated,
		Message:   fmt.Sprintf("Rendered %d item(s) as SVG (%d skipped)", rendered, skipped),
	}, nil
}
