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

// A full first page of a larger collection must advertise more and hand back
// a usable cursor. Regression: HasMore was `resp.Offset > 0 && ...`, which is
// always false on page 1 because the echoed offset is 0.
func TestListBoards_FirstPageAdvertisesMore(t *testing.T) {
	server := fakeBoardsPager(t, 500, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListBoards(context.Background(), ListBoardsArgs{Limit: 50})
	if err != nil {
		t.Fatalf("ListBoards: %v", err)
	}
	if !res.HasMore {
		t.Errorf("HasMore = false, want true (50 of 500 boards returned)")
	}
	if res.Offset != "50" {
		t.Errorf("Offset = %q, want \"50\"", res.Offset)
	}
	if res.Total != 500 {
		t.Errorf("Total = %d, want 500", res.Total)
	}
}

// Walking every page must terminate and yield each board exactly once.
// Regression: the cursor used to echo the requested offset, so a caller that
// followed it refetched the same page forever.
func TestListBoards_FullWalkTerminatesWithoutDuplicates(t *testing.T) {
	const total = 137
	server := fakeBoardsPager(t, total, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	seen := map[string]bool{}
	offset := ""
	for pages := 0; ; pages++ {
		if pages > 50 {
			t.Fatal("pagination did not terminate")
		}
		res, err := c.ListBoards(context.Background(), ListBoardsArgs{Limit: 50, Offset: offset})
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		for _, b := range res.Boards {
			if seen[b.ID] {
				t.Fatalf("duplicate board %s — cursor did not advance", b.ID)
			}
			seen[b.ID] = true
		}
		if !res.HasMore {
			break
		}
		if res.Offset == offset {
			t.Fatalf("cursor stalled at %q", offset)
		}
		offset = res.Offset
	}
	if len(seen) != total {
		t.Errorf("collected %d boards, want %d", len(seen), total)
	}
}

// A collection that fits in one page is terminal.
func TestListBoards_SinglePageIsTerminal(t *testing.T) {
	server := fakeBoardsPager(t, 13, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListBoards(context.Background(), ListBoardsArgs{Limit: 50})
	if err != nil {
		t.Fatalf("ListBoards: %v", err)
	}
	if res.HasMore || res.Offset != "" {
		t.Errorf("HasMore=%v Offset=%q, want false and \"\"", res.HasMore, res.Offset)
	}
}

// An empty collection must not advertise more.
func TestListBoards_EmptyIsTerminal(t *testing.T) {
	server := fakeBoardsPager(t, 0, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListBoards(context.Background(), ListBoardsArgs{Limit: 50})
	if err != nil {
		t.Fatalf("ListBoards: %v", err)
	}
	if res.HasMore || res.Count != 0 {
		t.Errorf("HasMore=%v Count=%d, want false and 0", res.HasMore, res.Count)
	}
}

// An offset past the end is terminal rather than looping.
func TestListBoards_OffsetPastEndIsTerminal(t *testing.T) {
	server := fakeBoardsPager(t, 13, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListBoards(context.Background(), ListBoardsArgs{Limit: 5, Offset: "100"})
	if err != nil {
		t.Fatalf("ListBoards: %v", err)
	}
	if res.HasMore {
		t.Errorf("HasMore = true past the end, want false")
	}
}

// `total` is not marked required in Miro's OpenAPI spec. When it is absent we
// fall back to the full-page heuristic rather than reporting a false end.
func TestListBoards_FallsBackWhenTotalAbsent(t *testing.T) {
	server := fakeBoardsPager(t, 500, true)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListBoards(context.Background(), ListBoardsArgs{Limit: 50})
	if err != nil {
		t.Fatalf("ListBoards: %v", err)
	}
	if !res.HasMore {
		t.Error("HasMore = false with total absent and a full page, want true")
	}
	if res.Offset != "50" {
		t.Errorf("Offset = %q, want \"50\"", res.Offset)
	}
}

// A short page with no total is the end of the collection.
func TestListBoards_ShortPageWithoutTotalIsTerminal(t *testing.T) {
	server := fakeBoardsPager(t, 13, true)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListBoards(context.Background(), ListBoardsArgs{Limit: 50})
	if err != nil {
		t.Fatalf("ListBoards: %v", err)
	}
	if res.HasMore {
		t.Error("HasMore = true on a short page with no total, want false")
	}
}

// A non-numeric offset is rejected instead of silently becoming 0.
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
