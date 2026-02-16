package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CreateDocFormat creates a doc format item from Markdown content on a board.
func (c *Client) CreateDocFormat(ctx context.Context, args CreateDocFormatArgs) (CreateDocFormatResult, error) {
	if args.BoardID == "" {
		return CreateDocFormatResult{}, fmt.Errorf("board_id is required")
	}
	if args.Content == "" {
		return CreateDocFormatResult{}, fmt.Errorf("content is required")
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"contentType": "markdown",
			"content":     args.Content,
		},
	}

	if args.X != 0 || args.Y != 0 {
		reqBody["position"] = map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		}
	}

	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/docs", reqBody)
	if err != nil {
		return CreateDocFormatResult{}, err
	}

	var resp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return CreateDocFormatResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate items list cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return CreateDocFormatResult{
		ID:      resp.ID,
		ItemURL: BuildItemURL(args.BoardID, resp.ID),
		Message: "Created doc format item",
	}, nil
}

// GetDocFormat gets the details of a doc format item.
func (c *Client) GetDocFormat(ctx context.Context, args GetDocFormatArgs) (GetDocFormatResult, error) {
	if args.BoardID == "" {
		return GetDocFormatResult{}, fmt.Errorf("board_id is required")
	}
	if args.ItemID == "" {
		return GetDocFormatResult{}, fmt.Errorf("item_id is required")
	}

	path := fmt.Sprintf("/boards/%s/docs/%s", args.BoardID, args.ItemID)
	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetDocFormatResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
		Position struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"position"`
		CreatedAt  string `json:"createdAt"`
		ModifiedAt string `json:"modifiedAt"`
		CreatedBy  struct {
			ID string `json:"id"`
		} `json:"createdBy"`
		ModifiedBy struct {
			ID string `json:"id"`
		} `json:"modifiedBy"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return GetDocFormatResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return GetDocFormatResult{
		ID:         resp.ID,
		Content:    resp.Data.Content,
		X:          resp.Position.X,
		Y:          resp.Position.Y,
		CreatedAt:  resp.CreatedAt,
		ModifiedAt: resp.ModifiedAt,
		CreatedBy:  resp.CreatedBy.ID,
		ModifiedBy: resp.ModifiedBy.ID,
		Message:    "Retrieved doc format item",
	}, nil
}

// DeleteDocFormat deletes a doc format item from a board.
func (c *Client) DeleteDocFormat(ctx context.Context, args DeleteDocFormatArgs) (DeleteDocFormatResult, error) {
	if args.BoardID == "" {
		return DeleteDocFormatResult{}, fmt.Errorf("board_id is required")
	}
	if args.ItemID == "" {
		return DeleteDocFormatResult{}, fmt.Errorf("item_id is required")
	}

	if args.DryRun {
		return DeleteDocFormatResult{
			Success: true,
			ItemID:  args.ItemID,
			Message: "[DRY RUN] Would delete doc format item " + args.ItemID,
		}, nil
	}

	path := fmt.Sprintf("/boards/%s/docs/%s", args.BoardID, args.ItemID)
	_, err := c.request(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return DeleteDocFormatResult{}, err
	}

	// Invalidate items list cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return DeleteDocFormatResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "Doc format item deleted successfully",
	}, nil
}
