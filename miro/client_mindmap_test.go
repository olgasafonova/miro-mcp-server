package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newMindmapTestClient starts a test server with the given handler and returns
// a client pointed at it. The server is closed automatically at test end.
func newMindmapTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

// checkMindmapField fails the test when got differs from want.
func checkMindmapField[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// mindmapNodeWire builds the wire-format JSON for one mindmap node.
func mindmapNodeWire(id, content string, isRoot bool, parentID string) map[string]interface{} {
	node := map[string]interface{}{
		"id": id,
		"data": map[string]interface{}{
			"isRoot": isRoot,
			"nodeView": map[string]interface{}{
				"data": map[string]interface{}{
					"content": content,
				},
			},
		},
	}
	if parentID != "" {
		node["parent"] = map[string]interface{}{"id": parentID}
	}
	return node
}

// mindmapListPayload builds the wire-format list response for mindmap nodes.
func mindmapListPayload(cursor string, nodes ...map[string]interface{}) map[string]interface{} {
	items := make([]interface{}, 0, len(nodes))
	for _, n := range nodes {
		items = append(items, n)
	}
	return map[string]interface{}{
		"data":   items,
		"cursor": cursor,
	}
}

func TestCreateMindmapNode_Success(t *testing.T) {
	client := newMindmapTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkMindmapField(t, "method", r.Method, http.MethodPost)
		checkMindmapField(t, "path has /mindmap_nodes suffix", strings.HasSuffix(r.URL.Path, "/mindmap_nodes"), true)
		writeJSON(w, map[string]interface{}{
			"id": "mindmap123",
			"data": map[string]interface{}{
				"content": "Root Node",
			},
		})
	})

	result, err := client.CreateMindmapNode(context.Background(), CreateMindmapNodeArgs{
		BoardID: "board123",
		Content: "Root Node",
		X:       100,
		Y:       200,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkMindmapField(t, "ID", result.ID, "mindmap123")
	checkMindmapField(t, "Content", result.Content, "Root Node")
}

func TestCreateMindmapNode_WithParent(t *testing.T) {
	client := newMindmapTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		parent, ok := body["parent"].(map[string]interface{})
		if !ok {
			t.Error("expected parent in request body")
		}
		checkMindmapField(t, "parent id", parent["id"], interface{}("parent123"))
		writeJSON(w, map[string]interface{}{
			"id": "child123",
			"data": map[string]interface{}{
				"content": "Child Node",
			},
			"parent": map[string]interface{}{
				"id": "parent123",
			},
		})
	})

	result, err := client.CreateMindmapNode(context.Background(), CreateMindmapNodeArgs{
		BoardID:  "board123",
		Content:  "Child Node",
		ParentID: "parent123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkMindmapField(t, "ParentID", result.ParentID, "parent123")
}

// TestMindmapNode_ValidationErrors covers input validation across the mindmap
// CRUD methods; every case must fail with a message naming the missing field.
func TestMindmapNode_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")
	ctx := context.Background()

	tests := []struct {
		name    string
		call    func() error
		wantErr string
	}{
		{
			name: "create with empty board ID",
			call: func() error {
				_, err := client.CreateMindmapNode(ctx, CreateMindmapNodeArgs{Content: "Test"})
				return err
			},
			wantErr: "board_id is required",
		},
		{
			name: "create with empty content",
			call: func() error {
				_, err := client.CreateMindmapNode(ctx, CreateMindmapNodeArgs{BoardID: "board123"})
				return err
			},
			wantErr: "content is required",
		},
		{
			name: "get with empty board ID",
			call: func() error {
				_, err := client.GetMindmapNode(ctx, GetMindmapNodeArgs{NodeID: "node123"})
				return err
			},
			wantErr: "board_id is required",
		},
		{
			name: "get with empty node ID",
			call: func() error {
				_, err := client.GetMindmapNode(ctx, GetMindmapNodeArgs{BoardID: "board123"})
				return err
			},
			wantErr: "node_id is required",
		},
		{
			name: "delete with empty board ID",
			call: func() error {
				_, err := client.DeleteMindmapNode(ctx, DeleteMindmapNodeArgs{NodeID: "node123"})
				return err
			},
			wantErr: "board_id is required",
		},
		{
			name: "delete with empty node ID",
			call: func() error {
				_, err := client.DeleteMindmapNode(ctx, DeleteMindmapNodeArgs{BoardID: "board123"})
				return err
			},
			wantErr: "node_id is required",
		},
		{
			name: "list with empty board ID",
			call: func() error {
				_, err := client.ListMindmapNodes(ctx, ListMindmapNodesArgs{})
				return err
			},
			wantErr: "board_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestMindmapNode_EmptyArgs exercises the same validation paths on a client
// built directly from config, without a test server behind it.
func TestMindmapNode_EmptyArgs(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "list with empty board_id",
			call: func() error {
				_, err := client.ListMindmapNodes(ctx, ListMindmapNodesArgs{BoardID: ""})
				return err
			},
		},
		{
			name: "create with empty content",
			call: func() error {
				_, err := client.CreateMindmapNode(ctx, CreateMindmapNodeArgs{BoardID: "board123", Content: ""})
				return err
			},
		},
		{
			name: "delete with empty board_id",
			call: func() error {
				_, err := client.DeleteMindmapNode(ctx, DeleteMindmapNodeArgs{BoardID: "", NodeID: "node123"})
				return err
			},
		},
		{
			name: "delete with empty node_id",
			call: func() error {
				_, err := client.DeleteMindmapNode(ctx, DeleteMindmapNodeArgs{BoardID: "board123", NodeID: ""})
				return err
			},
		},
		{
			name: "get with empty board_id",
			call: func() error {
				_, err := client.GetMindmapNode(ctx, GetMindmapNodeArgs{BoardID: "", NodeID: "node123"})
				return err
			},
		},
		{
			name: "get with empty node_id",
			call: func() error {
				_, err := client.GetMindmapNode(ctx, GetMindmapNodeArgs{BoardID: "board123", NodeID: ""})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil {
				t.Error("expected error for empty argument")
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
	client := newMindmapTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkMindmapField(t, "method", r.Method, http.MethodGet)
		checkMindmapField(t, "path contains /mindmap_nodes/", strings.Contains(r.URL.Path, "/mindmap_nodes/"), true)
		writeJSON(w, map[string]interface{}{
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
	})

	result, err := client.GetMindmapNode(context.Background(), GetMindmapNodeArgs{
		BoardID: "board123",
		NodeID:  "node123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkMindmapField(t, "ID", result.ID, "node123")
	checkMindmapField(t, "Content", result.Content, "Root Node")
	checkMindmapField(t, "IsRoot", result.IsRoot, true)
	checkMindmapField(t, "ChildIDs count", len(result.ChildIDs), 2)
	checkMindmapField(t, "X", result.X, 100.0)
}

func TestListMindmapNodes_Success(t *testing.T) {
	client := newMindmapTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkMindmapField(t, "method", r.Method, http.MethodGet)
		checkMindmapField(t, "path contains /mindmap_nodes", strings.Contains(r.URL.Path, "/mindmap_nodes"), true)
		writeJSON(w, mindmapListPayload("",
			mindmapNodeWire("node1", "Root", true, ""),
			mindmapNodeWire("node2", "Child", false, "node1"),
		))
	})

	result, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkMindmapField(t, "Count", result.Count, 2)
	checkMindmapField(t, "Nodes count", len(result.Nodes), 2)
	checkMindmapField(t, "Nodes[0].ID", result.Nodes[0].ID, "node1")
	checkMindmapField(t, "Nodes[0].IsRoot", result.Nodes[0].IsRoot, true)
	checkMindmapField(t, "Nodes[1].ParentID", result.Nodes[1].ParentID, "node1")
}

func TestListMindmapNodes_WithPagination(t *testing.T) {
	client := newMindmapTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, mindmapListPayload("next_page",
			mindmapNodeWire("node1", "Node", true, ""),
		))
	})

	result, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkMindmapField(t, "HasMore", result.HasMore, true)
}

func TestDeleteMindmapNode_Success(t *testing.T) {
	client := newMindmapTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkMindmapField(t, "method", r.Method, http.MethodDelete)
		checkMindmapField(t, "path contains /mindmap_nodes/", strings.Contains(r.URL.Path, "/mindmap_nodes/"), true)
		w.WriteHeader(http.StatusNoContent)
	})

	result, err := client.DeleteMindmapNode(context.Background(), DeleteMindmapNodeArgs{
		BoardID: "board123",
		NodeID:  "node123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkMindmapField(t, "Success", result.Success, true)
	checkMindmapField(t, "ID", result.ID, "node123")
}

func TestCreateMindmapNode_WithPosition(t *testing.T) {
	// Tests CreateMindmapNode with x, y position (root node)
	client := newMindmapTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Root node should have position
		if _, ok := body["position"]; !ok {
			t.Error("expected 'position' field for root node")
		}
		writeJSON(w, map[string]interface{}{
			"id":   "node123",
			"type": "mindmap_node",
		})
	})

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

// TestListMindmapNodes_QueryParams verifies that pagination arguments reach
// the API as query-string parameters.
func TestListMindmapNodes_QueryParams(t *testing.T) {
	tests := []struct {
		name      string
		args      ListMindmapNodesArgs
		wantQuery string
	}{
		{
			name:      "cursor is forwarded",
			args:      ListMindmapNodesArgs{BoardID: "board123", Cursor: "abc123"},
			wantQuery: "cursor=abc123",
		},
		{
			name:      "cursor pagination",
			args:      ListMindmapNodesArgs{BoardID: "board123", Cursor: "abc123"},
			wantQuery: "cursor=abc123",
		},
		{
			name:      "limit clamped to 50 (endpoint max; 51+ is a live 400)",
			args:      ListMindmapNodesArgs{BoardID: "board123", Limit: 500},
			wantQuery: "limit=50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newMindmapTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.RawQuery, tt.wantQuery) {
					t.Errorf("expected %q in query string, got: %s", tt.wantQuery, r.URL.RawQuery)
				}
				writeJSON(w, mindmapListPayload(""))
			})

			if _, err := client.ListMindmapNodes(context.Background(), tt.args); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestListMindmapNodes_JSONParseError(t *testing.T) {
	client := newMindmapTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	})

	_, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
	})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestListMindmapNodes_WithParentNode(t *testing.T) {
	client := newMindmapTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, mindmapListPayload("next_cursor",
			mindmapNodeWire("node1", "Root Node", true, ""),
			mindmapNodeWire("node2", "Child Node", false, "node1"),
		))
	})

	result, err := client.ListMindmapNodes(context.Background(), ListMindmapNodesArgs{
		BoardID: "board123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkMindmapField(t, "Count", result.Count, 2)
	checkMindmapField(t, "HasMore", result.HasMore, true)
	checkMindmapField(t, "Nodes[1].ParentID", result.Nodes[1].ParentID, "node1")
}

func TestDeleteItem_FallsBackToMindmapNodeOn404(t *testing.T) {
	var calls []string
	client := newMindmapTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
	})

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
	client := newMindmapTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
		checkMindmapField(t, "position.x", pos["x"], interface{}(float64(150)))
		checkMindmapField(t, "position.y", pos["y"], interface{}(float64(250)))
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
	})

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
