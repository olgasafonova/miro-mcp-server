package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/olgasafonova/miro-mcp-server/miro"
)

// =============================================================================
// Tag Handler Tests
// =============================================================================

func TestCreateTagHandler(t *testing.T) {
	result, err := (&MockClient{}).CreateTag(context.Background(), miro.CreateTagArgs{BoardID: "board123", Title: "Urgent", Color: "red"})
	mustSucceed(t, err)
	if result.Title != "Urgent" {
		t.Errorf("Title = %q, want 'Urgent'", result.Title)
	}
	if result.Color != "red" {
		t.Errorf("Color = %q, want 'red'", result.Color)
	}
}

func TestListTagsHandler(t *testing.T) {
	result, err := (&MockClient{}).ListTags(context.Background(), miro.ListTagsArgs{BoardID: "board123"})
	mustSucceed(t, err)
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
}

func TestAttachTagHandler(t *testing.T) {
	result, err := (&MockClient{}).AttachTag(context.Background(), miro.AttachTagArgs{BoardID: "board123", ItemID: "item456", TagID: "tag789"})
	mustSucceed(t, err)
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestDetachTagHandler(t *testing.T) {
	result, err := (&MockClient{}).DetachTag(context.Background(), miro.DetachTagArgs{BoardID: "board123", ItemID: "item456", TagID: "tag789"})
	mustSucceed(t, err)
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestGetTagHandler(t *testing.T) {
	result, err := (&MockClient{}).GetTag(context.Background(), miro.GetTagArgs{BoardID: "board123", TagID: "tag456"})
	mustSucceed(t, err)
	if result.ID != "tag456" {
		t.Errorf("ID = %q, want 'tag456'", result.ID)
	}
	if result.Title == "" {
		t.Error("Title should not be empty")
	}
}

func TestUpdateTagHandler(t *testing.T) {
	result, err := (&MockClient{}).UpdateTag(context.Background(), miro.UpdateTagArgs{
		BoardID: "board123",
		TagID:   "tag456",
		Title:   "Updated Title",
		Color:   "blue",
	})
	mustSucceed(t, err)
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Title != "Updated Title" {
		t.Errorf("Title = %q, want 'Updated Title'", result.Title)
	}
}

func TestDeleteTagHandler(t *testing.T) {
	result, err := (&MockClient{}).DeleteTag(context.Background(), miro.DeleteTagArgs{BoardID: "board123", TagID: "tag456"})
	mustSucceed(t, err)
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestGetItemTagsHandler(t *testing.T) {
	result, err := (&MockClient{}).GetItemTags(context.Background(), miro.GetItemTagsArgs{BoardID: "board123", ItemID: "item456"})
	mustSucceed(t, err)
	if result.Count == 0 {
		t.Error("expected at least one tag")
	}
	if result.ItemID != "item456" {
		t.Errorf("ItemID = %q, want 'item456'", result.ItemID)
	}
}

// =============================================================================
// Group Handler Tests
// =============================================================================

func TestCreateGroupHandler(t *testing.T) {
	result, err := (&MockClient{}).CreateGroup(context.Background(), miro.CreateGroupArgs{
		BoardID: "board123",
		ItemIDs: []string{"item1", "item2", "item3"},
	})
	mustSucceed(t, err)
	if len(result.ItemIDs) != 3 {
		t.Errorf("ItemIDs count = %d, want 3", len(result.ItemIDs))
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
			result, err := (&MockClient{}).CreateMindmapNode(context.Background(), tt.args)
			mustSucceed(t, err)
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
// Code Widget Handler Tests
// =============================================================================

func TestCodeWidgetCreate(t *testing.T) {
	mock := &MockClient{}
	result, err := mock.CreateCodeWidget(context.Background(), miro.CreateCodeWidgetArgs{
		BoardID:  "board123",
		Code:     `fmt.Println("hi")`,
		Language: "go",
		Title:    "Snippet",
	})
	mustSucceed(t, err)
	if result.ID == "" {
		t.Error("ID should not be empty")
	}
	if len(mock.Calls) != 1 || mock.Calls[0].Method != "CreateCodeWidget" {
		t.Errorf("expected one recorded CreateCodeWidget call, got %+v", mock.Calls)
	}
}

func TestCodeWidgetGetReturnsFullCode(t *testing.T) {
	result, err := (&MockClient{}).GetCodeWidget(context.Background(), miro.GetCodeWidgetArgs{BoardID: "board123", ItemID: "cw-1"})
	mustSucceed(t, err)
	if result.Code == "" {
		t.Error("Code should not be empty")
	}
}

func TestCodeWidgetListReturnsSummaries(t *testing.T) {
	result, err := (&MockClient{}).ListCodeWidgets(context.Background(), miro.ListCodeWidgetsArgs{BoardID: "board123"})
	mustSucceed(t, err)
	if result.Count != len(result.Widgets) {
		t.Errorf("Count = %d, widgets = %d", result.Count, len(result.Widgets))
	}
}

func TestCodeWidgetUpdate(t *testing.T) {
	result, err := (&MockClient{}).UpdateCodeWidget(context.Background(), miro.UpdateCodeWidgetArgs{BoardID: "board123", ItemID: "cw-1", Code: "updated"})
	mustSucceed(t, err)
	if result.ID != "cw-1" {
		t.Errorf("ID = %q, want 'cw-1'", result.ID)
	}
}

func TestCodeWidgetMoveEchoesCoordinates(t *testing.T) {
	result, err := (&MockClient{}).MoveCodeWidget(context.Background(), miro.MoveCodeWidgetArgs{BoardID: "board123", ItemID: "cw-1", X: 50, Y: -25})
	mustSucceed(t, err)
	if result.X != 50 || result.Y != -25 {
		t.Errorf("position = (%v, %v), want (50, -25)", result.X, result.Y)
	}
}

func TestCodeWidgetDeletePropagatesErrors(t *testing.T) {
	mock := &MockClient{
		DeleteCodeWidgetFn: func(ctx context.Context, args miro.DeleteCodeWidgetArgs) (miro.DeleteCodeWidgetResult, error) {
			return miro.DeleteCodeWidgetResult{Success: false, ID: args.ItemID}, errors.New("simulated API failure")
		},
	}
	_, err := mock.DeleteCodeWidget(context.Background(), miro.DeleteCodeWidgetArgs{BoardID: "board123", ItemID: "cw-1"})
	if err == nil {
		t.Fatal("expected error to propagate")
	}
}
