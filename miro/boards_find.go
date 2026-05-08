package miro

import (
	"context"
	"fmt"
	"strings"
)

// =============================================================================
// Board Search by Name
// =============================================================================

// FindBoardByName finds a board by exact or partial name match.
// Returns the best matching board, preferring exact matches.
func (c *Client) FindBoardByName(ctx context.Context, name string) (*BoardSummary, error) {
	if name == "" {
		return nil, fmt.Errorf("board name is required")
	}

	// Search for boards with the given name
	result, err := c.ListBoards(ctx, ListBoardsArgs{
		Query: name,
		Limit: 20,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search boards: %w", err)
	}

	if len(result.Boards) == 0 {
		return nil, fmt.Errorf("no board found matching '%s'", name)
	}

	nameLower := strings.ToLower(name)

	// First pass: exact match
	for i := range result.Boards {
		if strings.ToLower(result.Boards[i].Name) == nameLower {
			return &result.Boards[i], nil
		}
	}

	// Second pass: starts with match
	for i := range result.Boards {
		if strings.HasPrefix(strings.ToLower(result.Boards[i].Name), nameLower) {
			return &result.Boards[i], nil
		}
	}

	// Third pass: contains match
	for i := range result.Boards {
		if strings.Contains(strings.ToLower(result.Boards[i].Name), nameLower) {
			return &result.Boards[i], nil
		}
	}

	// Return first result as fallback
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
