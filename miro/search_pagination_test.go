package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// itemsCorpus is a board's worth of items for the fake pager to serve.
// contentAt decides what each item says, which is how each test plants its
// matches at a chosen depth.
type itemsCorpus struct {
	total     int
	contentAt func(i int) string
}

// fakeItemsPager mimics Miro's cursor-based /v2/boards/{id}/items paging. The
// cursor is the index of the next item; the live API returns an opaque string
// and omits the field on the last page, which is what terminates the walk.
// It records every page size it was asked for and how many requests it served.
type fakeItemsPager struct {
	requests int
	limits   []int
}

func newFakeItemsPager(t *testing.T, corpus itemsCorpus) (*httptest.Server, *fakeItemsPager) {
	t.Helper()
	pager := &fakeItemsPager{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < MinPagedLimit {
			http.Error(w, "minimum items page size is 10", http.StatusBadRequest)
			return
		}
		start, _ := strconv.Atoi(r.URL.Query().Get("cursor"))

		pager.requests++
		pager.limits = append(pager.limits, limit)

		data := []map[string]any{}
		i := start
		for ; i < start+limit && i < corpus.total; i++ {
			data = append(data, map[string]any{
				"id":       fmt.Sprintf("item%04d", i),
				"type":     "sticky_note",
				"position": map[string]any{"x": float64(i), "y": 0.0},
				"data":     map[string]any{"content": corpus.contentAt(i)},
			})
		}
		body := map[string]any{"data": data, "size": len(data)}
		if i < corpus.total {
			body["cursor"] = strconv.Itoa(i)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)
	return server, pager
}

// filler is board content guaranteed not to contain any test query.
func filler(i int) string { return fmt.Sprintf("ordinary sticky %d", i) }

// Regression: SearchBoard fetched one page and filtered it client-side without
// ever following the cursor, so a match beyond the first page came back as a
// confident "No items found matching X". The old code made exactly one request
// and returned zero matches for this corpus.
func TestSearchBoard_FindsMatchOnLaterPage(t *testing.T) {
	server, pager := newFakeItemsPager(t, itemsCorpus{
		total: 120,
		contentAt: func(i int) string {
			if i == 117 {
				return "the quarterly budget needle"
			}
			return filler(i)
		},
	})
	c := newTestClientWithServer(server.URL)

	res, err := c.SearchBoard(context.Background(), SearchBoardArgs{
		BoardID: "uXjVLmnBBBB=", Query: "needle",
	})
	if err != nil {
		t.Fatalf("SearchBoard: %v", err)
	}
	if res.Count != 1 {
		t.Fatalf("Count = %d, want 1 (the match sits on page 3)", res.Count)
	}
	if res.Matches[0].ID != "item0117" {
		t.Errorf("matched %q, want item0117", res.Matches[0].ID)
	}
	if pager.requests < 2 {
		t.Errorf("served %d requests, want more than one — the cursor was not followed", pager.requests)
	}
	if res.Truncated {
		t.Error("Truncated = true, want false — the board was fully scanned")
	}
	if res.ItemsScanned != 120 {
		t.Errorf("ItemsScanned = %d, want 120", res.ItemsScanned)
	}
}

// The absence claim must only be made about a board that was actually read to
// the end, and the message must say how far the scan got.
func TestSearchBoard_ExhaustedBoardReportsAbsenceHonestly(t *testing.T) {
	server, _ := newFakeItemsPager(t, itemsCorpus{total: 73, contentAt: filler})
	c := newTestClientWithServer(server.URL)

	res, err := c.SearchBoard(context.Background(), SearchBoardArgs{
		BoardID: "uXjVLmnBBBB=", Query: "needle",
	})
	if err != nil {
		t.Fatalf("SearchBoard: %v", err)
	}
	if res.Count != 0 || res.Truncated {
		t.Errorf("Count=%d Truncated=%v, want 0 and false", res.Count, res.Truncated)
	}
	if res.ItemsScanned != 73 {
		t.Errorf("ItemsScanned = %d, want 73", res.ItemsScanned)
	}
	if !strings.Contains(res.Message, "73") {
		t.Errorf("Message = %q, want the scan depth in it", res.Message)
	}
}

// Regression: the caller's limit was forwarded as the API page size, so it
// governed how much of the board was scanned instead of how many matches came
// back. Live, one board and one query gave 5 matches at limit=10 and 13 at
// limit=20. Every item here matches, so a result cap must return exactly the
// cap at every limit, and scan depth must not shrink with it.
func TestSearchBoard_LimitCapsMatchesNotScanDepth(t *testing.T) {
	for _, limit := range []int{10, 20, 50} {
		server, _ := newFakeItemsPager(t, itemsCorpus{
			total:     300,
			contentAt: func(i int) string { return fmt.Sprintf("needle %d", i) },
		})
		c := newTestClientWithServer(server.URL)

		res, err := c.SearchBoard(context.Background(), SearchBoardArgs{
			BoardID: "uXjVLmnBBBB=", Query: "needle", Limit: limit,
		})
		if err != nil {
			t.Fatalf("limit=%d: %v", limit, err)
		}
		if res.Count != limit {
			t.Errorf("limit=%d: Count = %d, want %d", limit, res.Count, limit)
		}
		if !res.Truncated {
			t.Errorf("limit=%d: Truncated = false, want true — 300 items matched", limit)
		}
	}
}

// A limit below the API's minimum page size must still reach deep into the
// board. Under the old code limit=5 was sent as the page size, so a match at
// item 110 was unreachable at any limit small enough to be useful.
func TestSearchBoard_SmallLimitStillScansPastTheFirstPage(t *testing.T) {
	server, pager := newFakeItemsPager(t, itemsCorpus{
		total: 120,
		contentAt: func(i int) string {
			if i >= 110 {
				return fmt.Sprintf("needle %d", i)
			}
			return filler(i)
		},
	})
	c := newTestClientWithServer(server.URL)

	res, err := c.SearchBoard(context.Background(), SearchBoardArgs{
		BoardID: "uXjVLmnBBBB=", Query: "needle", Limit: 5,
	})
	if err != nil {
		t.Fatalf("SearchBoard: %v", err)
	}
	if res.Count != 5 {
		t.Errorf("Count = %d, want 5", res.Count)
	}
	if res.ItemsScanned < 110 {
		t.Errorf("ItemsScanned = %d, want at least 110 — a small limit must not shorten the scan", res.ItemsScanned)
	}
	for _, sent := range pager.limits {
		if sent < MinPagedLimit {
			t.Fatalf("sent page size %d, below the API minimum", sent)
		}
	}
}

// A board larger than the scan cap must report that the scan stopped early
// rather than claiming the text is absent.
func TestSearchBoard_ScanCapStopsBeforeEndOfBoard(t *testing.T) {
	server, _ := newFakeItemsPager(t, itemsCorpus{
		total: DefaultSearchScanItems + 200, contentAt: filler,
	})
	c := newTestClientWithServer(server.URL)

	res, err := c.SearchBoard(context.Background(), SearchBoardArgs{
		BoardID: "uXjVLmnBBBB=", Query: "needle",
	})
	if err != nil {
		t.Fatalf("SearchBoard: %v", err)
	}
	if res.Count != 0 {
		t.Errorf("Count = %d, want 0", res.Count)
	}
	if !res.Truncated {
		t.Error("Truncated = false, want true — the scan cap was reached")
	}
	if res.ItemsScanned != DefaultSearchScanItems {
		t.Errorf("ItemsScanned = %d, want %d", res.ItemsScanned, DefaultSearchScanItems)
	}
	if strings.HasPrefix(res.Message, "No items found matching") {
		t.Errorf("Message = %q, want it to disclose the partial scan rather than assert absence", res.Message)
	}
	if !strings.Contains(res.Message, "scan cap") {
		t.Errorf("Message = %q, want the scan cap named", res.Message)
	}
}

// An item past the scan cap is not reachable, which is the honest limit of the
// fix: Truncated is the caller's signal to narrow the search.
func TestSearchBoard_MatchBeyondScanCapIsReportedTruncated(t *testing.T) {
	server, _ := newFakeItemsPager(t, itemsCorpus{
		total: DefaultSearchScanItems + 200,
		contentAt: func(i int) string {
			if i == DefaultSearchScanItems+100 {
				return "far away needle"
			}
			return filler(i)
		},
	})
	c := newTestClientWithServer(server.URL)

	res, err := c.SearchBoard(context.Background(), SearchBoardArgs{
		BoardID: "uXjVLmnBBBB=", Query: "needle",
	})
	if err != nil {
		t.Fatalf("SearchBoard: %v", err)
	}
	if res.Count != 0 || !res.Truncated {
		t.Errorf("Count=%d Truncated=%v, want 0 and true", res.Count, res.Truncated)
	}
}

// The type filter must survive the move to a paged scan.
func TestSearchBoard_ForwardsTypeFilterOnEveryPage(t *testing.T) {
	var types []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		types = append(types, r.URL.Query().Get("type"))
		body := map[string]any{"data": []any{}, "size": 0}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	if _, err := c.SearchBoard(context.Background(), SearchBoardArgs{
		BoardID: "uXjVLmnBBBB=", Query: "needle", Type: "sticky_note",
	}); err != nil {
		t.Fatalf("SearchBoard: %v", err)
	}
	if len(types) == 0 {
		t.Fatal("no request reached the server")
	}
	for i, got := range types {
		if got != "sticky_note" {
			t.Errorf("request %d sent type=%q, want sticky_note", i, got)
		}
	}
}

// An empty board is exhausted, not truncated.
func TestSearchBoard_EmptyBoardIsExhausted(t *testing.T) {
	server, _ := newFakeItemsPager(t, itemsCorpus{total: 0, contentAt: filler})
	c := newTestClientWithServer(server.URL)

	res, err := c.SearchBoard(context.Background(), SearchBoardArgs{
		BoardID: "uXjVLmnBBBB=", Query: "needle",
	})
	if err != nil {
		t.Fatalf("SearchBoard: %v", err)
	}
	if res.Truncated || res.ItemsScanned != 0 || res.Count != 0 {
		t.Errorf("Truncated=%v ItemsScanned=%d Count=%d, want false, 0, 0",
			res.Truncated, res.ItemsScanned, res.Count)
	}
}

func TestSearchBoardLimit(t *testing.T) {
	for _, tc := range []struct{ in, want int }{
		{0, DefaultSearchLimit}, {-1, DefaultSearchLimit},
		{1, 1}, {10, 10}, {50, MaxSearchLimit}, {51, DefaultSearchLimit},
	} {
		if got := searchBoardLimit(tc.in); got != tc.want {
			t.Errorf("searchBoardLimit(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
