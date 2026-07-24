package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// =============================================================================
// Frame Operations - Create, Get, Update, Delete, Get Items
// =============================================================================

// CreateFrame creates a frame container on a board.
func (c *Client) CreateFrame(ctx context.Context, args CreateFrameArgs) (CreateFrameResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateFrameResult{}, err
	}

	// Default dimensions
	width := args.Width
	if width == 0 {
		width = 800
	}
	height := args.Height
	if height == 0 {
		height = 600
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"title":  args.Title,
			"format": "custom",
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
		"geometry": map[string]interface{}{
			"width":  width,
			"height": height,
		},
	}

	if args.Color != "" {
		fillColor, err := normalizeColor(args.Color)
		if err != nil {
			return CreateFrameResult{}, fmt.Errorf("color: %w", err)
		}
		reqBody["style"] = map[string]interface{}{
			"fillColor": fillColor,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/frames", reqBody)
	if err != nil {
		return CreateFrameResult{}, err
	}

	var frame Frame
	if err := json.Unmarshal(respBody, &frame); err != nil {
		return CreateFrameResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate items list cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return CreateFrameResult{
		ID:      frame.ID,
		ItemURL: BuildItemURL(args.BoardID, frame.ID),
		Title:   frame.Data.Title,
		Message: fmt.Sprintf("Created frame '%s'", args.Title),
	}, nil
}

// GetFrame retrieves a specific frame by ID.
func (c *Client) GetFrame(ctx context.Context, args GetFrameArgs) (GetFrameResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetFrameResult{}, err
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

	return frameDetails(frame), nil
}

// frameDetails projects a full frame object onto the get result.
func frameDetails(frame Frame) GetFrameResult {
	result := GetFrameResult{
		ID:         frame.ID,
		Title:      frame.Data.Title,
		Color:      frame.Style.FillColor,
		ChildCount: len(frame.Children),
		CreatedAt:  formatTimestamp(frame.CreatedAt),
		ModifiedAt: formatTimestamp(frame.ModifiedAt),
		Message:    fmt.Sprintf("Retrieved frame '%s'", frame.Data.Title),
	}
	if frame.Position != nil {
		result.X = frame.Position.X
		result.Y = frame.Position.Y
	}
	if frame.Geometry != nil {
		result.Width = frame.Geometry.Width
		result.Height = frame.Geometry.Height
	}
	if frame.CreatedBy != nil {
		result.CreatedBy = frame.CreatedBy.ID
	}
	if frame.ModifiedBy != nil {
		result.ModifiedBy = frame.ModifiedBy.ID
	}
	return result
}

// buildUpdateFrameBody assembles the request body for a frame update; an
// empty map means no updatable field was supplied.
func buildUpdateFrameBody(args UpdateFrameArgs) (map[string]interface{}, error) {
	reqBody := make(map[string]interface{})
	if args.Title != nil {
		reqBody["data"] = map[string]interface{}{
			"title": *args.Title,
		}
	}
	if position := buildOptionalPointBody(args.X, args.Y); len(position) > 0 {
		reqBody["position"] = position
	}
	if geometry := buildOptionalSizeBody(args.Width, args.Height); len(geometry) > 0 {
		reqBody["geometry"] = geometry
	}
	if args.Color != nil {
		fillColor, err := normalizeColor(*args.Color)
		if err != nil {
			return nil, fmt.Errorf("color: %w", err)
		}
		reqBody["style"] = map[string]interface{}{
			"fillColor": fillColor,
		}
	}
	return reqBody, nil
}

// buildOptionalPointBody collects the position fields that were supplied.
func buildOptionalPointBody(x, y *float64) map[string]interface{} {
	position := make(map[string]interface{})
	if x != nil {
		position["x"] = *x
	}
	if y != nil {
		position["y"] = *y
	}
	return position
}

// buildOptionalSizeBody collects the geometry fields that were supplied.
func buildOptionalSizeBody(width, height *float64) map[string]interface{} {
	geometry := make(map[string]interface{})
	if width != nil {
		geometry["width"] = *width
	}
	if height != nil {
		geometry["height"] = *height
	}
	return geometry
}

// UpdateFrame updates an existing frame.
func (c *Client) UpdateFrame(ctx context.Context, args UpdateFrameArgs) (UpdateFrameResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateFrameResult{}, err
	}
	if args.FrameID == "" {
		return UpdateFrameResult{}, fmt.Errorf("frame_id is required")
	}

	reqBody, err := buildUpdateFrameBody(args)
	if err != nil {
		return UpdateFrameResult{}, err
	}
	if len(reqBody) == 0 {
		return UpdateFrameResult{}, fmt.Errorf("at least one update field is required")
	}

	path := fmt.Sprintf("/boards/%s/frames/%s", args.BoardID, args.FrameID)
	if _, err := c.request(ctx, http.MethodPatch, path, reqBody); err != nil {
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
	if err := ValidateBoardID(args.BoardID); err != nil {
		return DeleteFrameResult{}, err
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
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetFrameItemsResult{}, err
	}
	if args.FrameID == "" {
		return GetFrameItemsResult{}, fmt.Errorf("frame_id is required")
	}

	limit := clampFrameItemsLimit(args.Limit)

	// Use the items endpoint with parent_item_id filter to get items within a frame.
	// The /frames/{id}/items path does not exist in Miro API v2 and returns 404.
	// See: https://developers.miro.com/reference/get-items-within-frame
	path := fmt.Sprintf("/boards/%s/items?parent_item_id=%s&limit=%d", args.BoardID, args.FrameID, limit)
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

	items := parseFrameItems(resp.Data, strings.EqualFold(args.DetailLevel, "full"))

	return GetFrameItemsResult{
		Items:   items,
		Count:   len(items),
		HasMore: resp.Cursor != "",
		Cursor:  resp.Cursor,
		Message: fmt.Sprintf("Found %d items in frame", len(items)),
	}, nil
}

// clampFrameItemsLimit normalizes a requested page size to the items
// endpoint's bounds.
func clampFrameItemsLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 100 {
		return 100
	}
	return limit
}

// parseFrameItems converts raw item entries into summaries using the shared
// helper, skipping entries that fail to parse.
func parseFrameItems(entries []json.RawMessage, fullDetails bool) []ItemSummary {
	items := make([]ItemSummary, 0, len(entries))
	for _, raw := range entries {
		item := parseItemSummary(raw, fullDetails)
		if item.ID != "" {
			items = append(items, item)
		}
	}
	return items
}
