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
		if got := r.Method + " " + r.URL.Path; got != "GET /boards/board123/diagrams" {
			t.Errorf("request = %q, want GET /boards/board123/diagrams", got)
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

	requireIntField(t, "Count", result.Count, 1)
	requireIntField(t, "Total", result.Total, 1)
	requireStringField(t, "Cursor", result.Cursor, "next-cursor")
	requireStringField(t, "Message", result.Message, "Found 1 diagrams on board")

	d := result.Diagrams[0]
	requireStringField(t, "ID", d.ID, "diagram123")
	requireStringField(t, "Type", d.Type, "diagram")
	requireStringField(t, "Title", d.Title, "Architecture")
	if [4]float64{d.X, d.Y, d.Width, d.Height} != [4]float64{10.5, 20.5, 1200, 700} {
		t.Errorf("position/geometry = (%v, %v, %v, %v), want (10.5, 20.5, 1200, 700)", d.X, d.Y, d.Width, d.Height)
	}
	requireStringField(t, "CreatedAt", d.CreatedAt, "2026-05-07T11:14:41Z")
	requireStringField(t, "ModifiedAt", d.ModifiedAt, "2026-05-08T09:00:00Z")
	requireStringField(t, "CreatedBy", d.CreatedBy, "user1")
	requireStringField(t, "ModifiedBy", d.ModifiedBy, "user2")
	requireStringField(t, "ItemURL", d.ItemURL, BuildItemURL("board123", "diagram123"))
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
	requireIntField(t, "Count", result.Count, 0)
	requireIntField(t, "Diagrams entries", len(result.Diagrams), 0)
	requireStringField(t, "Message", result.Message, "Found 0 diagrams on board")
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

// TestDiagramReads_ErrorResponses covers API errors and malformed JSON for
// both ListDiagrams and GetDiagram.
func TestDiagramReads_ErrorResponses(t *testing.T) {
	listDiagrams := func(client *Client) error {
		_, err := client.ListDiagrams(context.Background(), ListDiagramsArgs{BoardID: "board123"})
		return err
	}
	getDiagram := func(client *Client) error {
		_, err := client.GetDiagram(context.Background(), GetDiagramArgs{BoardID: "board123", ItemID: "diagram123"})
		return err
	}

	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
		call    func(*Client) error
	}{
		{"list diagrams HTTP 500", http.StatusInternalServerError, "", "", listDiagrams},
		{"get diagram HTTP 404", http.StatusNotFound, "", "", getDiagram},
		{"list diagrams malformed JSON", http.StatusOK, "{not json", "failed to parse response", listDiagrams},
		{"get diagram malformed JSON", http.StatusOK, "{not json", "failed to parse response", getDiagram},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			err := tt.call(newTestClientWithServer(server.URL))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestGetDiagram_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Method + " " + r.URL.Path; got != "GET /boards/board123/diagrams/diagram123" {
			t.Errorf("request = %q, want GET /boards/board123/diagrams/diagram123", got)
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

	requireStringField(t, "ID", result.ID, "diagram123")
	requireStringField(t, "Title", result.Title, "Architecture")
	requireStringField(t, "ParentID", result.ParentID, "frame999")
	if [4]float64{result.X, result.Y, result.Width, result.Height} != [4]float64{1, 2, 1200, 700} {
		t.Errorf("position/geometry = (%v, %v, %v, %v), want (1, 2, 1200, 700)", result.X, result.Y, result.Width, result.Height)
	}
	requireStringField(t, "CreatedBy", result.CreatedBy, "user1")
	requireStringField(t, "ModifiedBy", result.ModifiedBy, "user2")
	requireStringField(t, "Message", result.Message, "Retrieved diagram metadata")
	requireStringField(t, "ItemURL", result.ItemURL, BuildItemURL("board123", "diagram123"))
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
	requireStringField(t, "ParentID", result.ParentID, "")
	requireStringField(t, "Title", result.Title, "")
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
