package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// ListTables lists data table format items on a board.
func (c *Client) ListTables(ctx context.Context, args ListTablesArgs) (ListTablesResult, error) {
	if args.BoardID == "" {
		return ListTablesResult{}, fmt.Errorf("board_id is required")
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	path := "/boards/" + args.BoardID + "/data_table_formats?limit=" + strconv.Itoa(limit)
	if args.Cursor != "" {
		path += "&cursor=" + args.Cursor
	}

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ListTablesResult{}, err
	}

	var resp struct {
		Data []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Position struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
			} `json:"position"`
			Geometry struct {
				Width  float64 `json:"width"`
				Height float64 `json:"height"`
			} `json:"geometry"`
			CreatedAt  string `json:"createdAt"`
			ModifiedAt string `json:"modifiedAt"`
			CreatedBy  struct {
				ID string `json:"id"`
			} `json:"createdBy"`
			ModifiedBy struct {
				ID string `json:"id"`
			} `json:"modifiedBy"`
		} `json:"data"`
		Total  int    `json:"total"`
		Size   int    `json:"size"`
		Cursor string `json:"cursor"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ListTablesResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	tables := make([]TableItem, len(resp.Data))
	for i, t := range resp.Data {
		tables[i] = TableItem{
			ID:         t.ID,
			Type:       t.Type,
			X:          t.Position.X,
			Y:          t.Position.Y,
			Width:      t.Geometry.Width,
			Height:     t.Geometry.Height,
			CreatedAt:  t.CreatedAt,
			ModifiedAt: t.ModifiedAt,
			CreatedBy:  t.CreatedBy.ID,
			ModifiedBy: t.ModifiedBy.ID,
			ItemURL:    BuildItemURL(args.BoardID, t.ID),
		}
	}

	return ListTablesResult{
		Tables:  tables,
		Count:   len(tables),
		Total:   resp.Total,
		Cursor:  resp.Cursor,
		Message: fmt.Sprintf("Found %d tables on board", len(tables)),
	}, nil
}

// GetTable gets metadata for a specific data table format item.
func (c *Client) GetTable(ctx context.Context, args GetTableArgs) (GetTableResult, error) {
	if args.BoardID == "" {
		return GetTableResult{}, fmt.Errorf("board_id is required")
	}
	if args.ItemID == "" {
		return GetTableResult{}, fmt.Errorf("item_id is required")
	}

	path := fmt.Sprintf("/boards/%s/data_table_formats/%s", args.BoardID, args.ItemID)
	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetTableResult{}, err
	}

	var resp struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Position struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"position"`
		Geometry struct {
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"geometry"`
		Parent *struct {
			ID string `json:"id"`
		} `json:"parent"`
		CreatedAt  string `json:"createdAt"`
		ModifiedAt string `json:"modifiedAt"`
		CreatedBy  struct {
			ID string `json:"id"`
		} `json:"createdBy"`
		ModifiedBy struct {
			ID string `json:"id"`
		} `json:"modifiedBy"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return GetTableResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	result := GetTableResult{
		ID:         resp.ID,
		Type:       resp.Type,
		X:          resp.Position.X,
		Y:          resp.Position.Y,
		Width:      resp.Geometry.Width,
		Height:     resp.Geometry.Height,
		CreatedAt:  resp.CreatedAt,
		ModifiedAt: resp.ModifiedAt,
		CreatedBy:  resp.CreatedBy.ID,
		ModifiedBy: resp.ModifiedBy.ID,
		ItemURL:    BuildItemURL(args.BoardID, resp.ID),
		Message:    "Retrieved table metadata",
	}

	if resp.Parent != nil {
		result.ParentID = resp.Parent.ID
	}

	return result, nil
}
