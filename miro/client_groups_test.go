package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newGroupTestClient starts a test server with the given handler and returns
// a client pointed at it. The server is closed via t.Cleanup.
func newGroupTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

// groupJSONHandler returns a handler that writes the given payload as JSON.
func groupJSONHandler(payload map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

// assertGetWithPrefix checks that the request is a GET on a path with the
// given prefix.
func assertGetWithPrefix(t *testing.T, r *http.Request, pathPrefix string) {
	t.Helper()
	if r.Method != http.MethodGet {
		t.Errorf("expected GET, got %s", r.Method)
	}
	if !strings.HasPrefix(r.URL.Path, pathPrefix) {
		t.Errorf("expected %s, got %s", pathPrefix, r.URL.Path)
	}
}

func TestCreateGroup_Success(t *testing.T) {
	client := newGroupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
	})

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

// TestGroupValidationErrors covers argument validation across CreateGroup,
// GetGroup, GetGroupItems, DeleteGroup, and UpdateGroup; every call must fail
// before any HTTP request is made.
func TestGroupValidationErrors(t *testing.T) {
	c := newTestClientWithServer("http://localhost")
	ctx := context.Background()
	create := func(args CreateGroupArgs) func() error {
		return func() error { _, err := c.CreateGroup(ctx, args); return err }
	}
	get := func(args GetGroupArgs) func() error {
		return func() error { _, err := c.GetGroup(ctx, args); return err }
	}
	getItems := func(args GetGroupItemsArgs) func() error {
		return func() error { _, err := c.GetGroupItems(ctx, args); return err }
	}
	del := func(args DeleteGroupArgs) func() error {
		return func() error { _, err := c.DeleteGroup(ctx, args); return err }
	}
	update := func(args UpdateGroupArgs) func() error {
		return func() error { _, err := c.UpdateGroup(ctx, args); return err }
	}
	tests := []struct {
		name    string
		call    func() error
		wantErr string
	}{
		{"create empty board_id", create(CreateGroupArgs{ItemIDs: []string{"item1", "item2"}}), "board_id"},
		{"create less than 2 items", create(CreateGroupArgs{BoardID: "board123", ItemIDs: []string{"item1"}}), "at least 2 items"},
		{"get empty board ID", get(GetGroupArgs{GroupID: "group123"}), "board_id is required"},
		{"get empty group ID", get(GetGroupArgs{BoardID: "board123"}), "invalid group_id"},
		{"get items empty board ID", getItems(GetGroupItemsArgs{GroupID: "group123"}), "board_id is required"},
		{"get items empty group ID", getItems(GetGroupItemsArgs{BoardID: "board123"}), "invalid group_id"},
		{"delete empty board ID", del(DeleteGroupArgs{GroupID: "group123"}), "board_id is required"},
		{"delete empty group ID", del(DeleteGroupArgs{BoardID: "board123"}), "invalid group_id"},
		{"update no item IDs", update(UpdateGroupArgs{BoardID: "board123", GroupID: "group123", ItemIDs: []string{}}), ""},
		{"update empty board ID", update(UpdateGroupArgs{GroupID: "group123", ItemIDs: []string{"item1"}}), ""},
		{"update empty group ID", update(UpdateGroupArgs{BoardID: "board123", ItemIDs: []string{"item1"}}), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestListGroups_Success(t *testing.T) {
	client := newGroupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertGetWithPrefix(t, r, "/boards/board123/groups")
		groupJSONHandler(map[string]interface{}{
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
		})(w, r)
	})

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

// groupSuccessSpec describes the request a group handler asserts and the
// payload it returns. Empty exactPath or pathPart skips that check.
type groupSuccessSpec struct {
	method    string
	exactPath string
	pathPart  string
	payload   map[string]interface{}
}

// groupSuccessHandler returns a handler asserting a request matching the spec,
// then responding with the spec payload.
func groupSuccessHandler(t *testing.T, spec groupSuccessSpec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != spec.method {
			t.Errorf("expected %s, got %s", spec.method, r.Method)
		}
		if spec.exactPath != "" && r.URL.Path != spec.exactPath {
			t.Errorf("expected %s, got %s", spec.exactPath, r.URL.Path)
		}
		if spec.pathPart != "" && !strings.Contains(r.URL.Path, spec.pathPart) {
			t.Errorf("expected %s in path, got %s", spec.pathPart, r.URL.Path)
		}
		groupJSONHandler(spec.payload)(w, r)
	}
}

// TestGroupReadWrite_Success covers the GetGroup and UpdateGroup happy paths.
func TestGroupReadWrite_Success(t *testing.T) {
	tests := []struct {
		name          string
		spec          groupSuccessSpec
		call          func(c *Client) (id string, itemCount int, err error)
		wantID        string
		wantItemCount int
	}{
		{
			name: "GetGroup",
			spec: groupSuccessSpec{
				method:    http.MethodGet,
				exactPath: "/boards/board123/groups/group456",
				payload: map[string]interface{}{
					"id":    "group456",
					"items": []string{"item1", "item2", "item3"},
				},
			},
			call: func(c *Client) (string, int, error) {
				result, err := c.GetGroup(context.Background(), GetGroupArgs{
					BoardID: "board123", GroupID: "group456",
				})
				if err != nil {
					return "", 0, err
				}
				return result.ID, len(result.Items), nil
			},
			wantID:        "group456",
			wantItemCount: 3,
		},
		{
			name: "UpdateGroup",
			spec: groupSuccessSpec{
				method:   http.MethodPut,
				pathPart: "/groups/",
				payload: map[string]interface{}{
					"id":      "group123",
					"itemIds": []string{"item1", "item2", "item3"},
				},
			},
			call: func(c *Client) (string, int, error) {
				result, err := c.UpdateGroup(context.Background(), UpdateGroupArgs{
					BoardID: "board123", GroupID: "group123", ItemIDs: []string{"item1", "item2", "item3"},
				})
				if err != nil {
					return "", 0, err
				}
				return result.ID, len(result.ItemIDs), nil
			},
			wantID:        "group123",
			wantItemCount: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newGroupTestClient(t, groupSuccessHandler(t, tt.spec))
			id, itemCount, err := tt.call(client)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.wantID {
				t.Errorf("ID = %q, want %q", id, tt.wantID)
			}
			if itemCount != tt.wantItemCount {
				t.Errorf("item count = %d, want %d", itemCount, tt.wantItemCount)
			}
		})
	}
}

func TestGetGroupItems_Success(t *testing.T) {
	client := newGroupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertGetWithPrefix(t, r, "/boards/board123/groups/group456/items")
		groupJSONHandler(map[string]interface{}{
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
		})(w, r)
	})

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

func TestGetGroupItems_WithPagination(t *testing.T) {
	client := newGroupTestClient(t, groupJSONHandler(map[string]interface{}{
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
	}))

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
	client := newGroupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/groups/group456" {
			t.Errorf("expected /boards/board123/groups/group456, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

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

// TestGroups_WithCursor covers cursor pagination for ListGroups and
// GetGroupItems: the cursor must be forwarded as a query parameter.
func TestGroups_WithCursor(t *testing.T) {
	tests := []struct {
		name      string
		cursor    string
		payload   map[string]interface{}
		call      func(c *Client) (count int, err error)
		wantCount int
	}{
		{
			name:   "ListGroups",
			cursor: "cursor=next123",
			payload: map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "group1", "items": []string{"item1", "item2"}},
				},
			},
			call: func(c *Client) (int, error) {
				result, err := c.ListGroups(context.Background(), ListGroupsArgs{
					BoardID: "board123", Cursor: "next123",
				})
				if err != nil {
					return 0, err
				}
				return result.Count, nil
			},
			wantCount: 1,
		},
		{
			name:   "GetGroupItems",
			cursor: "cursor=page2",
			payload: map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "item1", "type": "sticky_note"},
				},
			},
			call: func(c *Client) (int, error) {
				result, err := c.GetGroupItems(context.Background(), GetGroupItemsArgs{
					BoardID: "board123", GroupID: "group123", Cursor: "page2",
				})
				if err != nil {
					return 0, err
				}
				return result.Count, nil
			},
			wantCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newGroupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.RawQuery, tt.cursor) {
					t.Error("expected cursor parameter in request")
				}
				groupJSONHandler(tt.payload)(w, r)
			})
			count, err := tt.call(client)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if count != tt.wantCount {
				t.Errorf("expected %d results, got %d", tt.wantCount, count)
			}
		})
	}
}

func TestDeleteGroup_APIError(t *testing.T) {
	// Tests the error branch for DeleteGroup
	client := newGroupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  403,
			"message": "Access denied",
		})
	})

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
