package miro

import (
	"context"
	"fmt"
	"strings"
)

// =============================================================================
// Board Search
// =============================================================================

// searchBoardLimit clamps the caller's match cap to the configured bounds.
// It governs how many matches come back, never how much of the board is read:
// the items endpoint has no text filter, so matching happens client-side and
// scan depth is a separate budget (DefaultSearchScanItems).
func searchBoardLimit(requested int) int {
	if requested > 0 && requested <= MaxSearchLimit {
		return requested
	}
	return DefaultSearchLimit
}

// boardSearch carries the query in raw and lowercased form for matching.
type boardSearch struct {
	query      string
	queryLower string
}

// match returns a populated ItemMatch when the item's content contains the
// query, or nil otherwise. ItemSummary.Content already folds in data.title for
// the item types that carry a title instead of content.
func (s boardSearch) match(item ItemSummary) *ItemMatch {
	if item.Content == "" || !strings.Contains(strings.ToLower(item.Content), s.queryLower) {
		return nil
	}
	return &ItemMatch{
		ID:      item.ID,
		Type:    item.Type,
		Content: item.Content,
		Snippet: createSnippet(item.Content, s.query, 50),
		X:       item.X,
		Y:       item.Y,
	}
}

// searchStop records why the scan loop ended. It is what lets a caller tell an
// exhaustive "no matches" from a partial one.
type searchStop int

const (
	// stopExhausted means every item on the board was read.
	stopExhausted searchStop = iota
	// stopMatchLimit means the match cap filled before the board ran out.
	stopMatchLimit
	// stopScanCap means the scan cap was hit before the board ran out.
	stopScanCap
	// stopContinue means the page just read ended neither budget, so the walk
	// should fetch the next one. It never reaches a result.
	stopContinue
)

// message composes the human-readable result message. Scan depth and the stop
// reason belong in it: "no items found" after reading 1000 of 4000 items is a
// different claim from "no items found" after reading the whole board, and the
// old message made the first sound like the second.
func (s boardSearch) message(count, scanned int, stop searchStop) string {
	switch {
	case stop == stopScanCap && count == 0:
		return fmt.Sprintf("No items matching '%s' in the first %d items scanned; the scan cap stopped the search before the end of the board. Filter by type, or walk the board with miro_list_all_items.", s.query, scanned)
	case stop == stopScanCap:
		return fmt.Sprintf("Found %d items matching '%s' in the first %d items scanned; the scan cap stopped the search before the end of the board.", count, s.query, scanned)
	case stop == stopMatchLimit:
		return fmt.Sprintf("Found %d items matching '%s' after scanning %d items; the result limit was reached, so more matches may exist further into the board.", count, s.query, scanned)
	case count == 0:
		return fmt.Sprintf("No items found matching '%s' (scanned all %d items on the board)", s.query, scanned)
	default:
		return fmt.Sprintf("Found %d items matching '%s' (scanned all %d items on the board)", count, s.query, scanned)
	}
}

// SearchBoard searches for items containing specific text, paging through the
// board until the match cap is filled, the board is exhausted, or the scan cap
// is reached.
func (c *Client) SearchBoard(ctx context.Context, args SearchBoardArgs) (SearchBoardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return SearchBoardResult{}, err
	}
	if args.Query == "" {
		return SearchBoardResult{}, fmt.Errorf("query is required")
	}

	scan := &searchScan{
		search:     boardSearch{query: args.Query, queryLower: strings.ToLower(args.Query)},
		maxMatches: searchBoardLimit(args.Limit),
		maxScan:    DefaultSearchScanItems,
	}
	stop, err := c.collectSearchMatches(ctx, args, scan)
	if err != nil {
		return SearchBoardResult{}, err
	}

	return SearchBoardResult{
		Matches:      scan.matches,
		Count:        len(scan.matches),
		Query:        args.Query,
		Truncated:    stop != stopExhausted,
		ItemsScanned: scan.scanned,
		Message:      scan.search.message(len(scan.matches), scan.scanned, stop),
	}, nil
}

// searchScan carries the running state of one SearchBoard walk: the query
// being matched, the two budgets that can end it, and what has been found so
// far. Bundling them keeps the page loop from taking a long argument list.
type searchScan struct {
	search     boardSearch
	maxMatches int
	maxScan    int

	matches []ItemMatch
	scanned int
}

// consumePage folds one page of items into the scan. It returns the reason
// the walk should stop, or stopContinue when both budgets still have room.
func (s *searchScan) consumePage(items []ItemSummary) searchStop {
	for i := range items {
		s.scanned++
		if m := s.search.match(items[i]); m != nil {
			s.matches = append(s.matches, *m)
		}
		if len(s.matches) >= s.maxMatches {
			return stopMatchLimit
		}
		if s.scanned >= s.maxScan {
			return stopScanCap
		}
	}
	return stopContinue
}

// collectSearchMatches pages through ListItems, folding each page into scan
// and stopping at the match cap, the scan cap, or the end of the board. It
// mirrors collectAllItems' cursor loop; the difference is that the cap driving
// the loop counts items scanned rather than items returned.
//
// The page size is MaxItemLimit rather than collectAllItems' MaxItemLimitExtended
// because buildListItemsPath silently falls back to DefaultItemLimit for any
// value above MaxItemLimit, so the larger constant never reaches the wire.
func (c *Client) collectSearchMatches(ctx context.Context, args SearchBoardArgs, scan *searchScan) (searchStop, error) {
	cursor := ""
	for {
		result, err := c.ListItems(ctx, ListItemsArgs{
			BoardID: args.BoardID,
			Type:    args.Type,
			Limit:   MaxItemLimit,
			Cursor:  cursor,
		})
		if err != nil {
			return stopExhausted, err
		}
		if stop := scan.consumePage(result.Items); stop != stopContinue {
			return stop, nil
		}
		if !result.HasMore || result.Cursor == "" {
			return stopExhausted, nil
		}
		cursor = result.Cursor
	}
}

// createSnippet creates a text snippet around the matched query.
func createSnippet(content, query string, contextLen int) string {
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)

	idx := strings.Index(lowerContent, lowerQuery)
	if idx == -1 {
		return truncate(content, contextLen*2)
	}

	start := idx - contextLen
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + contextLen
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}

// truncate shortens a string to max length with ellipsis.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
