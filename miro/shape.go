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

// defaultShapeDimension is the fallback width/height applied when a shape is
// created without an explicit dimension.
const defaultShapeDimension = 200.0

// shapeDefaultDimensions applies the per-shape default to any zero coordinate.
func shapeDefaultDimensions(width, height float64) (float64, float64) {
	if width == 0 {
		width = defaultShapeDimension
	}
	if height == 0 {
		height = defaultShapeDimension
	}
	return width, height
}

// shapeCoreBody bundles the "core" parameters every shape-create call shares
// (data, position, geometry, parent). Style is built per-call because the
// experimental endpoint uses different style fields than the standard one.
type shapeCoreBody struct {
	boardID  string
	shape    string
	content  string
	x, y     float64
	width    float64
	height   float64
	parentID string
}

// buildShapeBaseBody assembles the data + position + geometry + parent sections
// shared by CreateShape and CreateShapeExperimental. The caller adds its own
// style block before sending.
func buildShapeBaseBody(c shapeCoreBody) map[string]interface{} {
	width, height := shapeDefaultDimensions(c.width, c.height)
	body := map[string]interface{}{
		"data": map[string]interface{}{
			"shape":   c.shape,
			"content": c.content,
		},
		"position": map[string]interface{}{
			"x":      c.x,
			"y":      c.y,
			"origin": "center",
		},
		"geometry": map[string]interface{}{
			"width":  width,
			"height": height,
		},
	}
	if c.parentID != "" {
		body["parent"] = map[string]interface{}{"id": c.parentID}
	}
	return body
}

// shapeColorSpec describes a single optional color slot in a style map: the
// key it lands under, the error tag if normalization fails, and the raw value.
type shapeColorSpec struct {
	styleKey string
	errorTag string
	value    string
}

// applyOptionalColor normalizes the supplied color and stores it under styleKey
// in the style map when the input is non-empty. errorTag is used to wrap any
// normalization error (e.g. "color", "fill_color", "border_color").
func applyOptionalColor(style map[string]interface{}, spec shapeColorSpec) error {
	if spec.value == "" {
		return nil
	}
	normalized, err := normalizeColor(spec.value)
	if err != nil {
		return fmt.Errorf("%s: %w", spec.errorTag, err)
	}
	style[spec.styleKey] = normalized
	return nil
}

// buildShapeStyle assembles a style map from a slice of optional color slots.
// Empty values are skipped; the first normalization error short-circuits.
func buildShapeStyle(specs []shapeColorSpec) (map[string]interface{}, error) {
	style := make(map[string]interface{})
	for _, spec := range specs {
		if err := applyOptionalColor(style, spec); err != nil {
			return nil, err
		}
	}
	return style, nil
}

// buildCreateShapeStyle assembles the style block for the standard CreateShape
// endpoint, where Color maps to fillColor and TextColor maps to color (text).
func buildCreateShapeStyle(color, textColor string) (map[string]interface{}, error) {
	return buildShapeStyle([]shapeColorSpec{
		{"fillColor", "color", color},
		{"color", "text_color", textColor},
	})
}

// buildExperimentalShapeStyle assembles the style block for the v2-experimental
// endpoint, where FillColor maps to fillColor and BorderColor maps to borderColor.
func buildExperimentalShapeStyle(fillColor, borderColor string) (map[string]interface{}, error) {
	return buildShapeStyle([]shapeColorSpec{
		{"fillColor", "fill_color", fillColor},
		{"borderColor", "border_color", borderColor},
	})
}

// applyOptionalColorPtr is the *string variant of applyOptionalColor used by
// PATCH-style endpoints where a nil pointer means "leave field unchanged".
// It delegates to applyOptionalColor after dereferencing.
func applyOptionalColorPtr(style map[string]interface{}, styleKey, errorTag string, value *string) error {
	if value == nil {
		return nil
	}
	return applyOptionalColor(style, shapeColorSpec{styleKey: styleKey, errorTag: errorTag, value: *value})
}

// enumField bundles the metadata for a string-enum slot in a style map:
// the key it lands under, the error tag for validation failures, and the
// allowed value set. Used by applyOptionalEnum / applyOptionalEnumPtr.
type enumField struct {
	styleKey string
	errorTag string
	allowed  []string
}

// shapeTextAlignField governs the horizontal text-alignment slot on shapes.
var shapeTextAlignField = enumField{
	styleKey: "textAlign",
	errorTag: "text_align",
	allowed:  []string{"left", "center", "right"},
}

// shapeTextAlignVerticalField governs the vertical text-alignment slot on shapes.
var shapeTextAlignVerticalField = enumField{
	styleKey: "textAlignVertical",
	errorTag: "text_align_vertical",
	allowed:  []string{"top", "middle", "bottom"},
}

// applyOptionalEnum validates value against field.allowed and stores it under
// field.styleKey when non-empty.
func applyOptionalEnum(style map[string]interface{}, field enumField, value string) error {
	if value == "" {
		return nil
	}
	for _, a := range field.allowed {
		if value == a {
			style[field.styleKey] = value
			return nil
		}
	}
	return fmt.Errorf("%s: must be one of %v, got %q", field.errorTag, field.allowed, value)
}

// applyOptionalEnumPtr is the *string variant for PATCH-style updates where a
// nil pointer means "leave field unchanged".
func applyOptionalEnumPtr(style map[string]interface{}, field enumField, value *string) error {
	if value == nil {
		return nil
	}
	return applyOptionalEnum(style, field, *value)
}

// applyShapeTextAlign attaches text_align and text_align_vertical to the style
// map after validation. Used by both CreateShape and UpdateShape.
func applyShapeTextAlign(style map[string]interface{}, textAlign, textAlignVertical string) error {
	if err := applyOptionalEnum(style, shapeTextAlignField, textAlign); err != nil {
		return err
	}
	return applyOptionalEnum(style, shapeTextAlignVerticalField, textAlignVertical)
}

// applyShapeTextAlignPtr is the *string variant for PATCH-style updates.
func applyShapeTextAlignPtr(style map[string]interface{}, textAlign, textAlignVertical *string) error {
	if err := applyOptionalEnumPtr(style, shapeTextAlignField, textAlign); err != nil {
		return err
	}
	return applyOptionalEnumPtr(style, shapeTextAlignVerticalField, textAlignVertical)
}

// shapeRequestFunc is the signature of c.request and c.requestExperimental.
// Used by executeShapeCreate to dispatch to the chosen API surface.
type shapeRequestFunc func(ctx context.Context, method, path string, body interface{}) ([]byte, error)

// shapeCreateExec bundles the per-call inputs to executeShapeCreate.
type shapeCreateExec struct {
	core          shapeCoreBody
	style         map[string]interface{}
	requestFunc   shapeRequestFunc
	successFormat string // fmt format with one %s for the shape name
}

// executeShapeCreate runs the shared body of CreateShape and CreateShapeExperimental:
// build the multipart-style body with optional style, dispatch via requestFunc,
// parse the response, invalidate cache, and assemble the result.
func (c *Client) executeShapeCreate(ctx context.Context, exec shapeCreateExec) (CreateShapeResult, error) {
	reqBody := buildShapeBaseBody(exec.core)
	if len(exec.style) > 0 {
		reqBody["style"] = exec.style
	}

	respBody, err := exec.requestFunc(ctx, http.MethodPost, "/boards/"+exec.core.boardID+"/shapes", reqBody)
	if err != nil {
		return CreateShapeResult{}, err
	}

	var shape Shape
	if err := json.Unmarshal(respBody, &shape); err != nil {
		return CreateShapeResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidatePrefix("items:" + exec.core.boardID)

	return CreateShapeResult{
		ID:      shape.ID,
		ItemURL: BuildItemURL(exec.core.boardID, shape.ID),
		Shape:   shape.Data.Shape,
		Content: shape.Data.Content,
		Message: fmt.Sprintf(exec.successFormat, exec.core.shape),
	}, nil
}

// validateShapeCreateArgs runs the shared up-front validation for both
// CreateShape and CreateShapeExperimental.
func validateShapeCreateArgs(boardID, shape string) error {
	if err := ValidateBoardID(boardID); err != nil {
		return err
	}
	if shape == "" {
		return fmt.Errorf("shape type is required")
	}
	return nil
}

// toCoreBody projects CreateShapeArgs onto the internal shapeCoreBody.
func (a CreateShapeArgs) toCoreBody() shapeCoreBody {
	return shapeCoreBody{
		boardID:  a.BoardID,
		shape:    a.Shape,
		content:  a.Content,
		x:        a.X,
		y:        a.Y,
		width:    a.Width,
		height:   a.Height,
		parentID: a.ParentID,
	}
}

// toCoreBody projects CreateShapeExperimentalArgs onto the internal shapeCoreBody.
func (a CreateShapeExperimentalArgs) toCoreBody() shapeCoreBody {
	return shapeCoreBody{
		boardID:  a.BoardID,
		shape:    a.Shape,
		content:  a.Content,
		x:        a.X,
		y:        a.Y,
		width:    a.Width,
		height:   a.Height,
		parentID: a.ParentID,
	}
}

// CreateShape creates a shape on a board.
func (c *Client) CreateShape(ctx context.Context, args CreateShapeArgs) (CreateShapeResult, error) {
	if err := validateShapeCreateArgs(args.BoardID, args.Shape); err != nil {
		return CreateShapeResult{}, err
	}

	style, err := buildCreateShapeStyle(args.Color, args.TextColor)
	if err != nil {
		return CreateShapeResult{}, err
	}
	if err := applyShapeTextAlign(style, args.TextAlign, args.TextAlignVertical); err != nil {
		return CreateShapeResult{}, err
	}

	return c.executeShapeCreate(ctx, shapeCreateExec{
		core:          args.toCoreBody(),
		style:         style,
		requestFunc:   c.request,
		successFormat: "Created %s shape",
	})
}

// CreateShapeExperimental creates a shape using the v2-experimental API.
// Used for flowchart stencil shapes that require the experimental endpoint.
func (c *Client) CreateShapeExperimental(ctx context.Context, args CreateShapeExperimentalArgs) (CreateShapeResult, error) {
	if err := validateShapeCreateArgs(args.BoardID, args.Shape); err != nil {
		return CreateShapeResult{}, err
	}

	style, err := buildExperimentalShapeStyle(args.FillColor, args.BorderColor)
	if err != nil {
		return CreateShapeResult{}, err
	}

	return c.executeShapeCreate(ctx, shapeCreateExec{
		core:          args.toCoreBody(),
		style:         style,
		requestFunc:   c.requestExperimental,
		successFormat: "Created %s stencil shape",
	})
}

// CreateFlowchartShape creates a flowchart shape using the v2-experimental API.
// Wraps CreateShapeExperimental with tool-friendly argument types.
func (c *Client) CreateFlowchartShape(ctx context.Context, args CreateFlowchartShapeArgs) (CreateShapeResult, error) {
	return c.CreateShapeExperimental(ctx, CreateShapeExperimentalArgs(args))
}

// buildShapeUpdateData assembles the "data" section for an UpdateShape call.
// Returns nil when nothing to update so the caller can omit the key entirely.
func buildShapeUpdateData(content, shapeType *string) map[string]interface{} {
	data := make(map[string]interface{})
	if content != nil {
		data["content"] = *content
	}
	if shapeType != nil {
		data["shape"] = *shapeType
	}
	if len(data) == 0 {
		return nil
	}
	return data
}

// buildShapeUpdateStyle assembles the "style" section for an UpdateShape call.
// Returns nil when no style fields are supplied. The standard shapes endpoint
// uses fillColor + fontColor (note: fontColor here, not color as in CreateShape).
func buildShapeUpdateStyle(color, textColor, textAlign, textAlignVertical *string) (map[string]interface{}, error) {
	style := make(map[string]interface{})
	if err := applyOptionalColorPtr(style, "fillColor", "color", color); err != nil {
		return nil, err
	}
	if err := applyOptionalColorPtr(style, "fontColor", "text_color", textColor); err != nil {
		return nil, err
	}
	if err := applyShapeTextAlignPtr(style, textAlign, textAlignVertical); err != nil {
		return nil, err
	}
	if len(style) == 0 {
		return nil, nil
	}
	return style, nil
}

// buildUpdateShapeBody assembles the PATCH body for UpdateShape, including
// only the sections the caller supplied. Returns an empty map when nothing to update.
func buildUpdateShapeBody(args UpdateShapeArgs) (map[string]interface{}, error) {
	reqBody := make(map[string]interface{})

	if data := buildShapeUpdateData(args.Content, args.ShapeType); data != nil {
		reqBody["data"] = data
	}

	style, err := buildShapeUpdateStyle(args.Color, args.TextColor, args.TextAlign, args.TextAlignVertical)
	if err != nil {
		return nil, err
	}
	if style != nil {
		reqBody["style"] = style
	}

	if pos := buildUpdatePosition(args.X, args.Y); pos != nil {
		reqBody["position"] = pos
	}
	if geom := buildUpdateGeometry(args.Width, args.Height); geom != nil {
		reqBody["geometry"] = geom
	}
	if args.ParentID != nil {
		// Empty string explicitly nulls parent (removes from frame).
		reqBody["parent"] = updateParentPayload(*args.ParentID)
	}
	return reqBody, nil
}

// UpdateShape updates a shape using the dedicated shapes endpoint.
func (c *Client) UpdateShape(ctx context.Context, args UpdateShapeArgs) (UpdateShapeResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateShapeResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateShapeResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody, err := buildUpdateShapeBody(args)
	if err != nil {
		return UpdateShapeResult{}, err
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
