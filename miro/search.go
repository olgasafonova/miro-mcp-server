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

// SearchBoard searches for items containing specific text.
func (c *Client) SearchBoard(ctx context.Context, args SearchBoardArgs) (SearchBoardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return SearchBoardResult{}, err
	}
	if args.Query == "" {
		return SearchBoardResult{}, fmt.Errorf("query is required")
	}

	limit := DefaultSearchLimit
	if args.Limit > 0 && args.Limit < MaxSearchLimit {
		limit = args.Limit
	}

	// Fetch items from the board
	params := url.Values{}
	if args.Type != "" {
		params.Set("type", args.Type)
	}
	params.Set("limit", strconv.Itoa(limit))

	path := "/boards/" + args.BoardID + "/items"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

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

	// Search through items for matching content
	queryLower := strings.ToLower(args.Query)
	var matches []ItemMatch

	for _, raw := range resp.Data {
		var item struct {
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
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}

		// Check content and title for matches
		content := item.Data.Content
		if content == "" {
			content = item.Data.Title
		}

		if content != "" && strings.Contains(strings.ToLower(content), queryLower) {
			match := ItemMatch{
				ID:      item.ID,
				Type:    item.Type,
				Content: content,
				Snippet: createSnippet(content, args.Query, 50),
			}
			if item.Position != nil {
				match.X = item.Position.X
				match.Y = item.Position.Y
			}
			matches = append(matches, match)
		}
	}

	message := fmt.Sprintf("Found %d items matching '%s'", len(matches), args.Query)
	if len(matches) == 0 {
		message = fmt.Sprintf("No items found matching '%s'", args.Query)
	}

	return SearchBoardResult{
		Matches: matches,
		Count:   len(matches),
		Query:   args.Query,
		Message: message,
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
