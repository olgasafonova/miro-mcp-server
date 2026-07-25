package diagrams

import (
	"strconv"
	"strings"
	"testing"
)

// =============================================================================
// Error constructor / keyword helper tests
// =============================================================================

func TestErrTooManyLinesError(t *testing.T) {
	err := ErrTooManyLinesError(500)

	if err == nil {
		t.Fatal("expected a DiagramError")
	}
	if err.Code != ErrCodeTooManyLines {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeTooManyLines)
	}
	if !strings.Contains(err.Message, "500") {
		t.Errorf("Message = %q, want the actual line count in it", err.Message)
	}
	if !strings.Contains(err.Message, strconv.Itoa(MaxDiagramLines)) {
		t.Errorf("Message = %q, want the limit in it", err.Message)
	}
	if err.Suggestion == "" {
		t.Error("Suggestion should be set")
	}
}

func TestErrLineTooLongError(t *testing.T) {
	err := ErrLineTooLongError(42, 9000)

	if err == nil {
		t.Fatal("expected a DiagramError")
	}
	if err.Code != ErrCodeLineTooLong {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeLineTooLong)
	}
	if !strings.Contains(err.Message, "42") {
		t.Errorf("Message = %q, want the line number in it", err.Message)
	}
	if !strings.Contains(err.Message, "9000") {
		t.Errorf("Message = %q, want the length in it", err.Message)
	}
	if err.Line != 42 {
		t.Errorf("Line = %d, want 42", err.Line)
	}
	if err.Suggestion == "" {
		t.Error("Suggestion should be set")
	}
}

func TestIsReservedKeyword(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"bare flowchart", "flowchart", true},
		{"flowchart with direction", "flowchart TD", true},
		{"bare graph", "graph", true},
		{"graph with direction", "graph LR", true},
		{"uppercase is matched", "FLOWCHART TD", true},
		{"mixed case is matched", "Graph LR", true},
		{"a node definition is not reserved", "A[Start]", false},
		{"an edge is not reserved", "A --> B", false},
		{"empty line is not reserved", "", false},
		{"sequenceDiagram is not in this set", "sequenceDiagram", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReservedKeyword(tt.line); got != tt.want {
				t.Errorf("isReservedKeyword(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}
