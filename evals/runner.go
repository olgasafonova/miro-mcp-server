// Package evals provides evaluation framework for testing MCP tool selection accuracy.
// It validates that LLMs select the correct Miro tools and extract proper arguments
// from natural language inputs.
package evals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

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

// LoadToolSelectionSuite loads tool selection tests from a JSON file
func LoadToolSelectionSuite(path string) (*ToolSelectionSuite, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is controlled by eval framework
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var suite ToolSelectionSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return &suite, nil
}

// LoadConfusionPairSuite loads confusion pair tests from a JSON file
func LoadConfusionPairSuite(path string) (*ConfusionPairSuite, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is controlled by eval framework
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var suite ConfusionPairSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return &suite, nil
}

// LoadArgumentSuite loads argument correctness tests from a JSON file
func LoadArgumentSuite(path string) (*ArgumentSuite, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is controlled by eval framework
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var suite ArgumentSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return &suite, nil
}

// ToolSelector is an interface that an LLM or mock can implement for testing
type ToolSelector interface {
	// SelectTool returns the tool name and arguments for a given natural language input
	SelectTool(prompt string) (toolName string, args map[string]interface{}, err error)
}

// EvaluateToolSelection runs tool selection tests against a selector
func EvaluateToolSelection(suite *ToolSelectionSuite, selector ToolSelector) (*EvalMetrics, []ToolSelectionResult) {
	metrics := &EvalMetrics{
		ByCategory:   make(map[string]*CategoryMetrics),
		ByDifficulty: make(map[string]*CategoryMetrics),
		ByTool:       make(map[string]*ToolMetrics),
	}
	var results []ToolSelectionResult

	for _, test := range suite.Tests {
		metrics.TotalTests++

		// Initialize category metrics
		if metrics.ByCategory[test.Category] == nil {
			metrics.ByCategory[test.Category] = &CategoryMetrics{}
		}
		metrics.ByCategory[test.Category].Total++

		// Initialize difficulty metrics
		if metrics.ByDifficulty[test.Difficulty] == nil {
			metrics.ByDifficulty[test.Difficulty] = &CategoryMetrics{}
		}
		metrics.ByDifficulty[test.Difficulty].Total++

		// Initialize tool metrics
		if metrics.ByTool[test.ExpectedTool] == nil {
			metrics.ByTool[test.ExpectedTool] = &ToolMetrics{}
		}
		metrics.ByTool[test.ExpectedTool].ExpectedCount++

		// Run the selector
		actualTool, _, err := selector.SelectTool(test.Prompt)

		result := ToolSelectionResult{
			TestID:       test.ID,
			Prompt:       test.Prompt,
			ExpectedTool: test.ExpectedTool,
			ActualTool:   actualTool,
			Passed:       true,
		}

		if err != nil {
			result.Passed = false
			result.Errors = append(result.Errors, fmt.Sprintf("selector error: %v", err))
		}

		// Check tool selection
		if actualTool != test.ExpectedTool {
			result.Passed = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("wrong tool: expected %s, got %s", test.ExpectedTool, actualTool))
			metrics.ByTool[test.ExpectedTool].FalseNegatives++

			if metrics.ByTool[actualTool] == nil {
				metrics.ByTool[actualTool] = &ToolMetrics{}
			}
			metrics.ByTool[actualTool].FalsePositives++
		} else {
			metrics.ByTool[test.ExpectedTool].CorrectCount++
		}

		// Track selected count
		if metrics.ByTool[actualTool] == nil {
			metrics.ByTool[actualTool] = &ToolMetrics{}
		}
		metrics.ByTool[actualTool].SelectedCount++

		// Update metrics
		if result.Passed {
			metrics.PassedTests++
			metrics.ByCategory[test.Category].Passed++
			metrics.ByDifficulty[test.Difficulty].Passed++
		} else {
			metrics.FailedTests++
			metrics.ByCategory[test.Category].Failed++
			metrics.ByDifficulty[test.Difficulty].Failed++
			metrics.FailedDetails = append(metrics.FailedDetails,
				fmt.Sprintf("[%s] %s: %s", test.ID, test.Prompt, strings.Join(result.Errors, "; ")))
		}

		results = append(results, result)
	}

	if metrics.TotalTests > 0 {
		metrics.Accuracy = float64(metrics.PassedTests) / float64(metrics.TotalTests)
	}

	return metrics, results
}

// EvaluateConfusionPairs runs confusion pair tests against a selector
func EvaluateConfusionPairs(suite *ConfusionPairSuite, selector ToolSelector) (*EvalMetrics, []ConfusionPairResult) {
	metrics := &EvalMetrics{
		ByCategory: make(map[string]*CategoryMetrics),
		ByTool:     make(map[string]*ToolMetrics),
	}
	var results []ConfusionPairResult

	for _, pair := range suite.Pairs {
		// Use tool pair as category key
		pairKey := strings.Join(pair.Tools, " vs ")
		if metrics.ByCategory[pairKey] == nil {
			metrics.ByCategory[pairKey] = &CategoryMetrics{}
		}

		for _, test := range pair.Tests {
			metrics.TotalTests++
			metrics.ByCategory[pairKey].Total++

			// Initialize tool metrics
			if metrics.ByTool[test.ExpectedTool] == nil {
				metrics.ByTool[test.ExpectedTool] = &ToolMetrics{}
			}
			metrics.ByTool[test.ExpectedTool].ExpectedCount++

			// Run the selector
			actualTool, _, err := selector.SelectTool(test.Prompt)

			result := ConfusionPairResult{
				PairTools:    pair.Tools,
				TestPrompt:   test.Prompt,
				ExpectedTool: test.ExpectedTool,
				ActualTool:   actualTool,
				Rationale:    test.Rationale,
				Passed:       err == nil && actualTool == test.ExpectedTool,
			}

			// Track metrics
			if metrics.ByTool[actualTool] == nil {
				metrics.ByTool[actualTool] = &ToolMetrics{}
			}
			metrics.ByTool[actualTool].SelectedCount++

			if result.Passed {
				metrics.PassedTests++
				metrics.ByCategory[pairKey].Passed++
				metrics.ByTool[test.ExpectedTool].CorrectCount++
			} else {
				metrics.FailedTests++
				metrics.ByCategory[pairKey].Failed++
				metrics.ByTool[test.ExpectedTool].FalseNegatives++
				metrics.ByTool[actualTool].FalsePositives++
				metrics.FailedDetails = append(metrics.FailedDetails,
					fmt.Sprintf("[%s] %s: expected %s, got %s (%s)",
						pairKey, test.Prompt, test.ExpectedTool, actualTool, test.Rationale))
			}

			results = append(results, result)
		}
	}

	if metrics.TotalTests > 0 {
		metrics.Accuracy = float64(metrics.PassedTests) / float64(metrics.TotalTests)
	}

	return metrics, results
}

// EvaluateArguments runs argument correctness tests against a selector
func EvaluateArguments(suite *ArgumentSuite, selector ToolSelector) (*EvalMetrics, []ArgumentResult) {
	metrics := &EvalMetrics{
		ByCategory: make(map[string]*CategoryMetrics),
		ByTool:     make(map[string]*ToolMetrics),
	}
	var results []ArgumentResult

	for _, test := range suite.Tests {
		metrics.TotalTests++

		// Use category from test
		if metrics.ByCategory[test.Category] == nil {
			metrics.ByCategory[test.Category] = &CategoryMetrics{}
		}
		metrics.ByCategory[test.Category].Total++

		// Run the selector
		actualTool, actualArgs, err := selector.SelectTool(test.Prompt)

		result := ArgumentResult{
			TestID:    test.ID,
			Tool:      test.Tool,
			Prompt:    test.Prompt,
			Passed:    true,
			WrongArgs: make(map[string]string),
		}

		if err != nil {
			result.Passed = false
			metrics.FailedTests++
			metrics.ByCategory[test.Category].Failed++
			results = append(results, result)
			continue
		}

		// Check correct tool was selected first
		if actualTool != test.Tool {
			result.Passed = false
			metrics.FailedTests++
			metrics.ByCategory[test.Category].Failed++
			results = append(results, result)
			continue
		}

		// Check required arguments
		for _, reqArg := range test.RequiredArgs {
			if _, exists := actualArgs[reqArg]; !exists {
				result.Passed = false
				result.MissingArgs = append(result.MissingArgs, reqArg)
			}
		}

		// Check expected argument values
		for key, expectedValue := range test.ExpectedArgs {
			actualValue, exists := actualArgs[key]
			if !exists {
				result.Passed = false
				result.MissingArgs = append(result.MissingArgs, key)
			} else if !compareValues(expectedValue, actualValue) {
				result.Passed = false
				result.WrongArgs[key] = fmt.Sprintf("expected %v, got %v", expectedValue, actualValue)
			}
		}

		// Update metrics
		if result.Passed {
			metrics.PassedTests++
			metrics.ByCategory[test.Category].Passed++
		} else {
			metrics.FailedTests++
			metrics.ByCategory[test.Category].Failed++

			var errDetails []string
			if len(result.MissingArgs) > 0 {
				errDetails = append(errDetails, fmt.Sprintf("missing: %v", result.MissingArgs))
			}
			for k, v := range result.WrongArgs {
				errDetails = append(errDetails, fmt.Sprintf("%s: %s", k, v))
			}
			metrics.FailedDetails = append(metrics.FailedDetails,
				fmt.Sprintf("[%s] %s: %s", test.ID, test.Prompt, strings.Join(errDetails, "; ")))
		}

		results = append(results, result)
	}

	if metrics.TotalTests > 0 {
		metrics.Accuracy = float64(metrics.PassedTests) / float64(metrics.TotalTests)
	}

	return metrics, results
}

// compareValues compares expected and actual values, handling type differences
func compareValues(expected, actual interface{}) bool {
	// Handle nil cases
	if expected == nil && actual == nil {
		return true
	}
	if expected == nil || actual == nil {
		return false
	}

	// Use reflect for deep comparison
	ev := reflect.ValueOf(expected)
	av := reflect.ValueOf(actual)

	// Handle numeric type differences (JSON unmarshals to float64)
	switch ev.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if av.Kind() == reflect.Float64 {
			return float64(ev.Int()) == av.Float()
		}
	case reflect.Float32, reflect.Float64:
		if av.Kind() == reflect.Float64 {
			return ev.Float() == av.Float()
		}
	}

	// Handle slice comparison
	if ev.Kind() == reflect.Slice && av.Kind() == reflect.Slice {
		if ev.Len() != av.Len() {
			return false
		}
		for i := 0; i < ev.Len(); i++ {
			if !compareValues(ev.Index(i).Interface(), av.Index(i).Interface()) {
				return false
			}
		}
		return true
	}

	// Default: use reflect.DeepEqual
	return reflect.DeepEqual(expected, actual)
}

// FormatMetrics returns a human-readable summary of evaluation metrics
func FormatMetrics(metrics *EvalMetrics, suiteName string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("\n=== %s ===\n", suiteName))
	b.WriteString(fmt.Sprintf("Total: %d tests\n", metrics.TotalTests))
	b.WriteString(fmt.Sprintf("Passed: %d (%.1f%%)\n", metrics.PassedTests, metrics.Accuracy*100))
	b.WriteString(fmt.Sprintf("Failed: %d\n", metrics.FailedTests))

	if len(metrics.ByCategory) > 0 {
		b.WriteString("\nBy Category:\n")
		for cat, m := range metrics.ByCategory {
			if m.Total > 0 {
				acc := float64(m.Passed) / float64(m.Total) * 100
				b.WriteString(fmt.Sprintf("  %-30s: %d/%d (%.0f%%)\n", cat, m.Passed, m.Total, acc))
			}
		}
	}

	if len(metrics.ByDifficulty) > 0 {
		b.WriteString("\nBy Difficulty:\n")
		for diff, m := range metrics.ByDifficulty {
			if m.Total > 0 {
				acc := float64(m.Passed) / float64(m.Total) * 100
				b.WriteString(fmt.Sprintf("  %-10s: %d/%d (%.0f%%)\n", diff, m.Passed, m.Total, acc))
			}
		}
	}

	if len(metrics.FailedDetails) > 0 && len(metrics.FailedDetails) <= 10 {
		b.WriteString("\nFailed Tests:\n")
		for _, detail := range metrics.FailedDetails {
			b.WriteString(fmt.Sprintf("  - %s\n", detail))
		}
	} else if len(metrics.FailedDetails) > 10 {
		b.WriteString(fmt.Sprintf("\nFailed Tests (showing first 10 of %d):\n", len(metrics.FailedDetails)))
		for _, detail := range metrics.FailedDetails[:10] {
			b.WriteString(fmt.Sprintf("  - %s\n", detail))
		}
	}

	return b.String()
}

// LoadAllEvals loads all evaluation suites from a directory
func LoadAllEvals(dir string) (*ToolSelectionSuite, *ConfusionPairSuite, *ArgumentSuite, error) {
	toolSelection, err := LoadToolSelectionSuite(filepath.Join(dir, "tool_selection.json"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading tool selection: %w", err)
	}

	confusionPairs, err := LoadConfusionPairSuite(filepath.Join(dir, "confusion_pairs.json"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading confusion pairs: %w", err)
	}

	arguments, err := LoadArgumentSuite(filepath.Join(dir, "argument_correctness.json"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading arguments: %w", err)
	}

	return toolSelection, confusionPairs, arguments, nil
}
