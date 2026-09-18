package miro

import (
	"context"
	"fmt"
	"strconv"
	"testing"
)

// membersEndpoint plugs /v2/boards/{id}/members into the shared offset suite
// in pagination_testkit_test.go. Like the boards endpoint it echoes back the
// offset of the page it served (verified against api.miro.com on 18-09-2026:
// offset=1 answers offset=1), which is the behaviour that made the previous
// HasMore computation wrong on page one.
var membersEndpoint = offsetEndpoint{
	noun: "member",
	row: func(i int) map[string]any {
		return map[string]any{
			"id":    fmt.Sprintf("member%d", i),
			"name":  fmt.Sprintf("User %d", i),
			"email": fmt.Sprintf("user%d@example.com", i),
			"role":  "viewer",
		}
	},
	call: func(c *Client, limit int, offset string) (offsetPage, error) {
		res, err := c.ListBoardMembers(context.Background(), ListBoardMembersArgs{
			BoardID: "board123", Limit: limit, Offset: offset,
		})
		if err != nil {
			return offsetPage{}, err
		}
		ids := make([]string, len(res.Members))
		for i, m := range res.Members {
			ids[i] = m.ID
		}
		return offsetPage{ids: ids, hasMore: res.HasMore, nextOffset: res.Offset, total: res.Total}, nil
	},
}

// Regression: HasMore was `resp.Offset > 0 && ...`, always false on page one
// because the echoed offset is 0, and the result carried no offset field for
// the caller to follow even once HasMore was right.
func TestListBoardMembers_Pagination(t *testing.T) {
	runOffsetPaginationContract(t, membersEndpoint, true)
}

func TestListBoardMembers_FullWalkTerminatesWithoutDuplicates(t *testing.T) {
	runOffsetFullWalk(t, membersEndpoint, 137, 50)
}

func TestListBoardMembers_RejectsInvalidOffset(t *testing.T) {
	server := fakeOffsetPager(t, membersEndpoint, 3, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	for _, bad := range []string{"abc", "-1", "1.5"} {
		_, err := c.ListBoardMembers(context.Background(), ListBoardMembersArgs{BoardID: "board123", Offset: bad})
		if err == nil {
			t.Errorf("offset %q: expected an error, got nil", bad)
		}
	}
}

// The members endpoint has no minimum page size, so a caller asking for one
// member gets one. Guards the deliberate absence of a floor in clampMemberLimit
// against a future sweep that generalizes the items floor across the client.
func TestListBoardMembers_HonoursSubMinimumPageSize(t *testing.T) {
	server := fakeOffsetPager(t, membersEndpoint, 3, false)
	defer server.Close()

	page, err := membersEndpoint.call(newTestClientWithServer(server.URL), 1, "")
	if err != nil {
		t.Fatalf("ListBoardMembers: %v", err)
	}
	if len(page.ids) != 1 {
		t.Errorf("returned %d members, want 1", len(page.ids))
	}
	if !page.hasMore || page.nextOffset != strconv.Itoa(1) {
		t.Errorf("HasMore=%v next=%q, want true and \"1\"", page.hasMore, page.nextOffset)
	}
}
