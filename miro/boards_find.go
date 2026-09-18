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

// boardMatchTier names how the returned board answered the query. It is
// reported to the caller verbatim, so an agent that asked for an exact board
// can tell what it actually got. boardMatchNone is the honest tier: the board
// came back because the walk had to answer with something, not because its
// name matched.
type boardMatchTier string

const (
	boardMatchExact    boardMatchTier = "exact"
	boardMatchPrefix   boardMatchTier = "prefix"
	boardMatchContains boardMatchTier = "contains"
	boardMatchNone     boardMatchTier = "none"
)

// boardQuery is a board name to search for, carried together with the
// lowercased form the tier tests compare against. Establishing that form once
// at construction is what lets the tests be a plain classification: the
// alternative is a bare string parameter and a comment at every call site
// promising it was lowercased already.
type boardQuery struct {
	name  string
	lower string
}

func newBoardQuery(name string) boardQuery {
	return boardQuery{name: name, lower: strings.ToLower(name)}
}

// tier classifies one board against the query. It is the single definition of
// what each tier means, so the accumulator only has to decide what to keep.
func (q boardQuery) tier(board BoardSummary) boardMatchTier {
	lower := strings.ToLower(board.Name)
	switch {
	case lower == q.lower:
		return boardMatchExact
	case strings.HasPrefix(lower, q.lower):
		return boardMatchPrefix
	case strings.Contains(lower, q.lower):
		return boardMatchContains
	}
	return boardMatchNone
}

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
// anyway.
func (a *boardMatchAccumulator) consider(boards []BoardSummary, query boardQuery) *BoardSummary {
	for i := range boards {
		if exact := a.fold(boards[i], query); exact != nil {
			return exact
		}
	}
	return nil
}

// fold keeps one board if it is the earliest of its tier, and returns it when
// the name matches exactly. Every non-exact board is also a candidate for the
// last-resort first field, whichever tier it reached.
func (a *boardMatchAccumulator) fold(board BoardSummary, query boardQuery) *BoardSummary {
	switch query.tier(board) {
	case boardMatchExact:
		return &board
	case boardMatchPrefix:
		if a.prefix == nil {
			a.prefix = &board
		}
	case boardMatchContains:
		if a.contains == nil {
			a.contains = &board
		}
	}
	if a.first == nil {
		a.first = &board
	}
	return nil
}

// best returns the surviving candidate once the walk has finished, with the
// tier that earned it. The last resort is the first board of the walk, which
// reached no tier at all: it is reported as boardMatchNone so the caller can
// tell a fallback from a find. A nil board means the query matched nothing and
// there was nothing to fall back to either.
func (a *boardMatchAccumulator) best() (*BoardSummary, boardMatchTier) {
	if a.prefix != nil {
		return a.prefix, boardMatchPrefix
	}
	if a.contains != nil {
		return a.contains, boardMatchContains
	}
	return a.first, boardMatchNone
}

// nextFindOffset returns the offset of the page after the one just read, and
// whether the walk should continue. A cursor that is empty or does not advance
// would re-scan the same page until the page cap, so it counts as the end
// rather than spending the remaining requests on it.
func nextFindOffset(result ListBoardsResult, current string) (string, bool) {
	if !result.HasMore {
		return "", false
	}
	if result.Offset == "" || result.Offset == current {
		return "", false
	}
	return result.Offset, true
}

// boardMatch is the whole answer to a name query: the board chosen, the tier
// that chose it, and the query it was judged against. The three travel
// together because describing the result honestly needs all of them.
type boardMatch struct {
	board *BoardSummary
	tier  boardMatchTier
	query boardQuery
}

// message describes the result in the caller's own terms. Only the three real
// tiers are allowed to say "Found"; boardMatchNone says plainly that nothing
// matched and that the board it hands back is a guess.
func (m boardMatch) message() string {
	switch m.tier {
	case boardMatchExact:
		return fmt.Sprintf("Found board '%s': exact name match for '%s'", m.board.Name, m.query.name)
	case boardMatchPrefix:
		return fmt.Sprintf("Found board '%s': name starts with '%s'", m.board.Name, m.query.name)
	case boardMatchContains:
		return fmt.Sprintf("Found board '%s': name contains '%s'", m.board.Name, m.query.name)
	}
	return fmt.Sprintf("No board name matched '%s'. Returning '%s' as the nearest candidate: it is a guess, not a match. Use miro_list_boards to see what exists", m.query.name, m.board.Name)
}

// findBoardMatch runs the search and reports which tier the answer reached.
//
// The query is forwarded to the API, so the set walked here is already
// server-filtered. It is not necessarily one page of it: any query matching
// more than a page of boards used to hide every board past the first page,
// including an exact match. The walk stops as soon as an exact match appears,
// when the filtered set is exhausted, or at MaxFindBoardPages.
func (c *Client) findBoardMatch(ctx context.Context, name string) (boardMatch, error) {
	if name == "" {
		return boardMatch{}, fmt.Errorf("board name is required")
	}

	query := newBoardQuery(name)
	var acc boardMatchAccumulator
	offset := ""

	for page := 0; page < MaxFindBoardPages; page++ {
		result, err := c.ListBoards(ctx, ListBoardsArgs{
			Query:  name,
			Limit:  MaxBoardLimit,
			Offset: offset,
		})
		if err != nil {
			return boardMatch{}, fmt.Errorf("failed to search boards: %w", err)
		}
		if hit := acc.consider(result.Boards, query); hit != nil {
			return boardMatch{board: hit, tier: boardMatchExact, query: query}, nil
		}
		next, ok := nextFindOffset(result, offset)
		if !ok {
			break
		}
		offset = next
	}

	if hit, tier := acc.best(); hit != nil {
		return boardMatch{board: hit, tier: tier, query: query}, nil
	}
	return boardMatch{}, fmt.Errorf("no board found matching '%s'", name)
}

// FindBoardByName finds a board by exact or partial name match, falling back
// to the first board of the filtered set when nothing matched any tier. A
// caller that needs to tell those apart should use FindBoardByNameTool, whose
// result names the tier.
func (c *Client) FindBoardByName(ctx context.Context, name string) (*BoardSummary, error) {
	match, err := c.findBoardMatch(ctx, name)
	return match.board, err
}

// FindBoardByNameTool wraps findBoardMatch with args/result types for MCP.
func (c *Client) FindBoardByNameTool(ctx context.Context, args FindBoardByNameArgs) (FindBoardByNameResult, error) {
	match, err := c.findBoardMatch(ctx, args.Name)
	if err != nil {
		return FindBoardByNameResult{}, err
	}
	board := match.board

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
		Match:       string(match.tier),
		Message:     match.message(),
	}, nil
}
