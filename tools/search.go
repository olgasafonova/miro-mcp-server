package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// ToolSearchName is the MCP name of the discovery meta-tool.
const ToolSearchName = "miro_tool_search"

// ToolSearchArgs are the inputs to miro_tool_search.
type ToolSearchArgs struct {
	// Query matches against tool name, title, and description (case-insensitive).
	// Empty query plus a Category returns all tools in that category.
	Query string `json:"query,omitempty" jsonschema:"Search terms matched against tool name, title, and description"`

	// Category filters results to a single category (e.g. boards, create, read).
	Category string `json:"category,omitempty" jsonschema:"Filter by category: boards, create, read, update, delete, tags, members, export, audit, diagrams"`

	// Limit caps the number of results. Default 10, max 50.
	Limit int `json:"limit,omitempty" jsonschema:"Maximum results to return (default 10, max 50)"`
}

// ToolSearchMatch is one tool returned from a search.
type ToolSearchMatch struct {
	Name        string  `json:"name"`
	Category    string  `json:"category,omitempty"`
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}

// ToolSearchResult is the structured response from miro_tool_search.
type ToolSearchResult struct {
	Tools   []ToolSearchMatch `json:"tools"`
	Total   int               `json:"total"`
	Message string            `json:"message"`
}

// SearchToolSpec is the ToolSpec for miro_tool_search. Defined here (rather
// than inlined into AllTools) so the registration site and the search
// implementation stay close.
var SearchToolSpec = ToolSpec{
	Name:       ToolSearchName,
	Method:     "SearchTools",
	Title:      "Find Miro Tools",
	Category:   "discovery",
	ReadOnly:   true,
	Idempotent: true,
	Description: `Find Miro tools by keyword or category. Returns matching tool names + short descriptions; call those tools directly afterward.

USE WHEN: you don't know which tool exists for a task, or you want to scope to a category before browsing. Examples: "find tools for stickies", "what can I do with frames?", "show me all destructive tools".

PARAMETERS:
- query: keywords matched against tool name, title, description (e.g. "sticky note", "share board", "diagram").
- category: filter to one of: boards, create, read, update, delete, tags, members, export, audit, diagrams. Optional.
- limit: max results (default 10, max 50).

Returns up to ` + "`limit`" + ` matches sorted by relevance, with name, category, title, and a short description excerpt. Empty query plus a category returns the category's tools sorted by name.

This is a discovery tool. After picking a tool from the result, call it directly; do not re-route through this search.`,
}

// SearchTools implements the miro_tool_search handler. It is a pure
// in-process search over the registered AllTools list — no Miro API calls,
// no network. Used to keep token cost low when an agent doesn't know
// which of the 90+ tools to reach for.
func (h *HandlerRegistry) SearchTools(_ context.Context, args ToolSearchArgs) (ToolSearchResult, error) {
	matches := scoreTools(args, AllTools)

	msg := buildSearchMessage(args, matches)
	return ToolSearchResult{
		Tools:   matches,
		Total:   len(matches),
		Message: msg,
	}, nil
}

// scoreTools is the deterministic ranking step. Exposed for testability.
func scoreTools(args ToolSearchArgs, tools []ToolSpec) []ToolSearchMatch {
	limit := args.Limit
	switch {
	case limit <= 0:
		limit = 10
	case limit > 50:
		limit = 50
	}

	query := strings.ToLower(strings.TrimSpace(args.Query))
	terms := tokenize(query)
	categoryFilter := strings.ToLower(strings.TrimSpace(args.Category))

	type scored struct {
		spec  ToolSpec
		score float64
	}

	candidates := make([]scored, 0, len(tools))
	for _, t := range tools {
		// Don't recommend the search tool itself; that's a recursion trap.
		if t.Name == ToolSearchName {
			continue
		}
		if categoryFilter != "" && !strings.EqualFold(t.Category, categoryFilter) {
			continue
		}
		s := computeScore(t, terms)
		// When there's no query, fall back to alphabetical within the category.
		if query == "" {
			candidates = append(candidates, scored{spec: t, score: 0})
			continue
		}
		if s == 0 {
			continue
		}
		candidates = append(candidates, scored{spec: t, score: s})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].spec.Name < candidates[j].spec.Name
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	out := make([]ToolSearchMatch, len(candidates))
	for i, c := range candidates {
		out[i] = ToolSearchMatch{
			Name:        c.spec.Name,
			Category:    c.spec.Category,
			Title:       c.spec.Title,
			Description: shortenDescription(c.spec.Description, 200),
			Score:       c.score,
		}
	}
	return out
}

// computeScore weights matches by where the term appears: name hits dominate
// (the agent is most likely to remember a fragment of the name), title and
// category bonuses help the user-facing-intent case, description is the fallback.
// Multi-word queries are OR-scored: each matching term contributes.
func computeScore(spec ToolSpec, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}

	name := strings.ToLower(spec.Name)
	title := strings.ToLower(spec.Title)
	category := strings.ToLower(spec.Category)
	desc := strings.ToLower(spec.Description)

	const (
		nameWeight     = 3.0
		titleWeight    = 2.0
		categoryWeight = 2.5
		descWeight     = 1.0
	)

	var score float64
	for _, term := range terms {
		if term == "" {
			continue
		}
		if strings.Contains(name, term) {
			score += nameWeight
		}
		if strings.Contains(title, term) {
			score += titleWeight
		}
		if category == term {
			score += categoryWeight
		}
		// Description matches are diminishing-returns — count once, not per occurrence,
		// so a single keyword spam in a long description doesn't dominate.
		if strings.Contains(desc, term) {
			score += descWeight
		}
	}
	return score
}

// tokenize splits on whitespace and common punctuation, drops empties and
// very-short noise tokens. Underscores and hyphens are kept as separators
// so "miro_create_sticky" indexes its constituent words.
func tokenize(s string) []string {
	if s == "" {
		return nil
	}
	splitter := func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', ',', '.', ';', ':', '!', '?', '"', '\'', '(', ')', '[', ']', '{', '}', '_', '-', '/':
			return true
		}
		return false
	}
	parts := strings.FieldsFunc(s, splitter)
	out := parts[:0]
	for _, p := range parts {
		if len(p) < 2 {
			continue
		}
		out = append(out, p)
	}
	return out
}

// shortenDescription returns at most maxLen runes of the description's first
// non-empty paragraph. Multi-line descriptions (USE WHEN / FAILS WHEN format)
// would otherwise blow the context budget that miro_tool_search is built to
// protect.
func shortenDescription(desc string, maxLen int) string {
	if desc == "" {
		return ""
	}
	// First non-empty line (skip the lead newline if any).
	firstLine := desc
	if i := strings.IndexAny(desc, "\r\n"); i >= 0 {
		firstLine = strings.TrimSpace(desc[:i])
	}
	firstLine = strings.TrimSpace(firstLine)
	runes := []rune(firstLine)
	if len(runes) <= maxLen {
		return firstLine
	}
	return string(runes[:maxLen]) + "..."
}

// buildSearchMessage produces a short status string for voice/log output.
func buildSearchMessage(args ToolSearchArgs, matches []ToolSearchMatch) string {
	switch {
	case len(matches) == 0 && args.Query != "" && args.Category != "":
		return fmt.Sprintf("No tools matched query %q in category %q", args.Query, args.Category)
	case len(matches) == 0 && args.Query != "":
		return fmt.Sprintf("No tools matched query %q", args.Query)
	case len(matches) == 0 && args.Category != "":
		return fmt.Sprintf("No tools in category %q", args.Category)
	case args.Query == "" && args.Category != "":
		return fmt.Sprintf("Found %d tools in category %q", len(matches), args.Category)
	case args.Category != "":
		return fmt.Sprintf("Found %d tools matching %q in category %q", len(matches), args.Query, args.Category)
	default:
		return fmt.Sprintf("Found %d tools matching %q", len(matches), args.Query)
	}
}
