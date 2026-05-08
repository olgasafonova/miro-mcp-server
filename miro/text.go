package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// =============================================================================
// Text Operations - Create, Update
// =============================================================================

// CreateText creates a text item on a board.
func (c *Client) CreateText(ctx context.Context, args CreateTextArgs) (CreateTextResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateTextResult{}, err
	}
	if args.Content == "" {
		return CreateTextResult{}, fmt.Errorf("content is required")
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"content": args.Content,
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
	}

	style := make(map[string]interface{})
	if args.FontSize > 0 {
		style["fontSize"] = strconv.Itoa(args.FontSize)
	}
	if args.Color != "" {
		color, err := normalizeColor(args.Color)
		if err != nil {
			return CreateTextResult{}, fmt.Errorf("color: %w", err)
		}
		style["color"] = color
	}
	if len(style) > 0 {
		reqBody["style"] = style
	}

	if args.Width > 0 {
		reqBody["geometry"] = map[string]interface{}{
			"width": args.Width,
		}
	}

	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/texts", reqBody)
	if err != nil {
		return CreateTextResult{}, err
	}

	var text TextItem
	if err := json.Unmarshal(respBody, &text); err != nil {
		return CreateTextResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate items list cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return CreateTextResult{
		ID:      text.ID,
		ItemURL: BuildItemURL(args.BoardID, text.ID),
		Content: text.Data.Content,
		Message: fmt.Sprintf("Created text '%s'", truncate(args.Content, 30)),
	}, nil
}

// UpdateText updates a text item using the dedicated text endpoint.
func (c *Client) UpdateText(ctx context.Context, args UpdateTextArgs) (UpdateTextResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateTextResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateTextResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := make(map[string]interface{})

	// Build data section
	if args.Content != nil {
		reqBody["data"] = map[string]interface{}{
			"content": *args.Content,
		}
	}

	// Build style section
	style := make(map[string]interface{})
	if args.FontSize != nil {
		style["fontSize"] = fmt.Sprintf("%d", *args.FontSize)
	}
	if args.TextAlign != nil {
		style["textAlign"] = *args.TextAlign
	}
	if args.Color != nil {
		color, err := normalizeColor(*args.Color)
		if err != nil {
			return UpdateTextResult{}, fmt.Errorf("color: %w", err)
		}
		style["color"] = color
	}
	if len(style) > 0 {
		reqBody["style"] = style
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
		return UpdateTextResult{
			ID:      args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	path := "/boards/" + args.BoardID + "/texts/" + args.ItemID
	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateTextResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
		Style struct {
			FontSize string `json:"fontSize"`
		} `json:"style"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateTextResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	var fontSize int
	fmt.Sscanf(resp.Style.FontSize, "%d", &fontSize)

	return UpdateTextResult{
		ID:       resp.ID,
		Content:  resp.Data.Content,
		FontSize: fontSize,
		Message:  "Text updated successfully",
	}, nil
}
