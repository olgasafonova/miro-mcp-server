package tools

import (
	"context"
	"fmt"

	"github.com/olgasafonova/miro-mcp-server/miro"
)

// =============================================================================
// Mock Client for Testing
// =============================================================================

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
	UngroupFn       func(ctx context.Context, args miro.UngroupArgs) (miro.UngroupResult, error)
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
	DeleteDocFormatFn func(ctx context.Context, args miro.DeleteDocFormatArgs) (miro.DeleteDocFormatResult, error)

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

// =============================================================================
// BoardService Implementation
// =============================================================================

func (m *MockClient) ListBoards(ctx context.Context, args miro.ListBoardsArgs) (miro.ListBoardsResult, error) {
	m.recordCall("ListBoards", args)
	if m.ListBoardsFn != nil {
		return m.ListBoardsFn(ctx, args)
	}
	return miro.ListBoardsResult{
		Boards: []miro.BoardSummary{
			{ID: "board1", Name: "Test Board 1", ViewLink: "https://miro.com/board1"},
			{ID: "board2", Name: "Test Board 2", ViewLink: "https://miro.com/board2"},
		},
		Count:   2,
		HasMore: false,
	}, nil
}

func (m *MockClient) GetBoard(ctx context.Context, args miro.GetBoardArgs) (miro.GetBoardResult, error) {
	m.recordCall("GetBoard", args)
	if m.GetBoardFn != nil {
		return m.GetBoardFn(ctx, args)
	}
	return miro.GetBoardResult{
		Board: miro.Board{
			ID:       args.BoardID,
			Name:     "Test Board",
			ViewLink: "https://miro.com/" + args.BoardID,
		},
	}, nil
}

func (m *MockClient) CreateBoard(ctx context.Context, args miro.CreateBoardArgs) (miro.CreateBoardResult, error) {
	m.recordCall("CreateBoard", args)
	if m.CreateBoardFn != nil {
		return m.CreateBoardFn(ctx, args)
	}
	return miro.CreateBoardResult{
		ID:       "new-board-123",
		Name:     args.Name,
		ViewLink: "https://miro.com/new-board-123",
		Message:  fmt.Sprintf("Created board '%s'", args.Name),
	}, nil
}

func (m *MockClient) CopyBoard(ctx context.Context, args miro.CopyBoardArgs) (miro.CopyBoardResult, error) {
	m.recordCall("CopyBoard", args)
	if m.CopyBoardFn != nil {
		return m.CopyBoardFn(ctx, args)
	}
	name := args.Name
	if name == "" {
		name = "Copy of Test Board"
	}
	return miro.CopyBoardResult{
		ID:       "copied-board-123",
		Name:     name,
		ViewLink: "https://miro.com/copied-board-123",
		Message:  fmt.Sprintf("Copied board to '%s'", name),
	}, nil
}

func (m *MockClient) DeleteBoard(ctx context.Context, args miro.DeleteBoardArgs) (miro.DeleteBoardResult, error) {
	m.recordCall("DeleteBoard", args)
	if m.DeleteBoardFn != nil {
		return m.DeleteBoardFn(ctx, args)
	}
	return miro.DeleteBoardResult{
		Success: true,
		BoardID: args.BoardID,
		Message: "Board deleted successfully",
	}, nil
}

func (m *MockClient) UpdateBoard(ctx context.Context, args miro.UpdateBoardArgs) (miro.UpdateBoardResult, error) {
	m.recordCall("UpdateBoard", args)
	if m.UpdateBoardFn != nil {
		return m.UpdateBoardFn(ctx, args)
	}
	return miro.UpdateBoardResult{
		ID:          args.BoardID,
		Name:        args.Name,
		Description: args.Description,
		ViewLink:    "https://miro.com/" + args.BoardID,
		Message:     "Board updated successfully",
	}, nil
}

func (m *MockClient) FindBoardByName(ctx context.Context, name string) (*miro.BoardSummary, error) {
	m.recordCall("FindBoardByName", name)
	if m.FindBoardByNameFn != nil {
		return m.FindBoardByNameFn(ctx, name)
	}
	return &miro.BoardSummary{
		ID:       "found-board-123",
		Name:     name,
		ViewLink: "https://miro.com/found-board-123",
	}, nil
}

func (m *MockClient) FindBoardByNameTool(ctx context.Context, args miro.FindBoardByNameArgs) (miro.FindBoardByNameResult, error) {
	m.recordCall("FindBoardByNameTool", args)
	if m.FindBoardByNameToolFn != nil {
		return m.FindBoardByNameToolFn(ctx, args)
	}
	return miro.FindBoardByNameResult{
		ID:       "found-board-123",
		Name:     args.Name,
		ViewLink: "https://miro.com/found-board-123",
		Message:  fmt.Sprintf("Found board '%s'", args.Name),
	}, nil
}

func (m *MockClient) GetBoardSummary(ctx context.Context, args miro.GetBoardSummaryArgs) (miro.GetBoardSummaryResult, error) {
	m.recordCall("GetBoardSummary", args)
	if m.GetBoardSummaryFn != nil {
		return m.GetBoardSummaryFn(ctx, args)
	}
	return miro.GetBoardSummaryResult{
		ID:          args.BoardID,
		Name:        "Test Board",
		TotalItems:  10,
		ItemCounts:  map[string]int{"sticky_note": 5, "shape": 3, "text": 2},
		RecentItems: []miro.ItemSummary{},
		Message:     "Board 'Test Board' has 10 items",
	}, nil
}

func (m *MockClient) GetBoardContent(ctx context.Context, args miro.GetBoardContentArgs) (miro.GetBoardContentResult, error) {
	m.recordCall("GetBoardContent", args)
	if m.GetBoardContentFn != nil {
		return m.GetBoardContentFn(ctx, args)
	}
	return miro.GetBoardContentResult{
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
	}, nil
}

// =============================================================================
// ItemService Implementation
// =============================================================================

func (m *MockClient) ListItems(ctx context.Context, args miro.ListItemsArgs) (miro.ListItemsResult, error) {
	m.recordCall("ListItems", args)
	if m.ListItemsFn != nil {
		return m.ListItemsFn(ctx, args)
	}
	return miro.ListItemsResult{
		Items: []miro.ItemSummary{
			{ID: "item1", Type: "sticky_note", Content: "Test sticky"},
			{ID: "item2", Type: "shape", Content: "Test shape"},
		},
		Count:   2,
		HasMore: false,
	}, nil
}

func (m *MockClient) ListAllItems(ctx context.Context, args miro.ListAllItemsArgs) (miro.ListAllItemsResult, error) {
	m.recordCall("ListAllItems", args)
	if m.ListAllItemsFn != nil {
		return m.ListAllItemsFn(ctx, args)
	}
	return miro.ListAllItemsResult{
		Items: []miro.ItemSummary{
			{ID: "item1", Type: "sticky_note", Content: "Test sticky"},
		},
		Count:      1,
		TotalPages: 1,
		Message:    "Retrieved 1 items in 1 pages",
	}, nil
}

func (m *MockClient) GetItem(ctx context.Context, args miro.GetItemArgs) (miro.GetItemResult, error) {
	m.recordCall("GetItem", args)
	if m.GetItemFn != nil {
		return m.GetItemFn(ctx, args)
	}
	return miro.GetItemResult{
		ID:      args.ItemID,
		Type:    "sticky_note",
		Content: "Test sticky content",
	}, nil
}

func (m *MockClient) GetImage(ctx context.Context, args miro.GetImageArgs) (miro.GetImageResult, error) {
	m.recordCall("GetImage", args)
	if m.GetImageFn != nil {
		return m.GetImageFn(ctx, args)
	}
	return miro.GetImageResult{
		ID:       args.ItemID,
		Title:    "Test Image",
		ImageURL: "https://miro.com/images/test.png",
		Width:    800,
		Height:   600,
		Message:  "Image retrieved successfully",
	}, nil
}

func (m *MockClient) GetDocument(ctx context.Context, args miro.GetDocumentArgs) (miro.GetDocumentResult, error) {
	m.recordCall("GetDocument", args)
	if m.GetDocumentFn != nil {
		return m.GetDocumentFn(ctx, args)
	}
	return miro.GetDocumentResult{
		ID:          args.ItemID,
		Title:       "Test Document",
		DocumentURL: "https://miro.com/documents/test.pdf",
		Message:     "Document retrieved successfully",
	}, nil
}

func (m *MockClient) UpdateItem(ctx context.Context, args miro.UpdateItemArgs) (miro.UpdateItemResult, error) {
	m.recordCall("UpdateItem", args)
	if m.UpdateItemFn != nil {
		return m.UpdateItemFn(ctx, args)
	}
	return miro.UpdateItemResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "Item updated successfully",
	}, nil
}

func (m *MockClient) UpdateSticky(ctx context.Context, args miro.UpdateStickyArgs) (miro.UpdateStickyResult, error) {
	m.recordCall("UpdateSticky", args)
	if m.UpdateStickyFn != nil {
		return m.UpdateStickyFn(ctx, args)
	}
	return miro.UpdateStickyResult{
		ID:      args.ItemID,
		Message: "Sticky updated successfully",
	}, nil
}

func (m *MockClient) UpdateShape(ctx context.Context, args miro.UpdateShapeArgs) (miro.UpdateShapeResult, error) {
	m.recordCall("UpdateShape", args)
	if m.UpdateShapeFn != nil {
		return m.UpdateShapeFn(ctx, args)
	}
	return miro.UpdateShapeResult{
		ID:      args.ItemID,
		Message: "Shape updated successfully",
	}, nil
}

func (m *MockClient) UpdateText(ctx context.Context, args miro.UpdateTextArgs) (miro.UpdateTextResult, error) {
	m.recordCall("UpdateText", args)
	if m.UpdateTextFn != nil {
		return m.UpdateTextFn(ctx, args)
	}
	return miro.UpdateTextResult{
		ID:      args.ItemID,
		Message: "Text updated successfully",
	}, nil
}

func (m *MockClient) UpdateCard(ctx context.Context, args miro.UpdateCardArgs) (miro.UpdateCardResult, error) {
	m.recordCall("UpdateCard", args)
	if m.UpdateCardFn != nil {
		return m.UpdateCardFn(ctx, args)
	}
	return miro.UpdateCardResult{
		ID:      args.ItemID,
		Message: "Card updated successfully",
	}, nil
}

func (m *MockClient) UpdateImage(ctx context.Context, args miro.UpdateImageArgs) (miro.UpdateImageResult, error) {
	m.recordCall("UpdateImage", args)
	if m.UpdateImageFn != nil {
		return m.UpdateImageFn(ctx, args)
	}
	return miro.UpdateImageResult{
		ID:      args.ItemID,
		Message: "Image updated successfully",
	}, nil
}

func (m *MockClient) UpdateDocument(ctx context.Context, args miro.UpdateDocumentArgs) (miro.UpdateDocumentResult, error) {
	m.recordCall("UpdateDocument", args)
	if m.UpdateDocumentFn != nil {
		return m.UpdateDocumentFn(ctx, args)
	}
	return miro.UpdateDocumentResult{
		ID:      args.ItemID,
		Message: "Document updated successfully",
	}, nil
}

func (m *MockClient) UpdateEmbed(ctx context.Context, args miro.UpdateEmbedArgs) (miro.UpdateEmbedResult, error) {
	m.recordCall("UpdateEmbed", args)
	if m.UpdateEmbedFn != nil {
		return m.UpdateEmbedFn(ctx, args)
	}
	return miro.UpdateEmbedResult{
		ID:      args.ItemID,
		Message: "Embed updated successfully",
	}, nil
}

func (m *MockClient) DeleteItem(ctx context.Context, args miro.DeleteItemArgs) (miro.DeleteItemResult, error) {
	m.recordCall("DeleteItem", args)
	if m.DeleteItemFn != nil {
		return m.DeleteItemFn(ctx, args)
	}
	return miro.DeleteItemResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "Item deleted successfully",
	}, nil
}

func (m *MockClient) SearchBoard(ctx context.Context, args miro.SearchBoardArgs) (miro.SearchBoardResult, error) {
	m.recordCall("SearchBoard", args)
	if m.SearchBoardFn != nil {
		return m.SearchBoardFn(ctx, args)
	}
	return miro.SearchBoardResult{
		Matches: []miro.ItemMatch{
			{ID: "item1", Type: "sticky_note", Content: "Found: " + args.Query, Snippet: args.Query},
		},
		Count:   1,
		Query:   args.Query,
		Message: fmt.Sprintf("Found 1 items matching '%s'", args.Query),
	}, nil
}

func (m *MockClient) BulkCreate(ctx context.Context, args miro.BulkCreateArgs) (miro.BulkCreateResult, error) {
	m.recordCall("BulkCreate", args)
	if m.BulkCreateFn != nil {
		return m.BulkCreateFn(ctx, args)
	}
	itemIDs := make([]string, len(args.Items))
	for i := range args.Items {
		itemIDs[i] = fmt.Sprintf("bulk-item-%d", i+1)
	}
	return miro.BulkCreateResult{
		Created: len(args.Items),
		ItemIDs: itemIDs,
		Errors:  []string{},
		Message: fmt.Sprintf("Created %d items", len(args.Items)),
	}, nil
}

func (m *MockClient) BulkUpdate(ctx context.Context, args miro.BulkUpdateArgs) (miro.BulkUpdateResult, error) {
	m.recordCall("BulkUpdate", args)
	if m.BulkUpdateFn != nil {
		return m.BulkUpdateFn(ctx, args)
	}
	itemIDs := make([]string, len(args.Items))
	for i, item := range args.Items {
		itemIDs[i] = item.ItemID
	}
	return miro.BulkUpdateResult{
		Updated: len(args.Items),
		ItemIDs: itemIDs,
		Errors:  []string{},
		Message: fmt.Sprintf("Updated %d items", len(args.Items)),
	}, nil
}

func (m *MockClient) BulkDelete(ctx context.Context, args miro.BulkDeleteArgs) (miro.BulkDeleteResult, error) {
	m.recordCall("BulkDelete", args)
	if m.BulkDeleteFn != nil {
		return m.BulkDeleteFn(ctx, args)
	}
	return miro.BulkDeleteResult{
		Deleted: len(args.ItemIDs),
		ItemIDs: args.ItemIDs,
		Errors:  []string{},
		Message: fmt.Sprintf("Deleted %d items", len(args.ItemIDs)),
	}, nil
}

// =============================================================================
// CreateService Implementation
// =============================================================================

func (m *MockClient) CreateSticky(ctx context.Context, args miro.CreateStickyArgs) (miro.CreateStickyResult, error) {
	m.recordCall("CreateSticky", args)
	if m.CreateStickyFn != nil {
		return m.CreateStickyFn(ctx, args)
	}
	return miro.CreateStickyResult{
		ID:      "sticky-123",
		Content: args.Content,
		Color:   args.Color,
		Message: fmt.Sprintf("Created sticky note '%s'", truncateForTest(args.Content, 30)),
	}, nil
}

func (m *MockClient) CreateShape(ctx context.Context, args miro.CreateShapeArgs) (miro.CreateShapeResult, error) {
	m.recordCall("CreateShape", args)
	if m.CreateShapeFn != nil {
		return m.CreateShapeFn(ctx, args)
	}
	return miro.CreateShapeResult{
		ID:      "shape-123",
		Shape:   args.Shape,
		Content: args.Content,
		Message: fmt.Sprintf("Created %s shape", args.Shape),
	}, nil
}

func (m *MockClient) CreateShapeExperimental(ctx context.Context, args miro.CreateShapeExperimentalArgs) (miro.CreateShapeResult, error) {
	m.recordCall("CreateShapeExperimental", args)
	if m.CreateShapeExperimentalFn != nil {
		return m.CreateShapeExperimentalFn(ctx, args)
	}
	return miro.CreateShapeResult{
		ID:      "shape-exp-123",
		Shape:   args.Shape,
		Content: args.Content,
		Message: fmt.Sprintf("Created experimental %s shape", args.Shape),
	}, nil
}

func (m *MockClient) CreateFlowchartShape(ctx context.Context, args miro.CreateFlowchartShapeArgs) (miro.CreateShapeResult, error) {
	m.recordCall("CreateFlowchartShape", args)
	if m.CreateFlowchartShapeFn != nil {
		return m.CreateFlowchartShapeFn(ctx, args)
	}
	return miro.CreateShapeResult{
		ID:      "flowchart-shape-123",
		Shape:   args.Shape,
		Content: args.Content,
		Message: fmt.Sprintf("Created flowchart %s shape", args.Shape),
	}, nil
}

func (m *MockClient) CreateText(ctx context.Context, args miro.CreateTextArgs) (miro.CreateTextResult, error) {
	m.recordCall("CreateText", args)
	if m.CreateTextFn != nil {
		return m.CreateTextFn(ctx, args)
	}
	return miro.CreateTextResult{
		ID:      "text-123",
		Content: args.Content,
		Message: "Created text element",
	}, nil
}

func (m *MockClient) CreateConnector(ctx context.Context, args miro.CreateConnectorArgs) (miro.CreateConnectorResult, error) {
	m.recordCall("CreateConnector", args)
	if m.CreateConnectorFn != nil {
		return m.CreateConnectorFn(ctx, args)
	}
	return miro.CreateConnectorResult{
		ID:      "connector-123",
		Message: fmt.Sprintf("Created connector from %s to %s", args.StartItemID, args.EndItemID),
	}, nil
}

func (m *MockClient) CreateFrame(ctx context.Context, args miro.CreateFrameArgs) (miro.CreateFrameResult, error) {
	m.recordCall("CreateFrame", args)
	if m.CreateFrameFn != nil {
		return m.CreateFrameFn(ctx, args)
	}
	return miro.CreateFrameResult{
		ID:      "frame-123",
		Title:   args.Title,
		Message: fmt.Sprintf("Created frame '%s'", args.Title),
	}, nil
}

func (m *MockClient) CreateCard(ctx context.Context, args miro.CreateCardArgs) (miro.CreateCardResult, error) {
	m.recordCall("CreateCard", args)
	if m.CreateCardFn != nil {
		return m.CreateCardFn(ctx, args)
	}
	return miro.CreateCardResult{
		ID:      "card-123",
		Title:   args.Title,
		Message: fmt.Sprintf("Created card '%s'", args.Title),
	}, nil
}

func (m *MockClient) CreateImage(ctx context.Context, args miro.CreateImageArgs) (miro.CreateImageResult, error) {
	m.recordCall("CreateImage", args)
	if m.CreateImageFn != nil {
		return m.CreateImageFn(ctx, args)
	}
	return miro.CreateImageResult{
		ID:      "image-123",
		Title:   args.Title,
		URL:     args.URL,
		Message: "Created image",
	}, nil
}

func (m *MockClient) CreateDocument(ctx context.Context, args miro.CreateDocumentArgs) (miro.CreateDocumentResult, error) {
	m.recordCall("CreateDocument", args)
	if m.CreateDocumentFn != nil {
		return m.CreateDocumentFn(ctx, args)
	}
	return miro.CreateDocumentResult{
		ID:      "doc-123",
		Title:   args.Title,
		Message: "Created document",
	}, nil
}

func (m *MockClient) CreateEmbed(ctx context.Context, args miro.CreateEmbedArgs) (miro.CreateEmbedResult, error) {
	m.recordCall("CreateEmbed", args)
	if m.CreateEmbedFn != nil {
		return m.CreateEmbedFn(ctx, args)
	}
	return miro.CreateEmbedResult{
		ID:      "embed-123",
		URL:     args.URL,
		Message: "Created embed",
	}, nil
}

func (m *MockClient) CreateStickyGrid(ctx context.Context, args miro.CreateStickyGridArgs) (miro.CreateStickyGridResult, error) {
	m.recordCall("CreateStickyGrid", args)
	if m.CreateStickyGridFn != nil {
		return m.CreateStickyGridFn(ctx, args)
	}
	itemIDs := make([]string, len(args.Contents))
	for i := range args.Contents {
		itemIDs[i] = fmt.Sprintf("grid-sticky-%d", i+1)
	}
	columns := args.Columns
	if columns == 0 {
		columns = 3
	}
	rows := (len(args.Contents) + columns - 1) / columns
	return miro.CreateStickyGridResult{
		Created: len(args.Contents),
		ItemIDs: itemIDs,
		Rows:    rows,
		Columns: columns,
		Message: fmt.Sprintf("Created %d stickies in a grid", len(args.Contents)),
	}, nil
}

// =============================================================================
// TagService Implementation
// =============================================================================

func (m *MockClient) CreateTag(ctx context.Context, args miro.CreateTagArgs) (miro.CreateTagResult, error) {
	m.recordCall("CreateTag", args)
	if m.CreateTagFn != nil {
		return m.CreateTagFn(ctx, args)
	}
	return miro.CreateTagResult{
		ID:      "tag-123",
		Title:   args.Title,
		Color:   args.Color,
		Message: fmt.Sprintf("Created tag '%s'", args.Title),
	}, nil
}

func (m *MockClient) ListTags(ctx context.Context, args miro.ListTagsArgs) (miro.ListTagsResult, error) {
	m.recordCall("ListTags", args)
	if m.ListTagsFn != nil {
		return m.ListTagsFn(ctx, args)
	}
	return miro.ListTagsResult{
		Tags: []miro.Tag{
			{ID: "tag1", Title: "Urgent", FillColor: "red"},
			{ID: "tag2", Title: "Done", FillColor: "green"},
		},
		Count: 2,
	}, nil
}

func (m *MockClient) AttachTag(ctx context.Context, args miro.AttachTagArgs) (miro.AttachTagResult, error) {
	m.recordCall("AttachTag", args)
	if m.AttachTagFn != nil {
		return m.AttachTagFn(ctx, args)
	}
	return miro.AttachTagResult{
		Success: true,
		ItemID:  args.ItemID,
		TagID:   args.TagID,
		Message: "Tag attached successfully",
	}, nil
}

func (m *MockClient) DetachTag(ctx context.Context, args miro.DetachTagArgs) (miro.DetachTagResult, error) {
	m.recordCall("DetachTag", args)
	if m.DetachTagFn != nil {
		return m.DetachTagFn(ctx, args)
	}
	return miro.DetachTagResult{
		Success: true,
		ItemID:  args.ItemID,
		TagID:   args.TagID,
		Message: "Tag detached successfully",
	}, nil
}

func (m *MockClient) GetItemTags(ctx context.Context, args miro.GetItemTagsArgs) (miro.GetItemTagsResult, error) {
	m.recordCall("GetItemTags", args)
	if m.GetItemTagsFn != nil {
		return m.GetItemTagsFn(ctx, args)
	}
	return miro.GetItemTagsResult{
		Tags: []miro.Tag{
			{ID: "tag1", Title: "Urgent", FillColor: "red"},
		},
		Count:  1,
		ItemID: args.ItemID,
	}, nil
}

func (m *MockClient) GetItemsByTag(ctx context.Context, args miro.GetItemsByTagArgs) (miro.GetItemsByTagResult, error) {
	m.recordCall("GetItemsByTag", args)
	if m.GetItemsByTagFn != nil {
		return m.GetItemsByTagFn(ctx, args)
	}
	return miro.GetItemsByTagResult{
		Items: []miro.ItemSummary{
			{ID: "item-1", Type: "sticky_note", Content: "Tagged item"},
		},
		Count:   1,
		HasMore: false,
		TagID:   args.TagID,
		Message: fmt.Sprintf("Found 1 items with tag %s", args.TagID),
	}, nil
}

func (m *MockClient) GetTag(ctx context.Context, args miro.GetTagArgs) (miro.GetTagResult, error) {
	m.recordCall("GetTag", args)
	if m.GetTagFn != nil {
		return m.GetTagFn(ctx, args)
	}
	return miro.GetTagResult{
		ID:      args.TagID,
		Title:   "Urgent",
		Color:   "red",
		Message: "Tag 'Urgent'",
	}, nil
}

func (m *MockClient) UpdateTag(ctx context.Context, args miro.UpdateTagArgs) (miro.UpdateTagResult, error) {
	m.recordCall("UpdateTag", args)
	if m.UpdateTagFn != nil {
		return m.UpdateTagFn(ctx, args)
	}
	title := args.Title
	if title == "" {
		title = "Updated Tag"
	}
	color := args.Color
	if color == "" {
		color = "green"
	}
	return miro.UpdateTagResult{
		Success: true,
		ID:      args.TagID,
		Title:   title,
		Color:   color,
		Message: fmt.Sprintf("Updated tag '%s'", title),
	}, nil
}

func (m *MockClient) DeleteTag(ctx context.Context, args miro.DeleteTagArgs) (miro.DeleteTagResult, error) {
	m.recordCall("DeleteTag", args)
	if m.DeleteTagFn != nil {
		return m.DeleteTagFn(ctx, args)
	}
	return miro.DeleteTagResult{
		Success: true,
		TagID:   args.TagID,
		Message: "Tag deleted successfully",
	}, nil
}

// =============================================================================
// ConnectorService Implementation
// =============================================================================

func (m *MockClient) ListConnectors(ctx context.Context, args miro.ListConnectorsArgs) (miro.ListConnectorsResult, error) {
	m.recordCall("ListConnectors", args)
	if m.ListConnectorsFn != nil {
		return m.ListConnectorsFn(ctx, args)
	}
	return miro.ListConnectorsResult{
		Connectors: []miro.ConnectorSummary{
			{ID: "conn-1", StartItemID: "item-1", EndItemID: "item-2", Style: "elbowed"},
			{ID: "conn-2", StartItemID: "item-2", EndItemID: "item-3", Style: "straight"},
		},
		Count:   2,
		HasMore: false,
		Message: "Found 2 connectors",
	}, nil
}

func (m *MockClient) GetConnector(ctx context.Context, args miro.GetConnectorArgs) (miro.GetConnectorResult, error) {
	m.recordCall("GetConnector", args)
	if m.GetConnectorFn != nil {
		return m.GetConnectorFn(ctx, args)
	}
	return miro.GetConnectorResult{
		ID:          args.ConnectorID,
		StartItemID: "item-1",
		EndItemID:   "item-2",
		Style:       "elbowed",
		EndCap:      "arrow",
		Message:     "Retrieved connector details",
	}, nil
}

func (m *MockClient) UpdateConnector(ctx context.Context, args miro.UpdateConnectorArgs) (miro.UpdateConnectorResult, error) {
	m.recordCall("UpdateConnector", args)
	if m.UpdateConnectorFn != nil {
		return m.UpdateConnectorFn(ctx, args)
	}
	return miro.UpdateConnectorResult{
		Success: true,
		ID:      args.ConnectorID,
		Message: "Connector updated successfully",
	}, nil
}

func (m *MockClient) DeleteConnector(ctx context.Context, args miro.DeleteConnectorArgs) (miro.DeleteConnectorResult, error) {
	m.recordCall("DeleteConnector", args)
	if m.DeleteConnectorFn != nil {
		return m.DeleteConnectorFn(ctx, args)
	}
	return miro.DeleteConnectorResult{
		Success: true,
		ID:      args.ConnectorID,
		Message: "Connector deleted successfully",
	}, nil
}

// =============================================================================
// GroupService Implementation
// =============================================================================

func (m *MockClient) CreateGroup(ctx context.Context, args miro.CreateGroupArgs) (miro.CreateGroupResult, error) {
	m.recordCall("CreateGroup", args)
	if m.CreateGroupFn != nil {
		return m.CreateGroupFn(ctx, args)
	}
	return miro.CreateGroupResult{
		ID:      "group-123",
		ItemIDs: args.ItemIDs,
		Message: fmt.Sprintf("Grouped %d items", len(args.ItemIDs)),
	}, nil
}

func (m *MockClient) Ungroup(ctx context.Context, args miro.UngroupArgs) (miro.UngroupResult, error) {
	m.recordCall("Ungroup", args)
	if m.UngroupFn != nil {
		return m.UngroupFn(ctx, args)
	}
	return miro.UngroupResult{
		Success: true,
		GroupID: args.GroupID,
		Message: "Items ungrouped successfully",
	}, nil
}

func (m *MockClient) ListGroups(ctx context.Context, args miro.ListGroupsArgs) (miro.ListGroupsResult, error) {
	m.recordCall("ListGroups", args)
	if m.ListGroupsFn != nil {
		return m.ListGroupsFn(ctx, args)
	}
	return miro.ListGroupsResult{
		Groups:  []miro.Group{{ID: "group-1", Items: []string{"item-1", "item-2"}}},
		Count:   1,
		HasMore: false,
		Message: "Found 1 groups",
	}, nil
}

func (m *MockClient) GetGroup(ctx context.Context, args miro.GetGroupArgs) (miro.GetGroupResult, error) {
	m.recordCall("GetGroup", args)
	if m.GetGroupFn != nil {
		return m.GetGroupFn(ctx, args)
	}
	return miro.GetGroupResult{
		ID:      args.GroupID,
		Items:   []string{"item-1", "item-2"},
		Message: "Group contains 2 items",
	}, nil
}

func (m *MockClient) GetGroupItems(ctx context.Context, args miro.GetGroupItemsArgs) (miro.GetGroupItemsResult, error) {
	m.recordCall("GetGroupItems", args)
	if m.GetGroupItemsFn != nil {
		return m.GetGroupItemsFn(ctx, args)
	}
	return miro.GetGroupItemsResult{
		Items: []miro.ItemSummary{
			{ID: "item-1", Type: "sticky_note", Content: "Test sticky"},
		},
		Count:   1,
		HasMore: false,
		Message: "Found 1 items in group",
	}, nil
}

func (m *MockClient) UpdateGroup(ctx context.Context, args miro.UpdateGroupArgs) (miro.UpdateGroupResult, error) {
	m.recordCall("UpdateGroup", args)
	if m.UpdateGroupFn != nil {
		return m.UpdateGroupFn(ctx, args)
	}
	return miro.UpdateGroupResult{
		ID:      args.GroupID,
		ItemIDs: args.ItemIDs,
		Message: fmt.Sprintf("Updated group with %d items", len(args.ItemIDs)),
	}, nil
}

func (m *MockClient) DeleteGroup(ctx context.Context, args miro.DeleteGroupArgs) (miro.DeleteGroupResult, error) {
	m.recordCall("DeleteGroup", args)
	if m.DeleteGroupFn != nil {
		return m.DeleteGroupFn(ctx, args)
	}
	msg := "Group deleted, items ungrouped"
	if args.DeleteItems {
		msg = "Group and its items deleted"
	}
	return miro.DeleteGroupResult{
		Success: true,
		GroupID: args.GroupID,
		Message: msg,
	}, nil
}

// =============================================================================
// MemberService Implementation
// =============================================================================

func (m *MockClient) ListBoardMembers(ctx context.Context, args miro.ListBoardMembersArgs) (miro.ListBoardMembersResult, error) {
	m.recordCall("ListBoardMembers", args)
	if m.ListBoardMembersFn != nil {
		return m.ListBoardMembersFn(ctx, args)
	}
	return miro.ListBoardMembersResult{
		Members: []miro.BoardMember{
			{ID: "user1", Name: "Test User", Email: "test@example.com", Role: "owner"},
		},
		Count: 1,
	}, nil
}

func (m *MockClient) ShareBoard(ctx context.Context, args miro.ShareBoardArgs) (miro.ShareBoardResult, error) {
	m.recordCall("ShareBoard", args)
	if m.ShareBoardFn != nil {
		return m.ShareBoardFn(ctx, args)
	}
	return miro.ShareBoardResult{
		Success: true,
		Email:   args.Email,
		Role:    args.Role,
		Message: fmt.Sprintf("Shared board with %s as %s", args.Email, args.Role),
	}, nil
}

func (m *MockClient) GetBoardMember(ctx context.Context, args miro.GetBoardMemberArgs) (miro.GetBoardMemberResult, error) {
	m.recordCall("GetBoardMember", args)
	if m.GetBoardMemberFn != nil {
		return m.GetBoardMemberFn(ctx, args)
	}
	return miro.GetBoardMemberResult{
		ID:      args.MemberID,
		Name:    "Test User",
		Email:   "test@example.com",
		Role:    "editor",
		Message: "Member 'Test User' has role 'editor'",
	}, nil
}

func (m *MockClient) RemoveBoardMember(ctx context.Context, args miro.RemoveBoardMemberArgs) (miro.RemoveBoardMemberResult, error) {
	m.recordCall("RemoveBoardMember", args)
	if m.RemoveBoardMemberFn != nil {
		return m.RemoveBoardMemberFn(ctx, args)
	}
	return miro.RemoveBoardMemberResult{
		Success:  true,
		MemberID: args.MemberID,
		Message:  "Member removed from board",
	}, nil
}

func (m *MockClient) UpdateBoardMember(ctx context.Context, args miro.UpdateBoardMemberArgs) (miro.UpdateBoardMemberResult, error) {
	m.recordCall("UpdateBoardMember", args)
	if m.UpdateBoardMemberFn != nil {
		return m.UpdateBoardMemberFn(ctx, args)
	}
	return miro.UpdateBoardMemberResult{
		ID:      args.MemberID,
		Name:    "Test User",
		Email:   "test@example.com",
		Role:    args.Role,
		Message: fmt.Sprintf("Updated 'Test User' to role '%s'", args.Role),
	}, nil
}

// =============================================================================
// MindmapService Implementation
// =============================================================================

func (m *MockClient) CreateMindmapNode(ctx context.Context, args miro.CreateMindmapNodeArgs) (miro.CreateMindmapNodeResult, error) {
	m.recordCall("CreateMindmapNode", args)
	if m.CreateMindmapNodeFn != nil {
		return m.CreateMindmapNodeFn(ctx, args)
	}
	return miro.CreateMindmapNodeResult{
		ID:       "mindmap-node-123",
		Content:  args.Content,
		ParentID: args.ParentID,
		Message:  fmt.Sprintf("Created mindmap node '%s'", truncateForTest(args.Content, 30)),
	}, nil
}

func (m *MockClient) GetMindmapNode(ctx context.Context, args miro.GetMindmapNodeArgs) (miro.GetMindmapNodeResult, error) {
	m.recordCall("GetMindmapNode", args)
	if m.GetMindmapNodeFn != nil {
		return m.GetMindmapNodeFn(ctx, args)
	}
	return miro.GetMindmapNodeResult{
		ID:       args.NodeID,
		Content:  "Test Node Content",
		NodeView: "text",
		IsRoot:   true,
		X:        100,
		Y:        100,
		Message:  "Retrieved mindmap node 'Test Node Content'",
	}, nil
}

func (m *MockClient) ListMindmapNodes(ctx context.Context, args miro.ListMindmapNodesArgs) (miro.ListMindmapNodesResult, error) {
	m.recordCall("ListMindmapNodes", args)
	if m.ListMindmapNodesFn != nil {
		return m.ListMindmapNodesFn(ctx, args)
	}
	return miro.ListMindmapNodesResult{
		Nodes: []miro.MindmapNodeSummary{
			{ID: "node-1", Content: "Root", IsRoot: true},
			{ID: "node-2", Content: "Child 1", ParentID: "node-1"},
		},
		Count:   2,
		HasMore: false,
		Message: "Found 2 mindmap nodes",
	}, nil
}

func (m *MockClient) DeleteMindmapNode(ctx context.Context, args miro.DeleteMindmapNodeArgs) (miro.DeleteMindmapNodeResult, error) {
	m.recordCall("DeleteMindmapNode", args)
	if m.DeleteMindmapNodeFn != nil {
		return m.DeleteMindmapNodeFn(ctx, args)
	}
	return miro.DeleteMindmapNodeResult{
		Success: true,
		ID:      args.NodeID,
		Message: "Mindmap node deleted successfully",
	}, nil
}

// =============================================================================
// FrameService Implementation (beyond create)
// =============================================================================

func (m *MockClient) GetFrame(ctx context.Context, args miro.GetFrameArgs) (miro.GetFrameResult, error) {
	m.recordCall("GetFrame", args)
	if m.GetFrameFn != nil {
		return m.GetFrameFn(ctx, args)
	}
	return miro.GetFrameResult{
		ID:         args.FrameID,
		Title:      "Test Frame",
		X:          0,
		Y:          0,
		Width:      800,
		Height:     600,
		ChildCount: 5,
		Message:    "Retrieved frame 'Test Frame'",
	}, nil
}

func (m *MockClient) UpdateFrame(ctx context.Context, args miro.UpdateFrameArgs) (miro.UpdateFrameResult, error) {
	m.recordCall("UpdateFrame", args)
	if m.UpdateFrameFn != nil {
		return m.UpdateFrameFn(ctx, args)
	}
	return miro.UpdateFrameResult{
		Success: true,
		ID:      args.FrameID,
		Message: "Frame updated successfully",
	}, nil
}

func (m *MockClient) DeleteFrame(ctx context.Context, args miro.DeleteFrameArgs) (miro.DeleteFrameResult, error) {
	m.recordCall("DeleteFrame", args)
	if m.DeleteFrameFn != nil {
		return m.DeleteFrameFn(ctx, args)
	}
	return miro.DeleteFrameResult{
		Success: true,
		ID:      args.FrameID,
		Message: "Frame deleted successfully",
	}, nil
}

func (m *MockClient) GetFrameItems(ctx context.Context, args miro.GetFrameItemsArgs) (miro.GetFrameItemsResult, error) {
	m.recordCall("GetFrameItems", args)
	if m.GetFrameItemsFn != nil {
		return m.GetFrameItemsFn(ctx, args)
	}
	return miro.GetFrameItemsResult{
		Items: []miro.ItemSummary{
			{ID: "item-1", Type: "sticky_note", Content: "Test sticky"},
			{ID: "item-2", Type: "shape", Content: "Test shape"},
		},
		Count:   2,
		HasMore: false,
		Message: "Found 2 items in frame",
	}, nil
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
// ExportService Implementation
// =============================================================================

func (m *MockClient) GetBoardPicture(ctx context.Context, args miro.GetBoardPictureArgs) (miro.GetBoardPictureResult, error) {
	m.recordCall("GetBoardPicture", args)
	if m.GetBoardPictureFn != nil {
		return m.GetBoardPictureFn(ctx, args)
	}
	return miro.GetBoardPictureResult{
		BoardID:  args.BoardID,
		ImageURL: "https://miro.com/boards/" + args.BoardID + "/picture.png",
		Message:  "Board picture URL retrieved successfully",
	}, nil
}

func (m *MockClient) CreateExportJob(ctx context.Context, args miro.CreateExportJobArgs) (miro.CreateExportJobResult, error) {
	m.recordCall("CreateExportJob", args)
	if m.CreateExportJobFn != nil {
		return m.CreateExportJobFn(ctx, args)
	}
	return miro.CreateExportJobResult{
		JobID:     "export-job-123",
		Status:    "in_progress",
		RequestID: "request-123",
		Message:   fmt.Sprintf("Export job created for %d board(s)", len(args.BoardIDs)),
	}, nil
}

func (m *MockClient) GetExportJobStatus(ctx context.Context, args miro.GetExportJobStatusArgs) (miro.GetExportJobStatusResult, error) {
	m.recordCall("GetExportJobStatus", args)
	if m.GetExportJobStatusFn != nil {
		return m.GetExportJobStatusFn(ctx, args)
	}
	return miro.GetExportJobStatusResult{
		JobID:          args.JobID,
		Status:         "completed",
		Progress:       100,
		BoardsTotal:    2,
		BoardsExported: 2,
		Message:        "Export job completed: 2/2 boards exported",
	}, nil
}

func (m *MockClient) GetExportJobResults(ctx context.Context, args miro.GetExportJobResultsArgs) (miro.GetExportJobResultsResult, error) {
	m.recordCall("GetExportJobResults", args)
	if m.GetExportJobResultsFn != nil {
		return m.GetExportJobResultsFn(ctx, args)
	}
	return miro.GetExportJobResultsResult{
		JobID:  args.JobID,
		Status: "completed",
		Boards: []miro.ExportedBoard{
			{
				BoardID:     "board1",
				BoardName:   "Test Board 1",
				DownloadURL: "https://miro.com/export/board1.pdf",
				Format:      "pdf",
			},
		},
		ExpiresIn: "15 minutes",
		Message:   "Export completed: 1 board(s) ready for download",
	}, nil
}

// =============================================================================
// DiagramService Implementation
// =============================================================================

func (m *MockClient) GenerateDiagram(ctx context.Context, args miro.GenerateDiagramArgs) (miro.GenerateDiagramResult, error) {
	m.recordCall("GenerateDiagram", args)
	if m.GenerateDiagramFn != nil {
		return m.GenerateDiagramFn(ctx, args)
	}
	return miro.GenerateDiagramResult{
		NodesCreated:      3,
		ConnectorsCreated: 2,
		FramesCreated:     0,
		NodeIDs:           []string{"node-1", "node-2", "node-3"},
		ConnectorIDs:      []string{"conn-1", "conn-2"},
		FrameIDs:          []string{},
		DiagramWidth:      400,
		DiagramHeight:     300,
		Message:           "Created diagram with 3 nodes and 2 connectors",
	}, nil
}

// =============================================================================
// AppCardService Implementation
// =============================================================================

func (m *MockClient) CreateAppCard(ctx context.Context, args miro.CreateAppCardArgs) (miro.CreateAppCardResult, error) {
	m.recordCall("CreateAppCard", args)
	if m.CreateAppCardFn != nil {
		return m.CreateAppCardFn(ctx, args)
	}
	return miro.CreateAppCardResult{
		ID:          "appcard-123",
		Title:       args.Title,
		Description: args.Description,
		Status:      args.Status,
		Message:     fmt.Sprintf("Created app card '%s'", truncateForTest(args.Title, 30)),
	}, nil
}

func (m *MockClient) GetAppCard(ctx context.Context, args miro.GetAppCardArgs) (miro.GetAppCardResult, error) {
	m.recordCall("GetAppCard", args)
	if m.GetAppCardFn != nil {
		return m.GetAppCardFn(ctx, args)
	}
	return miro.GetAppCardResult{
		ID:          args.ItemID,
		Title:       "Test App Card",
		Description: "Test description",
		Status:      "connected",
		Message:     "App card 'Test App Card'",
	}, nil
}

func (m *MockClient) UpdateAppCard(ctx context.Context, args miro.UpdateAppCardArgs) (miro.UpdateAppCardResult, error) {
	m.recordCall("UpdateAppCard", args)
	if m.UpdateAppCardFn != nil {
		return m.UpdateAppCardFn(ctx, args)
	}
	title := args.Title
	if title == "" {
		title = "Updated App Card"
	}
	return miro.UpdateAppCardResult{
		ID:      args.ItemID,
		Title:   title,
		Status:  args.Status,
		Message: "App card updated successfully",
	}, nil
}

func (m *MockClient) DeleteAppCard(ctx context.Context, args miro.DeleteAppCardArgs) (miro.DeleteAppCardResult, error) {
	m.recordCall("DeleteAppCard", args)
	if m.DeleteAppCardFn != nil {
		return m.DeleteAppCardFn(ctx, args)
	}
	return miro.DeleteAppCardResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "App card deleted successfully",
	}, nil
}

// =============================================================================
// DocFormatService Implementation
// =============================================================================

func (m *MockClient) CreateDocFormat(ctx context.Context, args miro.CreateDocFormatArgs) (miro.CreateDocFormatResult, error) {
	m.recordCall("CreateDocFormat", args)
	if m.CreateDocFormatFn != nil {
		return m.CreateDocFormatFn(ctx, args)
	}
	return miro.CreateDocFormatResult{
		ID:      "doc-format-123",
		Message: "Created Markdown document",
	}, nil
}

func (m *MockClient) GetDocFormat(ctx context.Context, args miro.GetDocFormatArgs) (miro.GetDocFormatResult, error) {
	m.recordCall("GetDocFormat", args)
	if m.GetDocFormatFn != nil {
		return m.GetDocFormatFn(ctx, args)
	}
	return miro.GetDocFormatResult{
		ID:      args.ItemID,
		Content: "# Test Document\n\nSample content",
		Message: "Retrieved doc format item",
	}, nil
}

func (m *MockClient) DeleteDocFormat(ctx context.Context, args miro.DeleteDocFormatArgs) (miro.DeleteDocFormatResult, error) {
	m.recordCall("DeleteDocFormat", args)
	if m.DeleteDocFormatFn != nil {
		return m.DeleteDocFormatFn(ctx, args)
	}
	if args.DryRun {
		return miro.DeleteDocFormatResult{
			Success: false,
			ItemID:  args.ItemID,
			Message: fmt.Sprintf("Dry run: would delete doc format item %s", args.ItemID),
		}, nil
	}
	return miro.DeleteDocFormatResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "Doc format item deleted successfully",
	}, nil
}

// =============================================================================
// UploadService Implementation
// =============================================================================

func (m *MockClient) UploadImage(ctx context.Context, args miro.UploadImageArgs) (miro.UploadImageResult, error) {
	m.recordCall("UploadImage", args)
	if m.UploadImageFn != nil {
		return m.UploadImageFn(ctx, args)
	}
	return miro.UploadImageResult{
		ID:      "uploaded-image-123",
		Title:   args.Title,
		Message: "Uploaded image from file",
	}, nil
}

func (m *MockClient) UploadDocument(ctx context.Context, args miro.UploadDocumentArgs) (miro.UploadDocumentResult, error) {
	m.recordCall("UploadDocument", args)
	if m.UploadDocumentFn != nil {
		return m.UploadDocumentFn(ctx, args)
	}
	return miro.UploadDocumentResult{
		ID:      "uploaded-doc-123",
		Title:   args.Title,
		Message: "Uploaded document from file",
	}, nil
}

func (m *MockClient) UpdateImageFromFile(ctx context.Context, args miro.UpdateImageFromFileArgs) (miro.UpdateImageFromFileResult, error) {
	m.recordCall("UpdateImageFromFile", args)
	if m.UpdateImageFromFileFn != nil {
		return m.UpdateImageFromFileFn(ctx, args)
	}
	return miro.UpdateImageFromFileResult{
		ID:      args.ItemID,
		Title:   args.Title,
		Message: "Updated image with new file",
	}, nil
}

func (m *MockClient) UpdateDocumentFromFile(ctx context.Context, args miro.UpdateDocumentFromFileArgs) (miro.UpdateDocumentFromFileResult, error) {
	m.recordCall("UpdateDocumentFromFile", args)
	if m.UpdateDocumentFromFileFn != nil {
		return m.UpdateDocumentFromFileFn(ctx, args)
	}
	return miro.UpdateDocumentFromFileResult{
		ID:      args.ItemID,
		Title:   args.Title,
		Message: "Updated document with new file",
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
