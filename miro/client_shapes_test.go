package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateShape_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Method + " " + r.URL.Path; got != "POST /boards/board123/shapes" {
			t.Errorf("request = %q, want POST /boards/board123/shapes", got)
		}

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		requireBodyField(t, body, "data.shape", "rectangle")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "shape123",
			"data": map[string]interface{}{
				"shape":   "rectangle",
				"content": "Test shape",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateShape(context.Background(), CreateShapeArgs{
		BoardID: "board123",
		Shape:   "rectangle",
		Content: "Test shape",
		X:       100,
		Y:       200,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "shape123" {
		t.Errorf("ID = %q, want 'shape123'", result.ID)
	}
	if result.Shape != "rectangle" {
		t.Errorf("Shape = %q, want 'rectangle'", result.Shape)
	}
}

// TestCreateShape_ValidationErrors covers argument-validation failures for
// shape creation, including text alignment values.
func TestCreateShape_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	createShape := func(args CreateShapeArgs) func() error {
		return func() error {
			_, err := client.CreateShape(context.Background(), args)
			return err
		}
	}

	runStickyValidationCases(t, []stickyValidationCase{
		{
			name:    "empty board_id",
			wantErr: "board_id is required",
			call:    createShape(CreateShapeArgs{Shape: "rectangle"}),
		},
		{
			name:    "empty shape",
			wantErr: "shape type is required",
			call:    createShape(CreateShapeArgs{BoardID: "board123"}),
		},
		{
			name:    "bad horizontal text align value",
			wantErr: "text_align",
			call:    createShape(CreateShapeArgs{BoardID: "board123", Shape: "rectangle", TextAlign: "centre"}),
		},
		{
			name:    "bad vertical text align value",
			wantErr: "text_align_vertical",
			call:    createShape(CreateShapeArgs{BoardID: "board123", Shape: "rectangle", TextAlignVertical: "centre"}),
		},
	})
}

func TestUpdateShape_Success(t *testing.T) {
	server := newStickyVerifyServer(t, "PATCH /boards/board123/shapes/shape123", http.StatusOK, map[string]interface{}{
		"id": "shape123",
		"data": map[string]interface{}{
			"content": "Updated shape",
			"shape":   "circle",
		},
		"style": map[string]interface{}{
			"fillColor": "#FF0000",
			"fontColor": "#FFFFFF",
		},
	})
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateShape(context.Background(), UpdateShapeArgs{
		BoardID:   "board123",
		ItemID:    "shape123",
		Content:   strPtr("Updated shape"),
		ShapeType: strPtr("circle"),
		Color:     strPtr("#FF0000"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "shape123" {
		t.Errorf("ID = %q, want 'shape123'", result.ID)
	}
	if result.ShapeType != "circle" {
		t.Errorf("ShapeType = %q, want 'circle'", result.ShapeType)
	}
}

func TestUpdateShape_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateShape(context.Background(), UpdateShapeArgs{
		BoardID: "board123",
		ItemID:  "shape123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateShape_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)

		requireBodyField(t, body, "data.content", "New shape content")
		requireBodyField(t, body, "data.shape", "circle")
		requireBodyField(t, body, "style.fillColor", "#FF0000")
		requireBodyField(t, body, "style.fontColor", "#FFFFFF")
		requireBodyField(t, body, "position.x", float64(50))
		requireBodyField(t, body, "position.y", float64(75))
		requireBodyField(t, body, "geometry.width", float64(200))
		requireBodyField(t, body, "geometry.height", float64(150))
		requireBodyField(t, body, "parent.id", "frame-xyz")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "shape123",
			"data": map[string]interface{}{"content": "New shape content", "shape": "circle"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	content := "New shape content"
	shapeType := "circle"
	color := "#FF0000"
	textColor := "#FFFFFF"
	x := float64(50)
	y := float64(75)
	width := float64(200)
	height := float64(150)
	parentID := "frame-xyz"

	result, err := client.UpdateShape(context.Background(), UpdateShapeArgs{
		BoardID:   "board123",
		ItemID:    "shape123",
		Content:   &content,
		ShapeType: &shapeType,
		Color:     &color,
		TextColor: &textColor,
		X:         &x,
		Y:         &y,
		Width:     &width,
		Height:    &height,
		ParentID:  &parentID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "shape123" {
		t.Errorf("ID = %v, want 'shape123'", result.ID)
	}
}

// TestShape_TextAlignSentToAPI pins that both the create and update paths
// forward text alignment into the style block of the request body.
func TestShape_TextAlignSentToAPI(t *testing.T) {
	tests := []struct {
		name         string
		wantMethod   string
		wantAlign    string
		wantVertical string
		call         func(*Client) error
	}{
		{
			name:         "create shape",
			wantMethod:   http.MethodPost,
			wantAlign:    "center",
			wantVertical: "middle",
			call: func(client *Client) error {
				_, err := client.CreateShape(context.Background(), CreateShapeArgs{
					BoardID:           "board123",
					Shape:             "triangle",
					Content:           "Centered",
					TextAlign:         "center",
					TextAlignVertical: "middle",
				})
				return err
			},
		},
		{
			name:         "update shape",
			wantMethod:   http.MethodPatch,
			wantAlign:    "left",
			wantVertical: "top",
			call: func(client *Client) error {
				textAlign := "left"
				textAlignVertical := "top"
				_, err := client.UpdateShape(context.Background(), UpdateShapeArgs{
					BoardID:           "board123",
					ItemID:            "shape123",
					TextAlign:         &textAlign,
					TextAlignVertical: &textAlignVertical,
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.wantMethod {
					t.Errorf("expected %s, got %s", tt.wantMethod, r.Method)
				}
				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				requireBodyField(t, body, "style.textAlign", tt.wantAlign)
				requireBodyField(t, body, "style.textAlignVertical", tt.wantVertical)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":   "shape123",
					"data": map[string]interface{}{"shape": "triangle", "content": "Aligned"},
				})
			}))
			defer server.Close()

			if err := tt.call(newTestClientWithServer(server.URL)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestBulkUpdate_TypeShapeRoutesToShapesEndpoint(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		// Shape-specific fields must reach the shapes endpoint
		requireBodyField(t, body, "style.textAlign", "center")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "shape1",
			"data": map[string]interface{}{"shape": "triangle", "content": "X"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	textAlign := "center"
	result, err := client.BulkUpdate(context.Background(), BulkUpdateArgs{
		BoardID: "board123",
		Items: []BulkUpdateItem{{
			ItemID:    "shape1",
			Type:      "shape",
			TextAlign: &textAlign,
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Updated != 1 {
		t.Errorf("Updated = %d, want 1", result.Updated)
	}
	if len(paths) != 1 || !strings.Contains(paths[0], "/shapes/shape1") {
		t.Errorf("expected PATCH to /shapes/shape1, got: %v", paths)
	}
}

// TestBulkCreate_ShapeWithTextStyleFields pins that a shape created through the
// bulk path carries the same style block a singly-created shape would.
//
// The body is now an ARRAY sent to /items/bulk rather than one object per
// /shapes call, because BulkCreate uses Miro's native bulk endpoint. The
// assertion is unchanged in substance and is the thing keeping bulk and single
// creation from drifting: both go through buildShapeBaseBody.
func TestBulkCreate_ShapeWithTextStyleFields(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path

		var body []map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(body) != 1 {
			t.Fatalf("sent %d items, want 1", len(body))
		}
		item := body[0]

		// The native endpoint discriminates on a top-level type; the URL no
		// longer carries it.
		if item["type"] != "shape" {
			t.Errorf("type = %v, want shape", item["type"])
		}

		requireBodyField(t, item, "style.color", "#ffffff") // text color
		requireBodyField(t, item, "style.textAlign", "right")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "bulk-list",
			"data": []map[string]interface{}{{
				"id":   "shape999",
				"data": map[string]interface{}{"shape": "circle", "content": "Bulk styled"},
			}},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.BulkCreate(context.Background(), BulkCreateArgs{
		BoardID: "board123",
		Items: []BulkCreateItem{{
			Type:      "shape",
			Shape:     "circle",
			Content:   "Bulk styled",
			Color:     "#000000",
			TextColor: "#ffffff",
			TextAlign: "right",
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Created != 1 {
		t.Errorf("Created = %d, want 1", result.Created)
	}
	if !strings.HasSuffix(gotPath, "/items/bulk") {
		t.Errorf("posted to %q, want the native /items/bulk endpoint", gotPath)
	}
}
