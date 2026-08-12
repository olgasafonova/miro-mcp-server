package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// boardListPayload mirrors the shape GET /v2/boards actually returns, verified
// live against api.miro.com on 13-08-2026. Every field asserted below is
// present on the listing itself, so none of them needs a follow-up request.
func boardListPayload() map[string]interface{} {
	return map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"id":          "uXjVHORa7hA=",
				"name":        "Test Board",
				"description": "A board with metadata",
				"viewLink":    "https://miro.com/app/board/uXjVHORa7hA=",
				"createdAt":   "2026-08-07T13:21:44Z",
				"modifiedAt":  "2026-08-10T12:37:32Z",
				"owner":       map[string]interface{}{"id": "3458764516184293817", "name": "Olga Safonova", "type": "user"},
				"team":        map[string]interface{}{"id": "3458764653228607818", "name": "Dev team", "type": "team"},
			},
			{
				// Minimal board: owner/team/timestamps absent. These must stay
				// nil or empty rather than surfacing as zero values.
				"id":       "uXjVBARE0N0=",
				"name":     "Bare Board",
				"viewLink": "https://miro.com/app/board/uXjVBARE0N0=",
			},
		},
		"size":  2,
		"total": 2,
	}
}

func serveBoardList(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(boardListPayload()); err != nil {
			t.Errorf("encode payload: %v", err)
		}
	}))
}

// TestListBoards_SurfacesOwnerTeamAndTimestamps is the regression test for the
// field set reported null by the official Miro MCP server's board_search_boards
// (team_name, owner, description, modified_at, created_at).
func TestListBoards_SurfacesOwnerTeamAndTimestamps(t *testing.T) {
	server := serveBoardList(t)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListBoards(context.Background(), ListBoardsArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Boards) != 2 {
		t.Fatalf("got %d boards, want 2", len(result.Boards))
	}

	got := result.Boards[0]
	checks := []struct {
		field string
		got   string
		want  string
	}{
		{"Description", got.Description, "A board with metadata"},
		{"TeamID", got.TeamID, "3458764653228607818"},
		{"TeamName", got.TeamName, "Dev team"},
		{"CreatedAt", got.CreatedAt, "2026-08-07T13:21:44Z"},
		{"ModifiedAt", got.ModifiedAt, "2026-08-10T12:37:32Z"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}

	if got.Owner == nil {
		t.Fatal("Owner is nil, want populated")
	}
	if got.Owner.Name != "Olga Safonova" {
		t.Errorf("Owner.Name = %q, want 'Olga Safonova'", got.Owner.Name)
	}
	if got.Owner.ID != "3458764516184293817" {
		t.Errorf("Owner.ID = %q, want '3458764516184293817'", got.Owner.ID)
	}
}

// TestListBoards_OmitsAbsentMetadata guards the other direction: a board with
// no owner, team or timestamps must not report zero values. An agent reading
// created_at "0001-01-01T00:00:00Z" is worse off than one reading nothing.
func TestListBoards_OmitsAbsentMetadata(t *testing.T) {
	server := serveBoardList(t)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListBoards(context.Background(), ListBoardsArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bare := result.Boards[1]
	if bare.Owner != nil {
		t.Errorf("Owner = %+v, want nil", bare.Owner)
	}
	if bare.TeamID != "" || bare.TeamName != "" {
		t.Errorf("team = %q/%q, want empty", bare.TeamID, bare.TeamName)
	}
	if bare.CreatedAt != "" || bare.ModifiedAt != "" {
		t.Errorf("timestamps = %q/%q, want empty", bare.CreatedAt, bare.ModifiedAt)
	}

	// The omitempty tags must actually drop the keys from the wire payload,
	// so the caller sees an absent field rather than a null or a zero date.
	encoded, err := json.Marshal(bare)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var wire map[string]interface{}
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"owner", "team_id", "team_name", "created_at", "modified_at", "description"} {
		if _, present := wire[key]; present {
			t.Errorf("key %q present in payload for a board without it: %s", key, encoded)
		}
	}
}

// TestFindBoardByNameTool_CarriesMetadata ensures the by-name lookup returns
// the same metadata as the listing, so it answers "who owns this" directly.
func TestFindBoardByNameTool_CarriesMetadata(t *testing.T) {
	server := serveBoardList(t)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.FindBoardByNameTool(context.Background(), FindBoardByNameArgs{Name: "Test Board"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TeamName != "Dev team" {
		t.Errorf("TeamName = %q, want 'Dev team'", result.TeamName)
	}
	if result.TeamID != "3458764653228607818" {
		t.Errorf("TeamID = %q, want '3458764653228607818'", result.TeamID)
	}
	if result.CreatedAt != "2026-08-07T13:21:44Z" {
		t.Errorf("CreatedAt = %q, want '2026-08-07T13:21:44Z'", result.CreatedAt)
	}
	if result.ModifiedAt != "2026-08-10T12:37:32Z" {
		t.Errorf("ModifiedAt = %q, want '2026-08-10T12:37:32Z'", result.ModifiedAt)
	}
	if result.Owner == nil || result.Owner.Name != "Olga Safonova" {
		t.Errorf("Owner = %+v, want Olga Safonova", result.Owner)
	}
}

// TestSummarizeBoard_CopiesOwner guards against aliasing: the summary must own
// its Owner pointer so a later mutation of the source Board cannot reach it.
func TestSummarizeBoard_CopiesOwner(t *testing.T) {
	src := Board{
		ID:    "b1",
		Name:  "Board",
		Owner: &User{ID: "u1", Name: "Original"},
	}
	summary := summarizeBoard(src)
	src.Owner.Name = "Mutated"

	if summary.Owner == nil {
		t.Fatal("Owner is nil")
	}
	if summary.Owner.Name != "Original" {
		t.Errorf("Owner.Name = %q, want 'Original' (summary aliases the source)", summary.Owner.Name)
	}
}
