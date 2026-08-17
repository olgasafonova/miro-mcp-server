package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTagTestClient starts a test server with the given handler and returns a
// client pointed at it. The server is closed automatically at test end.
func newTagTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

// checkTagField fails the test when got differs from want.
func checkTagField[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// tagWire builds the wire-format JSON for one tag.
func tagWire(id, title, color string) map[string]interface{} {
	return map[string]interface{}{
		"id":        id,
		"title":     title,
		"fillColor": color,
	}
}

// tagListPayload builds the wire-format list response for tags.
func tagListPayload(tags ...map[string]interface{}) map[string]interface{} {
	items := make([]interface{}, 0, len(tags))
	for _, tag := range tags {
		items = append(items, tag)
	}
	return map[string]interface{}{"data": items}
}

// newTagUpdateClient serves a two-step UpdateTag flow: the first GET returns
// the current tag so the client can merge fields, the PATCH returns updated.
func newTagUpdateClient(t *testing.T, current, updated map[string]interface{}) *Client {
	t.Helper()
	requestCount := 0
	return newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		// First request: GET to fetch existing tag
		if requestCount == 1 && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(current)
			return
		}

		// Second request: PATCH to update tag
		if r.Method == http.MethodPatch {
			json.NewEncoder(w).Encode(updated)
		}
	})
}

// tagOpCase describes one tag CRUD call: the expected request shape, the
// canned response, and a closure performing the call plus result assertions.
type tagOpCase struct {
	name       string
	wantMethod string
	wantPath   string
	wantTagID  string
	payload    map[string]interface{}
	call       func(t *testing.T, client *Client) error
}

// runTagOpCases drives each case against a server that verifies the request
// shape before responding; payload nil means a 204 No Content response.
func runTagOpCases(t *testing.T, tests []tagOpCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				checkTagField(t, "method", r.Method, tt.wantMethod)
				if tt.wantPath != "" {
					checkTagField(t, "path", r.URL.Path, tt.wantPath)
				}
				if tt.wantTagID != "" {
					checkTagField(t, "tag_id", r.URL.Query().Get("tag_id"), tt.wantTagID)
				}
				if tt.payload != nil {
					writeJSON(w, tt.payload)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			if err := tt.call(t, client); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCreateTag_Success(t *testing.T) {
	client := newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkTagField(t, "method", r.Method, http.MethodPost)
		checkTagField(t, "path", r.URL.Path, "/boards/board123/tags")

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		checkTagField(t, "title", body["title"], interface{}("Important"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(tagWire("tag123", "Important", "red"))
	})

	result, err := client.CreateTag(context.Background(), CreateTagArgs{
		BoardID: "board123",
		Title:   "Important",
		Color:   "red",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkTagField(t, "ID", result.ID, "tag123")
	checkTagField(t, "Title", result.Title, "Important")
}

// TestTagOps_Success covers the happy path of each tag CRUD method.
func TestTagOps_Success(t *testing.T) {
	runTagOpCases(t, []tagOpCase{
		{
			name: "attach", wantMethod: http.MethodPost,
			wantPath: "/boards/board123/items/item456", wantTagID: "tag789",
			call: func(t *testing.T, client *Client) error {
				result, err := client.AttachTag(context.Background(), AttachTagArgs{BoardID: "board123", ItemID: "item456", TagID: "tag789"})
				if err == nil {
					checkTagField(t, "Success", result.Success, true)
				}
				return err
			},
		},
		{
			name: "detach", wantMethod: http.MethodDelete,
			call: func(t *testing.T, client *Client) error {
				result, err := client.DetachTag(context.Background(), DetachTagArgs{BoardID: "board123", ItemID: "item456", TagID: "tag789"})
				if err == nil {
					checkTagField(t, "Success", result.Success, true)
				}
				return err
			},
		},
		{
			name: "delete", wantMethod: http.MethodDelete, wantPath: "/boards/board123/tags/tag456",
			call: func(t *testing.T, client *Client) error {
				result, err := client.DeleteTag(context.Background(), DeleteTagArgs{BoardID: "board123", TagID: "tag456"})
				if err == nil {
					checkTagField(t, "Success", result.Success, true)
					checkTagField(t, "TagID", result.TagID, "tag456")
				}
				return err
			},
		},
		{
			name: "update", wantMethod: http.MethodPatch, wantPath: "/boards/board123/tags/tag456",
			payload: tagWire("tag456", "Updated Title", "blue"),
			call: func(t *testing.T, client *Client) error {
				result, err := client.UpdateTag(context.Background(), UpdateTagArgs{BoardID: "board123", TagID: "tag456", Title: "Updated Title", Color: "blue"})
				if err == nil {
					checkTagField(t, "Success", result.Success, true)
					checkTagField(t, "Title", result.Title, "Updated Title")
				}
				return err
			},
		},
		{
			name: "get", wantMethod: http.MethodGet, wantPath: "/boards/board123/tags/tag456",
			payload: tagWire("tag456", "Urgent", "red"),
			call: func(t *testing.T, client *Client) error {
				result, err := client.GetTag(context.Background(), GetTagArgs{BoardID: "board123", TagID: "tag456"})
				if err == nil {
					checkTagField(t, "ID", result.ID, "tag456")
					checkTagField(t, "Title", result.Title, "Urgent")
					checkTagField(t, "Color", result.Color, "red")
				}
				return err
			},
		},
	})
}

// TestTag_ValidationErrors covers input validation across the tag methods;
// every case must fail with a message naming the missing field.
func TestTag_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	offline := newTestClientWithServer("http://localhost")
	ctx := context.Background()

	tests := []struct {
		name    string
		call    func() error
		errText string
	}{
		{"create with empty board_id", func() error { _, err := client.CreateTag(ctx, CreateTagArgs{Title: "Test"}); return err }, "board_id is required"},
		{"create with empty title", func() error { _, err := client.CreateTag(ctx, CreateTagArgs{BoardID: "board123"}); return err }, "title is required"},
		{"attach with empty board_id", func() error {
			_, err := client.AttachTag(ctx, AttachTagArgs{ItemID: "item123", TagID: "tag123"})
			return err
		}, "board_id is required"},
		{"attach with empty item_id", func() error {
			_, err := client.AttachTag(ctx, AttachTagArgs{BoardID: "board123", TagID: "tag123"})
			return err
		}, "item_id is required"},
		{"attach with empty tag_id", func() error {
			_, err := client.AttachTag(ctx, AttachTagArgs{BoardID: "board123", ItemID: "item123"})
			return err
		}, "tag_id is required"},
		{"update with empty board_id", func() error {
			_, err := client.UpdateTag(ctx, UpdateTagArgs{TagID: "tag123", Title: "Test"})
			return err
		}, "board_id is required"},
		{"update with empty tag_id", func() error {
			_, err := client.UpdateTag(ctx, UpdateTagArgs{BoardID: "board123", Title: "Test"})
			return err
		}, "tag_id is required"},
		{"update with no changes", func() error {
			_, err := client.UpdateTag(ctx, UpdateTagArgs{BoardID: "board123", TagID: "tag123"})
			return err
		}, "at least one of title or color is required"},
		{"delete with empty board_id", func() error { _, err := client.DeleteTag(ctx, DeleteTagArgs{TagID: "tag123"}); return err }, "board_id is required"},
		{"delete with empty tag_id", func() error { _, err := client.DeleteTag(ctx, DeleteTagArgs{BoardID: "board123"}); return err }, "tag_id is required"},
		{"get with empty board ID", func() error { _, err := offline.GetTag(ctx, GetTagArgs{TagID: "tag123"}); return err }, "board_id is required"},
		{"get with empty tag ID", func() error { _, err := offline.GetTag(ctx, GetTagArgs{BoardID: "board123"}); return err }, "tag_id is required"},
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

// TestTag_EmptyArgs exercises the same validation paths on a client built
// directly from config, without a test server behind it.
func TestTag_EmptyArgs(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{"detach with empty board_id", func() error {
			_, err := client.DetachTag(ctx, DetachTagArgs{BoardID: "", ItemID: "item123", TagID: "tag456"})
			return err
		}},
		{"detach with empty item_id", func() error {
			_, err := client.DetachTag(ctx, DetachTagArgs{BoardID: "board123", ItemID: "", TagID: "tag456"})
			return err
		}},
		{"detach with empty tag_id", func() error {
			_, err := client.DetachTag(ctx, DetachTagArgs{BoardID: "board123", ItemID: "item123", TagID: ""})
			return err
		}},
		{"get item tags with empty board_id", func() error {
			_, err := client.GetItemTags(ctx, GetItemTagsArgs{BoardID: "", ItemID: "item123"})
			return err
		}},
		{"get item tags with empty item_id", func() error {
			_, err := client.GetItemTags(ctx, GetItemTagsArgs{BoardID: "board123", ItemID: ""})
			return err
		}},
		{"list with empty board_id", func() error { _, err := client.ListTags(ctx, ListTagsArgs{BoardID: ""}); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil {
				t.Error("expected error for empty argument")
			}
		})
	}
}

func TestListTags_Success(t *testing.T) {
	client := newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkTagField(t, "method", r.Method, http.MethodGet)
		writeJSON(w, tagListPayload(
			tagWire("tag1", "Urgent", "red"),
			tagWire("tag2", "Done", "green"),
		))
	})

	result, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkTagField(t, "Count", result.Count, 2)
	checkTagField(t, "first tag title", result.Tags[0].Title, "Urgent")
}

func TestListTags_Empty(t *testing.T) {
	client := newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, tagListPayload())
	})

	result, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkTagField(t, "Count", result.Count, 0)
	checkTagField(t, "Message", result.Message, "No tags on this board")
}

func TestGetItemTags_Success(t *testing.T) {
	client := newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkTagField(t, "path", r.URL.Path, "/boards/board123/items/item456/tags")
		writeJSON(w, tagListPayload(
			tagWire("tag1", "Priority", "red"),
		))
	})

	result, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "board123",
		ItemID:  "item456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkTagField(t, "Count", result.Count, 1)
	checkTagField(t, "ItemID", result.ItemID, "item456")
}

// TestUpdateTag_PartialUpdate covers updates that set only one field; the
// client must fetch the existing tag first and preserve the other field.
func TestUpdateTag_PartialUpdate(t *testing.T) {
	tests := []struct {
		name             string
		current, updated map[string]interface{}
		args             UpdateTagArgs
		wantTitle        string
		wantColor        string
	}{
		{
			name:      "title only fetches existing color",
			current:   tagWire("tag456", "Old Title", "red"),
			updated:   tagWire("tag456", "New Title", "red"),
			args:      UpdateTagArgs{BoardID: "board123", TagID: "tag456", Title: "New Title"},
			wantTitle: "New Title",
		},
		{
			name:      "color only fetches existing title",
			current:   tagWire("tag456", "Existing Title", "red"),
			updated:   tagWire("tag456", "Existing Title", "blue"),
			args:      UpdateTagArgs{BoardID: "board123", TagID: "tag456", Color: "blue"},
			wantColor: "blue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTagUpdateClient(t, tt.current, tt.updated)
			result, err := client.UpdateTag(context.Background(), tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			checkTagField(t, "Success", result.Success, true)
			if tt.wantTitle != "" {
				checkTagField(t, "Title", result.Title, tt.wantTitle)
			}
			if tt.wantColor != "" {
				checkTagField(t, "Color", result.Color, tt.wantColor)
			}
		})
	}
}

func TestGetTag_NotFound(t *testing.T) {
	client := newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "not_found",
			"message": "Tag not found",
		})
	})

	_, err := client.GetTag(context.Background(), GetTagArgs{
		BoardID: "board123",
		TagID:   "nonexistent",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected not found error, got: %v", err)
	}
}

func TestDetachTag_APIError(t *testing.T) {
	// Tests the error branch for DetachTag
	client := newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  404,
			"message": "Tag not found",
		})
	})

	result, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "board123",
		ItemID:  "item123",
		TagID:   "tag123",
	})

	if err == nil {
		t.Fatal("expected error")
	}
	checkTagField(t, "Success", result.Success, false)
	if !strings.Contains(result.Message, "Failed to detach tag") {
		t.Errorf("expected failure message, got: %s", result.Message)
	}
}

func TestGetItemTags_EmptyTags(t *testing.T) {
	// Tests the branch where no tags are returned and message is "No tags on this item"
	client := newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, tagListPayload())
	})

	result, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "board123",
		ItemID:  "item123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkTagField(t, "Message", result.Message, "No tags on this item")
	checkTagField(t, "Count", result.Count, 0)
}

func TestGetItemTags_NilData(t *testing.T) {
	// Tests the branch where data is null/nil
	client := newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return null data
		w.Write([]byte(`{"data": null}`))
	})

	result, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "board123",
		ItemID:  "item123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Tags should be empty slice, not nil
	if result.Tags == nil {
		t.Error("expected Tags to be empty slice, not nil")
	}
	checkTagField(t, "len(Tags)", len(result.Tags), 0)
}

func TestCreateTag_WithDefaultColor(t *testing.T) {
	// Tests CreateTag when color is not specified (defaults to blue)
	client := newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		checkTagField(t, "fillColor (default)", body["fillColor"], interface{}("blue"))
		writeJSON(w, tagWire("tag123", "My Tag", "blue"))
	})

	_, err := client.CreateTag(context.Background(), CreateTagArgs{
		BoardID: "board123",
		Title:   "My Tag",
		// Color not specified - should default to blue
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDetachTag_APIError_NotFound(t *testing.T) {
	client := newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Tag not found"}`))
	})

	result, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "board123",
		ItemID:  "item123",
		TagID:   "tag456",
	})
	if err == nil {
		t.Error("expected error for 404 response")
	}
	checkTagField(t, "Success", result.Success, false)
}

// TestTags_JSONParseError verifies that malformed JSON responses surface as
// errors from the tag read methods.
func TestTags_JSONParseError(t *testing.T) {
	tests := []struct {
		name string
		call func(*Client) error
	}{
		{"get item tags", func(client *Client) error {
			_, err := client.GetItemTags(context.Background(), GetItemTagsArgs{BoardID: "board123", ItemID: "item123"})
			return err
		}},
		{"list tags", func(client *Client) error {
			_, err := client.ListTags(context.Background(), ListTagsArgs{BoardID: "board123"})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{invalid json}`))
			})
			if err := tt.call(client); err == nil {
				t.Error("expected error for invalid JSON response")
			}
		})
	}
}

func TestListTags_WithLimit(t *testing.T) {
	client := newTagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "limit=25") {
			t.Errorf("expected limit=25, got: %s", r.URL.RawQuery)
		}
		writeJSON(w, tagListPayload())
	})

	result, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "board123",
		Limit:   25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkTagField(t, "Message", result.Message, "No tags on this board")
}
