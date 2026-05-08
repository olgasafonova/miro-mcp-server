package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// =============================================================================
// Image Operations - Create, Get, Update
// =============================================================================

// CreateImage creates an image on a board from a URL.
func (c *Client) CreateImage(ctx context.Context, args CreateImageArgs) (CreateImageResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateImageResult{}, err
	}
	if args.URL == "" {
		return CreateImageResult{}, fmt.Errorf("url is required")
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"url": args.URL,
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
	}

	if args.Title != "" {
		data := reqBody["data"].(map[string]interface{})
		data["title"] = args.Title
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

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/images", reqBody)
	if err != nil {
		return CreateImageResult{}, err
	}

	var image Image
	if err := json.Unmarshal(respBody, &image); err != nil {
		return CreateImageResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	title := image.Data.Title
	if title == "" {
		title = "image"
	}

	// Invalidate items list cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return CreateImageResult{
		ID:      image.ID,
		ItemURL: BuildItemURL(args.BoardID, image.ID),
		Title:   title,
		URL:     image.Data.ImageURL,
		Message: fmt.Sprintf("Added image '%s'", truncate(title, 30)),
	}, nil
}

// GetImage retrieves details of a specific image item.
func (c *Client) GetImage(ctx context.Context, args GetImageArgs) (GetImageResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetImageResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return GetImageResult{}, err
	}

	cacheKey := CacheKeyItem(args.BoardID, args.ItemID)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if result, ok := cached.(GetImageResult); ok {
			return result, nil
		}
	}

	respBody, err := c.request(ctx, http.MethodGet, "/boards/"+args.BoardID+"/images/"+args.ItemID, nil)
	if err != nil {
		return GetImageResult{}, err
	}

	var resp struct {
		ID       string `json:"id"`
		Position *struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"position"`
		Geometry *struct {
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"geometry"`
		Data struct {
			Title    string `json:"title"`
			ImageURL string `json:"imageUrl"`
		} `json:"data"`
		ParentID string `json:"parentId"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return GetImageResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	result := GetImageResult{
		ID:       resp.ID,
		Title:    resp.Data.Title,
		ImageURL: resp.Data.ImageURL,
		ParentID: resp.ParentID,
		Message:  "Image retrieved successfully",
	}
	if resp.Position != nil {
		result.X = resp.Position.X
		result.Y = resp.Position.Y
	}
	if resp.Geometry != nil {
		result.Width = resp.Geometry.Width
		result.Height = resp.Geometry.Height
	}

	c.cache.Set(cacheKey, result, c.cacheConfig.ItemTTL)

	return result, nil
}

// UpdateImage updates an image using the dedicated images endpoint.
func (c *Client) UpdateImage(ctx context.Context, args UpdateImageArgs) (UpdateImageResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateImageResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateImageResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := make(map[string]interface{})

	// Build data section
	data := make(map[string]interface{})
	if args.Title != nil {
		data["title"] = *args.Title
	}
	if args.URL != nil {
		data["url"] = *args.URL
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
		return UpdateImageResult{
			ID:      args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	path := "/boards/" + args.BoardID + "/images/" + args.ItemID
	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateImageResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Title    string `json:"title"`
			ImageURL string `json:"imageUrl"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateImageResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateImageResult{
		ID:      resp.ID,
		Title:   resp.Data.Title,
		URL:     resp.Data.ImageURL,
		Message: "Image updated successfully",
	}, nil
}
