package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListConnectors_LimitBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		inputLimit  int
		expectLimit string
	}{
		{"zero limit defaults to 50", 0, "50"},
		{"limit below 10 becomes 10", 5, "10"},
		{"limit above 100 becomes 100", 200, "100"},
		{"valid limit passes through", 30, "30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				limit := r.URL.Query().Get("limit")
				if limit != tt.expectLimit {
					t.Errorf("limit = %q, want %q", limit, tt.expectLimit)
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": []interface{}{},
				})
			}))
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			_, err := client.ListConnectors(context.Background(), ListConnectorsArgs{
				BoardID: "board123",
				Limit:   tt.inputLimit,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDeleteConnector_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    DeleteConnectorArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    DeleteConnectorArgs{ConnectorID: "conn123"},
			errText: "board_id is required",
		},
		{
			name:    "empty connector_id",
			args:    DeleteConnectorArgs{BoardID: "board123"},
			errText: "connector_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.DeleteConnector(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestDeleteConnector_SuccessPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/connectors/conn456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteConnector(context.Background(), DeleteConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success to be true")
	}
}

func TestCreateConnector_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/connectors" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "connector123",
			"startItem": map[string]interface{}{
				"id": "item1",
			},
			"endItem": map[string]interface{}{
				"id": "item2",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateConnector(context.Background(), CreateConnectorArgs{
		BoardID:     "board123",
		StartItemID: "item1",
		EndItemID:   "item2",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "connector123" {
		t.Errorf("ID = %q, want 'connector123'", result.ID)
	}
}

func TestCreateConnector_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateConnectorArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateConnectorArgs{StartItemID: "item1", EndItemID: "item2"},
			errText: "board_id is required",
		},
		{
			name:    "empty start_item_id",
			args:    CreateConnectorArgs{BoardID: "board123", EndItemID: "item2"},
			errText: "start_item_id and end_item_id are required",
		},
		{
			name:    "empty end_item_id",
			args:    CreateConnectorArgs{BoardID: "board123", StartItemID: "item1"},
			errText: "start_item_id and end_item_id are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateConnector(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestListConnectors_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards/board123/connectors") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id": "conn1",
					"startItem": map[string]interface{}{
						"id": "item1",
					},
					"endItem": map[string]interface{}{
						"id": "item2",
					},
				},
			},
			"size": 1,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListConnectors(context.Background(), ListConnectorsArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 1 {
		t.Errorf("Count = %d, want 1", result.Count)
	}
	if result.Connectors[0].ID != "conn1" {
		t.Errorf("first connector ID = %q, want 'conn1'", result.Connectors[0].ID)
	}
}

func TestGetConnector_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/connectors/conn456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "conn456",
			"startItem": map[string]interface{}{
				"item": "start123",
			},
			"endItem": map[string]interface{}{
				"item": "end456",
			},
			"style": map[string]interface{}{
				"strokeColor": "#000000",
				"strokeWidth": "2.0",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetConnector(context.Background(), GetConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "conn456" {
		t.Errorf("ID = %q, want 'conn456'", result.ID)
	}
	if result.StartItemID != "start123" {
		t.Errorf("StartItemID = %q, want 'start123'", result.StartItemID)
	}
}

func TestGetConnector_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.GetConnector(context.Background(), GetConnectorArgs{
		BoardID:     "",
		ConnectorID: "conn456",
	})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestGetConnector_EmptyConnectorID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.GetConnector(context.Background(), GetConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "",
	})

	if err == nil {
		t.Fatal("expected error for empty connector_id")
	}
	if !strings.Contains(err.Error(), "connector_id is required") {
		t.Errorf("expected 'connector_id is required' error, got: %v", err)
	}
}

func TestGetConnector_WithAllDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "conn456",
			"shape": "elbowed",
			"startItem": map[string]interface{}{
				"item": "start123",
			},
			"endItem": map[string]interface{}{
				"item": "end456",
			},
			"style": map[string]interface{}{
				"startStrokeCap": "arrow",
				"endStrokeCap":   "stealth",
				"color":          "#FF0000",
			},
			"captions": []map[string]interface{}{
				{"content": "Label text"},
			},
			"createdAt":  "2024-01-15T10:00:00Z",
			"modifiedAt": "2024-01-16T15:30:00Z",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetConnector(context.Background(), GetConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StartCap != "arrow" {
		t.Errorf("StartCap = %q, want 'arrow'", result.StartCap)
	}
	if result.EndCap != "stealth" {
		t.Errorf("EndCap = %q, want 'stealth'", result.EndCap)
	}
	if result.Color != "#FF0000" {
		t.Errorf("Color = %q, want '#FF0000'", result.Color)
	}
	if result.Caption != "Label text" {
		t.Errorf("Caption = %q, want 'Label text'", result.Caption)
	}
	if result.CreatedAt == "" {
		t.Error("CreatedAt should be set")
	}
	if result.ModifiedAt == "" {
		t.Error("ModifiedAt should be set")
	}
}

func TestUpdateConnector_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/connectors/") {
			t.Errorf("expected connectors path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "conn123",
			"captions": []map[string]interface{}{
				{"content": "Updated Caption"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateConnector(context.Background(), UpdateConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn123",
		Caption:     "Updated Caption",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "conn123" {
		t.Errorf("ID = %q, want 'conn123'", result.ID)
	}
}

func TestUpdateConnector_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    UpdateConnectorArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    UpdateConnectorArgs{ConnectorID: "conn123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty connector ID",
			args:    UpdateConnectorArgs{BoardID: "board123"},
			wantErr: "connector_id is required",
		},
		{
			name:    "no updates provided",
			args:    UpdateConnectorArgs{BoardID: "board123", ConnectorID: "conn123"},
			wantErr: "at least one update field is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UpdateConnector(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestUpdateConnector_WithStyle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify shape (style) field
		if body["shape"] != "curved" {
			t.Errorf("shape = %v, want 'curved'", body["shape"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "conn123"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateConnector(context.Background(), UpdateConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn123",
		Style:       "curved",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success to be true")
	}
}

func TestUpdateConnector_WithCapsAndColor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify style object with caps and color
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected style object in request body")
		} else {
			if style["startStrokeCap"] != "arrow" {
				t.Errorf("startStrokeCap = %v, want 'arrow'", style["startStrokeCap"])
			}
			if style["endStrokeCap"] != "stealth" {
				t.Errorf("endStrokeCap = %v, want 'stealth'", style["endStrokeCap"])
			}
			if style["strokeColor"] != "#ff0000" {
				t.Errorf("strokeColor = %v, want '#ff0000'", style["strokeColor"])
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "conn123"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateConnector(context.Background(), UpdateConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn123",
		StartCap:    "arrow",
		EndCap:      "stealth",
		Color:       "#ff0000",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success to be true")
	}
}

func TestDeleteConnector_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/connectors/") {
			t.Errorf("expected connectors path, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteConnector(context.Background(), DeleteConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "conn123" {
		t.Errorf("ID = %q, want 'conn123'", result.ID)
	}
}

func TestListConnectors_WithCursor(t *testing.T) {
	// Tests ListConnectors with cursor pagination
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") != "next-page-cursor" {
			t.Errorf("cursor = %v, want 'next-page-cursor'", r.URL.Query().Get("cursor"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "conn1", "type": "connector"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListConnectors(context.Background(), ListConnectorsArgs{
		BoardID: "board123",
		Cursor:  "next-page-cursor",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListConnectors_WithLimitParam(t *testing.T) {
	// Tests ListConnectors with limit parameter
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "25" {
			t.Errorf("limit = %v, want '25'", r.URL.Query().Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListConnectors(context.Background(), ListConnectorsArgs{
		BoardID: "board123",
		Limit:   25,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
