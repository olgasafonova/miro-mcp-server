package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// =============================================================================
// Embed Operations - Create, Update
// =============================================================================

// CreateEmbed creates an embedded content item on a board.
func (c *Client) CreateEmbed(ctx context.Context, args CreateEmbedArgs) (CreateEmbedResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateEmbedResult{}, err
	}
	if args.URL == "" {
		return CreateEmbedResult{}, fmt.Errorf("url is required")
	}

	mode := args.Mode
	if mode == "" {
		mode = "inline"
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"url":  args.URL,
			"mode": mode,
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
	}

	// For embeds with fixed aspect ratio (like YouTube), only send width
	// Miro will calculate height automatically. Sending both causes an error.
	if args.Width > 0 {
		reqBody["geometry"] = map[string]interface{}{
			"width": args.Width,
		}
	} else if args.Height > 0 {
		reqBody["geometry"] = map[string]interface{}{
			"height": args.Height,
		}
	}
	// If neither specified, let Miro use defaults

	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/embeds", reqBody)
	if err != nil {
		return CreateEmbedResult{}, err
	}

	var embed Embed
	if err := json.Unmarshal(respBody, &embed); err != nil {
		return CreateEmbedResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate items list cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return CreateEmbedResult{
		ID:       embed.ID,
		ItemURL:  BuildItemURL(args.BoardID, embed.ID),
		URL:      embed.Data.URL,
		Provider: embed.Data.ProviderName,
		Message:  fmt.Sprintf("Embedded content from %s", embed.Data.ProviderName),
	}, nil
}

// UpdateEmbed updates an embed using the dedicated embeds endpoint.
func (c *Client) UpdateEmbed(ctx context.Context, args UpdateEmbedArgs) (UpdateEmbedResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateEmbedResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateEmbedResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := make(map[string]interface{})

	// Build data section
	data := make(map[string]interface{})
	if args.URL != nil {
		data["url"] = *args.URL
	}
	if args.Mode != nil {
		data["mode"] = *args.Mode
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
	geom := make(map[string]interface{})
	if args.Width != nil {
		geom["width"] = *args.Width
	}
	if args.Height != nil {
		geom["height"] = *args.Height
	}
	if len(geom) > 0 {
		reqBody["geometry"] = geom
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
		return UpdateEmbedResult{
			ID:      args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	path := "/boards/" + args.BoardID + "/embeds/" + args.ItemID
	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateEmbedResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			URL         string `json:"url"`
			ProviderURL string `json:"providerUrl"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateEmbedResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateEmbedResult{
		ID:       resp.ID,
		URL:      resp.Data.URL,
		Provider: resp.Data.ProviderURL,
		Message:  "Embed updated successfully",
	}, nil
}
