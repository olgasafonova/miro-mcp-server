package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateText_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/texts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "text123",
			"data": map[string]interface{}{
				"content": "Hello World",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateText(context.Background(), CreateTextArgs{
		BoardID: "board123",
		Content: "Hello World",
		X:       100,
		Y:       200,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "text123" {
		t.Errorf("ID = %q, want 'text123'", result.ID)
	}
	if result.Content != "Hello World" {
		t.Errorf("Content = %q, want 'Hello World'", result.Content)
	}
}

func TestCreateText_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateTextArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateTextArgs{Content: "test"},
			errText: "board_id is required",
		},
		{
			name:    "empty content",
			args:    CreateTextArgs{BoardID: "board123"},
			errText: "content is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateText(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestCreateText_WithStyle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify style is included
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected style in request body")
		} else {
			if style["fontSize"] != "24" {
				t.Errorf("fontSize = %v, want '24'", style["fontSize"])
			}
			if style["color"] != "#ff0000" {
				t.Errorf("color = %v, want '#ff0000'", style["color"])
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "text-styled",
			"data": map[string]interface{}{
				"content": "Styled text",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateText(context.Background(), CreateTextArgs{
		BoardID:  "board123",
		Content:  "Styled text",
		FontSize: 24,
		Color:    "#ff0000",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "text-styled" {
		t.Errorf("ID = %q, want 'text-styled'", result.ID)
	}
}

func TestCreateText_WithGeometryAndParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify geometry
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else if geom["width"] != float64(300) {
			t.Errorf("width = %v, want 300", geom["width"])
		}
		// Verify parent
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected parent in request body")
		} else if parent["id"] != "frame123" {
			t.Errorf("parent.id = %v, want 'frame123'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "text-geom",
			"data": map[string]interface{}{
				"content": "Text with width",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateText(context.Background(), CreateTextArgs{
		BoardID:  "board123",
		Content:  "Text with width",
		Width:    300,
		ParentID: "frame123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "text-geom" {
		t.Errorf("ID = %q, want 'text-geom'", result.ID)
	}
}

func TestUpdateText_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/texts/") {
			t.Errorf("expected /texts/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "text123",
			"data": map[string]interface{}{
				"content": "Updated text",
			},
			"style": map[string]interface{}{
				"fontSize":  "24",
				"fontColor": "#000000",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	fontSize := 24
	result, err := client.UpdateText(context.Background(), UpdateTextArgs{
		BoardID:  "board123",
		ItemID:   "text123",
		Content:  strPtr("Updated text"),
		FontSize: &fontSize,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "text123" {
		t.Errorf("ID = %q, want 'text123'", result.ID)
	}
	if result.Content != "Updated text" {
		t.Errorf("Content = %q, want 'Updated text'", result.Content)
	}
}

func TestUpdateText_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateText(context.Background(), UpdateTextArgs{
		BoardID: "board123",
		ItemID:  "text123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateText_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		if data, ok := body["data"].(map[string]interface{}); !ok {
			t.Error("expected data in request body")
		} else if data["content"] != "Updated text content" {
			t.Errorf("content = %v, want 'Updated text content'", data["content"])
		}

		// Verify style section
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected style in request body")
		} else {
			if style["fontSize"] != "18" {
				t.Errorf("fontSize = %v, want '18'", style["fontSize"])
			}
			if style["textAlign"] != "center" {
				t.Errorf("textAlign = %v, want 'center'", style["textAlign"])
			}
			if style["color"] != "#333333" {
				t.Errorf("color = %v, want '#333333'", style["color"])
			}
		}

		// Verify position section
		if pos, ok := body["position"].(map[string]interface{}); !ok {
			t.Error("expected position in request body")
		} else {
			if pos["x"] != float64(100) {
				t.Errorf("x = %v, want 100", pos["x"])
			}
			if pos["y"] != float64(200) {
				t.Errorf("y = %v, want 200", pos["y"])
			}
		}

		// Verify geometry section
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else if geom["width"] != float64(400) {
			t.Errorf("width = %v, want 400", geom["width"])
		}

		// Verify parent section
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected parent in request body")
		} else if parent["id"] != "frame-abc" {
			t.Errorf("parent.id = %v, want 'frame-abc'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "text123",
			"data":  map[string]interface{}{"content": "Updated text content"},
			"style": map[string]interface{}{"fontSize": "18"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	content := "Updated text content"
	fontSize := 18
	textAlign := "center"
	color := "#333333"
	x := float64(100)
	y := float64(200)
	width := float64(400)
	parentID := "frame-abc"

	result, err := client.UpdateText(context.Background(), UpdateTextArgs{
		BoardID:   "board123",
		ItemID:    "text123",
		Content:   &content,
		FontSize:  &fontSize,
		TextAlign: &textAlign,
		Color:     &color,
		X:         &x,
		Y:         &y,
		Width:     &width,
		ParentID:  &parentID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "text123" {
		t.Errorf("ID = %v, want 'text123'", result.ID)
	}
}
