package miro

import (
	"context"
	"fmt"
	"strconv"
	"testing"
)

// Both tag routes take their offset as an int while the shared harness passes
// the cursor as a string, so the two adapters below convert either way through
// these helpers rather than each carrying its own copy of the conversion.

// tagOffsetArg reads the harness cursor as the int offset the tag routes take.
// An empty cursor is the first page.
func tagOffsetArg(cursor string) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	return strconv.Atoi(cursor)
}

// tagOffsetCursor renders a returned offset back as a harness cursor, with ""
// for the end of the collection.
func tagOffsetCursor(offset int) string {
	if offset == 0 {
		return ""
	}
	return strconv.Itoa(offset)
}

// tagItemsEndpoint plugs /v2/boards/{id}/items?tag_id=... into the shared
// offset suite. Probed live on 18-09-2026: despite sharing the /items path
// with the cursor-based item listing, the tag route is offset-based, echoes
// the offset it was given, and reports total.
var tagItemsEndpoint = offsetEndpoint{
	noun: "tagged item",
	row: func(i int) map[string]any {
		return map[string]any{
			"id":   fmt.Sprintf("item%d", i),
			"type": "sticky_note",
			"data": map[string]any{"content": fmt.Sprintf("tagged %d", i)},
		}
	},
	call: func(c *Client, limit int, offset string) (offsetPage, error) {
		n, err := tagOffsetArg(offset)
		if err != nil {
			return offsetPage{}, err
		}
		res, err := c.GetItemsByTag(context.Background(), GetItemsByTagArgs{
			BoardID: "board123", TagID: "tag456", Limit: limit, Offset: n,
		})
		if err != nil {
			return offsetPage{}, err
		}
		ids := make([]string, len(res.Items))
		for i, it := range res.Items {
			ids[i] = it.ID
		}
		return offsetPage{
			ids:        ids,
			hasMore:    res.HasMore,
			nextOffset: tagOffsetCursor(res.Offset),
			total:      res.Total,
		}, nil
	},
}

// boardTagsEndpoint plugs /v2/boards/{id}/tags into the same suite. Probed
// live on 18-09-2026: offset-based, echoes the offset it was given, reports
// total, and accepts limit=1 — it carries no minimum page size, so no floor
// is applied to the requested limit.
var boardTagsEndpoint = offsetEndpoint{
	noun: "tag",
	row: func(i int) map[string]any {
		return map[string]any{
			"id":        fmt.Sprintf("tag%d", i),
			"title":     fmt.Sprintf("Tag %d", i),
			"fillColor": "red",
		}
	},
	call: func(c *Client, limit int, offset string) (offsetPage, error) {
		n, err := tagOffsetArg(offset)
		if err != nil {
			return offsetPage{}, err
		}
		res, err := c.ListTags(context.Background(), ListTagsArgs{
			BoardID: "board123", Limit: limit, Offset: n,
		})
		if err != nil {
			return offsetPage{}, err
		}
		ids := make([]string, len(res.Tags))
		for i, tag := range res.Tags {
			ids[i] = tag.ID
		}
		return offsetPage{
			ids:        ids,
			hasMore:    res.HasMore,
			nextOffset: tagOffsetCursor(res.Offset),
			total:      res.Total,
		}, nil
	},
}

// The contract table includes the exact-multiple-of-limit case, which is the
// boundary the previous `len(items) >= limit` heuristic got wrong: a full
// final page was reported as evidence of another page.
func TestGetItemsByTag_Pagination(t *testing.T) {
	runOffsetPaginationContract(t, tagItemsEndpoint, true)
}

func TestGetItemsByTag_FullWalkTerminatesWithoutDuplicates(t *testing.T) {
	runOffsetFullWalk(t, tagItemsEndpoint, 137, 50)
}

func TestGetItemsByTag_RejectsNegativeOffset(t *testing.T) {
	server := fakeOffsetPager(t, tagItemsEndpoint, 3, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	args := GetItemsByTagArgs{BoardID: "board123", TagID: "tag456", Limit: 10, Offset: -1}
	if _, err := c.GetItemsByTag(context.Background(), args); err == nil {
		t.Error("offset -1: expected an error, got nil")
	}
}

// Before this suite ListTags sent a limit and read back only the data array,
// so a board with more tags than one page presented as having exactly one
// page and no way to ask for the rest.
func TestListTags_Pagination(t *testing.T) {
	runOffsetPaginationContract(t, boardTagsEndpoint, true)
}

func TestListTags_FullWalkTerminatesWithoutDuplicates(t *testing.T) {
	runOffsetFullWalk(t, boardTagsEndpoint, 137, 50)
}

func TestListTags_RejectsNegativeOffset(t *testing.T) {
	server := fakeOffsetPager(t, boardTagsEndpoint, 3, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	args := ListTagsArgs{BoardID: "board123", Limit: 10, Offset: -1}
	if _, err := c.ListTags(context.Background(), args); err == nil {
		t.Error("offset -1: expected an error, got nil")
	}
}

// The tags endpoint answers limit=1 with HTTP 200, so the requested page size
// reaches Miro untouched. A minimum-page-size floor of the kind the items,
// connectors and groups routes need would show up here as a page longer than
// the caller asked for.
func TestListTags_SendsSmallLimitWithoutAFloor(t *testing.T) {
	server := fakeOffsetPager(t, boardTagsEndpoint, 5, false)
	defer server.Close()

	res, err := newTestClientWithServer(server.URL).ListTags(context.Background(), ListTagsArgs{
		BoardID: "board123", Limit: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Count != 1 {
		t.Errorf("Count = %d, want 1 — the requested limit did not reach the API", res.Count)
	}
	if !res.HasMore || res.Offset != 1 {
		t.Errorf("HasMore/Offset = %v/%d, want true/1", res.HasMore, res.Offset)
	}
}
