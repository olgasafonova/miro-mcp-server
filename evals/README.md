# Miro MCP Evaluations

This directory contains evaluation test suites for validating LLM tool selection accuracy with the Miro MCP server.

## Test Suites

### 1. Tool Selection (`tool_selection.json`)
Tests that the correct tool is selected for various natural language prompts.

- **50 tests** covering all major Miro operations
- Categories: boards, create, read, update, delete, tags
- Difficulty levels: easy, medium, hard

### 2. Confusion Pairs (`confusion_pairs.json`)
Tests for distinguishing between commonly confused tools.

- **11 tool pairs** with disambiguation guidance
- **44 tests** for subtle distinctions
- Examples:
  - `miro_list_boards` vs `miro_find_board`
  - `miro_create_sticky` vs `miro_create_sticky_grid`
  - `miro_get_board` vs `miro_get_board_summary`

### 3. Argument Correctness (`argument_correctness.json`)
Tests that arguments are correctly extracted from natural language.

- **25 tests** for argument extraction
- Validates required args, expected values
- Covers colors, positions, sizes, and content

## Running Tests

```bash
# Run all tests
go test ./evals/...

# Run with verbose output
go test ./evals/... -v

# Run specific test
go test ./evals/... -run TestLoadToolSelectionSuite
```

## Using the Framework

```go
package main

import (
    "fmt"
    "github.com/olgasafonova/miro-mcp-server/evals"
)

func main() {
    // Load all test suites
    toolSel, confPairs, args, err := evals.LoadAllEvals("evals/")
    if err != nil {
        panic(err)
    }

    // Create your LLM-based selector
    selector := NewMyLLMSelector()

    // Run evaluations
    metrics, _ := evals.EvaluateToolSelection(toolSel, selector)
    fmt.Println(evals.FormatMetrics(metrics, "Tool Selection"))
}
```

## Implementing a Selector

```go
type ToolSelector interface {
    SelectTool(prompt string) (toolName string, args map[string]interface{}, err error)
}
```

Your selector should:
1. Take a natural language prompt
2. Return the tool name to use
3. Return the arguments to pass
4. Return any errors

## Test Coverage

| Category | Tests | Description |
|----------|-------|-------------|
| Boards | 15 | Board CRUD and sharing |
| Create | 20 | Creating items (stickies, shapes, frames) |
| Read | 10 | Listing and searching items |
| Update | 8 | Modifying items |
| Delete | 5 | Removing items |
| Tags | 6 | Tag management |

## Adding New Tests

1. Add test cases to the appropriate JSON file
2. Follow the existing format:
   - `id`: Unique test identifier
   - `prompt`: Natural language input
   - `expected_tool`: Correct tool name
   - `category`: Logical grouping
   - `difficulty`: easy/medium/hard

3. Run tests to validate JSON syntax:
   ```bash
   go test ./evals/... -run TestLoad
   ```
