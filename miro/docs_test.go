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

// docServerSpec describes the fake Miro API behavior for one doc format test:
// the expected request line, an optional pointer that receives the decoded
// request body, and the response to send back.
type docServerSpec struct {
	wantMethod    string
	wantPath      string
	status        int
	rawBody       string
	response      map[string]interface{}
	captureBody   *map[string]interface{}
	failOnRequest bool
}

// newDocClient starts a fake Miro API described by spec and returns a client
// pointed at it.
func newDocClient(t *testing.T, spec docServerSpec) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if spec.failOnRequest {
			t.Error("the API must not be called in this scenario")
			return
		}
		assertDocRequestLine(t, r, spec.wantMethod, spec.wantPath)
		if spec.captureBody != nil {
			if err := json.NewDecoder(r.Body).Decode(spec.captureBody); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}
		}
		writeDocResponse(w, spec)
	}))
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

func assertDocRequestLine(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if method != "" && r.Method != method {
		t.Errorf("expected %s, got %s", method, r.Method)
	}
	if path != "" && r.URL.Path != path {
		t.Errorf("unexpected path %s", r.URL.Path)
	}
}

func writeDocResponse(w http.ResponseWriter, spec docServerSpec) {
	w.Header().Set("Content-Type", "application/json")
	status := spec.status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if _, err := w.Write(docResponseBody(spec)); err != nil {
		panic(err)
	}
}

// docResponseBody renders the response body for a spec: the raw body verbatim,
// the response payload as JSON, or nothing.
func docResponseBody(spec docServerSpec) []byte {
	if spec.rawBody != "" {
		return []byte(spec.rawBody)
	}
	if spec.response == nil {
		return nil
	}
	encoded, err := json.Marshal(spec.response)
	if err != nil {
		panic(err)
	}
	return encoded
}

// =============================================================================
// Doc Format Tests
// =============================================================================

func TestCreateDocFormat_Success(t *testing.T) {
	var gotBody map[string]interface{}
	client := newDocClient(t, docServerSpec{
		wantMethod:  http.MethodPost,
		wantPath:    "/boards/board123/docs",
		status:      http.StatusCreated,
		response:    map[string]interface{}{"id": "doc123"},
		captureBody: &gotBody,
	})

	result, err := client.CreateDocFormat(context.Background(), CreateDocFormatArgs{
		BoardID: "board123",
		Content: "# Heading",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "doc123" {
		t.Errorf("ID = %q, want doc123", result.ID)
	}
	if result.Message != "Created doc format item" {
		t.Errorf("Message = %q", result.Message)
	}
	if result.ItemURL != BuildItemURL("board123", "doc123") {
		t.Errorf("ItemURL = %q", result.ItemURL)
	}
	assertDocCreateBody(t, gotBody)
}

// assertDocCreateBody verifies the minimal create request body: markdown data
// present, optional position and parent sections omitted.
func assertDocCreateBody(t *testing.T, gotBody map[string]interface{}) {
	t.Helper()
	data := bodySection(t, gotBody, "data")
	if data["contentType"] != "markdown" {
		t.Errorf("contentType = %v, want markdown", data["contentType"])
	}
	if data["content"] != "# Heading" {
		t.Errorf("content = %v, want # Heading", data["content"])
	}
	if _, present := gotBody["position"]; present {
		t.Error("position should be omitted when x and y are both zero")
	}
	if _, present := gotBody["parent"]; present {
		t.Error("parent should be omitted when parent_id is empty")
	}
}

func TestCreateDocFormat_WithPositionAndParent(t *testing.T) {
	var gotBody map[string]interface{}
	client := newDocClient(t, docServerSpec{
		response:    map[string]interface{}{"id": "doc123"},
		captureBody: &gotBody,
	})

	if _, err := client.CreateDocFormat(context.Background(), CreateDocFormatArgs{
		BoardID:  "board123",
		Content:  "body",
		X:        100,
		Y:        -50,
		ParentID: "frame999",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pos := bodySection(t, gotBody, "position")
	if pos["x"] != 100.0 || pos["y"] != -50.0 {
		t.Errorf("position = (%v, %v), want (100, -50)", pos["x"], pos["y"])
	}
	if pos["origin"] != "center" {
		t.Errorf("origin = %v, want center", pos["origin"])
	}
	if parent := bodySection(t, gotBody, "parent"); parent["id"] != "frame999" {
		t.Errorf("parent id = %v, want frame999", parent["id"])
	}
}

func TestCreateDocFormat_YOnlyStillSendsPosition(t *testing.T) {
	var gotBody map[string]interface{}
	client := newDocClient(t, docServerSpec{
		response:    map[string]interface{}{"id": "doc123"},
		captureBody: &gotBody,
	})

	if _, err := client.CreateDocFormat(context.Background(), CreateDocFormatArgs{
		BoardID: "board123",
		Content: "body",
		Y:       42,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, present := gotBody["position"]; !present {
		t.Error("position should be sent when only y is non-zero")
	}
}

func TestDocFormat_ValidationErrors(t *testing.T) {
	client := newDocClient(t, docServerSpec{failOnRequest: true})
	ctx := context.Background()

	tests := []struct {
		name    string
		wantErr string
		call    func() error
	}{
		{"create empty board", "board_id is required", func() error {
			_, err := client.CreateDocFormat(ctx, CreateDocFormatArgs{BoardID: "", Content: "x"})
			return err
		}},
		{"create empty content", "content is required", func() error {
			_, err := client.CreateDocFormat(ctx, CreateDocFormatArgs{BoardID: "board123", Content: ""})
			return err
		}},
		{"get empty board", "board_id is required", func() error {
			_, err := client.GetDocFormat(ctx, GetDocFormatArgs{BoardID: "", ItemID: "doc123"})
			return err
		}},
		{"get empty item", "item_id is required", func() error {
			_, err := client.GetDocFormat(ctx, GetDocFormatArgs{BoardID: "board123", ItemID: ""})
			return err
		}},
		{"delete empty board", "board_id is required", func() error {
			_, err := client.DeleteDocFormat(ctx, DeleteDocFormatArgs{BoardID: "", ItemID: "doc123"})
			return err
		}},
		{"delete empty item", "item_id is required", func() error {
			_, err := client.DeleteDocFormat(ctx, DeleteDocFormatArgs{BoardID: "board123", ItemID: ""})
			return err
		}},
		{"update empty board", "board_id is required", func() error {
			_, err := client.UpdateDocFormat(ctx, UpdateDocFormatArgs{BoardID: "", ItemID: "doc123"})
			return err
		}},
		{"update empty item", "item_id is required", func() error {
			_, err := client.UpdateDocFormat(ctx, UpdateDocFormatArgs{BoardID: "board123", ItemID: ""})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestDocFormat_APIErrors(t *testing.T) {
	ctx := context.Background()
	createCall := func(c *Client) error {
		_, err := c.CreateDocFormat(ctx, CreateDocFormatArgs{BoardID: "board123", Content: "x"})
		return err
	}
	getCall := func(c *Client) error {
		_, err := c.GetDocFormat(ctx, GetDocFormatArgs{BoardID: "board123", ItemID: "doc123"})
		return err
	}
	deleteCall := func(c *Client) error {
		_, err := c.DeleteDocFormat(ctx, DeleteDocFormatArgs{BoardID: "board123", ItemID: "doc123"})
		return err
	}
	updateCall := func(c *Client) error {
		_, err := c.UpdateDocFormat(ctx, UpdateDocFormatArgs{BoardID: "board123", ItemID: "doc123", Content: "new"})
		return err
	}

	tests := []struct {
		name    string
		status  int
		rawBody string
		wantErr string // empty means any error is acceptable
		call    func(*Client) error
	}{
		{"create HTTP 400", http.StatusBadRequest, "", "", createCall},
		{"create malformed JSON", http.StatusOK, "{not json", "failed to parse response", createCall},
		{"get HTTP 404", http.StatusNotFound, "", "", getCall},
		{"get malformed JSON", http.StatusOK, "{not json", "failed to parse response", getCall},
		{"delete HTTP 403", http.StatusForbidden, "", "", deleteCall},
		{"update read fails", http.StatusNotFound, "", "failed to read current doc", updateCall},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newDocClient(t, docServerSpec{status: tt.status, rawBody: tt.rawBody})
			err := tt.call(client)
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestGetDocFormat_Success(t *testing.T) {
	client := newDocClient(t, docServerSpec{
		wantMethod: http.MethodGet,
		wantPath:   "/boards/board123/docs/doc123",
		response: map[string]interface{}{
			"id":         "doc123",
			"data":       map[string]interface{}{"content": "# Heading"},
			"position":   map[string]interface{}{"x": 10.0, "y": 20.0},
			"createdAt":  "2026-01-01T00:00:00Z",
			"modifiedAt": "2026-01-02T00:00:00Z",
			"createdBy":  map[string]interface{}{"id": "user1"},
			"modifiedBy": map[string]interface{}{"id": "user2"},
		},
	})

	result, err := client.GetDocFormat(context.Background(), GetDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "doc123" {
		t.Errorf("ID = %q, want doc123", result.ID)
	}
	if result.Content != "# Heading" {
		t.Errorf("Content = %q, want # Heading", result.Content)
	}
	assertDocMetadata(t, result)
}

// assertDocMetadata verifies the position, actor, and message fields returned
// by TestGetDocFormat_Success.
func assertDocMetadata(t *testing.T, result GetDocFormatResult) {
	t.Helper()
	if result.X != 10 || result.Y != 20 {
		t.Errorf("position = (%v, %v), want (10, 20)", result.X, result.Y)
	}
	if result.CreatedBy != "user1" || result.ModifiedBy != "user2" {
		t.Errorf("actors = (%q, %q), want (user1, user2)", result.CreatedBy, result.ModifiedBy)
	}
	if result.Message != "Retrieved doc format item" {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestDeleteDocFormat_Success(t *testing.T) {
	client := newDocClient(t, docServerSpec{
		wantMethod: http.MethodDelete,
		wantPath:   "/boards/board123/docs/doc123",
		status:     http.StatusNoContent,
	})

	result, err := client.DeleteDocFormat(context.Background(), DeleteDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success = false, want true")
	}
	if result.ItemID != "doc123" {
		t.Errorf("ItemID = %q, want doc123", result.ItemID)
	}
	if result.Message != "Doc format item deleted successfully" {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestDeleteDocFormat_DryRun(t *testing.T) {
	client := newDocClient(t, docServerSpec{failOnRequest: true})
	result, err := client.DeleteDocFormat(context.Background(), DeleteDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success = false, want true")
	}
	if !strings.HasPrefix(result.Message, "[DRY RUN]") {
		t.Errorf("Message = %q, want a [DRY RUN] prefix", result.Message)
	}
}

// docUpdateServer mimics the read-delete-recreate sequence UpdateDocFormat drives.
func docUpdateServer(t *testing.T, currentContent string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			writeDocResponse(w, docServerSpec{response: map[string]interface{}{
				"id":       "doc123",
				"data":     map[string]interface{}{"content": currentContent},
				"position": map[string]interface{}{"x": 10.0, "y": 20.0},
			}})
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost:
			writeDocResponse(w, docServerSpec{response: map[string]interface{}{"id": "doc456"}})
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
}

func TestUpdateDocFormat_FullReplace(t *testing.T) {
	server := docUpdateServer(t, "old body")
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateDocFormat(context.Background(), UpdateDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
		Content: "new body",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "doc456" {
		t.Errorf("ID = %q, want doc456", result.ID)
	}
	if result.OldID != "doc123" {
		t.Errorf("OldID = %q, want doc123", result.OldID)
	}
	if result.Content != "new body" {
		t.Errorf("Content = %q, want new body", result.Content)
	}
	if result.Replaced != 0 {
		t.Errorf("Replaced = %d, want 0", result.Replaced)
	}
	if result.Message != "Updated doc format item" {
		t.Errorf("Message = %q", result.Message)
	}
	if result.ItemURL != BuildItemURL("board123", "doc456") {
		t.Errorf("ItemURL = %q", result.ItemURL)
	}
}

func TestUpdateDocFormat_FindAndReplace(t *testing.T) {
	server := docUpdateServer(t, "foo bar foo")
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateDocFormat(context.Background(), UpdateDocFormatArgs{
		BoardID:    "board123",
		ItemID:     "doc123",
		OldContent: "foo",
		NewContent: "baz",
		ReplaceAll: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "baz bar baz" {
		t.Errorf("Content = %q, want baz bar baz", result.Content)
	}
	if result.Replaced != 2 {
		t.Errorf("Replaced = %d, want 2", result.Replaced)
	}
	if result.Message != "Replaced 2 occurrence(s) in doc format item" {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestUpdateDocFormat_ResolveFails(t *testing.T) {
	server := docUpdateServer(t, "old body")
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.UpdateDocFormat(context.Background(), UpdateDocFormatArgs{
		BoardID: "board123",
		ItemID:  "doc123",
	})
	if err == nil || !strings.Contains(err.Error(), "either content (full replace)") {
		t.Errorf("error = %v, want the either-content error", err)
	}
}

// TestUpdateDocFormat_StepFailures drives the read-delete-recreate sequence
// with failures injected at the delete and recreate steps.
func TestUpdateDocFormat_StepFailures(t *testing.T) {
	serverError := func(w http.ResponseWriter) { w.WriteHeader(http.StatusInternalServerError) }
	noContent := func(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }
	malformed := func(w http.ResponseWriter) {
		writeDocResponse(w, docServerSpec{rawBody: "{not json"})
	}

	tests := []struct {
		name     string
		onDelete func(http.ResponseWriter)
		onPost   func(http.ResponseWriter)
		wantErr  string
	}{
		{"delete fails", serverError, serverError, "failed to delete original doc"},
		{"recreate fails", noContent, serverError, "failed to recreate doc with updated content"},
		{"recreate malformed JSON", noContent, malformed, "failed to parse response"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					writeDocResponse(w, docServerSpec{response: map[string]interface{}{
						"id":   "doc123",
						"data": map[string]interface{}{"content": "old"},
					}})
				case http.MethodDelete:
					tt.onDelete(w)
				default:
					tt.onPost(w)
				}
			}))
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			_, err := client.UpdateDocFormat(context.Background(), UpdateDocFormatArgs{
				BoardID: "board123",
				ItemID:  "doc123",
				Content: "new",
			})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
