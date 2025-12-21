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

// =============================================================================
// Sequence Diagram Converter Tests
// =============================================================================

func TestConvertSequenceToMiro_BasicOutput(t *testing.T) {
	input := `sequenceDiagram
    Alice->>Bob: Hello Bob!
    Bob-->>Alice: Hi Alice!`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	output := ConvertSequenceToMiro(diagram)

	// Should have: 2 participants + 2 lifelines + 4 anchors (2 per message)
	expectedShapes := 2 + 2 + 4
	if len(output.Shapes) != expectedShapes {
		t.Errorf("Expected %d shapes, got %d", expectedShapes, len(output.Shapes))
	}

	// Should have 2 connectors (one per message)
	if len(output.Connectors) != 2 {
		t.Errorf("Expected 2 connectors, got %d", len(output.Connectors))
	}
}

func TestConvertSequenceToMiro_ParticipantPositions(t *testing.T) {
	input := `sequenceDiagram
    participant A
    participant B
    participant C
    A->>B: msg`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	output := ConvertSequenceToMiro(diagram)

	// First 3 shapes should be participant boxes
	if len(output.Shapes) < 3 {
		t.Fatalf("Expected at least 3 shapes for participants")
	}

	// Participants should be arranged left to right (X increasing)
	prevX := -1000.0
	for i := 0; i < 3; i++ {
		if output.Shapes[i].X <= prevX {
			t.Errorf("Participant %d X position (%f) should be > previous (%f)", i, output.Shapes[i].X, prevX)
		}
		prevX = output.Shapes[i].X
	}

	// All participants should have same Y position
	y0 := output.Shapes[0].Y
	for i := 1; i < 3; i++ {
		if output.Shapes[i].Y != y0 {
			t.Errorf("All participants should have same Y, got %f and %f", y0, output.Shapes[i].Y)
		}
	}
}

func TestConvertSequenceToMiro_LifelineCreated(t *testing.T) {
	input := `sequenceDiagram
    participant A
    A->>A: self call`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	output := ConvertSequenceToMiro(diagram)

	// Should have: 1 participant + 1 lifeline + 2 anchors
	if len(output.Shapes) < 2 {
		t.Fatalf("Expected at least 2 shapes")
	}

	// Second shape should be the lifeline (thin rectangle)
	lifeline := output.Shapes[1]
	if lifeline.Width >= lifeline.Height {
		t.Errorf("Lifeline should be taller than wide, got w=%f h=%f", lifeline.Width, lifeline.Height)
	}
}

func TestConvertSequenceToMiro_MessageYPositions(t *testing.T) {
	input := `sequenceDiagram
    Alice->>Bob: First
    Bob->>Alice: Second
    Alice->>Bob: Third`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	output := ConvertSequenceToMiro(diagram)

	// Check that connectors have captions
	if len(output.Connectors) != 3 {
		t.Fatalf("Expected 3 connectors")
	}

	if output.Connectors[0].Caption != "First" {
		t.Errorf("First connector caption should be 'First', got '%s'", output.Connectors[0].Caption)
	}
	if output.Connectors[1].Caption != "Second" {
		t.Errorf("Second connector caption should be 'Second', got '%s'", output.Connectors[1].Caption)
	}
	if output.Connectors[2].Caption != "Third" {
		t.Errorf("Third connector caption should be 'Third', got '%s'", output.Connectors[2].Caption)
	}
}

func TestConvertSequenceToMiro_MessageAnchorYProgression(t *testing.T) {
	input := `sequenceDiagram
    A->>B: msg1
    B->>A: msg2
    A->>B: msg3`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	output := ConvertSequenceToMiro(diagram)

	// Shapes: 2 participants + 2 lifelines + 6 anchors (2 per message)
	// Anchors start at index 4
	if len(output.Shapes) < 10 {
		t.Fatalf("Expected at least 10 shapes, got %d", len(output.Shapes))
	}

	// Get Y positions of message anchors (every pair of anchors)
	// First message anchors at index 4,5
	// Second message anchors at index 6,7
	// Third message anchors at index 8,9
	msg1Y := output.Shapes[4].Y
	msg2Y := output.Shapes[6].Y
	msg3Y := output.Shapes[8].Y

	// Each message should be lower (higher Y) than the previous
	if msg2Y <= msg1Y {
		t.Errorf("Message 2 Y (%f) should be > message 1 Y (%f)", msg2Y, msg1Y)
	}
	if msg3Y <= msg2Y {
		t.Errorf("Message 3 Y (%f) should be > message 2 Y (%f)", msg3Y, msg2Y)
	}
}

func TestConvertSequenceToMiro_ActorShape(t *testing.T) {
	input := `sequenceDiagram
    actor User
    participant System
    User->>System: Request`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	output := ConvertSequenceToMiro(diagram)

	// First two shapes are participants
	// Actor should be circle, participant should be rectangle
	foundCircle := false
	foundRectangle := false

	for i := 0; i < 2; i++ {
		if output.Shapes[i].Shape == "circle" && output.Shapes[i].Content == "User" {
			foundCircle = true
		}
		if output.Shapes[i].Shape == "rectangle" && output.Shapes[i].Content == "System" {
			foundRectangle = true
		}
	}

	if !foundCircle {
		t.Error("Actor should be rendered as circle")
	}
	if !foundRectangle {
		t.Error("Participant should be rendered as rectangle")
	}
}

func TestConvertSequenceToMiro_ConnectorStyle(t *testing.T) {
	input := `sequenceDiagram
    A->>B: Sync
    A-->>B: Async`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	output := ConvertSequenceToMiro(diagram)

	if len(output.Connectors) != 2 {
		t.Fatalf("Expected 2 connectors")
	}

	// Both should have arrow end cap
	if output.Connectors[0].EndCap != "arrow" {
		t.Errorf("Sync message should have arrow end cap, got %s", output.Connectors[0].EndCap)
	}
	if output.Connectors[1].EndCap != "arrow" {
		t.Errorf("Async message should have arrow end cap, got %s", output.Connectors[1].EndCap)
	}

	// Both should be straight style
	if output.Connectors[0].Style != "straight" {
		t.Errorf("Message style should be straight, got %s", output.Connectors[0].Style)
	}
}

func TestConvertSequenceToMiro_DiagramDetection(t *testing.T) {
	// Test that ConvertToMiro correctly routes to sequence converter
	input := `sequenceDiagram
    A->>B: test`

	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Use the main ConvertToMiro function
	output := ConvertToMiro(diagram)

	// Should have sequence diagram structure (anchors, lifelines)
	// Not flowchart structure (simple node shapes)
	if len(output.Shapes) < 4 { // At least: 2 participants + 2 lifelines
		t.Errorf("Expected at least 4 shapes for sequence diagram, got %d", len(output.Shapes))
	}
}

func TestConvertSequenceToMiro_EmptyDiagram(t *testing.T) {
	// Create a minimal sequence diagram with no messages
	diagram := NewDiagram(TypeSequence)
	diagram.AddNode(&Node{
		ID:     "A",
		Label:  "A",
		Shape:  ShapeRectangle,
		X:      0,
		Y:      50,
		Width:  120,
		Height: 50,
	})

	output := ConvertSequenceToMiro(diagram)

	// Should have 1 participant + 1 lifeline
	if len(output.Shapes) != 2 {
		t.Errorf("Expected 2 shapes for single participant, got %d", len(output.Shapes))
	}

	// No connectors
	if len(output.Connectors) != 0 {
		t.Errorf("Expected 0 connectors, got %d", len(output.Connectors))
	}
}
