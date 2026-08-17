package diagrams

import (
	"strings"
	"testing"
)

// =============================================================================
// DiagramError.Error() Method Tests
// =============================================================================

func TestDiagramError_Error_BasicMessage(t *testing.T) {
	err := NewDiagramError("TEST", "test message")
	if err.Error() != "test message" {
		t.Errorf("expected 'test message', got '%s'", err.Error())
	}
}

func TestDiagramError_Error_WithLine(t *testing.T) {
	err := NewDiagramError("TEST", "test message").WithLine(5)
	expected := "test message (line 5)"
	if err.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, err.Error())
	}
}

func TestDiagramError_Error_WithSuggestion(t *testing.T) {
	err := NewDiagramError("TEST", "test message").WithSuggestion("try this")
	expected := "test message. try this"
	if err.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, err.Error())
	}
}

func TestDiagramError_Error_WithLineAndSuggestion(t *testing.T) {
	err := NewDiagramError("TEST", "test message").WithLine(10).WithSuggestion("fix it")
	expected := "test message (line 10). fix it"
	if err.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, err.Error())
	}
}

func TestDiagramError_Error_ZeroLine(t *testing.T) {
	err := NewDiagramError("TEST", "test message").WithLine(0)
	// Line 0 should not be displayed
	if strings.Contains(err.Error(), "line") {
		t.Errorf("expected no line info for line 0, got '%s'", err.Error())
	}
}

// =============================================================================
// WithInput Tests
// =============================================================================

func TestDiagramError_WithInput_Short(t *testing.T) {
	err := NewDiagramError("TEST", "test").WithInput("short input")
	if err.Input != "short input" {
		t.Errorf("expected 'short input', got '%s'", err.Input)
	}
}

func TestDiagramError_WithInput_Truncation(t *testing.T) {
	longInput := strings.Repeat("x", 100) // 100 characters
	err := NewDiagramError("TEST", "test").WithInput(longInput)
	if len(err.Input) != 50 {
		t.Errorf("expected truncated input of length 50, got %d", len(err.Input))
	}
	if !strings.HasSuffix(err.Input, "...") {
		t.Error("expected truncated input to end with '...'")
	}
}

func TestDiagramError_WithInput_ExactlyAtLimit(t *testing.T) {
	input50 := strings.Repeat("y", 50)
	err := NewDiagramError("TEST", "test").WithInput(input50)
	if err.Input != input50 {
		t.Errorf("expected input of exactly 50 chars to not be truncated")
	}
}

// =============================================================================
// Error Factory Function Tests
// =============================================================================

// assertFactoryError checks a factory-built DiagramError for the expected
// code, message substrings, and a non-empty suggestion.
func assertFactoryError(t *testing.T, err *DiagramError, wantCode string, wantInMessage ...string) {
	t.Helper()
	if err.Code != wantCode {
		t.Errorf("expected code '%s', got '%s'", wantCode, err.Code)
	}
	for _, want := range wantInMessage {
		if !strings.Contains(err.Message, want) {
			t.Errorf("expected message to contain '%s', got '%s'", want, err.Message)
		}
	}
	if err.Suggestion == "" {
		t.Error("expected suggestion to be set")
	}
}

func TestErrTooManyNodes(t *testing.T) {
	err := ErrTooManyNodes(150, 100)
	assertFactoryError(t, err, ErrCodeTooManyNodes, "150", "100")
}

func TestErrInvalidNodeShape(t *testing.T) {
	err := ErrInvalidNodeShape("<<<weird>>>")
	assertFactoryError(t, err, ErrCodeInvalidShape, "<<<weird>>>")
}

func TestErrInvalidEdge(t *testing.T) {
	err := ErrInvalidEdge("A", "B")
	assertFactoryError(t, err, ErrCodeInvalidEdge, "A", "B")
}

func TestParseDiagramSyntaxError(t *testing.T) {
	err := ParseDiagramSyntaxError(3, "invalid content here", "unexpected token")
	if err.Code != ErrCodeInvalidSyntax {
		t.Errorf("expected code '%s', got '%s'", ErrCodeInvalidSyntax, err.Code)
	}
	if err.Line != 3 {
		t.Errorf("expected line 3, got %d", err.Line)
	}
	if err.Input != "invalid content here" {
		t.Errorf("expected input preserved, got '%s'", err.Input)
	}
	if !strings.Contains(err.Message, "unexpected token") {
		t.Errorf("expected message to contain reason, got '%s'", err.Message)
	}
}

// =============================================================================
// DiagramTypeHint Tests
// =============================================================================

func TestDiagramTypeHint_FlowchartArrow(t *testing.T) {
	hint := DiagramTypeHint("A -> B")
	if !strings.Contains(hint, "-->") {
		t.Errorf("expected hint about '-->' for flowcharts, got '%s'", hint)
	}
}

func TestDiagramTypeHint_SequenceArrow(t *testing.T) {
	hint := DiagramTypeHint("sequenceDiagram\nA -> B: hello")
	if !strings.Contains(hint, "->>") {
		t.Errorf("expected hint about '->>' for sequence diagrams, got '%s'", hint)
	}
}

func TestDiagramTypeHint_MissingSequenceHeader(t *testing.T) {
	hint := DiagramTypeHint("participant Alice")
	if !strings.Contains(hint, "sequenceDiagram") {
		t.Errorf("expected hint about sequenceDiagram header, got '%s'", hint)
	}
}

func TestDiagramTypeHint_MissingFlowchartHeader(t *testing.T) {
	hint := DiagramTypeHint("subgraph group1\n  A --> B\nend")
	if !strings.Contains(hint, "flowchart") {
		t.Errorf("expected hint about flowchart header, got '%s'", hint)
	}
}

func TestDiagramTypeHint_ValidSyntax(t *testing.T) {
	hint := DiagramTypeHint("flowchart TB\n  A --> B")
	if hint != "" {
		t.Errorf("expected no hint for valid syntax, got '%s'", hint)
	}
}

func TestDiagramTypeHint_EmptyInput(t *testing.T) {
	hint := DiagramTypeHint("")
	if hint != "" {
		t.Errorf("expected no hint for empty input, got '%s'", hint)
	}
}

// =============================================================================
// ValidateDiagramInput Tests
// =============================================================================

// assertValidateFails checks that ValidateDiagramInput rejects input with a
// DiagramError carrying the expected code.
func assertValidateFails(t *testing.T, input, wantCode, desc string) {
	t.Helper()
	err := ValidateDiagramInput(input)
	if err == nil {
		t.Errorf("expected error for %s", desc)
	}
	diagErr, ok := err.(*DiagramError)
	if !ok {
		t.Fatalf("expected *DiagramError, got %T", err)
	}
	if diagErr.Code != wantCode {
		t.Errorf("expected code '%s', got '%s'", wantCode, diagErr.Code)
	}
}

func TestValidateDiagramInput_Empty(t *testing.T) {
	assertValidateFails(t, "", ErrCodeEmptyDiagram, "empty input")
}

func TestValidateDiagramInput_Whitespace(t *testing.T) {
	err := ValidateDiagramInput("   \n\t\n   ")
	if err == nil {
		t.Error("expected error for whitespace-only input")
	}
}

func TestValidateDiagramInput_ValidFlowchart(t *testing.T) {
	err := ValidateDiagramInput("flowchart TB\n  A --> B")
	if err != nil {
		t.Errorf("expected no error for valid flowchart, got %v", err)
	}
}

func TestValidateDiagramInput_ValidFlowchartLR(t *testing.T) {
	err := ValidateDiagramInput("flowchart LR\n  A --> B")
	if err != nil {
		t.Errorf("expected no error for valid flowchart LR, got %v", err)
	}
}

func TestValidateDiagramInput_ValidGraph(t *testing.T) {
	err := ValidateDiagramInput("graph TD\n  A --> B")
	if err != nil {
		t.Errorf("expected no error for valid graph, got %v", err)
	}
}

func TestValidateDiagramInput_ValidSequence(t *testing.T) {
	err := ValidateDiagramInput("sequenceDiagram\n  A->>B: Hello")
	if err != nil {
		t.Errorf("expected no error for valid sequence diagram, got %v", err)
	}
}

func TestValidateDiagramInput_CaseInsensitive(t *testing.T) {
	err := ValidateDiagramInput("FLOWCHART tb\n  A --> B")
	if err != nil {
		t.Errorf("expected no error for uppercase header, got %v", err)
	}
}

func TestValidateDiagramInput_WithComments(t *testing.T) {
	input := `%% This is a comment
%% Another comment
flowchart TB
  A --> B`
	err := ValidateDiagramInput(input)
	if err != nil {
		t.Errorf("expected no error when comments precede header, got %v", err)
	}
}

func TestValidateDiagramInput_MissingHeader(t *testing.T) {
	assertValidateFails(t, "A --> B\nB --> C", ErrCodeMissingHeader, "missing header")
}

func TestValidateDiagramInput_WithHint(t *testing.T) {
	// Input has a hint-triggering pattern (participant without header)
	err := ValidateDiagramInput("participant Alice\nAlice->>Bob: Hello")
	if err == nil {
		t.Fatal("expected error")
	}
	diagErr, ok := err.(*DiagramError)
	if !ok {
		t.Fatalf("expected *DiagramError, got %T", err)
	}
	// Should have a hint appended to suggestion
	if !strings.Contains(diagErr.Suggestion, "sequenceDiagram") {
		t.Errorf("expected suggestion to contain sequenceDiagram hint, got '%s'", diagErr.Suggestion)
	}
}

// =============================================================================
// Predefined Error Variables Tests
// =============================================================================

func TestErrNoNodes_Fields(t *testing.T) {
	if ErrNoNodes.Code != ErrCodeNoNodes {
		t.Errorf("expected code '%s', got '%s'", ErrCodeNoNodes, ErrNoNodes.Code)
	}
	if ErrNoNodes.Suggestion == "" {
		t.Error("expected suggestion to be set")
	}
}

func TestErrEmptyDiagram_Fields(t *testing.T) {
	if ErrEmptyDiagram.Code != ErrCodeEmptyDiagram {
		t.Errorf("expected code '%s', got '%s'", ErrCodeEmptyDiagram, ErrEmptyDiagram.Code)
	}
	if ErrEmptyDiagram.Suggestion == "" {
		t.Error("expected suggestion to be set")
	}
}

func TestErrMissingFlowchartHeader_Fields(t *testing.T) {
	if ErrMissingFlowchartHeader.Code != ErrCodeMissingHeader {
		t.Errorf("expected code '%s', got '%s'", ErrCodeMissingHeader, ErrMissingFlowchartHeader.Code)
	}
}

func TestErrMissingSequenceHeader_Fields(t *testing.T) {
	if ErrMissingSequenceHeader.Code != ErrCodeMissingHeader {
		t.Errorf("expected code '%s', got '%s'", ErrCodeMissingHeader, ErrMissingSequenceHeader.Code)
	}
}

func TestErrNoParticipants_Fields(t *testing.T) {
	if ErrNoParticipants.Code != ErrCodeNoNodes {
		t.Errorf("expected code '%s', got '%s'", ErrCodeNoNodes, ErrNoParticipants.Code)
	}
}
