package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// =============================================================================
// Frame Operations - Get, Update, Delete, Get Items
// =============================================================================

// GetFrame retrieves a specific frame by ID.
func (c *Client) GetFrame(ctx context.Context, args GetFrameArgs) (GetFrameResult, error) {
	if args.BoardID == "" {
		return GetFrameResult{}, fmt.Errorf("board_id is required")
	}
	if args.FrameID == "" {
		return GetFrameResult{}, fmt.Errorf("frame_id is required")
	}

	path := fmt.Sprintf("/boards/%s/frames/%s", args.BoardID, args.FrameID)
	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetFrameResult{}, err
	}

	var frame Frame
	if err := json.Unmarshal(respBody, &frame); err != nil {
		return GetFrameResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	result := GetFrameResult{
		ID:      frame.ID,
		Title:   frame.Data.Title,
		Message: fmt.Sprintf("Retrieved frame '%s'", frame.Data.Title),
	}

	// Extract position
	if frame.Position != nil {
		result.X = frame.Position.X
		result.Y = frame.Position.Y
	}

	// Extract geometry
	if frame.Geometry != nil {
		result.Width = frame.Geometry.Width
		result.Height = frame.Geometry.Height
	}

	// Extract style
	if frame.Style.FillColor != "" {
		result.Color = frame.Style.FillColor
	}

	// Count children if available
	if len(frame.Children) > 0 {
		result.ChildCount = len(frame.Children)
	}

	// Format timestamps
	if !frame.CreatedAt.IsZero() {
		result.CreatedAt = frame.CreatedAt.Format(time.RFC3339)
	}
	if !frame.ModifiedAt.IsZero() {
		result.ModifiedAt = frame.ModifiedAt.Format(time.RFC3339)
	}
	if frame.CreatedBy != nil {
		result.CreatedBy = frame.CreatedBy.ID
	}
	if frame.ModifiedBy != nil {
		result.ModifiedBy = frame.ModifiedBy.ID
	}

	return result, nil
}

// UpdateFrame updates an existing frame.
func (c *Client) UpdateFrame(ctx context.Context, args UpdateFrameArgs) (UpdateFrameResult, error) {
	if args.BoardID == "" {
		return UpdateFrameResult{}, fmt.Errorf("board_id is required")
	}
	if args.FrameID == "" {
		return UpdateFrameResult{}, fmt.Errorf("frame_id is required")
	}

	reqBody := make(map[string]interface{})

	// Build data object for title
	if args.Title != nil {
		reqBody["data"] = map[string]interface{}{
			"title": *args.Title,
		}
	}

	// Build position object
	position := make(map[string]interface{})
	if args.X != nil {
		position["x"] = *args.X
	}
	if args.Y != nil {
		position["y"] = *args.Y
	}
	if len(position) > 0 {
		reqBody["position"] = position
	}

	// Build geometry object
	geometry := make(map[string]interface{})
	if args.Width != nil {
		geometry["width"] = *args.Width
	}
	if args.Height != nil {
		geometry["height"] = *args.Height
	}
	if len(geometry) > 0 {
		reqBody["geometry"] = geometry
	}

	// Build style object
	if args.Color != nil {
		reqBody["style"] = map[string]interface{}{
			"fillColor": *args.Color,
		}
	}

	// If nothing to update, return error
	if len(reqBody) == 0 {
		return UpdateFrameResult{}, fmt.Errorf("at least one update field is required")
	}

	path := fmt.Sprintf("/boards/%s/frames/%s", args.BoardID, args.FrameID)
	_, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateFrameResult{}, err
	}

	// Invalidate items cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return UpdateFrameResult{
		Success: true,
		ID:      args.FrameID,
		Message: "Frame updated successfully",
	}, nil
}

// DeleteFrame removes a frame from a board.
func (c *Client) DeleteFrame(ctx context.Context, args DeleteFrameArgs) (DeleteFrameResult, error) {
	if args.BoardID == "" {
		return DeleteFrameResult{}, fmt.Errorf("board_id is required")
	}
	if args.FrameID == "" {
		return DeleteFrameResult{}, fmt.Errorf("frame_id is required")
	}

	// Dry-run mode: return preview without deleting
	if args.DryRun {
		return DeleteFrameResult{
			Success: true,
			ID:      args.FrameID,
			Message: "[DRY RUN] Would delete frame " + args.FrameID + " from board " + args.BoardID,
		}, nil
	}

	path := fmt.Sprintf("/boards/%s/frames/%s", args.BoardID, args.FrameID)
	_, err := c.request(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return DeleteFrameResult{
			Success: false,
			ID:      args.FrameID,
			Message: fmt.Sprintf("Failed to delete frame: %v", err),
		}, err
	}

	// Invalidate items cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return DeleteFrameResult{
		Success: true,
		ID:      args.FrameID,
		Message: "Frame deleted successfully",
	}, nil
}

// GetFrameItems retrieves all items within a specific frame.
// When detail_level=full, additional fields (style, geometry, timestamps, user info) are included.
func (c *Client) GetFrameItems(ctx context.Context, args GetFrameItemsArgs) (GetFrameItemsResult, error) {
	if args.BoardID == "" {
		return GetFrameItemsResult{}, fmt.Errorf("board_id is required")
	}
	if args.FrameID == "" {
		return GetFrameItemsResult{}, fmt.Errorf("frame_id is required")
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	// Use the specific frame items endpoint
	path := fmt.Sprintf("/boards/%s/frames/%s/items?limit=%d", args.BoardID, args.FrameID, limit)
	if args.Type != "" {
		path += "&type=" + args.Type
	}
	if args.Cursor != "" {
		path += "&cursor=" + args.Cursor
	}

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetFrameItemsResult{}, err
	}

	var resp struct {
		Data   []json.RawMessage `json:"data"`
		Cursor string            `json:"cursor,omitempty"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return GetFrameItemsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check if full details are requested
	fullDetails := strings.EqualFold(args.DetailLevel, "full")

	// Parse items into summaries using shared helper
	items := make([]ItemSummary, 0, len(resp.Data))
	for _, raw := range resp.Data {
		item := parseItemSummary(raw, fullDetails)
		if item.ID != "" {
			items = append(items, item)
		}
	}

	hasMore := resp.Cursor != ""
	return GetFrameItemsResult{
		Items:   items,
		Count:   len(items),
		HasMore: hasMore,
		Cursor:  resp.Cursor,
		Message: fmt.Sprintf("Found %d items in frame", len(items)),
	}, nil
}
