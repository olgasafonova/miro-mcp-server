package tools

import (
	"context"
	"fmt"

	"github.com/olgasafonova/miro-mcp-server/miro"
)

// =============================================================================
// ItemService Implementation
// =============================================================================

// cannedItems is the default item fixture shared by the list-style item mocks.
var cannedItems = []miro.ItemSummary{
	{ID: "item1", Type: "sticky_note", Content: "Test sticky"},
	{ID: "item2", Type: "shape", Content: "Test shape"},
}

func (m *MockClient) ListItems(ctx context.Context, args miro.ListItemsArgs) (miro.ListItemsResult, error) {
	m.recordCall("ListItems", args)
	return stub(ctx, m.ListItemsFn, args, miro.ListItemsResult{Items: cannedItems, Count: 2, HasMore: false})
}

func (m *MockClient) ListAllItems(ctx context.Context, args miro.ListAllItemsArgs) (miro.ListAllItemsResult, error) {
	m.recordCall("ListAllItems", args)
	canned := miro.ListAllItemsResult{Items: cannedItems[:1], Count: 1, TotalPages: 1, Message: "Retrieved 1 items in 1 pages"}
	return stub(ctx, m.ListAllItemsFn, args, canned)
}

func (m *MockClient) GetItem(ctx context.Context, args miro.GetItemArgs) (miro.GetItemResult, error) {
	m.recordCall("GetItem", args)
	return stub(ctx, m.GetItemFn, args, miro.GetItemResult{
		ID:      args.ItemID,
		Type:    "sticky_note",
		Content: "Test sticky content",
	})
}

func (m *MockClient) GetImage(ctx context.Context, args miro.GetImageArgs) (miro.GetImageResult, error) {
	m.recordCall("GetImage", args)
	return stub(ctx, m.GetImageFn, args, miro.GetImageResult{
		ID:       args.ItemID,
		Title:    "Test Image",
		ImageURL: "https://miro.com/images/test.png",
		Width:    800,
		Height:   600,
		Message:  "Image retrieved successfully",
	})
}

func (m *MockClient) GetDocument(ctx context.Context, args miro.GetDocumentArgs) (miro.GetDocumentResult, error) {
	m.recordCall("GetDocument", args)
	return stub(ctx, m.GetDocumentFn, args, miro.GetDocumentResult{
		ID:          args.ItemID,
		Title:       "Test Document",
		DocumentURL: "https://miro.com/documents/test.pdf",
		Message:     "Document retrieved successfully",
	})
}

func (m *MockClient) UpdateItem(ctx context.Context, args miro.UpdateItemArgs) (miro.UpdateItemResult, error) {
	m.recordCall("UpdateItem", args)
	return stub(ctx, m.UpdateItemFn, args, miro.UpdateItemResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "Item updated successfully",
	})
}

func (m *MockClient) UpdateSticky(ctx context.Context, args miro.UpdateStickyArgs) (miro.UpdateStickyResult, error) {
	m.recordCall("UpdateSticky", args)
	return stub(ctx, m.UpdateStickyFn, args, miro.UpdateStickyResult{
		ID:      args.ItemID,
		Message: "Sticky updated successfully",
	})
}

func (m *MockClient) UpdateShape(ctx context.Context, args miro.UpdateShapeArgs) (miro.UpdateShapeResult, error) {
	m.recordCall("UpdateShape", args)
	return stub(ctx, m.UpdateShapeFn, args, miro.UpdateShapeResult{
		ID:      args.ItemID,
		Message: "Shape updated successfully",
	})
}

func (m *MockClient) UpdateText(ctx context.Context, args miro.UpdateTextArgs) (miro.UpdateTextResult, error) {
	m.recordCall("UpdateText", args)
	return stub(ctx, m.UpdateTextFn, args, miro.UpdateTextResult{
		ID:      args.ItemID,
		Message: "Text updated successfully",
	})
}

func (m *MockClient) UpdateCard(ctx context.Context, args miro.UpdateCardArgs) (miro.UpdateCardResult, error) {
	m.recordCall("UpdateCard", args)
	return stub(ctx, m.UpdateCardFn, args, miro.UpdateCardResult{
		ID:      args.ItemID,
		Message: "Card updated successfully",
	})
}

func (m *MockClient) UpdateImage(ctx context.Context, args miro.UpdateImageArgs) (miro.UpdateImageResult, error) {
	m.recordCall("UpdateImage", args)
	return stub(ctx, m.UpdateImageFn, args, miro.UpdateImageResult{
		ID:      args.ItemID,
		Message: "Image updated successfully",
	})
}

func (m *MockClient) UpdateDocument(ctx context.Context, args miro.UpdateDocumentArgs) (miro.UpdateDocumentResult, error) {
	m.recordCall("UpdateDocument", args)
	return stub(ctx, m.UpdateDocumentFn, args, miro.UpdateDocumentResult{
		ID:      args.ItemID,
		Message: "Document updated successfully",
	})
}

func (m *MockClient) UpdateEmbed(ctx context.Context, args miro.UpdateEmbedArgs) (miro.UpdateEmbedResult, error) {
	m.recordCall("UpdateEmbed", args)
	return stub(ctx, m.UpdateEmbedFn, args, miro.UpdateEmbedResult{
		ID:      args.ItemID,
		Message: "Embed updated successfully",
	})
}

func (m *MockClient) DeleteItem(ctx context.Context, args miro.DeleteItemArgs) (miro.DeleteItemResult, error) {
	m.recordCall("DeleteItem", args)
	return stub(ctx, m.DeleteItemFn, args, miro.DeleteItemResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "Item deleted successfully",
	})
}

func (m *MockClient) SearchBoard(ctx context.Context, args miro.SearchBoardArgs) (miro.SearchBoardResult, error) {
	m.recordCall("SearchBoard", args)
	return stub(ctx, m.SearchBoardFn, args, miro.SearchBoardResult{
		Matches: []miro.ItemMatch{
			{ID: "item1", Type: "sticky_note", Content: "Found: " + args.Query, Snippet: args.Query},
		},
		Count:   1,
		Query:   args.Query,
		Message: fmt.Sprintf("Found 1 items matching '%s'", args.Query),
	})
}

// fabricatedBulkIDs invents sequential bulk-item IDs for a create fixture.
func fabricatedBulkIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("bulk-item-%d", i+1)
	}
	return ids
}

// echoedUpdateIDs collects the item IDs named by a bulk-update request.
func echoedUpdateIDs(items []miro.BulkUpdateItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ItemID)
	}
	return ids
}

func (m *MockClient) BulkCreate(ctx context.Context, args miro.BulkCreateArgs) (miro.BulkCreateResult, error) {
	m.recordCall("BulkCreate", args)
	canned := miro.BulkCreateResult{Created: len(args.Items), ItemIDs: fabricatedBulkIDs(len(args.Items)), Errors: []string{}, Message: fmt.Sprintf("Created %d items", len(args.Items))}
	return stub(ctx, m.BulkCreateFn, args, canned)
}

func (m *MockClient) BulkUpdate(ctx context.Context, args miro.BulkUpdateArgs) (miro.BulkUpdateResult, error) {
	m.recordCall("BulkUpdate", args)
	canned := miro.BulkUpdateResult{Updated: len(args.Items), ItemIDs: echoedUpdateIDs(args.Items), Errors: []string{}, Message: fmt.Sprintf("Updated %d items", len(args.Items))}
	return stub(ctx, m.BulkUpdateFn, args, canned)
}

func (m *MockClient) BulkDelete(ctx context.Context, args miro.BulkDeleteArgs) (miro.BulkDeleteResult, error) {
	m.recordCall("BulkDelete", args)
	return stub(ctx, m.BulkDeleteFn, args, miro.BulkDeleteResult{
		Deleted: len(args.ItemIDs),
		ItemIDs: args.ItemIDs,
		Errors:  []string{},
		Message: fmt.Sprintf("Deleted %d items", len(args.ItemIDs)),
	})
}
