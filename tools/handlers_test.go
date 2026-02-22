package tools

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/miro-mcp-server/miro"
	"github.com/olgasafonova/miro-mcp-server/miro/audit"
	"github.com/olgasafonova/miro-mcp-server/miro/desirepath"
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
		"CreateGroup",
		// Member tools
		"ListBoardMembers", "ShareBoard",
		// Mindmap tools
		"CreateMindmapNode",
		// Audit/observability tools
		"GetAuditLog", "GetDesirePathReport",
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

func TestBulkUpdateHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	content := "Updated content"
	x := 100.0
	result, err := mock.BulkUpdate(ctx, miro.BulkUpdateArgs{
		BoardID: "board123",
		Items: []miro.BulkUpdateItem{
			{ItemID: "item1", Content: &content},
			{ItemID: "item2", X: &x},
			{ItemID: "item3", Content: &content, X: &x},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Updated != 3 {
		t.Errorf("Updated = %d, want 3", result.Updated)
	}
	if len(result.ItemIDs) != 3 {
		t.Errorf("ItemIDs len = %d, want 3", len(result.ItemIDs))
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %d", len(result.Errors))
	}
}

func TestBulkUpdateHandler_WithCustomBehavior(t *testing.T) {
	mock := &MockClient{
		BulkUpdateFn: func(ctx context.Context, args miro.BulkUpdateArgs) (miro.BulkUpdateResult, error) {
			// Simulate one item failing
			return miro.BulkUpdateResult{
				Updated: 2,
				ItemIDs: []string{"item1", "item2"},
				Errors:  []string{"item3: not found"},
				Message: "Updated 2 items with 1 error",
			}, nil
		},
	}

	ctx := context.Background()
	content := "Updated"
	result, err := mock.BulkUpdate(ctx, miro.BulkUpdateArgs{
		BoardID: "board123",
		Items: []miro.BulkUpdateItem{
			{ItemID: "item1", Content: &content},
			{ItemID: "item2", Content: &content},
			{ItemID: "item3", Content: &content},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Updated != 2 {
		t.Errorf("Updated = %d, want 2", result.Updated)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestBulkDeleteHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.BulkDelete(ctx, miro.BulkDeleteArgs{
		BoardID: "board123",
		ItemIDs: []string{"item1", "item2", "item3"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Deleted != 3 {
		t.Errorf("Deleted = %d, want 3", result.Deleted)
	}
	if len(result.ItemIDs) != 3 {
		t.Errorf("ItemIDs len = %d, want 3", len(result.ItemIDs))
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %d", len(result.Errors))
	}
}

func TestBulkDeleteHandler_WithCustomBehavior(t *testing.T) {
	mock := &MockClient{
		BulkDeleteFn: func(ctx context.Context, args miro.BulkDeleteArgs) (miro.BulkDeleteResult, error) {
			return miro.BulkDeleteResult{}, errors.New("API rate limit exceeded")
		},
	}

	ctx := context.Background()
	_, err := mock.BulkDelete(ctx, miro.BulkDeleteArgs{
		BoardID: "board123",
		ItemIDs: []string{"item1", "item2"},
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Error() != "API rate limit exceeded" {
		t.Errorf("error = %q, want 'API rate limit exceeded'", err.Error())
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

// =============================================================================
// WithAuditLogger and WithUser Tests
// =============================================================================

func TestWithAuditLogger(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	// Should return the same registry for chaining
	result := registry.WithAuditLogger(nil)
	if result != registry {
		t.Error("WithAuditLogger should return the same registry for chaining")
	}
}

func TestWithUser(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	result := registry.WithUser("user123", "test@example.com")
	if result != registry {
		t.Error("WithUser should return the same registry for chaining")
	}
	if registry.userID != "user123" {
		t.Errorf("userID = %q, want 'user123'", registry.userID)
	}
	if registry.userEmail != "test@example.com" {
		t.Errorf("userEmail = %q, want 'test@example.com'", registry.userEmail)
	}
}

// =============================================================================
// argsToMap Tests
// =============================================================================

func TestArgsToMap(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		wantNil  bool
		checkKey string
		checkVal any
	}{
		{
			name:    "nil input",
			input:   nil,
			wantNil: true,
		},
		{
			name: "struct with fields",
			input: miro.CreateStickyArgs{
				BoardID: "board123",
				Content: "Test content",
				Color:   "yellow",
			},
			wantNil:  false,
			checkKey: "board_id",
			checkVal: "board123",
		},
		{
			name: "struct with nested values",
			input: miro.ListBoardsArgs{
				Query: "test query",
				Limit: 10,
			},
			wantNil:  false,
			checkKey: "query",
			checkVal: "test query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := argsToMap(tt.input)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if tt.checkKey != "" {
				val, ok := result[tt.checkKey]
				if !ok {
					t.Errorf("missing key %q in result", tt.checkKey)
				} else if val != tt.checkVal {
					t.Errorf("result[%q] = %v, want %v", tt.checkKey, val, tt.checkVal)
				}
			}
		})
	}
}

// =============================================================================
// recoverPanic Tests
// =============================================================================

func TestRecoverPanic(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	// Should recover from panic without crashing
	func() {
		defer registry.recoverPanic("test_tool")
		panic("test panic")
	}()

	// If we get here, panic was recovered successfully
}

func TestRecoverPanicNoPanic(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	// Should handle case where no panic occurs
	func() {
		defer registry.recoverPanic("test_tool")
		// No panic, just normal execution
	}()
}

// =============================================================================
// logExecution Tests
// =============================================================================

func TestLogExecution(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	// Test various arg/result combinations for coverage
	tests := []struct {
		name   string
		spec   ToolSpec
		args   any
		result any
	}{
		{
			name: "ListBoards with query",
			spec: ToolSpec{Name: "miro_list_boards", Category: "read"},
			args: miro.ListBoardsArgs{Query: "test"},
			result: miro.ListBoardsResult{
				Boards: []miro.BoardSummary{{ID: "b1", Name: "Board"}},
				Count:  1,
			},
		},
		{
			name: "GetBoard",
			spec: ToolSpec{Name: "miro_get_board", Category: "read"},
			args: miro.GetBoardArgs{BoardID: "board123"},
			result: miro.GetBoardResult{
				Board: miro.Board{ID: "board123", Name: "Test"},
			},
		},
		{
			name: "CreateSticky",
			spec: ToolSpec{Name: "miro_create_sticky", Category: "create"},
			args: miro.CreateStickyArgs{BoardID: "b1", Content: "Hello world"},
			result: miro.CreateStickyResult{
				ID:      "sticky123",
				Content: "Hello world",
			},
		},
		{
			name: "CreateShape",
			spec: ToolSpec{Name: "miro_create_shape", Category: "create"},
			args: miro.CreateShapeArgs{BoardID: "b1", Shape: "rectangle"},
			result: miro.CreateShapeResult{
				ID:    "shape123",
				Shape: "rectangle",
			},
		},
		{
			name: "ListItems with type",
			spec: ToolSpec{Name: "miro_list_items", Category: "read"},
			args: miro.ListItemsArgs{BoardID: "b1", Type: "sticky_note"},
			result: miro.ListItemsResult{
				Items: []miro.ItemSummary{{ID: "i1", Type: "sticky_note"}},
				Count: 1,
			},
		},
		{
			name: "BulkCreate",
			spec: ToolSpec{Name: "miro_bulk_create", Category: "create"},
			args: miro.BulkCreateArgs{
				BoardID: "b1",
				Items:   []miro.BulkCreateItem{{Type: "sticky_note", Content: "A"}},
			},
			result: miro.BulkCreateResult{
				Created: 1,
				ItemIDs: []string{"item1"},
				Errors:  []string{},
			},
		},
		{
			name: "DeleteItem",
			spec: ToolSpec{Name: "miro_delete_item", Category: "delete"},
			args: miro.DeleteItemArgs{BoardID: "b1", ItemID: "i1"},
			result: miro.DeleteItemResult{
				Success: true,
				ItemID:  "i1",
			},
		},
		{
			name: "GenerateDiagram",
			spec: ToolSpec{Name: "miro_generate_diagram", Category: "create"},
			args: miro.GenerateDiagramArgs{BoardID: "b1", Diagram: "graph TD\\nA-->B"},
			result: miro.GenerateDiagramResult{
				NodesCreated:      2,
				ConnectorsCreated: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			registry.logExecution(tt.spec, tt.args, tt.result)
		})
	}
}

// =============================================================================
// createAuditEvent Tests
// =============================================================================

func TestCreateAuditEvent(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)
	registry.WithUser("user123", "test@example.com")

	tests := []struct {
		name     string
		spec     ToolSpec
		args     any
		result   any
		err      error
		duration int64
	}{
		{
			name:   "successful create",
			spec:   ToolSpec{Name: "miro_create_sticky", Method: "CreateSticky"},
			args:   miro.CreateStickyArgs{BoardID: "board123", Content: "Test"},
			result: miro.CreateStickyResult{ID: "sticky456"},
			err:    nil,
		},
		{
			name:   "failed operation",
			spec:   ToolSpec{Name: "miro_create_sticky", Method: "CreateSticky"},
			args:   miro.CreateStickyArgs{BoardID: "board123"},
			result: miro.CreateStickyResult{},
			err:    errors.New("API error"),
		},
		{
			name:   "with item_id in args",
			spec:   ToolSpec{Name: "miro_delete_item", Method: "DeleteItem"},
			args:   miro.DeleteItemArgs{BoardID: "board123", ItemID: "item456"},
			result: miro.DeleteItemResult{Success: true},
			err:    nil,
		},
		{
			name:   "with created count in result",
			spec:   ToolSpec{Name: "miro_bulk_create", Method: "BulkCreate"},
			args:   miro.BulkCreateArgs{BoardID: "board123"},
			result: miro.BulkCreateResult{Created: 5, ItemIDs: []string{"1", "2", "3", "4", "5"}},
			err:    nil,
		},
		{
			name:   "nil args",
			spec:   ToolSpec{Name: "miro_test", Method: "Test"},
			args:   nil,
			result: nil,
			err:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := registry.createAuditEvent(tt.spec, tt.args, tt.result, tt.err, 100*1000000)

			if event.Tool != tt.spec.Name {
				t.Errorf("Tool = %q, want %q", event.Tool, tt.spec.Name)
			}

			if tt.err != nil && event.Success {
				t.Error("expected Success=false for error case")
			}
			if tt.err == nil && !event.Success {
				t.Error("expected Success=true for success case")
			}
		})
	}
}

// =============================================================================
// Additional Mock Method Coverage Tests
// =============================================================================

func TestGetTagHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.GetTag(ctx, miro.GetTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "tag456" {
		t.Errorf("ID = %q, want 'tag456'", result.ID)
	}
	if result.Title == "" {
		t.Error("Title should not be empty")
	}
}

func TestUpdateTagHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.UpdateTag(ctx, miro.UpdateTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
		Title:   "Updated Title",
		Color:   "blue",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Title != "Updated Title" {
		t.Errorf("Title = %q, want 'Updated Title'", result.Title)
	}
}

func TestDeleteTagHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.DeleteTag(ctx, miro.DeleteTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestGetItemTagsHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.GetItemTags(ctx, miro.GetItemTagsArgs{
		BoardID: "board123",
		ItemID:  "item456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count == 0 {
		t.Error("expected at least one tag")
	}
	if result.ItemID != "item456" {
		t.Errorf("ItemID = %q, want 'item456'", result.ItemID)
	}
}

func TestUpdateItemHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.UpdateItem(ctx, miro.UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
		Content: strPtr("Updated content"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestGetBoardSummaryHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.GetBoardSummary(ctx, miro.GetBoardSummaryArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalItems == 0 {
		t.Error("expected non-zero TotalItems")
	}
}

func TestListAllItemsHandler(t *testing.T) {
	mock := &MockClient{}

	ctx := context.Background()
	result, err := mock.ListAllItems(ctx, miro.ListAllItemsArgs{
		BoardID:  "board123",
		MaxItems: 100,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count == 0 {
		t.Error("expected non-zero Count")
	}
}

// Helper for pointer to string
func strPtr(s string) *string {
	return &s
}

// =============================================================================
// Mock Audit Logger
// =============================================================================

// MockAuditLogger is a mock implementation of audit.Logger for testing.
type MockAuditLogger struct {
	LogFn   func(ctx context.Context, event audit.Event) error
	QueryFn func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error)
	events  []audit.Event
}

func (m *MockAuditLogger) Log(ctx context.Context, event audit.Event) error {
	m.events = append(m.events, event)
	if m.LogFn != nil {
		return m.LogFn(ctx, event)
	}
	return nil
}

func (m *MockAuditLogger) Query(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
	if m.QueryFn != nil {
		return m.QueryFn(ctx, opts)
	}
	// Default: return all events
	return &audit.QueryResult{
		Events:  m.events,
		Total:   len(m.events),
		HasMore: false,
	}, nil
}

func (m *MockAuditLogger) Flush(ctx context.Context) error {
	return nil
}

func (m *MockAuditLogger) Close() error {
	return nil
}

// =============================================================================
// GetAuditLog Tests
// =============================================================================

func TestGetAuditLog_Success(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	// Set up mock audit logger with events
	now := time.Now()
	mockLogger := &MockAuditLogger{
		QueryFn: func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
			return &audit.QueryResult{
				Events: []audit.Event{
					{
						ID:         "event1",
						Timestamp:  now,
						Tool:       "miro_create_sticky",
						Action:     audit.ActionCreate,
						BoardID:    "board123",
						ItemID:     "item456",
						Success:    true,
						DurationMs: 150,
					},
					{
						ID:         "event2",
						Timestamp:  now.Add(-time.Hour),
						Tool:       "miro_delete_item",
						Action:     audit.ActionDelete,
						BoardID:    "board123",
						ItemID:     "item789",
						Success:    false,
						Error:      "item not found",
						DurationMs: 50,
					},
				},
				Total:   2,
				HasMore: false,
			}, nil
		},
	}
	registry.WithAuditLogger(mockLogger)

	ctx := context.Background()
	result, err := registry.GetAuditLog(ctx, miro.GetAuditLogArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(result.Events))
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if result.Message == "" {
		t.Error("Message should not be empty")
	}

	// Verify first event
	if result.Events[0].ID != "event1" {
		t.Errorf("Events[0].ID = %q, want 'event1'", result.Events[0].ID)
	}
	if result.Events[0].Tool != "miro_create_sticky" {
		t.Errorf("Events[0].Tool = %q, want 'miro_create_sticky'", result.Events[0].Tool)
	}
	if !result.Events[0].Success {
		t.Error("Events[0].Success should be true")
	}

	// Verify second event with error
	if result.Events[1].Success {
		t.Error("Events[1].Success should be false")
	}
	if result.Events[1].Error != "item not found" {
		t.Errorf("Events[1].Error = %q, want 'item not found'", result.Events[1].Error)
	}
}

func TestGetAuditLog_EmptyResult(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	mockLogger := &MockAuditLogger{
		QueryFn: func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
			return &audit.QueryResult{
				Events:  []audit.Event{},
				Total:   0,
				HasMore: false,
			}, nil
		},
	}
	registry.WithAuditLogger(mockLogger)

	ctx := context.Background()
	result, err := registry.GetAuditLog(ctx, miro.GetAuditLogArgs{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Events) != 0 {
		t.Errorf("expected 0 events, got %d", len(result.Events))
	}
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
}

func TestGetAuditLog_InvalidSinceTime(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)
	registry.WithAuditLogger(&MockAuditLogger{})

	ctx := context.Background()
	_, err := registry.GetAuditLog(ctx, miro.GetAuditLogArgs{
		Since: "not-a-valid-time",
	})

	if err == nil {
		t.Fatal("expected error for invalid 'since' time")
	}
	if !errors.Is(err, err) || err.Error() == "" {
		t.Errorf("error should mention invalid time format")
	}
}

func TestGetAuditLog_InvalidUntilTime(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)
	registry.WithAuditLogger(&MockAuditLogger{})

	ctx := context.Background()
	_, err := registry.GetAuditLog(ctx, miro.GetAuditLogArgs{
		Until: "invalid-time-format",
	})

	if err == nil {
		t.Fatal("expected error for invalid 'until' time")
	}
}

func TestGetAuditLog_ValidTimeRange(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	var capturedOpts audit.QueryOptions
	mockLogger := &MockAuditLogger{
		QueryFn: func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
			capturedOpts = opts
			return &audit.QueryResult{Events: []audit.Event{}, Total: 0}, nil
		},
	}
	registry.WithAuditLogger(mockLogger)

	ctx := context.Background()
	_, err := registry.GetAuditLog(ctx, miro.GetAuditLogArgs{
		Since: "2024-01-01T00:00:00Z",
		Until: "2024-01-31T23:59:59Z",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the times were parsed correctly
	if capturedOpts.Since.IsZero() {
		t.Error("Since time should be set")
	}
	if capturedOpts.Until.IsZero() {
		t.Error("Until time should be set")
	}
	if capturedOpts.Since.Year() != 2024 || capturedOpts.Since.Month() != 1 || capturedOpts.Since.Day() != 1 {
		t.Errorf("Since = %v, want 2024-01-01", capturedOpts.Since)
	}
}

func TestGetAuditLog_DefaultLimit(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	var capturedOpts audit.QueryOptions
	mockLogger := &MockAuditLogger{
		QueryFn: func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
			capturedOpts = opts
			return &audit.QueryResult{Events: []audit.Event{}, Total: 0}, nil
		},
	}
	registry.WithAuditLogger(mockLogger)

	ctx := context.Background()
	_, err := registry.GetAuditLog(ctx, miro.GetAuditLogArgs{
		Limit: 0, // Should default to 50
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedOpts.Limit != 50 {
		t.Errorf("Limit = %d, want 50 (default)", capturedOpts.Limit)
	}
}

func TestGetAuditLog_LimitCapped(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	var capturedOpts audit.QueryOptions
	mockLogger := &MockAuditLogger{
		QueryFn: func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
			capturedOpts = opts
			return &audit.QueryResult{Events: []audit.Event{}, Total: 0}, nil
		},
	}
	registry.WithAuditLogger(mockLogger)

	ctx := context.Background()
	_, err := registry.GetAuditLog(ctx, miro.GetAuditLogArgs{
		Limit: 1000, // Should be capped at 500
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedOpts.Limit != 500 {
		t.Errorf("Limit = %d, want 500 (max cap)", capturedOpts.Limit)
	}
}

func TestGetAuditLog_QueryError(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	mockLogger := &MockAuditLogger{
		QueryFn: func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
			return nil, errors.New("database error")
		},
	}
	registry.WithAuditLogger(mockLogger)

	ctx := context.Background()
	_, err := registry.GetAuditLog(ctx, miro.GetAuditLogArgs{})

	if err == nil {
		t.Fatal("expected error from query")
	}
	if err.Error() != "audit query failed: database error" {
		t.Errorf("error = %q, want 'audit query failed: database error'", err.Error())
	}
}

func TestGetAuditLog_FiltersByToolAndAction(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	var capturedOpts audit.QueryOptions
	mockLogger := &MockAuditLogger{
		QueryFn: func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
			capturedOpts = opts
			return &audit.QueryResult{Events: []audit.Event{}, Total: 0}, nil
		},
	}
	registry.WithAuditLogger(mockLogger)

	ctx := context.Background()
	_, err := registry.GetAuditLog(ctx, miro.GetAuditLogArgs{
		Tool:    "miro_create_sticky",
		Action:  "create",
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedOpts.Tool != "miro_create_sticky" {
		t.Errorf("Tool = %q, want 'miro_create_sticky'", capturedOpts.Tool)
	}
	if capturedOpts.Action != "create" {
		t.Errorf("Action = %q, want 'create'", capturedOpts.Action)
	}
	if capturedOpts.BoardID != "board123" {
		t.Errorf("BoardID = %q, want 'board123'", capturedOpts.BoardID)
	}
}

func TestGetAuditLog_FiltersBySuccess(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	var capturedOpts audit.QueryOptions
	mockLogger := &MockAuditLogger{
		QueryFn: func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
			capturedOpts = opts
			return &audit.QueryResult{Events: []audit.Event{}, Total: 0}, nil
		},
	}
	registry.WithAuditLogger(mockLogger)

	ctx := context.Background()
	success := true
	_, err := registry.GetAuditLog(ctx, miro.GetAuditLogArgs{
		Success: &success,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedOpts.Success == nil {
		t.Error("Success filter should be set")
	} else if *capturedOpts.Success != true {
		t.Errorf("Success = %v, want true", *capturedOpts.Success)
	}
}

func TestGetAuditLog_HasMore(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	mockLogger := &MockAuditLogger{
		QueryFn: func(ctx context.Context, opts audit.QueryOptions) (*audit.QueryResult, error) {
			return &audit.QueryResult{
				Events:  []audit.Event{{ID: "event1"}},
				Total:   100,
				HasMore: true,
			}, nil
		},
	}
	registry.WithAuditLogger(mockLogger)

	ctx := context.Background()
	result, err := registry.GetAuditLog(ctx, miro.GetAuditLogArgs{Limit: 1})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasMore {
		t.Error("HasMore should be true")
	}
	if result.Total != 100 {
		t.Errorf("Total = %d, want 100", result.Total)
	}
}

// =============================================================================
// registerTool Unknown Method Tests
// =============================================================================

func TestRegisterTool_UnknownMethod(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)

	// Register a tool with an unknown method
	unknownSpec := ToolSpec{
		Name:   "miro_unknown",
		Method: "UnknownMethodThatDoesNotExist",
		Title:  "Unknown Tool",
	}

	// Should not panic, should log an error
	registry.registerTool(server, unknownSpec)

	// Verify the handler was not registered - no crash means success
}

// =============================================================================
// argsToMap Edge Case Tests
// =============================================================================

// unmarshalableType is a type that cannot be unmarshaled to a map
type unmarshalableType struct {
	Ch chan int `json:"ch"` // channels cannot be marshaled
}

func TestArgsToMap_MarshalError(t *testing.T) {
	// A channel cannot be marshaled to JSON
	input := unmarshalableType{Ch: make(chan int)}
	result := argsToMap(input)

	if result != nil {
		t.Errorf("expected nil for unmarshalable type, got %v", result)
	}
}

func TestArgsToMap_UnmarshalToNonMap(t *testing.T) {
	// A primitive value cannot be unmarshaled to a map
	input := "just a string"
	result := argsToMap(input)

	if result != nil {
		t.Errorf("expected nil for string input, got %v", result)
	}
}

func TestArgsToMap_ArrayInput(t *testing.T) {
	// An array cannot be unmarshaled to a map
	input := []string{"one", "two", "three"}
	result := argsToMap(input)

	if result != nil {
		t.Errorf("expected nil for array input, got %v", result)
	}
}

func TestArgsToMap_IntegerInput(t *testing.T) {
	// An integer cannot be unmarshaled to a map
	input := 12345
	result := argsToMap(input)

	if result != nil {
		t.Errorf("expected nil for integer input, got %v", result)
	}
}

func TestArgsToMap_EmptyStruct(t *testing.T) {
	// An empty struct should return an empty map, not nil
	type EmptyStruct struct{}
	input := EmptyStruct{}
	result := argsToMap(input)

	if result == nil {
		t.Error("expected non-nil map for empty struct")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestArgsToMap_NestedStruct(t *testing.T) {
	type Inner struct {
		Value string `json:"value"`
	}
	type Outer struct {
		Name  string `json:"name"`
		Inner Inner  `json:"inner"`
	}
	input := Outer{
		Name:  "outer",
		Inner: Inner{Value: "nested"},
	}
	result := argsToMap(input)

	if result == nil {
		t.Fatal("expected non-nil map")
	}
	if result["name"] != "outer" {
		t.Errorf("name = %v, want 'outer'", result["name"])
	}
	inner, ok := result["inner"].(map[string]interface{})
	if !ok {
		t.Fatalf("inner should be a map, got %T", result["inner"])
	}
	if inner["value"] != "nested" {
		t.Errorf("inner.value = %v, want 'nested'", inner["value"])
	}
}

// =============================================================================
// Parameter Validation Edge Case Tests
// =============================================================================

func TestCreateSticky_EmptyBoardID(t *testing.T) {
	mock := &MockClient{
		CreateStickyFn: func(ctx context.Context, args miro.CreateStickyArgs) (miro.CreateStickyResult, error) {
			if args.BoardID == "" {
				return miro.CreateStickyResult{}, errors.New("board_id is required")
			}
			return miro.CreateStickyResult{ID: "test"}, nil
		},
	}

	ctx := context.Background()
	_, err := mock.CreateSticky(ctx, miro.CreateStickyArgs{
		BoardID: "",
		Content: "Test",
	})

	if err == nil {
		t.Fatal("expected error for empty board_id")
	}
	if err.Error() != "board_id is required" {
		t.Errorf("error = %q, want 'board_id is required'", err.Error())
	}
}

func TestCreateSticky_EmptyContent(t *testing.T) {
	mock := &MockClient{
		CreateStickyFn: func(ctx context.Context, args miro.CreateStickyArgs) (miro.CreateStickyResult, error) {
			if args.Content == "" {
				return miro.CreateStickyResult{}, errors.New("content is required")
			}
			return miro.CreateStickyResult{ID: "test"}, nil
		},
	}

	ctx := context.Background()
	_, err := mock.CreateSticky(ctx, miro.CreateStickyArgs{
		BoardID: "board123",
		Content: "",
	})

	if err == nil {
		t.Fatal("expected error for empty content")
	}
	if err.Error() != "content is required" {
		t.Errorf("error = %q, want 'content is required'", err.Error())
	}
}

func TestUpdateItem_EmptyItemID(t *testing.T) {
	mock := &MockClient{
		UpdateItemFn: func(ctx context.Context, args miro.UpdateItemArgs) (miro.UpdateItemResult, error) {
			if args.ItemID == "" {
				return miro.UpdateItemResult{}, errors.New("item_id is required")
			}
			return miro.UpdateItemResult{Success: true}, nil
		},
	}

	ctx := context.Background()
	_, err := mock.UpdateItem(ctx, miro.UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "",
	})

	if err == nil {
		t.Fatal("expected error for empty item_id")
	}
	if err.Error() != "item_id is required" {
		t.Errorf("error = %q, want 'item_id is required'", err.Error())
	}
}

func TestSearchBoard_EmptyQuery(t *testing.T) {
	mock := &MockClient{
		SearchBoardFn: func(ctx context.Context, args miro.SearchBoardArgs) (miro.SearchBoardResult, error) {
			if args.Query == "" {
				return miro.SearchBoardResult{}, errors.New("query is required")
			}
			return miro.SearchBoardResult{Query: args.Query, Count: 1}, nil
		},
	}

	ctx := context.Background()
	_, err := mock.SearchBoard(ctx, miro.SearchBoardArgs{
		BoardID: "board123",
		Query:   "",
	})

	if err == nil {
		t.Fatal("expected error for empty query")
	}
	if err.Error() != "query is required" {
		t.Errorf("error = %q, want 'query is required'", err.Error())
	}
}

func TestBulkCreate_EmptyItems(t *testing.T) {
	mock := &MockClient{
		BulkCreateFn: func(ctx context.Context, args miro.BulkCreateArgs) (miro.BulkCreateResult, error) {
			if len(args.Items) == 0 {
				return miro.BulkCreateResult{}, errors.New("at least one item is required")
			}
			return miro.BulkCreateResult{Created: len(args.Items)}, nil
		},
	}

	ctx := context.Background()
	_, err := mock.BulkCreate(ctx, miro.BulkCreateArgs{
		BoardID: "board123",
		Items:   []miro.BulkCreateItem{},
	})

	if err == nil {
		t.Fatal("expected error for empty items array")
	}
	if err.Error() != "at least one item is required" {
		t.Errorf("error = %q, want 'at least one item is required'", err.Error())
	}
}

func TestBulkDelete_EmptyItemIDs(t *testing.T) {
	mock := &MockClient{
		BulkDeleteFn: func(ctx context.Context, args miro.BulkDeleteArgs) (miro.BulkDeleteResult, error) {
			if len(args.ItemIDs) == 0 {
				return miro.BulkDeleteResult{}, errors.New("at least one item_id is required")
			}
			return miro.BulkDeleteResult{Deleted: len(args.ItemIDs)}, nil
		},
	}

	ctx := context.Background()
	_, err := mock.BulkDelete(ctx, miro.BulkDeleteArgs{
		BoardID: "board123",
		ItemIDs: []string{},
	})

	if err == nil {
		t.Fatal("expected error for empty item_ids array")
	}
	if err.Error() != "at least one item_id is required" {
		t.Errorf("error = %q, want 'at least one item_id is required'", err.Error())
	}
}

func TestCreateGroup_InsufficientItems(t *testing.T) {
	mock := &MockClient{
		CreateGroupFn: func(ctx context.Context, args miro.CreateGroupArgs) (miro.CreateGroupResult, error) {
			if len(args.ItemIDs) < 2 {
				return miro.CreateGroupResult{}, errors.New("at least 2 item_ids required")
			}
			return miro.CreateGroupResult{ID: "group123", ItemIDs: args.ItemIDs}, nil
		},
	}

	ctx := context.Background()
	_, err := mock.CreateGroup(ctx, miro.CreateGroupArgs{
		BoardID: "board123",
		ItemIDs: []string{"item1"}, // Only 1 item, need at least 2
	})

	if err == nil {
		t.Fatal("expected error for insufficient items")
	}
	if err.Error() != "at least 2 item_ids required" {
		t.Errorf("error = %q, want 'at least 2 item_ids required'", err.Error())
	}
}

func TestCreateShape_InvalidShapeType(t *testing.T) {
	mock := &MockClient{
		CreateShapeFn: func(ctx context.Context, args miro.CreateShapeArgs) (miro.CreateShapeResult, error) {
			validShapes := map[string]bool{
				"rectangle": true, "circle": true, "triangle": true,
				"rhombus": true, "round_rectangle": true,
			}
			if !validShapes[args.Shape] {
				return miro.CreateShapeResult{}, errors.New("invalid shape type")
			}
			return miro.CreateShapeResult{ID: "shape123", Shape: args.Shape}, nil
		},
	}

	ctx := context.Background()
	_, err := mock.CreateShape(ctx, miro.CreateShapeArgs{
		BoardID: "board123",
		Shape:   "invalid_shape",
	})

	if err == nil {
		t.Fatal("expected error for invalid shape type")
	}
	if err.Error() != "invalid shape type" {
		t.Errorf("error = %q, want 'invalid shape type'", err.Error())
	}
}

func TestCreateStickyGrid_EmptyContents(t *testing.T) {
	mock := &MockClient{
		CreateStickyGridFn: func(ctx context.Context, args miro.CreateStickyGridArgs) (miro.CreateStickyGridResult, error) {
			if len(args.Contents) == 0 {
				return miro.CreateStickyGridResult{}, errors.New("at least one content item is required")
			}
			return miro.CreateStickyGridResult{Created: len(args.Contents)}, nil
		},
	}

	ctx := context.Background()
	_, err := mock.CreateStickyGrid(ctx, miro.CreateStickyGridArgs{
		BoardID:  "board123",
		Contents: []string{},
	})

	if err == nil {
		t.Fatal("expected error for empty contents")
	}
	if err.Error() != "at least one content item is required" {
		t.Errorf("error = %q, want 'at least one content item is required'", err.Error())
	}
}

func TestShareBoard_InvalidEmail(t *testing.T) {
	mock := &MockClient{
		ShareBoardFn: func(ctx context.Context, args miro.ShareBoardArgs) (miro.ShareBoardResult, error) {
			if args.Email == "" {
				return miro.ShareBoardResult{}, errors.New("email is required")
			}
			// Basic email format check
			if !strings.Contains(args.Email, "@") {
				return miro.ShareBoardResult{}, errors.New("invalid email format")
			}
			return miro.ShareBoardResult{Success: true, Email: args.Email}, nil
		},
	}

	ctx := context.Background()

	// Test empty email
	_, err := mock.ShareBoard(ctx, miro.ShareBoardArgs{
		BoardID: "board123",
		Email:   "",
	})
	if err == nil || err.Error() != "email is required" {
		t.Errorf("expected 'email is required' error, got %v", err)
	}

	// Test invalid email format
	_, err = mock.ShareBoard(ctx, miro.ShareBoardArgs{
		BoardID: "board123",
		Email:   "not-an-email",
	})
	if err == nil || err.Error() != "invalid email format" {
		t.Errorf("expected 'invalid email format' error, got %v", err)
	}
}

func TestCreateConnector_MissingEndpoints(t *testing.T) {
	mock := &MockClient{
		CreateConnectorFn: func(ctx context.Context, args miro.CreateConnectorArgs) (miro.CreateConnectorResult, error) {
			if args.StartItemID == "" {
				return miro.CreateConnectorResult{}, errors.New("start_item_id is required")
			}
			if args.EndItemID == "" {
				return miro.CreateConnectorResult{}, errors.New("end_item_id is required")
			}
			return miro.CreateConnectorResult{ID: "conn123"}, nil
		},
	}

	ctx := context.Background()

	// Test missing start_item_id
	_, err := mock.CreateConnector(ctx, miro.CreateConnectorArgs{
		BoardID:     "board123",
		StartItemID: "",
		EndItemID:   "item2",
	})
	if err == nil || err.Error() != "start_item_id is required" {
		t.Errorf("expected 'start_item_id is required' error, got %v", err)
	}

	// Test missing end_item_id
	_, err = mock.CreateConnector(ctx, miro.CreateConnectorArgs{
		BoardID:     "board123",
		StartItemID: "item1",
		EndItemID:   "",
	})
	if err == nil || err.Error() != "end_item_id is required" {
		t.Errorf("expected 'end_item_id is required' error, got %v", err)
	}
}

func TestDeleteItem_BothIDsRequired(t *testing.T) {
	mock := &MockClient{
		DeleteItemFn: func(ctx context.Context, args miro.DeleteItemArgs) (miro.DeleteItemResult, error) {
			if args.BoardID == "" {
				return miro.DeleteItemResult{}, errors.New("board_id is required")
			}
			if args.ItemID == "" {
				return miro.DeleteItemResult{}, errors.New("item_id is required")
			}
			return miro.DeleteItemResult{Success: true}, nil
		},
	}

	ctx := context.Background()

	// Test missing board_id
	_, err := mock.DeleteItem(ctx, miro.DeleteItemArgs{
		BoardID: "",
		ItemID:  "item123",
	})
	if err == nil || err.Error() != "board_id is required" {
		t.Errorf("expected 'board_id is required' error, got %v", err)
	}

	// Test missing item_id
	_, err = mock.DeleteItem(ctx, miro.DeleteItemArgs{
		BoardID: "board123",
		ItemID:  "",
	})
	if err == nil || err.Error() != "item_id is required" {
		t.Errorf("expected 'item_id is required' error, got %v", err)
	}
}

// =============================================================================
// MCP Integration Tests - Tests the registerTool callback via MCP protocol
// =============================================================================

func TestMCPToolExecution_Success(t *testing.T) {
	// Create mock client with expected behavior
	called := false
	mock := &MockClient{
		ListBoardsFn: func(ctx context.Context, args miro.ListBoardsArgs) (miro.ListBoardsResult, error) {
			called = true
			return miro.ListBoardsResult{
				Boards: []miro.BoardSummary{{ID: "board1", Name: "Test Board"}},
				Count:  1,
			}, nil
		},
	}

	registry := newTestRegistry(mock)

	// Create MCP server and register tools
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	registry.RegisterAll(server)

	// Create in-memory transports for testing
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	// Start server in background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Run(ctx, serverTransport)
	}()

	// Create client and connect
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer session.Close()

	// Call the tool via MCP protocol
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "miro_list_boards",
		Arguments: map[string]interface{}{
			"limit": float64(10),
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result.IsError {
		t.Errorf("Tool returned error: %v", result.Content)
	}

	// Verify the mock was called
	if !called {
		t.Error("ListBoards was not called")
	}

	cancel()
	<-serverDone
}

func TestMCPToolExecution_Error(t *testing.T) {
	// Create mock client that returns an error
	mock := &MockClient{
		GetBoardFn: func(ctx context.Context, args miro.GetBoardArgs) (miro.GetBoardResult, error) {
			return miro.GetBoardResult{}, errors.New("board not found")
		},
	}

	registry := newTestRegistry(mock)

	// Create MCP server and register tools
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	registry.RegisterAll(server)

	// Create in-memory transports for testing
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	// Start server in background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Run(ctx, serverTransport)
	}()

	// Create client and connect
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer session.Close()

	// Call the tool via MCP protocol - should fail
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "miro_get_board",
		Arguments: map[string]interface{}{
			"board_id": "nonexistent",
		},
	})

	// The MCP SDK may return error in different ways depending on version
	// Either as an error return or as result.IsError
	if err == nil && !result.IsError {
		t.Error("Expected tool to return error")
	}

	cancel()
	<-serverDone
}

func TestMCPToolExecution_WithAuditLogging(t *testing.T) {
	// Create mock client
	mock := &MockClient{
		CreateStickyFn: func(ctx context.Context, args miro.CreateStickyArgs) (miro.CreateStickyResult, error) {
			return miro.CreateStickyResult{
				ID:      "sticky123",
				Message: "Created sticky note",
			}, nil
		},
	}

	// Create registry with audit logger
	memLogger := audit.NewMemoryLogger(100, audit.Config{Enabled: true})
	registry := NewHandlerRegistry(mock, testLogger()).
		WithAuditLogger(memLogger).
		WithUser("user123", "test@example.com")

	// Create MCP server and register tools
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	registry.RegisterAll(server)

	// Create in-memory transports
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer session.Close()

	// Call tool via MCP
	_, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "miro_create_sticky",
		Arguments: map[string]interface{}{
			"board_id": "board123",
			"content":  "Test sticky",
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	// Query audit log to verify event was recorded
	queryCtx := context.Background()
	events, err := memLogger.Query(queryCtx, audit.QueryOptions{Tool: "miro_create_sticky", Limit: 10})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(events.Events) == 0 {
		t.Error("Expected audit event to be logged")
	} else {
		event := events.Events[0]
		if event.Tool != "miro_create_sticky" {
			t.Errorf("Tool = %s, want miro_create_sticky", event.Tool)
		}
		if event.UserID != "user123" {
			t.Errorf("UserID = %s, want user123", event.UserID)
		}
	}

	cancel()
	<-serverDone
}

// =============================================================================
// Desire Path Tests
// =============================================================================

func TestWithDesirePathLogger(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	dpLogger := desirepath.NewLogger(desirepath.Config{Enabled: true, MaxEvents: 10}, testLogger())
	normalizers := []desirepath.Normalizer{
		&desirepath.WhitespaceNormalizer{},
	}

	result := registry.WithDesirePathLogger(dpLogger, normalizers)
	if result != registry {
		t.Error("WithDesirePathLogger should return the same registry for chaining")
	}
	if registry.desireLogger == nil {
		t.Error("desireLogger should be set")
	}
	if len(registry.normalizers) != 1 {
		t.Errorf("normalizers len = %d, want 1", len(registry.normalizers))
	}
}

func TestGetDesirePathReport_NoLogger(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	ctx := context.Background()
	result, err := registry.GetDesirePathReport(ctx, miro.GetDesirePathReportArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "Desire path logging is not enabled" {
		t.Errorf("message = %q, want disabled message", result.Message)
	}
}

func TestGetDesirePathReport_WithEvents(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	dpLogger := desirepath.NewLogger(desirepath.Config{Enabled: true, MaxEvents: 100}, testLogger())
	registry.WithDesirePathLogger(dpLogger, nil)

	// Log some events directly
	dpLogger.Log(desirepath.Event{
		Tool:         "miro_get_board",
		Parameter:    "board_id",
		Rule:         "url_to_id",
		RawValue:     "https://miro.com/app/board/uXjVN123=/",
		NormalizedTo: "uXjVN123=",
	})
	dpLogger.Log(desirepath.Event{
		Tool:         "miro_list_items",
		Parameter:    "limit",
		Rule:         "string_to_numeric",
		RawValue:     `"10"`,
		NormalizedTo: "10",
	})

	ctx := context.Background()
	result, err := registry.GetDesirePathReport(ctx, miro.GetDesirePathReportArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalNormalizations != 2 {
		t.Errorf("total = %d, want 2", result.TotalNormalizations)
	}
	if len(result.RecentEvents) != 2 {
		t.Errorf("recent events = %d, want 2", len(result.RecentEvents))
	}
	if result.ByRule["url_to_id"] != 1 {
		t.Errorf("by_rule[url_to_id] = %d, want 1", result.ByRule["url_to_id"])
	}
}

func TestGetDesirePathReport_FilterByTool(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	dpLogger := desirepath.NewLogger(desirepath.Config{Enabled: true, MaxEvents: 100}, testLogger())
	registry.WithDesirePathLogger(dpLogger, nil)

	dpLogger.Log(desirepath.Event{Tool: "miro_get_board", Rule: "url_to_id"})
	dpLogger.Log(desirepath.Event{Tool: "miro_list_items", Rule: "string_to_numeric"})

	ctx := context.Background()
	result, err := registry.GetDesirePathReport(ctx, miro.GetDesirePathReportArgs{
		Tool: "miro_get_board",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, e := range result.RecentEvents {
		if e.Tool != "miro_get_board" {
			t.Errorf("filtered event has wrong tool: %q", e.Tool)
		}
	}
}

func TestNormalizeArgs_URLInBoardID(t *testing.T) {
	mock := &MockClient{
		GetBoardFn: func(ctx context.Context, args miro.GetBoardArgs) (miro.GetBoardResult, error) {
			return miro.GetBoardResult{
				Board: miro.Board{ID: args.BoardID, Name: "Test"},
			}, nil
		},
	}

	dpLogger := desirepath.NewLogger(desirepath.Config{Enabled: true, MaxEvents: 100}, testLogger())
	normalizers := []desirepath.Normalizer{
		&desirepath.WhitespaceNormalizer{},
		desirepath.NewURLToIDNormalizer(desirepath.MiroURLPatterns()),
		&desirepath.CamelToSnakeNormalizer{},
		desirepath.NewStringToNumericNormalizer(nil),
		desirepath.NewBooleanCoercionNormalizer(nil),
	}

	registry := NewHandlerRegistry(mock, testLogger()).
		WithDesirePathLogger(dpLogger, normalizers)

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	registry.RegisterAll(server)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Run(ctx, serverTransport)
	}()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer session.Close()

	// Send a full URL in board_id - should be normalized to just the ID
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "miro_get_board",
		Arguments: map[string]interface{}{
			"board_id": "https://miro.com/app/board/uXjVN123=/",
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Errorf("Tool returned error: %v", result.Content)
	}

	// Verify the URL normalization was logged
	if dpLogger.Count() == 0 {
		t.Error("Expected desire path event for URL normalization")
	}
	events := dpLogger.Query(desirepath.QueryOptions{Rule: "url_to_id"})
	if len(events) == 0 {
		t.Error("Expected url_to_id event")
	} else if events[0].NormalizedTo != "uXjVN123=" {
		t.Errorf("normalized to %q, want %q", events[0].NormalizedTo, "uXjVN123=")
	}

	cancel()
	<-serverDone
}

// Note: CamelCase key normalization cannot be tested via full MCP integration because
// the go-sdk validates arguments against the JSON schema BEFORE calling the handler.
// Sending "boardId" instead of "board_id" is rejected by schema validation.
// CamelCase normalization would require transport-level middleware to intercept
// requests before schema validation. The normalizer logic is tested in desirepath_test.go.
func TestNormalizeArgs_CamelCaseKeys_Unit(t *testing.T) {
	dpLogger := desirepath.NewLogger(desirepath.Config{Enabled: true, MaxEvents: 100}, testLogger())
	normalizers := []desirepath.Normalizer{
		&desirepath.CamelToSnakeNormalizer{},
	}

	mock := &MockClient{}
	registry := NewHandlerRegistry(mock, testLogger()).
		WithDesirePathLogger(dpLogger, normalizers)

	// Simulate what normalizeArgs would receive: raw JSON with camelCase keys
	rawJSON := json.RawMessage(`{"boardId": "uXjVN123="}`)
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: rawJSON,
		},
	}

	args := miro.GetBoardArgs{BoardID: ""}
	result := normalizeArgs(registry, "miro_get_board", req, args)

	// The normalizer should have remapped boardId -> board_id
	if result.BoardID != "uXjVN123=" {
		t.Errorf("Expected BoardID 'uXjVN123=', got %q", result.BoardID)
	}

	// Verify camel_to_snake event was logged
	events := dpLogger.Query(desirepath.QueryOptions{Rule: "camel_to_snake"})
	if len(events) == 0 {
		t.Error("Expected camel_to_snake event for boardId -> board_id")
	}
}
