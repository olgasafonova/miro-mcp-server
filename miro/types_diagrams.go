package miro

// =============================================================================
// Diagram Generation Types
// =============================================================================

// GenerateDiagramArgs contains arguments for generating a diagram.
type GenerateDiagramArgs struct {
	BoardID     string  `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID to create the diagram on"`
	Diagram     string  `json:"diagram" jsonschema:"required" jsonschema_description:"Diagram code in Mermaid format (flowchart/graph syntax)"`
	StartX      float64 `json:"start_x,omitempty" jsonschema_description:"Starting X position (default: 0)"`
	StartY      float64 `json:"start_y,omitempty" jsonschema_description:"Starting Y position (default: 0)"`
	NodeWidth   float64 `json:"node_width,omitempty" jsonschema_description:"Width of each node (default: 180)"`
	ParentID    string  `json:"parent_id,omitempty" jsonschema_description:"Parent frame ID to create diagram inside"`
	UseStencils bool    `json:"use_stencils,omitempty" jsonschema_description:"Use professional flowchart stencils instead of basic shapes. Provides better visual styling with proper flowchart symbols (terminator, process, decision, I/O)."`
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
}

// DiagramNode represents a node in the generated diagram (for response details).
type DiagramNode struct {
	ID    string  `json:"id"`
	Label string  `json:"label"`
	Shape string  `json:"shape"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
}
