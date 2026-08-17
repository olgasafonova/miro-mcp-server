package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// =============================================================================
// Native Diagram Reads (GET /v2/boards/{id}/diagrams)
//
// The public surface is read-only: POST returns 405 methodNotSupported, so
// diagram creation stays with Miro's hosted tooling. Distinct from
// GenerateDiagram, which is the local mermaid-to-shapes transform.
// =============================================================================

// ListDiagrams lists native diagram items on a board.
func (c *Client) ListDiagrams(ctx context.Context, args ListDiagramsArgs) (ListDiagramsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListDiagramsResult{}, err
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	path := "/boards/" + args.BoardID + "/diagrams?limit=" + strconv.Itoa(limit)
	if args.Cursor != "" {
		path += "&cursor=" + args.Cursor
	}

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ListDiagramsResult{}, err
	}

	var resp struct {
		Data []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Data struct {
				Title string `json:"title"`
			} `json:"data"`
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
		return ListDiagramsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	diagrams := make([]DiagramItem, len(resp.Data))
	for i, d := range resp.Data {
		diagrams[i] = DiagramItem{
			ID:         d.ID,
			Type:       d.Type,
			Title:      d.Data.Title,
			X:          d.Position.X,
			Y:          d.Position.Y,
			Width:      d.Geometry.Width,
			Height:     d.Geometry.Height,
			CreatedAt:  d.CreatedAt,
			ModifiedAt: d.ModifiedAt,
			CreatedBy:  d.CreatedBy.ID,
			ModifiedBy: d.ModifiedBy.ID,
			ItemURL:    BuildItemURL(args.BoardID, d.ID),
		}
	}

	return ListDiagramsResult{
		Diagrams: diagrams,
		Count:    len(diagrams),
		Total:    resp.Total,
		Cursor:   resp.Cursor,
		Message:  fmt.Sprintf("Found %d diagrams on board", len(diagrams)),
	}, nil
}

// GetDiagram gets metadata for a specific native diagram item.
func (c *Client) GetDiagram(ctx context.Context, args GetDiagramArgs) (GetDiagramResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetDiagramResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return GetDiagramResult{}, err
	}

	path := fmt.Sprintf("/boards/%s/diagrams/%s", args.BoardID, args.ItemID)
	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetDiagramResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Title string `json:"title"`
		} `json:"data"`
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
		return GetDiagramResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	result := GetDiagramResult{
		ID:         resp.ID,
		Type:       resp.Type,
		Title:      resp.Data.Title,
		X:          resp.Position.X,
		Y:          resp.Position.Y,
		Width:      resp.Geometry.Width,
		Height:     resp.Geometry.Height,
		CreatedAt:  resp.CreatedAt,
		ModifiedAt: resp.ModifiedAt,
		CreatedBy:  resp.CreatedBy.ID,
		ModifiedBy: resp.ModifiedBy.ID,
		ItemURL:    BuildItemURL(args.BoardID, resp.ID),
		Message:    "Retrieved diagram metadata",
	}

	if resp.Parent != nil {
		result.ParentID = resp.Parent.ID
	}

	return result, nil
}
