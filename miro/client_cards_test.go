package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateCard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/cards" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "card123",
			"data": map[string]interface{}{
				"title":       "Task Card",
				"description": "Do something",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateCard(context.Background(), CreateCardArgs{
		BoardID:     "board123",
		Title:       "Task Card",
		Description: "Do something",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "card123" {
		t.Errorf("ID = %q, want 'card123'", result.ID)
	}
	if result.Title != "Task Card" {
		t.Errorf("Title = %q, want 'Task Card'", result.Title)
	}
}

func TestCreateCard_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    CreateCardArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    CreateCardArgs{Title: "Test"},
			errText: "board_id is required",
		},
		{
			name:    "empty title",
			args:    CreateCardArgs{BoardID: "board123"},
			errText: "title is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateCard(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestUpdateCard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/cards/") {
			t.Errorf("expected /cards/ in path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "card123",
			"data": map[string]interface{}{
				"title":       "Updated card",
				"description": "New description",
			},
			"dueDate": "2025-01-01",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateCard(context.Background(), UpdateCardArgs{
		BoardID:     "board123",
		ItemID:      "card123",
		Title:       strPtr("Updated card"),
		Description: strPtr("New description"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "card123" {
		t.Errorf("ID = %q, want 'card123'", result.ID)
	}
	if result.Title != "Updated card" {
		t.Errorf("Title = %q, want 'Updated card'", result.Title)
	}
}

func TestUpdateCard_NoChanges(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	result, err := client.UpdateCard(context.Background(), UpdateCardArgs{
		BoardID: "board123",
		ItemID:  "card123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No changes specified" {
		t.Errorf("Message = %q, want 'No changes specified'", result.Message)
	}
}

func TestUpdateCard_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		if data, ok := body["data"].(map[string]interface{}); !ok {
			t.Error("expected data in request body")
		} else {
			if data["title"] != "Updated Title" {
				t.Errorf("title = %v, want 'Updated Title'", data["title"])
			}
			if data["description"] != "New description" {
				t.Errorf("description = %v, want 'New description'", data["description"])
			}
			if data["dueDate"] != "2025-12-31" {
				t.Errorf("dueDate = %v, want '2025-12-31'", data["dueDate"])
			}
		}

		// Verify position section
		if pos, ok := body["position"].(map[string]interface{}); !ok {
			t.Error("expected position in request body")
		} else {
			if pos["x"] != float64(150) {
				t.Errorf("x = %v, want 150", pos["x"])
			}
			if pos["y"] != float64(250) {
				t.Errorf("y = %v, want 250", pos["y"])
			}
		}

		// Verify geometry section
		if geom, ok := body["geometry"].(map[string]interface{}); !ok {
			t.Error("expected geometry in request body")
		} else if geom["width"] != float64(350) {
			t.Errorf("width = %v, want 350", geom["width"])
		}

		// Verify parent section
		if parent, ok := body["parent"].(map[string]interface{}); !ok {
			t.Error("expected parent in request body")
		} else if parent["id"] != "frame-card" {
			t.Errorf("parent.id = %v, want 'frame-card'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "card123",
			"data": map[string]interface{}{
				"title":       "Updated Title",
				"description": "New description",
				"dueDate":     "2025-12-31",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	title := "Updated Title"
	description := "New description"
	dueDate := "2025-12-31"
	x := float64(150)
	y := float64(250)
	width := float64(350)
	parentID := "frame-card"

	result, err := client.UpdateCard(context.Background(), UpdateCardArgs{
		BoardID:     "board123",
		ItemID:      "card123",
		Title:       &title,
		Description: &description,
		DueDate:     &dueDate,
		X:           &x,
		Y:           &y,
		Width:       &width,
		ParentID:    &parentID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "card123" {
		t.Errorf("ID = %v, want 'card123'", result.ID)
	}
}

func TestUpdateCard_ClearDueDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify dueDate is null when empty string provided
		if data, ok := body["data"].(map[string]interface{}); !ok {
			t.Error("expected data in request body")
		} else if data["dueDate"] != nil {
			t.Errorf("dueDate = %v, want nil", data["dueDate"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "card123",
			"data": map[string]interface{}{"title": "Test", "description": "", "dueDate": ""},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	emptyDueDate := ""

	result, err := client.UpdateCard(context.Background(), UpdateCardArgs{
		BoardID: "board123",
		ItemID:  "card123",
		DueDate: &emptyDueDate,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "card123" {
		t.Errorf("ID = %v, want 'card123'", result.ID)
	}
}

func TestCreateCard_WithAllFields(t *testing.T) {
	// Tests CreateCard with all optional fields to improve coverage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify data section
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Error("expected 'data' field")
		}
		if data["title"] != "Test Card" {
			t.Errorf("title = %v, want 'Test Card'", data["title"])
		}
		if data["description"] != "Card description" {
			t.Errorf("description = %v, want 'Card description'", data["description"])
		}
		if data["dueDate"] != "2024-12-31" {
			t.Errorf("dueDate = %v, want '2024-12-31'", data["dueDate"])
		}

		// Verify position
		pos, ok := body["position"].(map[string]interface{})
		if !ok {
			t.Error("expected 'position' field")
		}
		if pos["x"] != float64(100) || pos["y"] != float64(200) {
			t.Errorf("position = %v, want x=100, y=200", pos)
		}

		// Verify geometry
		geom, ok := body["geometry"].(map[string]interface{})
		if !ok {
			t.Error("expected 'geometry' field")
		}
		if geom["width"] != float64(300) {
			t.Errorf("width = %v, want 300", geom["width"])
		}

		// Verify parent
		parent, ok := body["parent"].(map[string]interface{})
		if !ok {
			t.Error("expected 'parent' field")
		}
		if parent["id"] != "frame123" {
			t.Errorf("parent.id = %v, want 'frame123'", parent["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "card123",
			"type": "card",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateCard(context.Background(), CreateCardArgs{
		BoardID:     "board123",
		Title:       "Test Card",
		Description: "Card description",
		DueDate:     "2024-12-31",
		X:           100,
		Y:           200,
		Width:       300,
		ParentID:    "frame123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAppCard_WithNilFields(t *testing.T) {
	// Tests GetAppCard when some fields are nil
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard123",
			"type": "app_card",
			"data": map[string]interface{}{
				"title":       "Test App Card",
				"description": "",
				"fields":      nil,
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetAppCard(context.Background(), GetAppCardArgs{
		BoardID: "board123",
		ItemID:  "appcard123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "appcard123" {
		t.Errorf("ID = %v, want 'appcard123'", result.ID)
	}
}

func TestUpdateAppCard_WithStatusOnly(t *testing.T) {
	// Tests UpdateAppCard with only status field
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify status is in the data section
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Error("expected 'data' field")
		}
		if data["status"] != "connected" {
			t.Errorf("status = %v, want 'connected'", data["status"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard123",
			"type": "app_card",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.UpdateAppCard(context.Background(), UpdateAppCardArgs{
		BoardID: "board123",
		ItemID:  "appcard123",
		Status:  "connected",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateAppCard_WithAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		data, _ := body["data"].(map[string]interface{})
		if data["title"] != "Updated Title" {
			t.Errorf("title = %v, want 'Updated Title'", data["title"])
		}
		if data["description"] != "Updated Desc" {
			t.Errorf("description = %v, want 'Updated Desc'", data["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard123",
			"type": "app_card",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.UpdateAppCard(context.Background(), UpdateAppCardArgs{
		BoardID:     "board123",
		ItemID:      "appcard123",
		Title:       "Updated Title",
		Description: "Updated Desc",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAppCard_WithFields(t *testing.T) {
	// Tests GetAppCard with custom fields in response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard123",
			"type": "app_card",
			"data": map[string]interface{}{
				"title":       "Test Card",
				"description": "A test description",
				"status":      "connected",
				"fields": []map[string]interface{}{
					{"value": "Field 1", "fillColor": "#FF0000"},
					{"value": "Field 2", "fillColor": "#00FF00"},
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetAppCard(context.Background(), GetAppCardArgs{
		BoardID: "board123",
		ItemID:  "appcard123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "Test Card" {
		t.Errorf("Title = %v, want 'Test Card'", result.Title)
	}
}

func TestCreateAppCard_WithMultipleFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		data, _ := body["data"].(map[string]interface{})
		fields, _ := data["fields"].([]interface{})
		if len(fields) != 2 {
			t.Errorf("fields count = %d, want 2", len(fields))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "appcard456",
			"type": "app_card",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateAppCard(context.Background(), CreateAppCardArgs{
		BoardID: "board123",
		Title:   "New App Card",
		Fields: []AppCardField{
			{Value: "Field 1", FillColor: "#FF0000"},
			{Value: "Field 2", FillColor: "#00FF00"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
