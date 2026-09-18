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

// fakeTagItemsPager mimics /v2/boards/{id}/items?tag_id=... offset paging.
// Probed live on 18-09-2026: the endpoint is offset-based, echoes the offset it
// was given, and reports total.
func fakeTagItemsPager(t *testing.T, total int, omitTotal bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		data := []map[string]any{}
		for i := offset; i < offset+limit && i < total; i++ {
			data = append(data, map[string]any{
				"id":   fmt.Sprintf("item%d", i),
				"type": "sticky_note",
				"data": map[string]any{"content": fmt.Sprintf("tagged %d", i)},
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

func tagItemsArgs(limit, offset int) GetItemsByTagArgs {
	return GetItemsByTagArgs{BoardID: "board123", TagID: "tag456", Limit: limit, Offset: offset}
}

// A full first page of a larger collection must advertise more and hand back a
// usable cursor. The result previously carried no offset at all.
func TestGetItemsByTag_FirstPageAdvertisesMore(t *testing.T) {
	server := fakeTagItemsPager(t, 120, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.GetItemsByTag(context.Background(), tagItemsArgs(50, 0))
	if err != nil {
		t.Fatalf("GetItemsByTag: %v", err)
	}
	if !res.HasMore {
		t.Error("HasMore = false, want true (50 of 120 items returned)")
	}
	if res.Offset != 50 {
		t.Errorf("Offset = %d, want 50", res.Offset)
	}
	if res.Total != 120 {
		t.Errorf("Total = %d, want 120", res.Total)
	}
}

// The boundary the old heuristic got wrong. HasMore was `len(items) >= limit`,
// so a collection whose size is an exact multiple of the page size always
// reported a phantom next page on its final full page. Total settles it.
func TestGetItemsByTag_ExactMultipleOfLimitIsTerminal(t *testing.T) {
	const total = 40 // exactly 4 pages of 10, so the last page comes back full
	server := fakeTagItemsPager(t, total, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.GetItemsByTag(context.Background(), tagItemsArgs(10, 30))
	if err != nil {
		t.Fatalf("GetItemsByTag: %v", err)
	}
	if res.Count != 10 {
		t.Fatalf("Count = %d, want a full page of 10", res.Count)
	}
	if res.HasMore {
		t.Error("HasMore = true on a full final page, want false — this is the exact-multiple false positive")
	}
	if res.Offset != 0 {
		t.Errorf("Offset = %d, want 0 (no next page)", res.Offset)
	}
}

// Walking every page must terminate and yield each item exactly once.
func TestGetItemsByTag_FullWalkTerminatesWithoutDuplicates(t *testing.T) {
	const total = 40
	server := fakeTagItemsPager(t, total, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	seen := map[string]bool{}
	offset := 0
	for pages := 0; ; pages++ {
		if pages > 20 {
			t.Fatal("pagination did not terminate")
		}
		res, err := c.GetItemsByTag(context.Background(), tagItemsArgs(10, offset))
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		for _, it := range res.Items {
			if seen[it.ID] {
				t.Fatalf("duplicate item %s — cursor did not advance", it.ID)
			}
			seen[it.ID] = true
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
		t.Errorf("collected %d items, want %d", len(seen), total)
	}
}

// A collection that fits in one page is terminal.
func TestGetItemsByTag_SinglePageIsTerminal(t *testing.T) {
	server := fakeTagItemsPager(t, 3, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.GetItemsByTag(context.Background(), tagItemsArgs(50, 0))
	if err != nil {
		t.Fatalf("GetItemsByTag: %v", err)
	}
	if res.HasMore || res.Offset != 0 {
		t.Errorf("HasMore=%v Offset=%d, want false and 0", res.HasMore, res.Offset)
	}
}

// An empty tag must not advertise more.
func TestGetItemsByTag_EmptyIsTerminalWithTotal(t *testing.T) {
	server := fakeTagItemsPager(t, 0, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.GetItemsByTag(context.Background(), tagItemsArgs(50, 0))
	if err != nil {
		t.Fatalf("GetItemsByTag: %v", err)
	}
	if res.HasMore || res.Count != 0 {
		t.Errorf("HasMore=%v Count=%d, want false and 0", res.HasMore, res.Count)
	}
}

// An offset past the end is terminal rather than looping.
func TestGetItemsByTag_OffsetPastEndIsTerminal(t *testing.T) {
	server := fakeTagItemsPager(t, 12, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.GetItemsByTag(context.Background(), tagItemsArgs(10, 100))
	if err != nil {
		t.Fatalf("GetItemsByTag: %v", err)
	}
	if res.HasMore {
		t.Error("HasMore = true past the end, want false")
	}
}

// With total absent the full-page heuristic is all there is, so a full page
// must still advertise more rather than reporting a false end.
func TestGetItemsByTag_FallsBackWhenTotalAbsent(t *testing.T) {
	server := fakeTagItemsPager(t, 120, true)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.GetItemsByTag(context.Background(), tagItemsArgs(50, 0))
	if err != nil {
		t.Fatalf("GetItemsByTag: %v", err)
	}
	if !res.HasMore {
		t.Error("HasMore = false with total absent and a full page, want true")
	}
	if res.Offset != 50 {
		t.Errorf("Offset = %d, want 50", res.Offset)
	}
}

// A short page with no total is the end of the collection.
func TestGetItemsByTag_ShortPageWithoutTotalIsTerminal(t *testing.T) {
	server := fakeTagItemsPager(t, 3, true)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.GetItemsByTag(context.Background(), tagItemsArgs(50, 0))
	if err != nil {
		t.Fatalf("GetItemsByTag: %v", err)
	}
	if res.HasMore {
		t.Error("HasMore = true on a short page with no total, want false")
	}
}

// A negative offset is rejected rather than forwarded to the API.
func TestGetItemsByTag_RejectsNegativeOffset(t *testing.T) {
	server := fakeTagItemsPager(t, 3, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	if _, err := c.GetItemsByTag(context.Background(), tagItemsArgs(10, -1)); err == nil {
		t.Error("offset -1: expected an error, got nil")
	}
}
