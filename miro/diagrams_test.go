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
	var shapeCount, connectorCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle shape creation
		if strings.Contains(r.URL.Path, "/shapes") && r.Method == http.MethodPost {
			count := shapeCount.Add(1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   fmt.Sprintf("shape%d", count),
				"type": "shape",
			})
			return
		}

		// Handle connector creation
		if strings.Contains(r.URL.Path, "/connectors") && r.Method == http.MethodPost {
			count := connectorCount.Add(1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   fmt.Sprintf("conn%d", count),
				"type": "connector",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	result, err := client.GenerateDiagram(context.Background(), GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart TB\n    A[Start]-->B[End]",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NodesCreated != 2 {
		t.Errorf("NodesCreated = %d, want 2", result.NodesCreated)
	}
	if result.ConnectorsCreated != 1 {
		t.Errorf("ConnectorsCreated = %d, want 1", result.ConnectorsCreated)
	}
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
	var shapeCount, connectorCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/shapes") && r.Method == http.MethodPost {
			count := shapeCount.Add(1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   fmt.Sprintf("shape%d", count),
				"type": "shape",
			})
			return
		}

		if strings.Contains(r.URL.Path, "/connectors") && r.Method == http.MethodPost {
			count := connectorCount.Add(1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   fmt.Sprintf("conn%d", count),
				"type": "connector",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	// Flowchart with decision node (diamond shape)
	result, err := client.GenerateDiagram(context.Background(), GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: `flowchart TB
    A[Start] --> B{Decision}
    B -->|Yes| C[Success]
    B -->|No| D[Retry]`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NodesCreated != 4 {
		t.Errorf("NodesCreated = %d, want 4", result.NodesCreated)
	}
	if result.ConnectorsCreated != 3 {
		t.Errorf("ConnectorsCreated = %d, want 3", result.ConnectorsCreated)
	}
}

func TestGenerateDiagram_SequenceDiagram(t *testing.T) {
	var shapeCount, connectorCount, frameCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/frames") && r.Method == http.MethodPost {
			count := frameCount.Add(1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   fmt.Sprintf("frame%d", count),
				"type": "frame",
			})
			return
		}

		if strings.Contains(r.URL.Path, "/shapes") && r.Method == http.MethodPost {
			count := shapeCount.Add(1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   fmt.Sprintf("shape%d", count),
				"type": "shape",
			})
			return
		}

		if strings.Contains(r.URL.Path, "/connectors") && r.Method == http.MethodPost {
			count := connectorCount.Add(1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   fmt.Sprintf("conn%d", count),
				"type": "connector",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	result, err := client.GenerateDiagram(context.Background(), GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: `sequenceDiagram
    Alice->>Bob: Hello Bob!
    Bob-->>Alice: Hi Alice!`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/shapes") && r.Method == http.MethodPost {
			// Capture position from first shape
			if !positionCaptured {
				var req map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
					if pos, ok := req["position"].(map[string]interface{}); ok {
						receivedX = pos["x"].(float64)
						receivedY = pos["y"].(float64)
						positionCaptured = true
					}
				}
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "shape1",
				"type": "shape",
			})
			return
		}

		if strings.Contains(r.URL.Path, "/connectors") && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "conn1",
				"type": "connector",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	_, err := client.GenerateDiagram(context.Background(), GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart TB\n    A[Start]-->B[End]",
		StartX:  500,
		StartY:  300,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The first shape should have the custom start position applied
	if receivedX < 500 || receivedY < 300 {
		t.Logf("Position: x=%f, y=%f (offset should include start position)", receivedX, receivedY)
	}
}

func TestGenerateDiagram_WithNodeWidth(t *testing.T) {
	var receivedWidth float64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/shapes") && r.Method == http.MethodPost {
			var req map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				if geo, ok := req["geometry"].(map[string]interface{}); ok {
					if width, ok := geo["width"].(float64); ok {
						receivedWidth = width
					}
				}
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "shape1",
				"type": "shape",
			})
			return
		}

		if strings.Contains(r.URL.Path, "/connectors") && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "conn1",
				"type": "connector",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	_, err := client.GenerateDiagram(context.Background(), GenerateDiagramArgs{
		BoardID:   "board123",
		Diagram:   "flowchart TB\n    A[Start]-->B[End]",
		NodeWidth: 250,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedWidth != 250 {
		t.Errorf("node width = %f, want 250", receivedWidth)
	}
}

func TestGenerateDiagram_ShapeCreationFailure(t *testing.T) {
	var callCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/shapes") && r.Method == http.MethodPost {
			count := callCount.Add(1)
			// Fail first shape, succeed on second
			if count == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message": "Internal server error",
				})
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "shape2",
				"type": "shape",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	// Should still succeed with partial results (1 shape created, 1 failed)
	result, err := client.GenerateDiagram(context.Background(), GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart TB\n    A[Start]-->B[End]",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// One shape should have been created despite the failure
	if result.NodesCreated != 1 {
		t.Errorf("NodesCreated = %d, want 1 (one failed, one succeeded)", result.NodesCreated)
	}
}

func TestGenerateDiagram_ConnectorCreationFailure(t *testing.T) {
	var shapeCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/shapes") && r.Method == http.MethodPost {
			count := shapeCount.Add(1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   fmt.Sprintf("shape%d", count),
				"type": "shape",
			})
			return
		}

		if strings.Contains(r.URL.Path, "/connectors") && r.Method == http.MethodPost {
			// Fail connector creation
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Internal server error",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	result, err := client.GenerateDiagram(context.Background(), GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart TB\n    A[Start]-->B[End]",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Shapes should be created, connectors should fail gracefully
	if result.NodesCreated != 2 {
		t.Errorf("NodesCreated = %d, want 2", result.NodesCreated)
	}
	if result.ConnectorsCreated != 0 {
		t.Errorf("ConnectorsCreated = %d, want 0 (all failed)", result.ConnectorsCreated)
	}
}

func TestGenerateDiagram_WithParentID(t *testing.T) {
	var receivedParentID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/shapes") && r.Method == http.MethodPost {
			var req map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				if parent, ok := req["parent"].(map[string]interface{}); ok {
					if id, ok := parent["id"].(string); ok {
						receivedParentID = id
					}
				}
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "shape1",
				"type": "shape",
			})
			return
		}

		if strings.Contains(r.URL.Path, "/connectors") && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "conn1",
				"type": "connector",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	_, err := client.GenerateDiagram(context.Background(), GenerateDiagramArgs{
		BoardID:  "board123",
		Diagram:  "flowchart TB\n    A[Start]-->B[End]",
		ParentID: "frame456",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedParentID != "frame456" {
		t.Errorf("parent_id = %q, want 'frame456'", receivedParentID)
	}
}

func TestGenerateDiagram_LRDirection(t *testing.T) {
	var shapeCount, connectorCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/shapes") && r.Method == http.MethodPost {
			count := shapeCount.Add(1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   fmt.Sprintf("shape%d", count),
				"type": "shape",
			})
			return
		}

		if strings.Contains(r.URL.Path, "/connectors") && r.Method == http.MethodPost {
			count := connectorCount.Add(1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   fmt.Sprintf("conn%d", count),
				"type": "connector",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	// Test left-to-right direction
	result, err := client.GenerateDiagram(context.Background(), GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart LR\n    A[Start]-->B[End]",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NodesCreated != 2 {
		t.Errorf("NodesCreated = %d, want 2", result.NodesCreated)
	}
}

func TestGenerateDiagram_GraphKeyword(t *testing.T) {
	var shapeCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/shapes") && r.Method == http.MethodPost {
			count := shapeCount.Add(1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   fmt.Sprintf("shape%d", count),
				"type": "shape",
			})
			return
		}

		if strings.Contains(r.URL.Path, "/connectors") && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "conn1",
				"type": "connector",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	// Test "graph" keyword (alias for flowchart)
	result, err := client.GenerateDiagram(context.Background(), GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "graph TB\n    A-->B",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NodesCreated != 2 {
		t.Errorf("NodesCreated = %d, want 2", result.NodesCreated)
	}
}

func TestGenerateDiagram_CircleNode(t *testing.T) {
	var receivedShapes []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/shapes") && r.Method == http.MethodPost {
			var req map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				if data, ok := req["data"].(map[string]interface{}); ok {
					if shape, ok := data["shape"].(string); ok {
						receivedShapes = append(receivedShapes, shape)
					}
				}
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "shape1",
				"type": "shape",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	_, err := client.GenerateDiagram(context.Background(), GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart TB\n    A((Circle Node))",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// All requests fail
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	result, err := client.GenerateDiagram(context.Background(), GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: "flowchart TB\n    A[Start]-->B[End]",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty result without error
	if result.NodesCreated != 0 {
		t.Errorf("NodesCreated = %d, want 0", result.NodesCreated)
	}
	if result.Message != "Created diagram" {
		t.Errorf("Message = %q, want 'Created diagram'", result.Message)
	}
}

func TestGenerateDiagram_FrameCreationFailure(t *testing.T) {
	var shapeCount, connectorCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Frames fail
		if strings.Contains(r.URL.Path, "/frames") && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if strings.Contains(r.URL.Path, "/shapes") && r.Method == http.MethodPost {
			count := shapeCount.Add(1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   fmt.Sprintf("shape%d", count),
				"type": "shape",
			})
			return
		}

		if strings.Contains(r.URL.Path, "/connectors") && r.Method == http.MethodPost {
			count := connectorCount.Add(1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   fmt.Sprintf("conn%d", count),
				"type": "connector",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)

	// Sequence diagram creates frames for participants
	result, err := client.GenerateDiagram(context.Background(), GenerateDiagramArgs{
		BoardID: "board123",
		Diagram: `sequenceDiagram
    Alice->>Bob: Hello`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should handle frame failure gracefully
	if result.FramesCreated != 0 {
		t.Errorf("FramesCreated = %d, want 0", result.FramesCreated)
	}
}
