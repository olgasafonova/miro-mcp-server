package miro

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// Shared harness for the offset-paginated endpoints. Every one of them —
// boards, board members, items-by-tag, comments — answers the same shape and
// has to satisfy the same contract, so the scenarios live here once instead of
// being copied per endpoint. Each endpoint supplies only what differs: the row
// payload its API returns, and an adapter that normalizes its result.

// offsetPage is one page of any offset-paginated endpoint, reduced to the
// fields the shared scenarios assert on.
type offsetPage struct {
	ids        []string
	hasMore    bool
	nextOffset string
	total      int
}

// offsetEndpoint describes one endpoint for the shared suite.
type offsetEndpoint struct {
	// noun names the thing being paged, for failure messages.
	noun string
	// row builds the API payload for record i.
	row func(i int) map[string]any
	// call invokes the client method against serverURL and normalizes the
	// result. offset is the cursor as the previous page handed it back; ""
	// means the first page.
	call func(c *Client, limit int, offset string) (offsetPage, error)
}

// fakeOffsetPager serves total records, echoing back the offset it was given.
// That echo is the live behaviour (verified against api.miro.com: a request
// for offset=1 answers offset=1) and it is what made the previous HasMore
// computations wrong on the first page.
func fakeOffsetPager(t *testing.T, ep offsetEndpoint, total int, omitTotal bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		data := []map[string]any{}
		for i := offset; i < offset+limit && i < total; i++ {
			data = append(data, ep.row(i))
		}
		body := map[string]any{"data": data, "size": len(data), "offset": offset}
		if !omitTotal {
			body["total"] = total
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
}

// offsetScenario is one row of the shared contract table.
type offsetScenario struct {
	name        string
	total       int
	omitTotal   bool
	limit       int
	offset      string
	wantHasMore bool
	wantOffset  string
	wantTotal   int
}

// offsetScenarios is the contract every offset-paginated endpoint must meet.
// wantTotal is only asserted for endpoints that surface a total.
func offsetScenarios() []offsetScenario {
	return []offsetScenario{
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
			// The boundary the len(items) >= limit heuristic got wrong: a full
			// final page is the end, not evidence of another page.
			name:  "exact multiple of the limit is terminal",
			total: 40, limit: 10, offset: "30",
			wantHasMore: false, wantOffset: "", wantTotal: 40,
		},
		{
			name:  "empty collection is terminal",
			total: 0, limit: 50,
			wantHasMore: false, wantOffset: "", wantTotal: 0,
		},
		{
			name:  "offset past the end is terminal",
			total: 13, limit: 10, offset: "100",
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
}

// runOffsetPaginationContract drives every shared scenario against one
// endpoint. checkTotal is false for endpoints whose result carries no total.
func runOffsetPaginationContract(t *testing.T, ep offsetEndpoint, checkTotal bool) {
	t.Helper()
	for _, sc := range offsetScenarios() {
		t.Run(sc.name, func(t *testing.T) {
			server := fakeOffsetPager(t, ep, sc.total, sc.omitTotal)
			defer server.Close()

			page, err := ep.call(newTestClientWithServer(server.URL), sc.limit, sc.offset)
			if err != nil {
				t.Fatalf("%s: %v", ep.noun, err)
			}
			if page.hasMore != sc.wantHasMore {
				t.Errorf("HasMore = %v, want %v", page.hasMore, sc.wantHasMore)
			}
			if page.nextOffset != sc.wantOffset {
				t.Errorf("next offset = %q, want %q", page.nextOffset, sc.wantOffset)
			}
			if checkTotal && !sc.omitTotal && page.total != sc.wantTotal {
				t.Errorf("Total = %d, want %d", page.total, sc.wantTotal)
			}
		})
	}
}

// walkOffsetPages pages from the start and returns every id in arrival order,
// failing the test on a cursor that stalls or a walk that will not terminate.
func walkOffsetPages(t *testing.T, c *Client, ep offsetEndpoint, limit int) []string {
	t.Helper()

	var ids []string
	offset := ""
	for pages := 0; pages <= 50; pages++ {
		page, err := ep.call(c, limit, offset)
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		ids = append(ids, page.ids...)
		if !page.hasMore {
			return ids
		}
		if page.nextOffset == offset {
			t.Fatalf("cursor stalled at %q — following it would refetch the same page", offset)
		}
		offset = page.nextOffset
	}
	t.Fatalf("%s pagination did not terminate within 50 pages", ep.noun)
	return nil
}

// runOffsetFullWalk asserts a complete walk yields each record exactly once.
func runOffsetFullWalk(t *testing.T, ep offsetEndpoint, total, limit int) {
	t.Helper()

	server := fakeOffsetPager(t, ep, total, false)
	defer server.Close()

	ids := walkOffsetPages(t, newTestClientWithServer(server.URL), ep, limit)

	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			t.Fatalf("duplicate %s %s — cursor did not advance", ep.noun, id)
		}
		seen[id] = true
	}
	if len(seen) != total {
		t.Errorf("collected %d %s, want %d", len(seen), ep.noun, total)
	}
}

// walkCursorPages is the cursor-paginated counterpart of walkOffsetPages, for
// the endpoints that hand back an opaque cursor instead of an offset.
func walkCursorPages(t *testing.T, noun string, next func(cursor string) (offsetPage, error)) []string {
	t.Helper()

	var ids []string
	cursor := ""
	for pages := 0; pages <= 50; pages++ {
		page, err := next(cursor)
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		ids = append(ids, page.ids...)
		if !page.hasMore {
			return ids
		}
		if page.nextOffset == cursor {
			t.Fatalf("cursor stalled at %q", cursor)
		}
		cursor = page.nextOffset
	}
	t.Fatalf("%s pagination did not terminate within 50 pages", noun)
	return nil
}
