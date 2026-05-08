package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// =============================================================================
// Card Operations - Create, Update
// =============================================================================

// CreateCard creates a card on a board.
func (c *Client) CreateCard(ctx context.Context, args CreateCardArgs) (CreateCardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateCardResult{}, err
	}
	if args.Title == "" {
		return CreateCardResult{}, fmt.Errorf("title is required")
	}

	// Default width
	width := args.Width
	if width == 0 {
		width = 320
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"title":       args.Title,
			"description": args.Description,
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
		"geometry": map[string]interface{}{
			"width": width,
		},
	}

	if args.DueDate != "" {
		data := reqBody["data"].(map[string]interface{})
		data["dueDate"] = args.DueDate
	}

	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/cards", reqBody)
	if err != nil {
		return CreateCardResult{}, err
	}

	var card Card
	if err := json.Unmarshal(respBody, &card); err != nil {
		return CreateCardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate items list cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return CreateCardResult{
		ID:      card.ID,
		ItemURL: BuildItemURL(args.BoardID, card.ID),
		Title:   card.Data.Title,
		Message: fmt.Sprintf("Created card '%s'", truncate(args.Title, 30)),
	}, nil
}

// UpdateCard updates a card using the dedicated cards endpoint.
func (c *Client) UpdateCard(ctx context.Context, args UpdateCardArgs) (UpdateCardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateCardResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateCardResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := make(map[string]interface{})

	// Build data section
	data := make(map[string]interface{})
	if args.Title != nil {
		data["title"] = *args.Title
	}
	if args.Description != nil {
		data["description"] = *args.Description
	}
	if args.DueDate != nil {
		if *args.DueDate == "" {
			data["dueDate"] = nil
		} else {
			data["dueDate"] = *args.DueDate
		}
	}
	if len(data) > 0 {
		reqBody["data"] = data
	}

	// Build position section
	if args.X != nil || args.Y != nil {
		pos := map[string]interface{}{"origin": "center"}
		if args.X != nil {
			pos["x"] = *args.X
		}
		if args.Y != nil {
			pos["y"] = *args.Y
		}
		reqBody["position"] = pos
	}

	// Build geometry section
	if args.Width != nil {
		reqBody["geometry"] = map[string]interface{}{
			"width": *args.Width,
		}
	}

	// Build parent section
	if args.ParentID != nil {
		if *args.ParentID == "" {
			reqBody["parent"] = nil
		} else {
			reqBody["parent"] = map[string]interface{}{"id": *args.ParentID}
		}
	}

	if len(reqBody) == 0 {
		return UpdateCardResult{
			ID:      args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	path := "/boards/" + args.BoardID + "/cards/" + args.ItemID
	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateCardResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			DueDate     string `json:"dueDate"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateCardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateCardResult{
		ID:          resp.ID,
		Title:       resp.Data.Title,
		Description: resp.Data.Description,
		DueDate:     resp.Data.DueDate,
		Message:     "Card updated successfully",
	}, nil
}
