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
		"items": args.ItemIDs,
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
