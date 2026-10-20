package miro

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// SVG document parsing for CreateFromSVG. The supported subset and the
// element-to-item mapping are documented at the top of svg_create.go.

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
