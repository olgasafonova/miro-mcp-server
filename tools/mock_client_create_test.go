package tools

import (
	"context"
	"fmt"

	"github.com/olgasafonova/miro-mcp-server/miro"
)

// =============================================================================
// CreateService Implementation
// =============================================================================

func (m *MockClient) CreateSticky(ctx context.Context, args miro.CreateStickyArgs) (miro.CreateStickyResult, error) {
	m.recordCall("CreateSticky", args)
	return stub(ctx, m.CreateStickyFn, args, miro.CreateStickyResult{
		ID:      "sticky-123",
		Content: args.Content,
		Color:   args.Color,
		Message: fmt.Sprintf("Created sticky note '%s'", truncateForTest(args.Content, 30)),
	})
}

func (m *MockClient) CreateShape(ctx context.Context, args miro.CreateShapeArgs) (miro.CreateShapeResult, error) {
	m.recordCall("CreateShape", args)
	return stub(ctx, m.CreateShapeFn, args, miro.CreateShapeResult{
		ID:      "shape-123",
		Shape:   args.Shape,
		Content: args.Content,
		Message: fmt.Sprintf("Created %s shape", args.Shape),
	})
}

func (m *MockClient) CreateShapeExperimental(ctx context.Context, args miro.CreateShapeExperimentalArgs) (miro.CreateShapeResult, error) {
	m.recordCall("CreateShapeExperimental", args)
	return stub(ctx, m.CreateShapeExperimentalFn, args, miro.CreateShapeResult{
		ID:      "shape-exp-123",
		Shape:   args.Shape,
		Content: args.Content,
		Message: fmt.Sprintf("Created experimental %s shape", args.Shape),
	})
}

func (m *MockClient) CreateFlowchartShape(ctx context.Context, args miro.CreateFlowchartShapeArgs) (miro.CreateShapeResult, error) {
	m.recordCall("CreateFlowchartShape", args)
	return stub(ctx, m.CreateFlowchartShapeFn, args, miro.CreateShapeResult{
		ID:      "flowchart-shape-123",
		Shape:   args.Shape,
		Content: args.Content,
		Message: fmt.Sprintf("Created flowchart %s shape", args.Shape),
	})
}

func (m *MockClient) CreateText(ctx context.Context, args miro.CreateTextArgs) (miro.CreateTextResult, error) {
	m.recordCall("CreateText", args)
	return stub(ctx, m.CreateTextFn, args, miro.CreateTextResult{
		ID:      "text-123",
		Content: args.Content,
		Message: "Created text element",
	})
}

func (m *MockClient) CreateConnector(ctx context.Context, args miro.CreateConnectorArgs) (miro.CreateConnectorResult, error) {
	m.recordCall("CreateConnector", args)
	return stub(ctx, m.CreateConnectorFn, args, miro.CreateConnectorResult{
		ID:      "connector-123",
		Message: fmt.Sprintf("Created connector from %s to %s", args.StartItemID, args.EndItemID),
	})
}

func (m *MockClient) CreateFrame(ctx context.Context, args miro.CreateFrameArgs) (miro.CreateFrameResult, error) {
	m.recordCall("CreateFrame", args)
	return stub(ctx, m.CreateFrameFn, args, miro.CreateFrameResult{
		ID:      "frame-123",
		Title:   args.Title,
		Message: fmt.Sprintf("Created frame '%s'", args.Title),
	})
}

func (m *MockClient) CreateCard(ctx context.Context, args miro.CreateCardArgs) (miro.CreateCardResult, error) {
	m.recordCall("CreateCard", args)
	return stub(ctx, m.CreateCardFn, args, miro.CreateCardResult{
		ID:      "card-123",
		Title:   args.Title,
		Message: fmt.Sprintf("Created card '%s'", args.Title),
	})
}

func (m *MockClient) CreateImage(ctx context.Context, args miro.CreateImageArgs) (miro.CreateImageResult, error) {
	m.recordCall("CreateImage", args)
	return stub(ctx, m.CreateImageFn, args, miro.CreateImageResult{
		ID:      "image-123",
		Title:   args.Title,
		URL:     args.URL,
		Message: "Created image",
	})
}

func (m *MockClient) CreateDocument(ctx context.Context, args miro.CreateDocumentArgs) (miro.CreateDocumentResult, error) {
	m.recordCall("CreateDocument", args)
	return stub(ctx, m.CreateDocumentFn, args, miro.CreateDocumentResult{
		ID:      "doc-123",
		Title:   args.Title,
		Message: "Created document",
	})
}

func (m *MockClient) CreateEmbed(ctx context.Context, args miro.CreateEmbedArgs) (miro.CreateEmbedResult, error) {
	m.recordCall("CreateEmbed", args)
	return stub(ctx, m.CreateEmbedFn, args, miro.CreateEmbedResult{
		ID:      "embed-123",
		URL:     args.URL,
		Message: "Created embed",
	})
}

func (m *MockClient) CreateStickyGrid(ctx context.Context, args miro.CreateStickyGridArgs) (miro.CreateStickyGridResult, error) {
	m.recordCall("CreateStickyGrid", args)
	itemIDs := make([]string, len(args.Contents))
	for i := range args.Contents {
		itemIDs[i] = fmt.Sprintf("grid-sticky-%d", i+1)
	}
	columns := args.Columns
	if columns == 0 {
		columns = 3
	}
	rows := (len(args.Contents) + columns - 1) / columns
	return stub(ctx, m.CreateStickyGridFn, args, miro.CreateStickyGridResult{
		Created: len(args.Contents),
		ItemIDs: itemIDs,
		Rows:    rows,
		Columns: columns,
		Message: fmt.Sprintf("Created %d stickies in a grid", len(args.Contents)),
	})
}

// =============================================================================
// UploadService Implementation
// =============================================================================

func (m *MockClient) UploadImage(ctx context.Context, args miro.UploadImageArgs) (miro.UploadImageResult, error) {
	m.recordCall("UploadImage", args)
	return stub(ctx, m.UploadImageFn, args, miro.UploadImageResult{
		ID:      "uploaded-image-123",
		Title:   args.Title,
		Message: "Uploaded image from file",
	})
}

func (m *MockClient) UploadDocument(ctx context.Context, args miro.UploadDocumentArgs) (miro.UploadDocumentResult, error) {
	m.recordCall("UploadDocument", args)
	return stub(ctx, m.UploadDocumentFn, args, miro.UploadDocumentResult{
		ID:      "uploaded-doc-123",
		Title:   args.Title,
		Message: "Uploaded document from file",
	})
}

func (m *MockClient) UpdateImageFromFile(ctx context.Context, args miro.UpdateImageFromFileArgs) (miro.UpdateImageFromFileResult, error) {
	m.recordCall("UpdateImageFromFile", args)
	return stub(ctx, m.UpdateImageFromFileFn, args, miro.UpdateImageFromFileResult{
		ID:      args.ItemID,
		Title:   args.Title,
		Message: "Updated image with new file",
	})
}

func (m *MockClient) UpdateDocumentFromFile(ctx context.Context, args miro.UpdateDocumentFromFileArgs) (miro.UpdateDocumentFromFileResult, error) {
	m.recordCall("UpdateDocumentFromFile", args)
	return stub(ctx, m.UpdateDocumentFromFileFn, args, miro.UpdateDocumentFromFileResult{
		ID:      args.ItemID,
		Title:   args.Title,
		Message: "Updated document with new file",
	})
}
