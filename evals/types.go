package evals

// ToolSelectionTest represents a single tool selection evaluation case
type ToolSelectionTest struct {
	ID           string `json:"id"`
	Prompt       string `json:"prompt"`
	ExpectedTool string `json:"expected_tool"`
	Category     string `json:"category"`
	Difficulty   string `json:"difficulty"`
}

// ToolSelectionSuite contains all tool selection tests
type ToolSelectionSuite struct {
	Name        string              `json:"name"`
	Version     string              `json:"version"`
	Description string              `json:"description"`
	Tests       []ToolSelectionTest `json:"tests"`
}

// ConfusionPairTest represents a single disambiguation test
type ConfusionPairTest struct {
	Prompt       string `json:"prompt"`
	ExpectedTool string `json:"expected_tool"`
	Rationale    string `json:"rationale"`
}

// ConfusionPair represents a pair of tools that are commonly confused
type ConfusionPair struct {
	Tools       []string            `json:"tools"`
	Distinction string              `json:"distinction"`
	Tests       []ConfusionPairTest `json:"tests"`
}

// ConfusionPairSuite contains all confusion pair tests
type ConfusionPairSuite struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Pairs       []ConfusionPair `json:"pairs"`
}

// ArgumentTest represents a single argument correctness test
type ArgumentTest struct {
	ID           string                 `json:"id"`
	Tool         string                 `json:"tool"`
	Prompt       string                 `json:"prompt"`
	ExpectedArgs map[string]interface{} `json:"expected_args"`
	RequiredArgs []string               `json:"required_args"`
	Category     string                 `json:"category"`
}

// ArgumentSuite contains all argument correctness tests
type ArgumentSuite struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Description string         `json:"description"`
	Tests       []ArgumentTest `json:"tests"`
}

// ToolSelectionResult represents the result of a single tool selection evaluation
type ToolSelectionResult struct {
	TestID       string
	Prompt       string
	ExpectedTool string
	ActualTool   string
	Passed       bool
	Errors       []string
}

// ConfusionPairResult represents the result of a confusion pair evaluation
type ConfusionPairResult struct {
	PairTools    []string
	TestPrompt   string
	ExpectedTool string
	ActualTool   string
	Rationale    string
	Passed       bool
}

// ArgumentResult represents the result of an argument correctness evaluation
type ArgumentResult struct {
	TestID      string
	Tool        string
	Prompt      string
	Passed      bool
	MissingArgs []string
	WrongArgs   map[string]string // arg -> "expected X, got Y"
}

// EvalMetrics contains aggregate metrics for an evaluation run
type EvalMetrics struct {
	TotalTests    int
	PassedTests   int
	FailedTests   int
	Accuracy      float64 // PassedTests / TotalTests
	ByCategory    map[string]*CategoryMetrics
	ByDifficulty  map[string]*CategoryMetrics
	ByTool        map[string]*ToolMetrics
	FailedDetails []string
}

// CategoryMetrics contains metrics per category
type CategoryMetrics struct {
	Total  int
	Passed int
	Failed int
}

// ToolMetrics contains metrics per tool
type ToolMetrics struct {
	ExpectedCount  int // times tool was expected
	SelectedCount  int // times tool was actually selected
	CorrectCount   int // times tool was correctly selected
	FalsePositives int // times wrong tool was selected instead
	FalseNegatives int // times this tool should have been selected but wasn't
}
