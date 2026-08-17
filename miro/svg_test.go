package miro

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// Shared SVG test helpers
// =============================================================================

// newBoardItemsServer serves the given items from the board items endpoint and
// an empty list from the connectors endpoint.
func newBoardItemsServer(items []map[string]interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/connectors") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}, "cursor": ""})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": items, "cursor": ""})
	}))
}

// requireGeometry asserts an element's center coordinates and size, given as
// [x, y, width, height].
func requireGeometry(t *testing.T, label string, got svgElement, want [4]float64) {
	t.Helper()
	if [4]float64{got.x, got.y, got.w, got.h} != want {
		t.Errorf("%s = %+v, want center (%v,%v) size %vx%v", label, got, want[0], want[1], want[2], want[3])
	}
}

// requireSVGContains asserts that the rendered SVG carries the given fragment.
func requireSVGContains(t *testing.T, svg, substr, what string) {
	t.Helper()
	if !strings.Contains(svg, substr) {
		t.Errorf("%s missing from SVG (want fragment %q)", what, substr)
	}
}

// requireWellFormedXML asserts the rendered SVG parses as XML.
func requireWellFormedXML(t *testing.T, svg string) {
	t.Helper()
	if err := xml.Unmarshal([]byte(svg), new(interface{})); err != nil {
		t.Errorf("rendered SVG is not well-formed XML: %v\n%s", err, svg)
	}
}

// requireCreatedMapping asserts one created item's source element and Miro
// type; only those two fields of want are compared.
func requireCreatedMapping(t *testing.T, label string, got, want SVGCreatedItem) {
	t.Helper()
	if got.Element != want.Element {
		t.Errorf("%s = %+v, want element %q", label, got, want.Element)
	}
	if got.Type != want.Type {
		t.Errorf("%s = %+v, want type %q", label, got, want.Type)
	}
}

// findElementByName returns the first parsed element with the given tag name.
func findElementByName(elements []svgElement, name string) *svgElement {
	for i := range elements {
		if elements[i].name == name {
			return &elements[i]
		}
	}
	return nil
}

// =============================================================================
// SVG parsing (create direction)
// =============================================================================

func TestParseSVGElements_SupportedSubset(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg">
		<rect x="10" y="20" width="100" height="50" fill="#ff0000"/>
		<rect x="0" y="0" width="40" height="40" rx="8"/>
		<circle cx="200" cy="100" r="30" fill="blue"/>
		<ellipse cx="300" cy="150" rx="40" ry="20"/>
		<text x="50" y="60">Hello board</text>
	</svg>`

	elements, skipped, err := parseSVGElements(svg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skipped) != 0 {
		t.Errorf("skipped = %+v, want none", skipped)
	}
	if len(elements) != 5 {
		t.Fatalf("got %d elements, want 5", len(elements))
	}

	// rect x/y are top-left; Miro coordinates are centers.
	requireGeometry(t, "rect", elements[0], [4]float64{60, 45, 100, 50})
	if elements[0].fill != "#ff0000" {
		t.Errorf("rect fill = %q, want '#ff0000'", elements[0].fill)
	}
	if elements[1].rounded != true {
		t.Error("rx>0 rect not marked rounded")
	}
	requireGeometry(t, "circle", elements[2], [4]float64{200, 100, 60, 60})
	if elements[4].text != "Hello board" {
		t.Errorf("text content = %q, want 'Hello board'", elements[4].text)
	}
}

func TestParseSVGElements_NestedTranslate(t *testing.T) {
	svg := `<svg><g transform="translate(100, 50)"><g transform="translate(10,5)">
		<rect x="0" y="0" width="20" height="20"/>
	</g></g></svg>`

	elements, _, err := parseSVGElements(svg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(elements) != 1 {
		t.Fatalf("got %d elements, want 1", len(elements))
	}
	// center = 0+10 (half size) + 100+10 offsets, 0+10 + 50+5
	if elements[0].x != 120 || elements[0].y != 65 {
		t.Errorf("nested translate center = (%v,%v), want (120,65)", elements[0].x, elements[0].y)
	}
}

func TestParseSVGElements_UnsupportedAreReported(t *testing.T) {
	svg := `<svg>
		<path d="M0 0 L10 10"/>
		<polygon points="0,0 10,0 5,10"/>
		<line x1="0" y1="0" x2="10" y2="10"/>
		<rect x="0" y="0" width="10" height="10"/>
	</svg>`

	elements, skipped, err := parseSVGElements(svg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(elements) != 1 {
		t.Errorf("got %d elements, want 1 (the rect)", len(elements))
	}
	if len(skipped) != 3 {
		t.Fatalf("skipped = %d, want 3 (path, polygon, line)", len(skipped))
	}
	names := map[string]bool{}
	for _, s := range skipped {
		if s.Reason == "" {
			t.Errorf("skipped %q has empty reason", s.Element)
		}
		names[s.Element] = true
	}
	for _, want := range []string{"path", "polygon", "line"} {
		if !names[want] {
			t.Errorf("skip report missing %q", want)
		}
	}
}

func TestParseSVGElements_DegenerateGeometry(t *testing.T) {
	svg := `<svg>
		<rect x="0" y="0" width="0" height="10"/>
		<circle cx="1" cy="1" r="0"/>
		<text x="0" y="0">   </text>
	</svg>`

	elements, skipped, err := parseSVGElements(svg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(elements) != 0 {
		t.Errorf("degenerate elements created: %+v", elements)
	}
	if len(skipped) != 3 {
		t.Errorf("skipped = %d, want 3", len(skipped))
	}
}

func TestCreateFromSVG_CreatesShapesAndText(t *testing.T) {
	var created []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		created = append(created, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "item-1", "data": map[string]interface{}{"shape": "rectangle"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateFromSVG(context.Background(), CreateFromSVGArgs{
		BoardID: "board1",
		SVG:     `<svg><rect x="0" y="0" width="20" height="20"/><text x="5" y="5">hi</text></svg>`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	requireCreatedMapping(t, "first created", result.Created[0], SVGCreatedItem{Element: "rect", Type: "shape"})
	requireCreatedMapping(t, "second created", result.Created[1], SVGCreatedItem{Element: "text", Type: "text"})
	joined := strings.Join(created, " ")
	if !strings.Contains(joined, "/shapes") {
		t.Errorf("API paths hit = %v, want shapes endpoint", created)
	}
	if !strings.Contains(joined, "/texts") {
		t.Errorf("API paths hit = %v, want texts endpoint", created)
	}
}

func TestCreateFromSVG_ElementCapRejects(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("<svg>")
	for range maxSVGCreateElements + 1 {
		sb.WriteString(`<rect x="0" y="0" width="5" height="5"/>`)
	}
	sb.WriteString("</svg>")

	client := newTestClientWithServer("http://unused")
	_, err := client.CreateFromSVG(context.Background(), CreateFromSVGArgs{BoardID: "board1", SVG: sb.String()})
	if err == nil || !strings.Contains(err.Error(), "cap") {
		t.Errorf("oversized SVG accepted or wrong error: %v", err)
	}
}

func TestCreateFromSVG_EmptyAndNoDrawables(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	if _, err := client.CreateFromSVG(context.Background(), CreateFromSVGArgs{BoardID: "board1", SVG: "  "}); err == nil {
		t.Error("blank SVG accepted")
	}
	result, err := client.CreateFromSVG(context.Background(), CreateFromSVGArgs{
		BoardID: "board1", SVG: `<svg><path d="M0 0"/></svg>`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 0 || result.Message == "" {
		t.Errorf("no-drawables result = %+v, want count 0 with explanatory message", result)
	}
}

func TestCreateFromSVG_PartialFailureNamesCreated(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls > 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "boom"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "item-1"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateFromSVG(context.Background(), CreateFromSVGArgs{
		BoardID: "board1",
		SVG:     `<svg><rect x="0" y="0" width="5" height="5"/><rect x="10" y="10" width="5" height="5"/></svg>`,
	})
	if err == nil {
		t.Fatal("expected error from second create")
	}
	if len(result.Created) != 1 || result.Created[0].ID != "item-1" {
		t.Errorf("partial result loses created IDs: %+v", result.Created)
	}
}

// =============================================================================
// Board rendering (read direction)
// =============================================================================

func svgBoardItems() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id": "f1", "type": "frame",
			"position": map[string]interface{}{"x": 100.0, "y": 100.0},
			"geometry": map[string]interface{}{"width": 400.0, "height": 300.0},
			"data":     map[string]interface{}{"title": "Sprint"},
		},
		{
			"id": "s1", "type": "sticky_note",
			"position": map[string]interface{}{"x": 50.0, "y": 60.0},
			"geometry": map[string]interface{}{"width": 100.0, "height": 100.0},
			"data":     map[string]interface{}{"content": "<p>Do the thing</p>"},
			"style":    map[string]interface{}{"fillColor": "light_yellow"},
		},
		{
			"id": "e1", "type": "embed",
			"position": map[string]interface{}{"x": 0.0, "y": 0.0},
		},
	}
}

func TestReadBoardSVG_RendersAndCounts(t *testing.T) {
	server := newBoardItemsServer(svgBoardItems())
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ReadBoardSVG(context.Background(), ReadBoardSVGArgs{BoardID: "board1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ItemCount != 2 {
		t.Errorf("ItemCount = %d, want 2 (frame + sticky)", result.ItemCount)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (the embed)", result.Skipped)
	}
	requireSVGContains(t, result.SVG, `data-miro-id="s1"`, "sticky's data-miro-id")
	requireSVGContains(t, result.SVG, "Do the thing", "sticky label")
	if strings.Contains(result.SVG, "<p>") {
		t.Error("HTML fragments leaked into SVG text")
	}
	// The document must be well-formed XML.
	requireWellFormedXML(t, result.SVG)
	// Frame must be drawn before the sticky so it sits underneath.
	if strings.Index(result.SVG, `data-miro-id="f1"`) > strings.Index(result.SVG, `data-miro-id="s1"`) {
		t.Error("frame rendered after items; z-order wrong")
	}
}

func TestReadBoardSVG_EscapesContent(t *testing.T) {
	server := newBoardItemsServer([]map[string]interface{}{{
		"id": "t1", "type": "text",
		"position": map[string]interface{}{"x": 0.0, "y": 0.0},
		"data":     map[string]interface{}{"content": `a < b & "c"`},
	}})
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ReadBoardSVG(context.Background(), ReadBoardSVGArgs{BoardID: "board1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Special characters must not break the XML.
	requireWellFormedXML(t, result.SVG)
}

// TestSVGRoundTrip feeds ReadBoardSVG's output into parseSVGElements: every
// rendered shape must come back out, proving read and create speak the same
// dialect.
func TestSVGRoundTrip(t *testing.T) {
	server := newBoardItemsServer([]map[string]interface{}{
		{
			"id": "sh1", "type": "shape",
			"position": map[string]interface{}{"x": 100.0, "y": 200.0},
			"geometry": map[string]interface{}{"width": 80.0, "height": 40.0},
			"style":    map[string]interface{}{"fillColor": "#00ff00"},
		},
		{
			// The API carries the shape kind in data.shape, not style.
			"id": "sh2", "type": "shape",
			"position": map[string]interface{}{"x": 300.0, "y": 300.0},
			"geometry": map[string]interface{}{"width": 60.0, "height": 60.0},
			"data":     map[string]interface{}{"shape": "circle"},
		},
	})
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	rendered, err := client.ReadBoardSVG(context.Background(), ReadBoardSVGArgs{BoardID: "board1"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	elements, skipped, err := parseSVGElements(rendered.SVG)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	_ = skipped // labels under shapes come back as extra text elements; that's fine
	rect := findElementByName(elements, "rect")
	ellipse := findElementByName(elements, "ellipse")
	if rect == nil {
		t.Fatalf("round trip lost the rect; got %+v", elements)
	}
	if ellipse == nil {
		t.Fatalf("round trip lost the ellipse; got %+v", elements)
	}
	requireGeometry(t, "rect round trip", *rect, [4]float64{100, 200, 80, 40})
	requireGeometry(t, "ellipse round trip", *ellipse, [4]float64{300, 300, 60, 60})
}
