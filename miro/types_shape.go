package miro

// =============================================================================
// Create Shape
// =============================================================================

// CreateShapeArgs contains parameters for creating a shape.
type CreateShapeArgs struct {
	BoardID           string  `json:"board_id" jsonschema:"Board ID"`
	Shape             string  `json:"shape" jsonschema:"Shape type: rectangle, circle, triangle, rhombus, round_rectangle, etc."`
	Content           string  `json:"content,omitempty" jsonschema:"Text inside the shape"`
	X                 float64 `json:"x,omitempty" jsonschema:"X position. On canvas: absolute. Inside a frame (parent_id set): relative to frame's TOP-LEFT (0 = frame's left edge), and the shape's center is placed at this x."`
	Y                 float64 `json:"y,omitempty" jsonschema:"Y position. On canvas: absolute. Inside a frame (parent_id set): relative to frame's TOP-LEFT, shape center is placed at this y. Y increases downward."`
	Width             float64 `json:"width,omitempty" jsonschema:"Width in pixels (default 200)"`
	Height            float64 `json:"height,omitempty" jsonschema:"Height in pixels (default 200)"`
	Color             string  `json:"color,omitempty" jsonschema:"Fill/background color: 6-char hex like #006400 or named (red, orange, yellow, green, blue, purple, pink, gray, white, black)"`
	TextColor         string  `json:"text_color,omitempty" jsonschema:"Text color: 6-char hex like #ffffff or named (red, orange, yellow, green, blue, purple, pink, gray, white, black)"`
	TextAlign         string  `json:"text_align,omitempty" jsonschema:"Horizontal text alignment: left, center (default), right"`
	TextAlignVertical string  `json:"text_align_vertical,omitempty" jsonschema:"Vertical text alignment: top, middle (default), bottom. Note: 'middle' aligns to the center of the bounding box, which for triangles/hexagons is not the visual centroid."`
	ParentID          string  `json:"parent_id,omitempty" jsonschema:"Frame ID to place shape in. Coords (x, y) are then relative to the frame's TOP-LEFT; the shape's CENTER is placed at (x, y). Account for shape width/height when picking coords to keep it inside the frame."`
}

// CreateShapeResult contains the created shape.
type CreateShapeResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Shape   string `json:"shape"`
	Content string `json:"content,omitempty"`
	Message string `json:"message"`
}

// =============================================================================
// Update Shape
// =============================================================================

// UpdateShapeArgs contains parameters for updating a shape via dedicated endpoint.
type UpdateShapeArgs struct {
	BoardID           string   `json:"board_id" jsonschema:"Board ID"`
	ItemID            string   `json:"item_id" jsonschema:"Shape ID to update"`
	Content           *string  `json:"content,omitempty" jsonschema:"New text inside shape"`
	ShapeType         *string  `json:"shape_type,omitempty" jsonschema:"New shape type: rectangle, circle, triangle, rhombus, round_rectangle, parallelogram, trapezoid, pentagon, hexagon, star, flow_chart_predefined_process, etc."`
	Color             *string  `json:"color,omitempty" jsonschema:"New fill color: 6-char hex like #006400 or named (red, orange, yellow, green, blue, purple, pink, gray, white, black)"`
	TextColor         *string  `json:"text_color,omitempty" jsonschema:"New text color: 6-char hex like #ffffff or named (red, orange, yellow, green, blue, purple, pink, gray, white, black)"`
	TextAlign         *string  `json:"text_align,omitempty" jsonschema:"Horizontal text alignment: left, center, right"`
	TextAlignVertical *string  `json:"text_align_vertical,omitempty" jsonschema:"Vertical text alignment: top, middle, bottom"`
	X                 *float64 `json:"x,omitempty" jsonschema:"New X position"`
	Y                 *float64 `json:"y,omitempty" jsonschema:"New Y position"`
	Width             *float64 `json:"width,omitempty" jsonschema:"New width"`
	Height            *float64 `json:"height,omitempty" jsonschema:"New height"`
	ParentID          *string  `json:"parent_id,omitempty" jsonschema:"Move to frame (empty string removes from frame)"`
}

// UpdateShapeResult contains the updated shape details.
type UpdateShapeResult struct {
	ID        string `json:"id"`
	ShapeType string `json:"shape_type,omitempty"`
	Content   string `json:"content,omitempty"`
	Message   string `json:"message"`
}

// =============================================================================
// Create Shape (Experimental)
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

// =============================================================================
// Create Flowchart Shape (Experimental)
// =============================================================================

// CreateFlowchartShapeArgs contains parameters for creating a flowchart shape
// via the experimental API. Supports additional stencil shapes beyond the standard API.
type CreateFlowchartShapeArgs struct {
	BoardID     string  `json:"board_id" jsonschema:"Board ID"`
	Shape       string  `json:"shape" jsonschema:"Flowchart shape type: rectangle, round_rectangle, circle, rhombus, parallelogram, trapezoid, pentagon, hexagon, star, flow_chart_predefined_process, wedge_round_rectangle_callout, etc."`
	Content     string  `json:"content,omitempty" jsonschema:"Text inside the shape"`
	X           float64 `json:"x,omitempty" jsonschema:"X position"`
	Y           float64 `json:"y,omitempty" jsonschema:"Y position"`
	Width       float64 `json:"width,omitempty" jsonschema:"Width in pixels (default 200)"`
	Height      float64 `json:"height,omitempty" jsonschema:"Height in pixels (default 200)"`
	FillColor   string  `json:"fill_color,omitempty" jsonschema:"Fill/background color: 6-char hex like #006400 or named (red, orange, yellow, green, blue, purple, pink, gray, white, black)"`
	BorderColor string  `json:"border_color,omitempty" jsonschema:"Border color: 6-char hex like #000000 or named (red, orange, yellow, green, blue, purple, pink, gray, white, black)"`
	ParentID    string  `json:"parent_id,omitempty" jsonschema:"Frame ID to place shape in"`
}
