package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// strictPageServer rejects a page size below MinPagedLimit the way Miro's
// items, connectors and groups endpoints do (HTTP 400, error code 2.0703,
// "minimum items page size is 10"). It records the limit it was asked for.
func strictPageServer(t *testing.T, seen *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		*seen = limit
		if limit < 10 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "error", "code": "2.0703", "status": 400,
				"message": "Invalid parameters",
				"context": map[string]any{"fields": []map[string]string{
					{"field": "limit", "message": "minimum items page size is 10"},
				}},
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}, "total": 0, "size": 0, "offset": 0})
	}))
}

func TestAtLeastMinPage(t *testing.T) {
	for _, tc := range []struct{ in, want int }{
		{0, MinPagedLimit}, {1, MinPagedLimit}, {9, MinPagedLimit},
		{10, 10}, {20, 20}, {100, 100},
	} {
		if got := atLeastMinPage(tc.in); got != tc.want {
			t.Errorf("atLeastMinPage(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// TestFlooredEndpoints_SmallLimitIsRaisedNotRejected covers every call path
// that reaches an endpoint enforcing the minimum page size. The regression is
// that each forwarded the caller's limit unclamped, so any value from 1 to 9
// came back as a bare HTTP 400 rather than a short page.
//
// GetFrameItems is included because it reaches the items endpoint through
// parent_item_id rather than a frames path of its own.
func TestFlooredEndpoints_SmallLimitIsRaisedNotRejected(t *testing.T) {
	const board = "uXjVLmnBBBB="

	calls := map[string]func(*Client, int) error{
		"SearchBoard": func(c *Client, limit int) error {
			_, err := c.SearchBoard(context.Background(), SearchBoardArgs{
				BoardID: board, Query: "x", Limit: limit,
			})
			return err
		},
		"ListItems": func(c *Client, limit int) error {
			_, err := c.ListItems(context.Background(), ListItemsArgs{BoardID: board, Limit: limit})
			return err
		},
		"GetFrameItems": func(c *Client, limit int) error {
			_, err := c.GetFrameItems(context.Background(), GetFrameItemsArgs{
				BoardID: board, FrameID: "123", Limit: limit,
			})
			return err
		},
		"ListGroups": func(c *Client, limit int) error {
			_, err := c.ListGroups(context.Background(), ListGroupsArgs{BoardID: board, Limit: limit})
			return err
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			for _, limit := range []int{1, 5, 9, 10, 20} {
				var seen int
				server := strictPageServer(t, &seen)
				err := call(newTestClientWithServer(server.URL), limit)
				server.Close()

				if err != nil {
					t.Errorf("limit=%d: %v (sent limit=%d)", limit, err, seen)
				}
				if seen < MinPagedLimit {
					t.Errorf("limit=%d: sent %d to the API, below the minimum", limit, seen)
				}
			}
		})
	}
}

// The members endpoint accepts a page size below 10, so the floor must not be
// applied there: a caller asking for five members receives five, not ten.
func TestClampMemberLimit_NoFloor(t *testing.T) {
	for _, tc := range []struct{ in, want int }{
		{1, 1}, {5, 5}, {9, 9}, {50, 50}, {0, DefaultItemLimit},
	} {
		if got := clampMemberLimit(tc.in); got != tc.want {
			t.Errorf("clampMemberLimit(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestListBoardMembers_SmallLimitIsForwardedUnchanged(t *testing.T) {
	var seen int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = strconv.Atoi(r.URL.Query().Get("limit"))
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}, "total": 0, "offset": 0})
	}))
	defer server.Close()

	if _, err := newTestClientWithServer(server.URL).ListBoardMembers(
		context.Background(),
		ListBoardMembersArgs{BoardID: "uXjVLmnBBBB=", Limit: 5},
	); err != nil {
		t.Fatalf("ListBoardMembers: %v", err)
	}
	if seen != 5 {
		t.Errorf("sent limit=%d, want 5 — the items floor must not leak onto members", seen)
	}
}
