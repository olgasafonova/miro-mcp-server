package prompts

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
			if len(result) != len(tt.expected) {
				t.Errorf("splitColumns(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitColumns(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
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

	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "create-sprint-board",
			Arguments: map[string]string{
				"board_name": "My Sprint Board",
				// sprint_number not provided
			},
		},
	}

	result, err := registry.handleSprintBoard(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSprintBoard() error = %v", err)
	}

	textContent := result.Messages[0].Content.(*mcp.TextContent)
	if !strings.Contains(textContent.Text, "Sprint N Planning") {
		t.Error("prompt should use default sprint number 'N'")
	}
}

func TestHandleRetrospective(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name       string
		args       map[string]string
		checkFor   []string
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
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "create-retrospective",
					Arguments: tt.args,
				},
			}

			result, err := registry.handleRetrospective(context.Background(), req)
			if err != nil {
				t.Fatalf("handleRetrospective() error = %v", err)
			}

			textContent := result.Messages[0].Content.(*mcp.TextContent)

			for _, check := range tt.checkFor {
				if !strings.Contains(textContent.Text, check) {
					t.Errorf("prompt should contain %q", check)
				}
			}

			for _, check := range tt.checkAgainst {
				if strings.Contains(textContent.Text, check) {
					t.Errorf("prompt should NOT contain %q", check)
				}
			}
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
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "create-brainstorm",
					Arguments: tt.args,
				},
			}

			result, err := registry.handleBrainstorm(context.Background(), req)
			if err != nil {
				t.Fatalf("handleBrainstorm() error = %v", err)
			}

			textContent := result.Messages[0].Content.(*mcp.TextContent)

			for _, check := range tt.checkFor {
				if !strings.Contains(textContent.Text, check) {
					t.Errorf("prompt should contain %q", check)
				}
			}
		})
	}
}

func TestHandleStoryMap(t *testing.T) {
	registry := NewRegistry()

	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "create-story-map",
			Arguments: map[string]string{
				"product_name": "MyApp",
			},
		},
	}

	result, err := registry.handleStoryMap(context.Background(), req)
	if err != nil {
		t.Fatalf("handleStoryMap() error = %v", err)
	}

	textContent := result.Messages[0].Content.(*mcp.TextContent)

	checkFor := []string{
		"MyApp",
		"Story Map",
		"Discovery",
		"Onboarding",
		"Core Usage",
		"MVP",
	}

	for _, check := range checkFor {
		if !strings.Contains(textContent.Text, check) {
			t.Errorf("prompt should contain %q", check)
		}
	}
}

func TestHandleKanban(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name           string
		args           map[string]string
		checkFor       []string
		expectedCols   int
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
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "create-kanban",
					Arguments: tt.args,
				},
			}

			result, err := registry.handleKanban(context.Background(), req)
			if err != nil {
				t.Fatalf("handleKanban() error = %v", err)
			}

			textContent := result.Messages[0].Content.(*mcp.TextContent)

			for _, check := range tt.checkFor {
				if !strings.Contains(textContent.Text, check) {
					t.Errorf("prompt should contain %q", check)
				}
			}
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
		if len(result) != len(tt.expected) {
			t.Errorf("splitString(%q, %q) = %v, want %v", tt.input, tt.sep, result, tt.expected)
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("splitString(%q, %q)[%d] = %q, want %q", tt.input, tt.sep, i, v, tt.expected[i])
			}
		}
	}
}
