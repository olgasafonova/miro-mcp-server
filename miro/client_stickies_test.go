package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestNormalizeStickyColor(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"yellow", "light_yellow"},
		{"Yellow", "light_yellow"},
		{"YELLOW", "light_yellow"},
		{"green", "light_green"},
		{"blue", "light_blue"},
		{"pink", "light_pink"},
		{"purple", "violet"},
		{"gray", "gray"},
		{"grey", "gray"},
		{"unknown_color", "unknown_color"},
		{"#FF0000", "#FF0000"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeStickyColor(tt.input)
			if result != tt.expect {
				t.Errorf("normalizeStickyColor(%q) = %q, want %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCreateStickyArgs(t *testing.T) {
	args := CreateStickyArgs{
		BoardID:  "board123",
		Content:  "Test sticky",
		X:        100,
		Y:        200,
		Color:    "yellow",
		Width:    150,
		ParentID: "frame123",
	}

	if args.BoardID != "board123" {
		t.Errorf("BoardID = %q, want %q", args.BoardID, "board123")
	}
	if args.Content != "Test sticky" {
		t.Errorf("Content = %q, want %q", args.Content, "Test sticky")
	}
}

func TestCreateSticky_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/sticky_notes" {
			t.Errorf("expected /boards/board123/sticky_notes, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "sticky-id",
			"data": map[string]interface{}{
				"content": "Test sticky",
			},
			"style": map[string]interface{}{
				"fillColor": "light_yellow",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateSticky(context.Background(), CreateStickyArgs{
		BoardID: "board123",
		Content: "Test sticky",
		Color:   "yellow",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "sticky-id" {
		t.Errorf("ID = %q, want 'sticky-id'", result.ID)
	}
	if result.Content != "Test sticky" {
		t.Errorf("Content = %q, want 'Test sticky'", result.Content)
	}
}

func TestCreateSticky_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateStickyArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateStickyArgs{Content: "Test"},
			errText: "board_id is required",
		},
		{
			name:    "empty content",
			args:    CreateStickyArgs{BoardID: "board123"},
			errText: "content is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateSticky(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestCreateSticky_WithWidthAndParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify geometry
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else if geom["width"] != float64(250) {
			t.Errorf("width = %v, want 250", geom["width"])
		}
		// Verify parent
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected parent in request body")
		} else if parent["id"] != "frame-abc" {
			t.Errorf("parent.id = %v, want 'frame-abc'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "sticky-geo",
			"data": map[string]interface{}{
				"content": "Wide sticky",
			},
			"style": map[string]interface{}{
				"fillColor": "light_yellow",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateSticky(context.Background(), CreateStickyArgs{
		BoardID:  "board123",
		Content:  "Wide sticky",
		Width:    250,
		ParentID: "frame-abc",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "sticky-geo" {
		t.Errorf("ID = %q, want 'sticky-geo'", result.ID)
	}
}

func TestCreateStickyGrid_Success(t *testing.T) {
	var callCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": fmt.Sprintf("sticky%d", count),
			"data": map[string]interface{}{
				"content": "Test",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateStickyGrid(context.Background(), CreateStickyGridArgs{
		BoardID:  "board123",
		Contents: []string{"Note 1", "Note 2", "Note 3"},
		Columns:  2,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Created != 3 {
		t.Errorf("Created = %d, want 3", result.Created)
	}
	if len(result.ItemIDs) != 3 {
		t.Errorf("ItemIDs length = %d, want 3", len(result.ItemIDs))
	}
}

func TestCreateStickyGrid_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    CreateStickyGridArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    CreateStickyGridArgs{Contents: []string{"Test"}},
			wantErr: "board_id is required",
		},
		{
			name:    "empty contents",
			args:    CreateStickyGridArgs{BoardID: "board123"},
			wantErr: "at least one content item is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateStickyGrid(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestUpdateSticky_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/sticky_notes/") {
			t.Errorf("expected /sticky_notes/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "sticky123",
			"data": map[string]interface{}{
				"content": "Updated content",
				"shape":   "square",
			},
			"style": map[string]interface{}{
				"fillColor": "light_blue",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
		BoardID: "board123",
		ItemID:  "sticky123",
		Content: strPtr("Updated content"),
		Color:   strPtr("blue"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "sticky123" {
		t.Errorf("ID = %q, want 'sticky123'", result.ID)
	}
	if result.Content != "Updated content" {
		t.Errorf("Content = %q, want 'Updated content'", result.Content)
	}
}

func TestUpdateSticky_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
		BoardID: "board123",
		ItemID:  "sticky123",
		// No fields to update
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateSticky_InvalidBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
		BoardID: "",
		ItemID:  "sticky123",
		Content: strPtr("test"),
	})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
}

func TestUpdateSticky_InvalidItemID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
		BoardID: "board123",
		ItemID:  "",
		Content: strPtr("test"),
	})

	if err == nil {
		t.Fatal("expected error for empty item_id")
	}
}

func TestUpdateSticky_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		if data, ok := body["data"].(map[string]interface{}); !ok {
			t.Error("expected data in request body")
		} else {
			if data["content"] != "Updated content" {
				t.Errorf("content = %v, want 'Updated content'", data["content"])
			}
			if data["shape"] != "circle" {
				t.Errorf("shape = %v, want 'circle'", data["shape"])
			}
		}

		// Verify style section
		if style, ok := body["style"].(map[string]interface{}); !ok {
			t.Error("expected style in request body")
		} else if style["fillColor"] != "blue" {
			t.Errorf("fillColor = %v, want 'blue'", style["fillColor"])
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
		} else if geom["width"] != float64(300) {
			t.Errorf("width = %v, want 300", geom["width"])
		}

		// Verify parent section
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected parent in request body")
		} else if parent["id"] != "frame123" {
			t.Errorf("parent.id = %v, want 'frame123'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "sticky123",
			"data":  map[string]interface{}{"content": "Updated content", "shape": "circle"},
			"style": map[string]interface{}{"fillColor": "blue"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	x := float64(100)
	y := float64(200)
	width := float64(300)
	parentID := "frame123"
	content := "Updated content"
	shape := "circle"
	color := "blue"

	result, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
		BoardID:  "board123",
		ItemID:   "sticky123",
		Content:  &content,
		Shape:    &shape,
		Color:    &color,
		X:        &x,
		Y:        &y,
		Width:    &width,
		ParentID: &parentID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "sticky123" {
		t.Errorf("ID = %v, want 'sticky123'", result.ID)
	}
}

func TestCreateSticky_EmptyBoardID(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	_, err := client.CreateSticky(context.Background(), CreateStickyArgs{
		BoardID: "",
		Content: "test",
	})

	if err == nil {
		t.Error("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id") {
		t.Errorf("error should mention board_id: %v", err)
	}
}

func TestCreateSticky_EmptyContent(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	_, err := client.CreateSticky(context.Background(), CreateStickyArgs{
		BoardID: "board123",
		Content: "",
	})

	if err == nil {
		t.Error("expected error for empty content")
	}
	if !strings.Contains(err.Error(), "content") {
		t.Errorf("error should mention content: %v", err)
	}
}
