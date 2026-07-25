package miro

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListBoards_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards") {
			t.Errorf("expected /boards path, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or incorrect Authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "board1", "name": "Design Sprint", "viewLink": "https://miro.com/board1"},
				{"id": "board2", "name": "Retro", "viewLink": "https://miro.com/board2"},
			},
			"size":  2,
			"total": 2,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListBoards(context.Background(), ListBoardsArgs{Query: "test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if result.Boards[0].Name != "Design Sprint" {
		t.Errorf("first board name = %q, want 'Design Sprint'", result.Boards[0].Name)
	}
}

func TestListBoards_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "unauthorized",
			"message": "Invalid access token",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListBoards(context.Background(), ListBoardsArgs{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Check it's an API error
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if apiErr.Code != "unauthorized" {
		t.Errorf("Code = %q, want 'unauthorized'", apiErr.Code)
	}
}

func TestGetBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boards/board123" {
			t.Errorf("expected /boards/board123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "board123",
			"name":        "Test Board",
			"description": "A test board",
			"viewLink":    "https://miro.com/board123",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetBoard(context.Background(), GetBoardArgs{BoardID: "board123"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "board123" {
		t.Errorf("ID = %q, want 'board123'", result.ID)
	}
	if result.Name != "Test Board" {
		t.Errorf("Name = %q, want 'Test Board'", result.Name)
	}
}

func TestGetBoard_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "not_found",
			"message": "Board not found",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetBoard(context.Background(), GetBoardArgs{BoardID: "nonexistent"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsNotFoundError(err) {
		t.Errorf("expected not found error, got: %v", err)
	}
}

func TestGetBoard_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.GetBoard(context.Background(), GetBoardArgs{BoardID: ""})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestGetBoard_Caching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "board123",
			"name": "Cached Board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	ctx := context.Background()

	// First call - should hit the server
	_, err := client.GetBoard(ctx, GetBoardArgs{BoardID: "board123"})
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Second call - should use cache
	_, err = client.GetBoard(ctx, GetBoardArgs{BoardID: "board123"})
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("server called %d times, want 1 (caching should prevent second call)", callCount)
	}
}

func TestCreateBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards" {
			t.Errorf("expected /boards, got %s", r.URL.Path)
		}

		// Verify request body
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "New Board" {
			t.Errorf("name = %v, want 'New Board'", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "new-board-id",
			"name":     "New Board",
			"viewLink": "https://miro.com/new-board-id",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateBoard(context.Background(), CreateBoardArgs{
		Name:        "New Board",
		Description: "Test description",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "new-board-id" {
		t.Errorf("ID = %q, want 'new-board-id'", result.ID)
	}
	if result.Name != "New Board" {
		t.Errorf("Name = %q, want 'New Board'", result.Name)
	}
}

func TestCreateBoard_EmptyName(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.CreateBoard(context.Background(), CreateBoardArgs{Name: ""})

	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("expected 'name is required' error, got: %v", err)
	}
}

func TestCreateBoard_WithAllOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify all optional fields
		if body["description"] != "Full description" {
			t.Errorf("description = %v, want 'Full description'", body["description"])
		}
		if body["teamId"] != "team123" {
			t.Errorf("teamId = %v, want 'team123'", body["teamId"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "full-board",
			"name":     "Full Board",
			"viewLink": "https://miro.com/full-board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateBoard(context.Background(), CreateBoardArgs{
		Name:        "Full Board",
		Description: "Full description",
		TeamID:      "team123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "full-board" {
		t.Errorf("ID = %q, want 'full-board'", result.ID)
	}
}

func TestCopyBoard_WithAllOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "copy_from=source-board") {
			t.Errorf("expected copy_from query param, got %s", r.URL.RawQuery)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify optional fields
		if body["name"] != "Copy Name" {
			t.Errorf("name = %v, want 'Copy Name'", body["name"])
		}
		if body["description"] != "Copy Desc" {
			t.Errorf("description = %v, want 'Copy Desc'", body["description"])
		}
		if body["teamId"] != "team456" {
			t.Errorf("teamId = %v, want 'team456'", body["teamId"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "copied-board",
			"name":     "Copy Name",
			"viewLink": "https://miro.com/copied-board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CopyBoard(context.Background(), CopyBoardArgs{
		BoardID:     "source-board",
		Name:        "Copy Name",
		Description: "Copy Desc",
		TeamID:      "team456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "copied-board" {
		t.Errorf("ID = %q, want 'copied-board'", result.ID)
	}
}

func TestDeleteBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123" {
			t.Errorf("expected /boards/board123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteBoard(context.Background(), DeleteBoardArgs{BoardID: "board123"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.BoardID != "board123" {
		t.Errorf("BoardID = %q, want 'board123'", result.BoardID)
	}
}

func TestDeleteBoard_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.DeleteBoard(context.Background(), DeleteBoardArgs{BoardID: ""})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestDeleteBoard_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  403,
			"message": "Access denied",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteBoard(context.Background(), DeleteBoardArgs{BoardID: "board123"})

	if err == nil {
		t.Fatal("expected error for API failure")
	}
	if result.Success {
		t.Error("Success should be false for API error")
	}
}

func TestGetBoardSummary_EmptyBoardID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.GetBoardSummary(context.Background(), GetBoardSummaryArgs{BoardID: ""})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestGetBoardSummary_GetBoardError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  404,
			"message": "Board not found",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetBoardSummary(context.Background(), GetBoardSummaryArgs{BoardID: "board123"})

	if err == nil {
		t.Fatal("expected error when board not found")
	}
	if !strings.Contains(err.Error(), "failed to get board") {
		t.Errorf("expected 'failed to get board' error, got: %v", err)
	}
}

func TestSearchBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "item1",
					"type": "sticky_note",
					"position": map[string]interface{}{
						"x": 100.0,
						"y": 200.0,
					},
					"data": map[string]interface{}{
						"content": "This is a test sticky note",
					},
				},
				{
					"id":   "item2",
					"type": "text",
					"data": map[string]interface{}{
						"content": "Another item without test",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.SearchBoard(context.Background(), SearchBoardArgs{
		BoardID: "board123",
		Query:   "test",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if result.Query != "test" {
		t.Errorf("Query = %q, want 'test'", result.Query)
	}
}

func TestSearchBoard_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    SearchBoardArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    SearchBoardArgs{Query: "test"},
			errText: "board_id is required",
		},
		{
			name:    "empty query",
			args:    SearchBoardArgs{BoardID: "board123"},
			errText: "query is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.SearchBoard(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestCopyBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/boards" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("copy_from") != "board123" {
			t.Errorf("expected copy_from=board123, got %s", r.URL.Query().Get("copy_from"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "newboard456",
			"name":     "Copied Board",
			"viewLink": "https://miro.com/newboard456",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CopyBoard(context.Background(), CopyBoardArgs{
		BoardID: "board123",
		Name:    "Copied Board",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "newboard456" {
		t.Errorf("ID = %q, want 'newboard456'", result.ID)
	}
	if result.Name != "Copied Board" {
		t.Errorf("Name = %q, want 'Copied Board'", result.Name)
	}
}

func TestCopyBoard_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.CopyBoard(context.Background(), CopyBoardArgs{Name: "Test"})
	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id is required") {
		t.Errorf("expected 'board_id is required' error, got: %v", err)
	}
}

func TestFindBoardByName_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query != "Design Sprint" {
			t.Errorf("query = %q, want 'Design Sprint'", query)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":       "board123",
					"name":     "Design Sprint",
					"viewLink": "https://miro.com/board123",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{
		Name: "Design Sprint",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "board123" {
		t.Errorf("ID = %q, want 'board123'", result.ID)
	}
	if result.Name != "Design Sprint" {
		t.Errorf("Name = %q, want 'Design Sprint'", result.Name)
	}
}

func TestFindBoardByName_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{
		Name: "Nonexistent Board",
	})

	if err == nil {
		t.Fatal("expected error for board not found")
	}
	if !strings.Contains(err.Error(), "no board found") {
		t.Errorf("expected 'no board found' error, got: %v", err)
	}
}

func TestFindBoardByName_EmptyName(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	_, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{
		Name: "",
	})

	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected 'required' error, got: %v", err)
	}
}

func TestFindBoardByName_StartsWithMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return boards where none is an exact match but one starts with the query
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":       "board2",
					"name":     "Something else",
					"viewLink": "https://miro.com/board2",
				},
				{
					"id":       "board1",
					"name":     "Sprint Planning Q1",
					"viewLink": "https://miro.com/board1",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{
		Name: "Sprint",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "board1" {
		t.Errorf("ID = %q, want 'board1'", result.ID)
	}
}

func TestFindBoardByName_ContainsMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return boards where none is an exact match or starts with, but one contains the query
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":       "board2",
					"name":     "Other board",
					"viewLink": "https://miro.com/board2",
				},
				{
					"id":       "board1",
					"name":     "Q1 Sprint Review",
					"viewLink": "https://miro.com/board1",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{
		Name: "Sprint",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "board1" {
		t.Errorf("ID = %q, want 'board1' (contains match)", result.ID)
	}
}

func TestFindBoardByName_Fallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return boards where none matches any criteria, should return first
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":       "board1",
					"name":     "Random Board ABC",
					"viewLink": "https://miro.com/board1",
				},
				{
					"id":       "board2",
					"name":     "Another Board XYZ",
					"viewLink": "https://miro.com/board2",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{
		Name: "Something completely different",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return first board as fallback
	if result.ID != "board1" {
		t.Errorf("ID = %q, want 'board1' (fallback)", result.ID)
	}
}

func TestGetBoardSummary_Success(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/items") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "item1", "type": "sticky_note"},
					{"id": "item2", "type": "sticky_note"},
					{"id": "item3", "type": "shape"},
				},
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "board123",
				"name":        "Test Board",
				"description": "A test board",
			})
		}
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetBoardSummary(context.Background(), GetBoardSummaryArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Test Board" {
		t.Errorf("Name = %q, want 'Test Board'", result.Name)
	}
	if result.TotalItems != 3 {
		t.Errorf("TotalItems = %d, want 3", result.TotalItems)
	}
}

func TestShareBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/members" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify request body
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		role := body["role"].(string)
		if role != "editor" {
			t.Errorf("expected role 'editor', got '%s'", role)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ShareBoard(context.Background(), ShareBoardArgs{
		BoardID: "board123",
		Email:   "user@example.com",
		Role:    "editor",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.Email != "user@example.com" {
		t.Errorf("Email = %q, want 'user@example.com'", result.Email)
	}
	if result.Role != "editor" {
		t.Errorf("Role = %q, want 'editor'", result.Role)
	}
}

func TestShareBoard_DefaultRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		role := body["role"].(string)
		if role != "viewer" {
			t.Errorf("expected default role 'viewer', got '%s'", role)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ShareBoard(context.Background(), ShareBoardArgs{
		BoardID: "board123",
		Email:   "user@example.com",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Role != "viewer" {
		t.Errorf("Role = %q, want 'viewer'", result.Role)
	}
}

func TestShareBoard_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())

	tests := []struct {
		name    string
		args    ShareBoardArgs
		errText string
	}{
		{
			name:    "empty board_id",
			args:    ShareBoardArgs{Email: "user@example.com"},
			errText: "board_id",
		},
		{
			name:    "empty email",
			args:    ShareBoardArgs{BoardID: "board123"},
			errText: "email is required",
		},
		{
			name:    "invalid role",
			args:    ShareBoardArgs{BoardID: "board123", Email: "user@example.com", Role: "admin"},
			errText: "invalid role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ShareBoard(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestGetBoardPicture_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "board123",
			"name": "Test Board",
			"picture": map[string]interface{}{
				"imageURL": "https://miro-media.com/board123/preview.png",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetBoardPicture(context.Background(), GetBoardPictureArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ImageURL != "https://miro-media.com/board123/preview.png" {
		t.Errorf("ImageURL = %q, want 'https://miro-media.com/board123/preview.png'", result.ImageURL)
	}
}

func TestGetBoardPicture_NoPicture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "board123",
			"name": "Test Board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetBoardPicture(context.Background(), GetBoardPictureArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ImageURL != "" {
		t.Errorf("expected empty ImageURL, got: %s", result.ImageURL)
	}
	if !strings.Contains(result.Message, "no picture") {
		t.Errorf("expected 'no picture' message, got: %s", result.Message)
	}
}

func TestUpdateBoard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123" {
			t.Errorf("expected /boards/board123, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "Updated Board Name" {
			t.Errorf("name = %v, want 'Updated Board Name'", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "board123",
			"name":        "Updated Board Name",
			"description": "Updated description",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateBoard(context.Background(), UpdateBoardArgs{
		BoardID:     "board123",
		Name:        "Updated Board Name",
		Description: "Updated description",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "board123" {
		t.Errorf("ID = %q, want 'board123'", result.ID)
	}
	if result.Name != "Updated Board Name" {
		t.Errorf("Name = %q, want 'Updated Board Name'", result.Name)
	}
}

func TestUpdateBoard_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")

	tests := []struct {
		name    string
		args    UpdateBoardArgs
		wantErr string
	}{
		{
			name:    "empty board ID",
			args:    UpdateBoardArgs{Name: "New Name"},
			wantErr: "board_id is required",
		},
		{
			name:    "no updates",
			args:    UpdateBoardArgs{BoardID: "board123"},
			wantErr: "at least one of name or description is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UpdateBoard(context.Background(), tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateToken_NoBoards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ValidateToken(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "validated" {
		t.Errorf("ID = %q, want 'validated' (default)", result.ID)
	}
}

func TestListTags_EmptyBoard(t *testing.T) {
	// Tests the empty board message branch
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListTags(context.Background(), ListTagsArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No tags on this board" {
		t.Errorf("expected 'No tags on this board', got: %s", result.Message)
	}
}

func TestListBoards_WithQueryAndTeamID(t *testing.T) {
	// Tests ListBoards with both query and team_id filters
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") != "design" {
			t.Errorf("query = %v, want 'design'", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("team_id") != "team123" {
			t.Errorf("team_id = %v, want 'team123'", r.URL.Query().Get("team_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "board1", "name": "Design Board"},
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListBoards(context.Background(), ListBoardsArgs{
		Query:  "design",
		TeamID: "team123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateBoard_WithDescription(t *testing.T) {
	// Tests CreateBoard with optional description
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "New Board" {
			t.Errorf("name = %v, want 'New Board'", body["name"])
		}
		if body["description"] != "Board description" {
			t.Errorf("description = %v, want 'Board description'", body["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "newboard123",
			"name": "New Board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CreateBoard(context.Background(), CreateBoardArgs{
		Name:        "New Board",
		Description: "Board description",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCopyBoard_WithNameAndDescription(t *testing.T) {
	// Tests CopyBoard with custom name and description
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "Copy of Board" {
			t.Errorf("name = %v, want 'Copy of Board'", body["name"])
		}
		if body["description"] != "Copied board" {
			t.Errorf("description = %v, want 'Copied board'", body["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "copyboard123",
			"name": "Copy of Board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CopyBoard(context.Background(), CopyBoardArgs{
		BoardID:     "board123",
		Name:        "Copy of Board",
		Description: "Copied board",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateBoard_WithTeamID(t *testing.T) {
	// Tests CreateBoard with teamId in request body (Miro uses camelCase)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "New Board" {
			t.Errorf("name = %v, want 'New Board'", body["name"])
		}
		if body["teamId"] != "team456" {
			t.Errorf("teamId = %v, want 'team456'", body["teamId"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "newboard123",
			"name": "New Board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateBoard(context.Background(), CreateBoardArgs{
		Name:   "New Board",
		TeamID: "team456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "newboard123" {
		t.Errorf("ID = %v, want 'newboard123'", result.ID)
	}
}

func TestGetBoardSummary_WithManyItemTypes(t *testing.T) {
	// Tests GetBoardSummary with various item types
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/items") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "1", "type": "sticky_note"},
					{"id": "2", "type": "shape"},
					{"id": "3", "type": "text"},
					{"id": "4", "type": "connector"},
					{"id": "5", "type": "frame"},
					{"id": "6", "type": "card"},
					{"id": "7", "type": "image"},
					{"id": "8", "type": "document"},
					{"id": "9", "type": "embed"},
					{"id": "10", "type": "app_card"},
				},
			})
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "board123",
				"name":        "Test Board",
				"description": "A test board",
			})
		}
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetBoardSummary(context.Background(), GetBoardSummaryArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalItems != 10 {
		t.Errorf("TotalItems = %v, want 10", result.TotalItems)
	}
}

func TestCopyBoard_WithDescription(t *testing.T) {
	// Tests CopyBoard with optional description
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["description"] != "Copied board description" {
			t.Errorf("description = %v, want 'Copied board description'", body["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "copied123",
			"name": "Board Copy",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CopyBoard(context.Background(), CopyBoardArgs{
		BoardID:     "original123",
		Name:        "Board Copy",
		Description: "Copied board description",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "copied123" {
		t.Errorf("ID = %v, want 'copied123'", result.ID)
	}
}

func TestUpdateBoard_WithBothNameAndDescription(t *testing.T) {
	// Tests UpdateBoard with both name and description
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "Updated Name" {
			t.Errorf("name = %v, want 'Updated Name'", body["name"])
		}
		if body["description"] != "Updated Description" {
			t.Errorf("description = %v, want 'Updated Description'", body["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "board123",
			"name":        "Updated Name",
			"description": "Updated Description",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateBoard(context.Background(), UpdateBoardArgs{
		BoardID:     "board123",
		Name:        "Updated Name",
		Description: "Updated Description",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated Name" {
		t.Errorf("Name = %v, want 'Updated Name'", result.Name)
	}
}

func TestListBoards_WithOffset(t *testing.T) {
	// Tests ListBoards with offset for pagination
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("offset") != "20" {
			t.Errorf("offset = %v, want '20'", r.URL.Query().Get("offset"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListBoards(context.Background(), ListBoardsArgs{
		Offset: "20",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateBoard_EmptyNameValidation(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	_, err := client.CreateBoard(context.Background(), CreateBoardArgs{
		Name: "",
	})

	if err == nil {
		t.Error("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should mention 'name': %v", err)
	}
}

func TestCopyBoard_EmptyBoardID(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	_, err := client.CopyBoard(context.Background(), CopyBoardArgs{
		BoardID: "",
	})

	if err == nil {
		t.Error("expected error for empty board_id")
	}
	if !strings.Contains(err.Error(), "board_id") {
		t.Errorf("error should mention 'board_id': %v", err)
	}
}

func TestCopyBoard_WithTeamID(t *testing.T) {
	// Tests CopyBoard with teamId (Miro uses camelCase)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["teamId"] != "team789" {
			t.Errorf("teamId = %v, want 'team789'", body["teamId"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "copied456",
			"name": "Copied Board",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.CopyBoard(context.Background(), CopyBoardArgs{
		BoardID: "original123",
		TeamID:  "team789",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateBoard_EmptyBoardID(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	_, err := client.UpdateBoard(context.Background(), UpdateBoardArgs{
		BoardID: "",
		Name:    "New Name",
	})

	if err == nil {
		t.Error("expected error for empty board_id")
	}
}

func TestUpdateBoard_NoChanges(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	_, err := client.UpdateBoard(context.Background(), UpdateBoardArgs{
		BoardID: "board123",
		// No name or description provided
	})

	if err == nil {
		t.Error("expected error when no changes specified")
	}
}
