package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// =============================================================================
// Items-by-tag Tests
// =============================================================================

func TestClampTagItemsLimit(t *testing.T) {
	tests := []struct {
		name      string
		requested int
		want      int
	}{
		{"zero uses the cap", 0, maxTagItemsPageSize},
		{"negative uses the cap", -3, maxTagItemsPageSize},
		{"above the cap is clamped", 500, maxTagItemsPageSize},
		{"exactly the cap is preserved", maxTagItemsPageSize, maxTagItemsPageSize},
		{"in range is preserved", 20, 20},
		{"one is preserved", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clampTagItemsLimit(tt.requested); got != tt.want {
				t.Errorf("clampTagItemsLimit(%d) = %d, want %d", tt.requested, got, tt.want)
			}
		})
	}
}

func TestBuildTagItemsPath(t *testing.T) {
	t.Run("omits offset when zero", func(t *testing.T) {
		got := buildTagItemsPath("board123", "tag456", 25, 0)

		if !strings.HasPrefix(got, "/boards/board123/items?") {
			t.Fatalf("path = %q, want a /boards/board123/items prefix", got)
		}
		q, err := url.ParseQuery(strings.SplitN(got, "?", 2)[1])
		if err != nil {
			t.Fatalf("parsing query: %v", err)
		}
		if q.Get("limit") != "25" {
			t.Errorf("limit = %q, want 25", q.Get("limit"))
		}
		if q.Get("tag_id") != "tag456" {
			t.Errorf("tag_id = %q, want tag456", q.Get("tag_id"))
		}
		if _, present := q["offset"]; present {
			t.Error("offset must be omitted when zero")
		}
	})

	t.Run("includes offset when positive", func(t *testing.T) {
		got := buildTagItemsPath("board123", "tag456", 10, 30)
		q, err := url.ParseQuery(strings.SplitN(got, "?", 2)[1])
		if err != nil {
			t.Fatalf("parsing query: %v", err)
		}
		if q.Get("offset") != "30" {
			t.Errorf("offset = %q, want 30", q.Get("offset"))
		}
	})

	t.Run("escapes values", func(t *testing.T) {
		got := buildTagItemsPath("board123", "tag with space", 10, 0)
		if strings.Contains(got, "tag with space") {
			t.Errorf("path = %q, want the tag_id percent-encoded", got)
		}
	})
}

func TestGetItemsByTag_Success(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		gotQuery = r.URL.Query()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":       "item1",
					"type":     "sticky_note",
					"position": map[string]interface{}{"x": 1.0, "y": 2.0},
					"data":     map[string]interface{}{"content": "tagged one"},
				},
				{
					"id":       "item2",
					"type":     "shape",
					"position": map[string]interface{}{"x": 3.0, "y": 4.0},
					"data":     map[string]interface{}{"content": "tagged two"},
				},
			},
			"size": 2,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetItemsByTag(context.Background(), GetItemsByTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if result.TagID != "tag456" {
		t.Errorf("TagID = %q, want tag456", result.TagID)
	}
	if result.HasMore {
		t.Error("HasMore should be false when fewer items than the limit came back")
	}
	if result.Items[0].ID != "item1" || result.Items[1].Type != "shape" {
		t.Errorf("items = %+v", result.Items)
	}
	if result.Message != "Found 2 items with tag tag456" {
		t.Errorf("Message = %q", result.Message)
	}
	if gotQuery.Get("tag_id") != "tag456" {
		t.Errorf("request tag_id = %q, want tag456", gotQuery.Get("tag_id"))
	}
	if gotQuery.Get("limit") != "10" {
		t.Errorf("request limit = %q, want 10", gotQuery.Get("limit"))
	}
}

func TestGetItemsByTag_HasMoreWhenPageIsFull(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "item1", "type": "sticky_note"},
				{"id": "item2", "type": "sticky_note"},
			},
			"size": 2,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetItemsByTag(context.Background(), GetItemsByTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
		Limit:   2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasMore {
		t.Error("HasMore should be true when the page came back full")
	}
}

func TestGetItemsByTag_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}, "size": 0})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetItemsByTag(context.Background(), GetItemsByTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 0 {
		t.Errorf("Count = %d, want 0", result.Count)
	}
	if result.Items == nil {
		t.Error("Items should be an empty slice, not nil")
	}
	if result.HasMore {
		t.Error("HasMore should be false on an empty page")
	}
}

func TestGetItemsByTag_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    GetItemsByTagArgs
		wantErr string
	}{
		{"empty board", GetItemsByTagArgs{BoardID: "", TagID: "tag456"}, "board_id is required"},
		{"empty tag", GetItemsByTagArgs{BoardID: "board123", TagID: ""}, "tag_id is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClientWithServer("http://unused.invalid")
			if _, err := client.GetItemsByTag(context.Background(), tt.args); err == nil ||
				!strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestGetItemsByTag_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.GetItemsByTag(context.Background(), GetItemsByTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
	}); err == nil {
		t.Fatal("expected error on HTTP 404")
	}
}

func TestGetItemsByTag_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{not json"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetItemsByTag(context.Background(), GetItemsByTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
	})
	if err == nil || !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error = %v, want failed to parse response", err)
	}
}
