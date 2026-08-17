package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// txCheckMethod verifies the HTTP method of a request.
func txCheckMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if r.Method != want {
		t.Errorf("expected %s, got %s", want, r.Method)
	}
}

// txDecodeBody decodes a JSON request body into a generic map.
func txDecodeBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	json.NewDecoder(r.Body).Decode(&body)
	return body
}

// txSection extracts a nested JSON object such as data, style, position, geometry, or parent.
func txSection(t *testing.T, body map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	m, ok := body[key].(map[string]interface{})
	if !ok {
		t.Fatalf("expected %s in request body, got %v", key, body[key])
	}
	return m
}

// txCheckField verifies a field of a decoded JSON object.
func txCheckField(t *testing.T, m map[string]interface{}, field string, want interface{}) {
	t.Helper()
	if m[field] != want {
		t.Errorf("%s = %v, want %v", field, m[field], want)
	}
}

// txCheckEq verifies a single result field against its expected value.
func txCheckEq(t *testing.T, name string, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// txServeCreated writes a JSON 201 response for a created text item.
func txServeCreated(w http.ResponseWriter, id, content string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": id,
		"data": map[string]interface{}{
			"content": content,
		},
	})
}

func TestCreateText_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		txCheckMethod(t, r, http.MethodPost)
		if r.URL.Path != "/boards/board123/texts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		txServeCreated(w, "text123", "Hello World")
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
	txCheckEq(t, "ID", result.ID, "text123")
	txCheckEq(t, "Content", result.Content, "Hello World")
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
		style := txSection(t, txDecodeBody(t, r), "style")
		txCheckField(t, style, "fontSize", "24")
		txCheckField(t, style, "color", "#ff0000")
		txServeCreated(w, "text-styled", "Styled text")
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
	txCheckEq(t, "ID", result.ID, "text-styled")
}

func TestCreateText_WithGeometryAndParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := txDecodeBody(t, r)
		txCheckField(t, txSection(t, body, "geometry"), "width", float64(300))
		txCheckField(t, txSection(t, body, "parent"), "id", "frame123")
		txServeCreated(w, "text-geom", "Text with width")
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
	txCheckEq(t, "ID", result.ID, "text-geom")
}

func TestUpdateText_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		txCheckMethod(t, r, http.MethodPatch)
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
	txCheckEq(t, "ID", result.ID, "text123")
	txCheckEq(t, "Content", result.Content, "Updated text")
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
	txCheckEq(t, "Message", result.Message, "No changes specified")
}

func TestUpdateText_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := txDecodeBody(t, r)

		txCheckField(t, txSection(t, body, "data"), "content", "Updated text content")

		style := txSection(t, body, "style")
		txCheckField(t, style, "fontSize", "18")
		txCheckField(t, style, "textAlign", "center")
		txCheckField(t, style, "color", "#333333")

		pos := txSection(t, body, "position")
		txCheckField(t, pos, "x", float64(100))
		txCheckField(t, pos, "y", float64(200))

		txCheckField(t, txSection(t, body, "geometry"), "width", float64(400))
		txCheckField(t, txSection(t, body, "parent"), "id", "frame-abc")

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
	txCheckEq(t, "ID", result.ID, "text123")
}
