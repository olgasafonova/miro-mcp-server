package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// embCheckRequest verifies the HTTP method and a URL path fragment of a request.
func embCheckRequest(t *testing.T, r *http.Request, method, pathFragment string) {
	t.Helper()
	if r.Method != method {
		t.Errorf("expected %s, got %s", method, r.Method)
	}
	if !strings.Contains(r.URL.Path, pathFragment) {
		t.Errorf("expected %s in path, got %s", pathFragment, r.URL.Path)
	}
}

// embSection extracts a nested JSON object such as data or geometry from a request body.
func embSection(t *testing.T, body map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	m, ok := body[key].(map[string]interface{})
	if !ok {
		t.Fatalf("expected %s in request body, got %v", key, body[key])
	}
	return m
}

// embCheckField verifies a field of a decoded JSON object.
func embCheckField(t *testing.T, m map[string]interface{}, field string, want interface{}) {
	t.Helper()
	if m[field] != want {
		t.Errorf("%s = %v, want %v", field, m[field], want)
	}
}

// embCheckEq verifies a single result field against its expected value.
func embCheckEq(t *testing.T, name string, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// embServeJSON writes a JSON response.
func embServeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func TestEmbeds_Success(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		pathFrag string
		respData map[string]interface{}
		call     func(*Client) (id, url string, err error)
		wantURL  string
	}{
		{
			name:     "create embed",
			method:   http.MethodPost,
			pathFrag: "/embeds",
			respData: map[string]interface{}{
				"url": "https://youtube.com/watch?v=test",
			},
			call: func(c *Client) (string, string, error) {
				result, err := c.CreateEmbed(context.Background(), CreateEmbedArgs{
					BoardID: "board123",
					URL:     "https://youtube.com/watch?v=test",
				})
				if err != nil {
					return "", "", err
				}
				return result.ID, "", nil
			},
		},
		{
			name:     "update embed",
			method:   http.MethodPatch,
			pathFrag: "/embeds/",
			respData: map[string]interface{}{
				"url":         "https://youtube.com/watch?v=123",
				"providerUrl": "youtube.com",
			},
			call: func(c *Client) (string, string, error) {
				result, err := c.UpdateEmbed(context.Background(), UpdateEmbedArgs{
					BoardID: "board123",
					ItemID:  "embed123",
					URL:     strPtr("https://youtube.com/watch?v=123"),
				})
				if err != nil {
					return "", "", err
				}
				return result.ID, result.URL, nil
			},
			wantURL: "https://youtube.com/watch?v=123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				embCheckRequest(t, r, tt.method, tt.pathFrag)
				embServeJSON(w, map[string]interface{}{"id": "embed123", "data": tt.respData})
			}))
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			id, url, err := tt.call(client)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			embCheckEq(t, "ID", id, "embed123")
			if tt.wantURL != "" {
				embCheckEq(t, "URL", url, tt.wantURL)
			}
		})
	}
}

func TestCreateEmbed_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    CreateEmbedArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    CreateEmbedArgs{URL: "https://youtube.com/watch?v=test"},
			wantErr: "board_id is required",
		},
		{
			name:    "empty URL",
			args:    CreateEmbedArgs{BoardID: "board123"},
			wantErr: "url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateEmbed(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCreateEmbed_WithAllFields(t *testing.T) {
	// Tests CreateEmbed with all optional fields
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		data := embSection(t, body, "data")
		embCheckField(t, data, "url", "https://youtube.com/watch?v=abc123")
		embCheckField(t, data, "mode", "modal")

		embServeJSON(w, map[string]interface{}{
			"id":   "embed123",
			"type": "embed",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateEmbed(context.Background(), CreateEmbedArgs{
		BoardID:  "board123",
		URL:      "https://youtube.com/watch?v=abc123",
		Mode:     "modal",
		X:        100,
		Y:        200,
		Width:    640,
		Height:   480,
		ParentID: "frame123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateEmbed_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateEmbed(context.Background(), UpdateEmbedArgs{
		BoardID: "board123",
		ItemID:  "embed123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	embCheckEq(t, "Message", result.Message, "No changes specified")
}

func TestUpdateEmbed_Validation(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	// Empty board_id
	_, err := client.UpdateEmbed(context.Background(), UpdateEmbedArgs{
		BoardID: "",
		ItemID:  "embed123",
	})
	if err == nil {
		t.Error("expected error for empty board_id")
	}

	// Empty item_id
	_, err = client.UpdateEmbed(context.Background(), UpdateEmbedArgs{
		BoardID: "board123",
		ItemID:  "",
	})
	if err == nil {
		t.Error("expected error for empty item_id")
	}
}

func TestUpdateEmbed_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		data := embSection(t, body, "data")
		embCheckField(t, data, "url", "https://youtube.com/watch?v=456")
		embCheckField(t, data, "mode", "modal")

		geom := embSection(t, body, "geometry")
		embCheckField(t, geom, "width", float64(800))
		embCheckField(t, geom, "height", float64(600))

		embServeJSON(w, map[string]interface{}{
			"id": "embed123",
			"data": map[string]interface{}{
				"url":         "https://youtube.com/watch?v=456",
				"providerUrl": "youtube.com",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	width := float64(800)
	height := float64(600)
	result, err := client.UpdateEmbed(context.Background(), UpdateEmbedArgs{
		BoardID: "board123",
		ItemID:  "embed123",
		URL:     strPtr("https://youtube.com/watch?v=456"),
		Mode:    strPtr("modal"),
		Width:   &width,
		Height:  &height,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	embCheckEq(t, "ID", result.ID, "embed123")
}
