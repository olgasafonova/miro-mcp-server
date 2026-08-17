package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCard_Success covers the create and update happy paths: the request line
// must match, and the response must map onto the result struct.
func TestCard_Success(t *testing.T) {
	tests := []struct {
		name        string
		wantRequest string
		status      int
		response    map[string]interface{}
		call        func(*Client) (id, title string, err error)
		wantID      string
		wantTitle   string
	}{
		{
			name:        "create card",
			wantRequest: "POST /boards/board123/cards",
			status:      http.StatusCreated,
			response: map[string]interface{}{
				"id": "card123",
				"data": map[string]interface{}{
					"title":       "Task Card",
					"description": "Do something",
				},
			},
			call: func(client *Client) (string, string, error) {
				result, err := client.CreateCard(context.Background(), CreateCardArgs{
					BoardID:     "board123",
					Title:       "Task Card",
					Description: "Do something",
				})
				if err != nil {
					return "", "", err
				}
				return result.ID, result.Title, nil
			},
			wantID:    "card123",
			wantTitle: "Task Card",
		},
		{
			name:        "update card",
			wantRequest: "PATCH /boards/board123/cards/card123",
			status:      http.StatusOK,
			response: map[string]interface{}{
				"id": "card123",
				"data": map[string]interface{}{
					"title":       "Updated card",
					"description": "New description",
				},
				"dueDate": "2025-01-01",
			},
			call: func(client *Client) (string, string, error) {
				result, err := client.UpdateCard(context.Background(), UpdateCardArgs{
					BoardID:     "board123",
					ItemID:      "card123",
					Title:       strPtr("Updated card"),
					Description: strPtr("New description"),
				})
				if err != nil {
					return "", "", err
				}
				return result.ID, result.Title, nil
			},
			wantID:    "card123",
			wantTitle: "Updated card",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newStickyVerifyServer(t, tt.wantRequest, tt.status, tt.response)
			defer server.Close()

			id, title, err := tt.call(newTestClientWithServer(server.URL))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			requireStringField(t, "ID", id, tt.wantID)
			requireStringField(t, "Title", title, tt.wantTitle)
		})
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
		_ = json.NewDecoder(r.Body).Decode(&body)

		requireBodyField(t, body, "data.title", "Updated Title")
		requireBodyField(t, body, "data.description", "New description")
		requireBodyField(t, body, "data.dueDate", "2025-12-31")
		requireBodyField(t, body, "position.x", float64(150))
		requireBodyField(t, body, "position.y", float64(250))
		requireBodyField(t, body, "geometry.width", float64(350))
		requireBodyField(t, body, "parent.id", "frame-card")

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
		_ = json.NewDecoder(r.Body).Decode(&body)

		requireBodyField(t, body, "data.title", "Test Card")
		requireBodyField(t, body, "data.description", "Card description")
		requireBodyField(t, body, "data.dueDate", "2024-12-31")
		requireBodyField(t, body, "position.x", float64(100))
		requireBodyField(t, body, "position.y", float64(200))
		requireBodyField(t, body, "geometry.width", float64(300))
		requireBodyField(t, body, "parent.id", "frame123")

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

// TestGetAppCard_ResponseParsing covers app card responses both with nil
// optional fields and with populated custom fields.
func TestGetAppCard_ResponseParsing(t *testing.T) {
	tests := []struct {
		name      string
		response  map[string]interface{}
		wantID    string
		wantTitle string
	}{
		{
			name: "nil optional fields",
			response: map[string]interface{}{
				"id":   "appcard123",
				"type": "app_card",
				"data": map[string]interface{}{
					"title":       "Test App Card",
					"description": "",
					"fields":      nil,
				},
			},
			wantID:    "appcard123",
			wantTitle: "Test App Card",
		},
		{
			name: "custom fields in response",
			response: map[string]interface{}{
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
			},
			wantID:    "appcard123",
			wantTitle: "Test Card",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.response)
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
			requireStringField(t, "ID", result.ID, tt.wantID)
			requireStringField(t, "Title", result.Title, tt.wantTitle)
		})
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
