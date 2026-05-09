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

// loadSuite reads path and unmarshals into T. Shared by every Load*Suite loader.
func loadSuite[T any](path string) (*T, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is controlled by eval framework
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	var suite T
	if err := json.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	return &suite, nil
}

// LoadToolSelectionSuite loads tool selection tests from a JSON file.
func LoadToolSelectionSuite(path string) (*ToolSelectionSuite, error) {
	return loadSuite[ToolSelectionSuite](path)
}

// LoadConfusionPairSuite loads confusion pair tests from a JSON file.
func LoadConfusionPairSuite(path string) (*ConfusionPairSuite, error) {
	return loadSuite[ConfusionPairSuite](path)
}

// LoadArgumentSuite loads argument correctness tests from a JSON file.
func LoadArgumentSuite(path string) (*ArgumentSuite, error) {
	return loadSuite[ArgumentSuite](path)
}

// ToolSelector is an interface that an LLM or mock can implement for testing
type ToolSelector interface {
	// SelectTool returns the tool name and arguments for a given natural language input
	SelectTool(prompt string) (toolName string, args map[string]interface{}, err error)
}

// EvaluateToolSelection runs tool selection tests against a selector.
func EvaluateToolSelection(suite *ToolSelectionSuite, selector ToolSelector) (*EvalMetrics, []ToolSelectionResult) {
	metrics := &EvalMetrics{
		ByCategory:   make(map[string]*CategoryMetrics),
		ByDifficulty: make(map[string]*CategoryMetrics),
		ByTool:       make(map[string]*ToolMetrics),
	}
	results := make([]ToolSelectionResult, 0, len(suite.Tests))

	for _, test := range suite.Tests {
		metrics.TotalTests++
		ensureCategory(metrics, test.Category)
		metrics.ByCategory[test.Category].Total++
		if metrics.ByDifficulty[test.Difficulty] == nil {
			metrics.ByDifficulty[test.Difficulty] = &CategoryMetrics{}
		}
		metrics.ByDifficulty[test.Difficulty].Total++
		ensureToolMetric(metrics, test.ExpectedTool)
		metrics.ByTool[test.ExpectedTool].ExpectedCount++

		result := evaluateToolSelectionTest(test, selector)
		recordToolMetrics(metrics, test, result)

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

// evaluateToolSelectionTest runs one ToolSelectionTest and returns a result.
func evaluateToolSelectionTest(test ToolSelectionTest, selector ToolSelector) ToolSelectionResult {
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
	if actualTool != test.ExpectedTool {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("wrong tool: expected %s, got %s", test.ExpectedTool, actualTool))
	}
	return result
}

// recordToolMetrics updates per-tool counters (correct, false positive, false
// negative, selected) based on a single tool-selection outcome.
func recordToolMetrics(metrics *EvalMetrics, test ToolSelectionTest, result ToolSelectionResult) {
	ensureToolMetric(metrics, result.ActualTool)
	if result.ActualTool == test.ExpectedTool {
		metrics.ByTool[test.ExpectedTool].CorrectCount++
	} else {
		metrics.ByTool[test.ExpectedTool].FalseNegatives++
		metrics.ByTool[result.ActualTool].FalsePositives++
	}
	metrics.ByTool[result.ActualTool].SelectedCount++
}

// EvaluateConfusionPairs runs confusion pair tests against a selector.
func EvaluateConfusionPairs(suite *ConfusionPairSuite, selector ToolSelector) (*EvalMetrics, []ConfusionPairResult) {
	metrics := &EvalMetrics{
		ByCategory: make(map[string]*CategoryMetrics),
		ByTool:     make(map[string]*ToolMetrics),
	}
	var results []ConfusionPairResult

	for _, pair := range suite.Pairs {
		pairKey := strings.Join(pair.Tools, " vs ")
		ensureCategory(metrics, pairKey)

		for _, test := range pair.Tests {
			metrics.TotalTests++
			metrics.ByCategory[pairKey].Total++
			ensureToolMetric(metrics, test.ExpectedTool)
			metrics.ByTool[test.ExpectedTool].ExpectedCount++

			result := evaluateConfusionTest(pair.Tools, test, selector)
			recordConfusionResult(metrics, pairKey, test, result)
			results = append(results, result)
		}
	}

	if metrics.TotalTests > 0 {
		metrics.Accuracy = float64(metrics.PassedTests) / float64(metrics.TotalTests)
	}
	return metrics, results
}

// evaluateConfusionTest runs one ConfusionPairTest and returns a result.
func evaluateConfusionTest(pairTools []string, test ConfusionPairTest, selector ToolSelector) ConfusionPairResult {
	actualTool, _, err := selector.SelectTool(test.Prompt)
	return ConfusionPairResult{
		PairTools:    pairTools,
		TestPrompt:   test.Prompt,
		ExpectedTool: test.ExpectedTool,
		ActualTool:   actualTool,
		Rationale:    test.Rationale,
		Passed:       err == nil && actualTool == test.ExpectedTool,
	}
}

// recordConfusionResult updates aggregate metrics for one confusion-pair outcome.
func recordConfusionResult(metrics *EvalMetrics, pairKey string, test ConfusionPairTest, result ConfusionPairResult) {
	ensureToolMetric(metrics, result.ActualTool)
	metrics.ByTool[result.ActualTool].SelectedCount++

	if result.Passed {
		metrics.PassedTests++
		metrics.ByCategory[pairKey].Passed++
		metrics.ByTool[test.ExpectedTool].CorrectCount++
		return
	}
	metrics.FailedTests++
	metrics.ByCategory[pairKey].Failed++
	metrics.ByTool[test.ExpectedTool].FalseNegatives++
	metrics.ByTool[result.ActualTool].FalsePositives++
	metrics.FailedDetails = append(metrics.FailedDetails,
		fmt.Sprintf("[%s] %s: expected %s, got %s (%s)",
			pairKey, test.Prompt, test.ExpectedTool, result.ActualTool, test.Rationale))
}

// EvaluateArguments runs argument correctness tests against a selector.
func EvaluateArguments(suite *ArgumentSuite, selector ToolSelector) (*EvalMetrics, []ArgumentResult) {
	metrics := &EvalMetrics{
		ByCategory: make(map[string]*CategoryMetrics),
		ByTool:     make(map[string]*ToolMetrics),
	}
	results := make([]ArgumentResult, 0, len(suite.Tests))

	for _, test := range suite.Tests {
		metrics.TotalTests++
		ensureCategory(metrics, test.Category)
		metrics.ByCategory[test.Category].Total++

		result := evaluateArgumentTest(test, selector)
		recordArgumentResult(metrics, test, result)
		results = append(results, result)
	}

	if metrics.TotalTests > 0 {
		metrics.Accuracy = float64(metrics.PassedTests) / float64(metrics.TotalTests)
	}
	return metrics, results
}

// evaluateArgumentTest runs one ArgumentTest and returns a result with all
// missing/wrong-arg detail filled in. Pure of metrics bookkeeping.
func evaluateArgumentTest(test ArgumentTest, selector ToolSelector) ArgumentResult {
	result := ArgumentResult{
		TestID:    test.ID,
		Tool:      test.Tool,
		Prompt:    test.Prompt,
		Passed:    true,
		WrongArgs: make(map[string]string),
	}

	actualTool, actualArgs, err := selector.SelectTool(test.Prompt)
	if err != nil || actualTool != test.Tool {
		result.Passed = false
		return result
	}

	checkRequiredArgs(actualArgs, test.RequiredArgs, &result)
	checkExpectedArgs(actualArgs, test.ExpectedArgs, &result)
	return result
}

// checkRequiredArgs marks any required-but-absent args as missing on result.
func checkRequiredArgs(actualArgs map[string]interface{}, required []string, result *ArgumentResult) {
	for _, reqArg := range required {
		if _, exists := actualArgs[reqArg]; !exists {
			result.Passed = false
			result.MissingArgs = append(result.MissingArgs, reqArg)
		}
	}
}

// checkExpectedArgs verifies each expected arg is present and value-equal,
// recording missing or mismatched entries on result.
func checkExpectedArgs(actualArgs, expected map[string]interface{}, result *ArgumentResult) {
	for key, expectedValue := range expected {
		actualValue, exists := actualArgs[key]
		if !exists {
			result.Passed = false
			result.MissingArgs = append(result.MissingArgs, key)
			continue
		}
		if !compareValues(expectedValue, actualValue) {
			result.Passed = false
			result.WrongArgs[key] = fmt.Sprintf("expected %v, got %v", expectedValue, actualValue)
		}
	}
}

// recordArgumentResult updates aggregate metrics based on a single ArgumentResult.
func recordArgumentResult(metrics *EvalMetrics, test ArgumentTest, result ArgumentResult) {
	if result.Passed {
		metrics.PassedTests++
		metrics.ByCategory[test.Category].Passed++
		return
	}
	metrics.FailedTests++
	metrics.ByCategory[test.Category].Failed++
	if details := formatArgumentFailure(result); details != "" {
		metrics.FailedDetails = append(metrics.FailedDetails,
			fmt.Sprintf("[%s] %s: %s", test.ID, test.Prompt, details))
	}
}

// formatArgumentFailure renders a one-line summary of what went wrong.
// Returns empty string when the failure was a wrong-tool / selector-error
// (no per-arg detail), matching the original "skip details" behavior.
func formatArgumentFailure(result ArgumentResult) string {
	if len(result.MissingArgs) == 0 && len(result.WrongArgs) == 0 {
		return ""
	}
	var parts []string
	if len(result.MissingArgs) > 0 {
		parts = append(parts, fmt.Sprintf("missing: %v", result.MissingArgs))
	}
	for k, v := range result.WrongArgs {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v))
	}
	return strings.Join(parts, "; ")
}

// ensureCategory creates a CategoryMetrics entry if absent.
func ensureCategory(metrics *EvalMetrics, name string) {
	if metrics.ByCategory[name] == nil {
		metrics.ByCategory[name] = &CategoryMetrics{}
	}
}

// ensureToolMetric creates a ToolMetrics entry if absent.
func ensureToolMetric(metrics *EvalMetrics, name string) {
	if metrics.ByTool[name] == nil {
		metrics.ByTool[name] = &ToolMetrics{}
	}
}

// compareValues compares expected and actual values, tolerating the float64
// coercion JSON unmarshaling applies to numbers.
func compareValues(expected, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	ev := reflect.ValueOf(expected)
	av := reflect.ValueOf(actual)

	if equal, ok := compareNumeric(ev, av); ok {
		return equal
	}
	if ev.Kind() == reflect.Slice && av.Kind() == reflect.Slice {
		return compareSlices(ev, av)
	}
	return reflect.DeepEqual(expected, actual)
}

// compareNumeric handles the JSON-unmarshals-numbers-to-float64 case. Returns
// (equal, true) when ev is numeric and av is float64; (_, false) otherwise.
func compareNumeric(ev, av reflect.Value) (bool, bool) {
	if av.Kind() != reflect.Float64 {
		return false, false
	}
	switch ev.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(ev.Int()) == av.Float(), true
	case reflect.Float32, reflect.Float64:
		return ev.Float() == av.Float(), true
	}
	return false, false
}

// compareSlices compares two slice reflect.Values element-by-element via compareValues.
func compareSlices(ev, av reflect.Value) bool {
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

// FormatMetrics returns a human-readable summary of evaluation metrics.
func FormatMetrics(metrics *EvalMetrics, suiteName string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("\n=== %s ===\n", suiteName))
	b.WriteString(fmt.Sprintf("Total: %d tests\n", metrics.TotalTests))
	b.WriteString(fmt.Sprintf("Passed: %d (%.1f%%)\n", metrics.PassedTests, metrics.Accuracy*100))
	b.WriteString(fmt.Sprintf("Failed: %d\n", metrics.FailedTests))

	writeBreakdown(&b, "By Category", metrics.ByCategory, 30)
	writeBreakdown(&b, "By Difficulty", metrics.ByDifficulty, 10)
	writeFailureSection(&b, metrics.FailedDetails)

	return b.String()
}

// writeBreakdown renders one "By X" block with right-aligned padding.
func writeBreakdown(b *strings.Builder, title string, breakdown map[string]*CategoryMetrics, padding int) {
	if len(breakdown) == 0 {
		return
	}
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString(":\n")
	for name, m := range breakdown {
		if m.Total > 0 {
			acc := float64(m.Passed) / float64(m.Total) * 100
			b.WriteString(fmt.Sprintf("  %-*s: %d/%d (%.0f%%)\n", padding, name, m.Passed, m.Total, acc))
		}
	}
}

// writeFailureSection renders the trailing "Failed Tests" block, capping at 10
// entries with a count when truncated.
func writeFailureSection(b *strings.Builder, details []string) {
	if len(details) == 0 {
		return
	}
	if len(details) <= 10 {
		b.WriteString("\nFailed Tests:\n")
	} else {
		b.WriteString(fmt.Sprintf("\nFailed Tests (showing first 10 of %d):\n", len(details)))
		details = details[:10]
	}
	for _, detail := range details {
		b.WriteString(fmt.Sprintf("  - %s\n", detail))
	}
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
