package miro

import "context"

// =============================================================================
// Service Interfaces
// =============================================================================
// These interfaces enable mock-based testing without real API calls.
// Each interface corresponds to a domain file for granular mocking.

// BoardService handles board-level operations.
type BoardService interface {
	ListBoards(ctx context.Context, args ListBoardsArgs) (ListBoardsResult, error)
	GetBoard(ctx context.Context, args GetBoardArgs) (GetBoardResult, error)
	CreateBoard(ctx context.Context, args CreateBoardArgs) (CreateBoardResult, error)
	CopyBoard(ctx context.Context, args CopyBoardArgs) (CopyBoardResult, error)
	DeleteBoard(ctx context.Context, args DeleteBoardArgs) (DeleteBoardResult, error)
	UpdateBoard(ctx context.Context, args UpdateBoardArgs) (UpdateBoardResult, error)
	FindBoardByName(ctx context.Context, name string) (*BoardSummary, error)
	FindBoardByNameTool(ctx context.Context, args FindBoardByNameArgs) (FindBoardByNameResult, error)
	GetBoardSummary(ctx context.Context, args GetBoardSummaryArgs) (GetBoardSummaryResult, error)
	GetBoardContent(ctx context.Context, args GetBoardContentArgs) (GetBoardContentResult, error)
}

// ItemService handles item CRUD and search operations.
type ItemService interface {
	ListItems(ctx context.Context, args ListItemsArgs) (ListItemsResult, error)
	ListAllItems(ctx context.Context, args ListAllItemsArgs) (ListAllItemsResult, error)
	GetItem(ctx context.Context, args GetItemArgs) (GetItemResult, error)
	UpdateItem(ctx context.Context, args UpdateItemArgs) (UpdateItemResult, error)
	DeleteItem(ctx context.Context, args DeleteItemArgs) (DeleteItemResult, error)
	SearchBoard(ctx context.Context, args SearchBoardArgs) (SearchBoardResult, error)
	BulkCreate(ctx context.Context, args BulkCreateArgs) (BulkCreateResult, error)
	BulkUpdate(ctx context.Context, args BulkUpdateArgs) (BulkUpdateResult, error)
	BulkDelete(ctx context.Context, args BulkDeleteArgs) (BulkDeleteResult, error)
	// Type-specific reads
	GetImage(ctx context.Context, args GetImageArgs) (GetImageResult, error)
	GetDocument(ctx context.Context, args GetDocumentArgs) (GetDocumentResult, error)
	// Type-specific updates
	UpdateSticky(ctx context.Context, args UpdateStickyArgs) (UpdateStickyResult, error)
	UpdateShape(ctx context.Context, args UpdateShapeArgs) (UpdateShapeResult, error)
	UpdateText(ctx context.Context, args UpdateTextArgs) (UpdateTextResult, error)
	UpdateCard(ctx context.Context, args UpdateCardArgs) (UpdateCardResult, error)
	UpdateImage(ctx context.Context, args UpdateImageArgs) (UpdateImageResult, error)
	UpdateDocument(ctx context.Context, args UpdateDocumentArgs) (UpdateDocumentResult, error)
	UpdateEmbed(ctx context.Context, args UpdateEmbedArgs) (UpdateEmbedResult, error)
}

// CreateService handles creation of specific item types.
type CreateService interface {
	CreateSticky(ctx context.Context, args CreateStickyArgs) (CreateStickyResult, error)
	CreateShape(ctx context.Context, args CreateShapeArgs) (CreateShapeResult, error)
	CreateShapeExperimental(ctx context.Context, args CreateShapeExperimentalArgs) (CreateShapeResult, error)
	CreateFlowchartShape(ctx context.Context, args CreateFlowchartShapeArgs) (CreateShapeResult, error)
	CreateText(ctx context.Context, args CreateTextArgs) (CreateTextResult, error)
	CreateFrame(ctx context.Context, args CreateFrameArgs) (CreateFrameResult, error)
	CreateCard(ctx context.Context, args CreateCardArgs) (CreateCardResult, error)
	CreateImage(ctx context.Context, args CreateImageArgs) (CreateImageResult, error)
	CreateDocument(ctx context.Context, args CreateDocumentArgs) (CreateDocumentResult, error)
	CreateEmbed(ctx context.Context, args CreateEmbedArgs) (CreateEmbedResult, error)
	CreateStickyGrid(ctx context.Context, args CreateStickyGridArgs) (CreateStickyGridResult, error)
}

// TagService handles tag operations.
type TagService interface {
	CreateTag(ctx context.Context, args CreateTagArgs) (CreateTagResult, error)
	ListTags(ctx context.Context, args ListTagsArgs) (ListTagsResult, error)
	GetTag(ctx context.Context, args GetTagArgs) (GetTagResult, error)
	AttachTag(ctx context.Context, args AttachTagArgs) (AttachTagResult, error)
	DetachTag(ctx context.Context, args DetachTagArgs) (DetachTagResult, error)
	GetItemTags(ctx context.Context, args GetItemTagsArgs) (GetItemTagsResult, error)
	GetItemsByTag(ctx context.Context, args GetItemsByTagArgs) (GetItemsByTagResult, error)
	UpdateTag(ctx context.Context, args UpdateTagArgs) (UpdateTagResult, error)
	DeleteTag(ctx context.Context, args DeleteTagArgs) (DeleteTagResult, error)
}

// ConnectorService handles connector operations.
type ConnectorService interface {
	ListConnectors(ctx context.Context, args ListConnectorsArgs) (ListConnectorsResult, error)
	GetConnector(ctx context.Context, args GetConnectorArgs) (GetConnectorResult, error)
	CreateConnector(ctx context.Context, args CreateConnectorArgs) (CreateConnectorResult, error)
	UpdateConnector(ctx context.Context, args UpdateConnectorArgs) (UpdateConnectorResult, error)
	DeleteConnector(ctx context.Context, args DeleteConnectorArgs) (DeleteConnectorResult, error)
}

// GroupService handles item grouping.
type GroupService interface {
	CreateGroup(ctx context.Context, args CreateGroupArgs) (CreateGroupResult, error)
	ListGroups(ctx context.Context, args ListGroupsArgs) (ListGroupsResult, error)
	GetGroup(ctx context.Context, args GetGroupArgs) (GetGroupResult, error)
	GetGroupItems(ctx context.Context, args GetGroupItemsArgs) (GetGroupItemsResult, error)
	UpdateGroup(ctx context.Context, args UpdateGroupArgs) (UpdateGroupResult, error)
	DeleteGroup(ctx context.Context, args DeleteGroupArgs) (DeleteGroupResult, error)
}

// MemberService handles board member operations.
type MemberService interface {
	ListBoardMembers(ctx context.Context, args ListBoardMembersArgs) (ListBoardMembersResult, error)
	ShareBoard(ctx context.Context, args ShareBoardArgs) (ShareBoardResult, error)
	GetBoardMember(ctx context.Context, args GetBoardMemberArgs) (GetBoardMemberResult, error)
	RemoveBoardMember(ctx context.Context, args RemoveBoardMemberArgs) (RemoveBoardMemberResult, error)
	UpdateBoardMember(ctx context.Context, args UpdateBoardMemberArgs) (UpdateBoardMemberResult, error)
}

// MindmapService handles mindmap operations.
type MindmapService interface {
	CreateMindmapNode(ctx context.Context, args CreateMindmapNodeArgs) (CreateMindmapNodeResult, error)
	GetMindmapNode(ctx context.Context, args GetMindmapNodeArgs) (GetMindmapNodeResult, error)
	ListMindmapNodes(ctx context.Context, args ListMindmapNodesArgs) (ListMindmapNodesResult, error)
	DeleteMindmapNode(ctx context.Context, args DeleteMindmapNodeArgs) (DeleteMindmapNodeResult, error)
}

// CodeWidgetService handles code widget operations (v2-experimental).
type CodeWidgetService interface {
	CreateCodeWidget(ctx context.Context, args CreateCodeWidgetArgs) (CreateCodeWidgetResult, error)
	GetCodeWidget(ctx context.Context, args GetCodeWidgetArgs) (GetCodeWidgetResult, error)
	ListCodeWidgets(ctx context.Context, args ListCodeWidgetsArgs) (ListCodeWidgetsResult, error)
	UpdateCodeWidget(ctx context.Context, args UpdateCodeWidgetArgs) (UpdateCodeWidgetResult, error)
	MoveCodeWidget(ctx context.Context, args MoveCodeWidgetArgs) (MoveCodeWidgetResult, error)
	DeleteCodeWidget(ctx context.Context, args DeleteCodeWidgetArgs) (DeleteCodeWidgetResult, error)
}

// CommentService handles comment thread operations (v2-experimental).
type CommentService interface {
	CreateComment(ctx context.Context, args CreateCommentArgs) (CreateCommentResult, error)
	ListComments(ctx context.Context, args ListCommentsArgs) (ListCommentsResult, error)
	GetComment(ctx context.Context, args GetCommentArgs) (GetCommentResult, error)
	ReplyComment(ctx context.Context, args ReplyCommentArgs) (ReplyCommentResult, error)
	ResolveComment(ctx context.Context, args ResolveCommentArgs) (ResolveCommentResult, error)
}

// OrgAuditService handles Miro's organization audit log (Enterprise).
// Distinct from the local execution log in miro/audit/, which is not an API
// surface and so has no service here.
type OrgAuditService interface {
	GetOrgAuditLogs(ctx context.Context, args GetOrgAuditLogsArgs) (GetOrgAuditLogsResult, error)
}

// SVGService handles local SVG rendering and parsing.
type SVGService interface {
	ReadBoardSVG(ctx context.Context, args ReadBoardSVGArgs) (ReadBoardSVGResult, error)
	CreateFromSVG(ctx context.Context, args CreateFromSVGArgs) (CreateFromSVGResult, error)
}

// FrameService handles frame-specific operations (beyond create).
type FrameService interface {
	GetFrame(ctx context.Context, args GetFrameArgs) (GetFrameResult, error)
	UpdateFrame(ctx context.Context, args UpdateFrameArgs) (UpdateFrameResult, error)
	DeleteFrame(ctx context.Context, args DeleteFrameArgs) (DeleteFrameResult, error)
	GetFrameItems(ctx context.Context, args GetFrameItemsArgs) (GetFrameItemsResult, error)
}

// TokenService handles authentication validation.
type TokenService interface {
	ValidateToken(ctx context.Context) (*UserInfo, error)
}

// ExportService handles board export operations.
type ExportService interface {
	GetBoardPicture(ctx context.Context, args GetBoardPictureArgs) (GetBoardPictureResult, error)
	CreateExportJob(ctx context.Context, args CreateExportJobArgs) (CreateExportJobResult, error)
	GetExportJobStatus(ctx context.Context, args GetExportJobStatusArgs) (GetExportJobStatusResult, error)
	GetExportJobResults(ctx context.Context, args GetExportJobResultsArgs) (GetExportJobResultsResult, error)
}

// DiagramService handles diagram generation from code and native diagram reads.
type DiagramService interface {
	GenerateDiagram(ctx context.Context, args GenerateDiagramArgs) (GenerateDiagramResult, error)
	ListDiagrams(ctx context.Context, args ListDiagramsArgs) (ListDiagramsResult, error)
	GetDiagram(ctx context.Context, args GetDiagramArgs) (GetDiagramResult, error)
}

// AppCardService handles app card operations.
type AppCardService interface {
	CreateAppCard(ctx context.Context, args CreateAppCardArgs) (CreateAppCardResult, error)
	GetAppCard(ctx context.Context, args GetAppCardArgs) (GetAppCardResult, error)
	UpdateAppCard(ctx context.Context, args UpdateAppCardArgs) (UpdateAppCardResult, error)
	DeleteAppCard(ctx context.Context, args DeleteAppCardArgs) (DeleteAppCardResult, error)
}

// DocFormatService handles doc format (Markdown document) operations.
type DocFormatService interface {
	CreateDocFormat(ctx context.Context, args CreateDocFormatArgs) (CreateDocFormatResult, error)
	GetDocFormat(ctx context.Context, args GetDocFormatArgs) (GetDocFormatResult, error)
	UpdateDocFormat(ctx context.Context, args UpdateDocFormatArgs) (UpdateDocFormatResult, error)
	DeleteDocFormat(ctx context.Context, args DeleteDocFormatArgs) (DeleteDocFormatResult, error)
}

// TableService handles data table format operations.
type TableService interface {
	ListTables(ctx context.Context, args ListTablesArgs) (ListTablesResult, error)
	GetTable(ctx context.Context, args GetTableArgs) (GetTableResult, error)
}

// UploadService handles file upload operations.
type UploadService interface {
	UploadImage(ctx context.Context, args UploadImageArgs) (UploadImageResult, error)
	UploadDocument(ctx context.Context, args UploadDocumentArgs) (UploadDocumentResult, error)
	UpdateImageFromFile(ctx context.Context, args UpdateImageFromFileArgs) (UpdateImageFromFileResult, error)
	UpdateDocumentFromFile(ctx context.Context, args UpdateDocumentFromFileArgs) (UpdateDocumentFromFileResult, error)
}

// =============================================================================
// Composite Interface
// =============================================================================

// MiroClient is the complete interface for the Miro API client.
// It embeds all domain-specific interfaces.
type MiroClient interface {
	BoardService
	ItemService
	CreateService
	TagService
	GroupService
	MemberService
	MindmapService
	CodeWidgetService
	CommentService
	OrgAuditService
	SVGService
	FrameService
	TokenService
	ExportService
	DiagramService
	ConnectorService
	AppCardService
	DocFormatService
	UploadService
	TableService
}

// Verify that Client implements MiroClient at compile time.
var _ MiroClient = (*Client)(nil)
