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
// Native Diagram Read Tests
// =============================================================================

func TestListDiagrams_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/diagrams" {
			t.Errorf("expected /boards/board123/diagrams, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":         "diagram123",
					"type":       "diagram",
					"data":       map[string]interface{}{"title": "Architecture"},
					"position":   map[string]interface{}{"x": 10.5, "y": 20.5},
					"geometry":   map[string]interface{}{"width": 1200.0, "height": 700.0},
					"createdAt":  "2026-05-07T11:14:41Z",
					"modifiedAt": "2026-05-08T09:00:00Z",
					"createdBy":  map[string]interface{}{"id": "user1"},
					"modifiedBy": map[string]interface{}{"id": "user2"},
				},
			},
			"total":  1,
			"size":   1,
			"cursor": "next-cursor",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListDiagrams(context.Background(), ListDiagramsArgs{BoardID: "board123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 1 {
		t.Errorf("Count = %d, want 1", result.Count)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if result.Cursor != "next-cursor" {
		t.Errorf("Cursor = %q, want %q", result.Cursor, "next-cursor")
	}
	if result.Message != "Found 1 diagrams on board" {
		t.Errorf("Message = %q", result.Message)
	}

	d := result.Diagrams[0]
	if d.ID != "diagram123" {
		t.Errorf("ID = %q, want diagram123", d.ID)
	}
	if d.Type != "diagram" {
		t.Errorf("Type = %q, want diagram", d.Type)
	}
	if d.Title != "Architecture" {
		t.Errorf("Title = %q, want Architecture", d.Title)
	}
	if d.X != 10.5 || d.Y != 20.5 {
		t.Errorf("position = (%v, %v), want (10.5, 20.5)", d.X, d.Y)
	}
	if d.Width != 1200 || d.Height != 700 {
		t.Errorf("geometry = (%v, %v), want (1200, 700)", d.Width, d.Height)
	}
	if d.CreatedAt != "2026-05-07T11:14:41Z" || d.ModifiedAt != "2026-05-08T09:00:00Z" {
		t.Errorf("timestamps = (%q, %q)", d.CreatedAt, d.ModifiedAt)
	}
	if d.CreatedBy != "user1" || d.ModifiedBy != "user2" {
		t.Errorf("actors = (%q, %q), want (user1, user2)", d.CreatedBy, d.ModifiedBy)
	}
	if d.ItemURL != BuildItemURL("board123", "diagram123") {
		t.Errorf("ItemURL = %q", d.ItemURL)
	}
}

func TestListDiagrams_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]interface{}{}, "total": 0})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListDiagrams(context.Background(), ListDiagramsArgs{BoardID: "board123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 0 {
		t.Errorf("Count = %d, want 0", result.Count)
	}
	if len(result.Diagrams) != 0 {
		t.Errorf("Diagrams = %d entries, want 0", len(result.Diagrams))
	}
	if result.Message != "Found 0 diagrams on board" {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestListDiagrams_LimitClamping(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		wantLimit string
	}{
		{"zero uses default", 0, "10"},
		{"negative uses default", -5, "10"},
		{"in range is preserved", 25, "25"},
		{"above max is clamped", 500, "50"},
		{"exactly max is preserved", 50, "50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotLimit string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotLimit = r.URL.Query().Get("limit")
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]interface{}{}})
			}))
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			if _, err := client.ListDiagrams(context.Background(), ListDiagramsArgs{
				BoardID: "board123",
				Limit:   tt.limit,
			}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotLimit != tt.wantLimit {
				t.Errorf("limit = %q, want %q", gotLimit, tt.wantLimit)
			}
		})
	}
}

func TestListDiagrams_Cursor(t *testing.T) {
	var gotCursor string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCursor = r.URL.Query().Get("cursor")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]interface{}{}})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.ListDiagrams(context.Background(), ListDiagramsArgs{
		BoardID: "board123",
		Cursor:  "abc123",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotCursor != "abc123" {
		t.Errorf("cursor = %q, want abc123", gotCursor)
	}
}

func TestListDiagrams_InvalidBoardID(t *testing.T) {
	client := newTestClientWithServer("http://unused.invalid")
	_, err := client.ListDiagrams(context.Background(), ListDiagramsArgs{BoardID: ""})
	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("error = %v, want board_id is required", err)
	}
}

func TestListDiagrams_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.ListDiagrams(context.Background(), ListDiagramsArgs{BoardID: "board123"}); err == nil {
		t.Fatal("expected error on HTTP 500")
	}
}

func TestListDiagrams_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{not json"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListDiagrams(context.Background(), ListDiagramsArgs{BoardID: "board123"})
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error = %v, want failed to parse response", err)
	}
}

func TestGetDiagram_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/diagrams/diagram123" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         "diagram123",
			"type":       "diagram",
			"data":       map[string]interface{}{"title": "Architecture"},
			"position":   map[string]interface{}{"x": 1.0, "y": 2.0},
			"geometry":   map[string]interface{}{"width": 1200.0, "height": 700.0},
			"parent":     map[string]interface{}{"id": "frame999"},
			"createdAt":  "2026-05-07T11:14:41Z",
			"modifiedAt": "2026-05-08T09:00:00Z",
			"createdBy":  map[string]interface{}{"id": "user1"},
			"modifiedBy": map[string]interface{}{"id": "user2"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetDiagram(context.Background(), GetDiagramArgs{
		BoardID: "board123",
		ItemID:  "diagram123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "diagram123" {
		t.Errorf("ID = %q, want diagram123", result.ID)
	}
	if result.Title != "Architecture" {
		t.Errorf("Title = %q, want Architecture", result.Title)
	}
	if result.ParentID != "frame999" {
		t.Errorf("ParentID = %q, want frame999", result.ParentID)
	}
	if result.X != 1 || result.Y != 2 {
		t.Errorf("position = (%v, %v), want (1, 2)", result.X, result.Y)
	}
	if result.Width != 1200 || result.Height != 700 {
		t.Errorf("geometry = (%v, %v), want (1200, 700)", result.Width, result.Height)
	}
	if result.CreatedBy != "user1" || result.ModifiedBy != "user2" {
		t.Errorf("actors = (%q, %q), want (user1, user2)", result.CreatedBy, result.ModifiedBy)
	}
	if result.Message != "Retrieved diagram metadata" {
		t.Errorf("Message = %q", result.Message)
	}
	if result.ItemURL != BuildItemURL("board123", "diagram123") {
		t.Errorf("ItemURL = %q", result.ItemURL)
	}
}

func TestGetDiagram_NoParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "diagram123",
			"type": "diagram",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetDiagram(context.Background(), GetDiagramArgs{
		BoardID: "board123",
		ItemID:  "diagram123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ParentID != "" {
		t.Errorf("ParentID = %q, want empty", result.ParentID)
	}
	if result.Title != "" {
		t.Errorf("Title = %q, want empty", result.Title)
	}
}

func TestGetDiagram_InvalidIDs(t *testing.T) {
	tests := []struct {
		name    string
		args    GetDiagramArgs
		wantErr string
	}{
		{"empty board", GetDiagramArgs{BoardID: "", ItemID: "diagram123"}, "board_id is required"},
		{"empty item", GetDiagramArgs{BoardID: "board123", ItemID: ""}, "item_id is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClientWithServer("http://unused.invalid")
			_, err := client.GetDiagram(context.Background(), tt.args)
			if err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestGetDiagram_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.GetDiagram(context.Background(), GetDiagramArgs{
		BoardID: "board123",
		ItemID:  "diagram123",
	}); err == nil {
		t.Fatal("expected error on HTTP 404")
	}
}

func TestGetDiagram_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{not json"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetDiagram(context.Background(), GetDiagramArgs{
		BoardID: "board123",
		ItemID:  "diagram123",
	})
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error = %v, want failed to parse response", err)
	}
}
