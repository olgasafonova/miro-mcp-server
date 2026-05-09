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

// setIfNonEmpty assigns value into m at key only when value is non-empty.
func setIfNonEmpty(m map[string]interface{}, key, value string) {
	if value != "" {
		m[key] = value
	}
}

// normalizeAppCardColor wraps normalizeColor with the per-field index-tagged
// error format. Returns ("", nil) when the input is empty so callers can pipe
// the result through setIfNonEmpty.
func normalizeAppCardColor(idx int, fieldKey, value string) (string, error) {
	if value == "" {
		return "", nil
	}
	normalized, err := normalizeColor(value)
	if err != nil {
		return "", fmt.Errorf("fields[%d].%s: %w", idx, fieldKey, err)
	}
	return normalized, nil
}

// buildAppCardField translates a single AppCardField into the API map representation,
// normalizing colors and dropping unset fields. Returns the field's index-tagged
// error if any color fails to normalize.
func buildAppCardField(i int, f AppCardField) (map[string]interface{}, error) {
	fillColor, err := normalizeAppCardColor(i, "fill_color", f.FillColor)
	if err != nil {
		return nil, err
	}
	textColor, err := normalizeAppCardColor(i, "text_color", f.TextColor)
	if err != nil {
		return nil, err
	}

	out := map[string]interface{}{}
	setIfNonEmpty(out, "value", f.Value)
	setIfNonEmpty(out, "fillColor", fillColor)
	setIfNonEmpty(out, "textColor", textColor)
	setIfNonEmpty(out, "iconShape", f.IconShape)
	setIfNonEmpty(out, "iconUrl", f.IconURL)
	return out, nil
}

// buildAppCardFields converts the input field slice to the API map slice,
// preserving order and surfacing the first per-field error.
func buildAppCardFields(in []AppCardField) ([]map[string]interface{}, error) {
	out := make([]map[string]interface{}, len(in))
	for i, f := range in {
		field, err := buildAppCardField(i, f)
		if err != nil {
			return nil, err
		}
		out[i] = field
	}
	return out, nil
}

// buildCreatePosition returns the position payload for an app-card create call,
// or nil when both coordinates are unset (zero).
func buildCreatePosition(x, y float64) map[string]interface{} {
	if x == 0 && y == 0 {
		return nil
	}
	return map[string]interface{}{"x": x, "y": y, "origin": "center"}
}

// buildUpdatePosition returns the position payload for an app-card update call,
// or nil when both coordinate pointers are nil. Either pointer can be set
// independently.
func buildUpdatePosition(x, y *float64) map[string]interface{} {
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

// CreateAppCard creates an app card on a board.
func (c *Client) CreateAppCard(ctx context.Context, args CreateAppCardArgs) (CreateAppCardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateAppCardResult{}, err
	}
	if args.Title == "" {
		return CreateAppCardResult{}, fmt.Errorf("title is required")
	}

	reqBody, err := buildCreateAppCardBody(args)
	if err != nil {
		return CreateAppCardResult{}, err
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
		ItemURL:     BuildItemURL(args.BoardID, resp.ID),
		Title:       resp.Data.Title,
		Description: resp.Data.Description,
		Status:      resp.Data.Status,
		Message:     fmt.Sprintf("Created app card '%s'", truncate(args.Title, 30)),
	}, nil
}

// applyAppCardCommonData adds the description, status, and fields keys (when set)
// to data. Shared between Create and Update which differ only on how title is treated.
func applyAppCardCommonData(data map[string]interface{}, description, status string, fields []AppCardField) error {
	setIfNonEmpty(data, "description", description)
	setIfNonEmpty(data, "status", status)
	if len(fields) == 0 {
		return nil
	}
	out, err := buildAppCardFields(fields)
	if err != nil {
		return err
	}
	data["fields"] = out
	return nil
}

// buildCreateAppCardData assembles the "data" object for a CreateAppCard call.
// Title is mandatory (validated by the caller); other fields are omitted when empty.
func buildCreateAppCardData(args CreateAppCardArgs) (map[string]interface{}, error) {
	data := map[string]interface{}{"title": args.Title}
	if err := applyAppCardCommonData(data, args.Description, args.Status, args.Fields); err != nil {
		return nil, err
	}
	return data, nil
}

// buildCreateAppCardBody assembles the full POST body for CreateAppCard.
func buildCreateAppCardBody(args CreateAppCardArgs) (map[string]interface{}, error) {
	data, err := buildCreateAppCardData(args)
	if err != nil {
		return nil, err
	}
	reqBody := map[string]interface{}{"data": data}
	if pos := buildCreatePosition(args.X, args.Y); pos != nil {
		reqBody["position"] = pos
	}
	if args.Width > 0 {
		reqBody["geometry"] = map[string]interface{}{"width": args.Width}
	}
	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{"id": args.ParentID}
	}
	return reqBody, nil
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

	reqBody, err := buildUpdateAppCardBody(args)
	if err != nil {
		return UpdateAppCardResult{}, err
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

// buildUpdateAppCardData assembles the "data" object for an UpdateAppCard call.
// All fields are optional; non-empty ones are included.
func buildUpdateAppCardData(args UpdateAppCardArgs) (map[string]interface{}, error) {
	data := map[string]interface{}{}
	setIfNonEmpty(data, "title", args.Title)
	if err := applyAppCardCommonData(data, args.Description, args.Status, args.Fields); err != nil {
		return nil, err
	}
	return data, nil
}

// buildUpdateAppCardBody assembles the full PATCH body, merging optional data,
// position, and geometry sections. Returns an empty map when nothing to update.
func buildUpdateAppCardBody(args UpdateAppCardArgs) (map[string]interface{}, error) {
	data, err := buildUpdateAppCardData(args)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]interface{}{}
	if len(data) > 0 {
		reqBody["data"] = data
	}
	if pos := buildUpdatePosition(args.X, args.Y); pos != nil {
		reqBody["position"] = pos
	}
	if args.Width != nil {
		reqBody["geometry"] = map[string]interface{}{"width": *args.Width}
	}
	return reqBody, nil
}

// DeleteAppCard deletes an app card from a board.
func (c *Client) DeleteAppCard(ctx context.Context, args DeleteAppCardArgs) (DeleteAppCardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return DeleteAppCardResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return DeleteAppCardResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	// Dry-run mode: return preview without deleting
	if args.DryRun {
		return DeleteAppCardResult{
			Success: true,
			ItemID:  args.ItemID,
			Message: "[DRY RUN] Would delete app card " + args.ItemID + " from board " + args.BoardID,
		}, nil
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
