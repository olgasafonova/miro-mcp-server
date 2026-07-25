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
// Table Tests
// =============================================================================

func TestListTables_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/data_table_formats" {
			t.Errorf("expected /boards/board123/data_table_formats, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":         "table123",
					"type":       "data_table_format",
					"position":   map[string]interface{}{"x": 10.5, "y": 20.5},
					"geometry":   map[string]interface{}{"width": 300.0, "height": 200.0},
					"createdAt":  "2026-01-01T00:00:00Z",
					"modifiedAt": "2026-01-02T00:00:00Z",
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
	result, err := client.ListTables(context.Background(), ListTablesArgs{BoardID: "board123"})
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
	if result.Message != "Found 1 tables on board" {
		t.Errorf("Message = %q", result.Message)
	}

	tbl := result.Tables[0]
	if tbl.ID != "table123" {
		t.Errorf("ID = %q, want table123", tbl.ID)
	}
	if tbl.Type != "data_table_format" {
		t.Errorf("Type = %q, want data_table_format", tbl.Type)
	}
	if tbl.X != 10.5 || tbl.Y != 20.5 {
		t.Errorf("position = (%v, %v), want (10.5, 20.5)", tbl.X, tbl.Y)
	}
	if tbl.Width != 300 || tbl.Height != 200 {
		t.Errorf("geometry = (%v, %v), want (300, 200)", tbl.Width, tbl.Height)
	}
	if tbl.CreatedAt != "2026-01-01T00:00:00Z" || tbl.ModifiedAt != "2026-01-02T00:00:00Z" {
		t.Errorf("timestamps = (%q, %q)", tbl.CreatedAt, tbl.ModifiedAt)
	}
	if tbl.CreatedBy != "user1" || tbl.ModifiedBy != "user2" {
		t.Errorf("actors = (%q, %q), want (user1, user2)", tbl.CreatedBy, tbl.ModifiedBy)
	}
	if tbl.ItemURL != BuildItemURL("board123", "table123") {
		t.Errorf("ItemURL = %q", tbl.ItemURL)
	}
}

func TestListTables_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]interface{}{}, "total": 0})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListTables(context.Background(), ListTablesArgs{BoardID: "board123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 0 {
		t.Errorf("Count = %d, want 0", result.Count)
	}
	if len(result.Tables) != 0 {
		t.Errorf("Tables = %d entries, want 0", len(result.Tables))
	}
	if result.Message != "Found 0 tables on board" {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestListTables_LimitClamping(t *testing.T) {
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
			if _, err := client.ListTables(context.Background(), ListTablesArgs{
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

func TestListTables_Cursor(t *testing.T) {
	var gotCursor string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCursor = r.URL.Query().Get("cursor")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]interface{}{}})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.ListTables(context.Background(), ListTablesArgs{
		BoardID: "board123",
		Cursor:  "abc123",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotCursor != "abc123" {
		t.Errorf("cursor = %q, want abc123", gotCursor)
	}
}

func TestListTables_InvalidBoardID(t *testing.T) {
	client := newTestClientWithServer("http://unused.invalid")
	_, err := client.ListTables(context.Background(), ListTablesArgs{BoardID: ""})
	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("error = %v, want board_id is required", err)
	}
}

func TestListTables_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.ListTables(context.Background(), ListTablesArgs{BoardID: "board123"}); err == nil {
		t.Fatal("expected error on HTTP 500")
	}
}

func TestListTables_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{not json"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListTables(context.Background(), ListTablesArgs{BoardID: "board123"})
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error = %v, want failed to parse response", err)
	}
}

func TestGetTable_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/data_table_formats/table123" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         "table123",
			"type":       "data_table_format",
			"position":   map[string]interface{}{"x": 1.0, "y": 2.0},
			"geometry":   map[string]interface{}{"width": 100.0, "height": 50.0},
			"parent":     map[string]interface{}{"id": "frame999"},
			"createdAt":  "2026-01-01T00:00:00Z",
			"modifiedAt": "2026-01-02T00:00:00Z",
			"createdBy":  map[string]interface{}{"id": "user1"},
			"modifiedBy": map[string]interface{}{"id": "user2"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetTable(context.Background(), GetTableArgs{
		BoardID: "board123",
		ItemID:  "table123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "table123" {
		t.Errorf("ID = %q, want table123", result.ID)
	}
	if result.ParentID != "frame999" {
		t.Errorf("ParentID = %q, want frame999", result.ParentID)
	}
	if result.X != 1 || result.Y != 2 {
		t.Errorf("position = (%v, %v), want (1, 2)", result.X, result.Y)
	}
	if result.Width != 100 || result.Height != 50 {
		t.Errorf("geometry = (%v, %v), want (100, 50)", result.Width, result.Height)
	}
	if result.CreatedBy != "user1" || result.ModifiedBy != "user2" {
		t.Errorf("actors = (%q, %q), want (user1, user2)", result.CreatedBy, result.ModifiedBy)
	}
	if result.Message != "Retrieved table metadata" {
		t.Errorf("Message = %q", result.Message)
	}
	if result.ItemURL != BuildItemURL("board123", "table123") {
		t.Errorf("ItemURL = %q", result.ItemURL)
	}
}

func TestGetTable_NoParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "table123",
			"type": "data_table_format",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetTable(context.Background(), GetTableArgs{
		BoardID: "board123",
		ItemID:  "table123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ParentID != "" {
		t.Errorf("ParentID = %q, want empty", result.ParentID)
	}
}

func TestGetTable_InvalidIDs(t *testing.T) {
	tests := []struct {
		name    string
		args    GetTableArgs
		wantErr string
	}{
		{"empty board", GetTableArgs{BoardID: "", ItemID: "table123"}, "board_id is required"},
		{"empty item", GetTableArgs{BoardID: "board123", ItemID: ""}, "item_id is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClientWithServer("http://unused.invalid")
			_, err := client.GetTable(context.Background(), tt.args)
			if err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestGetTable_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.GetTable(context.Background(), GetTableArgs{
		BoardID: "board123",
		ItemID:  "table123",
	}); err == nil {
		t.Fatal("expected error on HTTP 404")
	}
}

func TestGetTable_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{not json"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetTable(context.Background(), GetTableArgs{
		BoardID: "board123",
		ItemID:  "table123",
	})
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error = %v, want failed to parse response", err)
	}
}
