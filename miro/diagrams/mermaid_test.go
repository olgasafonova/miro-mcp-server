package diagrams

import (
	"testing"
)

func TestParseMermaid_SimpleFlowchart(t *testing.T) {
	input := `flowchart TB
    A[Start] --> B[Process]
    B --> C[End]`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if diagram.Direction != TopToBottom {
		t.Errorf("Expected direction TB, got %s", diagram.Direction)
	}

	if len(diagram.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(diagram.Nodes))
	}

	if len(diagram.Edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(diagram.Edges))
	}

	// Check node labels
	if node := diagram.Nodes["A"]; node == nil || node.Label != "Start" {
		t.Errorf("Node A should have label 'Start'")
	}
	if node := diagram.Nodes["B"]; node == nil || node.Label != "Process" {
		t.Errorf("Node B should have label 'Process'")
	}
	if node := diagram.Nodes["C"]; node == nil || node.Label != "End" {
		t.Errorf("Node C should have label 'End'")
	}
}

func TestParseMermaid_LeftToRight(t *testing.T) {
	input := `flowchart LR
    A --> B --> C`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if diagram.Direction != LeftToRight {
		t.Errorf("Expected direction LR, got %s", diagram.Direction)
	}
}

func TestParseMermaid_DecisionDiamond(t *testing.T) {
	input := `flowchart TB
    A[Start] --> B{Decision}
    B -->|Yes| C[Yes Path]
    B -->|No| D[No Path]`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(diagram.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(diagram.Nodes))
	}

	// Check diamond shape
	if node := diagram.Nodes["B"]; node == nil || node.Shape != ShapeDiamond {
		t.Errorf("Node B should be a diamond shape")
	}
}

func TestParseMermaid_CircleShape(t *testing.T) {
	input := `flowchart TB
    A((Circle Node))`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if node := diagram.Nodes["A"]; node == nil || node.Shape != ShapeCircle {
		t.Errorf("Node A should be a circle shape, got %v", diagram.Nodes["A"])
	}
}

func TestParseMermaid_StadiumShape(t *testing.T) {
	input := `flowchart TB
    A(Stadium Shape)`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if node := diagram.Nodes["A"]; node == nil || node.Shape != ShapeStadium {
		t.Errorf("Node A should be a stadium shape")
	}
}

func TestParseMermaid_HexagonShape(t *testing.T) {
	input := `flowchart TB
    A{{Hexagon}}`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if node := diagram.Nodes["A"]; node == nil || node.Shape != ShapeHexagon {
		t.Errorf("Node A should be a hexagon shape")
	}
}

func TestParseMermaid_ChainedNodes(t *testing.T) {
	input := `flowchart LR
    A --> B --> C --> D`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(diagram.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(diagram.Nodes))
	}

	if len(diagram.Edges) != 3 {
		t.Errorf("Expected 3 edges, got %d", len(diagram.Edges))
	}
}

func TestParseMermaid_GraphKeyword(t *testing.T) {
	input := `graph TD
    A --> B`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if diagram.Direction != TopToBottom {
		t.Errorf("Expected direction TB (TD), got %s", diagram.Direction)
	}
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

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(diagram.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(diagram.Nodes))
	}
}

func TestParseMermaid_Subgraph(t *testing.T) {
	input := `flowchart TB
    subgraph Group1
        A --> B
    end
    C --> A`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

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
