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

// groupItemsBody builds the request body shared by group create and update.
func groupItemsBody(itemIDs []string) map[string]interface{} {
	return map[string]interface{}{
		"data": map[string]interface{}{
			"items": itemIDs,
		},
	}
}

// CreateGroup groups multiple items together on a board.
func (c *Client) CreateGroup(ctx context.Context, args CreateGroupArgs) (CreateGroupResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateGroupResult{}, err
	}
	if len(args.ItemIDs) < MinGroupItems {
		return CreateGroupResult{}, fmt.Errorf("at least %d items are required to create a group", MinGroupItems)
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/groups", groupItemsBody(args.ItemIDs))
	if err != nil {
		return CreateGroupResult{}, err
	}

	var group Group
	if err := json.Unmarshal(respBody, &group); err != nil {
		return CreateGroupResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CreateGroupResult{
		ID:       group.ID,
		ItemURL:  BuildItemURL(args.BoardID, group.ID),
		ItemIDs:  args.ItemIDs,
		ItemURLs: BuildItemURLs(args.BoardID, args.ItemIDs),
		Message:  fmt.Sprintf("Grouped %d items together", len(args.ItemIDs)),
	}, nil
}

// ListGroups retrieves all groups on a board.
func (c *Client) ListGroups(ctx context.Context, args ListGroupsArgs) (ListGroupsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListGroupsResult{}, err
	}

	limit := clampGroupItemsLimit(args.Limit)

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

// validateGroupRef validates the (board, group) identifier pair shared by
// the group read/update/delete operations.
func validateGroupRef(boardID, groupID string) error {
	if err := ValidateBoardID(boardID); err != nil {
		return err
	}
	if err := ValidateItemID(groupID); err != nil {
		return fmt.Errorf("invalid group_id: %w", err)
	}
	return nil
}

// GetGroup retrieves a specific group by ID.
func (c *Client) GetGroup(ctx context.Context, args GetGroupArgs) (GetGroupResult, error) {
	if err := validateGroupRef(args.BoardID, args.GroupID); err != nil {
		return GetGroupResult{}, err
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
	if err := validateGroupRef(args.BoardID, args.GroupID); err != nil {
		return GetGroupItemsResult{}, err
	}

	limit := clampGroupItemsLimit(args.Limit)

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

	items := parseGroupItemSummaries(resp.Data)

	return GetGroupItemsResult{
		Items:   items,
		Count:   len(items),
		HasMore: resp.Cursor != "",
		Message: fmt.Sprintf("Found %d items in group", len(items)),
	}, nil
}

// clampGroupItemsLimit normalizes a requested page size; out-of-range values
// (including values above the extended maximum) fall back to the default.
func clampGroupItemsLimit(limit int) int {
	if limit > 0 && limit <= MaxItemLimitExtended {
		return limit
	}
	return DefaultItemLimit
}

// parseGroupItemSummaries converts raw group item entries into summaries,
// skipping entries that fail to parse.
func parseGroupItemSummaries(entries []json.RawMessage) []ItemSummary {
	items := make([]ItemSummary, 0, len(entries))
	for _, raw := range entries {
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
	return items
}

// UpdateGroup updates a group's items.
func (c *Client) UpdateGroup(ctx context.Context, args UpdateGroupArgs) (UpdateGroupResult, error) {
	if err := validateGroupRef(args.BoardID, args.GroupID); err != nil {
		return UpdateGroupResult{}, err
	}
	if len(args.ItemIDs) < MinGroupItems {
		return UpdateGroupResult{}, fmt.Errorf("at least %d items are required in a group", MinGroupItems)
	}

	path := "/boards/" + args.BoardID + "/groups/" + args.GroupID

	respBody, err := c.request(ctx, http.MethodPut, path, groupItemsBody(args.ItemIDs))
	if err != nil {
		return UpdateGroupResult{}, err
	}

	var group Group
	if err := json.Unmarshal(respBody, &group); err != nil {
		return UpdateGroupResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return UpdateGroupResult{
		ID:      group.ID,
		ItemIDs: args.ItemIDs,
		Message: fmt.Sprintf("Updated group with %d items", len(args.ItemIDs)),
	}, nil
}

// DeleteGroup deletes a group (items can optionally be deleted too).
func (c *Client) DeleteGroup(ctx context.Context, args DeleteGroupArgs) (DeleteGroupResult, error) {
	if err := validateGroupRef(args.BoardID, args.GroupID); err != nil {
		return DeleteGroupResult{}, err
	}

	// Dry-run mode: return preview without deleting
	if args.DryRun {
		return DeleteGroupResult{
			Success: true,
			GroupID: args.GroupID,
			Message: deleteGroupDryRunMessage(args),
		}, nil
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

	return DeleteGroupResult{
		Success: true,
		GroupID: args.GroupID,
		Message: deleteGroupDoneMessage(args),
	}, nil
}

// deleteGroupDryRunMessage previews what a group delete would do.
func deleteGroupDryRunMessage(args DeleteGroupArgs) string {
	if args.DeleteItems {
		return "[DRY RUN] Would delete group " + args.GroupID + " and its items"
	}
	return "[DRY RUN] Would delete group " + args.GroupID + ", items would be ungrouped"
}

// deleteGroupDoneMessage describes a completed group delete.
func deleteGroupDoneMessage(args DeleteGroupArgs) string {
	if args.DeleteItems {
		return "Group and its items deleted"
	}
	return "Group deleted, items ungrouped"
}
