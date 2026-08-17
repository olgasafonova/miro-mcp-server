package diagrams

import (
	"testing"
)

// =============================================================================
// Shared parse/assert helpers
// =============================================================================

// mustParseMermaid parses input and fails the test immediately on error.
func mustParseMermaid(t *testing.T, input string) *Diagram {
	t.Helper()
	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	return diagram
}

// assertDirection checks the diagram direction.
func assertDirection(t *testing.T, diagram *Diagram, want Direction) {
	t.Helper()
	if diagram.Direction != want {
		t.Errorf("Expected direction %s, got %s", want, diagram.Direction)
	}
}

// assertNodeCount checks the number of nodes in the diagram.
func assertNodeCount(t *testing.T, diagram *Diagram, want int) {
	t.Helper()
	if len(diagram.Nodes) != want {
		t.Errorf("Expected %d nodes, got %d", want, len(diagram.Nodes))
	}
}

// assertEdgeCount checks the number of edges in the diagram.
func assertEdgeCount(t *testing.T, diagram *Diagram, want int) {
	t.Helper()
	if len(diagram.Edges) != want {
		t.Errorf("Expected %d edges, got %d", want, len(diagram.Edges))
	}
}

// assertNodeLabel checks that a node exists and carries the wanted label.
func assertNodeLabel(t *testing.T, diagram *Diagram, id, label string) {
	t.Helper()
	if node := diagram.Nodes[id]; node == nil || node.Label != label {
		t.Errorf("Node %s should have label '%s'", id, label)
	}
}

// assertNodeShape checks that a node exists and has the wanted shape.
func assertNodeShape(t *testing.T, diagram *Diagram, id string, shape NodeShape) {
	t.Helper()
	if node := diagram.Nodes[id]; node == nil || node.Shape != shape {
		t.Errorf("Node %s should have shape %s, got %v", id, shape, diagram.Nodes[id])
	}
}

// =============================================================================
// Flowchart Parsing Tests
// =============================================================================

func TestParseMermaid_SimpleFlowchart(t *testing.T) {
	input := `flowchart TB
    A[Start] --> B[Process]
    B --> C[End]`

	diagram := mustParseMermaid(t, input)

	assertDirection(t, diagram, TopToBottom)
	assertNodeCount(t, diagram, 3)
	assertEdgeCount(t, diagram, 2)

	// Check node labels
	assertNodeLabel(t, diagram, "A", "Start")
	assertNodeLabel(t, diagram, "B", "Process")
	assertNodeLabel(t, diagram, "C", "End")
}

func TestParseMermaid_LeftToRight(t *testing.T) {
	input := `flowchart LR
    A --> B --> C`

	diagram := mustParseMermaid(t, input)

	assertDirection(t, diagram, LeftToRight)
}

func TestParseMermaid_DecisionDiamond(t *testing.T) {
	input := `flowchart TB
    A[Start] --> B{Decision}
    B -->|Yes| C[Yes Path]
    B -->|No| D[No Path]`

	diagram := mustParseMermaid(t, input)

	assertNodeCount(t, diagram, 4)
	assertNodeShape(t, diagram, "B", ShapeDiamond)
}

func TestParseMermaid_CircleShape(t *testing.T) {
	input := `flowchart TB
    A((Circle Node))`

	diagram := mustParseMermaid(t, input)

	assertNodeShape(t, diagram, "A", ShapeCircle)
}

func TestParseMermaid_StadiumShape(t *testing.T) {
	input := `flowchart TB
    A(Stadium Shape)`

	diagram := mustParseMermaid(t, input)

	assertNodeShape(t, diagram, "A", ShapeStadium)
}

func TestParseMermaid_HexagonShape(t *testing.T) {
	input := `flowchart TB
    A{{Hexagon}}`

	diagram := mustParseMermaid(t, input)

	assertNodeShape(t, diagram, "A", ShapeHexagon)
}

func TestParseMermaid_ChainedNodes(t *testing.T) {
	input := `flowchart LR
    A --> B --> C --> D`

	diagram := mustParseMermaid(t, input)

	assertNodeCount(t, diagram, 4)
	assertEdgeCount(t, diagram, 3)
}

func TestParseMermaid_GraphKeyword(t *testing.T) {
	input := `graph TD
    A --> B`

	diagram := mustParseMermaid(t, input)

	assertDirection(t, diagram, TopToBottom)
}

func TestParseMermaid_NoNodes(t *testing.T) {
	input := `flowchart TB`

	_, err := ParseMermaid(input)
	if err == nil {
		t.Error("Expected error for diagram with no nodes")
	}
}

func TestParseMermaid_Comments(t *testing.T) {
	input := `flowchart TB
    %% This is a comment
    A --> B
    %% Another comment`

	diagram := mustParseMermaid(t, input)

	assertNodeCount(t, diagram, 2)
}

func TestParseMermaid_Subgraph(t *testing.T) {
	input := `flowchart TB
    subgraph Group1
        A --> B
    end
    C --> A`

	diagram := mustParseMermaid(t, input)

	if len(diagram.SubGraphs) != 1 {
		t.Errorf("Expected 1 subgraph, got %d", len(diagram.SubGraphs))
	}

	sg := diagram.SubGraphs["Group1"]
	if sg == nil {
		t.Fatal("Subgraph Group1 not found")
	}

	if len(sg.NodeIDs) != 2 {
		t.Errorf("Expected 2 nodes in subgraph, got %d", len(sg.NodeIDs))
	}
}

func TestExtractNode(t *testing.T) {
	parser := NewMermaidParser()
	tests := []struct {
		input         string
		expectedID    string
		expectedLabel string
		expectedShape NodeShape
	}{
		{"A", "A", "A", ShapeRectangle},
		{"A[Text]", "A", "Text", ShapeRectangle},
		{"B{Decision}", "B", "Decision", ShapeDiamond},
		{"C((Circle))", "C", "Circle", ShapeCircle},
		{"D(Rounded)", "D", "Rounded", ShapeStadium},
		{"E{{Hexagon}}", "E", "Hexagon", ShapeHexagon},
		{"Node1[Complex Label]", "Node1", "Complex Label", ShapeRectangle},
	}

	for _, tt := range tests {
		id, label, shape := parser.extractNode(tt.input)
		if id != tt.expectedID {
			t.Errorf("Input %s: expected ID %s, got %s", tt.input, tt.expectedID, id)
		}
		if label != tt.expectedLabel {
			t.Errorf("Input %s: expected label %s, got %s", tt.input, tt.expectedLabel, label)
		}
		if shape != tt.expectedShape {
			t.Errorf("Input %s: expected shape %s, got %s", tt.input, tt.expectedShape, shape)
		}
	}
}
