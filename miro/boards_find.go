package miro

import (
	"context"
	"fmt"
	"strings"
)

// =============================================================================
// Board Search by Name
// =============================================================================

// findFirstBoardMatch returns the first board whose name (case-insensitive)
// satisfies match. nameLower must already be lowercased.
func findFirstBoardMatch(boards []BoardSummary, nameLower string, match func(boardName, target string) bool) *BoardSummary {
	for i := range boards {
		if match(strings.ToLower(boards[i].Name), nameLower) {
			return &boards[i]
		}
	}
	return nil
}

// FindBoardByName finds a board by exact or partial name match.
// Returns the best matching board, preferring exact matches.
func (c *Client) FindBoardByName(ctx context.Context, name string) (*BoardSummary, error) {
	if name == "" {
		return nil, fmt.Errorf("board name is required")
	}

	result, err := c.ListBoards(ctx, ListBoardsArgs{Query: name, Limit: 20})
	if err != nil {
		return nil, fmt.Errorf("failed to search boards: %w", err)
	}
	if len(result.Boards) == 0 {
		return nil, fmt.Errorf("no board found matching '%s'", name)
	}

	nameLower := strings.ToLower(name)
	matchers := []func(string, string) bool{
		func(a, b string) bool { return a == b },
		strings.HasPrefix,
		strings.Contains,
	}
	for _, m := range matchers {
		if hit := findFirstBoardMatch(result.Boards, nameLower, m); hit != nil {
			return hit, nil
		}
	}
	return &result.Boards[0], nil
}

// FindBoardByNameTool wraps FindBoardByName with args/result types for MCP.
func (c *Client) FindBoardByNameTool(ctx context.Context, args FindBoardByNameArgs) (FindBoardByNameResult, error) {
	board, err := c.FindBoardByName(ctx, args.Name)
	if err != nil {
		return FindBoardByNameResult{}, err
	}

	return FindBoardByNameResult{
		ID:          board.ID,
		Name:        board.Name,
		Description: board.Description,
		ViewLink:    board.ViewLink,
		Message:     fmt.Sprintf("Found board '%s'", board.Name),
	}, nil
}
