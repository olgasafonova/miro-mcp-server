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

	// Map node IDs to shape indices
	nodeToIndex := make(map[string]int)

	// Convert nodes to shapes
	for id, node := range diagram.Nodes {
		var shape MiroShape

		if useStencils {
			shape = MiroShape{
				Shape:       convertShapeToStencil(node.Shape),
				Content:     node.Label,
				X:           node.X + node.Width/2, // Miro uses center position
				Y:           node.Y + node.Height/2,
				Width:       node.Width,
				Height:      node.Height,
				Color:       getStencilColor(node.Shape),
				IsStencil:   true,
				BorderColor: getStencilBorderColor(node.Shape),
			}
		} else {
			shape = MiroShape{
				Shape:   convertShape(node.Shape),
				Content: node.Label,
				X:       node.X + node.Width/2, // Miro uses center position
				Y:       node.Y + node.Height/2,
				Width:   node.Width,
				Height:  node.Height,
				Color:   getShapeColor(node.Shape),
			}
		}

		if node.Color != "" {
			shape.Color = node.Color
		}

		nodeToIndex[id] = len(output.Shapes)
		output.Shapes = append(output.Shapes, shape)
	}

	// Convert edges to connectors
	for _, edge := range diagram.Edges {
		startIdx, ok1 := nodeToIndex[edge.FromID]
		endIdx, ok2 := nodeToIndex[edge.ToID]

		if !ok1 || !ok2 {
			continue
		}

		connector := MiroConnector{
			StartItemIndex: startIdx,
			EndItemIndex:   endIdx,
			Caption:        edge.Label,
			Style:          "elbowed",
			StartCap:       convertArrowType(edge.StartCap),
			EndCap:         convertArrowType(edge.EndCap),
		}

		output.Connectors = append(output.Connectors, connector)
	}

	// Convert subgraphs to frames
	for _, sg := range diagram.SubGraphs {
		if len(sg.NodeIDs) == 0 {
			continue
		}

		// Calculate bounding box of nodes in subgraph
		minX, minY := float64(1e9), float64(1e9)
		maxX, maxY := float64(-1e9), float64(-1e9)

		for _, nodeID := range sg.NodeIDs {
			node := diagram.Nodes[nodeID]
			if node == nil {
				continue
			}

			if node.X < minX {
				minX = node.X
			}
			if node.Y < minY {
				minY = node.Y
			}
			if node.X+node.Width > maxX {
				maxX = node.X + node.Width
			}
			if node.Y+node.Height > maxY {
				maxY = node.Y + node.Height
			}
		}

		padding := 40.0
		frame := MiroFrame{
			Title:  sg.Label,
			X:      minX - padding + (maxX-minX+2*padding)/2,
			Y:      minY - padding - 30 + (maxY-minY+2*padding+30)/2, // Extra space for title
			Width:  maxX - minX + 2*padding,
			Height: maxY - minY + 2*padding + 30,
			Color:  "#F5F5F5",
		}

		output.Frames = append(output.Frames, frame)
	}

	return output
}

// convertShape maps internal shape to Miro shape name.
func convertShape(shape NodeShape) string {
	switch shape {
	case ShapeRectangle:
		return "rectangle"
	case ShapeRoundedRectangle:
		return "round_rectangle"
	case ShapeDiamond:
		return "rhombus"
	case ShapeCircle:
		return "circle"
	case ShapeStadium:
		return "pill"
	case ShapeCylinder:
		return "can"
	case ShapeParallelogram:
		return "parallelogram"
	case ShapeHexagon:
		return "hexagon"
	case ShapeTrapezoid:
		return "trapezoid"
	default:
		return "rectangle"
	}
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

// convertShapeToStencil maps internal shape to Miro flowchart stencil shape name.
// These shapes require the v2-experimental API endpoint.
func convertShapeToStencil(shape NodeShape) string {
	switch shape {
	case ShapeCircle:
		return "flow_chart_terminator" // Stadium/pill shape for Start/End
	case ShapeDiamond:
		return "flow_chart_decision" // Diamond for decisions
	case ShapeRectangle:
		return "flow_chart_process" // Rectangle for process steps
	case ShapeRoundedRectangle:
		return "flow_chart_process" // Also process
	case ShapeStadium:
		return "flow_chart_terminator" // Stadium maps to terminator
	case ShapeParallelogram:
		return "flow_chart_input_output" // Parallelogram for I/O
	case ShapeHexagon:
		return "flow_chart_preparation" // Hexagon for preparation
	case ShapeCylinder:
		return "flow_chart_database" // Cylinder for database
	case ShapeTrapezoid:
		return "flow_chart_manual_operation" // Trapezoid for manual operation
	default:
		return "flow_chart_process" // Default to process
	}
}

// getStencilColor returns professional fill colors for flowchart stencil shapes.
// Uses a cohesive color palette that works well with Miro's visual style.
func getStencilColor(shape NodeShape) string {
	switch shape {
	case ShapeCircle, ShapeStadium:
		return "#C8E6C9" // Light green for Start/End (terminator)
	case ShapeDiamond:
		return "#FFF9C4" // Light yellow for decisions
	case ShapeRectangle, ShapeRoundedRectangle:
		return "#BBDEFB" // Light blue for process steps
	case ShapeParallelogram:
		return "#E1BEE7" // Light purple for I/O
	case ShapeHexagon:
		return "#FFE0B2" // Light orange for preparation
	case ShapeCylinder:
		return "#B3E5FC" // Cyan for database
	case ShapeTrapezoid:
		return "#FFCCBC" // Light coral for manual operation
	default:
		return "#E3F2FD" // Default light blue
	}
}

// getStencilBorderColor returns matching border colors for stencil shapes.
// Darker variants of the fill colors for visual definition.
func getStencilBorderColor(shape NodeShape) string {
	switch shape {
	case ShapeCircle, ShapeStadium:
		return "#4CAF50" // Green border for terminator
	case ShapeDiamond:
		return "#FFC107" // Amber border for decisions
	case ShapeRectangle, ShapeRoundedRectangle:
		return "#2196F3" // Blue border for process
	case ShapeParallelogram:
		return "#9C27B0" // Purple border for I/O
	case ShapeHexagon:
		return "#FF9800" // Orange border for preparation
	case ShapeCylinder:
		return "#00BCD4" // Cyan border for database
	case ShapeTrapezoid:
		return "#FF5722" // Deep orange border for manual operation
	default:
		return "#1976D2" // Default blue border
	}
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

// ConvertSequenceToMiro converts a sequence diagram to Miro items.
// Creates: participant boxes, lifeline anchors, and message connectors.
func ConvertSequenceToMiro(diagram *Diagram) *MiroOutput {
	output := &MiroOutput{
		Shapes:     make([]MiroShape, 0),
		Connectors: make([]MiroConnector, 0),
		Frames:     make([]MiroFrame, 0),
	}

	// Collect and sort participants by their X position (order)
	type participantInfo struct {
		id    string
		node  *Node
		index int // Index in output.Shapes for this participant
	}
	participants := make([]participantInfo, 0, len(diagram.Nodes))

	for id, node := range diagram.Nodes {
		participants = append(participants, participantInfo{
			id:   id,
			node: node,
		})
	}

	// Sort by X position (which was set based on Order in parser)
	for i := 0; i < len(participants); i++ {
		for j := i + 1; j < len(participants); j++ {
			if participants[i].node.X > participants[j].node.X {
				participants[i], participants[j] = participants[j], participants[i]
			}
		}
	}

	// Map participant ID to their center X coordinate and shape index
	participantCenterX := make(map[string]float64)
	participantShapeIndex := make(map[string]int)

	// Create participant header boxes
	for i, p := range participants {
		shape := MiroShape{
			Shape:   "rectangle",
			Content: p.node.Label,
			X:       p.node.X + p.node.Width/2,
			Y:       p.node.Y + p.node.Height/2,
			Width:   p.node.Width,
			Height:  p.node.Height,
			Color:   "#E3F2FD", // Light blue for participants
		}

		if p.node.Shape == ShapeCircle {
			// Actor represented as circle
			shape.Shape = "circle"
			shape.Color = "#FFF9C4" // Light yellow for actors
		}

		participantCenterX[p.id] = p.node.X + p.node.Width/2
		participantShapeIndex[p.id] = len(output.Shapes)
		participants[i].index = len(output.Shapes)
		output.Shapes = append(output.Shapes, shape)
	}

	// Create lifeline shapes (thin vertical rectangles below each participant)
	lifelineHeight := diagram.Height - seqParticipantHeight - 30
	if lifelineHeight < 50 {
		lifelineHeight = 100 // Minimum lifeline height
	}

	for _, p := range participants {
		lifeline := MiroShape{
			Shape:   "rectangle",
			Content: "", // No text on lifeline
			X:       participantCenterX[p.id],
			Y:       p.node.Y + p.node.Height + lifelineHeight/2 + 10,
			Width:   seqLifelineWidth,
			Height:  lifelineHeight,
			Color:   seqLifelineColor, // Visible blue lifeline
		}
		output.Shapes = append(output.Shapes, lifeline)
	}

	// Create anchor points and connectors for each message
	// We need small anchor shapes at each end of each message for connectors
	for _, edge := range diagram.Edges {
		fromX, fromExists := participantCenterX[edge.FromID]
		toX, toExists := participantCenterX[edge.ToID]

		if !fromExists || !toExists {
			continue
		}

		// Create anchor shape at the "from" position
		// Anchors are small circles that blend with lifeline color
		fromAnchorIdx := len(output.Shapes)
		fromAnchor := MiroShape{
			Shape:   "circle",
			Content: "",
			X:       fromX,
			Y:       edge.Y,
			Width:   seqAnchorSize,
			Height:  seqAnchorSize,
			Color:   seqAnchorColor, // Match lifeline so it blends in
		}
		output.Shapes = append(output.Shapes, fromAnchor)

		// Create anchor shape at the "to" position
		toAnchorIdx := len(output.Shapes)
		toAnchor := MiroShape{
			Shape:   "circle",
			Content: "",
			X:       toX,
			Y:       edge.Y,
			Width:   seqAnchorSize,
			Height:  seqAnchorSize,
			Color:   seqAnchorColor,
		}
		output.Shapes = append(output.Shapes, toAnchor)

		// Create connector between the anchors
		connector := MiroConnector{
			StartItemIndex: fromAnchorIdx,
			EndItemIndex:   toAnchorIdx,
			Caption:        edge.Label,
			Style:          "straight", // Sequence messages are straight horizontal lines
			StartCap:       convertArrowType(edge.StartCap),
			EndCap:         convertArrowType(edge.EndCap),
		}

		// Handle edge styles
		if edge.Style == EdgeDotted {
			connector.Style = "straight" // Miro doesn't have dotted style, keep straight
		}

		output.Connectors = append(output.Connectors, connector)
	}

	return output
}
