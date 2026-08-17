package prompts

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// assertSameStrings compares got against want element by element.
func assertSameStrings(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s = %v, want %v", label, got, want)
		return
	}
	for i, v := range got {
		if v != want[i] {
			t.Errorf("%s[%d] = %q, want %q", label, i, v, want[i])
		}
	}
}

// assertPromptContains checks that each wanted string appears in text.
func assertPromptContains(t *testing.T, text string, wants []string) {
	t.Helper()
	for _, check := range wants {
		if !strings.Contains(text, check) {
			t.Errorf("prompt should contain %q", check)
		}
	}
}

// assertPromptOmits checks that no unwanted string appears in text.
func assertPromptOmits(t *testing.T, text string, unwanted []string) {
	t.Helper()
	for _, check := range unwanted {
		if strings.Contains(text, check) {
			t.Errorf("prompt should NOT contain %q", check)
		}
	}
}

// promptText invokes a prompt handler and returns the text of its message.
func promptText(t *testing.T, handler func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error), name string, args map[string]string) string {
	t.Helper()

	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Name: name, Arguments: args},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("%s handler error = %v", name, err)
	}

	textContent := result.Messages[0].Content.(*mcp.TextContent)
	return textContent.Text
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Error("NewRegistry() returned nil")
	}
}

func TestSplitColumns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "default columns",
			input:    "To Do,In Progress,Review,Done",
			expected: []string{"To Do", "In Progress", "Review", "Done"},
		},
		{
			name:     "with spaces",
			input:    " To Do , In Progress , Done ",
			expected: []string{"To Do", "In Progress", "Done"},
		},
		{
			name:     "single column",
			input:    "Tasks",
			expected: []string{"Tasks"},
		},
		{
			name:     "empty strings filtered",
			input:    "A,,B,,C",
			expected: []string{"A", "B", "C"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitColumns(tt.input)
			assertSameStrings(t, fmt.Sprintf("splitColumns(%q)", tt.input), result, tt.expected)
		})
	}
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"\t\nworld\r\n", "world"},
		{"no-trim", "no-trim"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		result := trimSpace(tt.input)
		if result != tt.expected {
			t.Errorf("trimSpace(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGetColumnColor(t *testing.T) {
	// Test cycling through colors
	colors := []string{"gray", "blue", "yellow", "green", "pink", "orange", "cyan"}
	for i := 0; i < 14; i++ {
		expected := colors[i%len(colors)]
		result := getColumnColor(i)
		if result != expected {
			t.Errorf("getColumnColor(%d) = %q, want %q", i, result, expected)
		}
	}
}

func TestHandleSprintBoard(t *testing.T) {
	registry := NewRegistry()

	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "create-sprint-board",
			Arguments: map[string]string{
				"board_name":    "Sprint 42 Board",
				"sprint_number": "42",
			},
		},
	}

	result, err := registry.handleSprintBoard(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSprintBoard() error = %v", err)
	}

	if result == nil {
		t.Fatal("handleSprintBoard() returned nil result")
	}

	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Messages))
	}

	// Check that the prompt contains expected content
	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}

	if !strings.Contains(textContent.Text, "Sprint 42 Board") {
		t.Error("prompt should contain board name")
	}
	if !strings.Contains(textContent.Text, "Sprint 42 Planning") {
		t.Error("prompt should contain sprint number")
	}
	if !strings.Contains(textContent.Text, "Backlog") {
		t.Error("prompt should mention Backlog frame")
	}
}

func TestHandleSprintBoardDefaultSprintNumber(t *testing.T) {
	registry := NewRegistry()

	text := promptText(t, registry.handleSprintBoard, "create-sprint-board", map[string]string{
		"board_name": "My Sprint Board",
		// sprint_number not provided
	})

	if !strings.Contains(text, "Sprint N Planning") {
		t.Error("prompt should use default sprint number 'N'")
	}
}

func TestHandleRetrospective(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name         string
		args         map[string]string
		checkFor     []string
		checkAgainst []string
	}{
		{
			name: "new board",
			args: map[string]string{
				"team_name": "Platform Team",
			},
			checkFor: []string{
				"Platform Team Retrospective",
				"What Went Well",
				"What Could Improve",
				"Action Items",
				"miro_create_board",
			},
		},
		{
			name: "existing board",
			args: map[string]string{
				"board_id":  "abc123",
				"team_name": "Platform Team",
			},
			checkFor: []string{
				"Platform Team",
				"What Went Well",
				"board_id: abc123",
			},
			checkAgainst: []string{
				"miro_create_board", // should NOT create new board
			},
		},
		{
			name: "default team name",
			args: map[string]string{},
			checkFor: []string{
				"Team Retrospective",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := promptText(t, registry.handleRetrospective, "create-retrospective", tt.args)
			assertPromptContains(t, text, tt.checkFor)
			assertPromptOmits(t, text, tt.checkAgainst)
		})
	}
}

func TestHandleBrainstorm(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name     string
		args     map[string]string
		checkFor []string
	}{
		{
			name: "new board",
			args: map[string]string{
				"topic": "Product Ideas",
			},
			checkFor: []string{
				"Product Ideas",
				"Brainstorm:",
				"miro_create_board",
				"sticky notes",
			},
		},
		{
			name: "existing board",
			args: map[string]string{
				"topic":    "Feature Brainstorm",
				"board_id": "xyz789",
			},
			checkFor: []string{
				"Feature Brainstorm",
				"board_id: xyz789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := promptText(t, registry.handleBrainstorm, "create-brainstorm", tt.args)
			assertPromptContains(t, text, tt.checkFor)
		})
	}
}

func TestHandleStoryMap(t *testing.T) {
	registry := NewRegistry()

	text := promptText(t, registry.handleStoryMap, "create-story-map", map[string]string{
		"product_name": "MyApp",
	})

	assertPromptContains(t, text, []string{
		"MyApp",
		"Story Map",
		"Discovery",
		"Onboarding",
		"Core Usage",
		"MVP",
	})
}

func TestHandleKanban(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name         string
		args         map[string]string
		checkFor     []string
		expectedCols int
	}{
		{
			name: "default columns",
			args: map[string]string{
				"board_name": "My Kanban",
			},
			checkFor: []string{
				"My Kanban",
				"To Do",
				"In Progress",
				"Review",
				"Done",
			},
			expectedCols: 4,
		},
		{
			name: "custom columns",
			args: map[string]string{
				"board_name": "Custom Board",
				"columns":    "Backlog,Active,Testing,Released",
			},
			checkFor: []string{
				"Custom Board",
				"Backlog",
				"Active",
				"Testing",
				"Released",
			},
			expectedCols: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := promptText(t, registry.handleKanban, "create-kanban", tt.args)
			assertPromptContains(t, text, tt.checkFor)
		})
	}
}

func TestSplitString(t *testing.T) {
	tests := []struct {
		input    string
		sep      string
		expected []string
	}{
		{"a,b,c", ",", []string{"a", "b", "c"}},
		{"hello world", " ", []string{"hello", "world"}},
		{"single", ",", []string{"single"}},
		{"a::b::c", "::", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		result := splitString(tt.input, tt.sep)
		assertSameStrings(t, fmt.Sprintf("splitString(%q, %q)", tt.input, tt.sep), result, tt.expected)
	}
}
