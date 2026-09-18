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

// Regression: SearchBoard forwarded the caller's result limit as the API page
// size, so any limit below 10 returned a bare HTTP 400.
func TestSearchBoard_SmallLimitIsRaisedNotRejected(t *testing.T) {
	for _, lim := range []int{1, 5, 9, 10, 20} {
		var seen int
		server := strictPageServer(t, &seen)
		c := newTestClientWithServer(server.URL)
		_, err := c.SearchBoard(context.Background(), SearchBoardArgs{
			BoardID: "uXjVLmnBBBB=", Query: "x", Limit: lim,
		})
		server.Close()
		if err != nil {
			t.Errorf("limit=%d: %v (sent limit=%d)", lim, err, seen)
		}
		if seen < MinPagedLimit {
			t.Errorf("limit=%d: sent %d to the API, below the minimum", lim, seen)
		}
	}
}

func TestListItems_SmallLimitIsRaisedNotRejected(t *testing.T) {
	for _, lim := range []int{1, 5, 9, 10} {
		var seen int
		server := strictPageServer(t, &seen)
		c := newTestClientWithServer(server.URL)
		_, err := c.ListItems(context.Background(), ListItemsArgs{BoardID: "uXjVLmnBBBB=", Limit: lim})
		server.Close()
		if err != nil {
			t.Errorf("limit=%d: %v (sent limit=%d)", lim, err, seen)
		}
		if seen < MinPagedLimit {
			t.Errorf("limit=%d: sent %d to the API, below the minimum", lim, seen)
		}
	}
}

func TestGetFrameItems_SmallLimitIsRaised(t *testing.T) {
	var seen int
	server := strictPageServer(t, &seen)
	defer server.Close()
	c := newTestClientWithServer(server.URL)
	if _, err := c.GetFrameItems(context.Background(), GetFrameItemsArgs{
		BoardID: "uXjVLmnBBBB=", FrameID: "123", Limit: 3,
	}); err != nil {
		t.Errorf("GetFrameItems limit=3: %v (sent %d)", err, seen)
	}
	if seen < MinPagedLimit {
		t.Errorf("sent limit=%d, below the minimum", seen)
	}
}

func TestListGroups_SmallLimitIsRaised(t *testing.T) {
	var seen int
	server := strictPageServer(t, &seen)
	defer server.Close()
	c := newTestClientWithServer(server.URL)
	if _, err := c.ListGroups(context.Background(), ListGroupsArgs{BoardID: "uXjVLmnBBBB=", Limit: 2}); err != nil {
		t.Errorf("ListGroups limit=2: %v (sent %d)", err, seen)
	}
	if seen < MinPagedLimit {
		t.Errorf("sent limit=%d, below the minimum", seen)
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
	c := newTestClientWithServer(server.URL)
	if _, err := c.ListBoardMembers(context.Background(), ListBoardMembersArgs{
		BoardID: "uXjVLmnBBBB=", Limit: 5,
	}); err != nil {
		t.Fatalf("ListBoardMembers: %v", err)
	}
	if seen != 5 {
		t.Errorf("sent limit=%d, want 5 — the items floor must not leak onto members", seen)
	}
}
