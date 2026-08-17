package tools

import (
	"context"
	"fmt"

	"github.com/olgasafonova/miro-mcp-server/miro"
)

// =============================================================================
// ExportService Implementation
// =============================================================================

func (m *MockClient) GetBoardPicture(ctx context.Context, args miro.GetBoardPictureArgs) (miro.GetBoardPictureResult, error) {
	m.recordCall("GetBoardPicture", args)
	return stub(ctx, m.GetBoardPictureFn, args, miro.GetBoardPictureResult{
		BoardID:  args.BoardID,
		ImageURL: "https://miro.com/boards/" + args.BoardID + "/picture.png",
		Message:  "Board picture URL retrieved successfully",
	})
}

func (m *MockClient) CreateExportJob(ctx context.Context, args miro.CreateExportJobArgs) (miro.CreateExportJobResult, error) {
	m.recordCall("CreateExportJob", args)
	return stub(ctx, m.CreateExportJobFn, args, miro.CreateExportJobResult{
		JobID:     "export-job-123",
		Status:    "in_progress",
		RequestID: "request-123",
		Message:   fmt.Sprintf("Export job created for %d board(s)", len(args.BoardIDs)),
	})
}

func (m *MockClient) GetExportJobStatus(ctx context.Context, args miro.GetExportJobStatusArgs) (miro.GetExportJobStatusResult, error) {
	m.recordCall("GetExportJobStatus", args)
	return stub(ctx, m.GetExportJobStatusFn, args, miro.GetExportJobStatusResult{
		JobID:          args.JobID,
		Status:         "completed",
		Progress:       100,
		BoardsTotal:    2,
		BoardsExported: 2,
		Message:        "Export job completed: 2/2 boards exported",
	})
}

func (m *MockClient) GetExportJobResults(ctx context.Context, args miro.GetExportJobResultsArgs) (miro.GetExportJobResultsResult, error) {
	m.recordCall("GetExportJobResults", args)
	return stub(ctx, m.GetExportJobResultsFn, args, miro.GetExportJobResultsResult{
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
	})
}

// =============================================================================
// DiagramService Implementation
// =============================================================================

func (m *MockClient) GenerateDiagram(ctx context.Context, args miro.GenerateDiagramArgs) (miro.GenerateDiagramResult, error) {
	m.recordCall("GenerateDiagram", args)
	return stub(ctx, m.GenerateDiagramFn, args, miro.GenerateDiagramResult{
		NodesCreated:      3,
		ConnectorsCreated: 2,
		FramesCreated:     0,
		NodeIDs:           []string{"node-1", "node-2", "node-3"},
		ConnectorIDs:      []string{"conn-1", "conn-2"},
		FrameIDs:          []string{},
		DiagramWidth:      400,
		DiagramHeight:     300,
		Message:           "Created diagram with 3 nodes and 2 connectors",
	})
}

// =============================================================================
// AppCardService Implementation
// =============================================================================

func (m *MockClient) CreateAppCard(ctx context.Context, args miro.CreateAppCardArgs) (miro.CreateAppCardResult, error) {
	m.recordCall("CreateAppCard", args)
	return stub(ctx, m.CreateAppCardFn, args, miro.CreateAppCardResult{
		ID:          "appcard-123",
		Title:       args.Title,
		Description: args.Description,
		Status:      args.Status,
		Message:     fmt.Sprintf("Created app card '%s'", truncateForTest(args.Title, 30)),
	})
}

func (m *MockClient) GetAppCard(ctx context.Context, args miro.GetAppCardArgs) (miro.GetAppCardResult, error) {
	m.recordCall("GetAppCard", args)
	return stub(ctx, m.GetAppCardFn, args, miro.GetAppCardResult{
		ID:          args.ItemID,
		Title:       "Test App Card",
		Description: "Test description",
		Status:      "connected",
		Message:     "App card 'Test App Card'",
	})
}

func (m *MockClient) UpdateAppCard(ctx context.Context, args miro.UpdateAppCardArgs) (miro.UpdateAppCardResult, error) {
	m.recordCall("UpdateAppCard", args)
	title := args.Title
	if title == "" {
		title = "Updated App Card"
	}
	return stub(ctx, m.UpdateAppCardFn, args, miro.UpdateAppCardResult{
		ID:      args.ItemID,
		Title:   title,
		Status:  args.Status,
		Message: "App card updated successfully",
	})
}

func (m *MockClient) DeleteAppCard(ctx context.Context, args miro.DeleteAppCardArgs) (miro.DeleteAppCardResult, error) {
	m.recordCall("DeleteAppCard", args)
	return stub(ctx, m.DeleteAppCardFn, args, miro.DeleteAppCardResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "App card deleted successfully",
	})
}

// =============================================================================
// DocFormatService Implementation
// =============================================================================

func (m *MockClient) CreateDocFormat(ctx context.Context, args miro.CreateDocFormatArgs) (miro.CreateDocFormatResult, error) {
	m.recordCall("CreateDocFormat", args)
	return stub(ctx, m.CreateDocFormatFn, args, miro.CreateDocFormatResult{
		ID:      "doc-format-123",
		Message: "Created Markdown document",
	})
}

func (m *MockClient) GetDocFormat(ctx context.Context, args miro.GetDocFormatArgs) (miro.GetDocFormatResult, error) {
	m.recordCall("GetDocFormat", args)
	return stub(ctx, m.GetDocFormatFn, args, miro.GetDocFormatResult{
		ID:      args.ItemID,
		Content: "# Test Document\n\nSample content",
		Message: "Retrieved doc format item",
	})
}

func (m *MockClient) DeleteDocFormat(ctx context.Context, args miro.DeleteDocFormatArgs) (miro.DeleteDocFormatResult, error) {
	m.recordCall("DeleteDocFormat", args)
	canned := miro.DeleteDocFormatResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "Doc format item deleted successfully",
	}
	if args.DryRun {
		canned = miro.DeleteDocFormatResult{
			Success: false,
			ItemID:  args.ItemID,
			Message: fmt.Sprintf("Dry run: would delete doc format item %s", args.ItemID),
		}
	}
	return stub(ctx, m.DeleteDocFormatFn, args, canned)
}

func (m *MockClient) UpdateDocFormat(ctx context.Context, args miro.UpdateDocFormatArgs) (miro.UpdateDocFormatResult, error) {
	m.recordCall("UpdateDocFormat", args)
	content := args.Content
	if content == "" {
		content = "# Updated Document"
	}
	return stub(ctx, m.UpdateDocFormatFn, args, miro.UpdateDocFormatResult{
		ID:      "doc-format-456",
		OldID:   args.ItemID,
		Content: content,
		ItemURL: "https://miro.com/app/board/" + args.BoardID + "/?moveToWidget=doc-format-456",
		Message: "Updated doc format item",
	})
}

// =============================================================================
// TableService Implementation
// =============================================================================

// cannedTable and cannedGetTable are the shared table-metadata fixtures.
var cannedTable = miro.TableItem{
	ID: "table-123", Type: "data_table_format", X: 100, Y: 200, Width: 400, Height: 300,
	CreatedAt: "2026-03-23T10:00:00Z", ModifiedAt: "2026-03-23T10:00:00Z",
}

var cannedGetTable = miro.GetTableResult{
	Type: "data_table_format", X: 100, Y: 200, Width: 400, Height: 300,
	CreatedAt: "2026-03-23T10:00:00Z", ModifiedAt: "2026-03-23T10:00:00Z",
	Message: "Retrieved table metadata",
}

func (m *MockClient) ListTables(ctx context.Context, args miro.ListTablesArgs) (miro.ListTablesResult, error) {
	m.recordCall("ListTables", args)
	canned := miro.ListTablesResult{Tables: []miro.TableItem{cannedTable}, Count: 1, Total: 1, Message: "Found 1 tables on board"}
	return stub(ctx, m.ListTablesFn, args, canned)
}

func (m *MockClient) GetTable(ctx context.Context, args miro.GetTableArgs) (miro.GetTableResult, error) {
	m.recordCall("GetTable", args)
	canned := cannedGetTable
	canned.ID = args.ItemID
	return stub(ctx, m.GetTableFn, args, canned)
}

// =============================================================================
// Native Diagram Read Implementation
// =============================================================================

// cannedDiagram and cannedGetDiagram are the shared diagram-metadata fixtures.
var cannedDiagram = miro.DiagramItem{
	ID: "diagram-123", Type: "diagram", Title: "Diagram", X: 100, Y: 200, Width: 1200, Height: 700,
	CreatedAt: "2026-05-07T11:14:41Z", ModifiedAt: "2026-05-07T11:14:41Z",
}

var cannedGetDiagram = miro.GetDiagramResult{
	Type: "diagram", Title: "Diagram", X: 100, Y: 200, Width: 1200, Height: 700,
	CreatedAt: "2026-05-07T11:14:41Z", ModifiedAt: "2026-05-07T11:14:41Z",
	Message: "Retrieved diagram metadata",
}

func (m *MockClient) ListDiagrams(ctx context.Context, args miro.ListDiagramsArgs) (miro.ListDiagramsResult, error) {
	m.recordCall("ListDiagrams", args)
	canned := miro.ListDiagramsResult{Diagrams: []miro.DiagramItem{cannedDiagram}, Count: 1, Total: 1, Message: "Found 1 diagrams on board"}
	return stub(ctx, m.ListDiagramsFn, args, canned)
}

func (m *MockClient) GetDiagram(ctx context.Context, args miro.GetDiagramArgs) (miro.GetDiagramResult, error) {
	m.recordCall("GetDiagram", args)
	canned := cannedGetDiagram
	canned.ID = args.ItemID
	return stub(ctx, m.GetDiagramFn, args, canned)
}
