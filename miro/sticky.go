package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// =============================================================================
// Sticky Note Operations - Create, Update
// =============================================================================

// CreateSticky creates a sticky note on a board.
func (c *Client) CreateSticky(ctx context.Context, args CreateStickyArgs) (CreateStickyResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateStickyResult{}, err
	}
	if args.Content == "" {
		return CreateStickyResult{}, fmt.Errorf("content is required")
	}

	// Build request body
	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"content": args.Content,
			"shape":   "square",
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
	}

	// Add style if color specified
	if args.Color != "" {
		reqBody["style"] = map[string]interface{}{
			"fillColor": normalizeStickyColor(args.Color),
		}
	}

	// Add geometry if width specified
	if args.Width > 0 {
		reqBody["geometry"] = map[string]interface{}{
			"width": args.Width,
		}
	}

	// Add parent if specified
	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/sticky_notes", reqBody)
	if err != nil {
		return CreateStickyResult{}, err
	}

	var sticky StickyNote
	if err := json.Unmarshal(respBody, &sticky); err != nil {
		return CreateStickyResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate items list cache since we added a new item
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return CreateStickyResult{
		ID:      sticky.ID,
		ItemURL: BuildItemURL(args.BoardID, sticky.ID),
		Content: sticky.Data.Content,
		Color:   sticky.Style.FillColor,
		Message: fmt.Sprintf("Created sticky note '%s'", truncate(args.Content, 30)),
	}, nil
}

// UpdateSticky updates a sticky note using the dedicated sticky_notes endpoint.
func (c *Client) UpdateSticky(ctx context.Context, args UpdateStickyArgs) (UpdateStickyResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateStickyResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateStickyResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := make(map[string]interface{})

	// Build data section
	data := make(map[string]interface{})
	if args.Content != nil {
		data["content"] = *args.Content
	}
	if args.Shape != nil {
		data["shape"] = *args.Shape
	}
	if len(data) > 0 {
		reqBody["data"] = data
	}

	// Build style section
	if args.Color != nil {
		reqBody["style"] = map[string]interface{}{
			"fillColor": *args.Color,
		}
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
		return UpdateStickyResult{
			ID:      args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	path := "/boards/" + args.BoardID + "/sticky_notes/" + args.ItemID
	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateStickyResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Content string `json:"content"`
			Shape   string `json:"shape"`
		} `json:"data"`
		Style struct {
			FillColor string `json:"fillColor"`
		} `json:"style"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateStickyResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateStickyResult{
		ID:      resp.ID,
		Content: resp.Data.Content,
		Shape:   resp.Data.Shape,
		Color:   resp.Style.FillColor,
		Message: "Sticky note updated successfully",
	}, nil
}

// normalizeStickyColor converts color names to Miro's expected format.
func normalizeStickyColor(color string) string {
	// Miro uses specific color names
	colorMap := map[string]string{
		"yellow":     "light_yellow",
		"green":      "light_green",
		"blue":       "light_blue",
		"pink":       "light_pink",
		"purple":     "violet",
		"orange":     "orange",
		"red":        "red",
		"gray":       "gray",
		"grey":       "gray",
		"cyan":       "cyan",
		"dark_green": "dark_green",
		"dark_blue":  "dark_blue",
		"black":      "black",
	}

	lower := strings.ToLower(color)
	if mapped, ok := colorMap[lower]; ok {
		return mapped
	}
	return color // Return as-is if not in map
}
