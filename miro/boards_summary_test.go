package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Board Summary / Content Tests
// =============================================================================

// boardContentServer routes the four endpoints GetBoardContent drives.
// Any handler left nil falls through to a minimal valid response.
type boardContentServer struct {
	board      http.HandlerFunc
	items      http.HandlerFunc
	connectors http.HandlerFunc
	tags       http.HandlerFunc
}

func (b boardContentServer) start() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/connectors"):
			if b.connectors != nil {
				b.connectors(w, r)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		case strings.HasSuffix(r.URL.Path, "/tags"):
			if b.tags != nil {
				b.tags(w, r)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		case strings.HasSuffix(r.URL.Path, "/items"):
			if b.items != nil {
				b.items(w, r)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}, "size": 0})
		default:
			if b.board != nil {
				b.board(w, r)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "board123",
				"name":     "Test Board",
				"viewLink": "https://miro.com/board123",
			})
		}
	}))
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func itemJSON(id, itemType, content string, xy [2]float64) map[string]interface{} {
	return map[string]interface{}{
		"id":       id,
		"type":     itemType,
		"position": map[string]interface{}{"x": xy[0], "y": xy[1]},
		"data":     map[string]interface{}{"content": content},
	}
}

// requireIntField asserts one named int value.
func requireIntField(t *testing.T, name string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %d, want %d", name, got, want)
	}
}

// requireStringField asserts one named string value.
func requireStringField(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}

func TestGetBoardSummary_RecentItemsCappedAtFive(t *testing.T) {
	var items []map[string]interface{}
	for i := 0; i < 9; i++ {
		items = append(items, itemJSON(fmt.Sprintf("i%d", i), "sticky_note", "x", [2]float64{0, 0}))
	}

	srv := boardContentServer{
		items: func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, map[string]interface{}{"data": items, "size": len(items)})
		},
	}.start()
	defer srv.Close()

	client := newTestClientWithServer(srv.URL)
	result, err := client.GetBoardSummary(context.Background(), GetBoardSummaryArgs{BoardID: "board123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.RecentItems) != 5 {
		t.Errorf("RecentItems = %d, want 5", len(result.RecentItems))
	}
	if result.TotalItems != 9 {
		t.Errorf("TotalItems = %d, want 9", result.TotalItems)
	}
}

func TestGetBoardSummary_ItemsFetchFails(t *testing.T) {
	srv := boardContentServer{
		items: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	}.start()
	defer srv.Close()

	client := newTestClientWithServer(srv.URL)
	_, err := client.GetBoardSummary(context.Background(), GetBoardSummaryArgs{BoardID: "board123"})
	if err == nil || !strings.Contains(err.Error(), "failed to list items") {
		t.Errorf("error = %v, want failed to list items", err)
	}
}

func TestGetBoardContent_Success(t *testing.T) {
	srv := boardContentServer{
		items: func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, map[string]interface{}{
				"data": []map[string]interface{}{
					itemJSON("f1", "frame", "Frame One", [2]float64{0, 0}),
					itemJSON("s1", "sticky_note", "hello", [2]float64{10, 10}),
				},
				"size": 2,
			})
		},
	}.start()
	defer srv.Close()

	client := newTestClientWithServer(srv.URL)
	result, err := client.GetBoardContent(context.Background(), GetBoardContentArgs{BoardID: "board123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requireStringField(t, "ID", result.ID, "board123")
	if len(result.Frames) != 1 {
		t.Fatalf("Frames = %d, want 1", len(result.Frames))
	}
	requireStringField(t, "frame title", result.Frames[0].Title, "Frame One")
	requireIntField(t, "StickyNotes", len(result.ItemsByType.StickyNotes), 1)
	requireIntField(t, "UniqueEntries", result.ContentSummary.UniqueEntries, 2)
	if result.Connectors != nil {
		t.Error("Connectors should be nil when IncludeConnectors is false")
	}
	if result.Tags != nil {
		t.Error("Tags should be nil when IncludeTags is false")
	}
	if !strings.Contains(result.Message, "1 frames") {
		t.Errorf("Message = %q, want a frame count", result.Message)
	}
}

func TestGetBoardContent_WithConnectorsAndTags(t *testing.T) {
	srv := boardContentServer{
		items: func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, map[string]interface{}{
				"data": []map[string]interface{}{
					itemJSON("s1", "sticky_note", "start", [2]float64{0, 0}),
					itemJSON("s2", "shape", "end", [2]float64{10, 10}),
				},
				"size": 2,
			})
		},
		connectors: func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":        "c1",
						"startItem": map[string]interface{}{"id": "s1"},
						"endItem":   map[string]interface{}{"id": "s2"},
					},
				},
			})
		},
		tags: func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "t1", "title": "Urgent", "fillColor": "red"},
				},
			})
		},
	}.start()
	defer srv.Close()

	client := newTestClientWithServer(srv.URL)
	result, err := client.GetBoardContent(context.Background(), GetBoardContentArgs{
		BoardID:           "board123",
		IncludeConnectors: true,
		IncludeTags:       true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Connectors) != 1 {
		t.Fatalf("Connectors = %d, want 1", len(result.Connectors))
	}
	requireStringField(t, "StartItemType", result.Connectors[0].StartItemType, "sticky_note")
	requireStringField(t, "EndItemType", result.Connectors[0].EndItemType, "shape")
	if len(result.Tags) != 1 {
		t.Fatalf("Tags = %d, want 1", len(result.Tags))
	}
	requireStringField(t, "tag title", result.Tags[0].Title, "Urgent")
	if !strings.Contains(result.Message, "1 connectors") {
		t.Errorf("Message = %q, want a connector count", result.Message)
	}
	if !strings.Contains(result.Message, "1 tags") {
		t.Errorf("Message = %q, want a tag count", result.Message)
	}
}

func TestGetBoardContent_EnrichmentErrorsAreSwallowed(t *testing.T) {
	srv := boardContentServer{
		connectors: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
		tags:       func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	}.start()
	defer srv.Close()

	client := newTestClientWithServer(srv.URL)
	result, err := client.GetBoardContent(context.Background(), GetBoardContentArgs{
		BoardID:           "board123",
		IncludeConnectors: true,
		IncludeTags:       true,
	})
	if err != nil {
		t.Fatalf("connector/tag failures must not fail the call: %v", err)
	}
	if result.Connectors != nil {
		t.Errorf("Connectors = %v, want nil on fetch error", result.Connectors)
	}
	if result.Tags != nil {
		t.Errorf("Tags = %v, want nil on fetch error", result.Tags)
	}
}

func TestGetBoardContent_Errors(t *testing.T) {
	t.Run("invalid board id", func(t *testing.T) {
		client := newTestClientWithServer("http://unused.invalid")
		_, err := client.GetBoardContent(context.Background(), GetBoardContentArgs{BoardID: ""})
		if err == nil || !strings.Contains(err.Error(), "board_id is required") {
			t.Errorf("error = %v, want board_id is required", err)
		}
	})

	t.Run("board fetch fails", func(t *testing.T) {
		srv := boardContentServer{
			board: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
		}.start()
		defer srv.Close()

		client := newTestClientWithServer(srv.URL)
		_, err := client.GetBoardContent(context.Background(), GetBoardContentArgs{BoardID: "board123"})
		if err == nil || !strings.Contains(err.Error(), "failed to get board") {
			t.Errorf("error = %v, want failed to get board", err)
		}
	})

	t.Run("items fetch fails", func(t *testing.T) {
		srv := boardContentServer{
			items: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
		}.start()
		defer srv.Close()

		client := newTestClientWithServer(srv.URL)
		_, err := client.GetBoardContent(context.Background(), GetBoardContentArgs{BoardID: "board123"})
		if err == nil || !strings.Contains(err.Error(), "failed to list items") {
			t.Errorf("error = %v, want failed to list items", err)
		}
	})
}

func TestClampBoardContentMaxItems(t *testing.T) {
	tests := []struct {
		name      string
		requested int
		want      int
	}{
		{"zero uses default", 0, 500},
		{"negative uses default", -10, 500},
		{"in range is preserved", 250, 250},
		{"above cap is clamped", 5000, 2000},
		{"exactly cap is preserved", 2000, 2000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clampBoardContentMaxItems(tt.requested); got != tt.want {
				t.Errorf("clampBoardContentMaxItems(%d) = %d, want %d", tt.requested, got, tt.want)
			}
		})
	}
}

func TestAppendItemByType(t *testing.T) {
	var by ItemsByType
	types := []string{
		"sticky_note", "shape", "text", "card",
		"image", "document", "embed", "mystery_type",
	}
	for i, typ := range types {
		appendItemByType(&by, ItemSummary{ID: fmt.Sprintf("i%d", i), Type: typ})
	}

	checks := []struct {
		name string
		got  int
	}{
		{"StickyNotes", len(by.StickyNotes)},
		{"Shapes", len(by.Shapes)},
		{"Text", len(by.Text)},
		{"Cards", len(by.Cards)},
		{"Images", len(by.Images)},
		{"Documents", len(by.Documents)},
		{"Embeds", len(by.Embeds)},
		{"Other", len(by.Other)},
	}
	for _, c := range checks {
		if c.got != 1 {
			t.Errorf("%s = %d, want 1", c.name, c.got)
		}
	}
}

func TestAggregateBoardItems(t *testing.T) {
	items := []ItemSummary{
		{ID: "1", Type: "sticky_note", Content: "hello"},
		{ID: "2", Type: "sticky_note", Content: "world"},
		{ID: "3", Type: "shape", Content: ""},
	}

	agg := aggregateBoardItems(items)

	if agg.counts["sticky_note"] != 2 {
		t.Errorf("sticky_note count = %d, want 2", agg.counts["sticky_note"])
	}
	if agg.counts["shape"] != 1 {
		t.Errorf("shape count = %d, want 1", agg.counts["shape"])
	}
	if len(agg.itemMap) != 3 {
		t.Errorf("itemMap = %d entries, want 3", len(agg.itemMap))
	}
	if len(agg.allText) != 2 {
		t.Errorf("allText = %d entries, want 2 (empty content skipped)", len(agg.allText))
	}
	if agg.totalChars != len("hello")+len("world") {
		t.Errorf("totalChars = %d, want %d", agg.totalChars, len("hello")+len("world"))
	}
	if len(agg.itemsByType.StickyNotes) != 2 {
		t.Errorf("StickyNotes = %d, want 2", len(agg.itemsByType.StickyNotes))
	}
}

func TestChildrenOf(t *testing.T) {
	items := []ItemSummary{
		{ID: "c1", ParentID: "f1"},
		{ID: "c2", ParentID: "f2"},
		{ID: "c3", ParentID: "f1"},
		{ID: "orphan"},
	}

	got := childrenOf(items, "f1")
	if len(got) != 2 {
		t.Fatalf("childrenOf(f1) = %d items, want 2", len(got))
	}
	if got[0].ID != "c1" || got[1].ID != "c3" {
		t.Errorf("children = %q, %q; want c1, c3", got[0].ID, got[1].ID)
	}
	if childrenOf(items, "nonexistent") != nil {
		t.Error("childrenOf on an unknown frame should return nil")
	}
}

func TestFrameContextFromItem(t *testing.T) {
	frame := ItemSummary{
		ID: "f1", Type: "frame", Content: "Planning",
		X: 1, Y: 2, Width: 300, Height: 200,
	}
	items := []ItemSummary{frame, {ID: "c1", ParentID: "f1"}}

	got := frameContextFromItem(frame, items)

	requireStringField(t, "ID", got.ID, "f1")
	requireStringField(t, "Title", got.Title, "Planning")
	if [4]float64{got.X, got.Y, got.Width, got.Height} != [4]float64{1, 2, 300, 200} {
		t.Errorf("geometry = (%v,%v,%v,%v), want (1,2,300,200)", got.X, got.Y, got.Width, got.Height)
	}
	requireIntField(t, "Children", len(got.Children), 1)
}

func TestBuildFrameHierarchy(t *testing.T) {
	items := []ItemSummary{
		{ID: "f1", Type: "frame", Content: "One"},
		{ID: "s1", Type: "sticky_note", ParentID: "f1"},
		{ID: "f2", Type: "frame", Content: "Two"},
	}

	frames := buildFrameHierarchy(items)
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	if len(frames[0].Children) != 1 {
		t.Errorf("frame one children = %d, want 1", len(frames[0].Children))
	}
	if len(frames[1].Children) != 0 {
		t.Errorf("frame two children = %d, want 0", len(frames[1].Children))
	}

	if buildFrameHierarchy([]ItemSummary{{ID: "s1", Type: "sticky_note"}}) != nil {
		t.Error("a board with no frames should produce nil")
	}
}

func TestFormatOptionalRFC3339(t *testing.T) {
	if got := formatOptionalRFC3339(time.Time{}); got != "" {
		t.Errorf("zero time = %q, want empty", got)
	}
	ts := time.Date(2026, 7, 25, 12, 30, 0, 0, time.UTC)
	if got := formatOptionalRFC3339(ts); got != "2026-07-25T12:30:00Z" {
		t.Errorf("formatted = %q, want 2026-07-25T12:30:00Z", got)
	}
}

func TestAssembleBoardContentResult(t *testing.T) {
	board := GetBoardResult{}
	board.ID = "board123"
	board.Name = "Test Board"
	board.Description = "desc"
	board.ViewLink = "https://miro.com/board123"
	board.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	allItems := ListAllItemsResult{Count: 2, Truncated: true}
	agg := boardItemAggregation{
		counts:     map[string]int{"sticky_note": 2},
		allText:    []string{"a", "bb"},
		totalChars: 3,
	}
	frames := []FrameContext{{ID: "f1"}}

	got := assembleBoardContentResult(board, allItems, agg, frames)

	requireStringField(t, "ID", got.ID, "board123")
	requireStringField(t, "Name", got.Name, "Test Board")
	requireStringField(t, "CreatedAt", got.CreatedAt, "2026-01-01T00:00:00Z")
	requireStringField(t, "ModifiedAt (zero time)", got.ModifiedAt, "")
	requireIntField(t, "TotalItems", got.TotalItems, 2)
	if !got.Truncated {
		t.Error("Truncated should carry through")
	}
	requireIntField(t, "UniqueEntries", got.ContentSummary.UniqueEntries, 2)
	requireIntField(t, "TotalChars", got.ContentSummary.TotalChars, 3)
	requireIntField(t, "Frames", len(got.Frames), 1)
}

func TestBuildBoardContentMessage(t *testing.T) {
	tests := []struct {
		name   string
		counts boardContentCounts
		want   string
	}{
		{
			name:   "items only",
			counts: boardContentCounts{items: 3},
			want:   "Board 'Demo' has 3 items",
		},
		{
			name:   "frames appended",
			counts: boardContentCounts{items: 3, frames: 2},
			want:   "Board 'Demo' has 3 items, 2 frames",
		},
		{
			name:   "connectors appended",
			counts: boardContentCounts{items: 3, connectors: 1},
			want:   "Board 'Demo' has 3 items, 1 connectors",
		},
		{
			name:   "tags appended",
			counts: boardContentCounts{items: 3, tags: 4},
			want:   "Board 'Demo' has 3 items, 4 tags",
		},
		{
			name:   "all present",
			counts: boardContentCounts{items: 3, frames: 2, connectors: 1, tags: 4},
			want:   "Board 'Demo' has 3 items, 2 frames, 1 connectors, 4 tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildBoardContentMessage("Demo", tt.counts); got != tt.want {
				t.Errorf("message = %q, want %q", got, tt.want)
			}
		})
	}
}
