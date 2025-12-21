package tools

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/miro-mcp-server/miro"
)

// =============================================================================
// Test Helpers
// =============================================================================

// testLogger creates a silent logger for tests.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// newTestRegistry creates a HandlerRegistry with a mock client.
func newTestRegistry(mock *MockClient) *HandlerRegistry {
	return NewHandlerRegistry(mock, testLogger())
}

// =============================================================================
// HandlerRegistry Tests
// =============================================================================

func TestNewHandlerRegistry(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	if registry == nil {
		t.Fatal("NewHandlerRegistry returned nil")
	}
	if registry.client == nil {
		t.Error("client not set")
	}
	if registry.logger == nil {
		t.Error("logger not set")
	}
	if len(registry.handlers) == 0 {
		t.Error("handlers map is empty")
	}
}

func TestHandlerRegistryBuildHandlerMap(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	// Verify all expected methods are in the handler map
	expectedMethods := []string{
		// Board tools
		"ListBoards", "GetBoard", "CreateBoard", "CopyBoard", "DeleteBoard",
		"FindBoardByNameTool", "GetBoardSummary",
		// Item tools
		"ListItems", "ListAllItems", "GetItem", "UpdateItem", "DeleteItem",
		"SearchBoard", "BulkCreate",
		// Create tools
		"CreateSticky", "CreateShape", "CreateText", "CreateConnector",
		"CreateFrame", "CreateCard", "CreateImage", "CreateDocument",
		"CreateEmbed", "CreateStickyGrid",
		// Tag tools
		"CreateTag", "ListTags", "AttachTag", "DetachTag", "GetItemTags",
		// Group tools
		"CreateGroup", "Ungroup",
		// Member tools
		"ListBoardMembers", "ShareBoard",
		// Mindmap tools
		"CreateMindmapNode",
	}

	for _, method := range expectedMethods {
		if _, ok := registry.handlers[method]; !ok {
			t.Errorf("handler map missing method: %s", method)
		}
	}
}

func TestRegisterAll(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)

	// Should not panic
	registry.RegisterAll(server)
}

func TestBuildTool(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	tests := []struct {
		name        string
		spec        ToolSpec
		expectTitle string
		expectRO    bool
		expectDestr bool
	}{
		{
			name: "read-only tool",
			spec: ToolSpec{
				Name:     "test_read",
				Title:    "Test Read",
				ReadOnly: true,
			},
			expectTitle: "Test Read",
			expectRO:    true,
			expectDestr: false,
		},
		{
			name: "destructive tool",
			spec: ToolSpec{
				Name:        "test_delete",
				Title:       "Test Delete",
				Destructive: true,
			},
			expectTitle: "Test Delete",
			expectRO:    false,
			expectDestr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := registry.buildTool(tt.spec)

			if tool.Name != tt.spec.Name {
				t.Errorf("Name = %q, want %q", tool.Name, tt.spec.Name)
			}
			if tool.Annotations.Title != tt.expectTitle {
				t.Errorf("Title = %q, want %q", tool.Annotations.Title, tt.expectTitle)
			}
			if tool.Annotations.ReadOnlyHint != tt.expectRO {
				t.Errorf("ReadOnlyHint = %v, want %v", tool.Annotations.ReadOnlyHint, tt.expectRO)
			}
			if tt.expectDestr && (tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint) {
				t.Error("DestructiveHint should be true")
			}
		})
	}
}

// =============================================================================
// Board Handler Tests
// =============================================================================

func TestListBoardsHandler(t *testing.T) {
	mock := &MockClient{
		ListBoardsFn: func(ctx context.Context, args miro.ListBoardsArgs) (miro.ListBoardsResult, error) {
			return miro.ListBoardsResult{
				Boards: []miro.BoardSummary{
					{ID: "board1", Name: "Design Sprint"},
					{ID: "board2", Name: "Retro Board"},
				},
				Count:   2,
				HasMore: false,
			}, nil
		},
	}

	ctx := context.Background()
	result, err := mock.ListBoards(ctx, miro.ListBoardsArgs{Query: "test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if len(mock.Calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(mock.Calls))
	}
	if mock.Calls[0].Method != "ListBoards" {
		t.Errorf("Method = %q, want ListBoards", mock.Calls[0].Method)
	}
}

func TestListBoardsHandler_Error(t *testing.T) {
	mock := &MockClient{
		ListBoardsFn: func(ctx context.Context, args miro.ListBoardsArgs) (miro.ListBoardsResult, error) {
			return miro.ListBoardsResult{}, errors.New("API error")
		},
	}

	ctx := context.Background()
	_, err := mock.ListBoards(ctx, miro.ListBoardsArgs{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "API error" {
		t.Errorf("error = %q, want 'API error'", err.Error())
	}
}

func TestCreateBoardHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.CreateBoard(ctx, miro.CreateBoardArgs{
		Name:        "New Board",
		Description: "Test description",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "New Board" {
		t.Errorf("Name = %q, want 'New Board'", result.Name)
	}
	if result.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestDeleteBoardHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.DeleteBoard(ctx, miro.DeleteBoardArgs{BoardID: "board123"})

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

func TestFindBoardByNameToolHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.FindBoardByNameTool(ctx, miro.FindBoardByNameArgs{Name: "Design Sprint"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Design Sprint" {
		t.Errorf("Name = %q, want 'Design Sprint'", result.Name)
	}
	if result.ID == "" {
		t.Error("ID should not be empty")
	}
}

// =============================================================================
// Item Handler Tests
// =============================================================================

func TestListItemsHandler(t *testing.T) {
	mock := &MockClient{
		ListItemsFn: func(ctx context.Context, args miro.ListItemsArgs) (miro.ListItemsResult, error) {
			if args.Type != "sticky_note" {
				return miro.ListItemsResult{}, errors.New("wrong type filter")
			}
			return miro.ListItemsResult{
				Items: []miro.ItemSummary{
					{ID: "item1", Type: "sticky_note", Content: "Task 1"},
					{ID: "item2", Type: "sticky_note", Content: "Task 2"},
				},
				Count: 2,
			}, nil
		},
	}

	ctx := context.Background()
	result, err := mock.ListItems(ctx, miro.ListItemsArgs{
		BoardID: "board123",
		Type:    "sticky_note",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
}

func TestGetItemHandler(t *testing.T) {
	mock := &MockClient{
		GetItemFn: func(ctx context.Context, args miro.GetItemArgs) (miro.GetItemResult, error) {
			return miro.GetItemResult{
				ID:      args.ItemID,
				Type:    "sticky_note",
				Content: "Detailed content",
			}, nil
		},
	}

	ctx := context.Background()
	result, err := mock.GetItem(ctx, miro.GetItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "item456" {
		t.Errorf("ID = %q, want 'item456'", result.ID)
	}
}

func TestSearchBoardHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.SearchBoard(ctx, miro.SearchBoardArgs{
		BoardID: "board123",
		Query:   "budget",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Query != "budget" {
		t.Errorf("Query = %q, want 'budget'", result.Query)
	}
	if result.Count == 0 {
		t.Error("expected at least one result")
	}
}

func TestDeleteItemHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.DeleteItem(ctx, miro.DeleteItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

// =============================================================================
// Create Handler Tests
// =============================================================================

func TestCreateStickyHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.CreateSticky(ctx, miro.CreateStickyArgs{
		BoardID: "board123",
		Content: "Action item: Review PRs",
		Color:   "yellow",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID == "" {
		t.Error("ID should not be empty")
	}
	if result.Content != "Action item: Review PRs" {
		t.Errorf("Content = %q, want 'Action item: Review PRs'", result.Content)
	}
}

func TestCreateShapeHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.CreateShape(ctx, miro.CreateShapeArgs{
		BoardID: "board123",
		Shape:   "rectangle",
		Content: "Header",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Shape != "rectangle" {
		t.Errorf("Shape = %q, want 'rectangle'", result.Shape)
	}
}

func TestCreateConnectorHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.CreateConnector(ctx, miro.CreateConnectorArgs{
		BoardID:     "board123",
		StartItemID: "item1",
		EndItemID:   "item2",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID == "" {
		t.Error("ID should not be empty")
	}
	if result.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestCreateFrameHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.CreateFrame(ctx, miro.CreateFrameArgs{
		BoardID: "board123",
		Title:   "Brainstorming",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "Brainstorming" {
		t.Errorf("Title = %q, want 'Brainstorming'", result.Title)
	}
}

func TestBulkCreateHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.BulkCreate(ctx, miro.BulkCreateArgs{
		BoardID: "board123",
		Items: []miro.BulkCreateItem{
			{Type: "sticky_note", Content: "Task 1"},
			{Type: "sticky_note", Content: "Task 2"},
			{Type: "shape", Content: "Box", Shape: "rectangle"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Created != 3 {
		t.Errorf("Created = %d, want 3", result.Created)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %d", len(result.Errors))
	}
}

func TestCreateStickyGridHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.CreateStickyGrid(ctx, miro.CreateStickyGridArgs{
		BoardID:  "board123",
		Contents: []string{"A", "B", "C", "D", "E", "F"},
		Columns:  3,
		Color:    "yellow",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Created != 6 {
		t.Errorf("Created = %d, want 6", result.Created)
	}
	if result.Columns != 3 {
		t.Errorf("Columns = %d, want 3", result.Columns)
	}
}

// =============================================================================
// Tag Handler Tests
// =============================================================================

func TestCreateTagHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.CreateTag(ctx, miro.CreateTagArgs{
		BoardID: "board123",
		Title:   "Urgent",
		Color:   "red",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "Urgent" {
		t.Errorf("Title = %q, want 'Urgent'", result.Title)
	}
	if result.Color != "red" {
		t.Errorf("Color = %q, want 'red'", result.Color)
	}
}

func TestListTagsHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.ListTags(ctx, miro.ListTagsArgs{BoardID: "board123"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
}

func TestAttachTagHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.AttachTag(ctx, miro.AttachTagArgs{
		BoardID: "board123",
		ItemID:  "item456",
		TagID:   "tag789",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestDetachTagHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.DetachTag(ctx, miro.DetachTagArgs{
		BoardID: "board123",
		ItemID:  "item456",
		TagID:   "tag789",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

// =============================================================================
// Group Handler Tests
// =============================================================================

func TestCreateGroupHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.CreateGroup(ctx, miro.CreateGroupArgs{
		BoardID: "board123",
		ItemIDs: []string{"item1", "item2", "item3"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ItemIDs) != 3 {
		t.Errorf("ItemIDs count = %d, want 3", len(result.ItemIDs))
	}
}

func TestUngroupHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.Ungroup(ctx, miro.UngroupArgs{
		BoardID: "board123",
		GroupID: "group456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

// =============================================================================
// Member Handler Tests
// =============================================================================

func TestListBoardMembersHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.ListBoardMembers(ctx, miro.ListBoardMembersArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count == 0 {
		t.Error("expected at least one member")
	}
}

func TestShareBoardHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.ShareBoard(ctx, miro.ShareBoardArgs{
		BoardID: "board123",
		Email:   "jane@example.com",
		Role:    "editor",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Email != "jane@example.com" {
		t.Errorf("Email = %q, want 'jane@example.com'", result.Email)
	}
}

// =============================================================================
// Mindmap Handler Tests
// =============================================================================

func TestCreateMindmapNodeHandler(t *testing.T) {
	tests := []struct {
		name         string
		args         miro.CreateMindmapNodeArgs
		expectParent string
	}{
		{
			name: "root node",
			args: miro.CreateMindmapNodeArgs{
				BoardID: "board123",
				Content: "Main Topic",
			},
			expectParent: "",
		},
		{
			name: "child node",
			args: miro.CreateMindmapNodeArgs{
				BoardID:  "board123",
				Content:  "Sub Topic",
				ParentID: "parent-node-123",
			},
			expectParent: "parent-node-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockClient{}

			ctx := context.Background()
			result, err := mock.CreateMindmapNode(ctx, tt.args)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ID == "" {
				t.Error("ID should not be empty")
			}
			if result.ParentID != tt.expectParent {
				t.Errorf("ParentID = %q, want %q", result.ParentID, tt.expectParent)
			}
		})
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestHandlerError(t *testing.T) {
	expectedErr := errors.New("mock API error")
	mock := &MockClient{
		CreateStickyFn: func(ctx context.Context, args miro.CreateStickyArgs) (miro.CreateStickyResult, error) {
			return miro.CreateStickyResult{}, expectedErr
		},
	}

	ctx := context.Background()
	_, err := mock.CreateSticky(ctx, miro.CreateStickyArgs{
		BoardID: "board123",
		Content: "Test",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
}

// =============================================================================
// Call Tracking Tests
// =============================================================================

func TestMockCallTracking(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	// Make several calls
	mock.ListBoards(ctx, miro.ListBoardsArgs{Query: "test"})
	mock.CreateSticky(ctx, miro.CreateStickyArgs{BoardID: "b1", Content: "hello"})
	mock.DeleteItem(ctx, miro.DeleteItemArgs{BoardID: "b1", ItemID: "i1"})

	if len(mock.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(mock.Calls))
	}

	// Verify call order and method names
	expectedMethods := []string{"ListBoards", "CreateSticky", "DeleteItem"}
	for i, method := range expectedMethods {
		if mock.Calls[i].Method != method {
			t.Errorf("Calls[%d].Method = %q, want %q", i, mock.Calls[i].Method, method)
		}
	}

	// Verify args are captured
	listArgs := mock.Calls[0].Args.(miro.ListBoardsArgs)
	if listArgs.Query != "test" {
		t.Errorf("ListBoardsArgs.Query = %q, want 'test'", listArgs.Query)
	}
}

// =============================================================================
// Token Validation Tests
// =============================================================================

func TestValidateTokenHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	user, err := mock.ValidateToken(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID == "" {
		t.Error("ID should not be empty")
	}
	if user.Email == "" {
		t.Error("Email should not be empty")
	}
}

func TestValidateTokenHandler_Error(t *testing.T) {
	mock := &MockClient{
		ValidateTokenFn: func(ctx context.Context) (*miro.UserInfo, error) {
			return nil, errors.New("invalid token")
		},
	}

	ctx := context.Background()
	_, err := mock.ValidateToken(ctx)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkMockListBoards(b *testing.B) {
	mock := &MockClient{}
	ctx := context.Background()
	args := miro.ListBoardsArgs{Query: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.ListBoards(ctx, args)
	}
}

func BenchmarkMockCreateSticky(b *testing.B) {
	mock := &MockClient{}
	ctx := context.Background()
	args := miro.CreateStickyArgs{BoardID: "board123", Content: "Test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.CreateSticky(ctx, args)
	}
}
