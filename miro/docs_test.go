package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// Doc Format Tests
// =============================================================================

func TestCreateDocFormat_Success(t *testing.T) {
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/docs" {
			t.Errorf("expected /boards/board123/docs, got %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "doc123"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateDocFormat(context.Background(), CreateDocFormatArgs{
		BoardID: "board123",
		Content: "# Heading",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "doc123" {
		t.Errorf("ID = %q, want doc123", result.ID)
	}
	if result.Message != "Created doc format item" {
		t.Errorf("Message = %q", result.Message)
	}
	if result.ItemURL != BuildItemURL("board123", "doc123") {
		t.Errorf("ItemURL = %q", result.ItemURL)
	}

	data, ok := gotBody["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data field in request")
	}
	if data["contentType"] != "markdown" {
		t.Errorf("contentType = %v, want markdown", data["contentType"])
	}
	if data["content"] != "# Heading" {
		t.Errorf("content = %v, want # Heading", data["content"])
	}
	if _, present := gotBody["position"]; present {
		t.Error("position should be omitted when x and y are both zero")
	}
	if _, present := gotBody["parent"]; present {
		t.Error("parent should be omitted when parent_id is empty")
	}
}

func TestCreateDocFormat_WithPositionAndParent(t *testing.T) {
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "doc123"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.CreateDocFormat(context.Background(), CreateDocFormatArgs{
		BoardID:  "board123",
		Content:  "body",
		X:        100,
		Y:        -50,
		ParentID: "frame999",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pos, ok := gotBody["position"].(map[string]interface{})
	if !ok {
		t.Fatal("missing position field")
	}
	if pos["x"] != 100.0 || pos["y"] != -50.0 {
		t.Errorf("position = (%v, %v), want (100, -50)", pos["x"], pos["y"])
	}
	if pos["origin"] != "center" {
		t.Errorf("origin = %v, want center", pos["origin"])
	}

	parent, ok := gotBody["parent"].(map[string]interface{})
	if !ok {
		t.Fatal("missing parent field")
	}
	if parent["id"] != "frame999" {
		t.Errorf("parent id = %v, want frame999", parent["id"])
	}
}

func TestCreateDocFormat_YOnlyStillSendsPosition(t *testing.T) {
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "doc123"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.CreateDocFormat(context.Background(), CreateDocFormatArgs{
		BoardID: "board123",
		Content: "body",
		Y:       42,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, present := gotBody["position"]; !present {
		t.Error("position should be sent when only y is non-zero")
	}
}

func TestCreateDocFormat_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    CreateDocFormatArgs
		wantErr string
	}{
		{"empty board", CreateDocFormatArgs{BoardID: "", Content: "x"}, "board_id is required"},
		{"empty content", CreateDocFormatArgs{BoardID: "board123", Content: ""}, "content is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClientWithServer("http://unused.invalid")
			_, err := client.CreateDocFormat(context.Background(), tt.args)
			if err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestCreateDocFormat_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.CreateDocFormat(context.Background(), CreateDocFormatArgs{
		BoardID: "board123",
		Content: "x",
	}); err == nil {
		t.Fatal("expected error on HTTP 400")
	}
}

func TestCreateDocFormat_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{not json"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateDocFormat(context.Background(), CreateDocFormatArgs{
		BoardID: "board123",
		Content: "x",
	})
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error = %v, want failed to parse response", err)
	}
}

func TestGetDocFormat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/docs/doc123" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         "doc123",
			"data":       map[string]interface{}{"content": "# Heading"},
			"position":   map[string]interface{}{"x": 10.0, "y": 20.0},
			"createdAt":  "2026-01-01T00:00:00Z",
			"modifiedAt": "2026-01-02T00:00:00Z",
			"createdBy":  map[string]interface{}{"id": "user1"},
			"modifiedBy": map[string]interface{}{"id": "user2"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetDocFormat(context.Background(), GetDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "doc123" {
		t.Errorf("ID = %q, want doc123", result.ID)
	}
	if result.Content != "# Heading" {
		t.Errorf("Content = %q, want # Heading", result.Content)
	}
	if result.X != 10 || result.Y != 20 {
		t.Errorf("position = (%v, %v), want (10, 20)", result.X, result.Y)
	}
	if result.CreatedBy != "user1" || result.ModifiedBy != "user2" {
		t.Errorf("actors = (%q, %q), want (user1, user2)", result.CreatedBy, result.ModifiedBy)
	}
	if result.Message != "Retrieved doc format item" {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestGetDocFormat_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    GetDocFormatArgs
		wantErr string
	}{
		{"empty board", GetDocFormatArgs{BoardID: "", ItemID: "doc123"}, "board_id is required"},
		{"empty item", GetDocFormatArgs{BoardID: "board123", ItemID: ""}, "item_id is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClientWithServer("http://unused.invalid")
			if _, err := client.GetDocFormat(context.Background(), tt.args); err == nil ||
				!strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestGetDocFormat_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.GetDocFormat(context.Background(), GetDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
	}); err == nil {
		t.Fatal("expected error on HTTP 404")
	}
}

func TestGetDocFormat_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{not json"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetDocFormat(context.Background(), GetDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
	})
	if err == nil || !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error = %v, want failed to parse response", err)
	}
}

func TestDeleteDocFormat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/docs/doc123" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteDocFormat(context.Background(), DeleteDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success = false, want true")
	}
	if result.ItemID != "doc123" {
		t.Errorf("ItemID = %q, want doc123", result.ItemID)
	}
	if result.Message != "Doc format item deleted successfully" {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestDeleteDocFormat_DryRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("dry run must not call the API")
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteDocFormat(context.Background(), DeleteDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success = false, want true")
	}
	if !strings.HasPrefix(result.Message, "[DRY RUN]") {
		t.Errorf("Message = %q, want a [DRY RUN] prefix", result.Message)
	}
}

func TestDeleteDocFormat_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    DeleteDocFormatArgs
		wantErr string
	}{
		{"empty board", DeleteDocFormatArgs{BoardID: "", ItemID: "doc123"}, "board_id is required"},
		{"empty item", DeleteDocFormatArgs{BoardID: "board123", ItemID: ""}, "item_id is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClientWithServer("http://unused.invalid")
			if _, err := client.DeleteDocFormat(context.Background(), tt.args); err == nil ||
				!strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteDocFormat_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.DeleteDocFormat(context.Background(), DeleteDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
	}); err == nil {
		t.Fatal("expected error on HTTP 403")
	}
}

func TestFindAndReplaceContent(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		args        UpdateDocFormatArgs
		wantContent string
		wantCount   int
		wantErr     string
	}{
		{
			name:        "replaces first occurrence only",
			content:     "foo bar foo",
			args:        UpdateDocFormatArgs{OldContent: "foo", NewContent: "baz"},
			wantContent: "baz bar foo",
			wantCount:   1,
		},
		{
			name:        "replaces all occurrences",
			content:     "foo bar foo foo",
			args:        UpdateDocFormatArgs{OldContent: "foo", NewContent: "baz", ReplaceAll: true},
			wantContent: "baz bar baz baz",
			wantCount:   3,
		},
		{
			name:    "missing old content is an error",
			content: "foo bar",
			args:    UpdateDocFormatArgs{OldContent: "nope", NewContent: "baz"},
			wantErr: "old_content not found in document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, count, err := findAndReplaceContent(tt.content, tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantContent {
				t.Errorf("content = %q, want %q", got, tt.wantContent)
			}
			if count != tt.wantCount {
				t.Errorf("count = %d, want %d", count, tt.wantCount)
			}
		})
	}
}

func TestResolveDocFormatContent(t *testing.T) {
	tests := []struct {
		name        string
		current     string
		args        UpdateDocFormatArgs
		wantContent string
		wantCount   int
		wantErr     string
	}{
		{
			name:        "find-and-replace takes precedence",
			current:     "hello world",
			args:        UpdateDocFormatArgs{Content: "ignored", OldContent: "world", NewContent: "there"},
			wantContent: "hello there",
			wantCount:   1,
		},
		{
			name:        "full replacement when no old_content",
			current:     "hello world",
			args:        UpdateDocFormatArgs{Content: "brand new"},
			wantContent: "brand new",
			wantCount:   0,
		},
		{
			name:    "neither mode set is an error",
			current: "hello world",
			args:    UpdateDocFormatArgs{},
			wantErr: "either content (full replace) or old_content+new_content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, count, err := resolveDocFormatContent(tt.current, tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantContent {
				t.Errorf("content = %q, want %q", got, tt.wantContent)
			}
			if count != tt.wantCount {
				t.Errorf("count = %d, want %d", count, tt.wantCount)
			}
		})
	}
}

func TestDocFormatUpdateMessage(t *testing.T) {
	if got := docFormatUpdateMessage(0); got != "Updated doc format item" {
		t.Errorf("message(0) = %q", got)
	}
	if got := docFormatUpdateMessage(3); got != "Replaced 3 occurrence(s) in doc format item" {
		t.Errorf("message(3) = %q", got)
	}
}

// docUpdateServer mimics the read-delete-recreate sequence UpdateDocFormat drives.
func docUpdateServer(t *testing.T, currentContent string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "doc123",
				"data":     map[string]interface{}{"content": currentContent},
				"position": map[string]interface{}{"x": 10.0, "y": 20.0},
			})
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "doc456"})
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
}

func TestUpdateDocFormat_FullReplace(t *testing.T) {
	server := docUpdateServer(t, "old body")
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateDocFormat(context.Background(), UpdateDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
		Content: "new body",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "doc456" {
		t.Errorf("ID = %q, want doc456", result.ID)
	}
	if result.OldID != "doc123" {
		t.Errorf("OldID = %q, want doc123", result.OldID)
	}
	if result.Content != "new body" {
		t.Errorf("Content = %q, want new body", result.Content)
	}
	if result.Replaced != 0 {
		t.Errorf("Replaced = %d, want 0", result.Replaced)
	}
	if result.Message != "Updated doc format item" {
		t.Errorf("Message = %q", result.Message)
	}
	if result.ItemURL != BuildItemURL("board123", "doc456") {
		t.Errorf("ItemURL = %q", result.ItemURL)
	}
}

func TestUpdateDocFormat_FindAndReplace(t *testing.T) {
	server := docUpdateServer(t, "foo bar foo")
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateDocFormat(context.Background(), UpdateDocFormatArgs{
		BoardID:    "board123",
		ItemID:     "doc123",
		OldContent: "foo",
		NewContent: "baz",
		ReplaceAll: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "baz bar baz" {
		t.Errorf("Content = %q, want baz bar baz", result.Content)
	}
	if result.Replaced != 2 {
		t.Errorf("Replaced = %d, want 2", result.Replaced)
	}
	if result.Message != "Replaced 2 occurrence(s) in doc format item" {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestUpdateDocFormat_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    UpdateDocFormatArgs
		wantErr string
	}{
		{"empty board", UpdateDocFormatArgs{BoardID: "", ItemID: "doc123"}, "board_id is required"},
		{"empty item", UpdateDocFormatArgs{BoardID: "board123", ItemID: ""}, "item_id is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClientWithServer("http://unused.invalid")
			if _, err := client.UpdateDocFormat(context.Background(), tt.args); err == nil ||
				!strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateDocFormat_ReadFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.UpdateDocFormat(context.Background(), UpdateDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
		Content: "new",
	})
	if err == nil || !strings.Contains(err.Error(), "failed to read current doc") {
		t.Errorf("error = %v, want failed to read current doc", err)
	}
}

func TestUpdateDocFormat_ResolveFails(t *testing.T) {
	server := docUpdateServer(t, "old body")
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.UpdateDocFormat(context.Background(), UpdateDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
	})
	if err == nil || !strings.Contains(err.Error(), "either content (full replace)") {
		t.Errorf("error = %v, want the either-content error", err)
	}
}

func TestUpdateDocFormat_DeleteFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "doc123",
				"data": map[string]interface{}{"content": "old"},
			})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.UpdateDocFormat(context.Background(), UpdateDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
		Content: "new",
	})
	if err == nil || !strings.Contains(err.Error(), "failed to delete original doc") {
		t.Errorf("error = %v, want failed to delete original doc", err)
	}
}

func TestUpdateDocFormat_RecreateFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "doc123",
				"data": map[string]interface{}{"content": "old"},
			})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.UpdateDocFormat(context.Background(), UpdateDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
		Content: "new",
	})
	if err == nil || !strings.Contains(err.Error(), "failed to recreate doc with updated content") {
		t.Errorf("error = %v, want failed to recreate doc", err)
	}
}

func TestUpdateDocFormat_RecreateMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "doc123",
				"data": map[string]interface{}{"content": "old"},
			})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{not json"))
		}
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.UpdateDocFormat(context.Background(), UpdateDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
		Content: "new",
	})
	if err == nil || !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error = %v, want failed to parse response", err)
	}
}
