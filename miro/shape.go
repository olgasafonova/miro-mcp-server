package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// =============================================================================
// Shape Operations - Create, Update
// =============================================================================

// CreateShapeExperimentalArgs contains arguments for creating a shape via experimental API.
type CreateShapeExperimentalArgs struct {
	BoardID     string  `json:"board_id"`
	Shape       string  `json:"shape"` // Flowchart stencil shape type
	Content     string  `json:"content"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	FillColor   string  `json:"fill_color,omitempty"`
	BorderColor string  `json:"border_color,omitempty"`
	ParentID    string  `json:"parent_id,omitempty"`
}

// CreateShape creates a shape on a board.
func (c *Client) CreateShape(ctx context.Context, args CreateShapeArgs) (CreateShapeResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateShapeResult{}, err
	}
	if args.Shape == "" {
		return CreateShapeResult{}, fmt.Errorf("shape type is required")
	}

	// Default dimensions
	width := args.Width
	if width == 0 {
		width = 200
	}
	height := args.Height
	if height == 0 {
		height = 200
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"shape":   args.Shape,
			"content": args.Content,
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
		"geometry": map[string]interface{}{
			"width":  width,
			"height": height,
		},
	}

	// Build style object with fill color and/or text color
	style := make(map[string]interface{})
	if args.Color != "" {
		fillColor, err := normalizeColor(args.Color)
		if err != nil {
			return CreateShapeResult{}, fmt.Errorf("color: %w", err)
		}
		style["fillColor"] = fillColor
	}
	if args.TextColor != "" {
		textColor, err := normalizeColor(args.TextColor)
		if err != nil {
			return CreateShapeResult{}, fmt.Errorf("text_color: %w", err)
		}
		style["color"] = textColor
	}
	if len(style) > 0 {
		reqBody["style"] = style
	}

	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/shapes", reqBody)
	if err != nil {
		return CreateShapeResult{}, err
	}

	var shape Shape
	if err := json.Unmarshal(respBody, &shape); err != nil {
		return CreateShapeResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate items list cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return CreateShapeResult{
		ID:      shape.ID,
		ItemURL: BuildItemURL(args.BoardID, shape.ID),
		Shape:   shape.Data.Shape,
		Content: shape.Data.Content,
		Message: fmt.Sprintf("Created %s shape", args.Shape),
	}, nil
}

// CreateShapeExperimental creates a shape using the v2-experimental API.
// Used for flowchart stencil shapes that require the experimental endpoint.
func (c *Client) CreateShapeExperimental(ctx context.Context, args CreateShapeExperimentalArgs) (CreateShapeResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateShapeResult{}, err
	}
	if args.Shape == "" {
		return CreateShapeResult{}, fmt.Errorf("shape type is required")
	}

	// Default dimensions
	width := args.Width
	if width == 0 {
		width = 200
	}
	height := args.Height
	if height == 0 {
		height = 200
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"shape":   args.Shape,
			"content": args.Content,
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
		"geometry": map[string]interface{}{
			"width":  width,
			"height": height,
		},
	}

	// Build style with fill and border colors
	style := make(map[string]interface{})
	if args.FillColor != "" {
		fillColor, err := normalizeColor(args.FillColor)
		if err != nil {
			return CreateShapeResult{}, fmt.Errorf("fill_color: %w", err)
		}
		style["fillColor"] = fillColor
	}
	if args.BorderColor != "" {
		borderColor, err := normalizeColor(args.BorderColor)
		if err != nil {
			return CreateShapeResult{}, fmt.Errorf("border_color: %w", err)
		}
		style["borderColor"] = borderColor
	}
	if len(style) > 0 {
		reqBody["style"] = style
	}

	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.requestExperimental(ctx, http.MethodPost, "/boards/"+args.BoardID+"/shapes", reqBody)
	if err != nil {
		return CreateShapeResult{}, err
	}

	var shape Shape
	if err := json.Unmarshal(respBody, &shape); err != nil {
		return CreateShapeResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate items list cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return CreateShapeResult{
		ID:      shape.ID,
		ItemURL: BuildItemURL(args.BoardID, shape.ID),
		Shape:   shape.Data.Shape,
		Content: shape.Data.Content,
		Message: fmt.Sprintf("Created %s stencil shape", args.Shape),
	}, nil
}

// CreateFlowchartShape creates a flowchart shape using the v2-experimental API.
// Wraps CreateShapeExperimental with tool-friendly argument types.
func (c *Client) CreateFlowchartShape(ctx context.Context, args CreateFlowchartShapeArgs) (CreateShapeResult, error) {
	return c.CreateShapeExperimental(ctx, CreateShapeExperimentalArgs(args))
}

// UpdateShape updates a shape using the dedicated shapes endpoint.
func (c *Client) UpdateShape(ctx context.Context, args UpdateShapeArgs) (UpdateShapeResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateShapeResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateShapeResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := make(map[string]interface{})

	// Build data section
	data := make(map[string]interface{})
	if args.Content != nil {
		data["content"] = *args.Content
	}
	if args.ShapeType != nil {
		data["shape"] = *args.ShapeType
	}
	if len(data) > 0 {
		reqBody["data"] = data
	}

	// Build style section
	style := make(map[string]interface{})
	if args.Color != nil {
		fillColor, err := normalizeColor(*args.Color)
		if err != nil {
			return UpdateShapeResult{}, fmt.Errorf("color: %w", err)
		}
		style["fillColor"] = fillColor
	}
	if args.TextColor != nil {
		fontColor, err := normalizeColor(*args.TextColor)
		if err != nil {
			return UpdateShapeResult{}, fmt.Errorf("text_color: %w", err)
		}
		style["fontColor"] = fontColor
	}
	if len(style) > 0 {
		reqBody["style"] = style
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
		return UpdateShapeResult{
			ID:      args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	path := "/boards/" + args.BoardID + "/shapes/" + args.ItemID
	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateShapeResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Content string `json:"content"`
			Shape   string `json:"shape"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateShapeResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateShapeResult{
		ID:        resp.ID,
		ShapeType: resp.Data.Shape,
		Content:   resp.Data.Content,
		Message:   "Shape updated successfully",
	}, nil
}
