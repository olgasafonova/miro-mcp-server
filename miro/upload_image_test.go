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

// =============================================================================
// UploadImage Tests
//
// UploadDocument, UpdateImageFromFile, and UpdateDocumentFromFile are covered
// in client_test.go; UploadImage was the one member of the family with no test.
// =============================================================================

// writeTempImage creates a throwaway PNG-ish file and returns its path.
// The upload path guard rejects anything outside the allowed dirs, so this
// also opens TempDir for the duration of the test (same as the existing
// UploadDocument tests in client_test.go).
func writeTempImage(t *testing.T) string {
	t.Helper()
	t.Setenv("MIRO_UPLOAD_ALLOWED_DIRS", os.TempDir())

	tmpFile, err := os.CreateTemp("", "test-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	if _, err := tmpFile.Write([]byte("\x89PNG\r\n\x1a\n test content")); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}
	return tmpFile.Name()
}

func TestUploadImage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/images") {
			t.Errorf("expected an /images path, got %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("Content-Type = %s, want multipart/form-data", ct)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "image-upload-123",
			"data": map[string]interface{}{"title": "diagram.png"},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UploadImage(context.Background(), UploadImageArgs{
		BoardID:  "board123",
		FilePath: writeTempImage(t),
		Title:    "diagram.png",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "image-upload-123" {
		t.Errorf("ID = %q, want image-upload-123", result.ID)
	}
	if result.ItemURL != BuildItemURL("board123", "image-upload-123") {
		t.Errorf("ItemURL = %q", result.ItemURL)
	}
}

func TestUploadImage_WithPositionAndParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "image-upload-123"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UploadImage(context.Background(), UploadImageArgs{
		BoardID:  "board123",
		FilePath: writeTempImage(t),
		Title:    "diagram.png",
		X:        150,
		Y:        -75,
		ParentID: "frame999",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "image-upload-123" {
		t.Errorf("ID = %q, want image-upload-123", result.ID)
	}
}

func TestUploadImage_ValidationErrors(t *testing.T) {
	imagePath := writeTempImage(t)

	tests := []struct {
		name    string
		args    UploadImageArgs
		wantErr string
	}{
		{
			name:    "empty board",
			args:    UploadImageArgs{BoardID: "", FilePath: imagePath},
			wantErr: "board_id is required",
		},
		{
			name:    "empty file path",
			args:    UploadImageArgs{BoardID: "board123", FilePath: ""},
			wantErr: "file_path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClientWithServer("http://unused.invalid")
			_, err := client.UploadImage(context.Background(), tt.args)
			if err == nil {
				t.Fatalf("expected an error for %s", tt.name)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestUploadImage_NonexistentFile(t *testing.T) {
	client := newTestClientWithServer("http://unused.invalid")
	_, err := client.UploadImage(context.Background(), UploadImageArgs{
		BoardID:  "board123",
		FilePath: "/nonexistent/path/to/image.png",
	})
	if err == nil {
		t.Fatal("expected an error for a nonexistent file")
	}
}

func TestUploadImage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.UploadImage(context.Background(), UploadImageArgs{
		BoardID:  "board123",
		FilePath: writeTempImage(t),
	}); err == nil {
		t.Fatal("expected an error on HTTP 400")
	}
}
