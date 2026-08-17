package miro

// =============================================================================
// Diagram Generation Types
// =============================================================================

// GenerateDiagramArgs contains arguments for generating a diagram.
type GenerateDiagramArgs struct {
	BoardID     string  `json:"board_id" jsonschema:"Board ID to create the diagram on"`
	Diagram     string  `json:"diagram" jsonschema:"Diagram code in Mermaid format (flowchart/graph syntax)"`
	StartX      float64 `json:"start_x,omitempty" jsonschema:"Starting X position (default: 0)"`
	StartY      float64 `json:"start_y,omitempty" jsonschema:"Starting Y position (default: 0)"`
	NodeWidth   float64 `json:"node_width,omitempty" jsonschema:"Width of each node (default: 180)"`
	ParentID    string  `json:"parent_id,omitempty" jsonschema:"Parent frame ID to create diagram inside"`
	UseStencils bool    `json:"use_stencils,omitempty" jsonschema:"Use professional flowchart stencils instead of basic shapes. Provides better visual styling with proper flowchart symbols (terminator, process, decision, I/O)."`
	OutputMode  string  `json:"output_mode,omitempty" jsonschema:"Output mode: 'discrete' (default) returns individual items, 'grouped' groups all items together for easy move/delete, 'framed' creates a frame containing all items"`
}

// GenerateDiagramResult contains the result of diagram generation.
type GenerateDiagramResult struct {
	NodesCreated      int      `json:"nodes_created"`
	ConnectorsCreated int      `json:"connectors_created"`
	FramesCreated     int      `json:"frames_created"`
	NodeIDs           []string `json:"node_ids"`
	NodeURLs          []string `json:"node_urls,omitempty"`
	ConnectorIDs      []string `json:"connector_ids"`
	ConnectorURLs     []string `json:"connector_urls,omitempty"`
	FrameIDs          []string `json:"frame_ids,omitempty"`
	FrameURLs         []string `json:"frame_urls,omitempty"`
	DiagramWidth      float64  `json:"diagram_width"`
	DiagramHeight     float64  `json:"diagram_height"`
	Message           string   `json:"message"`
	// Compound output mode fields
	OutputMode  string `json:"output_mode,omitempty"`
	DiagramID   string `json:"diagram_id,omitempty"`
	DiagramURL  string `json:"diagram_url,omitempty"`
	DiagramType string `json:"diagram_type,omitempty"` // "group" or "frame"
	TotalItems  int    `json:"total_items,omitempty"`
}

// DiagramNode represents a node in the generated diagram (for response details).
type DiagramNode struct {
	ID    string  `json:"id"`
	Label string  `json:"label"`
	Shape string  `json:"shape"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
}

// =============================================================================
// Native Diagram Read Types (GET /v2/boards/{id}/diagrams)
// =============================================================================

// ListDiagramsArgs contains parameters for listing native diagram items on a board.
type ListDiagramsArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Max diagrams to return (default 10, max 50)"`
	Cursor  string `json:"cursor,omitempty" jsonschema:"Pagination cursor from previous response"`
}

// ListDiagramsResult contains the diagram items found on a board.
type ListDiagramsResult struct {
	Diagrams []DiagramItem `json:"diagrams"`
	Count    int           `json:"count"`
	Total    int           `json:"total"`
	Cursor   string        `json:"cursor,omitempty"`
	Message  string        `json:"message"`
}

// DiagramItem represents a native diagram item on a Miro board.
type DiagramItem struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Title      string  `json:"title,omitempty"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width,omitempty"`
	Height     float64 `json:"height,omitempty"`
	CreatedAt  string  `json:"created_at,omitempty"`
	ModifiedAt string  `json:"modified_at,omitempty"`
	CreatedBy  string  `json:"created_by,omitempty"`
	ModifiedBy string  `json:"modified_by,omitempty"`
	ItemURL    string  `json:"item_url,omitempty"`
}

// GetDiagramArgs contains parameters for getting a specific diagram item.
type GetDiagramArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"Diagram item ID"`
}

// GetDiagramResult contains the diagram item metadata.
type GetDiagramResult struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Title      string  `json:"title,omitempty"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width,omitempty"`
	Height     float64 `json:"height,omitempty"`
	CreatedAt  string  `json:"created_at,omitempty"`
	ModifiedAt string  `json:"modified_at,omitempty"`
	CreatedBy  string  `json:"created_by,omitempty"`
	ModifiedBy string  `json:"modified_by,omitempty"`
	ParentID   string  `json:"parent_id,omitempty"`
	ItemURL    string  `json:"item_url,omitempty"`
	Message    string  `json:"message"`
}
