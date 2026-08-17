package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newItemTestClient starts a test server with the given handler and returns a
// client pointed at it. The server is closed automatically at test end.
func newItemTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

// checkItemField fails the test when got differs from want.
func checkItemField[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func TestValidateItemID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid", "item123", false},
		{"empty", "", true},
		{"invalid chars", "item/123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateItemID(tt.id)
			checkItemField(t, "ValidateItemID("+tt.id+") error presence", err != nil, tt.wantErr)
		})
	}
}

func TestListItems_Success(t *testing.T) {
	client := newItemTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkItemField(t, "method", r.Method, http.MethodGet)
		checkItemField(t, "path prefix", strings.HasPrefix(r.URL.Path, "/boards/board123/items"), true)
		writeJSON(w, map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":       "item1",
					"type":     "sticky_note",
					"position": map[string]interface{}{"x": 100.0, "y": 200.0},
					"data":     map[string]interface{}{"content": "Test sticky"},
				},
				{
					"id":       "item2",
					"type":     "shape",
					"position": map[string]interface{}{"x": 300.0, "y": 400.0},
					"data":     map[string]interface{}{"content": "Test shape"},
				},
			},
			"cursor": "next-page-cursor",
			"size":   2,
		})
	})

	result, err := client.ListItems(context.Background(), ListItemsArgs{
		BoardID: "board123",
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkItemField(t, "Count", result.Count, 2)
	checkItemField(t, "HasMore", result.HasMore, true)
	checkItemField(t, "first item ID", result.Items[0].ID, "item1")
	checkItemField(t, "first item type", result.Items[0].Type, "sticky_note")
}

func TestListItems_WithTypeFilter(t *testing.T) {
	client := newItemTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkItemField(t, "type filter", r.URL.Query().Get("type"), "sticky_note")
		writeJSON(w, map[string]interface{}{
			"data": []map[string]interface{}{},
			"size": 0,
		})
	})

	_, err := client.ListItems(context.Background(), ListItemsArgs{
		BoardID: "board123",
		Type:    "sticky_note",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetItem_Success(t *testing.T) {
	client := newItemTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkItemField(t, "path", r.URL.Path, "/boards/board123/items/item456")
		writeJSON(w, map[string]interface{}{
			"id":         "item456",
			"type":       "sticky_note",
			"position":   map[string]interface{}{"x": 150.0, "y": 250.0},
			"geometry":   map[string]interface{}{"width": 200.0, "height": 200.0},
			"data":       map[string]interface{}{"content": "Detailed content"},
			"style":      map[string]interface{}{"fillColor": "light_yellow"},
			"createdAt":  "2024-01-01T00:00:00Z",
			"modifiedAt": "2024-01-02T00:00:00Z",
			"createdBy":  map[string]interface{}{"name": "John Doe"},
		})
	})

	result, err := client.GetItem(context.Background(), GetItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkItemField(t, "ID", result.ID, "item456")
	checkItemField(t, "Content", result.Content, "Detailed content")
	checkItemField(t, "X", result.X, 150.0)
	checkItemField(t, "Width", result.Width, 200.0)
	checkItemField(t, "CreatedBy", result.CreatedBy, "John Doe")
}

// TestItem_ValidationErrors covers input validation across the generic item
// methods; every case must fail with a message naming the missing field.
func TestItem_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	ctx := context.Background()

	tests := []struct {
		name    string
		call    func() error
		errText string
	}{
		{"get with empty board_id", func() error { _, err := client.GetItem(ctx, GetItemArgs{ItemID: "item123"}); return err }, "board_id is required"},
		{"get with empty item_id", func() error { _, err := client.GetItem(ctx, GetItemArgs{BoardID: "board123"}); return err }, "item_id is required"},
		{"delete with empty board_id", func() error { _, err := client.DeleteItem(ctx, DeleteItemArgs{ItemID: "item123"}); return err }, "board_id is required"},
		{"delete with empty item_id", func() error { _, err := client.DeleteItem(ctx, DeleteItemArgs{BoardID: "board123"}); return err }, "item_id is required"},
		{"list with empty board_id", func() error { _, err := client.ListItems(ctx, ListItemsArgs{}); return err }, "board_id is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestUpdateItem_Success(t *testing.T) {
	client := newItemTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkItemField(t, "method", r.Method, http.MethodPatch)
		checkItemField(t, "path", r.URL.Path, "/boards/board123/items/item456")

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify the update body structure
		if _, ok := body["data"]; !ok {
			t.Error("expected 'data' field in request body")
		}
		writeJSON(w, map[string]interface{}{"id": "item456"})
	})

	content := "Updated content"
	result, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
		Content: &content,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkItemField(t, "Success", result.Success, true)
	checkItemField(t, "ItemID", result.ItemID, "item456")
}

func TestUpdateItem_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
		// No changes specified
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkItemField(t, "Success", result.Success, true)
	checkItemField(t, "Message", result.Message, "No changes specified")
}

// TestUpdateItem_PartialFields verifies that position and geometry updates
// place the expected numeric values in the request body.
func TestUpdateItem_PartialFields(t *testing.T) {
	x := 500.0
	y := float64(300)
	width := float64(400)
	height := float64(250)

	tests := []struct {
		name      string
		args      UpdateItemArgs
		bodyField string
		want      map[string]float64
	}{
		{
			name:      "x position update",
			args:      UpdateItemArgs{BoardID: "board123", ItemID: "item456", X: &x},
			bodyField: "position",
			want:      map[string]float64{"x": 500.0},
		},
		{
			name:      "y position update",
			args:      UpdateItemArgs{BoardID: "board123", ItemID: "item456", Y: &y},
			bodyField: "position",
			want:      map[string]float64{"y": 300},
		},
		{
			name:      "geometry update",
			args:      UpdateItemArgs{BoardID: "board123", ItemID: "item456", Width: &width, Height: &height},
			bodyField: "geometry",
			want:      map[string]float64{"width": 400, "height": 250},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newItemTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				var body map[string]interface{}
				json.NewDecoder(r.Body).Decode(&body)

				field, ok := body[tt.bodyField].(map[string]interface{})
				if !ok {
					t.Errorf("expected %q field in request body", tt.bodyField)
				}
				for key, want := range tt.want {
					checkItemField(t, tt.bodyField+"."+key, field[key], interface{}(want))
				}
				writeJSON(w, map[string]interface{}{"id": "item456"})
			})

			if _, err := client.UpdateItem(context.Background(), tt.args); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateItem_WithColorAndParent(t *testing.T) {
	client := newItemTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify style: "green" is normalized to its hex equivalent before being sent
		// to Miro because the items endpoint requires hex (regression from 26-04-2026).
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected 'style' field in request body")
		} else {
			checkItemField(t, "fillColor (green normalized to hex)", style["fillColor"], interface{}("#008000"))
		}

		// Verify parent
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected 'parent' field in request body")
		} else {
			checkItemField(t, "parent.id", parent["id"], interface{}("frame-123"))
		}
		writeJSON(w, map[string]interface{}{"id": "item456"})
	})

	color := "green"
	parentID := "frame-123"
	_, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID:  "board123",
		ItemID:   "item456",
		Color:    &color,
		ParentID: &parentID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteItem_Success(t *testing.T) {
	client := newItemTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkItemField(t, "method", r.Method, http.MethodDelete)
		checkItemField(t, "path", r.URL.Path, "/boards/board123/items/item456")
		w.WriteHeader(http.StatusNoContent)
	})

	result, err := client.DeleteItem(context.Background(), DeleteItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkItemField(t, "Success", result.Success, true)
	checkItemField(t, "ItemID", result.ItemID, "item456")
}

func TestListAllItems_Success(t *testing.T) {
	callCount := 0
	client := newItemTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			writeJSON(w, map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "item1", "type": "sticky_note"},
					{"id": "item2", "type": "sticky_note"},
				},
				"cursor": "page2",
			})
			return
		}
		writeJSON(w, map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "item3", "type": "shape"},
			},
			"cursor": "",
		})
	})

	result, err := client.ListAllItems(context.Background(), ListAllItemsArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkItemField(t, "Count", result.Count, 3)
	checkItemField(t, "TotalPages", result.TotalPages, 2)
}

func TestListAllItems_MaxItemsLimit(t *testing.T) {
	client := newItemTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Return more items than max
		items := make([]map[string]interface{}, 10)
		for i := 0; i < 10; i++ {
			items[i] = map[string]interface{}{
				"id":   fmt.Sprintf("item%d", i),
				"type": "sticky_note",
			}
		}
		writeJSON(w, map[string]interface{}{
			"data":   items,
			"cursor": "more",
		})
	})

	result, err := client.ListAllItems(context.Background(), ListAllItemsArgs{
		BoardID:  "board123",
		MaxItems: 5,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkItemField(t, "Count (max items limit)", result.Count, 5)
	checkItemField(t, "Truncated", result.Truncated, true)
}

func TestGetItem_WithLinks(t *testing.T) {
	// Tests GetItem when item has links
	client := newItemTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"id":   "item123",
			"type": "sticky_note",
			"links": map[string]interface{}{
				"self":    "https://api.miro.com/v2/boards/board123/items/item123",
				"related": "https://api.miro.com/v2/boards/board123",
			},
		})
	})

	result, err := client.GetItem(context.Background(), GetItemArgs{
		BoardID: "board123",
		ItemID:  "item123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkItemField(t, "ID", result.ID, "item123")
}

func TestValidateItemID_TooLong(t *testing.T) {
	// Create an ID that exceeds max length (256 chars)
	longID := strings.Repeat("a", 300)
	err := ValidateItemID(longID)

	if err == nil {
		t.Error("expected error for ID that's too long")
	}
	checkItemField(t, "error mentions 'too long'", strings.Contains(err.Error(), "too long"), true)
}

func TestValidateItemID_InvalidCharacters(t *testing.T) {
	// Test ID with invalid characters
	err := ValidateItemID("item<>123")

	if err == nil {
		t.Error("expected error for ID with invalid characters")
	}
	checkItemField(t, "error mentions 'invalid characters'", strings.Contains(err.Error(), "invalid characters"), true)
}

func TestDeleteItem_NoFallbackOn500(t *testing.T) {
	var calls []string
	client := newItemTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := client.DeleteItem(context.Background(), DeleteItemArgs{
		BoardID: "board123",
		ItemID:  "item-xyz",
	})
	if err == nil {
		t.Fatal("expected error on 500")
	}
	// Should NOT fall back on 5xx (and our HTTP layer may retry the same
	// endpoint for 5xx, which is fine — the assertion is that we never call
	// /mindmap_nodes/ for a non-4xx failure).
	for _, c := range calls {
		if strings.Contains(c, "/mindmap_nodes/") {
			t.Errorf("did not expect mindmap_nodes call on 500, got: %v", calls)
		}
	}
}

func TestBulkUpdate_NoTypeFallsBackToGenericItemsEndpoint(t *testing.T) {
	var paths []string
	client := newItemTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		writeJSON(w, map[string]interface{}{"id": "item1"})
	})

	content := "renamed"
	_, err := client.BulkUpdate(context.Background(), BulkUpdateArgs{
		BoardID: "board123",
		Items: []BulkUpdateItem{{
			ItemID:  "item1",
			Content: &content,
			// Type intentionally omitted -> generic /items endpoint
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 1 || !strings.Contains(paths[0], "/items/item1") {
		t.Errorf("expected PATCH to /items/item1 (generic fallback), got: %v", paths)
	}
}
