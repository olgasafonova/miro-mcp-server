package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// boardNamePager serves a fixed, already-filtered list of board names through
// Miro's offset paging. It echoes back the offset it was asked for, which is
// what the live API does, and it counts the requests it answered so a test can
// assert that the common case still costs exactly one call.
//
// The handler deliberately ignores `query`: the server does the filtering in
// production, so the names a test supplies are the filtered set. It records
// the query instead, so a regression that drops the filter while paging is
// still visible.
type boardNamePager struct {
	names     []string
	requests  atomic.Int64
	lastQuery atomic.Value
}

func (p *boardNamePager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.requests.Add(1)
	p.lastQuery.Store(r.URL.Query().Get("query"))

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	data := []map[string]any{}
	for i := offset; i < offset+limit && i < len(p.names); i++ {
		data = append(data, map[string]any{
			"id":       fmt.Sprintf("board%d", i),
			"name":     p.names[i],
			"viewLink": fmt.Sprintf("https://miro.com/app/board/board%d", i),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":   data,
		"size":   len(data),
		"offset": offset,
		"total":  len(p.names),
	})
}

func (p *boardNamePager) query() string {
	q, _ := p.lastQuery.Load().(string)
	return q
}

// newBoardNameClient points a client at a boardNamePager serving names.
func newBoardNameClient(t *testing.T, names []string) (*Client, *boardNamePager) {
	t.Helper()
	pager := &boardNamePager{names: names}
	server := httptest.NewServer(pager)
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL), pager
}

// fillerBoards returns n board names that match no realistic query used here.
func fillerBoards(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("Untitled workspace %d", i)
	}
	return names
}

// A board past the first page used to be invisible: FindBoardByName asked for
// twenty boards and never followed the cursor. Because a full first page
// always supplied the first-result fallback, the caller was handed an
// unrelated board rather than an error — this test fails with ID "board0"
// against the old single-page implementation.
func TestFindBoardByName_FindsMatchOnSecondPage(t *testing.T) {
	names := fillerBoards(MaxBoardLimit + 10)
	names[MaxBoardLimit+5] = "Quarterly Roadmap"

	client, pager := newBoardNameClient(t, names)

	board, err := client.FindBoardByName(context.Background(), "Quarterly Roadmap")
	if err != nil {
		t.Fatalf("FindBoardByName: %v", err)
	}
	wantID := fmt.Sprintf("board%d", MaxBoardLimit+5)
	if board.ID != wantID {
		t.Errorf("ID = %q, want %q", board.ID, wantID)
	}
	if got := pager.requests.Load(); got != 2 {
		t.Errorf("requests = %d, want 2", got)
	}
	if pager.query() != "Quarterly Roadmap" {
		t.Errorf("query forwarded on the paged request = %q, want %q", pager.query(), "Quarterly Roadmap")
	}
}

// Tier beats position across pages, not just within one. A prefix match on
// page one must lose to an exact match on page two, which is the whole reason
// the walk cannot stop at the first candidate of any lower tier.
func TestFindBoardByName_ExactOnLaterPageBeatsEarlierPartial(t *testing.T) {
	names := fillerBoards(MaxBoardLimit + 10)
	names[0] = "Roadmap Q1 Planning" // prefix match, page one
	names[3] = "The Roadmap Archive" // contains match, page one
	names[MaxBoardLimit+4] = "Roadmap"

	client, pager := newBoardNameClient(t, names)

	board, err := client.FindBoardByName(context.Background(), "roadmap")
	if err != nil {
		t.Fatalf("FindBoardByName: %v", err)
	}
	wantID := fmt.Sprintf("board%d", MaxBoardLimit+4)
	if board.ID != wantID {
		t.Errorf("ID = %q (%q), want %q (the exact match on page two)", board.ID, board.Name, wantID)
	}
	if got := pager.requests.Load(); got != 2 {
		t.Errorf("requests = %d, want 2", got)
	}
}

// With no exact match anywhere, the prefix tier still wins over contains and
// over the first-result fallback, and the winner is the earliest board of the
// best tier regardless of which page it came from.
func TestFindBoardByName_PrefixOnLaterPageBeatsEarlierContains(t *testing.T) {
	names := fillerBoards(MaxBoardLimit + 10)
	names[2] = "The Roadmap Archive" // contains match, page one
	names[MaxBoardLimit+1] = "Roadmap Q3"

	client, _ := newBoardNameClient(t, names)

	board, err := client.FindBoardByName(context.Background(), "Roadmap")
	if err != nil {
		t.Fatalf("FindBoardByName: %v", err)
	}
	wantID := fmt.Sprintf("board%d", MaxBoardLimit+1)
	if board.ID != wantID {
		t.Errorf("ID = %q (%q), want %q (the prefix match)", board.ID, board.Name, wantID)
	}
}

// The common case must not pay for the fix. An account whose filtered set fits
// in one page resolves in exactly one API request, whichever tier it matches
// on, because a short page is already terminal.
func TestFindBoardByName_SinglePageCostsOneRequest(t *testing.T) {
	cases := []struct {
		name   string
		query  string
		wantID string
	}{
		{"exact match", "Design Sprint", "board1"},
		{"prefix match", "Design", "board1"},
		{"contains match", "Sprint", "board1"},
		{"no match at all falls back to the first board", "Unrelated", "board0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, pager := newBoardNameClient(t, []string{
				"Annual Planning",
				"Design Sprint",
				"Retro Notes",
			})

			board, err := client.FindBoardByName(context.Background(), tc.query)
			if err != nil {
				t.Fatalf("FindBoardByName: %v", err)
			}
			if board.ID != tc.wantID {
				t.Errorf("ID = %q (%q), want %q", board.ID, board.Name, tc.wantID)
			}
			if got := pager.requests.Load(); got != 1 {
				t.Errorf("requests = %d, want 1", got)
			}
		})
	}
}

// A query the server filters down to nothing still reports the board as
// absent, in the same words, after one request rather than a walk.
func TestFindBoardByName_AbsentBoardReportsClearError(t *testing.T) {
	client, pager := newBoardNameClient(t, nil)

	_, err := client.FindBoardByName(context.Background(), "Nonexistent Board")
	if err == nil {
		t.Fatal("expected an error for a board that does not exist")
	}
	if want := "no board found matching 'Nonexistent Board'"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
	if got := pager.requests.Load(); got != 1 {
		t.Errorf("requests = %d, want 1", got)
	}
}

// The walk is bounded. A query that matches nothing across a set larger than
// the cap stops at MaxFindBoardPages rather than dragging the whole account
// through the client.
func TestFindBoardByName_StopsAtPageCap(t *testing.T) {
	client, pager := newBoardNameClient(t, fillerBoards(MaxBoardLimit*(MaxFindBoardPages+3)))

	board, err := client.FindBoardByName(context.Background(), "Never Appears")
	if err != nil {
		t.Fatalf("FindBoardByName: %v", err)
	}
	if board.ID != "board0" {
		t.Errorf("ID = %q, want \"board0\" (the first-result fallback)", board.ID)
	}
	if got := pager.requests.Load(); got != int64(MaxFindBoardPages) {
		t.Errorf("requests = %d, want %d", got, MaxFindBoardPages)
	}
}

// The tool result must say which tier answered, and must not call a board a
// find when it reached no tier at all.
//
// The last row is the one that used to lie. It is close to unreachable
// through the live API, because Miro's `query` filters on the board name, so
// every board it returns already contains the query and the contains tier
// always hits. boardNamePager ignores `query` on purpose, which reproduces
// what a query that also matched on description or another field would hand
// back: a page of boards whose names do not contain what was asked for.
func TestFindBoardByNameTool_ReportsMatchTier(t *testing.T) {
	cases := []struct {
		name        string
		boards      []string
		query       string
		wantID      string
		wantMatch   string
		wantMessage string
	}{
		{
			name:        "exact match, ignoring case",
			boards:      []string{"Annual Planning", "Design Sprint"},
			query:       "design sprint",
			wantID:      "board1",
			wantMatch:   "exact",
			wantMessage: "Found board 'Design Sprint': exact name match for 'design sprint'",
		},
		{
			name:        "prefix match",
			boards:      []string{"Annual Planning", "Design Sprint Q1"},
			query:       "Design",
			wantID:      "board1",
			wantMatch:   "prefix",
			wantMessage: "Found board 'Design Sprint Q1': name starts with 'Design'",
		},
		{
			name:        "contains match",
			boards:      []string{"Annual Planning", "Q1 Design Sprint"},
			query:       "Design",
			wantID:      "board1",
			wantMatch:   "contains",
			wantMessage: "Found board 'Q1 Design Sprint': name contains 'Design'",
		},
		{
			name:        "no tier matched",
			boards:      []string{"Annual Planning", "Retro Notes"},
			query:       "Roadmap",
			wantID:      "board0",
			wantMatch:   "none",
			wantMessage: "No board name matched 'Roadmap'. Returning 'Annual Planning' as the nearest candidate: it is a guess, not a match. Use miro_list_boards to see what exists",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := newBoardNameClient(t, tc.boards)

			result, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{Name: tc.query})
			if err != nil {
				t.Fatalf("FindBoardByNameTool: %v", err)
			}
			if result.ID != tc.wantID {
				t.Errorf("ID = %q (%q), want %q", result.ID, result.Name, tc.wantID)
			}
			if result.Match != tc.wantMatch {
				t.Errorf("Match = %q, want %q", result.Match, tc.wantMatch)
			}
			if result.Message != tc.wantMessage {
				t.Errorf("Message = %q,\n want %q", result.Message, tc.wantMessage)
			}
		})
	}
}

// The board handed back when nothing matched is still the board the previous
// contract promised, so a caller that only reads ID keeps working. What
// changed is that the result now admits what it is.
func TestFindBoardByNameTool_UnmatchedResultKeepsTheBoardAndDropsTheClaim(t *testing.T) {
	client, _ := newBoardNameClient(t, []string{"Annual Planning", "Retro Notes"})

	result, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{Name: "Roadmap"})
	if err != nil {
		t.Fatalf("FindBoardByNameTool: %v", err)
	}
	if result.Name != "Annual Planning" {
		t.Errorf("Name = %q, want %q (the unchanged fallback)", result.Name, "Annual Planning")
	}
	if strings.Contains(result.Message, "Found board") {
		t.Errorf("Message = %q, want it not to claim a find for a board that matched no tier", result.Message)
	}
}

// An empty name is rejected before any request is made.
func TestFindBoardByName_EmptyNameMakesNoRequest(t *testing.T) {
	client, pager := newBoardNameClient(t, []string{"Annual Planning"})

	if _, err := client.FindBoardByName(context.Background(), ""); err == nil {
		t.Fatal("expected an error for an empty name")
	}
	if got := pager.requests.Load(); got != 0 {
		t.Errorf("requests = %d, want 0", got)
	}
}
