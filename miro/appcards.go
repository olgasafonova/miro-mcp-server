package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// =============================================================================
// App Card Operations
// =============================================================================
// App cards are custom cards with fields and external app integration.

// CreateAppCard creates an app card on a board.
func (c *Client) CreateAppCard(ctx context.Context, args CreateAppCardArgs) (CreateAppCardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateAppCardResult{}, err
	}
	if args.Title == "" {
		return CreateAppCardResult{}, fmt.Errorf("title is required")
	}

	// Build request body
	data := map[string]interface{}{
		"title": args.Title,
	}
	if args.Description != "" {
		data["description"] = args.Description
	}
	if args.Status != "" {
		data["status"] = args.Status
	}
	if len(args.Fields) > 0 {
		fields := make([]map[string]interface{}, len(args.Fields))
		for i, f := range args.Fields {
			field := map[string]interface{}{}
			if f.Value != "" {
				field["value"] = f.Value
			}
			if f.FillColor != "" {
				field["fillColor"] = f.FillColor
			}
			if f.TextColor != "" {
				field["textColor"] = f.TextColor
			}
			if f.IconShape != "" {
				field["iconShape"] = f.IconShape
			}
			if f.IconURL != "" {
				field["iconUrl"] = f.IconURL
			}
			fields[i] = field
		}
		data["fields"] = fields
	}

	reqBody := map[string]interface{}{
		"data": data,
	}

	// Add position if specified
	if args.X != 0 || args.Y != 0 {
		reqBody["position"] = map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
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

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/app_cards", reqBody)
	if err != nil {
		return CreateAppCardResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Status      string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return CreateAppCardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CreateAppCardResult{
		ID:          resp.ID,
		Title:       resp.Data.Title,
		Description: resp.Data.Description,
		Status:      resp.Data.Status,
		Message:     fmt.Sprintf("Created app card '%s'", truncate(args.Title, 30)),
	}, nil
}

// GetAppCard retrieves an app card by ID.
func (c *Client) GetAppCard(ctx context.Context, args GetAppCardArgs) (GetAppCardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetAppCardResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return GetAppCardResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	respBody, err := c.request(ctx, http.MethodGet, "/boards/"+args.BoardID+"/app_cards/"+args.ItemID, nil)
	if err != nil {
		return GetAppCardResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Status      string `json:"status"`
			Fields      []struct {
				Value     string `json:"value"`
				FillColor string `json:"fillColor"`
				TextColor string `json:"textColor"`
				IconShape string `json:"iconShape"`
				IconURL   string `json:"iconUrl"`
			} `json:"fields"`
		} `json:"data"`
		Position struct {
			X      float64 `json:"x"`
			Y      float64 `json:"y"`
			Origin string  `json:"origin"`
		} `json:"position"`
		Geometry struct {
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"geometry"`
		CreatedAt  string `json:"createdAt"`
		ModifiedAt string `json:"modifiedAt"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return GetAppCardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert fields
	fields := make([]AppCardField, len(resp.Data.Fields))
	for i, f := range resp.Data.Fields {
		fields[i] = AppCardField{
			Value:     f.Value,
			FillColor: f.FillColor,
			TextColor: f.TextColor,
			IconShape: f.IconShape,
			IconURL:   f.IconURL,
		}
	}

	return GetAppCardResult{
		ID:          resp.ID,
		Title:       resp.Data.Title,
		Description: resp.Data.Description,
		Status:      resp.Data.Status,
		Fields:      fields,
		Position: &Position{
			X:      resp.Position.X,
			Y:      resp.Position.Y,
			Origin: resp.Position.Origin,
		},
		Geometry: &Geometry{
			Width:  resp.Geometry.Width,
			Height: resp.Geometry.Height,
		},
		CreatedAt:  resp.CreatedAt,
		ModifiedAt: resp.ModifiedAt,
		Message:    fmt.Sprintf("App card '%s'", truncate(resp.Data.Title, 30)),
	}, nil
}

// UpdateAppCard updates an app card.
func (c *Client) UpdateAppCard(ctx context.Context, args UpdateAppCardArgs) (UpdateAppCardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateAppCardResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateAppCardResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	// Build request body with only provided fields
	data := map[string]interface{}{}
	if args.Title != "" {
		data["title"] = args.Title
	}
	if args.Description != "" {
		data["description"] = args.Description
	}
	if args.Status != "" {
		data["status"] = args.Status
	}
	if len(args.Fields) > 0 {
		fields := make([]map[string]interface{}, len(args.Fields))
		for i, f := range args.Fields {
			field := map[string]interface{}{}
			if f.Value != "" {
				field["value"] = f.Value
			}
			if f.FillColor != "" {
				field["fillColor"] = f.FillColor
			}
			if f.TextColor != "" {
				field["textColor"] = f.TextColor
			}
			if f.IconShape != "" {
				field["iconShape"] = f.IconShape
			}
			if f.IconURL != "" {
				field["iconUrl"] = f.IconURL
			}
			fields[i] = field
		}
		data["fields"] = fields
	}

	reqBody := map[string]interface{}{}
	if len(data) > 0 {
		reqBody["data"] = data
	}

	// Add position if specified
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

	// Add geometry if width specified
	if args.Width != nil {
		reqBody["geometry"] = map[string]interface{}{
			"width": *args.Width,
		}
	}

	if len(reqBody) == 0 {
		return UpdateAppCardResult{}, fmt.Errorf("at least one field must be provided for update")
	}

	respBody, err := c.request(ctx, http.MethodPatch, "/boards/"+args.BoardID+"/app_cards/"+args.ItemID, reqBody)
	if err != nil {
		return UpdateAppCardResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateAppCardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return UpdateAppCardResult{
		ID:      resp.ID,
		Title:   resp.Data.Title,
		Status:  resp.Data.Status,
		Message: "App card updated successfully",
	}, nil
}

// DeleteAppCard deletes an app card from a board.
func (c *Client) DeleteAppCard(ctx context.Context, args DeleteAppCardArgs) (DeleteAppCardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return DeleteAppCardResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return DeleteAppCardResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	_, err := c.request(ctx, http.MethodDelete, "/boards/"+args.BoardID+"/app_cards/"+args.ItemID, nil)
	if err != nil {
		return DeleteAppCardResult{
			Success: false,
			ItemID:  args.ItemID,
			Message: fmt.Sprintf("Failed to delete app card: %v", err),
		}, err
	}

	return DeleteAppCardResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "App card deleted successfully",
	}, nil
}
