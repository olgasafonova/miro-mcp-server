package diagrams

import (
	"testing"
)

// =============================================================================
// Sequence-specific helpers
// =============================================================================

// mustParseSequenceInput parses sequence-diagram input via ParseMermaid and
// fails the test immediately on error.
func mustParseSequenceInput(t *testing.T, input string) *Diagram {
	t.Helper()
	diagram, err := ParseMermaid(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	return diagram
}

// convertSequence parses input and runs it through ConvertSequenceToMiro.
func convertSequence(t *testing.T, input string) *MiroOutput {
	t.Helper()
	return ConvertSequenceToMiro(mustParseSequenceInput(t, input))
}

// findShapeWithContent reports whether any of the shapes has the given Miro
// shape type and content.
func findShapeWithContent(shapes []MiroShape, shape, content string) bool {
	for _, s := range shapes {
		if s.Shape == shape && s.Content == content {
			return true
		}
	}
	return false
}

// assertIncreasingX checks that the shapes are arranged left to right
// (X strictly increasing).
func assertIncreasingX(t *testing.T, shapes []MiroShape) {
	t.Helper()
	prevX := -1000.0
	for i, s := range shapes {
		if s.X <= prevX {
			t.Errorf("Participant %d X position (%f) should be > previous (%f)", i, s.X, prevX)
		}
		prevX = s.X
	}
}

// assertSameY checks that all shapes share the Y position of the first one.
func assertSameY(t *testing.T, shapes []MiroShape) {
	t.Helper()
	y0 := shapes[0].Y
	for i := 1; i < len(shapes); i++ {
		if shapes[i].Y != y0 {
			t.Errorf("All participants should have same Y, got %f and %f", y0, shapes[i].Y)
		}
	}
}

// assertConnectorCaption checks the caption of the i-th connector.
func assertConnectorCaption(t *testing.T, connectors []MiroConnector, i int, want string) {
	t.Helper()
	if connectors[i].Caption != want {
		t.Errorf("Connector %d caption should be '%s', got '%s'", i, want, connectors[i].Caption)
	}
}

// =============================================================================
// Sequence Diagram Parsing Tests
// =============================================================================

func TestParseMermaid_SequenceDiagram(t *testing.T) {
	input := `sequenceDiagram
    Alice->>Bob: Hello Bob!
    Bob-->>Alice: Hi Alice!`

	diagram := mustParseSequenceInput(t, input)

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

	diagram := mustParseSequenceInput(t, input)

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

	diagram := mustParseSequenceInput(t, input)

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

	diagram := mustParseSequenceInput(t, input)

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

	diagram := mustParseSequenceInput(t, input)

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

	diagram := mustParseSequenceInput(t, input)

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

	output := convertSequence(t, input)

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

	output := convertSequence(t, input)

	// First 3 shapes should be participant boxes
	if len(output.Shapes) < 3 {
		t.Fatalf("Expected at least 3 shapes for participants")
	}

	// Participants should be arranged left to right (X increasing)
	assertIncreasingX(t, output.Shapes[:3])

	// All participants should have same Y position
	assertSameY(t, output.Shapes[:3])
}

func TestConvertSequenceToMiro_LifelineCreated(t *testing.T) {
	input := `sequenceDiagram
    participant A
    A->>A: self call`

	output := convertSequence(t, input)

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

	output := convertSequence(t, input)

	// Check that connectors have captions
	if len(output.Connectors) != 3 {
		t.Fatalf("Expected 3 connectors")
	}

	assertConnectorCaption(t, output.Connectors, 0, "First")
	assertConnectorCaption(t, output.Connectors, 1, "Second")
	assertConnectorCaption(t, output.Connectors, 2, "Third")
}

func TestConvertSequenceToMiro_MessageAnchorYProgression(t *testing.T) {
	input := `sequenceDiagram
    A->>B: msg1
    B->>A: msg2
    A->>B: msg3`

	output := convertSequence(t, input)

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

	output := convertSequence(t, input)

	// First two shapes are participants
	// Actor should be circle, participant should be rectangle
	if !findShapeWithContent(output.Shapes[:2], "circle", "User") {
		t.Error("Actor should be rendered as circle")
	}
	if !findShapeWithContent(output.Shapes[:2], "rectangle", "System") {
		t.Error("Participant should be rendered as rectangle")
	}
}

func TestConvertSequenceToMiro_ConnectorStyle(t *testing.T) {
	input := `sequenceDiagram
    A->>B: Sync
    A-->>B: Async`

	output := convertSequence(t, input)

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

	diagram := mustParseSequenceInput(t, input)

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
