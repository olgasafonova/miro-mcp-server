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

// fakeCommentsPager mimics the offset paging of the v2-experimental comments
// endpoint, echoing back the offset it was given the way the rest of the
// offset-based Miro endpoints do.
func fakeCommentsPager(t *testing.T, total int, omitTotal bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		data := []map[string]any{}
		for i := offset; i < offset+limit && i < total; i++ {
			data = append(data, map[string]any{
				"id":       fmt.Sprintf("thread%d", i),
				"resolved": false,
				"messages": []map[string]any{{"id": "m1", "content": fmt.Sprintf("comment %d", i)}},
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

// ListComments computed has_more correctly all along and still stranded the
// caller, because the result exposed no offset to follow.
func TestListComments_FirstPageAdvertisesMore(t *testing.T) {
	server := fakeCommentsPager(t, 120, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListComments(context.Background(), ListCommentsArgs{BoardID: "board123", Limit: 50})
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if !res.HasMore {
		t.Error("HasMore = false, want true (50 of 120 threads returned)")
	}
	if res.Offset != 50 {
		t.Errorf("Offset = %d, want 50", res.Offset)
	}
	if res.Total != 120 {
		t.Errorf("Total = %d, want 120", res.Total)
	}
}

// Walking every page must terminate and yield each thread exactly once.
func TestListComments_FullWalkTerminatesWithoutDuplicates(t *testing.T) {
	const total = 47
	server := fakeCommentsPager(t, total, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	seen := map[string]bool{}
	offset := 0
	for pages := 0; ; pages++ {
		if pages > 20 {
			t.Fatal("pagination did not terminate")
		}
		res, err := c.ListComments(context.Background(), ListCommentsArgs{
			BoardID: "board123", Limit: 20, Offset: offset,
		})
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		for _, thread := range res.Comments {
			if seen[thread.ID] {
				t.Fatalf("duplicate thread %s — cursor did not advance", thread.ID)
			}
			seen[thread.ID] = true
		}
		if !res.HasMore {
			break
		}
		if res.Offset == offset {
			t.Fatalf("cursor stalled at %d", offset)
		}
		offset = res.Offset
	}
	if len(seen) != total {
		t.Errorf("collected %d threads, want %d", len(seen), total)
	}
}

// A board whose comments fit in one page is terminal.
func TestListComments_SinglePageIsTerminal(t *testing.T) {
	server := fakeCommentsPager(t, 3, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListComments(context.Background(), ListCommentsArgs{BoardID: "board123", Limit: 50})
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if res.HasMore || res.Offset != 0 {
		t.Errorf("HasMore=%v Offset=%d, want false and 0", res.HasMore, res.Offset)
	}
}

// A board with no comments must not advertise more.
func TestListComments_EmptyIsTerminal(t *testing.T) {
	server := fakeCommentsPager(t, 0, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListComments(context.Background(), ListCommentsArgs{BoardID: "board123", Limit: 50})
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if res.HasMore || res.Count != 0 {
		t.Errorf("HasMore=%v Count=%d, want false and 0", res.HasMore, res.Count)
	}
}

// An offset past the end is terminal rather than looping.
func TestListComments_OffsetPastEndIsTerminal(t *testing.T) {
	server := fakeCommentsPager(t, 12, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListComments(context.Background(), ListCommentsArgs{BoardID: "board123", Limit: 10, Offset: 100})
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if res.HasMore {
		t.Error("HasMore = true past the end, want false")
	}
}

// Total drove has_more on its own, so a response without it reported the end
// of the collection on a full page. It now falls back to the full-page
// heuristic, matching every other paginated endpoint on this client.
func TestListComments_FallsBackWhenTotalAbsent(t *testing.T) {
	server := fakeCommentsPager(t, 120, true)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListComments(context.Background(), ListCommentsArgs{BoardID: "board123", Limit: 50})
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if !res.HasMore {
		t.Error("HasMore = false with total absent and a full page, want true")
	}
	if res.Offset != 50 {
		t.Errorf("Offset = %d, want 50", res.Offset)
	}
}

// A short page with no total is the end of the collection.
func TestListComments_ShortPageWithoutTotalIsTerminal(t *testing.T) {
	server := fakeCommentsPager(t, 3, true)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListComments(context.Background(), ListCommentsArgs{BoardID: "board123", Limit: 50})
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if res.HasMore {
		t.Error("HasMore = true on a short page with no total, want false")
	}
}

// A negative offset is rejected rather than silently dropped from the query.
func TestListComments_RejectsNegativeOffset(t *testing.T) {
	server := fakeCommentsPager(t, 3, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	if _, err := c.ListComments(context.Background(), ListCommentsArgs{BoardID: "board123", Offset: -1}); err == nil {
		t.Error("offset -1: expected an error, got nil")
	}
}
