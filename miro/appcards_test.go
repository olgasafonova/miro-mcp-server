package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// =============================================================================
// App Card Tests
// =============================================================================

func TestCreateAppCard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/app_cards" {
			t.Errorf("expected /boards/board123/app_cards, got %s", r.URL.Path)
		}

		// Verify request body
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		data, ok := req["data"].(map[string]interface{})
		if !ok {
			t.Fatal("missing data field in request")
		}
		if data["title"] != "Test App Card" {
			t.Errorf("title = %v, want Test App Card", data["title"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "appcard123",
			"data": map[string]interface{}{
				"title":       "Test App Card",
				"description": "Test description",
				"status":      "connected",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		data := req["data"].(map[string]interface{})
		fields, ok := data["fields"].([]interface{})
		if !ok {
			t.Fatal("fields not included in request")
		}
		if len(fields) != 2 {
			t.Errorf("expected 2 fields, got %d", len(fields))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "appcard123",
			"data": map[string]interface{}{
				"title":  "Card with Fields",
				"status": "connected",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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

func TestCreateAppCard_WithPosition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		pos, ok := req["position"].(map[string]interface{})
		if !ok {
			t.Fatal("position not included in request")
		}
		if pos["x"] != float64(100) {
			t.Errorf("x = %v, want 100", pos["x"])
		}
		if pos["y"] != float64(200) {
			t.Errorf("y = %v, want 200", pos["y"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard123",
			"data": map[string]interface{}{"title": "Positioned Card"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateAppCard(context.Background(), CreateAppCardArgs{
		BoardID: "board123",
		Title:   "Positioned Card",
		X:       100,
		Y:       200,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateAppCard_WithGeometry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		geo, ok := req["geometry"].(map[string]interface{})
		if !ok {
			t.Fatal("geometry not included in request")
		}
		if geo["width"] != float64(400) {
			t.Errorf("width = %v, want 400", geo["width"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard123",
			"data": map[string]interface{}{"title": "Wide Card"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateAppCard(context.Background(), CreateAppCardArgs{
		BoardID: "board123",
		Title:   "Wide Card",
		Width:   400,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateAppCard_WithParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		parent, ok := req["parent"].(map[string]interface{})
		if !ok {
			t.Fatal("parent not included in request")
		}
		if parent["id"] != "frame123" {
			t.Errorf("parent id = %v, want frame123", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard123",
			"data": map[string]interface{}{"title": "Card in Frame"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateAppCard(context.Background(), CreateAppCardArgs{
		BoardID:  "board123",
		Title:    "Card in Frame",
		ParentID: "frame123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateAppCard_MissingBoardID(t *testing.T) {
	client := newTestClientWithServer("")
	_, err := client.CreateAppCard(context.Background(), CreateAppCardArgs{
		Title: "Test Card",
	})

	if err == nil {
		t.Error("expected error for missing board_id")
	}
}

func TestCreateAppCard_MissingTitle(t *testing.T) {
	client := newTestClientWithServer("")
	_, err := client.CreateAppCard(context.Background(), CreateAppCardArgs{
		BoardID: "board123",
	})

	if err == nil {
		t.Error("expected error for missing title")
	}
}

func TestGetAppCard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/app_cards/appcard456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
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
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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
	if result.Position == nil {
		t.Error("position is nil")
	} else if result.Position.X != 100.0 {
		t.Errorf("Position.X = %v, want 100", result.Position.X)
	}
	if result.Geometry == nil {
		t.Error("geometry is nil")
	} else if result.Geometry.Width != 320.0 {
		t.Errorf("Geometry.Width = %v, want 320", result.Geometry.Width)
	}
	if result.CreatedAt != "2024-01-15T10:00:00Z" {
		t.Errorf("CreatedAt = %q", result.CreatedAt)
	}
}

func TestGetAppCard_MissingBoardID(t *testing.T) {
	client := newTestClientWithServer("")
	_, err := client.GetAppCard(context.Background(), GetAppCardArgs{
		ItemID: "item123",
	})

	if err == nil {
		t.Error("expected error for missing board_id")
	}
}

func TestGetAppCard_MissingItemID(t *testing.T) {
	client := newTestClientWithServer("")
	_, err := client.GetAppCard(context.Background(), GetAppCardArgs{
		BoardID: "board123",
	})

	if err == nil {
		t.Error("expected error for missing item_id")
	}
}

func TestUpdateAppCard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/app_cards/appcard456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		data, ok := req["data"].(map[string]interface{})
		if !ok {
			t.Fatal("missing data in request")
		}
		if data["title"] != "Updated Title" {
			t.Errorf("title = %v, want Updated Title", data["title"])
		}
		if data["status"] != "disconnected" {
			t.Errorf("status = %v, want disconnected", data["status"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "appcard456",
			"data": map[string]interface{}{
				"title":  "Updated Title",
				"status": "disconnected",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		pos, ok := req["position"].(map[string]interface{})
		if !ok {
			t.Fatal("position not included in request")
		}
		if pos["x"] != float64(500) {
			t.Errorf("x = %v, want 500", pos["x"])
		}
		if pos["y"] != float64(600) {
			t.Errorf("y = %v, want 600", pos["y"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard456",
			"data": map[string]interface{}{"title": "Moved Card"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		data, ok := req["data"].(map[string]interface{})
		if !ok {
			t.Fatal("missing data in request")
		}
		fields, ok := data["fields"].([]interface{})
		if !ok {
			t.Fatal("fields not included in request")
		}
		if len(fields) != 1 {
			t.Errorf("expected 1 field, got %d", len(fields))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard456",
			"data": map[string]interface{}{"title": "Card with Updated Fields"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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

func TestUpdateAppCard_NoFieldsProvided(t *testing.T) {
	client := newTestClientWithServer("")
	_, err := client.UpdateAppCard(context.Background(), UpdateAppCardArgs{
		BoardID: "board123",
		ItemID:  "appcard456",
	})

	if err == nil {
		t.Error("expected error when no fields provided")
	}
}

func TestUpdateAppCard_MissingBoardID(t *testing.T) {
	client := newTestClientWithServer("")
	_, err := client.UpdateAppCard(context.Background(), UpdateAppCardArgs{
		ItemID: "item123",
		Title:  "Updated",
	})

	if err == nil {
		t.Error("expected error for missing board_id")
	}
}

func TestUpdateAppCard_MissingItemID(t *testing.T) {
	client := newTestClientWithServer("")
	_, err := client.UpdateAppCard(context.Background(), UpdateAppCardArgs{
		BoardID: "board123",
		Title:   "Updated",
	})

	if err == nil {
		t.Error("expected error for missing item_id")
	}
}

func TestDeleteAppCard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/app_cards/appcard456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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

func TestDeleteAppCard_MissingBoardID(t *testing.T) {
	client := newTestClientWithServer("")
	_, err := client.DeleteAppCard(context.Background(), DeleteAppCardArgs{
		ItemID: "item123",
	})

	if err == nil {
		t.Error("expected error for missing board_id")
	}
}

func TestDeleteAppCard_MissingItemID(t *testing.T) {
	client := newTestClientWithServer("")
	_, err := client.DeleteAppCard(context.Background(), DeleteAppCardArgs{
		BoardID: "board123",
	})

	if err == nil {
		t.Error("expected error for missing item_id")
	}
}

func TestDeleteAppCard_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  404,
			"message": "App card not found",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		data := req["data"].(map[string]interface{})
		fields := data["fields"].([]interface{})
		if len(fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(fields))
		}

		field := fields[0].(map[string]interface{})
		if field["value"] != "Status OK" {
			t.Errorf("value = %v, want Status OK", field["value"])
		}
		if field["fillColor"] != "#00FF00" {
			t.Errorf("fillColor = %v, want #00FF00", field["fillColor"])
		}
		if field["textColor"] != "#FFFFFF" {
			t.Errorf("textColor = %v, want #FFFFFF", field["textColor"])
		}
		if field["iconShape"] != "round" {
			t.Errorf("iconShape = %v, want round", field["iconShape"])
		}
		if field["iconUrl"] != "https://example.com/icon.png" {
			t.Errorf("iconUrl = %v, want https://example.com/icon.png", field["iconUrl"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard123",
			"data": map[string]interface{}{"title": "Full Field Card"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
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
