package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

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

// parseCodeWidgetSummaries converts raw list entries into summaries,
// skipping entries that fail to parse.
func parseCodeWidgetSummaries(entries []json.RawMessage) []CodeWidgetSummary {
	widgets := make([]CodeWidgetSummary, 0, len(entries))
	for _, raw := range entries {
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
	return widgets
}

// clampCodeWidgetLimit normalizes a requested page size to the API bounds.
func clampCodeWidgetLimit(limit int) int {
	if limit <= 0 {
		return DefaultItemLimit
	}
	if limit > MaxItemLimitExtended {
		return MaxItemLimitExtended
	}
	return limit
}

// ListCodeWidgets retrieves code widgets on a board with cursor pagination.
// Uses the v2-experimental API.
func (c *Client) ListCodeWidgets(ctx context.Context, args ListCodeWidgetsArgs) (ListCodeWidgetsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListCodeWidgetsResult{}, err
	}

	limit := clampCodeWidgetLimit(args.Limit)

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

	widgets := parseCodeWidgetSummaries(resp.Data)

	return ListCodeWidgetsResult{
		Widgets: widgets,
		Count:   len(widgets),
		HasMore: resp.Cursor != "",
		Cursor:  resp.Cursor,
		Message: fmt.Sprintf("Found %d code widgets", len(widgets)),
	}, nil
}
