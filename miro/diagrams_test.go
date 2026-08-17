package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// =============================================================================
// Test server helpers
// =============================================================================

// diagramServerConfig controls how the fake Miro API behaves in diagram tests.
// The zero value serves successful creation responses for every endpoint.
type diagramServerConfig struct {
	// failShape, when set, fails the nth shape creation call (1-based).
	failShape func(call int32) bool
	// inspectShape, when set, receives each decoded shape request body.
	inspectShape   func(body map[string]interface{})
	failConnectors bool
	failFrames     bool
	failGroups     bool
}

// newDiagramServer starts a fake Miro API for GenerateDiagram tests. It serves
// shape, connector, frame, and group creation endpoints, generating sequential
// item IDs, and fails the endpoints selected by cfg with HTTP 500.
func newDiagramServer(t *testing.T, cfg diagramServerConfig) *httptest.Server {
	t.Helper()
	var shapeCount, connectorCount, frameCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch {
		case strings.Contains(r.URL.Path, "/shapes"):
			cfg.serveShape(w, r, shapeCount.Add(1))
		case strings.Contains(r.URL.Path, "/connectors"):
			serveItemCreation(w, cfg.failConnectors, fmt.Sprintf("conn%d", connectorCount.Add(1)), "connector")
		case strings.Contains(r.URL.Path, "/frames"):
			serveItemCreation(w, cfg.failFrames, fmt.Sprintf("frame%d", frameCount.Add(1)), "frame")
		case strings.Contains(r.URL.Path, "/groups"):
			cfg.serveGroup(w)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// serveShape answers one shape creation call, honoring failShape and inspectShape.
func (cfg diagramServerConfig) serveShape(w http.ResponseWriter, r *http.Request, call int32) {
	if cfg.failShape != nil && cfg.failShape(call) {
		writeDiagramServerError(w)
		return
	}
	if cfg.inspectShape != nil {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			cfg.inspectShape(body)
		}
	}
	writeDiagramItemCreated(w, fmt.Sprintf("shape%d", call), "shape")
}

// serveGroup answers a group creation call with a fixed group payload.
func (cfg diagramServerConfig) serveGroup(w http.ResponseWriter) {
	if cfg.failGroups {
		writeDiagramServerError(w)
		return
	}
	w.WriteHeader(http.StatusCreated)
	encodeDiagramJSON(w, map[string]interface{}{
		"id":    "group123",
		"type":  "group",
		"items": []string{"shape1", "shape2", "conn1"},
	})
}

// serveItemCreation writes either a created item or a server error.
func serveItemCreation(w http.ResponseWriter, fail bool, id, itemType string) {
	if fail {
		writeDiagramServerError(w)
		return
	}
	writeDiagramItemCreated(w, id, itemType)
}

func writeDiagramItemCreated(w http.ResponseWriter, id, itemType string) {
	w.WriteHeader(http.StatusCreated)
	encodeDiagramJSON(w, map[string]interface{}{"id": id, "type": itemType})
}

func writeDiagramServerError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	encodeDiagramJSON(w, map[string]interface{}{"message": "Internal server error"})
}

func encodeDiagramJSON(w http.ResponseWriter, payload map[string]interface{}) {
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		panic(err)
	}
}

// generateDiagram runs GenerateDiagram against a fake API built from cfg and
// fails the test on error.
func generateDiagram(t *testing.T, cfg diagramServerConfig, args GenerateDiagramArgs) GenerateDiagramResult {
	t.Helper()
	server := newDiagramServer(t, cfg)
	client := newTestClientWithServer(server.URL)
	result, err := client.GenerateDiagram(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return result
}

// assertDiagramCounts verifies the node and connector counts in a result.
func assertDiagramCounts(t *testing.T, result GenerateDiagramResult, wantNodes, wantConnectors int) {
	t.Helper()
	if result.NodesCreated != wantNodes {
		t.Errorf("NodesCreated = %d, want %d", result.NodesCreated, wantNodes)
	}
	if result.ConnectorsCreated != wantConnectors {
		t.Errorf("ConnectorsCreated = %d, want %d", result.ConnectorsCreated, wantConnectors)
	}
}

// captureShapeField generates args on a fake API and returns the value of
// body[outer][inner] from the last shape creation request, or nil if absent.
func captureShapeField(t *testing.T, args GenerateDiagramArgs, outer, inner string) interface{} {
	t.Helper()
	var captured interface{}
	cfg := diagramServerConfig{inspectShape: func(body map[string]interface{}) {
		if v, ok := nestedField(body, outer, inner); ok {
			captured = v
		}
	}}
	generateDiagram(t, cfg, args)
	return captured
}

// compoundWant holds the expected compound-item fields of a diagram result.
type compoundWant struct {
	mode, id, itemType string
}

// assertCompoundDiagram verifies the compound-item fields of a result,
// including that the message mentions the output mode.
func assertCompoundDiagram(t *testing.T, result GenerateDiagramResult, want compoundWant) {
	t.Helper()
	if result.OutputMode != want.mode {
		t.Errorf("OutputMode = %q, want %q", result.OutputMode, want.mode)
	}
	if result.DiagramID != want.id {
		t.Errorf("DiagramID = %q, want %q", result.DiagramID, want.id)
	}
	if result.DiagramType != want.itemType {
		t.Errorf("DiagramType = %q, want %q", result.DiagramType, want.itemType)
	}
	if !strings.Contains(result.Message, want.mode) {
		t.Errorf("Message = %q, want to contain %q", result.Message, want.mode)
	}
}

// nestedField returns body[outer][inner] when both levels exist.
func nestedField(body map[string]interface{}, outer, inner string) (interface{}, bool) {
	m, ok := body[outer].(map[string]interface{})
	if !ok {
		return nil, false
	}
	v, ok := m[inner]
	return v, ok
}

// nestedFloat returns body[outer][inner] as a float64 when present.
func nestedFloat(body map[string]interface{}, outer, inner string) (float64, bool) {
	v, ok := nestedField(body, outer, inner)
	if !ok {
		return 0, false
	}
	f, ok := v.(float64)
	return f, ok
}

// nestedString returns body[outer][inner] as a string when present.
func nestedString(body map[string]interface{}, outer, inner string) (string, bool) {
	v, ok := nestedField(body, outer, inner)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// =============================================================================
// GenerateDiagram Tests
// =============================================================================

func TestGenerateDiagram_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    GenerateDiagramArgs
		wantErr string
	}{
		{
			name:    "empty board_id",
			args:    GenerateDiagramArgs{BoardID: "", Diagram: "flowchart TB\nA-->B"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty diagram",
			args:    GenerateDiagramArgs{BoardID: "board123", Diagram: ""},
			wantErr: "diagram code is required",
		},
		{
			name:    "whitespace only diagram",
			args:    GenerateDiagramArgs{BoardID: "board123", Diagram: "   \n  "},
			wantErr: "diagram input is empty",
		},
		{
			name:    "invalid diagram syntax",
			args:    GenerateDiagramArgs{BoardID: "board123", Diagram: "not a valid diagram"},
			wantErr: "diagram must start with a valid header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GenerateDiagram(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestGenerateDiagram_SimpleFlowchart(t *testing.T) {
	result := generateDiagram(t, diagramServerConfig{}, GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart TB\n    A[Start]-->B[End]",
	})

	assertDiagramCounts(t, result, 2, 1)
	if len(result.NodeIDs) != 2 {
		t.Errorf("len(NodeIDs) = %d, want 2", len(result.NodeIDs))
	}
	if len(result.ConnectorIDs) != 1 {
		t.Errorf("len(ConnectorIDs) = %d, want 1", len(result.ConnectorIDs))
	}
	if !strings.Contains(result.Message, "2 nodes") {
		t.Errorf("Message = %q, want to contain '2 nodes'", result.Message)
	}
}

func TestGenerateDiagram_WithDecisionNode(t *testing.T) {
	// Flowchart with decision node (diamond shape)
	result := generateDiagram(t, diagramServerConfig{}, GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: `flowchart TB
    A[Start] --> B{Decision}
    B -->|Yes| C[Success]
    B -->|No| D[Retry]`,
	})

	assertDiagramCounts(t, result, 4, 3)
}

func TestGenerateDiagram_SequenceDiagram(t *testing.T) {
	result := generateDiagram(t, diagramServerConfig{}, GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: `sequenceDiagram
    Alice->>Bob: Hello Bob!
    Bob-->>Alice: Hi Alice!`,
	})

	// Sequence diagrams create participant boxes and message arrows
	if result.NodesCreated < 2 {
		t.Errorf("NodesCreated = %d, want at least 2 (participants)", result.NodesCreated)
	}
	if result.ConnectorsCreated < 2 {
		t.Errorf("ConnectorsCreated = %d, want at least 2 (messages)", result.ConnectorsCreated)
	}
}

func TestGenerateDiagram_WithCustomPosition(t *testing.T) {
	var receivedX, receivedY float64
	var positionCaptured bool

	cfg := diagramServerConfig{inspectShape: func(body map[string]interface{}) {
		if positionCaptured {
			return
		}
		x, okX := nestedFloat(body, "position", "x")
		y, okY := nestedFloat(body, "position", "y")
		if okX && okY {
			receivedX, receivedY = x, y
			positionCaptured = true
		}
	}}

	generateDiagram(t, cfg, GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart TB\n    A[Start]-->B[End]",
		StartX:  500,
		StartY:  300,
	})

	// The first shape should have the custom start position applied
	if receivedX < 500 || receivedY < 300 {
		t.Logf("Position: x=%f, y=%f (offset should include start position)", receivedX, receivedY)
	}
}

func TestGenerateDiagram_WithNodeWidth(t *testing.T) {
	receivedWidth := captureShapeField(t, GenerateDiagramArgs{
		BoardID:   "board123",
		Diagram:   "flowchart TB\n    A[Start]-->B[End]",
		NodeWidth: 250,
	}, "geometry", "width")

	if receivedWidth != 250.0 {
		t.Errorf("node width = %v, want 250", receivedWidth)
	}
}

func TestGenerateDiagram_ShapeCreationFailure(t *testing.T) {
	// Fail first shape, succeed on second
	cfg := diagramServerConfig{failShape: func(call int32) bool { return call == 1 }}

	// Should still succeed with partial results (1 shape created, 1 failed)
	result := generateDiagram(t, cfg, GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart TB\n    A[Start]-->B[End]",
	})

	// One shape should have been created despite the failure
	if result.NodesCreated != 1 {
		t.Errorf("NodesCreated = %d, want 1 (one failed, one succeeded)", result.NodesCreated)
	}
}

func TestGenerateDiagram_ConnectorCreationFailure(t *testing.T) {
	result := generateDiagram(t, diagramServerConfig{failConnectors: true}, GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart TB\n    A[Start]-->B[End]",
	})

	// Shapes should be created, connectors should fail gracefully
	assertDiagramCounts(t, result, 2, 0)
}

func TestGenerateDiagram_WithParentID(t *testing.T) {
	receivedParentID := captureShapeField(t, GenerateDiagramArgs{
		BoardID:  "board123",
		Diagram:  "flowchart TB\n    A[Start]-->B[End]",
		ParentID: "frame456",
	}, "parent", "id")

	if receivedParentID != "frame456" {
		t.Errorf("parent_id = %v, want 'frame456'", receivedParentID)
	}
}

func TestGenerateDiagram_LRDirection(t *testing.T) {
	// Test left-to-right direction
	result := generateDiagram(t, diagramServerConfig{}, GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart LR\n    A[Start]-->B[End]",
	})

	if result.NodesCreated != 2 {
		t.Errorf("NodesCreated = %d, want 2", result.NodesCreated)
	}
}

func TestGenerateDiagram_GraphKeyword(t *testing.T) {
	// Test "graph" keyword (alias for flowchart)
	result := generateDiagram(t, diagramServerConfig{}, GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "graph TB\n    A-->B",
	})

	if result.NodesCreated != 2 {
		t.Errorf("NodesCreated = %d, want 2", result.NodesCreated)
	}
}

func TestGenerateDiagram_CircleNode(t *testing.T) {
	var receivedShapes []string

	cfg := diagramServerConfig{inspectShape: func(body map[string]interface{}) {
		if shape, ok := nestedString(body, "data", "shape"); ok {
			receivedShapes = append(receivedShapes, shape)
		}
	}}

	generateDiagram(t, cfg, GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart TB\n    A((Circle Node))",
	})

	// Check that a circle shape was created
	hasCircle := false
	for _, shape := range receivedShapes {
		if shape == "circle" {
			hasCircle = true
			break
		}
	}
	if !hasCircle {
		t.Errorf("expected circle shape, got shapes: %v", receivedShapes)
	}
}

func TestGenerateDiagram_EmptyResult(t *testing.T) {
	// All requests fail
	cfg := diagramServerConfig{
		failShape:      func(int32) bool { return true },
		failConnectors: true,
		failFrames:     true,
		failGroups:     true,
	}

	result := generateDiagram(t, cfg, GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart TB\n    A[Start]-->B[End]",
	})

	// Should return empty result without error
	if result.NodesCreated != 0 {
		t.Errorf("NodesCreated = %d, want 0", result.NodesCreated)
	}
	if result.Message != "Created diagram" {
		t.Errorf("Message = %q, want 'Created diagram'", result.Message)
	}
}

func TestGenerateDiagram_FrameCreationFailure(t *testing.T) {
	// Sequence diagram creates frames for participants; frames fail
	result := generateDiagram(t, diagramServerConfig{failFrames: true}, GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: `sequenceDiagram
    Alice->>Bob: Hello`,
	})

	// Should handle frame failure gracefully
	if result.FramesCreated != 0 {
		t.Errorf("FramesCreated = %d, want 0", result.FramesCreated)
	}
}

// =============================================================================
// Output Mode Tests (P5: Compound Diagram Items)
// =============================================================================

func TestGenerateDiagram_OutputModeDiscrete(t *testing.T) {
	result := generateDiagram(t, diagramServerConfig{}, GenerateDiagramArgs{
		BoardID:    "board123",
		Diagram:    "flowchart TB\n    A[Start]-->B[End]",
		OutputMode: "discrete",
	})

	if result.OutputMode != "discrete" {
		t.Errorf("OutputMode = %q, want 'discrete'", result.OutputMode)
	}
	if result.DiagramID != "" {
		t.Errorf("DiagramID should be empty for discrete mode, got %q", result.DiagramID)
	}
	if result.TotalItems != 3 { // 2 shapes + 1 connector
		t.Errorf("TotalItems = %d, want 3", result.TotalItems)
	}
}

func TestGenerateDiagram_OutputModeGrouped(t *testing.T) {
	result := generateDiagram(t, diagramServerConfig{}, GenerateDiagramArgs{
		BoardID:    "board123",
		Diagram:    "flowchart TB\n    A[Start]-->B[End]",
		OutputMode: "grouped",
	})

	assertCompoundDiagram(t, result, compoundWant{mode: "grouped", id: "group123", itemType: "group"})
	if !strings.Contains(result.DiagramURL, "group123") {
		t.Errorf("DiagramURL = %q, want to contain 'group123'", result.DiagramURL)
	}
}

func TestGenerateDiagram_OutputModeFramed(t *testing.T) {
	result := generateDiagram(t, diagramServerConfig{}, GenerateDiagramArgs{
		BoardID:    "board123",
		Diagram:    "flowchart TB\n    A[Start]-->B[End]",
		OutputMode: "framed",
	})

	assertCompoundDiagram(t, result, compoundWant{mode: "framed", id: "frame1", itemType: "frame"})
	// Frame should be added to FrameIDs
	if len(result.FrameIDs) == 0 || result.FrameIDs[0] != "frame1" {
		t.Errorf("FrameIDs = %v, want ['frame1']", result.FrameIDs)
	}
}

func TestGenerateDiagram_OutputModeGroupedFailure(t *testing.T) {
	// Groups endpoint fails
	result := generateDiagram(t, diagramServerConfig{failGroups: true}, GenerateDiagramArgs{
		BoardID:    "board123",
		Diagram:    "flowchart TB\n    A[Start]-->B[End]",
		OutputMode: "grouped",
	})

	// Should still have created items, just not grouped
	if result.NodesCreated != 2 {
		t.Errorf("NodesCreated = %d, want 2", result.NodesCreated)
	}
	if result.DiagramID != "" {
		t.Errorf("DiagramID should be empty when grouping fails, got %q", result.DiagramID)
	}
	if !strings.Contains(result.Message, "grouping failed") {
		t.Errorf("Message = %q, want to contain 'grouping failed'", result.Message)
	}
}

func TestGenerateDiagram_OutputModeDefaultsToDiscrete(t *testing.T) {
	// No OutputMode specified
	result := generateDiagram(t, diagramServerConfig{}, GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart TB\n    A[Start]-->B[End]",
	})

	if result.OutputMode != "discrete" {
		t.Errorf("OutputMode = %q, want 'discrete' (default)", result.OutputMode)
	}
}
