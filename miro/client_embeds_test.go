package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateEmbed_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/embeds") {
			t.Errorf("expected embeds path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "embed123",
			"data": map[string]interface{}{
				"url": "https://youtube.com/watch?v=test",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateEmbed(context.Background(), CreateEmbedArgs{
		BoardID: "board123",
		URL:     "https://youtube.com/watch?v=test",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "embed123" {
		t.Errorf("ID = %q, want 'embed123'", result.ID)
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

		// Verify data section
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Error("expected 'data' field")
		}
		if data["url"] != "https://youtube.com/watch?v=abc123" {
			t.Errorf("url = %v, want 'https://youtube.com/watch?v=abc123'", data["url"])
		}
		if data["mode"] != "modal" {
			t.Errorf("mode = %v, want 'modal'", data["mode"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
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

func TestUpdateEmbed_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/embeds/") {
			t.Errorf("expected /embeds/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "embed123",
			"data": map[string]interface{}{
				"url":         "https://youtube.com/watch?v=123",
				"providerUrl": "youtube.com",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateEmbed(context.Background(), UpdateEmbedArgs{
		BoardID: "board123",
		ItemID:  "embed123",
		URL:     strPtr("https://youtube.com/watch?v=123"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "embed123" {
		t.Errorf("ID = %q, want 'embed123'", result.ID)
	}
	if result.URL != "https://youtube.com/watch?v=123" {
		t.Errorf("URL = %q, want 'https://youtube.com/watch?v=123'", result.URL)
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
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
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

		// Verify data section
		if data, ok := body["data"].(map[string]interface{}); !ok {
			t.Error("expected data in request body")
		} else {
			if data["url"] != "https://youtube.com/watch?v=456" {
				t.Errorf("url = %v, want 'https://youtube.com/watch?v=456'", data["url"])
			}
			if data["mode"] != "modal" {
				t.Errorf("mode = %v, want 'modal'", data["mode"])
			}
		}

		// Verify geometry section
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else {
			if geom["width"] != float64(800) {
				t.Errorf("width = %v, want 800", geom["width"])
			}
			if geom["height"] != float64(600) {
				t.Errorf("height = %v, want 600", geom["height"])
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
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
	if result.ID != "embed123" {
		t.Errorf("ID = %q, want 'embed123'", result.ID)
	}
}
