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
// Test server helpers
// =============================================================================

// frameServerSpec describes the fake Miro API behavior for one frame test:
// the expected request line, query parameters, an optional request body
// verifier, and the response to send back.
type frameServerSpec struct {
	wantMethod    string
	wantPath      string
	wantQuery     map[string]string
	status        int
	response      map[string]interface{}
	verifyBody    func(t *testing.T, body map[string]interface{})
	failOnRequest bool
}

// newFrameClient starts a fake Miro API described by spec and returns a client
// pointed at it.
func newFrameClient(t *testing.T, spec frameServerSpec) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if spec.failOnRequest {
			t.Error("the API must not be called in this scenario")
			return
		}
		spec.assertRequest(t, r)
		writeFrameJSON(w, spec.status, spec.response)
	}))
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

func (spec frameServerSpec) assertRequest(t *testing.T, r *http.Request) {
	t.Helper()
	if spec.wantMethod != "" && r.Method != spec.wantMethod {
		t.Errorf("expected %s, got %s", spec.wantMethod, r.Method)
	}
	if spec.wantPath != "" && r.URL.Path != spec.wantPath {
		t.Errorf("expected path %s, got %s", spec.wantPath, r.URL.Path)
	}
	for key, want := range spec.wantQuery {
		if got := r.URL.Query().Get(key); got != want {
			t.Errorf("query %s = %q, want %q", key, got, want)
		}
	}
	spec.runBodyVerifier(t, r)
}

func (spec frameServerSpec) runBodyVerifier(t *testing.T, r *http.Request) {
	t.Helper()
	if spec.verifyBody == nil {
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("failed to decode request: %v", err)
		return
	}
	spec.verifyBody(t, body)
}

func writeFrameJSON(w http.ResponseWriter, status int, response map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if response == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		panic(err)
	}
}

// assertNullParent verifies the request body carries an explicit null parent,
// which is how items are removed from a frame.
func assertNullParent(t *testing.T, body map[string]interface{}) {
	t.Helper()
	parent, exists := body["parent"]
	if !exists {
		t.Error("expected 'parent' field in request body")
		return
	}
	if parent != nil {
		t.Errorf("parent = %v, want nil", parent)
	}
}

// =============================================================================
// Remove-from-frame Tests
// =============================================================================

// removeFromFrameCase is one scenario for TestUpdateItems_RemoveFromFrame:
// the fake API response, the update call to make, and the expected result ID
// (empty means the result ID is not asserted).
type removeFromFrameCase struct {
	name     string
	response map[string]interface{}
	call     func(*Client) (string, error)
	wantID   string
}

func removeFromFrameCases(ctx context.Context) []removeFromFrameCase {
	emptyParent := ""
	return []removeFromFrameCase{
		{
			name:     "generic item",
			response: map[string]interface{}{"id": "item456"},
			call: func(c *Client) (string, error) {
				_, err := c.UpdateItem(ctx, UpdateItemArgs{BoardID: "board123", ItemID: "item456", ParentID: &emptyParent})
				return "", err
			},
		},
		{
			name: "sticky",
			response: map[string]interface{}{
				"id":    "sticky123",
				"data":  map[string]interface{}{"content": "test"},
				"style": map[string]interface{}{"fillColor": "yellow"},
			},
			call: func(c *Client) (string, error) {
				result, err := c.UpdateSticky(ctx, UpdateStickyArgs{BoardID: "board123", ItemID: "sticky123", ParentID: &emptyParent})
				return result.ID, err
			},
			wantID: "sticky123",
		},
		{
			name: "shape",
			response: map[string]interface{}{
				"id":   "shape123",
				"data": map[string]interface{}{"content": "test", "shape": "rectangle"},
			},
			call: func(c *Client) (string, error) {
				result, err := c.UpdateShape(ctx, UpdateShapeArgs{BoardID: "board123", ItemID: "shape123", ParentID: &emptyParent})
				return result.ID, err
			},
			wantID: "shape123",
		},
		{
			name: "text",
			response: map[string]interface{}{
				"id":    "text123",
				"data":  map[string]interface{}{"content": "test"},
				"style": map[string]interface{}{"fontSize": "14"},
			},
			call: func(c *Client) (string, error) {
				result, err := c.UpdateText(ctx, UpdateTextArgs{BoardID: "board123", ItemID: "text123", ParentID: &emptyParent})
				return result.ID, err
			},
			wantID: "text123",
		},
		{
			name: "card",
			response: map[string]interface{}{
				"id":   "card123",
				"data": map[string]interface{}{"title": "Test", "description": "", "dueDate": ""},
			},
			call: func(c *Client) (string, error) {
				result, err := c.UpdateCard(ctx, UpdateCardArgs{BoardID: "board123", ItemID: "card123", ParentID: &emptyParent})
				return result.ID, err
			},
			wantID: "card123",
		},
	}
}

// TestUpdateItems_RemoveFromFrame verifies that passing an empty parent_id to
// the item update calls sends an explicit null parent to the API.
func TestUpdateItems_RemoveFromFrame(t *testing.T) {
	for _, tt := range removeFromFrameCases(context.Background()) {
		t.Run(tt.name, func(t *testing.T) {
			client := newFrameClient(t, frameServerSpec{
				response:   tt.response,
				verifyBody: assertNullParent,
			})

			gotID, err := tt.call(client)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantID != "" && gotID != tt.wantID {
				t.Errorf("ID = %v, want %q", gotID, tt.wantID)
			}
		})
	}
}

// =============================================================================
// Frame CRUD Tests
// =============================================================================

func TestCreateFrame_Success(t *testing.T) {
	client := newFrameClient(t, frameServerSpec{
		wantMethod: http.MethodPost,
		wantPath:   "/boards/board123/frames",
		status:     http.StatusCreated,
		response: map[string]interface{}{
			"id":   "frame123",
			"data": map[string]interface{}{"title": "Sprint 1"},
		},
	})

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

func TestCreateFrame_DefaultDimensions(t *testing.T) {
	client := newFrameClient(t, frameServerSpec{
		status: http.StatusCreated,
		verifyBody: func(t *testing.T, body map[string]interface{}) {
			geom := bodySection(t, body, "geometry")
			if geom["width"] != float64(800) {
				t.Errorf("default width = %v, want 800", geom["width"])
			}
			if geom["height"] != float64(600) {
				t.Errorf("default height = %v, want 600", geom["height"])
			}
		},
		response: map[string]interface{}{
			"id":   "frame-defaults",
			"data": map[string]interface{}{"title": "Default Frame"},
		},
	})

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
	client := newFrameClient(t, frameServerSpec{
		status: http.StatusCreated,
		verifyBody: func(t *testing.T, body map[string]interface{}) {
			if style := bodySection(t, body, "style"); style["fillColor"] != "#ffcc00" {
				t.Errorf("fillColor = %v, want '#ffcc00'", style["fillColor"])
			}
		},
		response: map[string]interface{}{
			"id":   "frame-color",
			"data": map[string]interface{}{"title": "Colored Frame"},
		},
	})

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
	client := newFrameClient(t, frameServerSpec{
		status: http.StatusCreated,
		verifyBody: func(t *testing.T, body map[string]interface{}) {
			style := bodySection(t, body, "style")
			fillColor, _ := style["fillColor"].(string)
			if fillColor != "#008000" {
				t.Errorf("fillColor = %q, want '#008000' (green normalized to hex)", fillColor)
			}
		},
		response: map[string]interface{}{
			"id":   "frame-named-color",
			"data": map[string]interface{}{"title": "Green Frame"},
		},
	})

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
	client := newFrameClient(t, frameServerSpec{failOnRequest: true})

	_, err := client.CreateFrame(context.Background(), CreateFrameArgs{
		BoardID: "board123",
		Title:   "Bad Frame",
		Color:   "chartreuse",
	})
	if err == nil {
		t.Fatal("expected error for unknown color name, got nil")
	}
}

func TestGetFrame_Success(t *testing.T) {
	client := newFrameClient(t, frameServerSpec{
		wantMethod: http.MethodGet,
		wantPath:   "/boards/board123/frames/frame456",
		response: map[string]interface{}{
			"id":         "frame456",
			"type":       "frame",
			"data":       map[string]interface{}{"title": "Sprint Planning"},
			"position":   map[string]interface{}{"x": 100.0, "y": 200.0},
			"geometry":   map[string]interface{}{"width": 800.0, "height": 600.0},
			"style":      map[string]interface{}{"fillColor": "#FFFFFF"},
			"children":   []string{"child1", "child2"},
			"createdAt":  "2024-01-01T10:00:00Z",
			"modifiedAt": "2024-01-02T15:30:00Z",
			"createdBy":  map[string]interface{}{"id": "user1"},
			"modifiedBy": map[string]interface{}{"id": "user2"},
		},
	})

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
	if result.ChildCount != 2 {
		t.Errorf("ChildCount = %d, want 2", result.ChildCount)
	}
	assertFrameBounds(t, result)
}

// assertFrameBounds verifies the position and dimensions returned by
// TestGetFrame_Success.
func assertFrameBounds(t *testing.T, result GetFrameResult) {
	t.Helper()
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
}

func TestUpdateFrame_Success(t *testing.T) {
	client := newFrameClient(t, frameServerSpec{
		wantMethod: http.MethodPatch,
		wantPath:   "/boards/board123/frames/frame456",
		verifyBody: func(t *testing.T, body map[string]interface{}) {
			if data, ok := body["data"].(map[string]interface{}); ok && data["title"] != "Updated Title" {
				t.Errorf("title = %v, want 'Updated Title'", data["title"])
			}
		},
		response: map[string]interface{}{
			"id":   "frame456",
			"data": map[string]interface{}{"title": "Updated Title"},
		},
	})

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

func TestUpdateFrame_WithPositionAndGeometry(t *testing.T) {
	client := newFrameClient(t, frameServerSpec{
		wantMethod: http.MethodPatch,
		verifyBody: func(t *testing.T, body map[string]interface{}) {
			if body["position"] == nil {
				t.Error("expected position in body")
			}
			if body["geometry"] == nil {
				t.Error("expected geometry in body")
			}
		},
		response: map[string]interface{}{
			"id":       "frame456",
			"position": map[string]interface{}{"x": 100, "y": 200},
			"geometry": map[string]interface{}{"width": 800, "height": 600},
		},
	})

	x := float64(100)
	y := float64(200)
	width := float64(800)
	height := float64(600)

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
	client := newFrameClient(t, frameServerSpec{
		verifyBody: func(t *testing.T, body map[string]interface{}) {
			if body["style"] == nil {
				t.Error("expected style in body")
			}
		},
		response: map[string]interface{}{
			"id":    "frame456",
			"style": map[string]interface{}{"fillColor": "#FF0000"},
		},
	})

	color := "#FF0000"
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
	client := newFrameClient(t, frameServerSpec{
		wantMethod: http.MethodDelete,
		wantPath:   "/boards/board123/frames/frame456",
		status:     http.StatusNoContent,
	})

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

// =============================================================================
// Frame Items Tests
// =============================================================================

func TestGetFrameItems_Success(t *testing.T) {
	client := newFrameClient(t, frameServerSpec{
		wantMethod: http.MethodGet,
		wantPath:   "/boards/board123/items",
		wantQuery:  map[string]string{"parent_item_id": "frame456"},
		response: map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "item1",
					"type": "sticky_note",
					"data": map[string]interface{}{"content": "Sticky content"},
				},
				{
					"id":   "item2",
					"type": "shape",
					"data": map[string]interface{}{"content": "Shape content"},
				},
			},
			"cursor": "next-cursor",
		},
	})

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

func TestGetFrameItems_WithCursor(t *testing.T) {
	// Tests pagination with cursor
	client := newFrameClient(t, frameServerSpec{
		wantQuery: map[string]string{"cursor": "nextpage"},
		response: map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "item1", "type": "sticky_note"},
			},
		},
	})

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
	client := newFrameClient(t, frameServerSpec{
		wantQuery: map[string]string{"type": "sticky_note"},
		response: map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "sticky1", "type": "sticky_note"},
			},
		},
	})

	_, err := client.GetFrameItems(context.Background(), GetFrameItemsArgs{
		BoardID: "board123",
		FrameID: "frame123",
		Type:    "sticky_note",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Validation Error Tests
// =============================================================================

// TestFrames_ValidationErrors covers every empty-argument rejection across the
// frame operations. No API call is made in any of these scenarios.
func TestFrames_ValidationErrors(t *testing.T) {
	client := newFrameClient(t, frameServerSpec{failOnRequest: true})
	ctx := context.Background()

	tests := []struct {
		name    string
		wantErr string
		call    func() error
	}{
		{"create empty board_id", "board_id is required", func() error { _, err := client.CreateFrame(ctx, CreateFrameArgs{Title: "Test"}); return err }},
		{"get empty board_id", "board_id is required", func() error { _, err := client.GetFrame(ctx, GetFrameArgs{FrameID: "frame123"}); return err }},
		{"get empty frame_id", "frame_id is required", func() error { _, err := client.GetFrame(ctx, GetFrameArgs{BoardID: "board123"}); return err }},
		{"update empty board_id", "board_id is required", func() error {
			_, err := client.UpdateFrame(ctx, UpdateFrameArgs{FrameID: "frame123", Title: ptrString("Test")})
			return err
		}},
		{"update empty frame_id", "frame_id is required", func() error {
			_, err := client.UpdateFrame(ctx, UpdateFrameArgs{BoardID: "board123", Title: ptrString("Test")})
			return err
		}},
		{"update no fields", "at least one update field is required", func() error {
			_, err := client.UpdateFrame(ctx, UpdateFrameArgs{BoardID: "board123", FrameID: "frame456"})
			return err
		}},
		{"delete empty board_id", "board_id is required", func() error { _, err := client.DeleteFrame(ctx, DeleteFrameArgs{FrameID: "frame123"}); return err }},
		{"delete empty frame_id", "frame_id is required", func() error { _, err := client.DeleteFrame(ctx, DeleteFrameArgs{BoardID: "board123"}); return err }},
		{"items empty board_id", "board_id is required", func() error { _, err := client.GetFrameItems(ctx, GetFrameItemsArgs{FrameID: "frame123"}); return err }},
		{"items empty frame_id", "frame_id is required", func() error { _, err := client.GetFrameItems(ctx, GetFrameItemsArgs{BoardID: "board123"}); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}
