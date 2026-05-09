package diagrams

// MiroShape represents a shape to be created in Miro.
type MiroShape struct {
	Shape       string  // Miro shape type
	Content     string  // Text content
	X           float64 // Center X position
	Y           float64 // Center Y position
	Width       float64
	Height      float64
	Color       string // Fill color (optional)
	IsStencil   bool   // True if this is a flowchart stencil shape (requires experimental API)
	BorderColor string // Border color (optional, for stencil styling)
}

// MiroConnector represents a connector to be created in Miro.
type MiroConnector struct {
	StartItemIndex int    // Index in shapes array
	EndItemIndex   int    // Index in shapes array
	Caption        string // Optional label
	Style          string // elbowed, straight, curved
	StartCap       string // none, arrow, etc.
	EndCap         string // none, arrow, etc.
}

// MiroFrame represents a frame to be created in Miro.
type MiroFrame struct {
	Title  string
	X      float64
	Y      float64
	Width  float64
	Height float64
	Color  string
}

// MiroOutput contains all Miro items to create.
type MiroOutput struct {
	Shapes     []MiroShape
	Connectors []MiroConnector
	Frames     []MiroFrame
}

// ConvertToMiro converts a laid-out diagram to Miro API items.
// Automatically detects sequence diagrams and uses appropriate converter.
func ConvertToMiro(diagram *Diagram) *MiroOutput {
	return ConvertToMiroWithOptions(diagram, false)
}

// ConvertToMiroWithOptions converts a diagram with configurable options.
// When useStencils is true, uses professional flowchart stencil shapes.
func ConvertToMiroWithOptions(diagram *Diagram, useStencils bool) *MiroOutput {
	if diagram.Type == TypeSequence {
		return ConvertSequenceToMiro(diagram)
	}
	return convertFlowchartToMiroWithOptions(diagram, useStencils)
}

// convertFlowchartToMiro converts a flowchart diagram to Miro items.
func convertFlowchartToMiro(diagram *Diagram) *MiroOutput {
	return convertFlowchartToMiroWithOptions(diagram, false)
}

// convertFlowchartToMiroWithOptions converts a flowchart with optional stencil shapes.
func convertFlowchartToMiroWithOptions(diagram *Diagram, useStencils bool) *MiroOutput {
	output := &MiroOutput{
		Shapes:     make([]MiroShape, 0),
		Connectors: make([]MiroConnector, 0),
		Frames:     make([]MiroFrame, 0),
	}

	nodeToIndex := appendFlowchartShapes(output, diagram, useStencils)
	appendFlowchartConnectors(output, diagram, nodeToIndex)
	appendFlowchartFrames(output, diagram)
	return output
}

// appendFlowchartShapes converts each diagram node to a MiroShape (stencil or
// plain) and appends to output. Returns the node-id→shape-index map needed by
// the connector pass.
func appendFlowchartShapes(output *MiroOutput, diagram *Diagram, useStencils bool) map[string]int {
	nodeToIndex := make(map[string]int, len(diagram.Nodes))
	for id, node := range diagram.Nodes {
		shape := buildFlowchartShape(node, useStencils)
		if node.Color != "" {
			shape.Color = node.Color
		}
		nodeToIndex[id] = len(output.Shapes)
		output.Shapes = append(output.Shapes, shape)
	}
	return nodeToIndex
}

// buildFlowchartShape produces a MiroShape for one diagram node, picking the
// stencil or plain shape mapping. Caller may override Color afterward.
func buildFlowchartShape(node *Node, useStencils bool) MiroShape {
	base := MiroShape{
		Content: node.Label,
		X:       node.X + node.Width/2,
		Y:       node.Y + node.Height/2,
		Width:   node.Width,
		Height:  node.Height,
	}
	if useStencils {
		base.Shape = convertShapeToStencil(node.Shape)
		base.Color = getStencilColor(node.Shape)
		base.IsStencil = true
		base.BorderColor = getStencilBorderColor(node.Shape)
		return base
	}
	base.Shape = convertShape(node.Shape)
	base.Color = getShapeColor(node.Shape)
	return base
}

// appendFlowchartConnectors emits one MiroConnector per edge whose endpoints
// were converted to shapes. Edges referencing missing nodes are skipped.
func appendFlowchartConnectors(output *MiroOutput, diagram *Diagram, nodeToIndex map[string]int) {
	for _, edge := range diagram.Edges {
		startIdx, ok1 := nodeToIndex[edge.FromID]
		endIdx, ok2 := nodeToIndex[edge.ToID]
		if !ok1 || !ok2 {
			continue
		}
		output.Connectors = append(output.Connectors, MiroConnector{
			StartItemIndex: startIdx,
			EndItemIndex:   endIdx,
			Caption:        edge.Label,
			Style:          "elbowed",
			StartCap:       convertArrowType(edge.StartCap),
			EndCap:         convertArrowType(edge.EndCap),
		})
	}
}

// appendFlowchartFrames emits one MiroFrame per non-empty subgraph, sized to
// the bounding box of its member nodes plus a fixed padding.
func appendFlowchartFrames(output *MiroOutput, diagram *Diagram) {
	const padding = 40.0
	const titleSpace = 30.0
	for _, sg := range diagram.SubGraphs {
		if len(sg.NodeIDs) == 0 {
			continue
		}
		bbox, ok := subgraphBounds(diagram, sg.NodeIDs)
		if !ok {
			continue
		}
		w := bbox.maxX - bbox.minX + 2*padding
		h := bbox.maxY - bbox.minY + 2*padding + titleSpace
		output.Frames = append(output.Frames, MiroFrame{
			Title:  sg.Label,
			X:      bbox.minX - padding + w/2,
			Y:      bbox.minY - padding - titleSpace + h/2,
			Width:  w,
			Height: h,
			Color:  "#F5F5F5",
		})
	}
}

// bounds is the axis-aligned bounding box of a set of nodes.
type bounds struct{ minX, minY, maxX, maxY float64 }

// subgraphBounds returns the bounding box covering nodeIDs that exist in
// diagram.Nodes. ok=false if every referenced node is missing.
func subgraphBounds(diagram *Diagram, nodeIDs []string) (bounds, bool) {
	bbox := bounds{minX: 1e9, minY: 1e9, maxX: -1e9, maxY: -1e9}
	found := false
	for _, id := range nodeIDs {
		node := diagram.Nodes[id]
		if node == nil {
			continue
		}
		found = true
		if node.X < bbox.minX {
			bbox.minX = node.X
		}
		if node.Y < bbox.minY {
			bbox.minY = node.Y
		}
		if node.X+node.Width > bbox.maxX {
			bbox.maxX = node.X + node.Width
		}
		if node.Y+node.Height > bbox.maxY {
			bbox.maxY = node.Y + node.Height
		}
	}
	return bbox, found
}

// miroShapeNames maps internal shapes to Miro's shape vocabulary.
var miroShapeNames = map[NodeShape]string{
	ShapeRectangle:        "rectangle",
	ShapeRoundedRectangle: "round_rectangle",
	ShapeDiamond:          "rhombus",
	ShapeCircle:           "circle",
	ShapeStadium:          "pill",
	ShapeCylinder:         "can",
	ShapeParallelogram:    "parallelogram",
	ShapeHexagon:          "hexagon",
	ShapeTrapezoid:        "trapezoid",
}

// convertShape maps internal shape to Miro shape name.
func convertShape(shape NodeShape) string {
	if name, ok := miroShapeNames[shape]; ok {
		return name
	}
	return "rectangle"
}

// convertArrowType maps internal arrow type to Miro cap style.
func convertArrowType(arrow ArrowType) string {
	switch arrow {
	case ArrowNone:
		return "none"
	case ArrowNormal:
		return "arrow"
	case ArrowCircle:
		return "filled_circle"
	case ArrowCross:
		return "diamond"
	default:
		return "none"
	}
}

// getShapeColor returns a default color based on shape type.
func getShapeColor(shape NodeShape) string {
	switch shape {
	case ShapeDiamond:
		return "#FFE066" // Yellow for decisions
	case ShapeCircle:
		return "#B8E986" // Green for start/end
	case ShapeStadium:
		return "#B3E5FC" // Light blue for process
	case ShapeParallelogram:
		return "#E1BEE7" // Light purple for I/O
	case ShapeHexagon:
		return "#FFCCBC" // Light orange for preparation
	default:
		return "#E3F2FD" // Default light blue
	}
}

// =============================================================================
// Flowchart Stencil Shape Conversion (v2-experimental API)
// =============================================================================

// stencilShapeNames maps internal shapes to Miro flowchart stencil names
// (v2-experimental API). Multiple internal shapes can map to the same stencil
// (e.g. Rectangle and RoundedRectangle both map to flow_chart_process).
var stencilShapeNames = map[NodeShape]string{
	ShapeCircle:           "flow_chart_terminator",
	ShapeStadium:          "flow_chart_terminator",
	ShapeDiamond:          "flow_chart_decision",
	ShapeRectangle:        "flow_chart_process",
	ShapeRoundedRectangle: "flow_chart_process",
	ShapeParallelogram:    "flow_chart_input_output",
	ShapeHexagon:          "flow_chart_preparation",
	ShapeCylinder:         "flow_chart_database",
	ShapeTrapezoid:        "flow_chart_manual_operation",
}

// convertShapeToStencil maps internal shape to Miro flowchart stencil shape name.
// These shapes require the v2-experimental API endpoint.
func convertShapeToStencil(shape NodeShape) string {
	if name, ok := stencilShapeNames[shape]; ok {
		return name
	}
	return "flow_chart_process"
}

// stencilStyle pairs the fill and border color used together for a stencil shape.
type stencilStyle struct {
	fill   string
	border string
}

var (
	stencilStyleDefault = stencilStyle{fill: "#E3F2FD", border: "#1976D2"}

	// stencilStyles maps a shape category to its cohesive fill+border palette.
	// Adjacent categories share an entry: rectangles share with round-rectangles,
	// circles share with stadiums.
	stencilStyles = map[NodeShape]stencilStyle{
		ShapeCircle:           {fill: "#C8E6C9", border: "#4CAF50"}, // terminator (start/end)
		ShapeStadium:          {fill: "#C8E6C9", border: "#4CAF50"},
		ShapeDiamond:          {fill: "#FFF9C4", border: "#FFC107"}, // decisions
		ShapeRectangle:        {fill: "#BBDEFB", border: "#2196F3"}, // process
		ShapeRoundedRectangle: {fill: "#BBDEFB", border: "#2196F3"},
		ShapeParallelogram:    {fill: "#E1BEE7", border: "#9C27B0"}, // I/O
		ShapeHexagon:          {fill: "#FFE0B2", border: "#FF9800"}, // preparation
		ShapeCylinder:         {fill: "#B3E5FC", border: "#00BCD4"}, // database
		ShapeTrapezoid:        {fill: "#FFCCBC", border: "#FF5722"}, // manual operation
	}
)

// getStencilStyle returns the fill+border palette for a stencil shape.
func getStencilStyle(shape NodeShape) stencilStyle {
	if s, ok := stencilStyles[shape]; ok {
		return s
	}
	return stencilStyleDefault
}

// getStencilColor returns the fill color for a flowchart stencil shape.
func getStencilColor(shape NodeShape) string {
	return getStencilStyle(shape).fill
}

// getStencilBorderColor returns the matching border color for a stencil shape.
func getStencilBorderColor(shape NodeShape) string {
	return getStencilStyle(shape).border
}

// =============================================================================
// Sequence Diagram Converter
// =============================================================================

// SequenceLayout constants for sequence diagram rendering
const (
	seqParticipantWidth  = 120.0
	seqParticipantHeight = 50.0
	seqLifelineWidth     = 10.0 // Wide enough to be clearly visible
	seqAnchorSize        = 8.0  // Minimum size allowed by Miro API
	seqMessageSpacing    = 60.0
	seqLifelineColor     = "#90CAF9" // Visible blue for lifelines
	seqAnchorColor       = "#90CAF9" // Match lifeline color so anchors blend in
)

// participantInfo carries one sequence-diagram participant alongside the index
// at which its header shape lands in MiroOutput.Shapes.
type participantInfo struct {
	id    string
	node  *Node
	index int
}

// ConvertSequenceToMiro converts a sequence diagram to Miro items.
// Creates: participant boxes, lifeline anchors, and message connectors.
func ConvertSequenceToMiro(diagram *Diagram) *MiroOutput {
	output := &MiroOutput{
		Shapes:     make([]MiroShape, 0),
		Connectors: make([]MiroConnector, 0),
		Frames:     make([]MiroFrame, 0),
	}

	participants := orderedParticipants(diagram)
	participantCenterX := appendParticipantHeaders(output, participants)
	appendLifelines(output, participants, diagram.Height, participantCenterX)
	appendSequenceMessages(output, diagram.Edges, participantCenterX)

	return output
}

// orderedParticipants returns the diagram's nodes sorted ascending by X.
func orderedParticipants(diagram *Diagram) []participantInfo {
	participants := make([]participantInfo, 0, len(diagram.Nodes))
	for id, node := range diagram.Nodes {
		participants = append(participants, participantInfo{id: id, node: node})
	}
	for i := 0; i < len(participants); i++ {
		for j := i + 1; j < len(participants); j++ {
			if participants[i].node.X > participants[j].node.X {
				participants[i], participants[j] = participants[j], participants[i]
			}
		}
	}
	return participants
}

// appendParticipantHeaders emits one header shape per participant and returns
// a map from participant id to its center X coordinate.
func appendParticipantHeaders(output *MiroOutput, participants []participantInfo) map[string]float64 {
	centerX := make(map[string]float64, len(participants))
	for i, p := range participants {
		shape := buildParticipantHeader(p.node)
		centerX[p.id] = p.node.X + p.node.Width/2
		participants[i].index = len(output.Shapes)
		output.Shapes = append(output.Shapes, shape)
	}
	return centerX
}

// buildParticipantHeader returns the header shape for a participant. Actors
// (shape=circle) get a circle + yellow palette; everyone else gets a rectangle
// with the participant blue.
func buildParticipantHeader(node *Node) MiroShape {
	shape := MiroShape{
		Shape:   "rectangle",
		Content: node.Label,
		X:       node.X + node.Width/2,
		Y:       node.Y + node.Height/2,
		Width:   node.Width,
		Height:  node.Height,
		Color:   "#E3F2FD",
	}
	if node.Shape == ShapeCircle {
		shape.Shape = "circle"
		shape.Color = "#FFF9C4"
	}
	return shape
}

// appendLifelines emits a thin vertical rectangle under each participant.
func appendLifelines(output *MiroOutput, participants []participantInfo, diagramHeight float64, centerX map[string]float64) {
	lifelineHeight := diagramHeight - seqParticipantHeight - 30
	if lifelineHeight < 50 {
		lifelineHeight = 100
	}
	for _, p := range participants {
		output.Shapes = append(output.Shapes, MiroShape{
			Shape:  "rectangle",
			X:      centerX[p.id],
			Y:      p.node.Y + p.node.Height + lifelineHeight/2 + 10,
			Width:  seqLifelineWidth,
			Height: lifelineHeight,
			Color:  seqLifelineColor,
		})
	}
}

// appendSequenceMessages emits two anchor circles plus a connector for each
// edge whose endpoints map to known participants.
func appendSequenceMessages(output *MiroOutput, edges []*Edge, centerX map[string]float64) {
	for _, edge := range edges {
		fromX, fromExists := centerX[edge.FromID]
		toX, toExists := centerX[edge.ToID]
		if !fromExists || !toExists {
			continue
		}
		fromAnchorIdx := appendAnchor(output, fromX, edge.Y)
		toAnchorIdx := appendAnchor(output, toX, edge.Y)
		output.Connectors = append(output.Connectors, MiroConnector{
			StartItemIndex: fromAnchorIdx,
			EndItemIndex:   toAnchorIdx,
			Caption:        edge.Label,
			Style:          "straight",
			StartCap:       convertArrowType(edge.StartCap),
			EndCap:         convertArrowType(edge.EndCap),
		})
	}
}

// appendAnchor adds a small circle at (x,y) used as a connector endpoint, and
// returns its index in output.Shapes.
func appendAnchor(output *MiroOutput, x, y float64) int {
	idx := len(output.Shapes)
	output.Shapes = append(output.Shapes, MiroShape{
		Shape:  "circle",
		X:      x,
		Y:      y,
		Width:  seqAnchorSize,
		Height: seqAnchorSize,
		Color:  seqAnchorColor,
	})
	return idx
}
