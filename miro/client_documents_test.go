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

// newDocumentTestClient starts a test server with the given handler and
// returns a client pointed at it. The server is closed via t.Cleanup.
func newDocumentTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

// documentJSONHandler returns a handler that writes the given payload as JSON.
func documentJSONHandler(payload map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

// createTempDocFile creates a temp file with the given name pattern and
// content, removed via t.Cleanup, and returns its path.
func createTempDocFile(t *testing.T, pattern string, content []byte) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })
	tmpFile.Write(content)
	tmpFile.Close()
	return tmpFile.Name()
}

// multipartDocExpect describes the request a multipart document handler
// asserts and the response payload it returns.
type multipartDocExpect struct {
	method   string
	pathPart string
	id       string
	title    string
}

// multipartDocHandler returns a handler asserting a multipart request matching
// the expectation, then responding with a document payload.
func multipartDocHandler(t *testing.T, expect multipartDocExpect) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != expect.method {
			t.Errorf("expected %s, got %s", expect.method, r.Method)
		}
		if !strings.Contains(r.URL.Path, expect.pathPart) {
			t.Errorf("expected %s in path, got %s", expect.pathPart, r.URL.Path)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("expected multipart/form-data, got %s", ct)
		}
		documentJSONHandler(map[string]interface{}{
			"id": expect.id,
			"data": map[string]interface{}{
				"title": expect.title,
			},
		})(w, r)
	}
}

func TestGetDocument_Success(t *testing.T) {
	client := newDocumentTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/boards/board123/documents/doc789") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		documentJSONHandler(map[string]interface{}{
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
		})(w, r)
	})

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

// TestDocumentValidationErrors covers argument validation across GetDocument,
// CreateDocument, UploadDocument, and UpdateDocumentFromFile; every call must
// fail before any HTTP request is made.
func TestDocumentValidationErrors(t *testing.T) {
	c := newTestClientWithServer("http://localhost")
	ctx := context.Background()
	get := func(args GetDocumentArgs) func() error {
		return func() error { _, err := c.GetDocument(ctx, args); return err }
	}
	create := func(args CreateDocumentArgs) func() error {
		return func() error { _, err := c.CreateDocument(ctx, args); return err }
	}
	upload := func(args UploadDocumentArgs) func() error {
		return func() error { _, err := c.UploadDocument(ctx, args); return err }
	}
	updateFromFile := func(args UpdateDocumentFromFileArgs) func() error {
		return func() error { _, err := c.UpdateDocumentFromFile(ctx, args); return err }
	}
	tests := []struct {
		name    string
		call    func() error
		wantErr string
	}{
		{"get empty board_id", get(GetDocumentArgs{ItemID: "doc789"}), "board_id is required"},
		{"get empty item_id", get(GetDocumentArgs{BoardID: "board123"}), "item_id is required"},
		{"create empty board ID", create(CreateDocumentArgs{URL: "https://example.com/doc.pdf"}), "board_id is required"},
		{"create empty URL", create(CreateDocumentArgs{BoardID: "board123"}), "url is required"},
		{"upload empty board ID", upload(UploadDocumentArgs{FilePath: "/tmp/test.pdf"}), "board_id is required"},
		{"upload empty file path", upload(UploadDocumentArgs{BoardID: "board123"}), "file_path is required"},
		{"upload nonexistent file", upload(UploadDocumentArgs{BoardID: "board123", FilePath: "/nonexistent/file.pdf"}), "cannot access file"},
		{"update from file empty board ID", updateFromFile(UpdateDocumentFromFileArgs{ItemID: "doc-1", FilePath: "/tmp/test.pdf"}), "board_id is required"},
		{"update from file empty item ID", updateFromFile(UpdateDocumentFromFileArgs{BoardID: "board123", FilePath: "/tmp/test.pdf"}), "item_id is required"},
		{"update from file empty file path", updateFromFile(UpdateDocumentFromFileArgs{BoardID: "board123", ItemID: "doc-1"}), "file_path is required"},
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

// documentWriteHandler asserts the request method and path fragment, then
// responds with the given document payload.
func documentWriteHandler(t *testing.T, method, pathPart string, payload map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			t.Errorf("expected %s, got %s", method, r.Method)
		}
		if !strings.Contains(r.URL.Path, pathPart) {
			t.Errorf("expected %s in path, got %s", pathPart, r.URL.Path)
		}
		documentJSONHandler(payload)(w, r)
	}
}

// TestDocumentWrite_Success covers the JSON create and update paths.
func TestDocumentWrite_Success(t *testing.T) {
	docPayload := func(title string) map[string]interface{} {
		return map[string]interface{}{"id": "doc123", "data": map[string]interface{}{"title": title}}
	}
	tests := []struct {
		name      string
		method    string
		pathPart  string
		payload   map[string]interface{}
		call      func(c *Client) (id, title string, err error)
		wantTitle string
	}{
		{
			name:     "CreateDocument",
			method:   http.MethodPost,
			pathPart: "/documents",
			payload:  docPayload("Test Document"),
			call: func(c *Client) (string, string, error) {
				result, err := c.CreateDocument(context.Background(), CreateDocumentArgs{
					BoardID: "board123", Title: "Test Document", URL: "https://example.com/doc.pdf",
				})
				if err != nil {
					return "", "", err
				}
				return result.ID, "", nil
			},
		},
		{
			name:     "UpdateDocument",
			method:   http.MethodPatch,
			pathPart: "/documents/",
			payload:  docPayload("Updated document"),
			call: func(c *Client) (string, string, error) {
				result, err := c.UpdateDocument(context.Background(), UpdateDocumentArgs{
					BoardID: "board123", ItemID: "doc123", Title: strPtr("Updated document"),
				})
				if err != nil {
					return "", "", err
				}
				return result.ID, result.Title, nil
			},
			wantTitle: "Updated document",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newDocumentTestClient(t, documentWriteHandler(t, tt.method, tt.pathPart, tt.payload))
			id, title, err := tt.call(client)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != "doc123" {
				t.Errorf("ID = %q, want 'doc123'", id)
			}
			if tt.wantTitle != "" && title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", title, tt.wantTitle)
			}
		})
	}
}

func TestCreateDocument_WithAllFields(t *testing.T) {
	// Tests CreateDocument with all optional fields
	client := newDocumentTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
		documentJSONHandler(map[string]interface{}{
			"id":   "doc123",
			"type": "document",
		})(w, r)
	})

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

// TestDocumentMultipart_Success covers the multipart upload paths:
// UploadDocument (POST) and UpdateDocumentFromFile (PATCH).
func TestDocumentMultipart_Success(t *testing.T) {
	tests := []struct {
		name        string
		expect      multipartDocExpect
		content     string
		call        func(c *Client, path string) (id, message string, err error)
		wantMessage string
	}{
		{
			name: "UploadDocument",
			expect: multipartDocExpect{
				method: http.MethodPost, pathPart: "/documents",
				id: "doc-upload-123", title: "report.pdf",
			},
			content: "%PDF-1.4 test content",
			call: func(c *Client, path string) (string, string, error) {
				result, err := c.UploadDocument(context.Background(), UploadDocumentArgs{
					BoardID: "board123", FilePath: path, Title: "report.pdf",
				})
				if err != nil {
					return "", "", err
				}
				return result.ID, result.Message, nil
			},
			wantMessage: "report.pdf",
		},
		{
			name: "UpdateDocumentFromFile",
			expect: multipartDocExpect{
				method: http.MethodPatch, pathPart: "/documents/doc-789",
				id: "doc-789", title: "updated-report.pdf",
			},
			content: "%PDF-1.4 updated content",
			call: func(c *Client, path string) (string, string, error) {
				result, err := c.UpdateDocumentFromFile(context.Background(), UpdateDocumentFromFileArgs{
					BoardID: "board123", ItemID: "doc-789", FilePath: path, Title: "updated-report.pdf",
				})
				if err != nil {
					return "", "", err
				}
				return result.ID, result.Message, nil
			},
			wantMessage: "Updated document",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MIRO_UPLOAD_ALLOWED_DIRS", os.TempDir())
			client := newDocumentTestClient(t, multipartDocHandler(t, tt.expect))
			filePath := createTempDocFile(t, "test-*.pdf", []byte(tt.content))

			id, message, err := tt.call(client, filePath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.expect.id {
				t.Errorf("ID = %q, want %q", id, tt.expect.id)
			}
			if !strings.Contains(message, tt.wantMessage) {
				t.Errorf("Message = %q, want containing %q", message, tt.wantMessage)
			}
		})
	}
}

// TestDocumentFileGuards covers local file checks (format, size, directory)
// for UploadDocument and UpdateDocumentFromFile; no HTTP request is made.
func TestDocumentFileGuards(t *testing.T) {
	oversized := make([]byte, 6*1024*1024+1)
	upload := func(c *Client, path string) error {
		_, err := c.UploadDocument(context.Background(), UploadDocumentArgs{
			BoardID: "board123", FilePath: path,
		})
		return err
	}
	updateFromFile := func(c *Client, path string) error {
		_, err := c.UpdateDocumentFromFile(context.Background(), UpdateDocumentFromFileArgs{
			BoardID: "board123", ItemID: "doc-1", FilePath: path,
		})
		return err
	}

	tests := []struct {
		name    string
		path    func(t *testing.T) string
		call    func(c *Client, path string) error
		wantErr string
	}{
		{
			name:    "UploadUnsupportedFormat",
			path:    func(t *testing.T) string { return createTempDocFile(t, "test-*.exe", []byte("not a doc")) },
			call:    upload,
			wantErr: "unsupported document format",
		},
		{
			name:    "UploadFileSizeExceeded",
			path:    func(t *testing.T) string { return createTempDocFile(t, "test-*.pdf", oversized) },
			call:    upload,
			wantErr: "exceeds 6 MB limit",
		},
		{
			name: "UploadDirectory",
			path: func(t *testing.T) string {
				tmpDir, err := os.MkdirTemp("", "test-dir")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				t.Cleanup(func() { os.RemoveAll(tmpDir) })
				return tmpDir
			},
			call:    upload,
			wantErr: "directory",
		},
		{
			name:    "UpdateFromFileUnsupportedFormat",
			path:    func(t *testing.T) string { return createTempDocFile(t, "test-*.exe", []byte("not a document")) },
			call:    updateFromFile,
			wantErr: "unsupported document format",
		},
		{
			name:    "UpdateFromFileFileSizeExceeded",
			path:    func(t *testing.T) string { return createTempDocFile(t, "test-*.pdf", oversized) },
			call:    updateFromFile,
			wantErr: "exceeds 6 MB limit",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClientWithServer("http://localhost")
			err := tt.call(client, tt.path(t))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}
