package tools

import (
	"context"
	"fmt"

	"github.com/olgasafonova/miro-mcp-server/miro"
)

// =============================================================================
// Mock Client for Testing
// =============================================================================
//
// The MockClient implementation is split by service area, mirroring the
// MiroClient interface. The struct definition, call tracking, and the shared
// stub dispatch helper live here; the method groups live in:
//
//   - mock_client_items_test.go   — item read/update/delete/bulk operations
//   - mock_client_create_test.go  — create + upload operations
//   - mock_client_tags_test.go    — tag, connector, and group operations
//   - mock_client_widgets_test.go — mindmap, comment, SVG, code widget, frame
//   - mock_client_misc_test.go    — export, diagram, app card, doc format, table

// MockClient implements miro.MiroClient for testing handlers without API calls.
// Each method can be configured with custom behavior via function fields.
type MockClient struct {
	// Board operations
	ListBoardsFn          func(ctx context.Context, args miro.ListBoardsArgs) (miro.ListBoardsResult, error)
	GetBoardFn            func(ctx context.Context, args miro.GetBoardArgs) (miro.GetBoardResult, error)
	CreateBoardFn         func(ctx context.Context, args miro.CreateBoardArgs) (miro.CreateBoardResult, error)
	CopyBoardFn           func(ctx context.Context, args miro.CopyBoardArgs) (miro.CopyBoardResult, error)
	DeleteBoardFn         func(ctx context.Context, args miro.DeleteBoardArgs) (miro.DeleteBoardResult, error)
	UpdateBoardFn         func(ctx context.Context, args miro.UpdateBoardArgs) (miro.UpdateBoardResult, error)
	FindBoardByNameFn     func(ctx context.Context, name string) (*miro.BoardSummary, error)
	FindBoardByNameToolFn func(ctx context.Context, args miro.FindBoardByNameArgs) (miro.FindBoardByNameResult, error)
	GetBoardSummaryFn     func(ctx context.Context, args miro.GetBoardSummaryArgs) (miro.GetBoardSummaryResult, error)
	GetBoardContentFn     func(ctx context.Context, args miro.GetBoardContentArgs) (miro.GetBoardContentResult, error)

	// Item operations
	ListItemsFn    func(ctx context.Context, args miro.ListItemsArgs) (miro.ListItemsResult, error)
	ListAllItemsFn func(ctx context.Context, args miro.ListAllItemsArgs) (miro.ListAllItemsResult, error)
	GetItemFn      func(ctx context.Context, args miro.GetItemArgs) (miro.GetItemResult, error)
	UpdateItemFn   func(ctx context.Context, args miro.UpdateItemArgs) (miro.UpdateItemResult, error)
	DeleteItemFn   func(ctx context.Context, args miro.DeleteItemArgs) (miro.DeleteItemResult, error)
	SearchBoardFn  func(ctx context.Context, args miro.SearchBoardArgs) (miro.SearchBoardResult, error)
	BulkCreateFn   func(ctx context.Context, args miro.BulkCreateArgs) (miro.BulkCreateResult, error)
	BulkUpdateFn   func(ctx context.Context, args miro.BulkUpdateArgs) (miro.BulkUpdateResult, error)
	BulkDeleteFn   func(ctx context.Context, args miro.BulkDeleteArgs) (miro.BulkDeleteResult, error)
	// Type-specific reads
	GetImageFn    func(ctx context.Context, args miro.GetImageArgs) (miro.GetImageResult, error)
	GetDocumentFn func(ctx context.Context, args miro.GetDocumentArgs) (miro.GetDocumentResult, error)
	// Type-specific updates
	UpdateStickyFn   func(ctx context.Context, args miro.UpdateStickyArgs) (miro.UpdateStickyResult, error)
	UpdateShapeFn    func(ctx context.Context, args miro.UpdateShapeArgs) (miro.UpdateShapeResult, error)
	UpdateTextFn     func(ctx context.Context, args miro.UpdateTextArgs) (miro.UpdateTextResult, error)
	UpdateCardFn     func(ctx context.Context, args miro.UpdateCardArgs) (miro.UpdateCardResult, error)
	UpdateImageFn    func(ctx context.Context, args miro.UpdateImageArgs) (miro.UpdateImageResult, error)
	UpdateDocumentFn func(ctx context.Context, args miro.UpdateDocumentArgs) (miro.UpdateDocumentResult, error)
	UpdateEmbedFn    func(ctx context.Context, args miro.UpdateEmbedArgs) (miro.UpdateEmbedResult, error)

	// Create operations
	CreateStickyFn            func(ctx context.Context, args miro.CreateStickyArgs) (miro.CreateStickyResult, error)
	CreateShapeFn             func(ctx context.Context, args miro.CreateShapeArgs) (miro.CreateShapeResult, error)
	CreateShapeExperimentalFn func(ctx context.Context, args miro.CreateShapeExperimentalArgs) (miro.CreateShapeResult, error)
	CreateFlowchartShapeFn    func(ctx context.Context, args miro.CreateFlowchartShapeArgs) (miro.CreateShapeResult, error)
	CreateTextFn              func(ctx context.Context, args miro.CreateTextArgs) (miro.CreateTextResult, error)
	CreateConnectorFn         func(ctx context.Context, args miro.CreateConnectorArgs) (miro.CreateConnectorResult, error)
	CreateFrameFn             func(ctx context.Context, args miro.CreateFrameArgs) (miro.CreateFrameResult, error)
	CreateCardFn              func(ctx context.Context, args miro.CreateCardArgs) (miro.CreateCardResult, error)
	CreateImageFn             func(ctx context.Context, args miro.CreateImageArgs) (miro.CreateImageResult, error)
	CreateDocumentFn          func(ctx context.Context, args miro.CreateDocumentArgs) (miro.CreateDocumentResult, error)
	CreateEmbedFn             func(ctx context.Context, args miro.CreateEmbedArgs) (miro.CreateEmbedResult, error)
	CreateStickyGridFn        func(ctx context.Context, args miro.CreateStickyGridArgs) (miro.CreateStickyGridResult, error)

	// Tag operations
	CreateTagFn     func(ctx context.Context, args miro.CreateTagArgs) (miro.CreateTagResult, error)
	ListTagsFn      func(ctx context.Context, args miro.ListTagsArgs) (miro.ListTagsResult, error)
	AttachTagFn     func(ctx context.Context, args miro.AttachTagArgs) (miro.AttachTagResult, error)
	DetachTagFn     func(ctx context.Context, args miro.DetachTagArgs) (miro.DetachTagResult, error)
	GetItemTagsFn   func(ctx context.Context, args miro.GetItemTagsArgs) (miro.GetItemTagsResult, error)
	GetItemsByTagFn func(ctx context.Context, args miro.GetItemsByTagArgs) (miro.GetItemsByTagResult, error)
	GetTagFn        func(ctx context.Context, args miro.GetTagArgs) (miro.GetTagResult, error)
	UpdateTagFn     func(ctx context.Context, args miro.UpdateTagArgs) (miro.UpdateTagResult, error)
	DeleteTagFn     func(ctx context.Context, args miro.DeleteTagArgs) (miro.DeleteTagResult, error)

	// Connector operations
	ListConnectorsFn  func(ctx context.Context, args miro.ListConnectorsArgs) (miro.ListConnectorsResult, error)
	GetConnectorFn    func(ctx context.Context, args miro.GetConnectorArgs) (miro.GetConnectorResult, error)
	UpdateConnectorFn func(ctx context.Context, args miro.UpdateConnectorArgs) (miro.UpdateConnectorResult, error)
	DeleteConnectorFn func(ctx context.Context, args miro.DeleteConnectorArgs) (miro.DeleteConnectorResult, error)

	// Group operations
	CreateGroupFn   func(ctx context.Context, args miro.CreateGroupArgs) (miro.CreateGroupResult, error)
	ListGroupsFn    func(ctx context.Context, args miro.ListGroupsArgs) (miro.ListGroupsResult, error)
	GetGroupFn      func(ctx context.Context, args miro.GetGroupArgs) (miro.GetGroupResult, error)
	GetGroupItemsFn func(ctx context.Context, args miro.GetGroupItemsArgs) (miro.GetGroupItemsResult, error)
	UpdateGroupFn   func(ctx context.Context, args miro.UpdateGroupArgs) (miro.UpdateGroupResult, error)
	DeleteGroupFn   func(ctx context.Context, args miro.DeleteGroupArgs) (miro.DeleteGroupResult, error)

	// Member operations
	ListBoardMembersFn  func(ctx context.Context, args miro.ListBoardMembersArgs) (miro.ListBoardMembersResult, error)
	ShareBoardFn        func(ctx context.Context, args miro.ShareBoardArgs) (miro.ShareBoardResult, error)
	GetBoardMemberFn    func(ctx context.Context, args miro.GetBoardMemberArgs) (miro.GetBoardMemberResult, error)
	RemoveBoardMemberFn func(ctx context.Context, args miro.RemoveBoardMemberArgs) (miro.RemoveBoardMemberResult, error)
	UpdateBoardMemberFn func(ctx context.Context, args miro.UpdateBoardMemberArgs) (miro.UpdateBoardMemberResult, error)

	// Mindmap operations
	CreateMindmapNodeFn func(ctx context.Context, args miro.CreateMindmapNodeArgs) (miro.CreateMindmapNodeResult, error)
	GetMindmapNodeFn    func(ctx context.Context, args miro.GetMindmapNodeArgs) (miro.GetMindmapNodeResult, error)
	ListMindmapNodesFn  func(ctx context.Context, args miro.ListMindmapNodesArgs) (miro.ListMindmapNodesResult, error)
	DeleteMindmapNodeFn func(ctx context.Context, args miro.DeleteMindmapNodeArgs) (miro.DeleteMindmapNodeResult, error)

	// Code widget operations
	CreateCommentFn  func(ctx context.Context, args miro.CreateCommentArgs) (miro.CreateCommentResult, error)
	ListCommentsFn   func(ctx context.Context, args miro.ListCommentsArgs) (miro.ListCommentsResult, error)
	GetCommentFn     func(ctx context.Context, args miro.GetCommentArgs) (miro.GetCommentResult, error)
	ReplyCommentFn   func(ctx context.Context, args miro.ReplyCommentArgs) (miro.ReplyCommentResult, error)
	ResolveCommentFn func(ctx context.Context, args miro.ResolveCommentArgs) (miro.ResolveCommentResult, error)
	ReadBoardSVGFn   func(ctx context.Context, args miro.ReadBoardSVGArgs) (miro.ReadBoardSVGResult, error)
	CreateFromSVGFn  func(ctx context.Context, args miro.CreateFromSVGArgs) (miro.CreateFromSVGResult, error)

	GetOrgAuditLogsFn func(ctx context.Context, args miro.GetOrgAuditLogsArgs) (miro.GetOrgAuditLogsResult, error)

	CreateCodeWidgetFn func(ctx context.Context, args miro.CreateCodeWidgetArgs) (miro.CreateCodeWidgetResult, error)
	GetCodeWidgetFn    func(ctx context.Context, args miro.GetCodeWidgetArgs) (miro.GetCodeWidgetResult, error)
	ListCodeWidgetsFn  func(ctx context.Context, args miro.ListCodeWidgetsArgs) (miro.ListCodeWidgetsResult, error)
	UpdateCodeWidgetFn func(ctx context.Context, args miro.UpdateCodeWidgetArgs) (miro.UpdateCodeWidgetResult, error)
	MoveCodeWidgetFn   func(ctx context.Context, args miro.MoveCodeWidgetArgs) (miro.MoveCodeWidgetResult, error)
	DeleteCodeWidgetFn func(ctx context.Context, args miro.DeleteCodeWidgetArgs) (miro.DeleteCodeWidgetResult, error)

	// Frame operations (beyond create)
	GetFrameFn      func(ctx context.Context, args miro.GetFrameArgs) (miro.GetFrameResult, error)
	UpdateFrameFn   func(ctx context.Context, args miro.UpdateFrameArgs) (miro.UpdateFrameResult, error)
	DeleteFrameFn   func(ctx context.Context, args miro.DeleteFrameArgs) (miro.DeleteFrameResult, error)
	GetFrameItemsFn func(ctx context.Context, args miro.GetFrameItemsArgs) (miro.GetFrameItemsResult, error)

	// Token operations
	ValidateTokenFn func(ctx context.Context) (*miro.UserInfo, error)

	// Export operations
	GetBoardPictureFn     func(ctx context.Context, args miro.GetBoardPictureArgs) (miro.GetBoardPictureResult, error)
	CreateExportJobFn     func(ctx context.Context, args miro.CreateExportJobArgs) (miro.CreateExportJobResult, error)
	GetExportJobStatusFn  func(ctx context.Context, args miro.GetExportJobStatusArgs) (miro.GetExportJobStatusResult, error)
	GetExportJobResultsFn func(ctx context.Context, args miro.GetExportJobResultsArgs) (miro.GetExportJobResultsResult, error)

	// Diagram operations
	GenerateDiagramFn func(ctx context.Context, args miro.GenerateDiagramArgs) (miro.GenerateDiagramResult, error)

	// App card operations
	CreateAppCardFn func(ctx context.Context, args miro.CreateAppCardArgs) (miro.CreateAppCardResult, error)
	GetAppCardFn    func(ctx context.Context, args miro.GetAppCardArgs) (miro.GetAppCardResult, error)
	UpdateAppCardFn func(ctx context.Context, args miro.UpdateAppCardArgs) (miro.UpdateAppCardResult, error)
	DeleteAppCardFn func(ctx context.Context, args miro.DeleteAppCardArgs) (miro.DeleteAppCardResult, error)

	// Doc format operations
	CreateDocFormatFn func(ctx context.Context, args miro.CreateDocFormatArgs) (miro.CreateDocFormatResult, error)
	GetDocFormatFn    func(ctx context.Context, args miro.GetDocFormatArgs) (miro.GetDocFormatResult, error)
	UpdateDocFormatFn func(ctx context.Context, args miro.UpdateDocFormatArgs) (miro.UpdateDocFormatResult, error)
	DeleteDocFormatFn func(ctx context.Context, args miro.DeleteDocFormatArgs) (miro.DeleteDocFormatResult, error)

	// Table operations
	ListTablesFn func(ctx context.Context, args miro.ListTablesArgs) (miro.ListTablesResult, error)
	GetTableFn   func(ctx context.Context, args miro.GetTableArgs) (miro.GetTableResult, error)

	// Native diagram read operations
	ListDiagramsFn func(ctx context.Context, args miro.ListDiagramsArgs) (miro.ListDiagramsResult, error)
	GetDiagramFn   func(ctx context.Context, args miro.GetDiagramArgs) (miro.GetDiagramResult, error)

	// Upload operations
	UploadImageFn            func(ctx context.Context, args miro.UploadImageArgs) (miro.UploadImageResult, error)
	UploadDocumentFn         func(ctx context.Context, args miro.UploadDocumentArgs) (miro.UploadDocumentResult, error)
	UpdateImageFromFileFn    func(ctx context.Context, args miro.UpdateImageFromFileArgs) (miro.UpdateImageFromFileResult, error)
	UpdateDocumentFromFileFn func(ctx context.Context, args miro.UpdateDocumentFromFileArgs) (miro.UpdateDocumentFromFileResult, error)

	// Call tracking
	Calls []MockCall
}

// MockCall records a method invocation for verification.
type MockCall struct {
	Method string
	Args   interface{}
}

// recordCall tracks method invocations.
func (m *MockClient) recordCall(method string, args interface{}) {
	m.Calls = append(m.Calls, MockCall{Method: method, Args: args})
}

// stub implements the shared mock-method contract: dispatch to the configured
// override when one is set, otherwise return the canned default result. Every
// MockClient method funnels through this helper after recording its call.
func stub[Args, Result any](ctx context.Context, override func(context.Context, Args) (Result, error), args Args, canned Result) (Result, error) {
	if override != nil {
		return override(ctx, args)
	}
	return canned, nil
}

// =============================================================================
// BoardService Implementation
// =============================================================================

func (m *MockClient) ListBoards(ctx context.Context, args miro.ListBoardsArgs) (miro.ListBoardsResult, error) {
	m.recordCall("ListBoards", args)
	return stub(ctx, m.ListBoardsFn, args, miro.ListBoardsResult{
		Boards: []miro.BoardSummary{
			{ID: "board1", Name: "Test Board 1", ViewLink: "https://miro.com/board1"},
			{ID: "board2", Name: "Test Board 2", ViewLink: "https://miro.com/board2"},
		},
		Count:   2,
		HasMore: false,
	})
}

func (m *MockClient) GetBoard(ctx context.Context, args miro.GetBoardArgs) (miro.GetBoardResult, error) {
	m.recordCall("GetBoard", args)
	return stub(ctx, m.GetBoardFn, args, miro.GetBoardResult{
		Board: miro.Board{
			ID:       args.BoardID,
			Name:     "Test Board",
			ViewLink: "https://miro.com/" + args.BoardID,
		},
	})
}

func (m *MockClient) CreateBoard(ctx context.Context, args miro.CreateBoardArgs) (miro.CreateBoardResult, error) {
	m.recordCall("CreateBoard", args)
	return stub(ctx, m.CreateBoardFn, args, miro.CreateBoardResult{
		ID:       "new-board-123",
		Name:     args.Name,
		ViewLink: "https://miro.com/new-board-123",
		Message:  fmt.Sprintf("Created board '%s'", args.Name),
	})
}

func (m *MockClient) CopyBoard(ctx context.Context, args miro.CopyBoardArgs) (miro.CopyBoardResult, error) {
	m.recordCall("CopyBoard", args)
	name := args.Name
	if name == "" {
		name = "Copy of Test Board"
	}
	return stub(ctx, m.CopyBoardFn, args, miro.CopyBoardResult{
		ID:       "copied-board-123",
		Name:     name,
		ViewLink: "https://miro.com/copied-board-123",
		Message:  fmt.Sprintf("Copied board to '%s'", name),
	})
}

func (m *MockClient) DeleteBoard(ctx context.Context, args miro.DeleteBoardArgs) (miro.DeleteBoardResult, error) {
	m.recordCall("DeleteBoard", args)
	return stub(ctx, m.DeleteBoardFn, args, miro.DeleteBoardResult{
		Success: true,
		BoardID: args.BoardID,
		Message: "Board deleted successfully",
	})
}

func (m *MockClient) UpdateBoard(ctx context.Context, args miro.UpdateBoardArgs) (miro.UpdateBoardResult, error) {
	m.recordCall("UpdateBoard", args)
	return stub(ctx, m.UpdateBoardFn, args, miro.UpdateBoardResult{
		ID:          args.BoardID,
		Name:        args.Name,
		Description: args.Description,
		ViewLink:    "https://miro.com/" + args.BoardID,
		Message:     "Board updated successfully",
	})
}

func (m *MockClient) FindBoardByName(ctx context.Context, name string) (*miro.BoardSummary, error) {
	m.recordCall("FindBoardByName", name)
	return stub(ctx, m.FindBoardByNameFn, name, &miro.BoardSummary{
		ID:       "found-board-123",
		Name:     name,
		ViewLink: "https://miro.com/found-board-123",
	})
}

func (m *MockClient) FindBoardByNameTool(ctx context.Context, args miro.FindBoardByNameArgs) (miro.FindBoardByNameResult, error) {
	m.recordCall("FindBoardByNameTool", args)
	return stub(ctx, m.FindBoardByNameToolFn, args, miro.FindBoardByNameResult{
		ID:       "found-board-123",
		Name:     args.Name,
		ViewLink: "https://miro.com/found-board-123",
		Message:  fmt.Sprintf("Found board '%s'", args.Name),
	})
}

func (m *MockClient) GetBoardSummary(ctx context.Context, args miro.GetBoardSummaryArgs) (miro.GetBoardSummaryResult, error) {
	m.recordCall("GetBoardSummary", args)
	return stub(ctx, m.GetBoardSummaryFn, args, miro.GetBoardSummaryResult{
		ID:          args.BoardID,
		Name:        "Test Board",
		TotalItems:  10,
		ItemCounts:  map[string]int{"sticky_note": 5, "shape": 3, "text": 2},
		RecentItems: []miro.ItemSummary{},
		Message:     "Board 'Test Board' has 10 items",
	})
}

func (m *MockClient) GetBoardContent(ctx context.Context, args miro.GetBoardContentArgs) (miro.GetBoardContentResult, error) {
	m.recordCall("GetBoardContent", args)
	return stub(ctx, m.GetBoardContentFn, args, miro.GetBoardContentResult{
		ID:         args.BoardID,
		Name:       "Test Board",
		ViewLink:   "https://miro.com/app/board/" + args.BoardID,
		TotalItems: 10,
		ItemCounts: map[string]int{"sticky_note": 5, "shape": 3, "text": 2},
		ItemsByType: miro.ItemsByType{
			StickyNotes: []miro.ItemSummary{{ID: "s1", Type: "sticky_note", Content: "Test"}},
		},
		ContentSummary: miro.ContentSummary{
			AllText:       []string{"Test"},
			UniqueEntries: 1,
			TotalChars:    4,
		},
		Message: "Board 'Test Board' has 10 items",
	})
}

// =============================================================================
// MemberService Implementation
// =============================================================================

func (m *MockClient) ListBoardMembers(ctx context.Context, args miro.ListBoardMembersArgs) (miro.ListBoardMembersResult, error) {
	m.recordCall("ListBoardMembers", args)
	return stub(ctx, m.ListBoardMembersFn, args, miro.ListBoardMembersResult{
		Members: []miro.BoardMember{
			{ID: "user1", Name: "Test User", Email: "test@example.com", Role: "owner"},
		},
		Count: 1,
	})
}

func (m *MockClient) ShareBoard(ctx context.Context, args miro.ShareBoardArgs) (miro.ShareBoardResult, error) {
	m.recordCall("ShareBoard", args)
	return stub(ctx, m.ShareBoardFn, args, miro.ShareBoardResult{
		Success: true,
		Email:   args.Email,
		Role:    args.Role,
		Message: fmt.Sprintf("Shared board with %s as %s", args.Email, args.Role),
	})
}

func (m *MockClient) GetBoardMember(ctx context.Context, args miro.GetBoardMemberArgs) (miro.GetBoardMemberResult, error) {
	m.recordCall("GetBoardMember", args)
	return stub(ctx, m.GetBoardMemberFn, args, miro.GetBoardMemberResult{
		ID: args.MemberID, Name: "Test User", Email: "test@example.com",
		Role: "editor", Message: "Member 'Test User' has role 'editor'",
	})
}

func (m *MockClient) RemoveBoardMember(ctx context.Context, args miro.RemoveBoardMemberArgs) (miro.RemoveBoardMemberResult, error) {
	m.recordCall("RemoveBoardMember", args)
	return stub(ctx, m.RemoveBoardMemberFn, args, miro.RemoveBoardMemberResult{
		Success:  true,
		MemberID: args.MemberID,
		Message:  "Member removed from board",
	})
}

func (m *MockClient) UpdateBoardMember(ctx context.Context, args miro.UpdateBoardMemberArgs) (miro.UpdateBoardMemberResult, error) {
	m.recordCall("UpdateBoardMember", args)
	return stub(ctx, m.UpdateBoardMemberFn, args, miro.UpdateBoardMemberResult{
		ID: args.MemberID, Name: "Test User", Email: "test@example.com",
		Role: args.Role, Message: fmt.Sprintf("Updated 'Test User' to role '%s'", args.Role),
	})
}

// =============================================================================
// TokenService Implementation
// =============================================================================

func (m *MockClient) ValidateToken(ctx context.Context) (*miro.UserInfo, error) {
	m.recordCall("ValidateToken", nil)
	if m.ValidateTokenFn != nil {
		return m.ValidateTokenFn(ctx)
	}
	return &miro.UserInfo{
		ID:    "user-123",
		Name:  "Test User",
		Email: "test@example.com",
	}, nil
}

// =============================================================================
// Test Helpers
// =============================================================================

// truncateForTest truncates a string for test output.
func truncateForTest(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// Verify MockClient implements miro.MiroClient at compile time.
var _ miro.MiroClient = (*MockClient)(nil)
