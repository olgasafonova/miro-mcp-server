package tools

import (
	"context"
	"strings"
	"testing"
)

// fixture used so tests don't depend on the volatile real AllTools list.
var searchTestTools = []ToolSpec{
	{
		Name:        "miro_create_sticky",
		Method:      "CreateSticky",
		Title:       "Create Sticky Note",
		Category:    "create",
		Description: "Add a sticky note to a Miro board with text content and optional position.",
	},
	{
		Name:        "miro_create_card",
		Method:      "CreateCard",
		Title:       "Create Card",
		Category:    "create",
		Description: "Create a card item with title and description on a board.",
	},
	{
		Name:        "miro_share_board",
		Method:      "ShareBoard",
		Title:       "Share Board",
		Category:    "members",
		Destructive: true,
		Description: "Invite an email to a Miro board with role (viewer, commenter, editor).",
	},
	{
		Name:        "miro_list_items",
		Method:      "ListItems",
		Title:       "List Items",
		Category:    "read",
		ReadOnly:    true,
		Description: "List items on a board, optionally filtered by type or tag.",
	},
	{
		Name:        "miro_delete_item",
		Method:      "DeleteItem",
		Title:       "Delete Item",
		Category:    "delete",
		Destructive: true,
		Description: "Permanently delete an item by ID. Cannot be undone.",
	},
}

func TestScoreTools_NameMatchDominates(t *testing.T) {
	got := scoreTools(ToolSearchArgs{Query: "sticky"}, searchTestTools)
	if len(got) == 0 || got[0].Name != "miro_create_sticky" {
		t.Fatalf("expected miro_create_sticky first, got %+v", names(got))
	}
}

func TestScoreTools_CategoryFilter(t *testing.T) {
	got := scoreTools(ToolSearchArgs{Query: "create", Category: "create"}, searchTestTools)
	if len(got) != 2 {
		t.Fatalf("expected 2 create-category hits, got %d: %+v", len(got), names(got))
	}
	for _, m := range got {
		if m.Category != "create" {
			t.Errorf("non-create category leaked through: %s/%s", m.Name, m.Category)
		}
	}
}

func TestScoreTools_EmptyQueryWithCategoryReturnsAll(t *testing.T) {
	got := scoreTools(ToolSearchArgs{Category: "create"}, searchTestTools)
	if len(got) != 2 {
		t.Fatalf("expected 2 create tools when query empty, got %d", len(got))
	}
}

func TestScoreTools_NoMatchReturnsEmpty(t *testing.T) {
	got := scoreTools(ToolSearchArgs{Query: "completely_unrelated_term_xyzzy"}, searchTestTools)
	if len(got) != 0 {
		t.Errorf("expected 0 matches, got %d: %+v", len(got), names(got))
	}
}

func TestScoreTools_LimitDefault(t *testing.T) {
	// Build a fixture larger than the default limit (10).
	manyTools := make([]ToolSpec, 0, 15)
	for i := 0; i < 15; i++ {
		manyTools = append(manyTools, ToolSpec{
			Name:        "miro_match_" + string(rune('a'+i)),
			Category:    "read",
			Description: "matches the query foo",
		})
	}
	got := scoreTools(ToolSearchArgs{Query: "foo"}, manyTools)
	if len(got) != 10 {
		t.Errorf("expected default limit 10, got %d", len(got))
	}
}

func TestScoreTools_LimitCappedAt50(t *testing.T) {
	got := scoreTools(ToolSearchArgs{Query: "anything", Limit: 9999}, searchTestTools)
	// Limit doesn't add tools, just doesn't exceed 50; the test fixture has 5.
	if len(got) > 50 {
		t.Errorf("expected limit cap at 50, got %d", len(got))
	}
}

func TestScoreTools_DoesNotRecommendSelf(t *testing.T) {
	tools := append([]ToolSpec{SearchToolSpec}, searchTestTools...)
	got := scoreTools(ToolSearchArgs{Query: "tool"}, tools)
	for _, m := range got {
		if m.Name == ToolSearchName {
			t.Errorf("search tool should not recommend itself, got %s in results", m.Name)
		}
	}
}

func TestScoreTools_MultiTermQuery(t *testing.T) {
	got := scoreTools(ToolSearchArgs{Query: "delete item"}, searchTestTools)
	if len(got) == 0 || got[0].Name != "miro_delete_item" {
		t.Fatalf("expected miro_delete_item first for 'delete item', got %+v", names(got))
	}
}

func TestScoreTools_DescriptionShortened(t *testing.T) {
	longTools := []ToolSpec{
		{
			Name:     "miro_create_long",
			Method:   "CreateLong",
			Title:    "Long",
			Category: "create",
			Description: "Short summary line.\n\n" +
				"USE WHEN: very long second paragraph that should not propagate through " +
				strings.Repeat("X", 500),
		},
	}
	got := scoreTools(ToolSearchArgs{Query: "create"}, longTools)
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d", len(got))
	}
	if strings.Contains(got[0].Description, "USE WHEN") {
		t.Errorf("description shortener should have stopped at first newline: %q", got[0].Description)
	}
	if len([]rune(got[0].Description)) > 200 {
		t.Errorf("description longer than cap: %d runes", len([]rune(got[0].Description)))
	}
}

func TestSearchTools_ReturnsStructuredResult(t *testing.T) {
	h := &HandlerRegistry{}
	res, err := h.SearchTools(context.Background(), ToolSearchArgs{Query: "sticky"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Total != len(res.Tools) {
		t.Errorf("Total %d != len(Tools) %d", res.Total, len(res.Tools))
	}
	if res.Message == "" {
		t.Errorf("expected non-empty message")
	}
}

func TestTokenize_FiltersShortTokens(t *testing.T) {
	got := tokenize("a b ab create_sticky")
	want := []string{"ab", "create", "sticky"}
	if len(got) != len(want) {
		t.Fatalf("tokenize = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("token[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func names(matches []ToolSearchMatch) []string {
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = m.Name
	}
	return out
}
