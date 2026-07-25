package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateTag_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["title"] != "Important" {
			t.Errorf("title = %v, want 'Important'", body["title"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        "tag123",
			"title":     "Important",
			"fillColor": "red",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateTag(context.Background(), CreateTagArgs{
		BoardID: "board123",
		Title:   "Important",
		Color:   "red",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "tag123" {
		t.Errorf("ID = %q, want 'tag123'", result.ID)
	}
	if result.Title != "Important" {
		t.Errorf("Title = %q, want 'Important'", result.Title)
	}
}

func TestCreateTag_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateTagArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateTagArgs{Title: "Test"},
			errText: "board_id is required",
		},
		{
			name:    "empty title",
			args:    CreateTagArgs{BoardID: "board123"},
			errText: "title is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateTag(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestListTags_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "tag1", "title": "Urgent", "fillColor": "red"},
				{"id": "tag2", "title": "Done", "fillColor": "green"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if result.Tags[0].Title != "Urgent" {
		t.Errorf("first tag title = %q, want 'Urgent'", result.Tags[0].Title)
	}
}

func TestListTags_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 0 {
		t.Errorf("Count = %d, want 0", result.Count)
	}
	if result.Message != "No tags on this board" {
		t.Errorf("Message = %q, want 'No tags on this board'", result.Message)
	}
}

func TestAttachTag_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		expectedPath := "/boards/board123/items/item456"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %q, want %q", r.URL.Path, expectedPath)
		}
		if r.URL.Query().Get("tag_id") != "tag789" {
			t.Errorf("tag_id = %q, want 'tag789'", r.URL.Query().Get("tag_id"))
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.AttachTag(context.Background(), AttachTagArgs{
		BoardID: "board123",
		ItemID:  "item456",
		TagID:   "tag789",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestAttachTag_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    AttachTagArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    AttachTagArgs{ItemID: "item123", TagID: "tag123"},
			errText: "board_id is required",
		},
		{
			name:    "empty item_id",
			args:    AttachTagArgs{BoardID: "board123", TagID: "tag123"},
			errText: "item_id is required",
		},
		{
			name:    "empty tag_id",
			args:    AttachTagArgs{BoardID: "board123", ItemID: "item123"},
			errText: "tag_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.AttachTag(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestDetachTag_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "board123",
		ItemID:  "item456",
		TagID:   "tag789",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestGetItemTags_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boards/board123/items/item456/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "tag1", "title": "Priority", "fillColor": "red"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "board123",
		ItemID:  "item456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 1 {
		t.Errorf("Count = %d, want 1", result.Count)
	}
	if result.ItemID != "item456" {
		t.Errorf("ItemID = %q, want 'item456'", result.ItemID)
	}
}

func TestUpdateTag_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/tags/tag456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        "tag456",
			"title":     "Updated Title",
			"fillColor": "blue",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateTag(context.Background(), UpdateTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
		Title:   "Updated Title",
		Color:   "blue",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Title != "Updated Title" {
		t.Errorf("Title = %q, want 'Updated Title'", result.Title)
	}
}

func TestUpdateTag_TitleOnly(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		// First request: GET to fetch existing tag
		if requestCount == 1 && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "tag456",
				"title":     "Old Title",
				"fillColor": "red",
			})
			return
		}

		// Second request: PATCH to update tag
		if r.Method == http.MethodPatch {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "tag456",
				"title":     "New Title",
				"fillColor": "red",
			})
			return
		}
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateTag(context.Background(), UpdateTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
		Title:   "New Title", // Only title, no color - should fetch existing
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Title != "New Title" {
		t.Errorf("Title = %q, want 'New Title'", result.Title)
	}
}

func TestUpdateTag_ColorOnly(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		// First request: GET to fetch existing tag
		if requestCount == 1 && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "tag456",
				"title":     "Existing Title",
				"fillColor": "red",
			})
			return
		}

		// Second request: PATCH to update tag
		if r.Method == http.MethodPatch {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "tag456",
				"title":     "Existing Title",
				"fillColor": "blue",
			})
			return
		}
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateTag(context.Background(), UpdateTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
		Color:   "blue", // Only color, no title - should fetch existing
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Color != "blue" {
		t.Errorf("Color = %q, want 'blue'", result.Color)
	}
}

func TestUpdateTag_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    UpdateTagArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    UpdateTagArgs{TagID: "tag123", Title: "Test"},
			errText: "board_id is required",
		},
		{
			name:    "empty tag_id",
			args:    UpdateTagArgs{BoardID: "board123", Title: "Test"},
			errText: "tag_id is required",
		},
		{
			name:    "no changes",
			args:    UpdateTagArgs{BoardID: "board123", TagID: "tag123"},
			errText: "at least one of title or color is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UpdateTag(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestDeleteTag_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/tags/tag456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteTag(context.Background(), DeleteTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.TagID != "tag456" {
		t.Errorf("TagID = %q, want 'tag456'", result.TagID)
	}
}

func TestDeleteTag_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    DeleteTagArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    DeleteTagArgs{TagID: "tag123"},
			errText: "board_id is required",
		},
		{
			name:    "empty tag_id",
			args:    DeleteTagArgs{BoardID: "board123"},
			errText: "tag_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.DeleteTag(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestGetTag_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/tags/tag456" {
			t.Errorf("expected /boards/board123/tags/tag456, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        "tag456",
			"title":     "Urgent",
			"fillColor": "red",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetTag(context.Background(), GetTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "tag456" {
		t.Errorf("ID = %q, want 'tag456'", result.ID)
	}
	if result.Title != "Urgent" {
		t.Errorf("Title = %q, want 'Urgent'", result.Title)
	}
	if result.Color != "red" {
		t.Errorf("Color = %q, want 'red'", result.Color)
	}
}

func TestGetTag_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    GetTagArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    GetTagArgs{TagID: "tag123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty tag ID",
			args:    GetTagArgs{BoardID: "board123"},
			wantErr: "tag_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetTag(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestGetTag_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "not_found",
			"message": "Tag not found",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  404,
			"message": "Tag not found",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "board123",
		ItemID:  "item123",
		TagID:   "tag123",
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if result.Success {
		t.Error("expected Success to be false")
	}
	if !strings.Contains(result.Message, "Failed to detach tag") {
		t.Errorf("expected failure message, got: %s", result.Message)
	}
}

func TestGetItemTags_EmptyTags(t *testing.T) {
	// Tests the branch where no tags are returned and message is "No tags on this item"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "board123",
		ItemID:  "item123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No tags on this item" {
		t.Errorf("expected 'No tags on this item', got: %s", result.Message)
	}
	if result.Count != 0 {
		t.Errorf("expected Count = 0, got %d", result.Count)
	}
}

func TestGetItemTags_NilData(t *testing.T) {
	// Tests the branch where data is null/nil (line 183-184)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return null data
		w.Write([]byte(`{"data": null}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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
	if len(result.Tags) != 0 {
		t.Errorf("expected empty Tags, got %d items", len(result.Tags))
	}
}

func TestCreateTag_WithDefaultColor(t *testing.T) {
	// Tests CreateTag when color is not specified (defaults to blue)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["fillColor"] != "blue" {
			t.Errorf("fillColor = %v, want 'blue' (default)", body["fillColor"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        "tag123",
			"title":     "My Tag",
			"fillColor": "blue",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateTag(context.Background(), CreateTagArgs{
		BoardID: "board123",
		Title:   "My Tag",
		// Color not specified - should default to blue
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Tests for DetachTag error paths
func TestDetachTag_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "",
		ItemID:  "item123",
		TagID:   "tag456",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

func TestDetachTag_EmptyItemID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "board123",
		ItemID:  "",
		TagID:   "tag456",
	})
	if err == nil {
		t.Error("expected error for empty item_id")
	}
}

func TestDetachTag_EmptyTagID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "board123",
		ItemID:  "item123",
		TagID:   "",
	})
	if err == nil {
		t.Error("expected error for empty tag_id")
	}
}

func TestDetachTag_APIError_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Tag not found"}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DetachTag(context.Background(), DetachTagArgs{
		BoardID: "board123",
		ItemID:  "item123",
		TagID:   "tag456",
	})
	if err == nil {
		t.Error("expected error for 404 response")
	}
	if result.Success {
		t.Error("expected Success=false on error")
	}
}

// Tests for GetItemTags error paths
func TestGetItemTags_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "",
		ItemID:  "item123",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

func TestGetItemTags_EmptyItemID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "board123",
		ItemID:  "",
	})
	if err == nil {
		t.Error("expected error for empty item_id")
	}
}

func TestGetItemTags_JSONParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetItemTags(context.Background(), GetItemTagsArgs{
		BoardID: "board123",
		ItemID:  "item123",
	})
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

// Tests for ListTags error paths
func TestListTags_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

// Tests for ListTags
func TestListTags_WithLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "limit=25") {
			t.Errorf("expected limit=25, got: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "board123",
		Limit:   25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No tags on this board" {
		t.Errorf("expected no tags message, got: %s", result.Message)
	}
}

func TestListTags_JSONParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "board123",
	})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
