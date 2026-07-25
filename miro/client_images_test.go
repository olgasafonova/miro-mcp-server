package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCreateImage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/images" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "image123",
			"data": map[string]interface{}{
				"title": "Logo",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateImage(context.Background(), CreateImageArgs{
		BoardID: "board123",
		URL:     "https://example.com/image.png",
		Title:   "Logo",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "image123" {
		t.Errorf("ID = %q, want 'image123'", result.ID)
	}
}

func TestCreateImage_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateImageArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateImageArgs{URL: "https://example.com/img.png"},
			errText: "board_id is required",
		},
		{
			name:    "empty url",
			args:    CreateImageArgs{BoardID: "board123"},
			errText: "url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateImage(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestGetImage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/boards/board123/images/img456") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "img456",
			"type": "image",
			"data": map[string]interface{}{
				"title":    "Logo",
				"imageUrl": "https://miro.com/images/img456.png",
			},
			"position": map[string]interface{}{
				"x": 100.0,
				"y": 200.0,
			},
			"geometry": map[string]interface{}{
				"width":  800.0,
				"height": 600.0,
			},
			"parentId": "frame1",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetImage(context.Background(), GetImageArgs{
		BoardID: "board123",
		ItemID:  "img456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "img456" {
		t.Errorf("ID = %q, want 'img456'", result.ID)
	}
	if result.Title != "Logo" {
		t.Errorf("Title = %q, want 'Logo'", result.Title)
	}
	if result.ImageURL != "https://miro.com/images/img456.png" {
		t.Errorf("ImageURL = %q, want 'https://miro.com/images/img456.png'", result.ImageURL)
	}
	if result.Width != 800.0 {
		t.Errorf("Width = %f, want 800", result.Width)
	}
	if result.Height != 600.0 {
		t.Errorf("Height = %f, want 600", result.Height)
	}
	if result.ParentID != "frame1" {
		t.Errorf("ParentID = %q, want 'frame1'", result.ParentID)
	}
}

func TestGetImage_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    GetImageArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    GetImageArgs{ItemID: "img456"},
			errText: "board_id is required",
		},
		{
			name:    "empty item_id",
			args:    GetImageArgs{BoardID: "board123"},
			errText: "item_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetImage(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestCreateImage_WithAllFields(t *testing.T) {
	// Tests CreateImage with all optional fields
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Error("expected 'data' field")
		}
		if data["url"] != "https://example.com/image.png" {
			t.Errorf("url = %v, want 'https://example.com/image.png'", data["url"])
		}
		if data["title"] != "Test Image" {
			t.Errorf("title = %v, want 'Test Image'", data["title"])
		}

		// Verify position and geometry
		if _, ok := body["position"]; !ok {
			t.Error("expected 'position' field")
		}
		if _, ok := body["geometry"]; !ok {
			t.Error("expected 'geometry' field")
		}
		if _, ok := body["parent"]; !ok {
			t.Error("expected 'parent' field")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "image123",
			"type": "image",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateImage(context.Background(), CreateImageArgs{
		BoardID:  "board123",
		URL:      "https://example.com/image.png",
		Title:    "Test Image",
		X:        100,
		Y:        200,
		Width:    400,
		ParentID: "frame123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateImage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/images/") {
			t.Errorf("expected /images/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "image123",
			"data": map[string]interface{}{
				"title":    "Updated image",
				"imageUrl": "https://example.com/new.png",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateImage(context.Background(), UpdateImageArgs{
		BoardID: "board123",
		ItemID:  "image123",
		Title:   strPtr("Updated image"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "image123" {
		t.Errorf("ID = %q, want 'image123'", result.ID)
	}
	if result.Title != "Updated image" {
		t.Errorf("Title = %q, want 'Updated image'", result.Title)
	}
}

func TestUpdateImage_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateImage(context.Background(), UpdateImageArgs{
		BoardID: "board123",
		ItemID:  "image123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateImage_Validation(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Empty board_id
	_, err := client.UpdateImage(context.Background(), UpdateImageArgs{
		BoardID: "",
		ItemID:  "image123",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}

	// Empty item_id
	_, err = client.UpdateImage(context.Background(), UpdateImageArgs{
		BoardID: "board123",
		ItemID:  "",
	})
	if err == nil {
		t.Error("expected error for empty item_id")
	}
}

func TestUpdateImageFromFile_Success(t *testing.T) {
	t.Setenv("MIRO_UPLOAD_ALLOWED_DIRS", os.TempDir())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/images/img-456") {
			t.Errorf("expected images/img-456 in path, got %s", r.URL.Path)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("expected multipart/form-data, got %s", ct)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "img-456",
			"data": map[string]interface{}{
				"title": "updated-logo.png",
			},
		})
	}))
	defer server.Close()

	tmpFile, err := os.CreateTemp("", "test-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("PNG fake content"))
	tmpFile.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateImageFromFile(context.Background(), UpdateImageFromFileArgs{
		BoardID:  "board123",
		ItemID:   "img-456",
		FilePath: tmpFile.Name(),
		Title:    "updated-logo.png",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "img-456" {
		t.Errorf("ID = %q, want 'img-456'", result.ID)
	}
	if !strings.Contains(result.Message, "Updated image") {
		t.Errorf("Message = %q, want containing 'Updated image'", result.Message)
	}
}

func TestUpdateImageFromFile_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    UpdateImageFromFileArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    UpdateImageFromFileArgs{ItemID: "img-1", FilePath: "/tmp/test.png"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty item ID",
			args:    UpdateImageFromFileArgs{BoardID: "board123", FilePath: "/tmp/test.png"},
			wantErr: "item_id is required",
		},
		{
			name:    "empty file path",
			args:    UpdateImageFromFileArgs{BoardID: "board123", ItemID: "img-1"},
			wantErr: "file_path is required",
		},
		{
			name:    "nonexistent file",
			args:    UpdateImageFromFileArgs{BoardID: "board123", ItemID: "img-1", FilePath: "/nonexistent/file.png"},
			wantErr: "cannot access file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UpdateImageFromFile(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestUpdateImageFromFile_UnsupportedFormat(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.pdf")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("not an image"))
	tmpFile.Close()

	client := newTestClientWithServer("http://localhost")
	_, err = client.UpdateImageFromFile(context.Background(), UpdateImageFromFileArgs{
		BoardID:  "board123",
		ItemID:   "img-1",
		FilePath: tmpFile.Name(),
	})
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported image format") {
		t.Errorf("error = %q, want containing 'unsupported image format'", err.Error())
	}
}
