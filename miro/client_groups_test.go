package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateGroup_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/groups" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify request body
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		data := body["data"].(map[string]interface{})
		items := data["items"].([]interface{})
		if len(items) != 3 {
			t.Errorf("expected 3 items, got %d", len(items))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "group123",
			"type": "group",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateGroup(context.Background(), CreateGroupArgs{
		BoardID: "board123",
		ItemIDs: []string{"item1", "item2", "item3"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "group123" {
		t.Errorf("ID = %q, want 'group123'", result.ID)
	}
	if len(result.ItemIDs) != 3 {
		t.Errorf("ItemIDs count = %d, want 3", len(result.ItemIDs))
	}
}

func TestCreateGroup_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateGroupArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateGroupArgs{ItemIDs: []string{"item1", "item2"}},
			errText: "board_id",
		},
		{
			name:    "less than 2 items",
			args:    CreateGroupArgs{BoardID: "board123", ItemIDs: []string{"item1"}},
			errText: "at least 2 items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateGroup(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestListGroups_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards/board123/groups") {
			t.Errorf("expected /boards/board123/groups, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":    "group1",
					"items": []string{"item1", "item2"},
				},
				{
					"id":    "group2",
					"items": []string{"item3"},
				},
			},
			"cursor": "",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListGroups(context.Background(), ListGroupsArgs{
		BoardID: "board123",
		Limit:   50,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if result.Groups[0].ID != "group1" {
		t.Errorf("first group ID = %q, want 'group1'", result.Groups[0].ID)
	}
}

func TestGetGroup_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/groups/group456" {
			t.Errorf("expected /boards/board123/groups/group456, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "group456",
			"items": []string{"item1", "item2", "item3"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetGroup(context.Background(), GetGroupArgs{
		BoardID: "board123",
		GroupID: "group456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "group456" {
		t.Errorf("ID = %q, want 'group456'", result.ID)
	}
	if len(result.Items) != 3 {
		t.Errorf("Items count = %d, want 3", len(result.Items))
	}
}

func TestGetGroup_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    GetGroupArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    GetGroupArgs{GroupID: "group123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty group ID",
			args:    GetGroupArgs{BoardID: "board123"},
			wantErr: "invalid group_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetGroup(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestGetGroupItems_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards/board123/groups/group456/items") {
			t.Errorf("expected /boards/board123/groups/group456/items, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "item1",
					"type": "sticky_note",
					"data": map[string]interface{}{
						"content": "First item",
					},
				},
				{
					"id":   "item2",
					"type": "shape",
					"data": map[string]interface{}{
						"content": "Second item",
					},
				},
			},
			"cursor": "",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetGroupItems(context.Background(), GetGroupItemsArgs{
		BoardID: "board123",
		GroupID: "group456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if len(result.Items) != 2 {
		t.Errorf("Items count = %d, want 2", len(result.Items))
	}
	if result.Items[0].ID != "item1" {
		t.Errorf("Items[0].ID = %q, want 'item1'", result.Items[0].ID)
	}
	if result.Items[0].Type != "sticky_note" {
		t.Errorf("Items[0].Type = %q, want 'sticky_note'", result.Items[0].Type)
	}
	if result.HasMore {
		t.Error("HasMore = true, want false")
	}
}

func TestGetGroupItems_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    GetGroupItemsArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    GetGroupItemsArgs{GroupID: "group123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty group ID",
			args:    GetGroupItemsArgs{BoardID: "board123"},
			wantErr: "invalid group_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetGroupItems(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestGetGroupItems_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "item1",
					"type": "sticky_note",
					"data": map[string]interface{}{
						"content": "Item",
					},
				},
			},
			"cursor": "next_page_token",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetGroupItems(context.Background(), GetGroupItemsArgs{
		BoardID: "board123",
		GroupID: "group456",
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasMore {
		t.Error("HasMore = false, want true")
	}
}

func TestDeleteGroup_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/groups/group456" {
			t.Errorf("expected /boards/board123/groups/group456, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteGroup(context.Background(), DeleteGroupArgs{
		BoardID: "board123",
		GroupID: "group456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.GroupID != "group456" {
		t.Errorf("GroupID = %q, want 'group456'", result.GroupID)
	}
}

func TestDeleteGroup_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    DeleteGroupArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    DeleteGroupArgs{GroupID: "group123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty group ID",
			args:    DeleteGroupArgs{BoardID: "board123"},
			wantErr: "invalid group_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.DeleteGroup(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestUpdateGroup_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/groups/") {
			t.Errorf("expected /groups/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "group123",
			"itemIds": []string{"item1", "item2", "item3"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateGroup(context.Background(), UpdateGroupArgs{
		BoardID: "board123",
		GroupID: "group123",
		ItemIDs: []string{"item1", "item2", "item3"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "group123" {
		t.Errorf("ID = %q, want 'group123'", result.ID)
	}
	if len(result.ItemIDs) != 3 {
		t.Errorf("ItemIDs count = %d, want 3", len(result.ItemIDs))
	}
}

func TestUpdateGroup_NoItemIDs(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.UpdateGroup(context.Background(), UpdateGroupArgs{
		BoardID: "board123",
		GroupID: "group123",
		ItemIDs: []string{}, // Empty item list
	})

	if err == nil {
		t.Fatal("expected error for empty item_ids")
	}
}

func TestUpdateGroup_InvalidBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.UpdateGroup(context.Background(), UpdateGroupArgs{
		BoardID: "",
		GroupID: "group123",
		ItemIDs: []string{"item1"},
	})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
}

func TestUpdateGroup_InvalidGroupID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.UpdateGroup(context.Background(), UpdateGroupArgs{
		BoardID: "board123",
		GroupID: "",
		ItemIDs: []string{"item1"},
	})

	if err == nil {
		t.Fatal("expected error for empty group_id")
	}
}

func TestListGroups_WithCursor(t *testing.T) {
	// Tests pagination with cursor
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "cursor=next123") {
			t.Error("expected cursor parameter in request")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "group1", "items": []string{"item1", "item2"}},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListGroups(context.Background(), ListGroupsArgs{
		BoardID: "board123",
		Cursor:  "next123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 1 {
		t.Errorf("expected 1 group, got %d", result.Count)
	}
}

func TestGetGroupItems_WithCursor(t *testing.T) {
	// Tests pagination with cursor
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "cursor=page2") {
			t.Error("expected cursor parameter in request")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "item1", "type": "sticky_note"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetGroupItems(context.Background(), GetGroupItemsArgs{
		BoardID: "board123",
		GroupID: "group123",
		Cursor:  "page2",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteGroup_APIError(t *testing.T) {
	// Tests the error branch for DeleteGroup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  403,
			"message": "Access denied",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteGroup(context.Background(), DeleteGroupArgs{
		BoardID: "board123",
		GroupID: "group123",
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if result.Success {
		t.Error("expected Success to be false")
	}
}
