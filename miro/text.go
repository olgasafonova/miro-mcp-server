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

// buildTextStyleSection returns the optional "style" map for a text update.
func buildTextStyleSection(args UpdateTextArgs) (map[string]interface{}, error) {
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
			return nil, fmt.Errorf("color: %w", err)
		}
		style["color"] = color
	}
	return style, nil
}

// buildPositionSection returns the optional "position" map when X or Y is set.
func buildPositionSection(x, y *float64) map[string]interface{} {
	if x == nil && y == nil {
		return nil
	}
	pos := map[string]interface{}{"origin": "center"}
	if x != nil {
		pos["x"] = *x
	}
	if y != nil {
		pos["y"] = *y
	}
	return pos
}

// applyParentField sets the "parent" key on reqBody if parentID is non-nil.
// An empty string clears the parent (assigns nil).
func applyParentField(reqBody map[string]interface{}, parentID *string) {
	if parentID == nil {
		return
	}
	if *parentID == "" {
		reqBody["parent"] = nil
		return
	}
	reqBody["parent"] = map[string]interface{}{"id": *parentID}
}

// buildUpdateTextBody assembles the full PATCH body for a text update.
func buildUpdateTextBody(args UpdateTextArgs) (map[string]interface{}, error) {
	reqBody := make(map[string]interface{})

	if args.Content != nil {
		reqBody["data"] = map[string]interface{}{"content": *args.Content}
	}

	style, err := buildTextStyleSection(args)
	if err != nil {
		return nil, err
	}
	if len(style) > 0 {
		reqBody["style"] = style
	}

	if pos := buildPositionSection(args.X, args.Y); pos != nil {
		reqBody["position"] = pos
	}

	if args.Width != nil {
		reqBody["geometry"] = map[string]interface{}{"width": *args.Width}
	}

	applyParentField(reqBody, args.ParentID)
	return reqBody, nil
}

// UpdateText updates a text item using the dedicated text endpoint.
func (c *Client) UpdateText(ctx context.Context, args UpdateTextArgs) (UpdateTextResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateTextResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateTextResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody, err := buildUpdateTextBody(args)
	if err != nil {
		return UpdateTextResult{}, err
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
