package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// Parser extensions (identity attributes, data-type hints, new elements)
// =============================================================================

// mustParseSVG parses the document and fails the test on error or an
// unexpected element count.
func mustParseSVG(t *testing.T, svg string, wantElements int) ([]svgElement, []SVGSkippedElement) {
	t.Helper()
	elements, skipped, err := parseSVGElements(svg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(elements) != wantElements {
		t.Fatalf("got %d elements, want %d", len(elements), wantElements)
	}
	return elements, skipped
}

// requireField compares one string field of a parsed element.
func requireField(t *testing.T, label, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", label, got, want)
	}
}

// requireOneSkip asserts exactly one skip whose reason carries the fragment.
func requireOneSkip(t *testing.T, skipped []SVGSkippedElement, reasonFragment string) {
	t.Helper()
	if len(skipped) != 1 {
		t.Fatalf("skipped = %+v, want exactly 1", skipped)
	}
	if !strings.Contains(skipped[0].Reason, reasonFragment) {
		t.Errorf("skip reason = %q, want fragment %q", skipped[0].Reason, reasonFragment)
	}
}

func TestParseSVGElements_IdentityAttributes(t *testing.T) {
	elements, _ := mustParseSVG(t, `<svg>
		<rect id="a" data-miro-id="m1" x="0" y="0" width="10" height="10"/>
		<rect data-miro-id="m2" data-deleted="true" x="0" y="0" width="10" height="10"/>
		<circle id="b" cx="5" cy="5" r="5"/>
	</svg>`, 3)

	requireField(t, "rect authoredID", elements[0].authoredID, "a")
	requireField(t, "rect miroID", elements[0].miroID, "m1")
	if elements[0].deleted {
		t.Error("first rect marked deleted without data-deleted")
	}
	requireField(t, "second rect miroID", elements[1].miroID, "m2")
	if !elements[1].deleted {
		t.Error("data-deleted rect not marked deleted")
	}
	requireField(t, "circle authoredID", elements[2].authoredID, "b")
}

func TestParseSVGElements_RectDataTypeHints(t *testing.T) {
	elements, skipped := mustParseSVG(t, `<svg>
		<rect data-type="sticky" data-content="Do it" x="0" y="0" width="200" height="200" fill="yellow"/>
		<rect data-type="frame" data-title="Zone A" x="0" y="0" width="800" height="600"/>
		<rect data-type="banana" x="0" y="0" width="10" height="10"/>
	</svg>`, 2)

	requireField(t, "sticky dataType", elements[0].dataType, "sticky")
	requireField(t, "sticky content", elements[0].text, "Do it")
	requireField(t, "frame dataType", elements[1].dataType, "frame")
	requireField(t, "frame title", elements[1].title, "Zone A")
	requireOneSkip(t, skipped, "banana")
}

func TestParseSVGElements_TrianglePolygon(t *testing.T) {
	elements, _ := mustParseSVG(t, `<svg><polygon points="0,0 40,0 20,30" fill="#ff0000"/></svg>`, 1)

	requireGeometry(t, "triangle", elements[0], [4]float64{20, 15, 40, 30})
	requireField(t, "polygon name", elements[0].name, "polygon")
	requireField(t, "polygon fill", elements[0].fill, "#ff0000")
}

func TestParseSVGElements_Image(t *testing.T) {
	elements, skipped := mustParseSVG(t, `<svg>
		<image href="https://example.com/pic.png" x="0" y="0" width="100" height="80" data-title="Pic"/>
		<image x="0" y="0" width="100" height="80"/>
	</svg>`, 1)

	requireField(t, "image href", elements[0].href, "https://example.com/pic.png")
	requireField(t, "image title", elements[0].title, "Pic")
	requireOneSkip(t, skipped, "href")
}

func TestParseSVGElements_Line(t *testing.T) {
	elements, _ := mustParseSVG(t, `<svg><line data-start="a" data-end="b" data-caption="flows to"/></svg>`, 1)

	requireField(t, "line start", elements[0].start, "a")
	requireField(t, "line end", elements[0].end, "b")
	requireField(t, "line caption", elements[0].text, "flows to")
}

// =============================================================================
// Create extensions (sticky, frame, image, connector routing)
// =============================================================================

// newRoutingCreateServer records every request path and answers each create
// with a fresh id.
func newRoutingCreateServer(paths *[]string) *httptest.Server {
	n := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*paths = append(*paths, r.Method+" "+r.URL.Path)
		n++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "item-" + string(rune('0'+n))})
	}))
}

func TestCreateFromSVG_RoutesNewElementTypes(t *testing.T) {
	var paths []string
	server := newRoutingCreateServer(&paths)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateFromSVG(context.Background(), CreateFromSVGArgs{
		BoardID: "board1",
		SVG: `<svg>
			<rect data-type="frame" data-title="Zone" x="0" y="0" width="800" height="600"/>
			<rect data-type="sticky" data-content="hi" x="10" y="10" width="200" height="200" fill="yellow"/>
			<image href="https://example.com/p.png" x="0" y="0" width="100" height="80"/>
			<rect id="a" x="0" y="0" width="50" height="50"/>
			<rect id="b" x="100" y="0" width="50" height="50"/>
			<line data-start="a" data-end="b" data-caption="then"/>
		</svg>`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 6 {
		t.Errorf("Count = %d, want 6", result.Count)
	}
	joined := strings.Join(paths, " ")
	for _, want := range []string{"/frames", "/sticky_notes", "/images", "/shapes", "/connectors"} {
		if !strings.Contains(joined, want) {
			t.Errorf("API paths %v missing %s endpoint", paths, want)
		}
	}
	types := map[string]bool{}
	for _, c := range result.Created {
		types[c.Type] = true
	}
	for _, want := range []string{"frame", "sticky_note", "image", "shape", "connector"} {
		if !types[want] {
			t.Errorf("created types %v missing %q", types, want)
		}
	}
}

func TestCreateFromSVG_UnresolvedConnectorReferenceSkipped(t *testing.T) {
	var paths []string
	server := newRoutingCreateServer(&paths)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateFromSVG(context.Background(), CreateFromSVGArgs{
		BoardID: "board1",
		SVG: `<svg>
			<rect id="a" x="0" y="0" width="50" height="50"/>
			<line data-start="a" data-end="ghost"/>
		</svg>`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 1 {
		t.Errorf("Count = %d, want 1 (the rect only)", result.Count)
	}
	if len(result.Skipped) != 1 || !strings.Contains(result.Skipped[0].Reason, "ghost") {
		t.Errorf("skipped = %+v, want unresolved reference naming 'ghost'", result.Skipped)
	}
	if strings.Contains(strings.Join(paths, " "), "/connectors") {
		t.Error("connector endpoint hit despite unresolved reference")
	}
}

// =============================================================================
// Update direction
// =============================================================================

// recordedRequest captures one request the update server saw.
type recordedRequest struct {
	method, path string
	body         map[string]interface{}
}

// newUpdateServer records method, path and JSON body of every request. Paths
// listed in failPaths answer 404.
func newUpdateServer(reqs *[]recordedRequest, failPaths ...string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		*reqs = append(*reqs, recordedRequest{method: r.Method, path: r.URL.Path, body: body})
		w.Header().Set("Content-Type", "application/json")
		for _, fp := range failPaths {
			if strings.HasSuffix(r.URL.Path, fp) {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "not found"})
				return
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "new-1"})
	}))
}

func findRequest(reqs []recordedRequest, method, pathSuffix string) *recordedRequest {
	for i := range reqs {
		if reqs[i].method == method && strings.HasSuffix(reqs[i].path, pathSuffix) {
			return &reqs[i]
		}
	}
	return nil
}

// requirePatchWithKeys asserts a PATCH landed on the item and its body
// carries every listed top-level key.
func requirePatchWithKeys(t *testing.T, reqs []recordedRequest, itemID string, keys ...string) {
	t.Helper()
	patch := findRequest(reqs, http.MethodPatch, "/items/"+itemID)
	if patch == nil {
		t.Fatalf("no PATCH to /items/%s", itemID)
	}
	for _, key := range keys {
		if patch.body[key] == nil {
			t.Errorf("PATCH body for %s = %+v, missing %q", itemID, patch.body, key)
		}
	}
}

func TestUpdateFromSVG_UpdatesGeometryColorAndContent(t *testing.T) {
	var reqs []recordedRequest
	server := newUpdateServer(&reqs)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateFromSVG(context.Background(), UpdateFromSVGArgs{
		BoardID: "board1",
		SVG: `<svg>
			<rect data-miro-id="m1" x="10" y="20" width="100" height="50" fill="#ff0000"/>
			<text data-miro-id="m2" x="5" y="5">renamed</text>
		</svg>`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Updated) != 2 {
		t.Fatalf("result = %+v, want 2 updated", result)
	}
	if len(result.Failed) != 0 {
		t.Fatalf("Failed = %+v, want none", result.Failed)
	}
	requirePatchWithKeys(t, reqs, "m1", "position", "geometry", "style")
	requirePatchWithKeys(t, reqs, "m2", "data")
}

func TestUpdateFromSVG_DeletesAndCreatesAdditively(t *testing.T) {
	var reqs []recordedRequest
	server := newUpdateServer(&reqs)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateFromSVG(context.Background(), UpdateFromSVGArgs{
		BoardID: "board1",
		SVG: `<svg>
			<rect data-miro-id="gone" data-deleted="true" x="0" y="0" width="10" height="10"/>
			<circle cx="5" cy="5" r="5"/>
		</svg>`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Deleted) != 1 || result.Deleted[0] != "gone" {
		t.Errorf("Deleted = %v, want [gone]", result.Deleted)
	}
	if len(result.Created) != 1 {
		t.Errorf("Created = %+v, want the additive circle", result.Created)
	}
	if findRequest(reqs, http.MethodDelete, "/items/gone") == nil {
		t.Error("no DELETE for the data-deleted element")
	}
	if findRequest(reqs, http.MethodPost, "/shapes") == nil {
		t.Error("no POST for the additive circle")
	}
}

func TestUpdateFromSVG_SemanticFailureDoesNotSinkBatch(t *testing.T) {
	var reqs []recordedRequest
	server := newUpdateServer(&reqs, "/items/missing")
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateFromSVG(context.Background(), UpdateFromSVGArgs{
		BoardID: "board1",
		SVG: `<svg>
			<rect data-miro-id="missing" x="0" y="0" width="10" height="10"/>
			<rect data-miro-id="fine" x="0" y="0" width="10" height="10"/>
		</svg>`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Failed) != 1 || result.Failed[0].ID != "missing" {
		t.Errorf("Failed = %+v, want the missing item only", result.Failed)
	}
	if len(result.Updated) != 1 || result.Updated[0].ID != "fine" {
		t.Errorf("Updated = %+v, want the fine item applied", result.Updated)
	}
}

func TestUpdateFromSVG_ConnectorUpdateRefused(t *testing.T) {
	var reqs []recordedRequest
	server := newUpdateServer(&reqs)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateFromSVG(context.Background(), UpdateFromSVGArgs{
		BoardID: "board1",
		SVG:     `<svg><line data-miro-id="c1" data-start="a" data-end="b"/></svg>`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Reason, "miro_update_connector") {
		t.Errorf("Failed = %+v, want connector refusal pointing at miro_update_connector", result.Failed)
	}
}

func TestUpdateFromSVG_MalformedDocumentFailsWhole(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	if _, err := client.UpdateFromSVG(context.Background(), UpdateFromSVGArgs{
		BoardID: "board1", SVG: `<svg><rect data-miro-id="m1" x="0"`,
	}); err == nil {
		t.Error("malformed SVG accepted")
	}
	if _, err := client.UpdateFromSVG(context.Background(), UpdateFromSVGArgs{BoardID: "board1", SVG: "  "}); err == nil {
		t.Error("blank SVG accepted")
	}
}

// =============================================================================
// Frame-scoped read
// =============================================================================

func newFrameScopedServer(children []map[string]interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/frames/") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "f1",
				"data":     map[string]interface{}{"title": "Zone A"},
				"position": map[string]interface{}{"x": 500.0, "y": 400.0},
				"geometry": map[string]interface{}{"width": 800.0, "height": 600.0},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": children, "cursor": ""})
	}))
}

func TestReadBoardSVG_FrameScoped(t *testing.T) {
	server := newFrameScopedServer([]map[string]interface{}{
		{
			"id": "s1", "type": "sticky_note",
			"position": map[string]interface{}{"x": 100.0, "y": 120.0},
			"geometry": map[string]interface{}{"width": 200.0, "height": 200.0},
			"data":     map[string]interface{}{"content": "<p>Child sticky</p>"},
		},
	})
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ReadBoardSVG(context.Background(), ReadBoardSVGArgs{BoardID: "board1", FrameID: "f1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ItemCount != 1 {
		t.Errorf("ItemCount = %d, want 1", result.ItemCount)
	}
	requireSVGContains(t, result.SVG, `data-miro-id="f1"`, "frame outline")
	requireSVGContains(t, result.SVG, `data-miro-id="s1"`, "child sticky")
	requireSVGContains(t, result.SVG, "Child sticky", "child label")
	requireWellFormedXML(t, result.SVG)
	if !strings.Contains(result.Message, "relative to the frame") {
		t.Errorf("Message = %q, want the frame-relative coordinate note", result.Message)
	}
}

func TestUpdateFromSVG_DeleteFailureLandsInFailed(t *testing.T) {
	var reqs []recordedRequest
	// DeleteItem retries a failed delete against the experimental
	// mindmap-node endpoint, so both paths end in "locked" and must 404.
	server := newUpdateServer(&reqs, "locked")
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateFromSVG(context.Background(), UpdateFromSVGArgs{
		BoardID: "board1",
		SVG: `<svg>
			<rect data-miro-id="locked" data-deleted="true" x="0" y="0" width="10" height="10"/>
			<rect data-miro-id="gone" data-deleted="true" x="0" y="0" width="10" height="10"/>
		</svg>`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Failed) != 1 || result.Failed[0].ID != "locked" {
		t.Errorf("Failed = %+v, want the locked item only", result.Failed)
	}
	if len(result.Deleted) != 1 || result.Deleted[0] != "gone" {
		t.Errorf("Deleted = %v, want [gone] still applied", result.Deleted)
	}
}

func TestUpdateFromSVG_PartialGeometryFailsThatItem(t *testing.T) {
	var reqs []recordedRequest
	server := newUpdateServer(&reqs)
	defer server.Close()

	// An image parses with height 0 (only href and width are required), so an
	// identified image without height exercises the geometry-as-a-unit rule.
	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateFromSVG(context.Background(), UpdateFromSVGArgs{
		BoardID: "board1",
		SVG:     `<svg><image data-miro-id="m1" href="https://example.com/p.png" x="0" y="0" width="100"/></svg>`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Reason, "full geometry") {
		t.Errorf("Failed = %+v, want geometry-as-a-unit refusal", result.Failed)
	}
	if findRequest(reqs, http.MethodPatch, "/items/m1") != nil {
		t.Error("PATCH sent despite partial geometry")
	}
}

// newPagedFrameServer serves the frame plus two pages of children linked by a
// cursor.
func newPagedFrameServer() *httptest.Server {
	child := func(id string, x float64) map[string]interface{} {
		return map[string]interface{}{
			"id": id, "type": "sticky_note",
			"position": map[string]interface{}{"x": x, "y": 100.0},
			"geometry": map[string]interface{}{"width": 100.0, "height": 100.0},
			"data":     map[string]interface{}{"content": "<p>s</p>"},
		}
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/frames/") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "f1",
				"geometry": map[string]interface{}{"width": 800.0, "height": 600.0},
			})
			return
		}
		if r.URL.Query().Get("cursor") == "page2" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{child("s2", 300.0)}, "cursor": ""})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{child("s1", 100.0)}, "cursor": "page2"})
	}))
}

func TestReadBoardSVG_FrameScopedPaginates(t *testing.T) {
	server := newPagedFrameServer()
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ReadBoardSVG(context.Background(), ReadBoardSVGArgs{BoardID: "board1", FrameID: "f1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ItemCount != 2 {
		t.Errorf("ItemCount = %d, want 2 (both pages collected)", result.ItemCount)
	}
	requireSVGContains(t, result.SVG, `data-miro-id="s1"`, "page-1 child")
	requireSVGContains(t, result.SVG, `data-miro-id="s2"`, "page-2 child")
	if result.Truncated {
		t.Error("Truncated = true, want false when all pages fit")
	}
}

func TestUpdateFromSVG_ElementCapRejects(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("<svg>")
	for range maxSVGCreateElements + 1 {
		sb.WriteString(`<rect x="0" y="0" width="5" height="5"/>`)
	}
	sb.WriteString("</svg>")

	client := newTestClientWithServer("http://unused")
	_, err := client.UpdateFromSVG(context.Background(), UpdateFromSVGArgs{BoardID: "board1", SVG: sb.String()})
	if err == nil || !strings.Contains(err.Error(), "cap") {
		t.Errorf("oversized SVG accepted or wrong error: %v", err)
	}
}

func TestReadBoardSVG_FrameScopedTruncatesAtMaxItems(t *testing.T) {
	server := newPagedFrameServer()
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ReadBoardSVG(context.Background(), ReadBoardSVGArgs{BoardID: "board1", FrameID: "f1", MaxItems: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ItemCount != 1 {
		t.Errorf("ItemCount = %d, want 1 (capped)", result.ItemCount)
	}
	if !result.Truncated {
		t.Error("Truncated = false, want true when more children remain")
	}
}
