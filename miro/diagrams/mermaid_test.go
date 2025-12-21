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

// Sequence Diagram Tests

func TestParseMermaid_SequenceDiagram(t *testing.T) {
	input := `sequenceDiagram
    Alice->>Bob: Hello Bob!
    Bob-->>Alice: Hi Alice!`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if diagram.Type != TypeSequence {
		t.Errorf("Expected type sequence, got %s", diagram.Type)
	}

	if len(diagram.Nodes) != 2 {
		t.Errorf("Expected 2 participants, got %d", len(diagram.Nodes))
	}

	if len(diagram.Edges) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(diagram.Edges))
	}

	// Check participants
	if node := diagram.Nodes["Alice"]; node == nil {
		t.Error("Participant Alice should exist")
	}
	if node := diagram.Nodes["Bob"]; node == nil {
		t.Error("Participant Bob should exist")
	}
}

func TestParseMermaid_SequenceWithParticipants(t *testing.T) {
	input := `sequenceDiagram
    participant A as Alice
    participant B as Bob
    A->>B: Hello!`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(diagram.Nodes) != 2 {
		t.Errorf("Expected 2 participants, got %d", len(diagram.Nodes))
	}

	// Check that alias labels are used
	if node := diagram.Nodes["A"]; node == nil || node.Label != "Alice" {
		t.Errorf("Participant A should have label 'Alice', got %v", diagram.Nodes["A"])
	}
	if node := diagram.Nodes["B"]; node == nil || node.Label != "Bob" {
		t.Errorf("Participant B should have label 'Bob', got %v", diagram.Nodes["B"])
	}
}

func TestParseMermaid_SequenceActors(t *testing.T) {
	input := `sequenceDiagram
    actor User
    participant System
    User->>System: Request`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Actors should have circle shape
	if node := diagram.Nodes["User"]; node == nil || node.Shape != ShapeCircle {
		t.Errorf("Actor User should be circle shape")
	}

	// Participants should have rectangle shape
	if node := diagram.Nodes["System"]; node == nil || node.Shape != ShapeRectangle {
		t.Errorf("Participant System should be rectangle shape")
	}
}

func TestParseMermaid_SequenceMessageTypes(t *testing.T) {
	input := `sequenceDiagram
    A->>B: Sync message
    A-->>B: Async message
    A-xB: Lost message`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(diagram.Edges) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(diagram.Edges))
	}

	// Check first edge is solid (sync)
	if diagram.Edges[0].Style != EdgeSolid {
		t.Errorf("First message should be solid style, got %s", diagram.Edges[0].Style)
	}

	// Check second edge is dotted (async)
	if diagram.Edges[1].Style != EdgeDotted {
		t.Errorf("Second message should be dotted style, got %s", diagram.Edges[1].Style)
	}
}

func TestParseMermaid_SequenceNotASequence(t *testing.T) {
	input := `flowchart TB
    A --> B`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if diagram.Type != TypeFlowchart {
		t.Errorf("Expected flowchart type, got %s", diagram.Type)
	}
}

func TestParseMermaid_SequenceNoHeader(t *testing.T) {
	input := `Alice->>Bob: Hello`

	// This should fall back to flowchart parser and fail
	_, err := ParseMermaid(input)
	if err == nil {
		t.Error("Expected error for sequence without header")
	}
}

func TestParseMermaid_SequenceWithLoop(t *testing.T) {
	input := `sequenceDiagram
    Alice->>Bob: Hello
    loop Every minute
        Bob->>Alice: Ping
    end`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Loop should be parsed without error, messages inside included
	if len(diagram.Edges) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(diagram.Edges))
	}
}

func TestIsSequenceDiagram(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"sequenceDiagram\nA->>B: msg", true},
		{"  sequenceDiagram  \nA->>B: msg", true},
		{"SEQUENCEDIAGRAM\nA->>B: msg", true},
		{"flowchart TB\nA-->B", false},
		{"graph LR\nA-->B", false},
		{"", false},
		{"%% comment\nsequenceDiagram\nA->>B: msg", true},
	}

	for _, tt := range tests {
		result := isSequenceDiagram(tt.input)
		if result != tt.expected {
			t.Errorf("isSequenceDiagram(%q) = %v, want %v", tt.input[:min(30, len(tt.input))], result, tt.expected)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
