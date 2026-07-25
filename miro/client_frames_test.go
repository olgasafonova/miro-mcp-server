package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateItem_RemoveFromFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify parent is null
		if parent, exists := body["parent"]; !exists {
			t.Error("expected 'parent' field in request body")
		} else if parent != nil {
			t.Errorf("parent = %v, want nil", parent)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "item456"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	emptyParent := ""
	_, err := client.UpdateItem(context.Background(), UpdateItemArgs{
		BoardID:  "board123",
		ItemID:   "item456",
		ParentID: &emptyParent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateFrame_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/frames" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "frame123",
			"data": map[string]interface{}{
				"title": "Sprint 1",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateFrame(context.Background(), CreateFrameArgs{
		BoardID: "board123",
		Title:   "Sprint 1",
		X:       0,
		Y:       0,
		Width:   800,
		Height:  600,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "frame123" {
		t.Errorf("ID = %q, want 'frame123'", result.ID)
	}
	if result.Title != "Sprint 1" {
		t.Errorf("Title = %q, want 'Sprint 1'", result.Title)
	}
}

func TestCreateFrame_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Only board_id is required - title is optional
	_, err := client.CreateFrame(context.Background(), CreateFrameArgs{Title: "Test"})
	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestCreateFrame_DefaultDimensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify default dimensions
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else {
			if geom["width"] != float64(800) {
				t.Errorf("default width = %v, want 800", geom["width"])
			}
			if geom["height"] != float64(600) {
				t.Errorf("default height = %v, want 600", geom["height"])
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "frame-defaults",
			"data": map[string]interface{}{"title": "Default Frame"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	// Width and Height are 0, should get defaults
	result, err := client.CreateFrame(context.Background(), CreateFrameArgs{
		BoardID: "board123",
		Title:   "Default Frame",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "frame-defaults" {
		t.Errorf("ID = %q, want 'frame-defaults'", result.ID)
	}
}

func TestCreateFrame_WithColor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify style with fillColor
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected style in request body")
		} else if style["fillColor"] != "#ffcc00" {
			t.Errorf("fillColor = %v, want '#ffcc00'", style["fillColor"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "frame-color",
			"data": map[string]interface{}{"title": "Colored Frame"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateFrame(context.Background(), CreateFrameArgs{
		BoardID: "board123",
		Title:   "Colored Frame",
		Color:   "#ffcc00",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "frame-color" {
		t.Errorf("ID = %q, want 'frame-color'", result.ID)
	}
}

// TestCreateFrame_WithColorName covers the regression surfaced by trace-mine on
// 26-04-2026: agents passed semantic color names ("green", "pink", "blue") to
// the Color field, which Miro's API rejected because frames require a 6-char
// hex string. CreateFrame now normalizes names to hex before sending.
func TestCreateFrame_WithColorName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		style, ok := body["style"].(map[string]interface{})
		if !ok {
			t.Fatal("expected style in request body")
		}
		fillColor, _ := style["fillColor"].(string)
		if fillColor != "#008000" {
			t.Errorf("fillColor = %q, want '#008000' (green normalized to hex)", fillColor)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "frame-named-color",
			"data": map[string]interface{}{"title": "Green Frame"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.CreateFrame(context.Background(), CreateFrameArgs{
		BoardID: "board123",
		Title:   "Green Frame",
		Color:   "green",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCreateFrame_RejectsUnknownColor verifies normalizeColor errors for garbage
// surface back through the create call as a structured Go error rather than a
// silent passthrough that fails at Miro's API layer.
func TestCreateFrame_RejectsUnknownColor(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateFrame(context.Background(), CreateFrameArgs{
		BoardID: "board123",
		Title:   "Bad Frame",
		Color:   "chartreuse",
	})
	if err == nil {
		t.Fatal("expected error for unknown color name, got nil")
	}
	if called {
		t.Error("HTTP server should not be called when normalizeColor rejects input pre-flight")
	}
}

func TestGetFrame_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/frames/frame456" {
			t.Errorf("expected /boards/board123/frames/frame456, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "frame456",
			"type": "frame",
			"data": map[string]interface{}{
				"title": "Sprint Planning",
			},
			"position": map[string]interface{}{
				"x": 100.0,
				"y": 200.0,
			},
			"geometry": map[string]interface{}{
				"width":  800.0,
				"height": 600.0,
			},
			"style": map[string]interface{}{
				"fillColor": "#FFFFFF",
			},
			"children":   []string{"child1", "child2"},
			"createdAt":  "2024-01-01T10:00:00Z",
			"modifiedAt": "2024-01-02T15:30:00Z",
			"createdBy":  map[string]interface{}{"id": "user1"},
			"modifiedBy": map[string]interface{}{"id": "user2"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetFrame(context.Background(), GetFrameArgs{
		BoardID: "board123",
		FrameID: "frame456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "frame456" {
		t.Errorf("ID = %q, want 'frame456'", result.ID)
	}
	if result.Title != "Sprint Planning" {
		t.Errorf("Title = %q, want 'Sprint Planning'", result.Title)
	}
	if result.X != 100 {
		t.Errorf("X = %f, want 100", result.X)
	}
	if result.Y != 200 {
		t.Errorf("Y = %f, want 200", result.Y)
	}
	if result.Width != 800 {
		t.Errorf("Width = %f, want 800", result.Width)
	}
	if result.Height != 600 {
		t.Errorf("Height = %f, want 600", result.Height)
	}
	if result.ChildCount != 2 {
		t.Errorf("ChildCount = %d, want 2", result.ChildCount)
	}
}

func TestGetFrame_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    GetFrameArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    GetFrameArgs{FrameID: "frame123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty frame ID",
			args:    GetFrameArgs{BoardID: "board123"},
			wantErr: "frame_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetFrame(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestUpdateFrame_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/frames/frame456" {
			t.Errorf("expected /boards/board123/frames/frame456, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if data, ok := body["data"].(map[string]interface{}); ok {
			if data["title"] != "Updated Title" {
				t.Errorf("title = %v, want 'Updated Title'", data["title"])
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "frame456",
			"data": map[string]interface{}{
				"title": "Updated Title",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	title := "Updated Title"
	result, err := client.UpdateFrame(context.Background(), UpdateFrameArgs{
		BoardID: "board123",
		FrameID: "frame456",
		Title:   &title,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.ID != "frame456" {
		t.Errorf("ID = %q, want 'frame456'", result.ID)
	}
}

func TestUpdateFrame_NoUpdates(t *testing.T) {
	client := newTestClientWithServer("http://localhost")
	_, err := client.UpdateFrame(context.Background(), UpdateFrameArgs{
		BoardID: "board123",
		FrameID: "frame456",
		// No update fields provided
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "at least one update field is required") {
		t.Errorf("expected 'at least one update field is required', got: %v", err)
	}
}

func TestUpdateFrame_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    UpdateFrameArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    UpdateFrameArgs{FrameID: "frame123", Title: ptrString("Test")},
			errText: "board_id is required",
		},
		{
			name:    "empty frame_id",
			args:    UpdateFrameArgs{BoardID: "board123", Title: ptrString("Test")},
			errText: "frame_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UpdateFrame(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestUpdateFrame_WithPositionAndGeometry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		// Verify request body contains position and geometry
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["position"] == nil {
			t.Error("expected position in body")
		}
		if body["geometry"] == nil {
			t.Error("expected geometry in body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "frame456",
			"position": map[string]interface{}{
				"x": 100,
				"y": 200,
			},
			"geometry": map[string]interface{}{
				"width":  800,
				"height": 600,
			},
		})
	}))
	defer server.Close()

	x := float64(100)
	y := float64(200)
	width := float64(800)
	height := float64(600)

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateFrame(context.Background(), UpdateFrameArgs{
		BoardID: "board123",
		FrameID: "frame456",
		X:       &x,
		Y:       &y,
		Width:   &width,
		Height:  &height,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestUpdateFrame_WithColor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["style"] == nil {
			t.Error("expected style in body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "frame456",
			"style": map[string]interface{}{
				"fillColor": "#FF0000",
			},
		})
	}))
	defer server.Close()

	color := "#FF0000"
	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateFrame(context.Background(), UpdateFrameArgs{
		BoardID: "board123",
		FrameID: "frame456",
		Color:   &color,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestDeleteFrame_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/frames/frame456" {
			t.Errorf("expected /boards/board123/frames/frame456, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteFrame(context.Background(), DeleteFrameArgs{
		BoardID: "board123",
		FrameID: "frame456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.ID != "frame456" {
		t.Errorf("ID = %q, want 'frame456'", result.ID)
	}
}

func TestDeleteFrame_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    DeleteFrameArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    DeleteFrameArgs{FrameID: "frame123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty frame ID",
			args:    DeleteFrameArgs{BoardID: "board123"},
			wantErr: "frame_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.DeleteFrame(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestGetFrameItems_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/items" {
			t.Errorf("expected /boards/board123/items, got %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "parent_item_id=frame456") {
			t.Errorf("expected parent_item_id=frame456 in query, got %s", r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "item1",
					"type": "sticky_note",
					"data": map[string]interface{}{
						"content": "Sticky content",
					},
				},
				{
					"id":   "item2",
					"type": "shape",
					"data": map[string]interface{}{
						"content": "Shape content",
					},
				},
			},
			"cursor": "next-cursor",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetFrameItems(context.Background(), GetFrameItemsArgs{
		BoardID: "board123",
		FrameID: "frame456",
		Limit:   50,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if !result.HasMore {
		t.Error("HasMore should be true when cursor is present")
	}
	if result.Items[0].ID != "item1" {
		t.Errorf("first item ID = %q, want 'item1'", result.Items[0].ID)
	}
}

func TestGetFrameItems_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    GetFrameItemsArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    GetFrameItemsArgs{FrameID: "frame123"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty frame ID",
			args:    GetFrameItemsArgs{BoardID: "board123"},
			wantErr: "frame_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetFrameItems(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestUpdateSticky_RemoveFromFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify parent is null when empty string provided
		if parent, exists := body["parent"]; !exists {
			t.Error("expected parent in request body")
		} else if parent != nil {
			t.Errorf("parent = %v, want nil", parent)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "sticky123",
			"data":  map[string]interface{}{"content": "test"},
			"style": map[string]interface{}{"fillColor": "yellow"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	emptyParent := ""

	result, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
		BoardID:  "board123",
		ItemID:   "sticky123",
		ParentID: &emptyParent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "sticky123" {
		t.Errorf("ID = %v, want 'sticky123'", result.ID)
	}
}

func TestUpdateShape_RemoveFromFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify parent is null
		if parent, exists := body["parent"]; !exists {
			t.Error("expected parent in request body")
		} else if parent != nil {
			t.Errorf("parent = %v, want nil", parent)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "shape123",
			"data": map[string]interface{}{"content": "test", "shape": "rectangle"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	emptyParent := ""

	result, err := client.UpdateShape(context.Background(), UpdateShapeArgs{
		BoardID:  "board123",
		ItemID:   "shape123",
		ParentID: &emptyParent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "shape123" {
		t.Errorf("ID = %v, want 'shape123'", result.ID)
	}
}

func TestUpdateText_RemoveFromFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify parent is null
		if parent, exists := body["parent"]; !exists {
			t.Error("expected parent in request body")
		} else if parent != nil {
			t.Errorf("parent = %v, want nil", parent)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "text123",
			"data":  map[string]interface{}{"content": "test"},
			"style": map[string]interface{}{"fontSize": "14"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	emptyParent := ""

	result, err := client.UpdateText(context.Background(), UpdateTextArgs{
		BoardID:  "board123",
		ItemID:   "text123",
		ParentID: &emptyParent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "text123" {
		t.Errorf("ID = %v, want 'text123'", result.ID)
	}
}

func TestUpdateCard_RemoveFromFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify parent is null
		if parent, exists := body["parent"]; !exists {
			t.Error("expected parent in request body")
		} else if parent != nil {
			t.Errorf("parent = %v, want nil", parent)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "card123",
			"data": map[string]interface{}{"title": "Test", "description": "", "dueDate": ""},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	emptyParent := ""

	result, err := client.UpdateCard(context.Background(), UpdateCardArgs{
		BoardID:  "board123",
		ItemID:   "card123",
		ParentID: &emptyParent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "card123" {
		t.Errorf("ID = %v, want 'card123'", result.ID)
	}
}

func TestGetFrameItems_WithCursor(t *testing.T) {
	// Tests pagination with cursor
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "cursor=nextpage") {
			t.Error("expected cursor parameter in request")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "item1", "type": "sticky_note"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetFrameItems(context.Background(), GetFrameItemsArgs{
		BoardID: "board123",
		FrameID: "frame123",
		Cursor:  "nextpage",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetFrameItems_WithTypeFilter(t *testing.T) {
	// Tests type filtering
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "type=sticky_note") {
			t.Error("expected type parameter in request")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "sticky1", "type": "sticky_note"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetFrameItems(context.Background(), GetFrameItemsArgs{
		BoardID: "board123",
		FrameID: "frame123",
		Type:    "sticky_note",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
