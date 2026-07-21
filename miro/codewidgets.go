package miro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// =============================================================================
// Code Widget Operations (v2-experimental)
// =============================================================================

// maxCodeWidgetCodeLength is the API's cap on the code field (spec:
// CodeWidgetData.code maxLength).
const maxCodeWidgetCodeLength = 6000

// maxCodeWidgetTitleLength is the API's cap on the title field (spec:
// CodeWidgetData.title maxLength).
const maxCodeWidgetTitleLength = 100

// codeWidgetPreviewLength bounds the code excerpt in list summaries; full
// source is available via GetCodeWidget.
const codeWidgetPreviewLength = 80

// wrapExperimentalCodeWidgetErr adds an experimental-availability hint to
// 403/404 responses from the code_widgets endpoints. A plain 404 there is
// ambiguous: the item may not exist, or the account may simply not have
// access to the v2-experimental API. Only wraps *APIError; validation and
// transport errors pass through untouched.
func wrapExperimentalCodeWidgetErr(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && (apiErr.IsNotFound() || apiErr.IsForbidden()) {
		return fmt.Errorf("%w (note: code_widgets is a v2-experimental Miro API and may be unavailable for your account or plan)", err)
	}
	return err
}

// validateCodeWidgetData checks the API-documented field caps shared by
// create and update.
func validateCodeWidgetData(code, title string) error {
	if len(code) > maxCodeWidgetCodeLength {
		return fmt.Errorf("code exceeds %d characters (got %d); split the snippet across multiple widgets", maxCodeWidgetCodeLength, len(code))
	}
	if len(title) > maxCodeWidgetTitleLength {
		return fmt.Errorf("title exceeds %d characters (got %d)", maxCodeWidgetTitleLength, len(title))
	}
	return nil
}

// truncateCodeWidget shortens a string to max length with ellipsis.
// Local copy per file to avoid import cycles, matching mindmaps.go.
func truncateCodeWidget(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// buildCodeWidgetData assembles the data block from optional fields; returns
// nil when no data field is set.
func buildCodeWidgetData(code, language, title string, lineNumbersVisible *bool) map[string]interface{} {
	data := map[string]interface{}{}
	if code != "" {
		data["code"] = code
	}
	if language != "" {
		data["language"] = language
	}
	if title != "" {
		data["title"] = title
	}
	if lineNumbersVisible != nil {
		data["lineNumbersVisible"] = *lineNumbersVisible
	}
	if len(data) == 0 {
		return nil
	}
	return data
}

// CreateCodeWidget creates a code widget on a board.
// Uses the v2-experimental API.
func (c *Client) CreateCodeWidget(ctx context.Context, args CreateCodeWidgetArgs) (CreateCodeWidgetResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateCodeWidgetResult{}, err
	}
	if args.Code == "" {
		return CreateCodeWidgetResult{}, fmt.Errorf("code is required")
	}
	if err := validateCodeWidgetData(args.Code, args.Title); err != nil {
		return CreateCodeWidgetResult{}, err
	}
	if args.ParentID != "" {
		if err := ValidateItemID(args.ParentID); err != nil {
			return CreateCodeWidgetResult{}, fmt.Errorf("invalid parent_id: %w", err)
		}
	}

	reqBody := map[string]interface{}{
		"data": buildCodeWidgetData(args.Code, args.Language, args.Title, args.LineNumbersVisible),
	}
	if args.X != 0 || args.Y != 0 {
		reqBody["position"] = map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		}
	}
	if geo := buildCodeWidgetGeometry(args.Width, args.Height); geo != nil {
		reqBody["geometry"] = geo
	}
	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{"id": args.ParentID}
	}

	respBody, err := c.requestExperimental(ctx, http.MethodPost, "/boards/"+args.BoardID+"/code_widgets", reqBody)
	if err != nil {
		return CreateCodeWidgetResult{}, wrapExperimentalCodeWidgetErr(err)
	}

	var widget struct {
		ID   string `json:"id"`
		Data struct {
			Language string `json:"language"`
			Title    string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &widget); err != nil {
		return CreateCodeWidgetResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidatePrefix("items:" + args.BoardID)

	label := widget.Data.Title
	if label == "" {
		label = truncateCodeWidget(args.Code, 30)
	}
	return CreateCodeWidgetResult{
		ID:       widget.ID,
		ItemURL:  BuildItemURL(args.BoardID, widget.ID),
		Title:    widget.Data.Title,
		Language: widget.Data.Language,
		Message:  fmt.Sprintf("Created code widget '%s'", label),
	}, nil
}

// buildCodeWidgetGeometry assembles the geometry block; nil when neither
// dimension is set.
func buildCodeWidgetGeometry(width, height *float64) map[string]interface{} {
	if width == nil && height == nil {
		return nil
	}
	geo := map[string]interface{}{}
	if width != nil {
		geo["width"] = *width
	}
	if height != nil {
		geo["height"] = *height
	}
	return geo
}

// GetCodeWidget retrieves a specific code widget.
// Uses the v2-experimental API.
func (c *Client) GetCodeWidget(ctx context.Context, args GetCodeWidgetArgs) (GetCodeWidgetResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetCodeWidgetResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return GetCodeWidgetResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	path := fmt.Sprintf("/boards/%s/code_widgets/%s", args.BoardID, args.ItemID)
	respBody, err := c.requestExperimental(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetCodeWidgetResult{}, wrapExperimentalCodeWidgetErr(err)
	}

	var widget struct {
		ID   string `json:"id"`
		Data struct {
			Code               string `json:"code"`
			Language           string `json:"language"`
			Title              string `json:"title"`
			LineNumbersVisible bool   `json:"lineNumbersVisible"`
		} `json:"data"`
		Position *struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"position"`
		Geometry *struct {
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"geometry"`
		Parent *struct {
			ID string `json:"id"`
		} `json:"parent"`
		CreatedAt  string `json:"createdAt"`
		ModifiedAt string `json:"modifiedAt"`
	}
	if err := json.Unmarshal(respBody, &widget); err != nil {
		return GetCodeWidgetResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	result := GetCodeWidgetResult{
		ID:                 widget.ID,
		Code:               widget.Data.Code,
		Language:           widget.Data.Language,
		Title:              widget.Data.Title,
		LineNumbersVisible: widget.Data.LineNumbersVisible,
		CreatedAt:          widget.CreatedAt,
		ModifiedAt:         widget.ModifiedAt,
		Message:            fmt.Sprintf("Retrieved code widget %s", widget.ID),
	}
	if widget.Position != nil {
		result.X = widget.Position.X
		result.Y = widget.Position.Y
	}
	if widget.Geometry != nil {
		result.Width = widget.Geometry.Width
		result.Height = widget.Geometry.Height
	}
	if widget.Parent != nil {
		result.ParentID = widget.Parent.ID
	}
	return result, nil
}

// ListCodeWidgets retrieves code widgets on a board with cursor pagination.
// Uses the v2-experimental API.
func (c *Client) ListCodeWidgets(ctx context.Context, args ListCodeWidgetsArgs) (ListCodeWidgetsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListCodeWidgetsResult{}, err
	}

	limit := args.Limit
	if limit <= 0 {
		limit = DefaultItemLimit
	}
	if limit > MaxItemLimitExtended {
		limit = MaxItemLimitExtended
	}

	path := fmt.Sprintf("/boards/%s/code_widgets?limit=%d", args.BoardID, limit)
	if args.Cursor != "" {
		path += "&cursor=" + args.Cursor
	}

	respBody, err := c.requestExperimental(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ListCodeWidgetsResult{}, wrapExperimentalCodeWidgetErr(err)
	}

	var resp struct {
		Data   []json.RawMessage `json:"data"`
		Cursor string            `json:"cursor,omitempty"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ListCodeWidgetsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	widgets := make([]CodeWidgetSummary, 0, len(resp.Data))
	for _, raw := range resp.Data {
		var widget struct {
			ID   string `json:"id"`
			Data struct {
				Code     string `json:"code"`
				Language string `json:"language"`
				Title    string `json:"title"`
			} `json:"data"`
			Parent *struct {
				ID string `json:"id"`
			} `json:"parent"`
		}
		if err := json.Unmarshal(raw, &widget); err != nil {
			continue
		}

		summary := CodeWidgetSummary{
			ID:          widget.ID,
			Title:       widget.Data.Title,
			Language:    widget.Data.Language,
			CodePreview: truncateCodeWidget(widget.Data.Code, codeWidgetPreviewLength),
		}
		if widget.Parent != nil {
			summary.ParentID = widget.Parent.ID
		}
		widgets = append(widgets, summary)
	}

	return ListCodeWidgetsResult{
		Widgets: widgets,
		Count:   len(widgets),
		HasMore: resp.Cursor != "",
		Cursor:  resp.Cursor,
		Message: fmt.Sprintf("Found %d code widgets", len(widgets)),
	}, nil
}

// UpdateCodeWidget updates a code widget's content, appearance, or parent.
// Position moves go through MoveCodeWidget. Uses the v2-experimental API.
func (c *Client) UpdateCodeWidget(ctx context.Context, args UpdateCodeWidgetArgs) (UpdateCodeWidgetResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateCodeWidgetResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateCodeWidgetResult{}, fmt.Errorf("invalid item_id: %w", err)
	}
	if err := validateCodeWidgetData(args.Code, args.Title); err != nil {
		return UpdateCodeWidgetResult{}, err
	}
	if args.ParentID != "" {
		if err := ValidateItemID(args.ParentID); err != nil {
			return UpdateCodeWidgetResult{}, fmt.Errorf("invalid parent_id: %w", err)
		}
	}

	reqBody := map[string]interface{}{}
	if data := buildCodeWidgetData(args.Code, args.Language, args.Title, args.LineNumbersVisible); data != nil {
		reqBody["data"] = data
	}
	if geo := buildCodeWidgetGeometry(args.Width, args.Height); geo != nil {
		reqBody["geometry"] = geo
	}
	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{"id": args.ParentID}
	}
	if len(reqBody) == 0 {
		return UpdateCodeWidgetResult{}, fmt.Errorf("no fields to update; supply at least one of code, language, title, line_numbers_visible, width, height, or parent_id (use miro_move_code_widget to change position)")
	}

	path := fmt.Sprintf("/boards/%s/code_widgets/%s", args.BoardID, args.ItemID)
	if _, err := c.requestExperimental(ctx, http.MethodPatch, path, reqBody); err != nil {
		return UpdateCodeWidgetResult{}, wrapExperimentalCodeWidgetErr(err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateCodeWidgetResult{
		ID:      args.ItemID,
		Message: "Code widget updated successfully",
	}, nil
}

// MoveCodeWidget moves a code widget to a new position via the dedicated
// position endpoint. Uses the v2-experimental API.
func (c *Client) MoveCodeWidget(ctx context.Context, args MoveCodeWidgetArgs) (MoveCodeWidgetResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return MoveCodeWidgetResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return MoveCodeWidgetResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := map[string]interface{}{
		"x":      args.X,
		"y":      args.Y,
		"origin": "center",
	}

	path := fmt.Sprintf("/boards/%s/code_widgets/%s/position", args.BoardID, args.ItemID)
	if _, err := c.requestExperimental(ctx, http.MethodPatch, path, reqBody); err != nil {
		return MoveCodeWidgetResult{}, wrapExperimentalCodeWidgetErr(err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return MoveCodeWidgetResult{
		ID:      args.ItemID,
		X:       args.X,
		Y:       args.Y,
		Message: fmt.Sprintf("Moved code widget to (%.0f, %.0f)", args.X, args.Y),
	}, nil
}

// DeleteCodeWidget removes a code widget.
// Uses the v2-experimental API.
func (c *Client) DeleteCodeWidget(ctx context.Context, args DeleteCodeWidgetArgs) (DeleteCodeWidgetResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return DeleteCodeWidgetResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return DeleteCodeWidgetResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	// Dry-run mode: return preview without deleting
	if args.DryRun {
		return DeleteCodeWidgetResult{
			Success: true,
			ID:      args.ItemID,
			Message: "[DRY RUN] Would delete code widget " + args.ItemID + " from board " + args.BoardID,
		}, nil
	}

	path := fmt.Sprintf("/boards/%s/code_widgets/%s", args.BoardID, args.ItemID)
	if _, err := c.requestExperimental(ctx, http.MethodDelete, path, nil); err != nil {
		return DeleteCodeWidgetResult{
			Success: false,
			ID:      args.ItemID,
			Message: fmt.Sprintf("Failed to delete code widget: %v", err),
		}, wrapExperimentalCodeWidgetErr(err)
	}

	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return DeleteCodeWidgetResult{
		Success: true,
		ID:      args.ItemID,
		Message: "Code widget deleted successfully",
	}, nil
}
