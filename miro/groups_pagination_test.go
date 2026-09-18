package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeGroupItemsPager mimics the cursor paging of
// /v2/boards/{id}/groups/{id}/items. Cursors are opaque to the client, so the
// fake uses the index of the next row as its token.
func fakeGroupItemsPager(t *testing.T, total, pageSize int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := 0
		if cursor := r.URL.Query().Get("cursor"); cursor != "" {
			if _, err := fmt.Sscanf(cursor, "at-%d", &start); err != nil {
				t.Errorf("unrecognized cursor %q", cursor)
			}
		}

		data := []map[string]any{}
		for i := start; i < start+pageSize && i < total; i++ {
			data = append(data, map[string]any{
				"id":   fmt.Sprintf("item%d", i),
				"type": "sticky_note",
				"data": map[string]any{"content": fmt.Sprintf("grouped %d", i)},
			})
		}
		body := map[string]any{"data": data}
		if next := start + len(data); next < total {
			body["cursor"] = fmt.Sprintf("at-%d", next)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
}

func groupItemsArgs(limit int, cursor string) GetGroupItemsArgs {
	return GetGroupItemsArgs{
		BoardID: "board123",
		GroupID: "3458764517819745391",
		Limit:   limit,
		Cursor:  cursor,
	}
}

// The cursor was in scope on the HasMore line and dropped on the floor, so a
// caller was told more existed with no way to reach it. ListGroups, twenty
// lines above in the same file, had always returned it.
func TestGetGroupItems_FirstPageHandsBackCursor(t *testing.T) {
	server := fakeGroupItemsPager(t, 25, 10)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.GetGroupItems(context.Background(), groupItemsArgs(10, ""))
	if err != nil {
		t.Fatalf("GetGroupItems: %v", err)
	}
	if !res.HasMore {
		t.Error("HasMore = false, want true (10 of 25 items returned)")
	}
	if res.Cursor == "" {
		t.Error("Cursor is empty while HasMore is true — the caller is stranded")
	}
}

// Walking every page must terminate and yield each item exactly once.
func TestGetGroupItems_FullWalkTerminatesWithoutDuplicates(t *testing.T) {
	const total = 137
	server := fakeGroupItemsPager(t, total, 50)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	ids := walkCursorPages(t, "group item", func(cursor string) (offsetPage, error) {
		res, err := c.GetGroupItems(context.Background(), groupItemsArgs(50, cursor))
		if err != nil {
			return offsetPage{}, err
		}
		out := make([]string, len(res.Items))
		for i, it := range res.Items {
			out[i] = it.ID
		}
		return offsetPage{ids: out, hasMore: res.HasMore, nextOffset: res.Cursor}, nil
	})

	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			t.Fatalf("duplicate group item %s — cursor did not advance", id)
		}
		seen[id] = true
	}
	if len(seen) != total {
		t.Errorf("collected %d group items, want %d", len(seen), total)
	}
}

func TestGetGroupItems_SinglePageIsTerminal(t *testing.T) {
	server := fakeGroupItemsPager(t, 4, 10)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.GetGroupItems(context.Background(), groupItemsArgs(10, ""))
	if err != nil {
		t.Fatalf("GetGroupItems: %v", err)
	}
	if res.HasMore || res.Cursor != "" {
		t.Errorf("HasMore=%v Cursor=%q, want false and \"\"", res.HasMore, res.Cursor)
	}
}

// An empty group must not advertise more.
func TestGetGroupItems_EmptyIsTerminal(t *testing.T) {
	server := fakeGroupItemsPager(t, 0, 10)
	defer server.Close()
	c := newTestClientWithServer(server.URL)

	res, err := c.GetGroupItems(context.Background(), groupItemsArgs(10, ""))
	if err != nil {
		t.Fatalf("GetGroupItems: %v", err)
	}
	if res.Count != 0 {
		t.Errorf("Count = %d, want 0", res.Count)
	}
	if res.HasMore || res.Cursor != "" {
		t.Errorf("HasMore=%v Cursor=%q, want false and \"\"", res.HasMore, res.Cursor)
	}
}
