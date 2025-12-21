package diagrams

// MiroShape represents a shape to be created in Miro.
type MiroShape struct {
	Shape   string  // Miro shape type
	Content string  // Text content
	X       float64 // Center X position
	Y       float64 // Center Y position
	Width   float64
	Height  float64
	Color   string // Fill color (optional)
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
func ConvertToMiro(diagram *Diagram) *MiroOutput {
	output := &MiroOutput{
		Shapes:     make([]MiroShape, 0),
		Connectors: make([]MiroConnector, 0),
		Frames:     make([]MiroFrame, 0),
	}

	// Map node IDs to shape indices
	nodeToIndex := make(map[string]int)

	// Convert nodes to shapes
	for id, node := range diagram.Nodes {
		shape := MiroShape{
			Shape:   convertShape(node.Shape),
			Content: node.Label,
			X:       node.X + node.Width/2,  // Miro uses center position
			Y:       node.Y + node.Height/2,
			Width:   node.Width,
			Height:  node.Height,
			Color:   getShapeColor(node.Shape),
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
			X:      minX - padding + (maxX - minX + 2*padding) / 2,
			Y:      minY - padding - 30 + (maxY - minY + 2*padding + 30) / 2, // Extra space for title
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
