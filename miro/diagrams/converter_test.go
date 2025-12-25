package diagrams

import (
	"testing"
)

// =============================================================================
// Flowchart Converter Tests
// =============================================================================

func TestConvertFlowchartToMiro_SimpleNodes(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)
	diagram.AddNode(&Node{
		ID:     "A",
		Label:  "Start",
		Shape:  ShapeRectangle,
		X:      0,
		Y:      0,
		Width:  100,
		Height: 50,
	})
	diagram.AddNode(&Node{
		ID:     "B",
		Label:  "End",
		Shape:  ShapeCircle,
		X:      0,
		Y:      100,
		Width:  100,
		Height: 50,
	})
	diagram.AddEdge(&Edge{
		FromID:   "A",
		ToID:     "B",
		Label:    "next",
		EndCap:   ArrowNormal,
		StartCap: ArrowNone,
	})

	output := convertFlowchartToMiro(diagram)

	if len(output.Shapes) != 2 {
		t.Errorf("expected 2 shapes, got %d", len(output.Shapes))
	}
	if len(output.Connectors) != 1 {
		t.Errorf("expected 1 connector, got %d", len(output.Connectors))
	}

	// Check connector caption
	if output.Connectors[0].Caption != "next" {
		t.Errorf("expected caption 'next', got '%s'", output.Connectors[0].Caption)
	}
}

func TestConvertFlowchartToMiro_WithSubgraph(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)
	diagram.AddNode(&Node{
		ID:     "A",
		Label:  "Node A",
		Shape:  ShapeRectangle,
		X:      0,
		Y:      0,
		Width:  100,
		Height: 50,
	})
	diagram.AddNode(&Node{
		ID:     "B",
		Label:  "Node B",
		Shape:  ShapeRectangle,
		X:      0,
		Y:      100,
		Width:  100,
		Height: 50,
	})
	diagram.SubGraphs["Group1"] = &SubGraph{
		ID:      "Group1",
		Label:   "My Group",
		NodeIDs: []string{"A", "B"},
	}

	output := convertFlowchartToMiro(diagram)

	if len(output.Frames) != 1 {
		t.Errorf("expected 1 frame, got %d", len(output.Frames))
	}
	if output.Frames[0].Title != "My Group" {
		t.Errorf("expected frame title 'My Group', got '%s'", output.Frames[0].Title)
	}
}

func TestConvertFlowchartToMiro_EdgeWithMissingNode(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)
	diagram.AddNode(&Node{
		ID:    "A",
		Label: "Node A",
	})
	// Edge to non-existent node
	diagram.AddEdge(&Edge{
		FromID: "A",
		ToID:   "X", // Does not exist
	})

	output := convertFlowchartToMiro(diagram)

	// Edge should be skipped
	if len(output.Connectors) != 0 {
		t.Errorf("expected 0 connectors (edge to missing node), got %d", len(output.Connectors))
	}
}

func TestConvertFlowchartToMiro_EmptySubgraph(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)
	diagram.AddNode(&Node{
		ID:    "A",
		Label: "Node A",
	})
	diagram.SubGraphs["Empty"] = &SubGraph{
		ID:      "Empty",
		Label:   "Empty Group",
		NodeIDs: []string{}, // No nodes
	}

	output := convertFlowchartToMiro(diagram)

	// Empty subgraph should be skipped
	if len(output.Frames) != 0 {
		t.Errorf("expected 0 frames (empty subgraph), got %d", len(output.Frames))
	}
}

func TestConvertFlowchartToMiro_SubgraphWithMissingNode(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)
	diagram.AddNode(&Node{
		ID:     "A",
		Label:  "Node A",
		X:      0,
		Y:      0,
		Width:  100,
		Height: 50,
	})
	diagram.SubGraphs["Group"] = &SubGraph{
		ID:      "Group",
		Label:   "Group",
		NodeIDs: []string{"A", "X"}, // X doesn't exist
	}

	output := convertFlowchartToMiro(diagram)

	// Frame should still be created for node A
	if len(output.Frames) != 1 {
		t.Errorf("expected 1 frame, got %d", len(output.Frames))
	}
}

func TestConvertFlowchartToMiro_NodeWithCustomColor(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)
	diagram.AddNode(&Node{
		ID:     "A",
		Label:  "Custom",
		Shape:  ShapeRectangle,
		Color:  "#FF0000",
		X:      0,
		Y:      0,
		Width:  100,
		Height: 50,
	})

	output := convertFlowchartToMiro(diagram)

	if len(output.Shapes) != 1 {
		t.Fatalf("expected 1 shape, got %d", len(output.Shapes))
	}
	if output.Shapes[0].Color != "#FF0000" {
		t.Errorf("expected custom color '#FF0000', got '%s'", output.Shapes[0].Color)
	}
}

// =============================================================================
// Shape Conversion Tests
// =============================================================================

func TestConvertShape(t *testing.T) {
	tests := []struct {
		shape    NodeShape
		expected string
	}{
		{ShapeRectangle, "rectangle"},
		{ShapeRoundedRectangle, "round_rectangle"},
		{ShapeDiamond, "rhombus"},
		{ShapeCircle, "circle"},
		{ShapeStadium, "pill"},
		{ShapeCylinder, "can"},
		{ShapeParallelogram, "parallelogram"},
		{ShapeHexagon, "hexagon"},
		{ShapeTrapezoid, "trapezoid"},
		{NodeShape("unknown"), "rectangle"}, // Unknown defaults to rectangle
	}

	for _, tt := range tests {
		t.Run(string(tt.shape), func(t *testing.T) {
			result := convertShape(tt.shape)
			if result != tt.expected {
				t.Errorf("convertShape(%s) = %s, want %s", tt.shape, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Arrow Type Conversion Tests
// =============================================================================

func TestConvertArrowType(t *testing.T) {
	tests := []struct {
		arrow    ArrowType
		expected string
	}{
		{ArrowNone, "none"},
		{ArrowNormal, "arrow"},
		{ArrowCircle, "filled_circle"},
		{ArrowCross, "diamond"},
		{ArrowType("unknown"), "none"}, // Unknown defaults to none
	}

	for _, tt := range tests {
		t.Run(string(tt.arrow), func(t *testing.T) {
			result := convertArrowType(tt.arrow)
			if result != tt.expected {
				t.Errorf("convertArrowType(%s) = %s, want %s", tt.arrow, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Shape Color Tests
// =============================================================================

func TestGetShapeColor(t *testing.T) {
	tests := []struct {
		shape         NodeShape
		expectedColor string
	}{
		{ShapeDiamond, "#FFE066"},          // Yellow for decisions
		{ShapeCircle, "#B8E986"},           // Green for start/end
		{ShapeStadium, "#B3E5FC"},          // Light blue for process
		{ShapeParallelogram, "#E1BEE7"},    // Light purple for I/O
		{ShapeHexagon, "#FFCCBC"},          // Light orange for preparation
		{ShapeRectangle, "#E3F2FD"},        // Default light blue
		{ShapeRoundedRectangle, "#E3F2FD"}, // Default
	}

	for _, tt := range tests {
		t.Run(string(tt.shape), func(t *testing.T) {
			result := getShapeColor(tt.shape)
			if result != tt.expectedColor {
				t.Errorf("getShapeColor(%s) = %s, want %s", tt.shape, result, tt.expectedColor)
			}
		})
	}
}

// =============================================================================
// ConvertToMiro Dispatch Tests
// =============================================================================

func TestConvertToMiro_Flowchart(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)
	diagram.AddNode(&Node{
		ID:     "A",
		Label:  "Test",
		Shape:  ShapeRectangle,
		Width:  100,
		Height: 50,
	})

	output := ConvertToMiro(diagram)

	if len(output.Shapes) != 1 {
		t.Errorf("expected 1 shape for flowchart, got %d", len(output.Shapes))
	}
}

func TestConvertToMiro_Sequence(t *testing.T) {
	diagram := NewDiagram(TypeSequence)
	diagram.AddNode(&Node{
		ID:     "A",
		Label:  "Alice",
		Shape:  ShapeRectangle,
		X:      0,
		Y:      50,
		Width:  120,
		Height: 50,
	})
	diagram.Height = 300

	output := ConvertToMiro(diagram)

	// Sequence diagrams create participant + lifeline
	if len(output.Shapes) < 2 {
		t.Errorf("expected at least 2 shapes for sequence (participant + lifeline), got %d", len(output.Shapes))
	}
}
