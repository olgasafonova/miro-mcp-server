package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// =============================================================================
// Board Search
// =============================================================================

// searchBoardItem is the partial item shape needed for content search.
type searchBoardItem struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Position *struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"position"`
	Data struct {
		Content string `json:"content"`
		Title   string `json:"title"`
	} `json:"data"`
}

// searchBoardLimit clamps the user-supplied limit to the configured bounds.
func searchBoardLimit(requested int) int {
	if requested > 0 && requested < MaxSearchLimit {
		return requested
	}
	return DefaultSearchLimit
}

// searchBoardPath builds the GET path with type/limit query parameters.
func searchBoardPath(boardID, itemType string, limit int) string {
	params := url.Values{}
	if itemType != "" {
		params.Set("type", itemType)
	}
	params.Set("limit", strconv.Itoa(limit))
	return "/boards/" + boardID + "/items?" + params.Encode()
}

// boardItemMatch returns a populated ItemMatch when the item's content or
// title contains queryLower, or nil otherwise. raw is parsed in place.
func boardItemMatch(raw json.RawMessage, query, queryLower string) *ItemMatch {
	var item searchBoardItem
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil
	}
	content := item.Data.Content
	if content == "" {
		content = item.Data.Title
	}
	if content == "" || !strings.Contains(strings.ToLower(content), queryLower) {
		return nil
	}
	match := ItemMatch{
		ID:      item.ID,
		Type:    item.Type,
		Content: content,
		Snippet: createSnippet(content, query, 50),
	}
	if item.Position != nil {
		match.X = item.Position.X
		match.Y = item.Position.Y
	}
	return &match
}

// searchBoardMessage composes the human-readable result message.
func searchBoardMessage(count int, query string) string {
	if count == 0 {
		return fmt.Sprintf("No items found matching '%s'", query)
	}
	return fmt.Sprintf("Found %d items matching '%s'", count, query)
}

// SearchBoard searches for items containing specific text.
func (c *Client) SearchBoard(ctx context.Context, args SearchBoardArgs) (SearchBoardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return SearchBoardResult{}, err
	}
	if args.Query == "" {
		return SearchBoardResult{}, fmt.Errorf("query is required")
	}

	path := searchBoardPath(args.BoardID, args.Type, searchBoardLimit(args.Limit))
	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return SearchBoardResult{}, err
	}

	var resp struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return SearchBoardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	queryLower := strings.ToLower(args.Query)
	var matches []ItemMatch
	for _, raw := range resp.Data {
		if m := boardItemMatch(raw, args.Query, queryLower); m != nil {
			matches = append(matches, *m)
		}
	}

	return SearchBoardResult{
		Matches: matches,
		Count:   len(matches),
		Query:   args.Query,
		Message: searchBoardMessage(len(matches), args.Query),
	}, nil
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
