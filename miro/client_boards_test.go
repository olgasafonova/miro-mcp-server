package miro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// Test server helpers
// =============================================================================

// boardServerSpec describes the fake Miro API behavior for one board test:
// the expected request line, query parameters, auth header, body fields, and
// the response to send back.
type boardServerSpec struct {
	wantMethod string
	wantPath   string
	wantAuth   string
	wantQuery  map[string]string
	wantBody   map[string]interface{}
	status     int
	response   map[string]interface{}
}

// newBoardClient starts a fake Miro API described by spec and returns a client
// pointed at it.
func newBoardClient(t *testing.T, spec boardServerSpec) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spec.assertRequest(t, r)
		writeBoardJSON(w, spec.status, spec.response)
	}))
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

func (spec boardServerSpec) assertRequest(t *testing.T, r *http.Request) {
	t.Helper()
	spec.assertRequestLine(t, r)
	spec.assertQueryAndAuth(t, r)
	spec.assertRequestBody(t, r)
}

func (spec boardServerSpec) assertRequestLine(t *testing.T, r *http.Request) {
	t.Helper()
	if spec.wantMethod != "" && r.Method != spec.wantMethod {
		t.Errorf("expected %s, got %s", spec.wantMethod, r.Method)
	}
	if spec.wantPath != "" && r.URL.Path != spec.wantPath {
		t.Errorf("expected path %s, got %s", spec.wantPath, r.URL.Path)
	}
}

func (spec boardServerSpec) assertQueryAndAuth(t *testing.T, r *http.Request) {
	t.Helper()
	if spec.wantAuth != "" && r.Header.Get("Authorization") != spec.wantAuth {
		t.Errorf("missing or incorrect Authorization header")
	}
	for key, want := range spec.wantQuery {
		if got := r.URL.Query().Get(key); got != want {
			t.Errorf("query %s = %q, want %q", key, got, want)
		}
	}
}

func (spec boardServerSpec) assertRequestBody(t *testing.T, r *http.Request) {
	t.Helper()
	if spec.wantBody == nil {
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("failed to decode request: %v", err)
		return
	}
	for key, want := range spec.wantBody {
		if body[key] != want {
			t.Errorf("%s = %v, want %v", key, body[key], want)
		}
	}
}

func writeBoardJSON(w http.ResponseWriter, status int, response map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if response == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		panic(err)
	}
}

// boardsData wraps a set of board objects in the list response envelope.
func boardsData(boards ...map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"data": boards}
}

// typedItems builds one board item per type, with sequential IDs.
func typedItems(types ...string) []map[string]interface{} {
	items := make([]map[string]interface{}, len(types))
	for i, itemType := range types {
		items[i] = map[string]interface{}{"id": fmt.Sprintf("item%d", i+1), "type": itemType}
	}
	return items
}

// newBoardSummaryClient starts a fake API serving a fixed board plus the given
// items on the /items endpoint, for GetBoardSummary tests.
func newBoardSummaryClient(t *testing.T, items []map[string]interface{}) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/items") {
			writeBoardJSON(w, 0, map[string]interface{}{"data": items})
			return
		}
		writeBoardJSON(w, 0, map[string]interface{}{
			"id":          "board123",
			"name":        "Test Board",
			"description": "A test board",
		})
	}))
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

// =============================================================================
// List / Get / Create / Copy / Delete / Update
// =============================================================================

func TestListBoards_Success(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{
		wantMethod: http.MethodGet,
		wantPath:   "/boards",
		wantAuth:   "Bearer test-token",
		response: map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "board1", "name": "Design Sprint", "viewLink": "https://miro.com/board1"},
				{"id": "board2", "name": "Retro", "viewLink": "https://miro.com/board2"},
			},
			"size":  2,
			"total": 2,
		},
	})

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
	client := newBoardClient(t, boardServerSpec{
		status: http.StatusUnauthorized,
		response: map[string]interface{}{
			"code":    "unauthorized",
			"message": "Invalid access token",
		},
	})

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

func TestListBoards_WithQueryAndTeamID(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{
		wantQuery: map[string]string{"query": "design", "team_id": "team123"},
		response:  boardsData(map[string]interface{}{"id": "board1", "name": "Design Board"}),
	})

	_, err := client.ListBoards(context.Background(), ListBoardsArgs{
		Query:  "design",
		TeamID: "team123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListBoards_WithOffset(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{
		wantQuery: map[string]string{"offset": "20"},
		response:  boardsData(),
	})

	_, err := client.ListBoards(context.Background(), ListBoardsArgs{Offset: "20"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetBoard_Success(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{
		wantPath: "/boards/board123",
		response: map[string]interface{}{
			"id":          "board123",
			"name":        "Test Board",
			"description": "A test board",
			"viewLink":    "https://miro.com/board123",
		},
	})

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
	client := newBoardClient(t, boardServerSpec{
		status: http.StatusNotFound,
		response: map[string]interface{}{
			"code":    "not_found",
			"message": "Board not found",
		},
	})

	_, err := client.GetBoard(context.Background(), GetBoardArgs{BoardID: "nonexistent"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsNotFoundError(err) {
		t.Errorf("expected not found error, got: %v", err)
	}
}

func TestGetBoard_Caching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		writeBoardJSON(w, 0, map[string]interface{}{
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
	client := newBoardClient(t, boardServerSpec{
		wantMethod: http.MethodPost,
		wantPath:   "/boards",
		wantBody:   map[string]interface{}{"name": "New Board"},
		status:     http.StatusCreated,
		response: map[string]interface{}{
			"id":       "new-board-id",
			"name":     "New Board",
			"viewLink": "https://miro.com/new-board-id",
		},
	})

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

func TestCopyBoard_Success(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{
		wantMethod: http.MethodPut,
		wantPath:   "/boards",
		wantQuery:  map[string]string{"copy_from": "board123"},
		status:     http.StatusCreated,
		response: map[string]interface{}{
			"id":       "newboard456",
			"name":     "Copied Board",
			"viewLink": "https://miro.com/newboard456",
		},
	})

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

func TestDeleteBoard_Success(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{
		wantMethod: http.MethodDelete,
		wantPath:   "/boards/board123",
		status:     http.StatusNoContent,
	})

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

func TestDeleteBoard_APIError(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{
		status: http.StatusForbidden,
		response: map[string]interface{}{
			"status":  403,
			"message": "Access denied",
		},
	})

	result, err := client.DeleteBoard(context.Background(), DeleteBoardArgs{BoardID: "board123"})

	if err == nil {
		t.Fatal("expected error for API failure")
	}
	if result.Success {
		t.Error("Success should be false for API error")
	}
}

func TestUpdateBoard_Success(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{
		wantMethod: http.MethodPatch,
		wantPath:   "/boards/board123",
		wantBody:   map[string]interface{}{"name": "Updated Board Name"},
		response: map[string]interface{}{
			"id":          "board123",
			"name":        "Updated Board Name",
			"description": "Updated description",
		},
	})

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

// boardBodyCase is one scenario for TestBoards_RequestBodyFields: the server
// spec, the client call to make, and the identity field the result must carry.
type boardBodyCase struct {
	name string
	spec boardServerSpec
	call func(*Client) (string, error)
	want string
}

func createBoardBodyCases(ctx context.Context) []boardBodyCase {
	return []boardBodyCase{
		{
			name: "create with all options",
			spec: boardServerSpec{
				wantBody: map[string]interface{}{"description": "Full description", "teamId": "team123"},
				status:   http.StatusCreated,
				response: map[string]interface{}{"id": "full-board", "name": "Full Board", "viewLink": "https://miro.com/full-board"},
			},
			call: func(c *Client) (string, error) {
				result, err := c.CreateBoard(ctx, CreateBoardArgs{Name: "Full Board", Description: "Full description", TeamID: "team123"})
				return result.ID, err
			},
			want: "full-board",
		},
		{
			name: "create with description",
			spec: boardServerSpec{
				wantBody: map[string]interface{}{"name": "New Board", "description": "Board description"},
				response: map[string]interface{}{"id": "newboard123", "name": "New Board"},
			},
			call: func(c *Client) (string, error) {
				result, err := c.CreateBoard(ctx, CreateBoardArgs{Name: "New Board", Description: "Board description"})
				return result.ID, err
			},
			want: "newboard123",
		},
		{
			name: "create with team ID",
			spec: boardServerSpec{
				wantBody: map[string]interface{}{"name": "New Board", "teamId": "team456"},
				response: map[string]interface{}{"id": "newboard123", "name": "New Board"},
			},
			call: func(c *Client) (string, error) {
				result, err := c.CreateBoard(ctx, CreateBoardArgs{Name: "New Board", TeamID: "team456"})
				return result.ID, err
			},
			want: "newboard123",
		},
	}
}

func copyBoardBodyCases(ctx context.Context) []boardBodyCase {
	return []boardBodyCase{
		{
			name: "copy with all options",
			spec: boardServerSpec{
				wantMethod: http.MethodPut,
				wantQuery:  map[string]string{"copy_from": "source-board"},
				wantBody:   map[string]interface{}{"name": "Copy Name", "description": "Copy Desc", "teamId": "team456"},
				response:   map[string]interface{}{"id": "copied-board", "name": "Copy Name", "viewLink": "https://miro.com/copied-board"},
			},
			call: func(c *Client) (string, error) {
				result, err := c.CopyBoard(ctx, CopyBoardArgs{BoardID: "source-board", Name: "Copy Name", Description: "Copy Desc", TeamID: "team456"})
				return result.ID, err
			},
			want: "copied-board",
		},
		{
			name: "copy with name and description",
			spec: boardServerSpec{
				wantBody: map[string]interface{}{"name": "Copy of Board", "description": "Copied board"},
				response: map[string]interface{}{"id": "copyboard123", "name": "Copy of Board"},
			},
			call: func(c *Client) (string, error) {
				result, err := c.CopyBoard(ctx, CopyBoardArgs{BoardID: "board123", Name: "Copy of Board", Description: "Copied board"})
				return result.ID, err
			},
			want: "copyboard123",
		},
		{
			name: "copy with description",
			spec: boardServerSpec{
				wantBody: map[string]interface{}{"description": "Copied board description"},
				response: map[string]interface{}{"id": "copied123", "name": "Board Copy"},
			},
			call: func(c *Client) (string, error) {
				result, err := c.CopyBoard(ctx, CopyBoardArgs{BoardID: "original123", Name: "Board Copy", Description: "Copied board description"})
				return result.ID, err
			},
			want: "copied123",
		},
		{
			name: "copy with team ID",
			spec: boardServerSpec{
				wantBody: map[string]interface{}{"teamId": "team789"},
				response: map[string]interface{}{"id": "copied456", "name": "Copied Board"},
			},
			call: func(c *Client) (string, error) {
				result, err := c.CopyBoard(ctx, CopyBoardArgs{BoardID: "original123", TeamID: "team789"})
				return result.ID, err
			},
			want: "copied456",
		},
	}
}

func updateBoardBodyCases(ctx context.Context) []boardBodyCase {
	return []boardBodyCase{
		{
			name: "update with both name and description",
			spec: boardServerSpec{
				wantBody: map[string]interface{}{"name": "Updated Name", "description": "Updated Description"},
				response: map[string]interface{}{"id": "board123", "name": "Updated Name", "description": "Updated Description"},
			},
			call: func(c *Client) (string, error) {
				result, err := c.UpdateBoard(ctx, UpdateBoardArgs{BoardID: "board123", Name: "Updated Name", Description: "Updated Description"})
				return result.Name, err
			},
			want: "Updated Name",
		},
	}
}

// TestBoards_RequestBodyFields verifies that optional args for create, copy,
// and update land in the request body, and that the returned identity fields
// round-trip.
func TestBoards_RequestBodyFields(t *testing.T) {
	ctx := context.Background()
	var tests []boardBodyCase
	tests = append(tests, createBoardBodyCases(ctx)...)
	tests = append(tests, copyBoardBodyCases(ctx)...)
	tests = append(tests, updateBoardBodyCases(ctx)...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newBoardClient(t, tt.spec)
			got, err := tt.call(client)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("result = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestBoards_ValidationErrors covers every empty-argument rejection across the
// board operations. No API call is made in any of these scenarios.
func TestBoards_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	ctx := context.Background()

	tests := []struct {
		name    string
		wantErr string
		call    func() error
	}{
		{"get board empty board_id", "board_id is required", func() error { _, err := client.GetBoard(ctx, GetBoardArgs{BoardID: ""}); return err }},
		{"create board empty name", "name is required", func() error { _, err := client.CreateBoard(ctx, CreateBoardArgs{Name: ""}); return err }},
		{"create board empty name mentions name", "name", func() error { _, err := client.CreateBoard(ctx, CreateBoardArgs{Name: ""}); return err }},
		{"delete board empty board_id", "board_id is required", func() error { _, err := client.DeleteBoard(ctx, DeleteBoardArgs{BoardID: ""}); return err }},
		{"board summary empty board_id", "board_id is required", func() error { _, err := client.GetBoardSummary(ctx, GetBoardSummaryArgs{BoardID: ""}); return err }},
		{"search empty board_id", "board_id is required", func() error { _, err := client.SearchBoard(ctx, SearchBoardArgs{Query: "test"}); return err }},
		{"search empty query", "query is required", func() error { _, err := client.SearchBoard(ctx, SearchBoardArgs{BoardID: "board123"}); return err }},
		{"copy empty board_id", "board_id is required", func() error { _, err := client.CopyBoard(ctx, CopyBoardArgs{Name: "Test"}); return err }},
		{"copy empty board_id mentions board_id", "board_id", func() error { _, err := client.CopyBoard(ctx, CopyBoardArgs{BoardID: ""}); return err }},
		{"find by name empty name", "required", func() error { _, err := client.FindBoardByNameTool(ctx, FindBoardByNameArgs{Name: ""}); return err }},
		{"share empty board_id", "board_id", func() error { _, err := client.ShareBoard(ctx, ShareBoardArgs{Email: "user@example.com"}); return err }},
		{"share empty email", "email is required", func() error { _, err := client.ShareBoard(ctx, ShareBoardArgs{BoardID: "board123"}); return err }},
		{"share invalid role", "invalid role", func() error {
			_, err := client.ShareBoard(ctx, ShareBoardArgs{BoardID: "board123", Email: "user@example.com", Role: "admin"})
			return err
		}},
		{"update empty board_id", "board_id is required", func() error { _, err := client.UpdateBoard(ctx, UpdateBoardArgs{Name: "New Name"}); return err }},
		{"update no changes", "at least one of name or description is required", func() error { _, err := client.UpdateBoard(ctx, UpdateBoardArgs{BoardID: "board123"}); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// =============================================================================
// Search / Find / Summary / Share / Picture / Misc
// =============================================================================

func TestSearchBoard_Success(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{
		response: map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":       "item1",
					"type":     "sticky_note",
					"position": map[string]interface{}{"x": 100.0, "y": 200.0},
					"data":     map[string]interface{}{"content": "This is a test sticky note"},
				},
				{
					"id":   "item2",
					"type": "text",
					"data": map[string]interface{}{"content": "Another item without test"},
				},
			},
		},
	})

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

func TestFindBoardByName_Success(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{
		wantQuery: map[string]string{"query": "Design Sprint"},
		response: boardsData(map[string]interface{}{
			"id":       "board123",
			"name":     "Design Sprint",
			"viewLink": "https://miro.com/board123",
		}),
	})

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
	client := newBoardClient(t, boardServerSpec{response: boardsData()})

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

// TestFindBoardByName_MatchPriority verifies the non-exact match tiers:
// starts-with beats contains, and the first board is the final fallback.
func TestFindBoardByName_MatchPriority(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		boards []map[string]interface{}
		wantID string
	}{
		{
			name:  "starts-with match",
			query: "Sprint",
			boards: []map[string]interface{}{
				{"id": "board2", "name": "Something else", "viewLink": "https://miro.com/board2"},
				{"id": "board1", "name": "Sprint Planning Q1", "viewLink": "https://miro.com/board1"},
			},
			wantID: "board1",
		},
		{
			name:  "contains match",
			query: "Sprint",
			boards: []map[string]interface{}{
				{"id": "board2", "name": "Other board", "viewLink": "https://miro.com/board2"},
				{"id": "board1", "name": "Q1 Sprint Review", "viewLink": "https://miro.com/board1"},
			},
			wantID: "board1",
		},
		{
			name:  "fallback to first board",
			query: "Something completely different",
			boards: []map[string]interface{}{
				{"id": "board1", "name": "Random Board ABC", "viewLink": "https://miro.com/board1"},
				{"id": "board2", "name": "Another Board XYZ", "viewLink": "https://miro.com/board2"},
			},
			wantID: "board1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newBoardClient(t, boardServerSpec{response: boardsData(tt.boards...)})

			result, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{Name: tt.query})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", result.ID, tt.wantID)
			}
		})
	}
}

func TestGetBoardSummary_Success(t *testing.T) {
	client := newBoardSummaryClient(t, typedItems("sticky_note", "sticky_note", "shape"))

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

func TestGetBoardSummary_WithManyItemTypes(t *testing.T) {
	client := newBoardSummaryClient(t, typedItems(
		"sticky_note", "shape", "text", "connector", "frame",
		"card", "image", "document", "embed", "app_card",
	))

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

func TestGetBoardSummary_GetBoardError(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{
		status: http.StatusNotFound,
		response: map[string]interface{}{
			"status":  404,
			"message": "Board not found",
		},
	})

	_, err := client.GetBoardSummary(context.Background(), GetBoardSummaryArgs{BoardID: "board123"})

	if err == nil {
		t.Fatal("expected error when board not found")
	}
	if !strings.Contains(err.Error(), "failed to get board") {
		t.Errorf("expected 'failed to get board' error, got: %v", err)
	}
}

func TestShareBoard_Success(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{
		wantMethod: http.MethodPost,
		wantPath:   "/boards/board123/members",
		wantBody:   map[string]interface{}{"role": "editor"},
		status:     http.StatusNoContent,
	})

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
	client := newBoardClient(t, boardServerSpec{
		wantBody: map[string]interface{}{"role": "viewer"},
		status:   http.StatusNoContent,
	})

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

func TestGetBoardPicture_Success(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{
		response: map[string]interface{}{
			"id":   "board123",
			"name": "Test Board",
			"picture": map[string]interface{}{
				"imageURL": "https://miro-media.com/board123/preview.png",
			},
		},
	})

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
	client := newBoardClient(t, boardServerSpec{
		response: map[string]interface{}{
			"id":   "board123",
			"name": "Test Board",
		},
	})

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

func TestValidateToken_NoBoards(t *testing.T) {
	client := newBoardClient(t, boardServerSpec{response: boardsData()})

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
	client := newBoardClient(t, boardServerSpec{
		response: map[string]interface{}{"data": []interface{}{}},
	})

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
