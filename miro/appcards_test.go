package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// =============================================================================
// Test server helpers
// =============================================================================

// appCardServerSpec describes the fake Miro API behavior for one app card test:
// the expected request line, an optional request body verifier, and the
// response to send back.
type appCardServerSpec struct {
	wantMethod string
	wantPath   string
	status     int
	response   map[string]interface{}
	verifyBody func(t *testing.T, req map[string]interface{})
}

// newAppCardClient starts a fake Miro API described by spec and returns a
// client pointed at it. The server asserts the request line, decodes and
// verifies the request body, then writes the configured response.
func newAppCardClient(t *testing.T, spec appCardServerSpec) *Client {
	t.Helper()
	if spec.status == 0 {
		spec.status = http.StatusOK
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spec.assertRequestLine(t, r)
		spec.runBodyVerifier(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(spec.status)
		if spec.response == nil {
			return
		}
		if err := json.NewEncoder(w).Encode(spec.response); err != nil {
			panic(err)
		}
	}))
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

func (spec appCardServerSpec) assertRequestLine(t *testing.T, r *http.Request) {
	t.Helper()
	if spec.wantMethod != "" && r.Method != spec.wantMethod {
		t.Errorf("expected %s, got %s", spec.wantMethod, r.Method)
	}
	if spec.wantPath != "" && r.URL.Path != spec.wantPath {
		t.Errorf("unexpected path: %s", r.URL.Path)
	}
}

func (spec appCardServerSpec) runBodyVerifier(t *testing.T, r *http.Request) {
	t.Helper()
	if spec.verifyBody == nil {
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Errorf("failed to decode request: %v", err)
		return
	}
	spec.verifyBody(t, req)
}

// bodySection returns req[key] as an object, failing the test when absent.
func bodySection(t *testing.T, req map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	section, ok := req[key].(map[string]interface{})
	if !ok {
		t.Fatalf("%s not included in request", key)
	}
	return section
}

// bodyFields returns req["data"]["fields"] as a list, failing the test when absent.
func bodyFields(t *testing.T, req map[string]interface{}) []interface{} {
	t.Helper()
	fields, ok := bodySection(t, req, "data")["fields"].([]interface{})
	if !ok {
		t.Fatal("fields not included in request")
	}
	return fields
}

func assertPositionX(t *testing.T, pos *Position, wantX float64) {
	t.Helper()
	if pos == nil {
		t.Error("position is nil")
		return
	}
	if pos.X != wantX {
		t.Errorf("Position.X = %v, want %v", pos.X, wantX)
	}
}

func assertGeometryWidth(t *testing.T, geo *Geometry, wantWidth float64) {
	t.Helper()
	if geo == nil {
		t.Error("geometry is nil")
		return
	}
	if geo.Width != wantWidth {
		t.Errorf("Geometry.Width = %v, want %v", geo.Width, wantWidth)
	}
}

// appCardCreatedResponse builds a minimal created-card payload with the given title.
func appCardCreatedResponse(title string) map[string]interface{} {
	return map[string]interface{}{
		"id":   "appcard123",
		"data": map[string]interface{}{"title": title},
	}
}

// =============================================================================
// App Card Tests
// =============================================================================

func TestCreateAppCard_Success(t *testing.T) {
	client := newAppCardClient(t, appCardServerSpec{
		wantMethod: http.MethodPost,
		wantPath:   "/boards/board123/app_cards",
		status:     http.StatusCreated,
		verifyBody: func(t *testing.T, req map[string]interface{}) {
			data := bodySection(t, req, "data")
			if data["title"] != "Test App Card" {
				t.Errorf("title = %v, want Test App Card", data["title"])
			}
		},
		response: map[string]interface{}{
			"id": "appcard123",
			"data": map[string]interface{}{
				"title":       "Test App Card",
				"description": "Test description",
				"status":      "connected",
			},
		},
	})

	result, err := client.CreateAppCard(context.Background(), CreateAppCardArgs{
		BoardID:     "board123",
		Title:       "Test App Card",
		Description: "Test description",
		Status:      "connected",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "appcard123" {
		t.Errorf("ID = %q, want %q", result.ID, "appcard123")
	}
	if result.Title != "Test App Card" {
		t.Errorf("Title = %q, want %q", result.Title, "Test App Card")
	}
	if result.Status != "connected" {
		t.Errorf("Status = %q, want %q", result.Status, "connected")
	}
}

func TestCreateAppCard_WithFields(t *testing.T) {
	client := newAppCardClient(t, appCardServerSpec{
		status: http.StatusCreated,
		verifyBody: func(t *testing.T, req map[string]interface{}) {
			if fields := bodyFields(t, req); len(fields) != 2 {
				t.Errorf("expected 2 fields, got %d", len(fields))
			}
		},
		response: appCardCreatedResponse("Card with Fields"),
	})

	result, err := client.CreateAppCard(context.Background(), CreateAppCardArgs{
		BoardID: "board123",
		Title:   "Card with Fields",
		Fields: []AppCardField{
			{Value: "Field 1", FillColor: "#FF0000"},
			{Value: "Field 2", TextColor: "#00FF00"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "appcard123" {
		t.Errorf("ID = %q, want %q", result.ID, "appcard123")
	}
}

// TestCreateAppCard_OptionalSections verifies that optional args land in the
// matching top-level request sections (position, geometry, parent).
func TestCreateAppCard_OptionalSections(t *testing.T) {
	tests := []struct {
		name    string
		args    CreateAppCardArgs
		section string
		want    map[string]interface{}
	}{
		{
			name:    "with position",
			args:    CreateAppCardArgs{BoardID: "board123", Title: "Positioned Card", X: 100, Y: 200},
			section: "position",
			want:    map[string]interface{}{"x": float64(100), "y": float64(200)},
		},
		{
			name:    "with geometry",
			args:    CreateAppCardArgs{BoardID: "board123", Title: "Wide Card", Width: 400},
			section: "geometry",
			want:    map[string]interface{}{"width": float64(400)},
		},
		{
			name:    "with parent",
			args:    CreateAppCardArgs{BoardID: "board123", Title: "Card in Frame", ParentID: "frame123"},
			section: "parent",
			want:    map[string]interface{}{"id": "frame123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newAppCardClient(t, appCardServerSpec{
				status: http.StatusCreated,
				verifyBody: func(t *testing.T, req map[string]interface{}) {
					section := bodySection(t, req, tt.section)
					for key, want := range tt.want {
						if section[key] != want {
							t.Errorf("%s.%s = %v, want %v", tt.section, key, section[key], want)
						}
					}
				},
				response: appCardCreatedResponse(tt.args.Title),
			})

			if _, err := client.CreateAppCard(context.Background(), tt.args); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAppCard_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("")
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{"create missing board_id", func() error {
			_, err := client.CreateAppCard(ctx, CreateAppCardArgs{Title: "Test Card"})
			return err
		}},
		{"create missing title", func() error {
			_, err := client.CreateAppCard(ctx, CreateAppCardArgs{BoardID: "board123"})
			return err
		}},
		{"get missing board_id", func() error {
			_, err := client.GetAppCard(ctx, GetAppCardArgs{ItemID: "item123"})
			return err
		}},
		{"get missing item_id", func() error {
			_, err := client.GetAppCard(ctx, GetAppCardArgs{BoardID: "board123"})
			return err
		}},
		{"update no fields provided", func() error {
			_, err := client.UpdateAppCard(ctx, UpdateAppCardArgs{BoardID: "board123", ItemID: "appcard456"})
			return err
		}},
		{"update missing board_id", func() error {
			_, err := client.UpdateAppCard(ctx, UpdateAppCardArgs{ItemID: "item123", Title: "Updated"})
			return err
		}},
		{"update missing item_id", func() error {
			_, err := client.UpdateAppCard(ctx, UpdateAppCardArgs{BoardID: "board123", Title: "Updated"})
			return err
		}},
		{"delete missing board_id", func() error {
			_, err := client.DeleteAppCard(ctx, DeleteAppCardArgs{ItemID: "item123"})
			return err
		}},
		{"delete missing item_id", func() error {
			_, err := client.DeleteAppCard(ctx, DeleteAppCardArgs{BoardID: "board123"})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.call() == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestGetAppCard_Success(t *testing.T) {
	client := newAppCardClient(t, appCardServerSpec{
		wantMethod: http.MethodGet,
		wantPath:   "/boards/board123/app_cards/appcard456",
		response: map[string]interface{}{
			"id": "appcard456",
			"data": map[string]interface{}{
				"title":       "Integration Status",
				"description": "Shows API connection status",
				"status":      "connected",
				"fields": []map[string]interface{}{
					{"value": "Active", "fillColor": "#00FF00"},
					{"value": "Last sync: 1h ago", "textColor": "#888888"},
				},
			},
			"position": map[string]interface{}{
				"x":      100.0,
				"y":      200.0,
				"origin": "center",
			},
			"geometry": map[string]interface{}{
				"width":  320.0,
				"height": 240.0,
			},
			"createdAt":  "2024-01-15T10:00:00Z",
			"modifiedAt": "2024-01-15T12:00:00Z",
		},
	})

	result, err := client.GetAppCard(context.Background(), GetAppCardArgs{
		BoardID: "board123",
		ItemID:  "appcard456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "appcard456" {
		t.Errorf("ID = %q, want %q", result.ID, "appcard456")
	}
	if result.Title != "Integration Status" {
		t.Errorf("Title = %q, want %q", result.Title, "Integration Status")
	}
	if result.Status != "connected" {
		t.Errorf("Status = %q, want %q", result.Status, "connected")
	}
	if len(result.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(result.Fields))
	}
	assertPositionX(t, result.Position, 100.0)
	assertGeometryWidth(t, result.Geometry, 320.0)
	if result.CreatedAt != "2024-01-15T10:00:00Z" {
		t.Errorf("CreatedAt = %q", result.CreatedAt)
	}
}

func TestUpdateAppCard_Success(t *testing.T) {
	client := newAppCardClient(t, appCardServerSpec{
		wantMethod: http.MethodPatch,
		wantPath:   "/boards/board123/app_cards/appcard456",
		verifyBody: func(t *testing.T, req map[string]interface{}) {
			data := bodySection(t, req, "data")
			if data["title"] != "Updated Title" {
				t.Errorf("title = %v, want Updated Title", data["title"])
			}
			if data["status"] != "disconnected" {
				t.Errorf("status = %v, want disconnected", data["status"])
			}
		},
		response: map[string]interface{}{
			"id": "appcard456",
			"data": map[string]interface{}{
				"title":  "Updated Title",
				"status": "disconnected",
			},
		},
	})

	result, err := client.UpdateAppCard(context.Background(), UpdateAppCardArgs{
		BoardID: "board123",
		ItemID:  "appcard456",
		Title:   "Updated Title",
		Status:  "disconnected",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "appcard456" {
		t.Errorf("ID = %q, want %q", result.ID, "appcard456")
	}
	if result.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", result.Title, "Updated Title")
	}
	if result.Status != "disconnected" {
		t.Errorf("Status = %q, want %q", result.Status, "disconnected")
	}
}

func TestUpdateAppCard_WithPosition(t *testing.T) {
	client := newAppCardClient(t, appCardServerSpec{
		verifyBody: func(t *testing.T, req map[string]interface{}) {
			pos := bodySection(t, req, "position")
			if pos["x"] != float64(500) {
				t.Errorf("x = %v, want 500", pos["x"])
			}
			if pos["y"] != float64(600) {
				t.Errorf("y = %v, want 600", pos["y"])
			}
		},
		response: map[string]interface{}{
			"id":   "appcard456",
			"data": map[string]interface{}{"title": "Moved Card"},
		},
	})

	x := float64(500)
	y := float64(600)
	_, err := client.UpdateAppCard(context.Background(), UpdateAppCardArgs{
		BoardID: "board123",
		ItemID:  "appcard456",
		X:       &x,
		Y:       &y,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateAppCard_WithFields(t *testing.T) {
	client := newAppCardClient(t, appCardServerSpec{
		verifyBody: func(t *testing.T, req map[string]interface{}) {
			if fields := bodyFields(t, req); len(fields) != 1 {
				t.Errorf("expected 1 field, got %d", len(fields))
			}
		},
		response: map[string]interface{}{
			"id":   "appcard456",
			"data": map[string]interface{}{"title": "Card with Updated Fields"},
		},
	})

	_, err := client.UpdateAppCard(context.Background(), UpdateAppCardArgs{
		BoardID: "board123",
		ItemID:  "appcard456",
		Fields: []AppCardField{
			{Value: "New Field", FillColor: "#0000FF"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAppCard_Success(t *testing.T) {
	client := newAppCardClient(t, appCardServerSpec{
		wantMethod: http.MethodDelete,
		wantPath:   "/boards/board123/app_cards/appcard456",
		status:     http.StatusNoContent,
	})

	result, err := client.DeleteAppCard(context.Background(), DeleteAppCardArgs{
		BoardID: "board123",
		ItemID:  "appcard456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success = true")
	}
	if result.ItemID != "appcard456" {
		t.Errorf("ItemID = %q, want %q", result.ItemID, "appcard456")
	}
}

func TestDeleteAppCard_APIError(t *testing.T) {
	client := newAppCardClient(t, appCardServerSpec{
		status: http.StatusNotFound,
		response: map[string]interface{}{
			"status":  404,
			"message": "App card not found",
		},
	})

	result, err := client.DeleteAppCard(context.Background(), DeleteAppCardArgs{
		BoardID: "board123",
		ItemID:  "nonexistent",
	})

	if err == nil {
		t.Error("expected error for not found")
	}
	if result.Success {
		t.Error("expected success = false for error case")
	}
}

// =============================================================================
// App Card Field Tests
// =============================================================================

func TestAppCardField_AllFieldOptions(t *testing.T) {
	client := newAppCardClient(t, appCardServerSpec{
		status: http.StatusCreated,
		verifyBody: func(t *testing.T, req map[string]interface{}) {
			fields := bodyFields(t, req)
			if len(fields) != 1 {
				t.Fatalf("expected 1 field, got %d", len(fields))
			}
			field, ok := fields[0].(map[string]interface{})
			if !ok {
				t.Fatalf("field is not an object: %v", fields[0])
			}
			assertFieldAttributes(t, field)
		},
		response: appCardCreatedResponse("Full Field Card"),
	})

	_, err := client.CreateAppCard(context.Background(), CreateAppCardArgs{
		BoardID: "board123",
		Title:   "Full Field Card",
		Fields: []AppCardField{
			{
				Value:     "Status OK",
				FillColor: "#00FF00",
				TextColor: "#FFFFFF",
				IconShape: "round",
				IconURL:   "https://example.com/icon.png",
			},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// assertFieldAttributes verifies every attribute of the full app card field
// used by TestAppCardField_AllFieldOptions.
func assertFieldAttributes(t *testing.T, field map[string]interface{}) {
	t.Helper()
	want := map[string]string{
		"value":     "Status OK",
		"fillColor": "#00FF00",
		"textColor": "#FFFFFF",
		"iconShape": "round",
		"iconUrl":   "https://example.com/icon.png",
	}
	for key, wantValue := range want {
		if field[key] != wantValue {
			t.Errorf("%s = %v, want %s", key, field[key], wantValue)
		}
	}
}
