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
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateItemID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestListItems_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards/board123/items") {
			t.Errorf("expected /boards/board123/items, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "item1",
					"type": "sticky_note",
					"position": map[string]interface{}{
						"x": 100.0,
						"y": 200.0,
					},
					"data": map[string]interface{}{
						"content": "Test sticky",
					},
				},
				{
					"id":   "item2",
					"type": "shape",
					"position": map[string]interface{}{
						"x": 300.0,
						"y": 400.0,
					},
					"data": map[string]interface{}{
						"content": "Test shape",
					},
				},
			},
			"cursor": "next-page-cursor",
			"size":   2,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListItems(context.Background(), ListItemsArgs{
		BoardID: "board123",
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if !result.HasMore {
		t.Error("HasMore should be true")
	}
	if result.Items[0].ID != "item1" {
		t.Errorf("first item ID = %q, want 'item1'", result.Items[0].ID)
	}
	if result.Items[0].Type != "sticky_note" {
		t.Errorf("first item type = %q, want 'sticky_note'", result.Items[0].Type)
	}
}

func TestListItems_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.ListItems(context.Background(), ListItemsArgs{})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestListItems_WithTypeFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") != "sticky_note" {
			t.Errorf("expected type=sticky_note, got %s", r.URL.Query().Get("type"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
			"size": 0,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListItems(context.Background(), ListItemsArgs{
		BoardID: "board123",
		Type:    "sticky_note",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetItem_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boards/board123/items/item456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "item456",
			"type": "sticky_note",
			"position": map[string]interface{}{
				"x": 150.0,
				"y": 250.0,
			},
			"geometry": map[string]interface{}{
				"width":  200.0,
				"height": 200.0,
			},
			"data": map[string]interface{}{
				"content": "Detailed content",
			},
			"style": map[string]interface{}{
				"fillColor": "light_yellow",
			},
			"createdAt":  "2024-01-01T00:00:00Z",
			"modifiedAt": "2024-01-02T00:00:00Z",
			"createdBy": map[string]interface{}{
				"name": "John Doe",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetItem(context.Background(), GetItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "item456" {
		t.Errorf("ID = %q, want 'item456'", result.ID)
	}
	if result.Content != "Detailed content" {
		t.Errorf("Content = %q, want 'Detailed content'", result.Content)
	}
	if result.X != 150.0 {
		t.Errorf("X = %f, want 150.0", result.X)
	}
	if result.Width != 200.0 {
		t.Errorf("Width = %f, want 200.0", result.Width)
	}
	if result.CreatedBy != "John Doe" {
		t.Errorf("CreatedBy = %q, want 'John Doe'", result.CreatedBy)
	}
}

func TestGetItem_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    GetItemArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    GetItemArgs{ItemID: "item123"},
			errText: "board_id is required",
		},
		{
			name:    "empty item_id",
			args:    GetItemArgs{BoardID: "board123"},
			errText: "item_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetItem(context.Background(), tt.args)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/items/item456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify the update body structure
		if _, ok := body["data"]; !ok {
			t.Error("expected 'data' field in request body")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "item456",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	content := "Updated content"
	result, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
		Content: &content,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.ItemID != "item456" {
		t.Errorf("ItemID = %q, want 'item456'", result.ItemID)
	}
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
	if !result.Success {
		t.Error("Success should be true even with no changes")
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateItem_PositionUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		pos, ok := body["position"].(map[string]interface{})
		if !ok {
			t.Error("expected 'position' field in request body")
		}
		if pos["x"] != 500.0 {
			t.Errorf("x = %v, want 500.0", pos["x"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "item456"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	x := 500.0
	_, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
		X:       &x,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateItem_WithYPosition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		pos, ok := body["position"].(map[string]interface{})
		if !ok {
			t.Error("expected 'position' field in request body")
		}
		if pos["y"] != float64(300) {
			t.Errorf("y = %v, want 300", pos["y"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "item456"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	y := float64(300)
	_, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
		Y:       &y,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateItem_WithGeometry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		geom, ok := body["geometry"].(map[string]interface{})
		if !ok {
			t.Error("expected 'geometry' field in request body")
		}
		if geom["width"] != float64(400) {
			t.Errorf("width = %v, want 400", geom["width"])
		}
		if geom["height"] != float64(250) {
			t.Errorf("height = %v, want 250", geom["height"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "item456"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	width := float64(400)
	height := float64(250)
	_, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
		Width:   &width,
		Height:  &height,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateItem_WithColorAndParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify style: "green" is normalized to its hex equivalent before being sent
		// to Miro because the items endpoint requires hex (regression from 26-04-2026).
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected 'style' field in request body")
		} else if style["fillColor"] != "#008000" {
			t.Errorf("fillColor = %v, want '#008000' (green normalized to hex)", style["fillColor"])
		}

		// Verify parent
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected 'parent' field in request body")
		} else if parent["id"] != "frame-123" {
			t.Errorf("parent.id = %v, want 'frame-123'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "item456"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/items/item456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteItem(context.Background(), DeleteItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.ItemID != "item456" {
		t.Errorf("ItemID = %q, want 'item456'", result.ItemID)
	}
}

func TestDeleteItem_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    DeleteItemArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    DeleteItemArgs{ItemID: "item123"},
			errText: "board_id is required",
		},
		{
			name:    "empty item_id",
			args:    DeleteItemArgs{BoardID: "board123"},
			errText: "item_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.DeleteItem(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestListAllItems_Success(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "item1", "type": "sticky_note"},
					{"id": "item2", "type": "sticky_note"},
				},
				"cursor": "page2",
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "item3", "type": "shape"},
				},
				"cursor": "",
			})
		}
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListAllItems(context.Background(), ListAllItemsArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 3 {
		t.Errorf("Count = %d, want 3", result.Count)
	}
	if result.TotalPages != 2 {
		t.Errorf("TotalPages = %d, want 2", result.TotalPages)
	}
}

func TestListAllItems_MaxItemsLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return more items than max
		items := make([]map[string]interface{}, 10)
		for i := 0; i < 10; i++ {
			items[i] = map[string]interface{}{
				"id":   fmt.Sprintf("item%d", i),
				"type": "sticky_note",
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":   items,
			"cursor": "more",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListAllItems(context.Background(), ListAllItemsArgs{
		BoardID:  "board123",
		MaxItems: 5,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 5 {
		t.Errorf("Count = %d, want 5 (max items limit)", result.Count)
	}
	if !result.Truncated {
		t.Error("Truncated should be true")
	}
}

func TestGetItem_WithLinks(t *testing.T) {
	// Tests GetItem when item has links
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "item123",
			"type": "sticky_note",
			"links": map[string]interface{}{
				"self":    "https://api.miro.com/v2/boards/board123/items/item123",
				"related": "https://api.miro.com/v2/boards/board123",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetItem(context.Background(), GetItemArgs{
		BoardID: "board123",
		ItemID:  "item123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "item123" {
		t.Errorf("ID = %v, want 'item123'", result.ID)
	}
}

func TestValidateItemID_TooLong(t *testing.T) {
	// Create an ID that exceeds max length (256 chars)
	longID := strings.Repeat("a", 300)
	err := ValidateItemID(longID)

	if err == nil {
		t.Error("expected error for ID that's too long")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("error should mention 'too long': %v", err)
	}
}

func TestValidateItemID_InvalidCharacters(t *testing.T) {
	// Test ID with invalid characters
	err := ValidateItemID("item<>123")

	if err == nil {
		t.Error("expected error for ID with invalid characters")
	}
	if !strings.Contains(err.Error(), "invalid characters") {
		t.Errorf("error should mention 'invalid characters': %v", err)
	}
}

func TestDeleteItem_NoFallbackOn500(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "item1"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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
