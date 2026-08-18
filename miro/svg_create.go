package miro

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
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

// translateRe extracts translate(x[,y]) from a transform attribute.
var translateRe = regexp.MustCompile(`translate\(\s*(-?[\d.]+)\s*[,\s]\s*(-?[\d.]+)\s*\)|translate\(\s*(-?[\d.]+)\s*\)`)

// svgElement is one drawable element pulled out of the document.
type svgElement struct {
	name    string  // rect | circle | ellipse | text | polygon | image | line
	x, y    float64 // Miro center coordinates (after transform + recentering)
	w, h    float64
	rounded bool
	fill    string
	text    string

	authoredID string // id attribute; line data-start/data-end reference it
	miroID     string // data-miro-id; routes the element to update-in-place
	deleted    bool   // data-deleted="true"; routes the element to deletion
	dataType   string // rect data-type hint: "" | sticky | frame
	title      string // data-title (frame title, image alt text)
	href       string // image source URL
	start, end string // line endpoints: authored ids of the connected elements
}

// parseTranslate reads a transform attribute; only translate is honored,
// anything else returns ok=false so the caller can report the element.
func parseTranslate(transform string) (svgOffset, bool) {
	if strings.TrimSpace(transform) == "" {
		return svgOffset{}, true
	}
	m := translateRe.FindStringSubmatch(transform)
	if m == nil {
		return svgOffset{}, false
	}
	if m[3] != "" { // single-argument form
		dx, _ := strconv.ParseFloat(m[3], 64)
		return svgOffset{dx: dx}, true
	}
	dx, _ := strconv.ParseFloat(m[1], 64)
	dy, _ := strconv.ParseFloat(m[2], 64)
	return svgOffset{dx: dx, dy: dy}, true
}

// svgAttrs is one element's attribute set, keyed by local name.
type svgAttrs map[string]string

// attrMap flattens the element attributes for lookup.
func attrMap(attrs []xml.Attr) svgAttrs {
	m := make(svgAttrs, len(attrs))
	for _, a := range attrs {
		m[a.Name.Local] = a.Value
	}
	return m
}

// float parses a numeric attribute, tolerating a trailing "px".
func (a svgAttrs) float(key string) float64 {
	v := strings.TrimSuffix(strings.TrimSpace(a[key]), "px")
	f, _ := strconv.ParseFloat(v, 64)
	return f
}

// svgOffset is a cumulative translation applied to nested elements.
type svgOffset struct{ dx, dy float64 }

// plus returns the sum of two offsets.
func (o svgOffset) plus(other svgOffset) svgOffset {
	return svgOffset{o.dx + other.dx, o.dy + other.dy}
}

// svgParseState carries the offset stack and accumulators through the token
// walk.
type svgParseState struct {
	offsets  []svgOffset
	elements []svgElement
	skipped  []SVGSkippedElement
	textBuf  *svgElement // non-nil while inside <text>
}

func (st *svgParseState) offset() svgOffset {
	var total svgOffset
	for _, o := range st.offsets {
		total = total.plus(o)
	}
	return total
}

func (st *svgParseState) skip(name, reason string) {
	st.skipped = append(st.skipped, SVGSkippedElement{Element: name, Reason: reason})
}

// pushGroup handles an opening <g>, honoring translate transforms only.
func (st *svgParseState) pushGroup(attrs svgAttrs) {
	off, ok := parseTranslate(attrs["transform"])
	if !ok {
		st.skip("g", "unsupported transform (only translate is honored); children are placed untransformed")
		off = svgOffset{}
	}
	st.offsets = append(st.offsets, off)
}

// addDrawable appends a parsed element, or records a skip when its geometry
// is degenerate. The shared exit point for every shape converter. A deletion
// marker (data-miro-id + data-deleted) is kept regardless of geometry:
// deleting an item needs its identity, nothing else, so
// `<rect data-miro-id="X" data-deleted="true"/>` is a valid minimal diff.
func (st *svgParseState) addDrawable(el svgElement, valid bool, reason string) {
	deletionMarker := el.miroID != "" && el.deleted
	if !valid && !deletionMarker {
		st.skip(el.name, reason)
		return
	}
	st.elements = append(st.elements, el)
}

// identity extracts the id/update/delete attributes shared by every element.
func (a svgAttrs) identity() (authoredID, miroID string, deleted bool) {
	return a["id"], a["data-miro-id"], a["data-deleted"] == "true"
}

// stamp copies the identity attributes onto a parsed element.
func stamp(el svgElement, attrs svgAttrs) svgElement {
	el.authoredID, el.miroID, el.deleted = attrs.identity()
	return el
}

// validRectType reports whether a rect data-type hint is one this parser maps.
func validRectType(dataType string) bool {
	return dataType == "" || dataType == "sticky" || dataType == "frame"
}

// addRect converts a <rect> to a centered element, honoring the optional
// data-type hint (sticky, frame).
func (st *svgParseState) addRect(attrs svgAttrs, off svgOffset) {
	w, h := attrs.float("width"), attrs.float("height")
	dataType := attrs["data-type"]
	if !validRectType(dataType) {
		st.skip("rect", fmt.Sprintf("unsupported data-type %q (supported: sticky, frame)", dataType))
		return
	}
	st.addDrawable(stamp(svgElement{
		name: "rect",
		x:    attrs.float("x") + w/2 + off.dx,
		y:    attrs.float("y") + h/2 + off.dy,
		w:    w, h: h,
		rounded:  attrs.float("rx") > 0,
		fill:     attrs["fill"],
		dataType: dataType,
		text:     attrs["data-content"],
		title:    attrs["data-title"],
	}, attrs), w > 0 && h > 0, "zero or missing width/height")
}

// polygonBounds computes the bounding box of a points attribute and reports
// how many coordinate pairs it holds.
func polygonBounds(points string) (b svgBounds, count int) {
	fields := strings.FieldsFunc(points, func(r rune) bool { return r == ' ' || r == ',' || r == '\n' || r == '\t' })
	for i := 0; i+1 < len(fields); i += 2 {
		x, errX := strconv.ParseFloat(fields[i], 64)
		y, errY := strconv.ParseFloat(fields[i+1], 64)
		if errX != nil || errY != nil {
			return svgBounds{}, 0
		}
		b.add(x, y, 0, 0)
		count++
	}
	return b, count
}

// addPolygon converts a 3-point <polygon> to a triangle shape spanning its
// bounding box. Other point counts have no faithful Miro shape and are
// skipped.
func (st *svgParseState) addPolygon(attrs svgAttrs, off svgOffset) {
	b, count := polygonBounds(attrs["points"])
	if count != 3 {
		st.skip("polygon", fmt.Sprintf("%d points; only 3-point polygons map to a shape (triangle)", count))
		return
	}
	w, h := b.maxX-b.minX, b.maxY-b.minY
	st.addDrawable(stamp(svgElement{
		name: "polygon",
		x:    b.minX + w/2 + off.dx,
		y:    b.minY + h/2 + off.dy,
		w:    w, h: h,
		fill: attrs["fill"],
	}, attrs), w > 0 && h > 0, "degenerate points (zero-area bounding box)")
}

// addImage converts an <image> with a public href to an image element.
func (st *svgParseState) addImage(attrs svgAttrs, off svgOffset) {
	href := attrs["href"]
	w, h := attrs.float("width"), attrs.float("height")
	st.addDrawable(stamp(svgElement{
		name: "image",
		x:    attrs.float("x") + w/2 + off.dx,
		y:    attrs.float("y") + h/2 + off.dy,
		w:    w, h: h,
		href:  href,
		title: attrs["data-title"],
	}, attrs), href != "" && w > 0, "image requires href and width")
}

// addLine converts a <line> carrying data-start/data-end references to a
// connector between the two referenced elements.
func (st *svgParseState) addLine(attrs svgAttrs) {
	start, end := attrs["data-start"], attrs["data-end"]
	st.addDrawable(stamp(svgElement{
		name:  "line",
		start: start, end: end,
		text: attrs["data-caption"],
	}, attrs), start != "" && end != "", "line requires data-start and data-end referencing element ids")
}

// svgRadii reads the radius pair for a circle (r) or ellipse (rx/ry).
func svgRadii(name string, attrs svgAttrs) (rx, ry float64) {
	if name == "circle" {
		r := attrs.float("r")
		return r, r
	}
	return attrs.float("rx"), attrs.float("ry")
}

// addEllipseKind converts a <circle> or <ellipse>; the two differ only in
// how their radii are spelled.
func (st *svgParseState) addEllipseKind(name string, attrs svgAttrs, off svgOffset) {
	rx, ry := svgRadii(name, attrs)
	st.addDrawable(stamp(svgElement{
		name: name,
		x:    attrs.float("cx") + off.dx,
		y:    attrs.float("cy") + off.dy,
		w:    2 * rx, h: 2 * ry,
		fill: attrs["fill"],
	}, attrs), rx > 0 && ry > 0, "zero or missing radius")
}

// beginText opens a <text> buffer; its content arrives as CharData tokens.
func (st *svgParseState) beginText(attrs svgAttrs, off svgOffset) {
	el := stamp(svgElement{
		name: "text",
		x:    attrs.float("x") + off.dx,
		y:    attrs.float("y") + off.dy,
	}, attrs)
	st.textBuf = &el
}

// svgElementHandlers maps each supported tag to its parser. Structural tags
// map to nil (nothing to draw); an absent tag is an unsupported element.
var svgElementHandlers = map[string]func(*svgParseState, svgAttrs, svgOffset){
	"svg": nil, "title": nil, "desc": nil, "defs": nil,
	"g":       func(st *svgParseState, a svgAttrs, _ svgOffset) { st.pushGroup(a) },
	"rect":    (*svgParseState).addRect,
	"circle":  func(st *svgParseState, a svgAttrs, off svgOffset) { st.addEllipseKind("circle", a, off) },
	"ellipse": func(st *svgParseState, a svgAttrs, off svgOffset) { st.addEllipseKind("ellipse", a, off) },
	"polygon": (*svgParseState).addPolygon,
	"image":   (*svgParseState).addImage,
	"line":    func(st *svgParseState, a svgAttrs, _ svgOffset) { st.addLine(a) },
	"text":    (*svgParseState).beginText,
}

// startElement dispatches one opening tag to its element handler.
func (st *svgParseState) startElement(tok xml.StartElement) {
	handler, known := svgElementHandlers[tok.Name.Local]
	if !known {
		st.skip(tok.Name.Local, "unsupported element")
		return
	}
	if handler != nil {
		handler(st, attrMap(tok.Attr), st.offset())
	}
}

// closeGroup pops the offset stack for a closing </g>.
func (st *svgParseState) closeGroup() {
	if len(st.offsets) > 0 {
		st.offsets = st.offsets[:len(st.offsets)-1]
	}
}

// closeText finalizes a </text>, keeping non-blank content only — or a blank
// one when it is a deletion marker, which needs no content.
func (st *svgParseState) closeText() {
	if st.textBuf == nil {
		return
	}
	deletion := st.textBuf.miroID != "" && st.textBuf.deleted
	if s := strings.TrimSpace(st.textBuf.text); s != "" || deletion {
		st.textBuf.text = s
		st.elements = append(st.elements, *st.textBuf)
	} else {
		st.skip("text", "empty content")
	}
	st.textBuf = nil
}

// endElement dispatches one closing tag.
func (st *svgParseState) endElement(tok xml.EndElement) {
	switch tok.Name.Local {
	case "g":
		st.closeGroup()
	case "text":
		st.closeText()
	}
}

// handleToken routes one XML token to the matching state handler.
func (st *svgParseState) handleToken(tok xml.Token) {
	switch t := tok.(type) {
	case xml.StartElement:
		st.startElement(t)
	case xml.CharData:
		if st.textBuf != nil {
			st.textBuf.text += string(t)
		}
	case xml.EndElement:
		st.endElement(t)
	}
}

// parseSVGElements walks the document and returns drawable elements plus the
// skip report.
func parseSVGElements(svgSource string) ([]svgElement, []SVGSkippedElement, error) {
	dec := xml.NewDecoder(strings.NewReader(svgSource))
	// SVG documents commonly carry entity-free straightforward XML; anything
	// exotic fails the parse and surfaces as an error rather than a guess.
	dec.Strict = false

	st := &svgParseState{}
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return st.elements, st.skipped, nil
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse SVG: %w", err)
		}
		st.handleToken(tok)
	}
}

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
