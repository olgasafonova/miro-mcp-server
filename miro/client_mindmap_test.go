package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateMindmapNode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/mindmap_nodes") {
			t.Errorf("expected mindmap_nodes path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "mindmap123",
			"data": map[string]interface{}{
				"content": "Root Node",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateMindmapNode(context.Background(), CreateMindmapNodeArgs{
		BoardID: "board123",
		Content: "Root Node",
		X:       100,
		Y:       200,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "mindmap123" {
		t.Errorf("ID = %q, want 'mindmap123'", result.ID)
	}
	if result.Content != "Root Node" {
		t.Errorf("Content = %q, want 'Root Node'", result.Content)
	}
}

func TestCreateMindmapNode_WithParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		parent, ok := body["parent"].(map[string]interface{})
		if !ok {
			t.Error("expected parent in request body")
		}
		if parent["id"] != "parent123" {
			t.Errorf("parent id = %v, want 'parent123'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "child123",
			"data": map[string]interface{}{
				"content": "Child Node",
			},
			"parent": map[string]interface{}{
				"id": "parent123",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateMindmapNode(context.Background(), CreateMindmapNodeArgs{
		BoardID:  "board123",
		Content:  "Child Node",
		ParentID: "parent123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ParentID != "parent123" {
		t.Errorf("ParentID = %q, want 'parent123'", result.ParentID)
	}
}

func TestCreateMindmapNode_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    CreateMindmapNodeArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    CreateMindmapNodeArgs{Content: "Test"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty content",
			args:    CreateMindmapNodeArgs{BoardID: "board123"},
			wantErr: "content is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateMindmapNode(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestTruncateMindmap(t *testing.T) {
	tests := []struct {
		input  string
		max    int
		expect string
	}{
		{"short", 10, "short"},
		{"exactly ten", 10, "exactly..."},
		{"longer string", 10, "longer ..."},
	}

	for _, tt := range tests {
		result := truncateMindmap(tt.input, tt.max)
		if result != tt.expect {
			t.Errorf("truncateMindmap(%q, %d) = %q, want %q", tt.input, tt.max, result, tt.expect)
		}
	}
}

func TestGetMindmapNode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/mindmap_nodes/") {
			t.Errorf("expected mindmap_nodes path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "node123",
			"position": map[string]interface{}{
				"x": 100.0,
				"y": 200.0,
			},
			"data": map[string]interface{}{
				"isRoot": true,
				"nodeView": map[string]interface{}{
					"type": "text",
					"data": map[string]interface{}{
						"content": "Root Node",
					},
				},
			},
			"children": []map[string]interface{}{
				{"id": "child1"},
				{"id": "child2"},
			},
			"createdAt":  "2024-01-01T00:00:00Z",
			"modifiedAt": "2024-01-02T00:00:00Z",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetMindmapNode(context.Background(), GetMindmapNodeArgs{
		BoardID: "board123",
		NodeID:  "node123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "node123" {
		t.Errorf("ID = %q, want 'node123'", result.ID)
	}
	if result.Content != "Root Node" {
		t.Errorf("Content = %q, want 'Root Node'", result.Content)
	}
	if !result.IsRoot {
		t.Error("IsRoot = false, want true")
	}
	if len(result.ChildIDs) != 2 {
		t.Errorf("ChildIDs count = %d, want 2", len(result.ChildIDs))
	}
	if result.X != 100.0 {
		t.Errorf("X = %f, want 100.0", result.X)
	}
}

func TestGetMindmapNode_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    GetMindmapNodeArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    GetMindmapNodeArgs{NodeID: "node123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty node ID",
			args:    GetMindmapNodeArgs{BoardID: "board123"},
			wantErr: "node_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetMindmapNode(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestListMindmapNodes_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/mindmap_nodes") {
			t.Errorf("expected mindmap_nodes path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id": "node1",
					"data": map[string]interface{}{
						"isRoot": true,
						"nodeView": map[string]interface{}{
							"data": map[string]interface{}{
								"content": "Root",
							},
						},
					},
				},
				{
					"id": "node2",
					"data": map[string]interface{}{
						"isRoot": false,
						"nodeView": map[string]interface{}{
							"data": map[string]interface{}{
								"content": "Child",
							},
						},
					},
					"parent": map[string]interface{}{
						"id": "node1",
					},
				},
			},
			"cursor": "",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if len(result.Nodes) != 2 {
		t.Errorf("Nodes count = %d, want 2", len(result.Nodes))
	}
	if result.Nodes[0].ID != "node1" {
		t.Errorf("Nodes[0].ID = %q, want 'node1'", result.Nodes[0].ID)
	}
	if !result.Nodes[0].IsRoot {
		t.Error("Nodes[0].IsRoot = false, want true")
	}
	if result.Nodes[1].ParentID != "node1" {
		t.Errorf("Nodes[1].ParentID = %q, want 'node1'", result.Nodes[1].ParentID)
	}
}

func TestListMindmapNodes_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	_, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("error = %q, want containing 'board_id is required'", err.Error())
	}
}

func TestListMindmapNodes_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id": "node1",
					"data": map[string]interface{}{
						"isRoot": true,
						"nodeView": map[string]interface{}{
							"data": map[string]interface{}{
								"content": "Node",
							},
						},
					},
				},
			},
			"cursor": "next_page",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasMore {
		t.Error("HasMore = false, want true")
	}
}

func TestDeleteMindmapNode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/mindmap_nodes/") {
			t.Errorf("expected mindmap_nodes path, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteMindmapNode(context.Background(), DeleteMindmapNodeArgs{
		BoardID: "board123",
		NodeID:  "node123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success = false, want true")
	}
	if result.ID != "node123" {
		t.Errorf("ID = %q, want 'node123'", result.ID)
	}
}

func TestDeleteMindmapNode_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    DeleteMindmapNodeArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    DeleteMindmapNodeArgs{NodeID: "node123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty node ID",
			args:    DeleteMindmapNodeArgs{BoardID: "board123"},
			wantErr: "node_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.DeleteMindmapNode(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCreateMindmapNode_WithPosition(t *testing.T) {
	// Tests CreateMindmapNode with x, y position (root node)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Root node should have position
		if _, ok := body["position"]; !ok {
			t.Error("expected 'position' field for root node")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "node123",
			"type": "mindmap_node",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateMindmapNode(context.Background(), CreateMindmapNodeArgs{
		BoardID: "board123",
		Content: "Root Node",
		X:       100,
		Y:       200,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListMindmapNodes_WithCursor(t *testing.T) {
	// Tests ListMindmapNodes with cursor pagination
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") != "abc123" {
			t.Errorf("cursor = %v, want 'abc123'", r.URL.Query().Get("cursor"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "node1", "type": "mindmap_node"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
		Cursor:  "abc123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Tests for ListMindmapNodes error paths
func TestListMindmapNodes_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

func TestListMindmapNodes_WithCursorPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "cursor=abc123") {
			t.Errorf("expected cursor in query string, got: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
		Cursor:  "abc123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListMindmapNodes_LimitClampTo100(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "limit=100") {
			t.Errorf("expected limit=100, got: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
		Limit:   500, // Should be clamped to 100
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListMindmapNodes_JSONParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
	})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestListMindmapNodes_WithParentNode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{
					"id": "node1",
					"data": map[string]interface{}{
						"isRoot": true,
						"nodeView": map[string]interface{}{
							"data": map[string]interface{}{
								"content": "Root Node",
							},
						},
					},
				},
				map[string]interface{}{
					"id": "node2",
					"data": map[string]interface{}{
						"isRoot": false,
						"nodeView": map[string]interface{}{
							"data": map[string]interface{}{
								"content": "Child Node",
							},
						},
					},
					"parent": map[string]interface{}{
						"id": "node1",
					},
				},
			},
			"cursor": "next_cursor",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("expected 2 nodes, got: %d", result.Count)
	}
	if !result.HasMore {
		t.Error("expected HasMore=true when cursor is present")
	}
	if result.Nodes[1].ParentID != "node1" {
		t.Errorf("expected parent_id=node1, got: %s", result.Nodes[1].ParentID)
	}
}

// Tests for CreateMindmapNode error paths
func TestCreateMindmapNode_EmptyContent(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.CreateMindmapNode(context.Background(), CreateMindmapNodeArgs{
		BoardID: "board123",
		Content: "",
	})
	if err == nil {
		t.Error("expected error for empty content")
	}
}

// Tests for DeleteMindmapNode error paths
func TestDeleteMindmapNode_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.DeleteMindmapNode(context.Background(), DeleteMindmapNodeArgs{
		BoardID: "",
		NodeID:  "node123",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

func TestDeleteMindmapNode_EmptyNodeID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.DeleteMindmapNode(context.Background(), DeleteMindmapNodeArgs{
		BoardID: "board123",
		NodeID:  "",
	})
	if err == nil {
		t.Error("expected error for empty node_id")
	}
}

// Tests for GetMindmapNode error paths
func TestGetMindmapNode_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.GetMindmapNode(context.Background(), GetMindmapNodeArgs{
		BoardID: "",
		NodeID:  "node123",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

func TestGetMindmapNode_EmptyNodeID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.GetMindmapNode(context.Background(), GetMindmapNodeArgs{
		BoardID: "board123",
		NodeID:  "",
	})
	if err == nil {
		t.Error("expected error for empty node_id")
	}
}

func TestDeleteItem_FallsBackToMindmapNodeOn404(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if strings.Contains(r.URL.Path, "/items/") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Item not found"}`))
			return
		}
		if strings.Contains(r.URL.Path, "/mindmap_nodes/") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Fatalf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteItem(context.Background(), DeleteItemArgs{
		BoardID: "board123",
		ItemID:  "node-abc",
	})
	if err != nil {
		t.Fatalf("unexpected error after fallback: %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, want true after mindmap fallback")
	}
	if len(calls) != 2 {
		t.Errorf("expected 2 API calls (items then mindmap_nodes), got %d: %v", len(calls), calls)
	}
}

func TestCreateMindmapNode_ChildAcceptsPosition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := body["parent"]; !ok {
			t.Error("expected parent in request body for child node")
		}
		pos, ok := body["position"].(map[string]interface{})
		if !ok {
			t.Fatal("expected position in request body even with parent set")
		}
		if pos["x"] != float64(150) {
			t.Errorf("position.x = %v, want 150", pos["x"])
		}
		if pos["y"] != float64(250) {
			t.Errorf("position.y = %v, want 250", pos["y"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "child123",
			"data": map[string]interface{}{
				"isRoot":   false,
				"nodeView": map[string]interface{}{"data": map[string]interface{}{"content": "Child"}},
			},
			"parent": map[string]interface{}{"id": "root123"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateMindmapNode(context.Background(), CreateMindmapNodeArgs{
		BoardID:  "board123",
		Content:  "Child",
		ParentID: "root123",
		X:        150,
		Y:        250,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
