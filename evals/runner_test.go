package evals

import (
	"testing"
)

// MockSelector implements ToolSelector for testing
type MockSelector struct {
	responses map[string]struct {
		tool string
		args map[string]interface{}
	}
}

func NewMockSelector() *MockSelector {
	return &MockSelector{
		responses: make(map[string]struct {
			tool string
			args map[string]interface{}
		}),
	}
}

func (m *MockSelector) SetResponse(prompt, tool string, args map[string]interface{}) {
	m.responses[prompt] = struct {
		tool string
		args map[string]interface{}
	}{tool: tool, args: args}
}

func (m *MockSelector) SelectTool(prompt string) (string, map[string]interface{}, error) {
	if resp, ok := m.responses[prompt]; ok {
		return resp.tool, resp.args, nil
	}
	return "", nil, nil
}

// mustLoadSuite loads an eval suite from path, failing the test on error.
func mustLoadSuite[S any](t *testing.T, load func(string) (S, error), path string) S {
	t.Helper()
	suite, err := load(path)
	if err != nil {
		t.Fatalf("Failed to load %s: %v", path, err)
	}
	return suite
}

// assertSuiteHeader checks the common suite fields: a name and 1+ tests.
func assertSuiteHeader(t *testing.T, name string, testCount int) {
	t.Helper()
	if name == "" {
		t.Error("Suite name should not be empty")
	}
	if testCount == 0 {
		t.Error("Suite should have tests")
	}
}

// assertRequiredField fails when a loaded test's required field is empty.
func assertRequiredField(t *testing.T, value, label string) {
	t.Helper()
	if value == "" {
		t.Errorf("Test %s should not be empty", label)
	}
}

func TestLoadToolSelectionSuite(t *testing.T) {
	suite := mustLoadSuite(t, LoadToolSelectionSuite, "tool_selection.json")
	assertSuiteHeader(t, suite.Name, len(suite.Tests))

	// Check first test has required fields
	test := suite.Tests[0]
	assertRequiredField(t, test.ID, "ID")
	assertRequiredField(t, test.Prompt, "prompt")
	assertRequiredField(t, test.ExpectedTool, "expected_tool")
}

func TestLoadConfusionPairSuite(t *testing.T) {
	suite := mustLoadSuite(t, LoadConfusionPairSuite, "confusion_pairs.json")

	if suite.Name == "" {
		t.Error("Suite name should not be empty")
	}

	if len(suite.Pairs) == 0 {
		t.Error("Suite should have pairs")
	}

	// Check first pair has required fields
	pair := suite.Pairs[0]
	if len(pair.Tools) < 2 {
		t.Error("Pair should have at least 2 tools")
	}
	if len(pair.Tests) == 0 {
		t.Error("Pair should have tests")
	}
}

func TestLoadArgumentSuite(t *testing.T) {
	suite := mustLoadSuite(t, LoadArgumentSuite, "argument_correctness.json")
	assertSuiteHeader(t, suite.Name, len(suite.Tests))

	// Check first test has required fields
	test := suite.Tests[0]
	assertRequiredField(t, test.ID, "ID")
	assertRequiredField(t, test.Tool, "tool")
	assertRequiredField(t, test.Prompt, "prompt")
}

func TestEvaluateToolSelection(t *testing.T) {
	suite := &ToolSelectionSuite{
		Name: "Test Suite",
		Tests: []ToolSelectionTest{
			{
				ID:           "test-1",
				Prompt:       "Show my boards",
				ExpectedTool: "miro_list_boards",
				Category:     "boards",
				Difficulty:   "easy",
			},
			{
				ID:           "test-2",
				Prompt:       "Create a sticky",
				ExpectedTool: "miro_create_sticky",
				Category:     "create",
				Difficulty:   "easy",
			},
		},
	}

	selector := NewMockSelector()
	selector.SetResponse("Show my boards", "miro_list_boards", nil)
	selector.SetResponse("Create a sticky", "miro_create_sticky", nil)

	metrics, results := EvaluateToolSelection(suite, selector)

	if metrics.TotalTests != 2 {
		t.Errorf("Expected 2 total tests, got %d", metrics.TotalTests)
	}
	if metrics.PassedTests != 2 {
		t.Errorf("Expected 2 passed tests, got %d", metrics.PassedTests)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestEvaluateToolSelectionWithFailure(t *testing.T) {
	suite := &ToolSelectionSuite{
		Name: "Test Suite",
		Tests: []ToolSelectionTest{
			{
				ID:           "test-1",
				Prompt:       "Show my boards",
				ExpectedTool: "miro_list_boards",
				Category:     "boards",
				Difficulty:   "easy",
			},
		},
	}

	selector := NewMockSelector()
	selector.SetResponse("Show my boards", "miro_get_board", nil) // Wrong tool

	metrics, results := EvaluateToolSelection(suite, selector)

	if metrics.PassedTests != 0 {
		t.Errorf("Expected 0 passed tests, got %d", metrics.PassedTests)
	}
	if metrics.FailedTests != 1 {
		t.Errorf("Expected 1 failed test, got %d", metrics.FailedTests)
	}
	if results[0].Passed {
		t.Error("Expected test to fail")
	}
}

func TestEvaluateConfusionPairs(t *testing.T) {
	suite := &ConfusionPairSuite{
		Name: "Test Suite",
		Pairs: []ConfusionPair{
			{
				Tools:       []string{"miro_list_boards", "miro_find_board"},
				Distinction: "list returns all, find searches",
				Tests: []ConfusionPairTest{
					{
						Prompt:       "Show all boards",
						ExpectedTool: "miro_list_boards",
						Rationale:    "Listing all boards",
					},
					{
						Prompt:       "Find Sprint board",
						ExpectedTool: "miro_find_board",
						Rationale:    "Searching for specific board",
					},
				},
			},
		},
	}

	selector := NewMockSelector()
	selector.SetResponse("Show all boards", "miro_list_boards", nil)
	selector.SetResponse("Find Sprint board", "miro_find_board", nil)

	metrics, results := EvaluateConfusionPairs(suite, selector)

	if metrics.TotalTests != 2 {
		t.Errorf("Expected 2 total tests, got %d", metrics.TotalTests)
	}
	if metrics.PassedTests != 2 {
		t.Errorf("Expected 2 passed tests, got %d", metrics.PassedTests)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestEvaluateArguments(t *testing.T) {
	suite := &ArgumentSuite{
		Name: "Test Suite",
		Tests: []ArgumentTest{
			{
				ID:     "test-1",
				Tool:   "miro_create_sticky",
				Prompt: "Create yellow sticky saying Hello",
				ExpectedArgs: map[string]interface{}{
					"content": "Hello",
					"color":   "yellow",
				},
				RequiredArgs: []string{"board_id", "content"},
				Category:     "create",
			},
		},
	}

	selector := NewMockSelector()
	selector.SetResponse("Create yellow sticky saying Hello", "miro_create_sticky", map[string]interface{}{
		"board_id": "abc123",
		"content":  "Hello",
		"color":    "yellow",
	})

	metrics, results := EvaluateArguments(suite, selector)

	if metrics.TotalTests != 1 {
		t.Errorf("Expected 1 total test, got %d", metrics.TotalTests)
	}
	if metrics.PassedTests != 1 {
		t.Errorf("Expected 1 passed test, got %d", metrics.PassedTests)
	}
	if !results[0].Passed {
		t.Errorf("Expected test to pass, got: missing=%v, wrong=%v", results[0].MissingArgs, results[0].WrongArgs)
	}
}

func TestEvaluateArgumentsWithMissingArg(t *testing.T) {
	suite := &ArgumentSuite{
		Name: "Test Suite",
		Tests: []ArgumentTest{
			{
				ID:           "test-1",
				Tool:         "miro_create_sticky",
				Prompt:       "Create sticky",
				ExpectedArgs: map[string]interface{}{},
				RequiredArgs: []string{"board_id", "content"},
				Category:     "create",
			},
		},
	}

	selector := NewMockSelector()
	selector.SetResponse("Create sticky", "miro_create_sticky", map[string]interface{}{
		"board_id": "abc123",
		// Missing "content"
	})

	metrics, results := EvaluateArguments(suite, selector)

	if metrics.PassedTests != 0 {
		t.Errorf("Expected 0 passed tests, got %d", metrics.PassedTests)
	}
	if len(results[0].MissingArgs) == 0 {
		t.Error("Expected missing args")
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
		want     bool
	}{
		{"equal strings", "hello", "hello", true},
		{"different strings", "hello", "world", false},
		{"equal ints", 42, 42, true},
		{"int vs float64", 42, float64(42), true},
		{"different numbers", 42, float64(43), false},
		{"equal slices", []string{"a", "b"}, []string{"a", "b"}, true},
		{"different slices", []string{"a", "b"}, []string{"a", "c"}, false},
		{"nil values", nil, nil, true},
		{"nil vs value", nil, "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareValues(tt.expected, tt.actual)
			if got != tt.want {
				t.Errorf("compareValues(%v, %v) = %v, want %v", tt.expected, tt.actual, got, tt.want)
			}
		})
	}
}

func TestFormatMetrics(t *testing.T) {
	metrics := &EvalMetrics{
		TotalTests:  10,
		PassedTests: 8,
		FailedTests: 2,
		Accuracy:    0.8,
		ByCategory: map[string]*CategoryMetrics{
			"boards": {Total: 5, Passed: 4, Failed: 1},
			"create": {Total: 5, Passed: 4, Failed: 1},
		},
		FailedDetails: []string{"[test-1] prompt: error"},
	}

	output := FormatMetrics(metrics, "Test Suite")

	if output == "" {
		t.Error("FormatMetrics should return non-empty string")
	}
	if !contains(output, "Test Suite") {
		t.Error("Output should contain suite name")
	}
	if !contains(output, "80.0%") {
		t.Error("Output should contain accuracy percentage")
	}
}

func TestLoadAllEvals(t *testing.T) {
	toolSelection, confusionPairs, arguments, err := LoadAllEvals(".")
	if err != nil {
		t.Fatalf("Failed to load all evals: %v", err)
	}

	if toolSelection == nil {
		t.Fatal("Tool selection suite should not be nil")
	}
	if confusionPairs == nil {
		t.Fatal("Confusion pairs suite should not be nil")
	}
	if arguments == nil {
		t.Fatal("Arguments suite should not be nil")
	}

	// Count total tests
	total := len(toolSelection.Tests)
	for _, pair := range confusionPairs.Pairs {
		total += len(pair.Tests)
	}
	total += len(arguments.Tests)

	t.Logf("Loaded %d total evaluation tests", total)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
