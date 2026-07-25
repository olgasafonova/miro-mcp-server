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

func TestGetDocument_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/boards/board123/documents/doc789") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "doc789",
			"type": "document",
			"data": map[string]interface{}{
				"title":       "Q4 Report",
				"documentUrl": "https://miro.com/documents/doc789.pdf",
			},
			"position": map[string]interface{}{
				"x": 300.0,
				"y": 400.0,
			},
			"geometry": map[string]interface{}{
				"width":  400.0,
				"height": 300.0,
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetDocument(context.Background(), GetDocumentArgs{
		BoardID: "board123",
		ItemID:  "doc789",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "doc789" {
		t.Errorf("ID = %q, want 'doc789'", result.ID)
	}
	if result.Title != "Q4 Report" {
		t.Errorf("Title = %q, want 'Q4 Report'", result.Title)
	}
	if result.DocumentURL != "https://miro.com/documents/doc789.pdf" {
		t.Errorf("DocumentURL = %q, want 'https://miro.com/documents/doc789.pdf'", result.DocumentURL)
	}
	if result.Width != 400.0 {
		t.Errorf("Width = %f, want 400", result.Width)
	}
}

func TestGetDocument_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    GetDocumentArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    GetDocumentArgs{ItemID: "doc789"},
			errText: "board_id is required",
		},
		{
			name:    "empty item_id",
			args:    GetDocumentArgs{BoardID: "board123"},
			errText: "item_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetDocument(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestCreateDocument_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/documents") {
			t.Errorf("expected documents path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "doc123",
			"data": map[string]interface{}{
				"title": "Test Document",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateDocument(context.Background(), CreateDocumentArgs{
		BoardID: "board123",
		Title:   "Test Document",
		URL:     "https://example.com/doc.pdf",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "doc123" {
		t.Errorf("ID = %q, want 'doc123'", result.ID)
	}
}

func TestCreateDocument_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    CreateDocumentArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    CreateDocumentArgs{URL: "https://example.com/doc.pdf"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty URL",
			args:    CreateDocumentArgs{BoardID: "board123"},
			wantErr: "url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateDocument(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCreateDocument_WithAllFields(t *testing.T) {
	// Tests CreateDocument with all optional fields
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Error("expected 'data' field")
		}
		if data["url"] != "https://example.com/doc.pdf" {
			t.Errorf("url = %v, want 'https://example.com/doc.pdf'", data["url"])
		}
		if data["title"] != "Test Document" {
			t.Errorf("title = %v, want 'Test Document'", data["title"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "doc123",
			"type": "document",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateDocument(context.Background(), CreateDocumentArgs{
		BoardID:  "board123",
		URL:      "https://example.com/doc.pdf",
		Title:    "Test Document",
		X:        100,
		Y:        200,
		Width:    500,
		ParentID: "frame123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateDocument_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/documents/") {
			t.Errorf("expected /documents/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "doc123",
			"data": map[string]interface{}{
				"title": "Updated document",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateDocument(context.Background(), UpdateDocumentArgs{
		BoardID: "board123",
		ItemID:  "doc123",
		Title:   strPtr("Updated document"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "doc123" {
		t.Errorf("ID = %q, want 'doc123'", result.ID)
	}
	if result.Title != "Updated document" {
		t.Errorf("Title = %q, want 'Updated document'", result.Title)
	}
}

func TestUpdateDocument_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateDocument(context.Background(), UpdateDocumentArgs{
		BoardID: "board123",
		ItemID:  "doc123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateDocument_Validation(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Empty board_id
	_, err := client.UpdateDocument(context.Background(), UpdateDocumentArgs{
		BoardID: "",
		ItemID:  "doc123",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}

	// Empty item_id
	_, err = client.UpdateDocument(context.Background(), UpdateDocumentArgs{
		BoardID: "board123",
		ItemID:  "",
	})
	if err == nil {
		t.Error("expected error for empty item_id")
	}
}

func TestUploadDocument_Success(t *testing.T) {
	t.Setenv("MIRO_UPLOAD_ALLOWED_DIRS", os.TempDir())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/documents") {
			t.Errorf("expected documents path, got %s", r.URL.Path)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("expected multipart/form-data, got %s", ct)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "doc-upload-123",
			"data": map[string]interface{}{
				"title": "report.pdf",
			},
		})
	}))
	defer server.Close()

	// Create a temp PDF file
	tmpFile, err := os.CreateTemp("", "test-*.pdf")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("%PDF-1.4 test content"))
	tmpFile.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UploadDocument(context.Background(), UploadDocumentArgs{
		BoardID:  "board123",
		FilePath: tmpFile.Name(),
		Title:    "report.pdf",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "doc-upload-123" {
		t.Errorf("ID = %q, want 'doc-upload-123'", result.ID)
	}
	if !strings.Contains(result.Message, "report.pdf") {
		t.Errorf("Message = %q, want containing 'report.pdf'", result.Message)
	}
}

func TestUploadDocument_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    UploadDocumentArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    UploadDocumentArgs{FilePath: "/tmp/test.pdf"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty file path",
			args:    UploadDocumentArgs{BoardID: "board123"},
			wantErr: "file_path is required",
		},
		{
			name:    "nonexistent file",
			args:    UploadDocumentArgs{BoardID: "board123", FilePath: "/nonexistent/file.pdf"},
			wantErr: "cannot access file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UploadDocument(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestUploadDocument_UnsupportedFormat(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.exe")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("not a doc"))
	tmpFile.Close()

	client := newTestClientWithServer("http://localhost")
	_, err = client.UploadDocument(context.Background(), UploadDocumentArgs{
		BoardID:  "board123",
		FilePath: tmpFile.Name(),
	})
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported document format") {
		t.Errorf("error = %q, want containing 'unsupported document format'", err.Error())
	}
}

func TestUploadDocument_FileSizeExceeded(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.pdf")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	data := make([]byte, 6*1024*1024+1)
	tmpFile.Write(data)
	tmpFile.Close()

	client := newTestClientWithServer("http://localhost")
	_, err = client.UploadDocument(context.Background(), UploadDocumentArgs{
		BoardID:  "board123",
		FilePath: tmpFile.Name(),
	})
	if err == nil {
		t.Fatal("expected error for file too large")
	}
	if !strings.Contains(err.Error(), "exceeds 6 MB limit") {
		t.Errorf("error = %q, want containing 'exceeds 6 MB limit'", err.Error())
	}
}

func TestUploadDocument_Directory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-dir")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	client := newTestClientWithServer("http://localhost")
	_, err = client.UploadDocument(context.Background(), UploadDocumentArgs{
		BoardID:  "board123",
		FilePath: tmpDir,
	})
	if err == nil {
		t.Fatal("expected error for directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error = %q, want containing 'directory'", err.Error())
	}
}

func TestUpdateDocumentFromFile_Success(t *testing.T) {
	t.Setenv("MIRO_UPLOAD_ALLOWED_DIRS", os.TempDir())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/documents/doc-789") {
			t.Errorf("expected documents/doc-789 in path, got %s", r.URL.Path)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("expected multipart/form-data, got %s", ct)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "doc-789",
			"data": map[string]interface{}{
				"title": "updated-report.pdf",
			},
		})
	}))
	defer server.Close()

	tmpFile, err := os.CreateTemp("", "test-*.pdf")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("%PDF-1.4 updated content"))
	tmpFile.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateDocumentFromFile(context.Background(), UpdateDocumentFromFileArgs{
		BoardID:  "board123",
		ItemID:   "doc-789",
		FilePath: tmpFile.Name(),
		Title:    "updated-report.pdf",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "doc-789" {
		t.Errorf("ID = %q, want 'doc-789'", result.ID)
	}
	if !strings.Contains(result.Message, "Updated document") {
		t.Errorf("Message = %q, want containing 'Updated document'", result.Message)
	}
}

func TestUpdateDocumentFromFile_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    UpdateDocumentFromFileArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    UpdateDocumentFromFileArgs{ItemID: "doc-1", FilePath: "/tmp/test.pdf"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty item ID",
			args:    UpdateDocumentFromFileArgs{BoardID: "board123", FilePath: "/tmp/test.pdf"},
			wantErr: "item_id is required",
		},
		{
			name:    "empty file path",
			args:    UpdateDocumentFromFileArgs{BoardID: "board123", ItemID: "doc-1"},
			wantErr: "file_path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UpdateDocumentFromFile(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestUpdateDocumentFromFile_UnsupportedFormat(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.exe")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("not a document"))
	tmpFile.Close()

	client := newTestClientWithServer("http://localhost")
	_, err = client.UpdateDocumentFromFile(context.Background(), UpdateDocumentFromFileArgs{
		BoardID:  "board123",
		ItemID:   "doc-1",
		FilePath: tmpFile.Name(),
	})
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported document format") {
		t.Errorf("error = %q, want containing 'unsupported document format'", err.Error())
	}
}

func TestUpdateDocumentFromFile_FileSizeExceeded(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.pdf")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	data := make([]byte, 6*1024*1024+1)
	tmpFile.Write(data)
	tmpFile.Close()

	client := newTestClientWithServer("http://localhost")
	_, err = client.UpdateDocumentFromFile(context.Background(), UpdateDocumentFromFileArgs{
		BoardID:  "board123",
		ItemID:   "doc-1",
		FilePath: tmpFile.Name(),
	})
	if err == nil {
		t.Fatal("expected error for file too large")
	}
	if !strings.Contains(err.Error(), "exceeds 6 MB limit") {
		t.Errorf("error = %q, want containing 'exceeds 6 MB limit'", err.Error())
	}
}
