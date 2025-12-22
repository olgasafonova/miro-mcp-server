package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// =============================================================================
// Group Operations
// =============================================================================

// CreateGroup groups multiple items together on a board.
func (c *Client) CreateGroup(ctx context.Context, args CreateGroupArgs) (CreateGroupResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateGroupResult{}, err
	}
	if len(args.ItemIDs) < 2 {
		return CreateGroupResult{}, fmt.Errorf("at least 2 items are required to create a group")
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"items": args.ItemIDs,
		},
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/groups", reqBody)
	if err != nil {
		return CreateGroupResult{}, err
	}

	var group Group
	if err := json.Unmarshal(respBody, &group); err != nil {
		return CreateGroupResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CreateGroupResult{
		ID:      group.ID,
		ItemIDs: args.ItemIDs,
		Message: fmt.Sprintf("Grouped %d items together", len(args.ItemIDs)),
	}, nil
}

// Ungroup removes a group, releasing its items.
func (c *Client) Ungroup(ctx context.Context, args UngroupArgs) (UngroupResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UngroupResult{}, err
	}
	if err := ValidateItemID(args.GroupID); err != nil {
		return UngroupResult{}, fmt.Errorf("invalid group_id: %w", err)
	}

	_, err := c.request(ctx, http.MethodDelete, "/boards/"+args.BoardID+"/groups/"+args.GroupID, nil)
	if err != nil {
		return UngroupResult{
			Success: false,
			GroupID: args.GroupID,
			Message: fmt.Sprintf("Failed to ungroup: %v", err),
		}, err
	}

	return UngroupResult{
		Success: true,
		GroupID: args.GroupID,
		Message: "Items ungrouped successfully",
	}, nil
}

// ListGroups retrieves all groups on a board.
func (c *Client) ListGroups(ctx context.Context, args ListGroupsArgs) (ListGroupsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListGroupsResult{}, err
	}

	limit := 50
	if args.Limit > 0 && args.Limit <= 100 {
		limit = args.Limit
	}

	path := fmt.Sprintf("/boards/%s/groups?limit=%d", args.BoardID, limit)
	if args.Cursor != "" {
		path += "&cursor=" + args.Cursor
	}

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ListGroupsResult{}, err
	}

	var resp struct {
		Data   []Group `json:"data"`
		Cursor string  `json:"cursor,omitempty"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ListGroupsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return ListGroupsResult{
		Groups:  resp.Data,
		Count:   len(resp.Data),
		HasMore: resp.Cursor != "",
		Cursor:  resp.Cursor,
		Message: fmt.Sprintf("Found %d groups", len(resp.Data)),
	}, nil
}

// GetGroup retrieves a specific group by ID.
func (c *Client) GetGroup(ctx context.Context, args GetGroupArgs) (GetGroupResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetGroupResult{}, err
	}
	if err := ValidateItemID(args.GroupID); err != nil {
		return GetGroupResult{}, fmt.Errorf("invalid group_id: %w", err)
	}

	path := "/boards/" + args.BoardID + "/groups/" + args.GroupID

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetGroupResult{}, err
	}

	var group Group
	if err := json.Unmarshal(respBody, &group); err != nil {
		return GetGroupResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return GetGroupResult{
		ID:      group.ID,
		Items:   group.Items,
		Message: fmt.Sprintf("Group contains %d items", len(group.Items)),
	}, nil
}

// GetGroupItems retrieves the items in a group.
func (c *Client) GetGroupItems(ctx context.Context, args GetGroupItemsArgs) (GetGroupItemsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetGroupItemsResult{}, err
	}
	if err := ValidateItemID(args.GroupID); err != nil {
		return GetGroupItemsResult{}, fmt.Errorf("invalid group_id: %w", err)
	}

	limit := 50
	if args.Limit > 0 && args.Limit <= 100 {
		limit = args.Limit
	}

	path := fmt.Sprintf("/boards/%s/groups/%s/items?limit=%d", args.BoardID, args.GroupID, limit)
	if args.Cursor != "" {
		path += "&cursor=" + args.Cursor
	}

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetGroupItemsResult{}, err
	}

	var resp struct {
		Data   []json.RawMessage `json:"data"`
		Cursor string            `json:"cursor,omitempty"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return GetGroupItemsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to summaries
	items := make([]ItemSummary, 0, len(resp.Data))
	for _, raw := range resp.Data {
		var item struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Data struct {
				Content string `json:"content"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		items = append(items, ItemSummary{
			ID:      item.ID,
			Type:    item.Type,
			Content: item.Data.Content,
		})
	}

	return GetGroupItemsResult{
		Items:   items,
		Count:   len(items),
		HasMore: resp.Cursor != "",
		Message: fmt.Sprintf("Found %d items in group", len(items)),
	}, nil
}

// DeleteGroup deletes a group (items can optionally be deleted too).
func (c *Client) DeleteGroup(ctx context.Context, args DeleteGroupArgs) (DeleteGroupResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return DeleteGroupResult{}, err
	}
	if err := ValidateItemID(args.GroupID); err != nil {
		return DeleteGroupResult{}, fmt.Errorf("invalid group_id: %w", err)
	}

	path := "/boards/" + args.BoardID + "/groups/" + args.GroupID
	if args.DeleteItems {
		path += "?deleteItems=true"
	}

	_, err := c.request(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return DeleteGroupResult{
			Success: false,
			GroupID: args.GroupID,
			Message: fmt.Sprintf("Failed to delete group: %v", err),
		}, err
	}

	msg := "Group deleted, items ungrouped"
	if args.DeleteItems {
		msg = "Group and its items deleted"
	}

	return DeleteGroupResult{
		Success: true,
		GroupID: args.GroupID,
		Message: msg,
	}, nil
}
