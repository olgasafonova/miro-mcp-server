package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newConnectorTestClient starts a test server with the given handler and
// returns a client pointed at it. The server closes automatically at test end.
func newConnectorTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

// checkConnectorField fails the test when got differs from want.
func checkConnectorField[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// connectorOpCase describes one connector CRUD call: request-shape checks,
// the canned response, and a closure running the call plus result assertions.
type connectorOpCase struct {
	name     string
	payload  map[string]interface{} // nil means a 204 No Content response
	status   int                    // optional explicit status before payload
	checkReq func(t *testing.T, r *http.Request)
	call     func(t *testing.T, client *Client) error
}

// runConnectorOpCases drives each case against a server that verifies the
// request shape before responding.
func runConnectorOpCases(t *testing.T, tests []connectorOpCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newConnectorTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				tt.checkReq(t, r)
				if tt.payload == nil {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if tt.status != 0 {
					w.WriteHeader(tt.status)
				}
				json.NewEncoder(w).Encode(tt.payload)
			})
			if err := tt.call(t, client); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

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
			client := newConnectorTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				checkConnectorField(t, "limit", r.URL.Query().Get("limit"), tt.expectLimit)
				writeJSON(w, map[string]interface{}{"data": []interface{}{}})
			})

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

// TestConnector_ValidationErrors covers input validation across the connector
// methods; every case must fail with a message naming the missing field.
func TestConnector_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	offline := newTestClientWithServer("http://localhost")
	ctx := context.Background()

	tests := []struct {
		name    string
		call    func() error
		errText string
	}{
		{"delete with empty board_id", func() error {
			_, err := client.DeleteConnector(ctx, DeleteConnectorArgs{ConnectorID: "conn123"})
			return err
		}, "board_id is required"},
		{"delete with empty connector_id", func() error {
			_, err := client.DeleteConnector(ctx, DeleteConnectorArgs{BoardID: "board123"})
			return err
		}, "connector_id is required"},
		{"create with empty board_id", func() error {
			_, err := client.CreateConnector(ctx, CreateConnectorArgs{StartItemID: "item1", EndItemID: "item2"})
			return err
		}, "board_id is required"},
		{"create with empty start_item_id", func() error {
			_, err := client.CreateConnector(ctx, CreateConnectorArgs{BoardID: "board123", EndItemID: "item2"})
			return err
		}, "start_item_id and end_item_id are required"},
		{"create with empty end_item_id", func() error {
			_, err := client.CreateConnector(ctx, CreateConnectorArgs{BoardID: "board123", StartItemID: "item1"})
			return err
		}, "start_item_id and end_item_id are required"},
		{"get with empty board_id", func() error {
			_, err := client.GetConnector(ctx, GetConnectorArgs{BoardID: "", ConnectorID: "conn456"})
			return err
		}, "board_id is required"},
		{"get with empty connector_id", func() error {
			_, err := client.GetConnector(ctx, GetConnectorArgs{BoardID: "board123", ConnectorID: ""})
			return err
		}, "connector_id is required"},
		{"update with empty board ID", func() error {
			_, err := offline.UpdateConnector(ctx, UpdateConnectorArgs{ConnectorID: "conn123"})
			return err
		}, "board_id is required"},
		{"update with empty connector ID", func() error {
			_, err := offline.UpdateConnector(ctx, UpdateConnectorArgs{BoardID: "board123"})
			return err
		}, "connector_id is required"},
		{"update with no updates provided", func() error {
			_, err := offline.UpdateConnector(ctx, UpdateConnectorArgs{BoardID: "board123", ConnectorID: "conn123"})
			return err
		}, "at least one update field is required"},
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

// TestConnectorWriteOps_Success covers the happy path of the connector
// create, update, and delete methods.
func TestConnectorWriteOps_Success(t *testing.T) {
	runConnectorOpCases(t, []connectorOpCase{
		{
			name:   "create",
			status: http.StatusCreated,
			payload: map[string]interface{}{
				"id":        "connector123",
				"startItem": map[string]interface{}{"id": "item1"},
				"endItem":   map[string]interface{}{"id": "item2"},
			},
			checkReq: func(t *testing.T, r *http.Request) {
				checkConnectorField(t, "method", r.Method, http.MethodPost)
				checkConnectorField(t, "path", r.URL.Path, "/boards/board123/connectors")
			},
			call: func(t *testing.T, client *Client) error {
				result, err := client.CreateConnector(context.Background(), CreateConnectorArgs{BoardID: "board123", StartItemID: "item1", EndItemID: "item2"})
				if err == nil {
					checkConnectorField(t, "ID", result.ID, "connector123")
				}
				return err
			},
		},
		{
			name: "update caption",
			payload: map[string]interface{}{
				"id":       "conn123",
				"captions": []map[string]interface{}{{"content": "Updated Caption"}},
			},
			checkReq: func(t *testing.T, r *http.Request) {
				checkConnectorField(t, "method", r.Method, http.MethodPatch)
				checkConnectorField(t, "path has /connectors/", strings.Contains(r.URL.Path, "/connectors/"), true)
			},
			call: func(t *testing.T, client *Client) error {
				result, err := client.UpdateConnector(context.Background(), UpdateConnectorArgs{BoardID: "board123", ConnectorID: "conn123", Caption: "Updated Caption"})
				if err == nil {
					checkConnectorField(t, "ID", result.ID, "conn123")
				}
				return err
			},
		},
		{
			name: "delete reports success",
			checkReq: func(t *testing.T, r *http.Request) {
				checkConnectorField(t, "method", r.Method, http.MethodDelete)
				checkConnectorField(t, "path", r.URL.Path, "/boards/board123/connectors/conn456")
			},
			call: func(t *testing.T, client *Client) error {
				result, err := client.DeleteConnector(context.Background(), DeleteConnectorArgs{BoardID: "board123", ConnectorID: "conn456"})
				if err == nil {
					checkConnectorField(t, "Success", result.Success, true)
				}
				return err
			},
		},
		{
			name: "delete reports connector ID",
			checkReq: func(t *testing.T, r *http.Request) {
				checkConnectorField(t, "method", r.Method, http.MethodDelete)
				checkConnectorField(t, "path has /connectors/", strings.Contains(r.URL.Path, "/connectors/"), true)
			},
			call: func(t *testing.T, client *Client) error {
				result, err := client.DeleteConnector(context.Background(), DeleteConnectorArgs{BoardID: "board123", ConnectorID: "conn123"})
				if err == nil {
					checkConnectorField(t, "ID", result.ID, "conn123")
				}
				return err
			},
		},
	})
}

// TestConnectorReadOps_Success covers the happy path of the connector list
// and get methods.
func TestConnectorReadOps_Success(t *testing.T) {
	runConnectorOpCases(t, []connectorOpCase{
		{
			name: "list",
			payload: map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":        "conn1",
						"startItem": map[string]interface{}{"id": "item1"},
						"endItem":   map[string]interface{}{"id": "item2"},
					},
				},
				"size": 1,
			},
			checkReq: func(t *testing.T, r *http.Request) {
				checkConnectorField(t, "method", r.Method, http.MethodGet)
				checkConnectorField(t, "path prefix", strings.HasPrefix(r.URL.Path, "/boards/board123/connectors"), true)
			},
			call: func(t *testing.T, client *Client) error {
				result, err := client.ListConnectors(context.Background(), ListConnectorsArgs{BoardID: "board123"})
				if err == nil {
					checkConnectorField(t, "Count", result.Count, 1)
					checkConnectorField(t, "first connector ID", result.Connectors[0].ID, "conn1")
				}
				return err
			},
		},
		{
			name: "get",
			payload: map[string]interface{}{
				"id":        "conn456",
				"startItem": map[string]interface{}{"item": "start123"},
				"endItem":   map[string]interface{}{"item": "end456"},
				"style":     map[string]interface{}{"strokeColor": "#000000", "strokeWidth": "2.0"},
			},
			checkReq: func(t *testing.T, r *http.Request) {
				checkConnectorField(t, "method", r.Method, http.MethodGet)
				checkConnectorField(t, "path", r.URL.Path, "/boards/board123/connectors/conn456")
			},
			call: func(t *testing.T, client *Client) error {
				result, err := client.GetConnector(context.Background(), GetConnectorArgs{BoardID: "board123", ConnectorID: "conn456"})
				if err == nil {
					checkConnectorField(t, "ID", result.ID, "conn456")
					checkConnectorField(t, "StartItemID", result.StartItemID, "start123")
				}
				return err
			},
		},
	})
}

func TestGetConnector_WithAllDetails(t *testing.T) {
	client := newConnectorTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"id":        "conn456",
			"shape":     "elbowed",
			"startItem": map[string]interface{}{"item": "start123"},
			"endItem":   map[string]interface{}{"item": "end456"},
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
	})

	result, err := client.GetConnector(context.Background(), GetConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkConnectorField(t, "StartCap", result.StartCap, "arrow")
	checkConnectorField(t, "EndCap", result.EndCap, "stealth")
	checkConnectorField(t, "Color", result.Color, "#FF0000")
	checkConnectorField(t, "Caption", result.Caption, "Label text")
	if result.CreatedAt == "" {
		t.Error("CreatedAt should be set")
	}
	if result.ModifiedAt == "" {
		t.Error("ModifiedAt should be set")
	}
}

func TestUpdateConnector_WithStyle(t *testing.T) {
	client := newConnectorTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify shape (style) field
		checkConnectorField(t, "shape", body["shape"], interface{}("curved"))
		writeJSON(w, map[string]interface{}{"id": "conn123"})
	})

	result, err := client.UpdateConnector(context.Background(), UpdateConnectorArgs{
		BoardID:     "board123",
		ConnectorID: "conn123",
		Style:       "curved",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkConnectorField(t, "Success", result.Success, true)
}

func TestUpdateConnector_WithCapsAndColor(t *testing.T) {
	client := newConnectorTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify style object with caps and color
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected style object in request body")
		} else {
			checkConnectorField(t, "startStrokeCap", style["startStrokeCap"], interface{}("arrow"))
			checkConnectorField(t, "endStrokeCap", style["endStrokeCap"], interface{}("stealth"))
			checkConnectorField(t, "strokeColor", style["strokeColor"], interface{}("#ff0000"))
		}
		writeJSON(w, map[string]interface{}{"id": "conn123"})
	})

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
	checkConnectorField(t, "Success", result.Success, true)
}

// TestListConnectors_QueryParams verifies that pagination arguments reach the
// API as query-string parameters.
func TestListConnectors_QueryParams(t *testing.T) {
	tests := []struct {
		name      string
		args      ListConnectorsArgs
		wantParam string
		wantValue string
	}{
		{
			name:      "cursor is forwarded",
			args:      ListConnectorsArgs{BoardID: "board123", Cursor: "next-page-cursor"},
			wantParam: "cursor",
			wantValue: "next-page-cursor",
		},
		{
			name:      "limit is forwarded",
			args:      ListConnectorsArgs{BoardID: "board123", Limit: 25},
			wantParam: "limit",
			wantValue: "25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newConnectorTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				checkConnectorField(t, tt.wantParam, r.URL.Query().Get(tt.wantParam), tt.wantValue)
				writeJSON(w, map[string]interface{}{"data": []map[string]interface{}{}})
			})

			if _, err := client.ListConnectors(context.Background(), tt.args); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
