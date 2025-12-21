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
	FindBoardByName(ctx context.Context, name string) (*BoardSummary, error)
	FindBoardByNameTool(ctx context.Context, args FindBoardByNameArgs) (FindBoardByNameResult, error)
	GetBoardSummary(ctx context.Context, args GetBoardSummaryArgs) (GetBoardSummaryResult, error)
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
}

// CreateService handles creation of specific item types.
type CreateService interface {
	CreateSticky(ctx context.Context, args CreateStickyArgs) (CreateStickyResult, error)
	CreateShape(ctx context.Context, args CreateShapeArgs) (CreateShapeResult, error)
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
	AttachTag(ctx context.Context, args AttachTagArgs) (AttachTagResult, error)
	DetachTag(ctx context.Context, args DetachTagArgs) (DetachTagResult, error)
	GetItemTags(ctx context.Context, args GetItemTagsArgs) (GetItemTagsResult, error)
	UpdateTag(ctx context.Context, args UpdateTagArgs) (UpdateTagResult, error)
	DeleteTag(ctx context.Context, args DeleteTagArgs) (DeleteTagResult, error)
}

// ConnectorService handles connector operations.
type ConnectorService interface {
	CreateConnector(ctx context.Context, args CreateConnectorArgs) (CreateConnectorResult, error)
	UpdateConnector(ctx context.Context, args UpdateConnectorArgs) (UpdateConnectorResult, error)
	DeleteConnector(ctx context.Context, args DeleteConnectorArgs) (DeleteConnectorResult, error)
}

// GroupService handles item grouping.
type GroupService interface {
	CreateGroup(ctx context.Context, args CreateGroupArgs) (CreateGroupResult, error)
	Ungroup(ctx context.Context, args UngroupArgs) (UngroupResult, error)
}

// MemberService handles board member operations.
type MemberService interface {
	ListBoardMembers(ctx context.Context, args ListBoardMembersArgs) (ListBoardMembersResult, error)
	ShareBoard(ctx context.Context, args ShareBoardArgs) (ShareBoardResult, error)
}

// MindmapService handles mindmap operations.
type MindmapService interface {
	CreateMindmapNode(ctx context.Context, args CreateMindmapNodeArgs) (CreateMindmapNodeResult, error)
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

// WebhookService handles webhook subscription operations.
type WebhookService interface {
	CreateWebhook(ctx context.Context, args CreateWebhookArgs) (CreateWebhookResult, error)
	ListWebhooks(ctx context.Context, args ListWebhooksArgs) (ListWebhooksResult, error)
	DeleteWebhook(ctx context.Context, args DeleteWebhookArgs) (DeleteWebhookResult, error)
	GetWebhook(ctx context.Context, args GetWebhookArgs) (GetWebhookResult, error)
}

// DiagramService handles diagram generation from code.
type DiagramService interface {
	GenerateDiagram(ctx context.Context, args GenerateDiagramArgs) (GenerateDiagramResult, error)
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
	TokenService
	ExportService
	WebhookService
	DiagramService
	ConnectorService
}

// Verify that Client implements MiroClient at compile time.
var _ MiroClient = (*Client)(nil)
