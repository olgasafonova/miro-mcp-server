package diagrams

import (
	"testing"
)

// =============================================================================
// ParseSequence Function Tests
// =============================================================================

func TestParseSequence_Valid(t *testing.T) {
	input := `sequenceDiagram
    Alice->>Bob: Hello Bob
    Bob-->>Alice: Hi Alice`

	diagram, err := ParseSequence(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diagram == nil {
		t.Fatal("expected diagram, got nil")
	}
	if diagram.Type != TypeSequence {
		t.Errorf("expected TypeSequence, got %s", diagram.Type)
	}
	if len(diagram.Nodes) != 2 {
		t.Errorf("expected 2 participants, got %d", len(diagram.Nodes))
	}
}

func TestParseSequence_Invalid(t *testing.T) {
	input := "not a sequence diagram"
	_, err := ParseSequence(input)
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestParseSequence_Empty(t *testing.T) {
	_, err := ParseSequence("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}
