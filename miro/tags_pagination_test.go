package miro

import (
	"context"
	"fmt"
	"strconv"
	"testing"
)

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
		n := 0
		if offset != "" {
			parsed, err := strconv.Atoi(offset)
			if err != nil {
				return offsetPage{}, err
			}
			n = parsed
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
		next := ""
		if res.Offset != 0 {
			next = strconv.Itoa(res.Offset)
		}
		return offsetPage{ids: ids, hasMore: res.HasMore, nextOffset: next, total: res.Total}, nil
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
