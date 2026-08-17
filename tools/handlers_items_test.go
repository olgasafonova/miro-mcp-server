package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/olgasafonova/miro-mcp-server/miro"
)

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

	result, err := mock.ListItems(context.Background(), miro.ListItemsArgs{
		BoardID: "board123",
		Type:    "sticky_note",
	})
	mustSucceed(t, err)
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

	result, err := mock.GetItem(context.Background(), miro.GetItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
	})
	mustSucceed(t, err)
	if result.ID != "item456" {
		t.Errorf("ID = %q, want 'item456'", result.ID)
	}
}

func TestSearchBoardHandler(t *testing.T) {
	result, err := (&MockClient{}).SearchBoard(context.Background(), miro.SearchBoardArgs{BoardID: "board123", Query: "budget"})
	mustSucceed(t, err)
	if result.Query != "budget" {
		t.Errorf("Query = %q, want 'budget'", result.Query)
	}
	if result.Count == 0 {
		t.Error("expected at least one result")
	}
}

func TestDeleteItemHandler(t *testing.T) {
	result, err := (&MockClient{}).DeleteItem(context.Background(), miro.DeleteItemArgs{BoardID: "board123", ItemID: "item456"})
	mustSucceed(t, err)
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.ItemID != "item456" {
		t.Errorf("ItemID = %q, want 'item456'", result.ItemID)
	}
}

func TestUpdateItemHandler(t *testing.T) {
	result, err := (&MockClient{}).UpdateItem(context.Background(), miro.UpdateItemArgs{
		BoardID: "board123",
		ItemID:  "item456",
		Content: strPtr("Updated content"),
	})
	mustSucceed(t, err)
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Message != "Item updated successfully" {
		t.Errorf("Message = %q, want 'Item updated successfully'", result.Message)
	}
}

func TestListAllItemsHandler(t *testing.T) {
	result, err := (&MockClient{}).ListAllItems(context.Background(), miro.ListAllItemsArgs{BoardID: "board123", MaxItems: 100})
	mustSucceed(t, err)
	if result.Count == 0 {
		t.Error("expected non-zero Count")
	}
}

// =============================================================================
// Create Handler Tests
// =============================================================================

func TestCreateStickyHandler(t *testing.T) {
	result, err := (&MockClient{}).CreateSticky(context.Background(), miro.CreateStickyArgs{
		BoardID: "board123",
		Content: "Action item: Review PRs",
		Color:   "yellow",
	})
	mustSucceed(t, err)
	if result.ID == "" {
		t.Error("ID should not be empty")
	}
	if result.Content != "Action item: Review PRs" {
		t.Errorf("Content = %q, want 'Action item: Review PRs'", result.Content)
	}
}

func TestCreateShapeHandler(t *testing.T) {
	result, err := (&MockClient{}).CreateShape(context.Background(), miro.CreateShapeArgs{BoardID: "board123", Shape: "rectangle", Content: "Header"})
	mustSucceed(t, err)
	if result.Shape != "rectangle" {
		t.Errorf("Shape = %q, want 'rectangle'", result.Shape)
	}
}

func TestCreateConnectorHandler(t *testing.T) {
	result, err := (&MockClient{}).CreateConnector(context.Background(), miro.CreateConnectorArgs{BoardID: "board123", StartItemID: "item1", EndItemID: "item2"})
	mustSucceed(t, err)
	if result.ID == "" {
		t.Error("ID should not be empty")
	}
	if result.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestCreateFrameHandler(t *testing.T) {
	result, err := (&MockClient{}).CreateFrame(context.Background(), miro.CreateFrameArgs{BoardID: "board123", Title: "Brainstorming"})
	mustSucceed(t, err)
	if result.Title != "Brainstorming" {
		t.Errorf("Title = %q, want 'Brainstorming'", result.Title)
	}
}

func TestCreateStickyGridHandler(t *testing.T) {
	result, err := (&MockClient{}).CreateStickyGrid(context.Background(), miro.CreateStickyGridArgs{
		BoardID:  "board123",
		Contents: []string{"A", "B", "C", "D", "E", "F"},
		Columns:  3,
		Color:    "yellow",
	})
	mustSucceed(t, err)
	if result.Created != 6 {
		t.Errorf("Created = %d, want 6", result.Created)
	}
	if result.Columns != 3 {
		t.Errorf("Columns = %d, want 3", result.Columns)
	}
}

// =============================================================================
// Bulk Handler Tests
// =============================================================================

func TestBulkCreateHandler(t *testing.T) {
	result, err := (&MockClient{}).BulkCreate(context.Background(), miro.BulkCreateArgs{
		BoardID: "board123",
		Items: []miro.BulkCreateItem{
			{Type: "sticky_note", Content: "Task 1"},
			{Type: "sticky_note", Content: "Task 2"},
			{Type: "shape", Content: "Box", Shape: "rectangle"},
		},
	})
	mustSucceed(t, err)
	if result.Created != 3 {
		t.Errorf("Created = %d, want 3", result.Created)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %d", len(result.Errors))
	}
}

func TestBulkUpdateHandler(t *testing.T) {
	content := "Updated content"
	x := 100.0
	result, err := (&MockClient{}).BulkUpdate(context.Background(), miro.BulkUpdateArgs{
		BoardID: "board123",
		Items: []miro.BulkUpdateItem{
			{ItemID: "item1", Content: &content},
			{ItemID: "item2", X: &x},
			{ItemID: "item3", Content: &content, X: &x},
		},
	})
	mustSucceed(t, err)
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

	content := "Updated"
	result, err := mock.BulkUpdate(context.Background(), miro.BulkUpdateArgs{
		BoardID: "board123",
		Items: []miro.BulkUpdateItem{
			{ItemID: "item1", Content: &content},
			{ItemID: "item2", Content: &content},
			{ItemID: "item3", Content: &content},
		},
	})
	mustSucceed(t, err)
	if result.Updated != 2 {
		t.Errorf("Updated = %d, want 2", result.Updated)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestBulkDeleteHandler(t *testing.T) {
	result, err := (&MockClient{}).BulkDelete(context.Background(), miro.BulkDeleteArgs{
		BoardID: "board123",
		ItemIDs: []string{"item1", "item2", "item3"},
	})
	mustSucceed(t, err)
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

	_, err := mock.BulkDelete(context.Background(), miro.BulkDeleteArgs{
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

// =============================================================================
// Parameter Validation Edge Case Tests
// =============================================================================

// rejecting returns a mock override that fails with msg when reject reports
// the args as invalid, and returns ok otherwise. It stands in for the
// server-side required-field validation the cases below exercise.
func rejecting[A, R any](ok R, reject func(A) bool, msg string) func(context.Context, A) (R, error) {
	return func(_ context.Context, args A) (R, error) {
		if reject(args) {
			var zero R
			return zero, errors.New(msg)
		}
		return ok, nil
	}
}

// parameterValidationCases enumerates required-field and format checks, one
// case per rejected input. Each case wires a rejecting override into the mock
// and invokes the handler with the offending arguments.
var parameterValidationCases = []struct {
	name    string
	wantErr string
	call    func(ctx context.Context) error
}{
	{
		name:    "create sticky empty board_id",
		wantErr: "board_id is required",
		call: func(ctx context.Context) error {
			mock := &MockClient{CreateStickyFn: rejecting(miro.CreateStickyResult{ID: "test"},
				func(a miro.CreateStickyArgs) bool { return a.BoardID == "" }, "board_id is required")}
			_, err := mock.CreateSticky(ctx, miro.CreateStickyArgs{BoardID: "", Content: "Test"})
			return err
		},
	},
	{
		name:    "create sticky empty content",
		wantErr: "content is required",
		call: func(ctx context.Context) error {
			mock := &MockClient{CreateStickyFn: rejecting(miro.CreateStickyResult{ID: "test"},
				func(a miro.CreateStickyArgs) bool { return a.Content == "" }, "content is required")}
			_, err := mock.CreateSticky(ctx, miro.CreateStickyArgs{BoardID: "board123", Content: ""})
			return err
		},
	},
	{
		name:    "update item empty item_id",
		wantErr: "item_id is required",
		call: func(ctx context.Context) error {
			mock := &MockClient{UpdateItemFn: rejecting(miro.UpdateItemResult{Success: true},
				func(a miro.UpdateItemArgs) bool { return a.ItemID == "" }, "item_id is required")}
			_, err := mock.UpdateItem(ctx, miro.UpdateItemArgs{BoardID: "board123", ItemID: ""})
			return err
		},
	},
	{
		name:    "search board empty query",
		wantErr: "query is required",
		call: func(ctx context.Context) error {
			mock := &MockClient{SearchBoardFn: rejecting(miro.SearchBoardResult{Count: 1},
				func(a miro.SearchBoardArgs) bool { return a.Query == "" }, "query is required")}
			_, err := mock.SearchBoard(ctx, miro.SearchBoardArgs{BoardID: "board123", Query: ""})
			return err
		},
	},
	{
		name:    "bulk create empty items",
		wantErr: "at least one item is required",
		call: func(ctx context.Context) error {
			mock := &MockClient{BulkCreateFn: rejecting(miro.BulkCreateResult{},
				func(a miro.BulkCreateArgs) bool { return len(a.Items) == 0 }, "at least one item is required")}
			_, err := mock.BulkCreate(ctx, miro.BulkCreateArgs{BoardID: "board123", Items: []miro.BulkCreateItem{}})
			return err
		},
	},
	{
		name:    "bulk delete empty item_ids",
		wantErr: "at least one item_id is required",
		call: func(ctx context.Context) error {
			mock := &MockClient{BulkDeleteFn: rejecting(miro.BulkDeleteResult{},
				func(a miro.BulkDeleteArgs) bool { return len(a.ItemIDs) == 0 }, "at least one item_id is required")}
			_, err := mock.BulkDelete(ctx, miro.BulkDeleteArgs{BoardID: "board123", ItemIDs: []string{}})
			return err
		},
	},
	{
		name:    "create group insufficient items",
		wantErr: "at least 2 item_ids required",
		call: func(ctx context.Context) error {
			mock := &MockClient{CreateGroupFn: rejecting(miro.CreateGroupResult{ID: "group123"},
				func(a miro.CreateGroupArgs) bool { return len(a.ItemIDs) < 2 }, "at least 2 item_ids required")}
			// Only 1 item, need at least 2.
			_, err := mock.CreateGroup(ctx, miro.CreateGroupArgs{BoardID: "board123", ItemIDs: []string{"item1"}})
			return err
		},
	},
	{
		name:    "create shape invalid shape type",
		wantErr: "invalid shape type",
		call: func(ctx context.Context) error {
			validShapes := map[string]bool{
				"rectangle": true, "circle": true, "triangle": true,
				"rhombus": true, "round_rectangle": true,
			}
			mock := &MockClient{CreateShapeFn: rejecting(miro.CreateShapeResult{ID: "shape123"},
				func(a miro.CreateShapeArgs) bool { return !validShapes[a.Shape] }, "invalid shape type")}
			_, err := mock.CreateShape(ctx, miro.CreateShapeArgs{BoardID: "board123", Shape: "invalid_shape"})
			return err
		},
	},
	{
		name:    "create sticky grid empty contents",
		wantErr: "at least one content item is required",
		call: func(ctx context.Context) error {
			mock := &MockClient{CreateStickyGridFn: rejecting(miro.CreateStickyGridResult{},
				func(a miro.CreateStickyGridArgs) bool { return len(a.Contents) == 0 }, "at least one content item is required")}
			_, err := mock.CreateStickyGrid(ctx, miro.CreateStickyGridArgs{BoardID: "board123", Contents: []string{}})
			return err
		},
	},
	{
		name:    "share board empty email",
		wantErr: "email is required",
		call: func(ctx context.Context) error {
			mock := &MockClient{ShareBoardFn: rejecting(miro.ShareBoardResult{Success: true},
				func(a miro.ShareBoardArgs) bool { return a.Email == "" }, "email is required")}
			_, err := mock.ShareBoard(ctx, miro.ShareBoardArgs{BoardID: "board123", Email: ""})
			return err
		},
	},
	{
		name:    "share board invalid email format",
		wantErr: "invalid email format",
		call: func(ctx context.Context) error {
			// Basic email format check.
			mock := &MockClient{ShareBoardFn: rejecting(miro.ShareBoardResult{Success: true},
				func(a miro.ShareBoardArgs) bool { return !strings.Contains(a.Email, "@") }, "invalid email format")}
			_, err := mock.ShareBoard(ctx, miro.ShareBoardArgs{BoardID: "board123", Email: "not-an-email"})
			return err
		},
	},
	{
		name:    "create connector missing start_item_id",
		wantErr: "start_item_id is required",
		call: func(ctx context.Context) error {
			mock := &MockClient{CreateConnectorFn: rejecting(miro.CreateConnectorResult{ID: "conn123"},
				func(a miro.CreateConnectorArgs) bool { return a.StartItemID == "" }, "start_item_id is required")}
			_, err := mock.CreateConnector(ctx, miro.CreateConnectorArgs{BoardID: "board123", StartItemID: "", EndItemID: "item2"})
			return err
		},
	},
	{
		name:    "create connector missing end_item_id",
		wantErr: "end_item_id is required",
		call: func(ctx context.Context) error {
			mock := &MockClient{CreateConnectorFn: rejecting(miro.CreateConnectorResult{ID: "conn123"},
				func(a miro.CreateConnectorArgs) bool { return a.EndItemID == "" }, "end_item_id is required")}
			_, err := mock.CreateConnector(ctx, miro.CreateConnectorArgs{BoardID: "board123", StartItemID: "item1", EndItemID: ""})
			return err
		},
	},
	{
		name:    "delete item missing board_id",
		wantErr: "board_id is required",
		call: func(ctx context.Context) error {
			mock := &MockClient{DeleteItemFn: rejecting(miro.DeleteItemResult{Success: true},
				func(a miro.DeleteItemArgs) bool { return a.BoardID == "" }, "board_id is required")}
			_, err := mock.DeleteItem(ctx, miro.DeleteItemArgs{BoardID: "", ItemID: "item123"})
			return err
		},
	},
	{
		name:    "delete item missing item_id",
		wantErr: "item_id is required",
		call: func(ctx context.Context) error {
			mock := &MockClient{DeleteItemFn: rejecting(miro.DeleteItemResult{Success: true},
				func(a miro.DeleteItemArgs) bool { return a.ItemID == "" }, "item_id is required")}
			_, err := mock.DeleteItem(ctx, miro.DeleteItemArgs{BoardID: "board123", ItemID: ""})
			return err
		},
	},
}

func TestHandlerParameterValidation(t *testing.T) {
	for _, tt := range parameterValidationCases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(context.Background())
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}
