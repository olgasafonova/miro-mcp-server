package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// fakeBoardsPager mimics Miro's /v2/boards offset paging. The live API echoes
// back the offset of the page it served (verified against api.miro.com:
// request offset=1 returns offset=1), which is the behaviour that made the
// previous HasMore computation wrong on the first page.
func fakeBoardsPager(t *testing.T, total int, omitTotal bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		data := []map[string]any{}
		for i := offset; i < offset+limit && i < total; i++ {
			data = append(data, map[string]any{
				"id":   fmt.Sprintf("uXjVL%09d=", i),
				"name": fmt.Sprintf("Board %d", i),
			})
		}
		body := map[string]any{
			"data":   data,
			"size":   len(data),
			"offset": offset,
		}
		if !omitTotal {
			body["total"] = total
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
}

// TestListBoards_Pagination covers how one page reports the existence of the
// next. The regressions guarded here are that HasMore was `resp.Offset > 0 &&
// ...`, always false on page one because the echoed offset is 0, and that the
// cursor returned was the offset just requested rather than the following one.
func TestListBoards_Pagination(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		omitTotal   bool
		limit       int
		offset      string
		wantHasMore bool
		wantOffset  string
		wantTotal   int
	}{
		{
			name:  "full first page of a larger collection advertises more",
			total: 500, limit: 50,
			wantHasMore: true, wantOffset: "50", wantTotal: 500,
		},
		{
			name:  "second page advances the cursor rather than repeating it",
			total: 500, limit: 50, offset: "50",
			wantHasMore: true, wantOffset: "100", wantTotal: 500,
		},
		{
			name:  "collection fitting in one page is terminal",
			total: 13, limit: 50,
			wantHasMore: false, wantOffset: "", wantTotal: 13,
		},
		{
			name:  "empty collection is terminal",
			total: 0, limit: 50,
			wantHasMore: false, wantOffset: "", wantTotal: 0,
		},
		{
			name:  "offset past the end is terminal",
			total: 13, limit: 5, offset: "100",
			wantHasMore: false, wantOffset: "", wantTotal: 13,
		},
		{
			// total is not marked required in Miro's OpenAPI spec, so its
			// absence must not be read as the end of the collection.
			name:  "absent total falls back to the full-page heuristic",
			total: 500, omitTotal: true, limit: 50,
			wantHasMore: true, wantOffset: "50",
		},
		{
			name:  "absent total with a short page is terminal",
			total: 13, omitTotal: true, limit: 50,
			wantHasMore: false, wantOffset: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := fakeBoardsPager(t, tt.total, tt.omitTotal)
			defer server.Close()

			res, err := newTestClientWithServer(server.URL).ListBoards(
				context.Background(),
				ListBoardsArgs{Limit: tt.limit, Offset: tt.offset},
			)
			if err != nil {
				t.Fatalf("ListBoards: %v", err)
			}
			if res.HasMore != tt.wantHasMore {
				t.Errorf("HasMore = %v, want %v", res.HasMore, tt.wantHasMore)
			}
			if res.Offset != tt.wantOffset {
				t.Errorf("Offset = %q, want %q", res.Offset, tt.wantOffset)
			}
			if res.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", res.Total, tt.wantTotal)
			}
		})
	}
}

// walkAllBoards pages from the start and returns the ids seen, in order of
// arrival. It fails the test rather than returning an error, so the caller
// stays a straight-line assertion.
func walkAllBoards(t *testing.T, c *Client, limit int) []string {
	t.Helper()

	var ids []string
	offset := ""
	for pages := 0; pages <= 50; pages++ {
		res, err := c.ListBoards(context.Background(), ListBoardsArgs{Limit: limit, Offset: offset})
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		for _, b := range res.Boards {
			ids = append(ids, b.ID)
		}
		if !res.HasMore {
			return ids
		}
		if res.Offset == offset {
			t.Fatalf("cursor stalled at %q — following it would refetch the same page", offset)
		}
		offset = res.Offset
	}
	t.Fatal("pagination did not terminate within 50 pages")
	return nil
}

// Walking every page must terminate and yield each board exactly once.
// Regression: the cursor used to echo the requested offset, so a caller that
// followed it refetched the same page forever.
func TestListBoards_FullWalkTerminatesWithoutDuplicates(t *testing.T) {
	const total = 137
	server := fakeBoardsPager(t, total, false)
	defer server.Close()

	ids := walkAllBoards(t, newTestClientWithServer(server.URL), 50)

	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			t.Fatalf("duplicate board %s — cursor did not advance", id)
		}
		seen[id] = true
	}
	if len(seen) != total {
		t.Errorf("collected %d boards, want %d", len(seen), total)
	}
}

// A non-numeric or negative offset is rejected instead of silently becoming 0.
func TestListBoards_RejectsInvalidOffset(t *testing.T) {
	server := fakeBoardsPager(t, 13, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	for _, bad := range []string{"abc", "-1", "1.5"} {
		if _, err := c.ListBoards(context.Background(), ListBoardsArgs{Offset: bad}); err == nil {
			t.Errorf("offset %q: expected an error, got nil", bad)
		}
	}
}
