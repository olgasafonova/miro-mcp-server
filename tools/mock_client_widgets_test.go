package tools

import (
	"context"
	"fmt"

	"github.com/olgasafonova/miro-mcp-server/miro"
)

// =============================================================================
// MindmapService Implementation
// =============================================================================

func (m *MockClient) CreateMindmapNode(ctx context.Context, args miro.CreateMindmapNodeArgs) (miro.CreateMindmapNodeResult, error) {
	m.recordCall("CreateMindmapNode", args)
	return stub(ctx, m.CreateMindmapNodeFn, args, miro.CreateMindmapNodeResult{
		ID:       "mindmap-node-123",
		Content:  args.Content,
		ParentID: args.ParentID,
		Message:  fmt.Sprintf("Created mindmap node '%s'", truncateForTest(args.Content, 30)),
	})
}

func (m *MockClient) GetMindmapNode(ctx context.Context, args miro.GetMindmapNodeArgs) (miro.GetMindmapNodeResult, error) {
	m.recordCall("GetMindmapNode", args)
	return stub(ctx, m.GetMindmapNodeFn, args, miro.GetMindmapNodeResult{
		ID:       args.NodeID,
		Content:  "Test Node Content",
		NodeView: "text",
		IsRoot:   true,
		X:        100,
		Y:        100,
		Message:  "Retrieved mindmap node 'Test Node Content'",
	})
}

func (m *MockClient) ListMindmapNodes(ctx context.Context, args miro.ListMindmapNodesArgs) (miro.ListMindmapNodesResult, error) {
	m.recordCall("ListMindmapNodes", args)
	return stub(ctx, m.ListMindmapNodesFn, args, miro.ListMindmapNodesResult{
		Nodes: []miro.MindmapNodeSummary{
			{ID: "node-1", Content: "Root", IsRoot: true},
			{ID: "node-2", Content: "Child 1", ParentID: "node-1"},
		},
		Count:   2,
		HasMore: false,
		Message: "Found 2 mindmap nodes",
	})
}

func (m *MockClient) DeleteMindmapNode(ctx context.Context, args miro.DeleteMindmapNodeArgs) (miro.DeleteMindmapNodeResult, error) {
	m.recordCall("DeleteMindmapNode", args)
	return stub(ctx, m.DeleteMindmapNodeFn, args, miro.DeleteMindmapNodeResult{
		Success: true,
		ID:      args.NodeID,
		Message: "Mindmap node deleted successfully",
	})
}

// =============================================================================
// CommentService Implementation
// =============================================================================

func (m *MockClient) CreateComment(ctx context.Context, args miro.CreateCommentArgs) (miro.CreateCommentResult, error) {
	m.recordCall("CreateComment", args)
	return stub(ctx, m.CreateCommentFn, args, miro.CreateCommentResult{
		ID:      "comment-123",
		ItemID:  args.ItemID,
		Message: "Created comment thread",
	})
}

func (m *MockClient) GetOrgAuditLogs(ctx context.Context, args miro.GetOrgAuditLogsArgs) (miro.GetOrgAuditLogsResult, error) {
	m.recordCall("GetOrgAuditLogs", args)
	return stub(ctx, m.GetOrgAuditLogsFn, args, miro.GetOrgAuditLogsResult{
		Events: []miro.OrgAuditEvent{{ID: "audit-123", Event: "board_created", Category: "boards"}},
		Count:  1,
	})
}

func (m *MockClient) ListComments(ctx context.Context, args miro.ListCommentsArgs) (miro.ListCommentsResult, error) {
	m.recordCall("ListComments", args)
	return stub(ctx, m.ListCommentsFn, args, miro.ListCommentsResult{
		Comments: []miro.CommentSummary{{ID: "comment-123", Messages: []miro.CommentMessage{{ID: "m1", Content: "hello"}}}},
		Count:    1,
		Total:    1,
	})
}

func (m *MockClient) GetComment(ctx context.Context, args miro.GetCommentArgs) (miro.GetCommentResult, error) {
	m.recordCall("GetComment", args)
	return stub(ctx, m.GetCommentFn, args, miro.GetCommentResult{
		CommentSummary: miro.CommentSummary{ID: args.CommentID, Messages: []miro.CommentMessage{{ID: "m1", Content: "hello"}}},
		Message:        "Comment thread with 1 message(s)",
	})
}

func (m *MockClient) ReplyComment(ctx context.Context, args miro.ReplyCommentArgs) (miro.ReplyCommentResult, error) {
	m.recordCall("ReplyComment", args)
	return stub(ctx, m.ReplyCommentFn, args, miro.ReplyCommentResult{
		ID:           args.CommentID,
		MessageCount: 2,
		Message:      "Replied; thread now has 2 message(s)",
	})
}

func (m *MockClient) ResolveComment(ctx context.Context, args miro.ResolveCommentArgs) (miro.ResolveCommentResult, error) {
	m.recordCall("ResolveComment", args)
	resolved := true
	if args.Resolved != nil {
		resolved = *args.Resolved
	}
	return stub(ctx, m.ResolveCommentFn, args, miro.ResolveCommentResult{
		ID:       args.CommentID,
		Resolved: resolved,
		Message:  "Resolved comment thread " + args.CommentID,
	})
}

// =============================================================================
// SVGService Implementation
// =============================================================================

func (m *MockClient) ReadBoardSVG(ctx context.Context, args miro.ReadBoardSVGArgs) (miro.ReadBoardSVGResult, error) {
	m.recordCall("ReadBoardSVG", args)
	return stub(ctx, m.ReadBoardSVGFn, args, miro.ReadBoardSVGResult{
		SVG:       `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"></svg>`,
		ItemCount: 0,
		Message:   "Rendered 0 item(s) as SVG (0 skipped)",
	})
}

func (m *MockClient) CreateFromSVG(ctx context.Context, args miro.CreateFromSVGArgs) (miro.CreateFromSVGResult, error) {
	m.recordCall("CreateFromSVG", args)
	return stub(ctx, m.CreateFromSVGFn, args, miro.CreateFromSVGResult{
		Created: []miro.SVGCreatedItem{{ID: "shape-1", Type: "shape", Element: "rect"}},
		Count:   1,
		Message: "Created 1 item(s) from SVG (0 element(s) skipped)",
	})
}

func (m *MockClient) UpdateFromSVG(ctx context.Context, args miro.UpdateFromSVGArgs) (miro.UpdateFromSVGResult, error) {
	m.recordCall("UpdateFromSVG", args)
	return stub(ctx, m.UpdateFromSVGFn, args, miro.UpdateFromSVGResult{
		Updated: []miro.SVGUpdatedItem{{ID: "item-1", Element: "rect"}},
		Message: "Updated 1, deleted 0, created 0 item(s) (0 failed, 0 skipped)",
	})
}

// =============================================================================
// CodeWidgetService Implementation
// =============================================================================

func (m *MockClient) CreateCodeWidget(ctx context.Context, args miro.CreateCodeWidgetArgs) (miro.CreateCodeWidgetResult, error) {
	m.recordCall("CreateCodeWidget", args)
	return stub(ctx, m.CreateCodeWidgetFn, args, miro.CreateCodeWidgetResult{
		ID:       "code-widget-123",
		Title:    args.Title,
		Language: args.Language,
		Message:  "Created code widget 'test'",
	})
}

func (m *MockClient) GetCodeWidget(ctx context.Context, args miro.GetCodeWidgetArgs) (miro.GetCodeWidgetResult, error) {
	m.recordCall("GetCodeWidget", args)
	return stub(ctx, m.GetCodeWidgetFn, args, miro.GetCodeWidgetResult{
		ID:       args.ItemID,
		Code:     `console.log("hello");`,
		Language: "javascript",
		Title:    "Test Snippet",
		X:        100,
		Y:        200,
		Message:  fmt.Sprintf("Retrieved code widget %s", args.ItemID),
	})
}

func (m *MockClient) ListCodeWidgets(ctx context.Context, args miro.ListCodeWidgetsArgs) (miro.ListCodeWidgetsResult, error) {
	m.recordCall("ListCodeWidgets", args)
	return stub(ctx, m.ListCodeWidgetsFn, args, miro.ListCodeWidgetsResult{
		Widgets: []miro.CodeWidgetSummary{
			{ID: "cw-1", Title: "Snippet A", Language: "go", CodePreview: "package main"},
			{ID: "cw-2", Title: "Snippet B", Language: "python", CodePreview: "print('hi')"},
		},
		Count:   2,
		HasMore: false,
		Message: "Found 2 code widgets",
	})
}

func (m *MockClient) UpdateCodeWidget(ctx context.Context, args miro.UpdateCodeWidgetArgs) (miro.UpdateCodeWidgetResult, error) {
	m.recordCall("UpdateCodeWidget", args)
	return stub(ctx, m.UpdateCodeWidgetFn, args, miro.UpdateCodeWidgetResult{
		ID:      args.ItemID,
		Message: "Code widget updated successfully",
	})
}

func (m *MockClient) MoveCodeWidget(ctx context.Context, args miro.MoveCodeWidgetArgs) (miro.MoveCodeWidgetResult, error) {
	m.recordCall("MoveCodeWidget", args)
	return stub(ctx, m.MoveCodeWidgetFn, args, miro.MoveCodeWidgetResult{
		ID:      args.ItemID,
		X:       args.X,
		Y:       args.Y,
		Message: "Moved code widget to (100, 200)",
	})
}

func (m *MockClient) DeleteCodeWidget(ctx context.Context, args miro.DeleteCodeWidgetArgs) (miro.DeleteCodeWidgetResult, error) {
	m.recordCall("DeleteCodeWidget", args)
	return stub(ctx, m.DeleteCodeWidgetFn, args, miro.DeleteCodeWidgetResult{
		Success: true,
		ID:      args.ItemID,
		Message: "Code widget deleted successfully",
	})
}

// =============================================================================
// FrameService Implementation (beyond create)
// =============================================================================

func (m *MockClient) GetFrame(ctx context.Context, args miro.GetFrameArgs) (miro.GetFrameResult, error) {
	m.recordCall("GetFrame", args)
	return stub(ctx, m.GetFrameFn, args, miro.GetFrameResult{
		ID:         args.FrameID,
		Title:      "Test Frame",
		X:          0,
		Y:          0,
		Width:      800,
		Height:     600,
		ChildCount: 5,
		Message:    "Retrieved frame 'Test Frame'",
	})
}

func (m *MockClient) UpdateFrame(ctx context.Context, args miro.UpdateFrameArgs) (miro.UpdateFrameResult, error) {
	m.recordCall("UpdateFrame", args)
	return stub(ctx, m.UpdateFrameFn, args, miro.UpdateFrameResult{
		Success: true,
		ID:      args.FrameID,
		Message: "Frame updated successfully",
	})
}

func (m *MockClient) DeleteFrame(ctx context.Context, args miro.DeleteFrameArgs) (miro.DeleteFrameResult, error) {
	m.recordCall("DeleteFrame", args)
	return stub(ctx, m.DeleteFrameFn, args, miro.DeleteFrameResult{
		Success: true,
		ID:      args.FrameID,
		Message: "Frame deleted successfully",
	})
}

func (m *MockClient) GetFrameItems(ctx context.Context, args miro.GetFrameItemsArgs) (miro.GetFrameItemsResult, error) {
	m.recordCall("GetFrameItems", args)
	return stub(ctx, m.GetFrameItemsFn, args, miro.GetFrameItemsResult{
		Items: []miro.ItemSummary{
			{ID: "item-1", Type: "sticky_note", Content: "Test sticky"},
			{ID: "item-2", Type: "shape", Content: "Test shape"},
		},
		Count:   2,
		HasMore: false,
		Message: "Found 2 items in frame",
	})
}
