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

// tblCheckRequest verifies the HTTP method and exact URL path of a request.
func tblCheckRequest(t *testing.T, r *http.Request, method, wantPath string) {
	t.Helper()
	if r.Method != method {
		t.Errorf("expected %s, got %s", method, r.Method)
	}
	if r.URL.Path != wantPath {
		t.Errorf("expected %s, got %s", wantPath, r.URL.Path)
	}
}

// tblCheckEq verifies a single result field against its expected value.
func tblCheckEq(t *testing.T, name string, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// tblServeJSON writes a JSON response.
func tblServeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// tblListTables calls ListTables with the default test board and discards the result.
func tblListTables(c *Client) error {
	_, err := c.ListTables(context.Background(), ListTablesArgs{BoardID: "board123"})
	return err
}

// tblGetTable calls GetTable with the default test board and item and discards the result.
func tblGetTable(c *Client) error {
	_, err := c.GetTable(context.Background(), GetTableArgs{BoardID: "board123", ItemID: "table123"})
	return err
}

func TestListTables_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tblCheckRequest(t, r, http.MethodGet, "/boards/board123/data_table_formats")

		tblServeJSON(w, map[string]interface{}{
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

	tblCheckEq(t, "Count", result.Count, 1)
	tblCheckEq(t, "Total", result.Total, 1)
	tblCheckEq(t, "Cursor", result.Cursor, "next-cursor")
	tblCheckEq(t, "Message", result.Message, "Found 1 tables on board")

	tbl := result.Tables[0]
	tblCheckEq(t, "ID", tbl.ID, "table123")
	tblCheckEq(t, "Type", tbl.Type, "data_table_format")
	tblCheckEq(t, "X", tbl.X, 10.5)
	tblCheckEq(t, "Y", tbl.Y, 20.5)
	tblCheckEq(t, "Width", tbl.Width, float64(300))
	tblCheckEq(t, "Height", tbl.Height, float64(200))
	tblCheckEq(t, "CreatedAt", tbl.CreatedAt, "2026-01-01T00:00:00Z")
	tblCheckEq(t, "ModifiedAt", tbl.ModifiedAt, "2026-01-02T00:00:00Z")
	tblCheckEq(t, "CreatedBy", tbl.CreatedBy, "user1")
	tblCheckEq(t, "ModifiedBy", tbl.ModifiedBy, "user2")
	tblCheckEq(t, "ItemURL", tbl.ItemURL, BuildItemURL("board123", "table123"))
}

func TestListTables_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		tblServeJSON(w, map[string]interface{}{"data": []map[string]interface{}{}, "total": 0})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListTables(context.Background(), ListTablesArgs{BoardID: "board123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tblCheckEq(t, "Count", result.Count, 0)
	tblCheckEq(t, "len(Tables)", len(result.Tables), 0)
	tblCheckEq(t, "Message", result.Message, "Found 0 tables on board")
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
				tblServeJSON(w, map[string]interface{}{"data": []map[string]interface{}{}})
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
		tblServeJSON(w, map[string]interface{}{"data": []map[string]interface{}{}})
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

func TestTables_APIError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		call   func(*Client) error
	}{
		{"list tables HTTP 500", http.StatusInternalServerError, tblListTables},
		{"get table HTTP 404", http.StatusNotFound, tblGetTable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			if err := tt.call(client); err == nil {
				t.Fatalf("expected error on HTTP %d", tt.status)
			}
		})
	}
}

func TestTables_MalformedJSON(t *testing.T) {
	tests := []struct {
		name string
		call func(*Client) error
	}{
		{"list tables", tblListTables},
		{"get table", tblGetTable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte("{not json"))
			}))
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			err := tt.call(client)
			if err == nil {
				t.Fatal("expected parse error")
			}
			if !strings.Contains(err.Error(), "failed to parse response") {
				t.Errorf("error = %v, want failed to parse response", err)
			}
		})
	}
}

func TestGetTable_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tblCheckRequest(t, r, http.MethodGet, "/boards/board123/data_table_formats/table123")

		tblServeJSON(w, map[string]interface{}{
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

	tblCheckEq(t, "ID", result.ID, "table123")
	tblCheckEq(t, "ParentID", result.ParentID, "frame999")
	tblCheckEq(t, "X", result.X, float64(1))
	tblCheckEq(t, "Y", result.Y, float64(2))
	tblCheckEq(t, "Width", result.Width, float64(100))
	tblCheckEq(t, "Height", result.Height, float64(50))
	tblCheckEq(t, "CreatedBy", result.CreatedBy, "user1")
	tblCheckEq(t, "ModifiedBy", result.ModifiedBy, "user2")
	tblCheckEq(t, "Message", result.Message, "Retrieved table metadata")
	tblCheckEq(t, "ItemURL", result.ItemURL, BuildItemURL("board123", "table123"))
}

func TestGetTable_NoParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		tblServeJSON(w, map[string]interface{}{
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
	tblCheckEq(t, "ParentID", result.ParentID, "")
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
