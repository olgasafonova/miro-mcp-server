package evals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

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
