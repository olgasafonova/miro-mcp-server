package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// itemsWindowBoard is a syntactically valid board ID for the window tests.
const itemsWindowBoard = "uXjVLmnBBBB="

// itemsWindowServer answers the way /boards/{id}/items does: a page size
// outside [MinPagedLimit, MaxItemLimit] is HTTP 400 rather than a short or a
// long page. limit=9 answers "minimum items page size is 10" and limit=51
// answers "maximum items page size is 50", both verified live against
// api.miro.com on 18-09-2026.
//
// It records every limit it was asked for and serves one item per page,
// handing out a cursor until pages is exhausted.
func itemsWindowServer(t *testing.T, seen *[]int, pages int) *httptest.Server {
	t.Helper()
	served := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		*seen = append(*seen, limit)
		if limit < MinPagedLimit || limit > MaxItemLimit {
			writeItemsLimitRejection(w, limit)
			return
		}
		served++
		cursor := ""
		if served < pages {
			cursor = "page" + strconv.Itoa(served+1)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":   []map[string]any{{"id": "item" + strconv.Itoa(served), "type": "sticky_note"}},
			"cursor": cursor,
		})
	}))
	t.Cleanup(server.Close)
	return server
}

// writeItemsLimitRejection reproduces Miro's 400 for an out-of-window page
// size, including the field message that names which end was breached.
func writeItemsLimitRejection(w http.ResponseWriter, limit int) {
	message := "minimum items page size is 10"
	if limit > MaxItemLimit {
		message = "maximum items page size is 50"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type": "error", "code": "2.0703", "status": 400,
		"message": "Invalid parameters",
		"context": map[string]any{"fields": []map[string]string{
			{"field": "limit", "message": message},
		}},
	})
}

// TestListItems_PageSizeClampedIntoWindow pins the limit that reaches the wire
// for every requested value. The schema used to advertise a maximum of 100,
// which the endpoint rejects; a caller taking that at its word must still get
// a page rather than a 400.
func TestListItems_PageSizeClampedIntoWindow(t *testing.T) {
	for _, tc := range []struct{ requested, wire int }{
		{0, DefaultItemLimit},
		{1, MinPagedLimit},
		{9, MinPagedLimit},
		{10, 10},
		{25, 25},
		{50, MaxItemLimit},
		{51, MaxItemLimit},
		{100, MaxItemLimit},
		{10000, MaxItemLimit},
	} {
		t.Run(strconv.Itoa(tc.requested), func(t *testing.T) {
			var seen []int
			server := itemsWindowServer(t, &seen, 1)

			_, err := newTestClientWithServer(server.URL).ListItems(
				context.Background(),
				ListItemsArgs{BoardID: itemsWindowBoard, Limit: tc.requested},
			)
			if err != nil {
				t.Fatalf("limit=%d: %v (sent %v)", tc.requested, err, seen)
			}
			if len(seen) != 1 || seen[0] != tc.wire {
				t.Errorf("limit=%d: sent %v to the API, want [%d]", tc.requested, seen, tc.wire)
			}
		})
	}
}

// TestListAllItems_EveryPageFitsTheWindow covers the pager. It used to request
// MaxItemLimitExtended on every page and only worked because buildListItemsPath
// reduced that to 50 on the way out; a server that enforces the real ceiling
// catches the request as it is written rather than as it is sent.
func TestListAllItems_EveryPageFitsTheWindow(t *testing.T) {
	const pages = 3

	var seen []int
	server := itemsWindowServer(t, &seen, pages)

	result, err := newTestClientWithServer(server.URL).ListAllItems(
		context.Background(),
		ListAllItemsArgs{BoardID: itemsWindowBoard},
	)
	if err != nil {
		t.Fatalf("ListAllItems: %v (sent %v)", err, seen)
	}
	if result.TotalPages != pages {
		t.Errorf("paged %d times, want %d", result.TotalPages, pages)
	}
	if len(seen) != pages {
		t.Fatalf("server saw %d requests, want %d", len(seen), pages)
	}
	for i, limit := range seen {
		if limit < MinPagedLimit || limit > MaxItemLimit {
			t.Errorf("page %d asked for limit=%d, outside the [%d, %d] window",
				i+1, limit, MinPagedLimit, MaxItemLimit)
		}
	}
}
