package miro

import (
	"context"
	"fmt"
	"strings"
)

// =============================================================================
// Board Search by Name
// =============================================================================

// MaxFindBoardPages caps how many pages of server-filtered results
// FindBoardByName walks before it gives up and answers from the best partial
// match it has. At MaxBoardLimit per page that is 500 candidate boards, which
// is far past the point where a name query is still doing useful work.
const MaxFindBoardPages = 10

// boardMatchAccumulator carries the best non-exact candidate seen so far while
// FindBoardByName walks pages. Each field holds the earliest board in API
// order that reached that tier, so tier dominates and position only breaks
// ties within a tier — the same preference the single-page implementation
// applied, extended across the whole walk.
type boardMatchAccumulator struct {
	prefix   *BoardSummary
	contains *BoardSummary
	first    *BoardSummary
}

// consider folds one page into the accumulator. An exact (case-insensitive)
// name match is returned immediately: exact is the top tier, so no later page
// can beat it, and two boards with the same name are indistinguishable by name
// anyway. nameLower must already be lowercased.
func (a *boardMatchAccumulator) consider(boards []BoardSummary, nameLower string) *BoardSummary {
	for i := range boards {
		board := boards[i]
		lower := strings.ToLower(board.Name)

		if lower == nameLower {
			return &board
		}
		if a.first == nil {
			a.first = &board
		}
		if strings.HasPrefix(lower, nameLower) {
			if a.prefix == nil {
				a.prefix = &board
			}
			continue
		}
		if a.contains == nil && strings.Contains(lower, nameLower) {
			a.contains = &board
		}
	}
	return nil
}

// best returns the surviving candidate once the walk has finished, or nil when
// the query matched nothing at all.
func (a *boardMatchAccumulator) best() *BoardSummary {
	for _, hit := range []*BoardSummary{a.prefix, a.contains, a.first} {
		if hit != nil {
			return hit
		}
	}
	return nil
}

// FindBoardByName finds a board by exact or partial name match.
//
// The query is forwarded to the API, so the set walked here is already
// server-filtered. It is not necessarily one page of it: any query matching
// more than a page of boards used to hide every board past the first page,
// including an exact match. The walk stops as soon as an exact match appears,
// when the filtered set is exhausted, or at MaxFindBoardPages.
func (c *Client) FindBoardByName(ctx context.Context, name string) (*BoardSummary, error) {
	if name == "" {
		return nil, fmt.Errorf("board name is required")
	}

	nameLower := strings.ToLower(name)
	var acc boardMatchAccumulator
	offset := ""

	for page := 0; page < MaxFindBoardPages; page++ {
		result, err := c.ListBoards(ctx, ListBoardsArgs{
			Query:  name,
			Limit:  MaxBoardLimit,
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search boards: %w", err)
		}
		if hit := acc.consider(result.Boards, nameLower); hit != nil {
			return hit, nil
		}
		// A cursor that does not advance would re-scan the same page until
		// the page cap, so treat it as the end rather than spending the
		// remaining requests on it.
		if !result.HasMore || result.Offset == "" || result.Offset == offset {
			break
		}
		offset = result.Offset
	}

	if hit := acc.best(); hit != nil {
		return hit, nil
	}
	return nil, fmt.Errorf("no board found matching '%s'", name)
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
		TeamID:      board.TeamID,
		TeamName:    board.TeamName,
		Owner:       board.Owner,
		CreatedAt:   board.CreatedAt,
		ModifiedAt:  board.ModifiedAt,
		Message:     fmt.Sprintf("Found board '%s'", board.Name),
	}, nil
}
