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
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/shapes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		data := body["data"].(map[string]interface{})
		if data["shape"] != "rectangle" {
			t.Errorf("shape = %v, want 'rectangle'", data["shape"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
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

func TestCreateShape_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateShapeArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateShapeArgs{Shape: "rectangle"},
			errText: "board_id is required",
		},
		{
			name:    "empty shape",
			args:    CreateShapeArgs{BoardID: "board123"},
			errText: "shape type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateShape(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestUpdateShape_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/shapes/") {
			t.Errorf("expected /shapes/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
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
	}))
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
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		if data, ok := body["data"].(map[string]interface{}); !ok {
			t.Error("expected data in request body")
		} else {
			if data["content"] != "New shape content" {
				t.Errorf("content = %v, want 'New shape content'", data["content"])
			}
			if data["shape"] != "circle" {
				t.Errorf("shape = %v, want 'circle'", data["shape"])
			}
		}

		// Verify style section
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected style in request body")
		} else {
			if style["fillColor"] != "#FF0000" {
				t.Errorf("fillColor = %v, want '#FF0000'", style["fillColor"])
			}
			if style["fontColor"] != "#FFFFFF" {
				t.Errorf("fontColor = %v, want '#FFFFFF'", style["fontColor"])
			}
		}

		// Verify position section
		if pos, ok := body["position"].(map[string]interface{}); !ok {
			t.Error("expected position in request body")
		} else {
			if pos["x"] != float64(50) {
				t.Errorf("x = %v, want 50", pos["x"])
			}
			if pos["y"] != float64(75) {
				t.Errorf("y = %v, want 75", pos["y"])
			}
		}

		// Verify geometry section
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else {
			if geom["width"] != float64(200) {
				t.Errorf("width = %v, want 200", geom["width"])
			}
			if geom["height"] != float64(150) {
				t.Errorf("height = %v, want 150", geom["height"])
			}
		}

		// Verify parent section
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected parent in request body")
		} else if parent["id"] != "frame-xyz" {
			t.Errorf("parent.id = %v, want 'frame-xyz'", parent["id"])
		}

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

func TestCreateShape_WithTextAlign(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		style, ok := body["style"].(map[string]interface{})
		if !ok {
			t.Fatal("expected style in request body")
		}
		if style["textAlign"] != "center" {
			t.Errorf("style.textAlign = %v, want 'center'", style["textAlign"])
		}
		if style["textAlignVertical"] != "middle" {
			t.Errorf("style.textAlignVertical = %v, want 'middle'", style["textAlignVertical"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "shape123",
			"data": map[string]interface{}{"shape": "triangle", "content": "Centered"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateShape(context.Background(), CreateShapeArgs{
		BoardID:           "board123",
		Shape:             "triangle",
		Content:           "Centered",
		TextAlign:         "center",
		TextAlignVertical: "middle",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateShape_InvalidTextAlign(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateShapeArgs
		errText string
	}{
		{
			name: "bad horizontal value",
			args: CreateShapeArgs{
				BoardID:   "board123",
				Shape:     "rectangle",
				TextAlign: "centre",
			},
			errText: "text_align",
		},
		{
			name: "bad vertical value",
			args: CreateShapeArgs{
				BoardID:           "board123",
				Shape:             "rectangle",
				TextAlignVertical: "centre",
			},
			errText: "text_align_vertical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateShape(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestUpdateShape_WithTextAlign(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		style, ok := body["style"].(map[string]interface{})
		if !ok {
			t.Fatal("expected style in request body")
		}
		if style["textAlign"] != "left" {
			t.Errorf("style.textAlign = %v, want 'left'", style["textAlign"])
		}
		if style["textAlignVertical"] != "top" {
			t.Errorf("style.textAlignVertical = %v, want 'top'", style["textAlignVertical"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "shape123",
			"data": map[string]interface{}{"shape": "rectangle", "content": "Left-top"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	textAlign := "left"
	textAlignVertical := "top"
	_, err := client.UpdateShape(context.Background(), UpdateShapeArgs{
		BoardID:           "board123",
		ItemID:            "shape123",
		TextAlign:         &textAlign,
		TextAlignVertical: &textAlignVertical,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
		style, ok := body["style"].(map[string]interface{})
		if !ok {
			t.Fatal("expected style in PATCH body")
		}
		if style["textAlign"] != "center" {
			t.Errorf("style.textAlign = %v, want 'center'", style["textAlign"])
		}
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

func TestBulkCreate_ShapeWithTextStyleFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		style, ok := body["style"].(map[string]interface{})
		if !ok {
			t.Fatal("expected style in request body for shape item")
		}
		if style["color"] != "#ffffff" {
			t.Errorf("style.color (text color) = %v, want '#ffffff'", style["color"])
		}
		if style["textAlign"] != "right" {
			t.Errorf("style.textAlign = %v, want 'right'", style["textAlign"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "shape999",
			"data": map[string]interface{}{"shape": "circle", "content": "Bulk styled"},
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
}
