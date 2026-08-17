package tools

import (
	"context"
	"fmt"

	"github.com/olgasafonova/miro-mcp-server/miro"
)

// =============================================================================
// TagService Implementation
// =============================================================================

func (m *MockClient) CreateTag(ctx context.Context, args miro.CreateTagArgs) (miro.CreateTagResult, error) {
	m.recordCall("CreateTag", args)
	return stub(ctx, m.CreateTagFn, args, miro.CreateTagResult{
		ID:      "tag-123",
		Title:   args.Title,
		Color:   args.Color,
		Message: fmt.Sprintf("Created tag '%s'", args.Title),
	})
}

func (m *MockClient) ListTags(ctx context.Context, args miro.ListTagsArgs) (miro.ListTagsResult, error) {
	m.recordCall("ListTags", args)
	return stub(ctx, m.ListTagsFn, args, miro.ListTagsResult{
		Tags: []miro.Tag{
			{ID: "tag1", Title: "Urgent", FillColor: "red"},
			{ID: "tag2", Title: "Done", FillColor: "green"},
		},
		Count: 2,
	})
}

func (m *MockClient) AttachTag(ctx context.Context, args miro.AttachTagArgs) (miro.AttachTagResult, error) {
	m.recordCall("AttachTag", args)
	return stub(ctx, m.AttachTagFn, args, miro.AttachTagResult{
		Success: true,
		ItemID:  args.ItemID,
		TagID:   args.TagID,
		Message: "Tag attached successfully",
	})
}

func (m *MockClient) DetachTag(ctx context.Context, args miro.DetachTagArgs) (miro.DetachTagResult, error) {
	m.recordCall("DetachTag", args)
	return stub(ctx, m.DetachTagFn, args, miro.DetachTagResult{
		Success: true,
		ItemID:  args.ItemID,
		TagID:   args.TagID,
		Message: "Tag detached successfully",
	})
}

func (m *MockClient) GetItemTags(ctx context.Context, args miro.GetItemTagsArgs) (miro.GetItemTagsResult, error) {
	m.recordCall("GetItemTags", args)
	return stub(ctx, m.GetItemTagsFn, args, miro.GetItemTagsResult{
		Tags: []miro.Tag{
			{ID: "tag1", Title: "Urgent", FillColor: "red"},
		},
		Count:  1,
		ItemID: args.ItemID,
	})
}

func (m *MockClient) GetItemsByTag(ctx context.Context, args miro.GetItemsByTagArgs) (miro.GetItemsByTagResult, error) {
	m.recordCall("GetItemsByTag", args)
	return stub(ctx, m.GetItemsByTagFn, args, miro.GetItemsByTagResult{
		Items: []miro.ItemSummary{
			{ID: "item-1", Type: "sticky_note", Content: "Tagged item"},
		},
		Count:   1,
		HasMore: false,
		TagID:   args.TagID,
		Message: fmt.Sprintf("Found 1 items with tag %s", args.TagID),
	})
}

func (m *MockClient) GetTag(ctx context.Context, args miro.GetTagArgs) (miro.GetTagResult, error) {
	m.recordCall("GetTag", args)
	return stub(ctx, m.GetTagFn, args, miro.GetTagResult{
		ID:      args.TagID,
		Title:   "Urgent",
		Color:   "red",
		Message: "Tag 'Urgent'",
	})
}

func (m *MockClient) UpdateTag(ctx context.Context, args miro.UpdateTagArgs) (miro.UpdateTagResult, error) {
	m.recordCall("UpdateTag", args)
	title := args.Title
	if title == "" {
		title = "Updated Tag"
	}
	color := args.Color
	if color == "" {
		color = "green"
	}
	return stub(ctx, m.UpdateTagFn, args, miro.UpdateTagResult{
		Success: true,
		ID:      args.TagID,
		Title:   title,
		Color:   color,
		Message: fmt.Sprintf("Updated tag '%s'", title),
	})
}

func (m *MockClient) DeleteTag(ctx context.Context, args miro.DeleteTagArgs) (miro.DeleteTagResult, error) {
	m.recordCall("DeleteTag", args)
	return stub(ctx, m.DeleteTagFn, args, miro.DeleteTagResult{
		Success: true,
		TagID:   args.TagID,
		Message: "Tag deleted successfully",
	})
}

// =============================================================================
// ConnectorService Implementation
// =============================================================================

func (m *MockClient) ListConnectors(ctx context.Context, args miro.ListConnectorsArgs) (miro.ListConnectorsResult, error) {
	m.recordCall("ListConnectors", args)
	return stub(ctx, m.ListConnectorsFn, args, miro.ListConnectorsResult{
		Connectors: []miro.ConnectorSummary{
			{ID: "conn-1", StartItemID: "item-1", EndItemID: "item-2", Style: "elbowed"},
			{ID: "conn-2", StartItemID: "item-2", EndItemID: "item-3", Style: "straight"},
		},
		Count:   2,
		HasMore: false,
		Message: "Found 2 connectors",
	})
}

func (m *MockClient) GetConnector(ctx context.Context, args miro.GetConnectorArgs) (miro.GetConnectorResult, error) {
	m.recordCall("GetConnector", args)
	return stub(ctx, m.GetConnectorFn, args, miro.GetConnectorResult{
		ID:          args.ConnectorID,
		StartItemID: "item-1",
		EndItemID:   "item-2",
		Style:       "elbowed",
		EndCap:      "arrow",
		Message:     "Retrieved connector details",
	})
}

func (m *MockClient) UpdateConnector(ctx context.Context, args miro.UpdateConnectorArgs) (miro.UpdateConnectorResult, error) {
	m.recordCall("UpdateConnector", args)
	return stub(ctx, m.UpdateConnectorFn, args, miro.UpdateConnectorResult{
		Success: true,
		ID:      args.ConnectorID,
		Message: "Connector updated successfully",
	})
}

func (m *MockClient) DeleteConnector(ctx context.Context, args miro.DeleteConnectorArgs) (miro.DeleteConnectorResult, error) {
	m.recordCall("DeleteConnector", args)
	return stub(ctx, m.DeleteConnectorFn, args, miro.DeleteConnectorResult{
		Success: true,
		ID:      args.ConnectorID,
		Message: "Connector deleted successfully",
	})
}

// =============================================================================
// GroupService Implementation
// =============================================================================

func (m *MockClient) CreateGroup(ctx context.Context, args miro.CreateGroupArgs) (miro.CreateGroupResult, error) {
	m.recordCall("CreateGroup", args)
	return stub(ctx, m.CreateGroupFn, args, miro.CreateGroupResult{
		ID:      "group-123",
		ItemIDs: args.ItemIDs,
		Message: fmt.Sprintf("Grouped %d items", len(args.ItemIDs)),
	})
}

func (m *MockClient) ListGroups(ctx context.Context, args miro.ListGroupsArgs) (miro.ListGroupsResult, error) {
	m.recordCall("ListGroups", args)
	return stub(ctx, m.ListGroupsFn, args, miro.ListGroupsResult{
		Groups:  []miro.Group{{ID: "group-1", Items: []string{"item-1", "item-2"}}},
		Count:   1,
		HasMore: false,
		Message: "Found 1 groups",
	})
}

func (m *MockClient) GetGroup(ctx context.Context, args miro.GetGroupArgs) (miro.GetGroupResult, error) {
	m.recordCall("GetGroup", args)
	return stub(ctx, m.GetGroupFn, args, miro.GetGroupResult{
		ID:      args.GroupID,
		Items:   []string{"item-1", "item-2"},
		Message: "Group contains 2 items",
	})
}

func (m *MockClient) GetGroupItems(ctx context.Context, args miro.GetGroupItemsArgs) (miro.GetGroupItemsResult, error) {
	m.recordCall("GetGroupItems", args)
	return stub(ctx, m.GetGroupItemsFn, args, miro.GetGroupItemsResult{
		Items: []miro.ItemSummary{
			{ID: "item-1", Type: "sticky_note", Content: "Test sticky"},
		},
		Count:   1,
		HasMore: false,
		Message: "Found 1 items in group",
	})
}

func (m *MockClient) UpdateGroup(ctx context.Context, args miro.UpdateGroupArgs) (miro.UpdateGroupResult, error) {
	m.recordCall("UpdateGroup", args)
	return stub(ctx, m.UpdateGroupFn, args, miro.UpdateGroupResult{
		ID:      args.GroupID,
		ItemIDs: args.ItemIDs,
		Message: fmt.Sprintf("Updated group with %d items", len(args.ItemIDs)),
	})
}

func (m *MockClient) DeleteGroup(ctx context.Context, args miro.DeleteGroupArgs) (miro.DeleteGroupResult, error) {
	m.recordCall("DeleteGroup", args)
	msg := "Group deleted, items ungrouped"
	if args.DeleteItems {
		msg = "Group and its items deleted"
	}
	return stub(ctx, m.DeleteGroupFn, args, miro.DeleteGroupResult{
		Success: true,
		GroupID: args.GroupID,
		Message: msg,
	})
}
