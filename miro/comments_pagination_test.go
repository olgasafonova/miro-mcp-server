package miro

import (
	"context"
	"fmt"
	"strconv"
	"testing"
)

// commentsEndpoint plugs the v2-experimental comments listing into the shared
// offset suite. ListComments computed has_more correctly all along and still
// stranded the caller, because the result exposed no offset to follow.
var commentsEndpoint = offsetEndpoint{
	noun: "comment thread",
	row: func(i int) map[string]any {
		return map[string]any{
			"id":       fmt.Sprintf("thread%d", i),
			"resolved": false,
			"messages": []map[string]any{{"id": "m1", "content": fmt.Sprintf("comment %d", i)}},
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
		res, err := c.ListComments(context.Background(), ListCommentsArgs{
			BoardID: "board123", Limit: limit, Offset: n,
		})
		if err != nil {
			return offsetPage{}, err
		}
		ids := make([]string, len(res.Comments))
		for i, cm := range res.Comments {
			ids[i] = cm.ID
		}
		next := ""
		if res.Offset != 0 {
			next = strconv.Itoa(res.Offset)
		}
		return offsetPage{ids: ids, hasMore: res.HasMore, nextOffset: next, total: res.Total}, nil
	},
}

func TestListComments_Pagination(t *testing.T) {
	runOffsetPaginationContract(t, commentsEndpoint, true)
}

func TestListComments_FullWalkTerminatesWithoutDuplicates(t *testing.T) {
	runOffsetFullWalk(t, commentsEndpoint, 137, 50)
}

func TestListComments_RejectsNegativeOffset(t *testing.T) {
	server := fakeOffsetPager(t, commentsEndpoint, 3, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	if _, err := c.ListComments(context.Background(), ListCommentsArgs{BoardID: "board123", Offset: -1}); err == nil {
		t.Error("offset -1: expected an error, got nil")
	}
}
