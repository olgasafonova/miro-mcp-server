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

// fakeMembersPager mimics /v2/boards/{id}/members offset paging. Like the
// boards endpoint it echoes back the offset of the page it served (verified
// against api.miro.com on 18-09-2026: offset=1 answers offset=1), which is the
// behaviour that made the previous HasMore computation wrong on page one.
func fakeMembersPager(t *testing.T, total int, omitTotal bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		data := []map[string]any{}
		for i := offset; i < offset+limit && i < total; i++ {
			data = append(data, map[string]any{
				"id":    fmt.Sprintf("member%d", i),
				"name":  fmt.Sprintf("User %d", i),
				"email": fmt.Sprintf("user%d@example.com", i),
				"role":  "viewer",
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

// A full first page of a larger collection must advertise more and hand back a
// usable cursor. Regression: HasMore was `resp.Offset > 0 && ...`, always false
// on page one, and the result carried no offset field for the caller to follow.
func TestListBoardMembers_FirstPageAdvertisesMore(t *testing.T) {
	server := fakeMembersPager(t, 120, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListBoardMembers(context.Background(), ListBoardMembersArgs{BoardID: "board123", Limit: 50})
	if err != nil {
		t.Fatalf("ListBoardMembers: %v", err)
	}
	if !res.HasMore {
		t.Errorf("HasMore = false, want true (50 of 120 members returned)")
	}
	if res.Offset != "50" {
		t.Errorf("Offset = %q, want \"50\"", res.Offset)
	}
	if res.Total != 120 {
		t.Errorf("Total = %d, want 120", res.Total)
	}
}

// Walking every page must terminate and yield each member exactly once.
func TestListBoardMembers_FullWalkTerminatesWithoutDuplicates(t *testing.T) {
	const total = 137
	server := fakeMembersPager(t, total, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	seen := map[string]bool{}
	offset := ""
	for pages := 0; ; pages++ {
		if pages > 50 {
			t.Fatal("pagination did not terminate")
		}
		res, err := c.ListBoardMembers(context.Background(), ListBoardMembersArgs{
			BoardID: "board123", Limit: 50, Offset: offset,
		})
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		for _, m := range res.Members {
			if seen[m.ID] {
				t.Fatalf("duplicate member %s — cursor did not advance", m.ID)
			}
			seen[m.ID] = true
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
		t.Errorf("collected %d members, want %d", len(seen), total)
	}
}

// A collection that fits in one page is terminal.
func TestListBoardMembers_SinglePageIsTerminal(t *testing.T) {
	server := fakeMembersPager(t, 3, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListBoardMembers(context.Background(), ListBoardMembersArgs{BoardID: "board123", Limit: 50})
	if err != nil {
		t.Fatalf("ListBoardMembers: %v", err)
	}
	if res.HasMore || res.Offset != "" {
		t.Errorf("HasMore=%v Offset=%q, want false and \"\"", res.HasMore, res.Offset)
	}
}

// An empty board must not advertise more.
func TestListBoardMembers_EmptyIsTerminal(t *testing.T) {
	server := fakeMembersPager(t, 0, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListBoardMembers(context.Background(), ListBoardMembersArgs{BoardID: "board123", Limit: 50})
	if err != nil {
		t.Fatalf("ListBoardMembers: %v", err)
	}
	if res.HasMore || res.Count != 0 {
		t.Errorf("HasMore=%v Count=%d, want false and 0", res.HasMore, res.Count)
	}
}

// An offset past the end is terminal rather than looping.
func TestListBoardMembers_OffsetPastEndIsTerminal(t *testing.T) {
	server := fakeMembersPager(t, 3, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListBoardMembers(context.Background(), ListBoardMembersArgs{
		BoardID: "board123", Limit: 2, Offset: "100",
	})
	if err != nil {
		t.Fatalf("ListBoardMembers: %v", err)
	}
	if res.HasMore {
		t.Errorf("HasMore = true past the end, want false")
	}
}

// `total` is not marked required in Miro's OpenAPI spec. When it is absent we
// fall back to the full-page heuristic rather than reporting a false end.
func TestListBoardMembers_FallsBackWhenTotalAbsent(t *testing.T) {
	server := fakeMembersPager(t, 120, true)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListBoardMembers(context.Background(), ListBoardMembersArgs{BoardID: "board123", Limit: 50})
	if err != nil {
		t.Fatalf("ListBoardMembers: %v", err)
	}
	if !res.HasMore {
		t.Error("HasMore = false with total absent and a full page, want true")
	}
	if res.Offset != "50" {
		t.Errorf("Offset = %q, want \"50\"", res.Offset)
	}
}

// A short page with no total is the end of the collection.
func TestListBoardMembers_ShortPageWithoutTotalIsTerminal(t *testing.T) {
	server := fakeMembersPager(t, 3, true)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListBoardMembers(context.Background(), ListBoardMembersArgs{BoardID: "board123", Limit: 50})
	if err != nil {
		t.Fatalf("ListBoardMembers: %v", err)
	}
	if res.HasMore {
		t.Error("HasMore = true on a short page with no total, want false")
	}
}

// A non-numeric offset is rejected instead of silently becoming 0.
func TestListBoardMembers_RejectsInvalidOffset(t *testing.T) {
	server := fakeMembersPager(t, 3, false)
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
	server := fakeMembersPager(t, 3, false)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.ListBoardMembers(context.Background(), ListBoardMembersArgs{BoardID: "board123", Limit: 1})
	if err != nil {
		t.Fatalf("ListBoardMembers: %v", err)
	}
	if res.Count != 1 {
		t.Errorf("Count = %d, want 1", res.Count)
	}
	if !res.HasMore || res.Offset != "1" {
		t.Errorf("HasMore=%v Offset=%q, want true and \"1\"", res.HasMore, res.Offset)
	}
}
