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

// =============================================================================
// Shared sticky test helpers
// =============================================================================

// newStickyVerifyServer asserts each request matches "METHOD /path" and then
// serves the given JSON response with the given status.
func newStickyVerifyServer(t *testing.T, wantMethodAndPath string, status int, response map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Method + " " + r.URL.Path; got != wantMethodAndPath {
			t.Errorf("request = %q, want %q", got, wantMethodAndPath)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(response)
	}))
}

// requireBodyField asserts one field of a decoded request body, addressed as
// "section.field" (e.g. "geometry.width").
func requireBodyField(t *testing.T, body map[string]interface{}, path string, want interface{}) {
	t.Helper()
	section, field, _ := strings.Cut(path, ".")
	sec, ok := body[section].(map[string]interface{})
	if !ok {
		t.Errorf("expected %s in request body", section)
		return
	}
	if sec[field] != want {
		t.Errorf("%s = %v, want %v", path, sec[field], want)
	}
}

// stickyValidationCase is one expected-error scenario for a sticky API call.
type stickyValidationCase struct {
	name    string
	wantErr string
	call    func() error
}

func runStickyValidationCases(t *testing.T, cases []stickyValidationCase) {
	t.Helper()
	for _, tt := range cases {
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

// =============================================================================
// Tests
// =============================================================================

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

// TestSticky_Success covers the create and update happy paths: the request
// line must match, and the response must map onto the result struct.
func TestSticky_Success(t *testing.T) {
	tests := []struct {
		name        string
		wantRequest string
		status      int
		response    map[string]interface{}
		call        func(*Client) (id, content string, err error)
		wantID      string
		wantContent string
	}{
		{
			name:        "create sticky",
			wantRequest: "POST /boards/board123/sticky_notes",
			status:      http.StatusCreated,
			response: map[string]interface{}{
				"id":    "sticky-id",
				"data":  map[string]interface{}{"content": "Test sticky"},
				"style": map[string]interface{}{"fillColor": "light_yellow"},
			},
			call: func(client *Client) (string, string, error) {
				result, err := client.CreateSticky(context.Background(), CreateStickyArgs{
					BoardID: "board123",
					Content: "Test sticky",
					Color:   "yellow",
				})
				if err != nil {
					return "", "", err
				}
				return result.ID, result.Content, nil
			},
			wantID:      "sticky-id",
			wantContent: "Test sticky",
		},
		{
			name:        "update sticky",
			wantRequest: "PATCH /boards/board123/sticky_notes/sticky123",
			status:      http.StatusOK,
			response: map[string]interface{}{
				"id":    "sticky123",
				"data":  map[string]interface{}{"content": "Updated content", "shape": "square"},
				"style": map[string]interface{}{"fillColor": "light_blue"},
			},
			call: func(client *Client) (string, string, error) {
				result, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
					BoardID: "board123",
					ItemID:  "sticky123",
					Content: strPtr("Updated content"),
					Color:   strPtr("blue"),
				})
				if err != nil {
					return "", "", err
				}
				return result.ID, result.Content, nil
			},
			wantID:      "sticky123",
			wantContent: "Updated content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newStickyVerifyServer(t, tt.wantRequest, tt.status, tt.response)
			defer server.Close()

			id, content, err := tt.call(newTestClientWithServer(server.URL))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.wantID {
				t.Errorf("ID = %q, want %q", id, tt.wantID)
			}
			if content != tt.wantContent {
				t.Errorf("Content = %q, want %q", content, tt.wantContent)
			}
		})
	}
}

func TestCreateSticky_WithWidthAndParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		requireBodyField(t, body, "geometry.width", float64(250))
		requireBodyField(t, body, "parent.id", "frame-abc")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "sticky-geo",
			"data":  map[string]interface{}{"content": "Wide sticky"},
			"style": map[string]interface{}{"fillColor": "light_yellow"},
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
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
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

// TestSticky_ValidationErrors covers every argument-validation failure across
// CreateSticky, CreateStickyGrid, and UpdateSticky. Validation fires before
// any HTTP request, so no server is needed.
func TestSticky_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://unused")

	runStickyValidationCases(t, []stickyValidationCase{
		{
			name:    "create sticky empty board_id",
			wantErr: "board_id is required",
			call: func() error {
				_, err := client.CreateSticky(context.Background(), CreateStickyArgs{Content: "Test"})
				return err
			},
		},
		{
			name:    "create sticky empty content",
			wantErr: "content is required",
			call: func() error {
				_, err := client.CreateSticky(context.Background(), CreateStickyArgs{BoardID: "board123"})
				return err
			},
		},
		{
			name:    "create sticky empty board_id mentions board_id",
			wantErr: "board_id",
			call: func() error {
				_, err := client.CreateSticky(context.Background(), CreateStickyArgs{BoardID: "", Content: "test"})
				return err
			},
		},
		{
			name:    "create sticky empty content mentions content",
			wantErr: "content",
			call: func() error {
				_, err := client.CreateSticky(context.Background(), CreateStickyArgs{BoardID: "board123", Content: ""})
				return err
			},
		},
		{
			name:    "grid empty board ID",
			wantErr: "board_id is required",
			call: func() error {
				_, err := client.CreateStickyGrid(context.Background(), CreateStickyGridArgs{Contents: []string{"Test"}})
				return err
			},
		},
		{
			name:    "grid empty contents",
			wantErr: "at least one content item is required",
			call: func() error {
				_, err := client.CreateStickyGrid(context.Background(), CreateStickyGridArgs{BoardID: "board123"})
				return err
			},
		},
		{
			name:    "update empty board_id",
			wantErr: "",
			call: func() error {
				_, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
					BoardID: "",
					ItemID:  "sticky123",
					Content: strPtr("test"),
				})
				return err
			},
		},
		{
			name:    "update empty item_id",
			wantErr: "",
			call: func() error {
				_, err := client.UpdateSticky(context.Background(), UpdateStickyArgs{
					BoardID: "board123",
					ItemID:  "",
					Content: strPtr("test"),
				})
				return err
			},
		},
	})
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

func TestUpdateSticky_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)

		requireBodyField(t, body, "data.content", "Updated content")
		requireBodyField(t, body, "data.shape", "circle")
		requireBodyField(t, body, "style.fillColor", "blue")
		requireBodyField(t, body, "position.x", float64(100))
		requireBodyField(t, body, "position.y", float64(200))
		requireBodyField(t, body, "geometry.width", float64(300))
		requireBodyField(t, body, "parent.id", "frame123")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
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
