package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Error-path coverage for the v2-experimental code widget operations. Happy
// paths live in client_codewidgets_test.go; this file is split by concern
// following the bulk.go / bulk_errors_test.go precedent.

// codeWidgetOp binds an operation name to a client call using valid IDs, so
// transport and HTTP error behaviour can be asserted across all six
// operations from one table.
type codeWidgetOp struct {
	name string
	call func(ctx context.Context, c *Client) error
}

// allCodeWidgetOps returns every exported code widget operation with
// arguments that pass validation, so failures come from the transport or the
// API response rather than from argument checks.
func allCodeWidgetOps() []codeWidgetOp {
	return []codeWidgetOp{
		{"CreateCodeWidget", func(ctx context.Context, c *Client) error {
			_, err := c.CreateCodeWidget(ctx, CreateCodeWidgetArgs{BoardID: "board123", Code: "x = 1"})
			return err
		}},
		{"GetCodeWidget", func(ctx context.Context, c *Client) error {
			_, err := c.GetCodeWidget(ctx, GetCodeWidgetArgs{BoardID: "board123", ItemID: "cw123"})
			return err
		}},
		{"ListCodeWidgets", func(ctx context.Context, c *Client) error {
			_, err := c.ListCodeWidgets(ctx, ListCodeWidgetsArgs{BoardID: "board123"})
			return err
		}},
		{"UpdateCodeWidget", func(ctx context.Context, c *Client) error {
			_, err := c.UpdateCodeWidget(ctx, UpdateCodeWidgetArgs{BoardID: "board123", ItemID: "cw123", Code: "y = 2"})
			return err
		}},
		{"MoveCodeWidget", func(ctx context.Context, c *Client) error {
			_, err := c.MoveCodeWidget(ctx, MoveCodeWidgetArgs{BoardID: "board123", ItemID: "cw123", X: 1, Y: 2})
			return err
		}},
		{"DeleteCodeWidget", func(ctx context.Context, c *Client) error {
			_, err := c.DeleteCodeWidget(ctx, DeleteCodeWidgetArgs{BoardID: "board123", ItemID: "cw123"})
			return err
		}},
	}
}

// newCodeWidgetErrorServer returns a server that fails every request with the
// given status.
func newCodeWidgetErrorServer(t *testing.T, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "boom"})
	}))
}

// TestCodeWidgets_ExperimentalHintByStatus pins which statuses get the
// v2-experimental availability hint. 403 and 404 are ambiguous (missing item
// vs missing experimental access) and get the hint; other statuses are
// unambiguous and must pass through untouched.
func TestCodeWidgets_ExperimentalHintByStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		wantHint bool
	}{
		{"not_found_is_ambiguous", http.StatusNotFound, true},
		{"forbidden_is_ambiguous", http.StatusForbidden, true},
		{"bad_request_is_unambiguous", http.StatusBadRequest, false},
		{"conflict_is_unambiguous", http.StatusConflict, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newCodeWidgetErrorServer(t, tt.status)
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			_, err := client.GetCodeWidget(context.Background(), GetCodeWidgetArgs{
				BoardID: "board123",
				ItemID:  "cw123",
			})

			if err == nil {
				t.Fatalf("expected error for status %d", tt.status)
			}
			gotHint := strings.Contains(err.Error(), "v2-experimental")
			if gotHint != tt.wantHint {
				t.Errorf("hint present = %v, want %v (err: %v)", gotHint, tt.wantHint, err)
			}
		})
	}
}

// TestCodeWidgets_ForbiddenHintAllOps asserts every operation routes its API
// errors through wrapExperimentalCodeWidgetErr, not just the read paths.
func TestCodeWidgets_ForbiddenHintAllOps(t *testing.T) {
	for _, op := range allCodeWidgetOps() {
		t.Run(op.name, func(t *testing.T) {
			server := newCodeWidgetErrorServer(t, http.StatusForbidden)
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			err := op.call(context.Background(), client)

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "v2-experimental") {
				t.Errorf("expected experimental hint, got: %v", err)
			}
		})
	}
}

// TestCodeWidgets_TransportError covers the non-APIError branch of
// wrapExperimentalCodeWidgetErr: a transport failure is not an availability
// problem, so it must not acquire the experimental hint.
//
// A cancelled context is used rather than an unreachable port because the
// client retries connection refusals with backoff, which costs ~7s per
// operation; cancellation exercises the same non-APIError branch instantly.
func TestCodeWidgets_TransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("cancelled context must not reach the server; got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	for _, op := range allCodeWidgetOps() {
		t.Run(op.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			client := newTestClientWithServer(server.URL)
			err := op.call(ctx, client)

			if err == nil {
				t.Fatal("expected transport error, got nil")
			}
			if strings.Contains(err.Error(), "v2-experimental") {
				t.Errorf("transport error must pass through unwrapped, got: %v", err)
			}
		})
	}
}

// TestWrapExperimentalCodeWidgetErr_NilPassthrough pins the nil guard, so a
// future refactor that calls the wrapper unconditionally cannot manufacture
// an error out of a successful call.
func TestWrapExperimentalCodeWidgetErr_NilPassthrough(t *testing.T) {
	if err := wrapExperimentalCodeWidgetErr(nil); err != nil {
		t.Errorf("wrapExperimentalCodeWidgetErr(nil) = %v, want nil", err)
	}
}

// TestListCodeWidgets_ForwardsCursor covers the pagination branch: a supplied
// cursor must reach the API as a query parameter.
func TestListCodeWidgets_ForwardsCursor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("cursor"); got != "page-2" {
			t.Errorf("cursor = %q, want 'page-2'", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]interface{}{}})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListCodeWidgets(context.Background(), ListCodeWidgetsArgs{
		BoardID: "board123",
		Cursor:  "page-2",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasMore {
		t.Error("expected HasMore false when the API returns no cursor")
	}
}

// TestCodeWidgets_MalformedJSON covers the response-parsing branch of the
// three operations that unmarshal a body.
func TestCodeWidgets_MalformedJSON(t *testing.T) {
	tests := []struct {
		name string
		call func(ctx context.Context, c *Client) error
	}{
		{"CreateCodeWidget", func(ctx context.Context, c *Client) error {
			_, err := c.CreateCodeWidget(ctx, CreateCodeWidgetArgs{BoardID: "board123", Code: "x = 1"})
			return err
		}},
		{"GetCodeWidget", func(ctx context.Context, c *Client) error {
			_, err := c.GetCodeWidget(ctx, GetCodeWidgetArgs{BoardID: "board123", ItemID: "cw123"})
			return err
		}},
		{"ListCodeWidgets", func(ctx context.Context, c *Client) error {
			_, err := c.ListCodeWidgets(ctx, ListCodeWidgetsArgs{BoardID: "board123"})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"id": not-json`))
			}))
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			err := tt.call(context.Background(), client)

			if err == nil {
				t.Fatal("expected parse error, got nil")
			}
			if !strings.Contains(err.Error(), "failed to parse response") {
				t.Errorf("expected parse error, got: %v", err)
			}
		})
	}
}

// TestCodeWidgets_IDValidation asserts malformed IDs are rejected before any
// request is issued. The server fails the test if it is ever reached.
func TestCodeWidgets_IDValidation(t *testing.T) {
	tests := []struct {
		name    string
		wantErr string
		call    func(ctx context.Context, c *Client) error
	}{
		{"create_empty_board", "board_id is required", func(ctx context.Context, c *Client) error {
			_, err := c.CreateCodeWidget(ctx, CreateCodeWidgetArgs{Code: "x = 1"})
			return err
		}},
		{"create_bad_parent", "invalid parent_id", func(ctx context.Context, c *Client) error {
			_, err := c.CreateCodeWidget(ctx, CreateCodeWidgetArgs{BoardID: "board123", Code: "x = 1", ParentID: "../escape"})
			return err
		}},
		{"get_bad_board", "board_id contains invalid characters", func(ctx context.Context, c *Client) error {
			_, err := c.GetCodeWidget(ctx, GetCodeWidgetArgs{BoardID: "board/../123", ItemID: "cw123"})
			return err
		}},
		{"get_empty_item", "item_id is required", func(ctx context.Context, c *Client) error {
			_, err := c.GetCodeWidget(ctx, GetCodeWidgetArgs{BoardID: "board123"})
			return err
		}},
		{"list_bad_board", "board_id contains invalid characters", func(ctx context.Context, c *Client) error {
			_, err := c.ListCodeWidgets(ctx, ListCodeWidgetsArgs{BoardID: "board 123"})
			return err
		}},
		{"update_bad_item", "invalid item_id", func(ctx context.Context, c *Client) error {
			_, err := c.UpdateCodeWidget(ctx, UpdateCodeWidgetArgs{BoardID: "board123", ItemID: "cw/../123", Code: "x"})
			return err
		}},
		{"update_bad_board", "board_id contains invalid characters", func(ctx context.Context, c *Client) error {
			_, err := c.UpdateCodeWidget(ctx, UpdateCodeWidgetArgs{BoardID: "board 123", ItemID: "cw123", Code: "x"})
			return err
		}},
		{"update_code_too_long", "6000", func(ctx context.Context, c *Client) error {
			_, err := c.UpdateCodeWidget(ctx, UpdateCodeWidgetArgs{
				BoardID: "board123", ItemID: "cw123", Code: strings.Repeat("x", 6001),
			})
			return err
		}},
		{"move_empty_board", "board_id is required", func(ctx context.Context, c *Client) error {
			_, err := c.MoveCodeWidget(ctx, MoveCodeWidgetArgs{ItemID: "cw123"})
			return err
		}},
		{"move_bad_item", "invalid item_id", func(ctx context.Context, c *Client) error {
			_, err := c.MoveCodeWidget(ctx, MoveCodeWidgetArgs{BoardID: "board123", ItemID: "cw 123"})
			return err
		}},
		{"delete_empty_board", "board_id is required", func(ctx context.Context, c *Client) error {
			_, err := c.DeleteCodeWidget(ctx, DeleteCodeWidgetArgs{ItemID: "cw123"})
			return err
		}},
		{"delete_bad_item", "invalid item_id", func(ctx context.Context, c *Client) error {
			_, err := c.DeleteCodeWidget(ctx, DeleteCodeWidgetArgs{BoardID: "board123", ItemID: "cw/../123"})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				t.Errorf("validation must reject before any request; got %s %s", r.Method, r.URL.Path)
			}))
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			err := tt.call(context.Background(), client)

			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

// TestDeleteCodeWidget_FailureResult asserts the delete failure path returns
// a populated result alongside the error, so callers can report which widget
// failed without re-parsing the error string.
func TestDeleteCodeWidget_FailureResult(t *testing.T) {
	server := newCodeWidgetErrorServer(t, http.StatusForbidden)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteCodeWidget(context.Background(), DeleteCodeWidgetArgs{
		BoardID: "board123",
		ItemID:  "cw123",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result.Success {
		t.Error("expected Success false on delete failure")
	}
	if result.ID != "cw123" {
		t.Errorf("ID = %q, want 'cw123'", result.ID)
	}
	if !strings.Contains(result.Message, "Failed to delete") {
		t.Errorf("Message = %q, want a failure message", result.Message)
	}
}

// TestClampCodeWidgetLimit pins the page-size normalisation bounds.
func TestClampCodeWidgetLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{"zero_uses_default", 0, DefaultItemLimit},
		{"negative_uses_default", -5, DefaultItemLimit},
		{"one_passes_through", 1, 1},
		{"at_max_passes_through", MaxItemLimitExtended, MaxItemLimitExtended},
		{"over_max_is_capped", MaxItemLimitExtended + 1, MaxItemLimitExtended},
		{"far_over_max_is_capped", 100000, MaxItemLimitExtended},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clampCodeWidgetLimit(tt.limit); got != tt.want {
				t.Errorf("clampCodeWidgetLimit(%d) = %d, want %d", tt.limit, got, tt.want)
			}
		})
	}
}

// TestBuildCodeWidgetGeometry covers each combination of optional dimensions,
// including the height-only case the happy-path tests never reach.
func TestBuildCodeWidgetGeometry(t *testing.T) {
	width, height := 400.0, 250.0

	tests := []struct {
		name       string
		width      *float64
		height     *float64
		wantNil    bool
		wantWidth  interface{}
		wantHeight interface{}
	}{
		{name: "neither_is_nil", wantNil: true},
		{name: "width_only", width: &width, wantWidth: width},
		{name: "height_only", height: &height, wantHeight: height},
		{name: "both", width: &width, height: &height, wantWidth: width, wantHeight: height},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			geo := buildCodeWidgetGeometry(tt.width, tt.height)

			if tt.wantNil {
				if geo != nil {
					t.Fatalf("expected nil geometry, got %v", geo)
				}
				return
			}
			if geo == nil {
				t.Fatal("expected non-nil geometry")
			}
			if got, ok := geo["width"]; ok != (tt.wantWidth != nil) || (ok && got != tt.wantWidth) {
				t.Errorf("width = %v (present %v), want %v", got, ok, tt.wantWidth)
			}
			if got, ok := geo["height"]; ok != (tt.wantHeight != nil) || (ok && got != tt.wantHeight) {
				t.Errorf("height = %v (present %v), want %v", got, ok, tt.wantHeight)
			}
		})
	}
}

// TestBuildCodeWidgetData covers the omit-empty behaviour of each optional
// data field, including the lineNumbersVisible=false case that must still be
// sent (a nil pointer means "unset", not "false").
func TestBuildCodeWidgetData(t *testing.T) {
	lineNumbersOff := false

	tests := []struct {
		name    string
		code    string
		lang    string
		title   string
		lineNum *bool
		wantNil bool
		wantKey map[string]interface{}
	}{
		{name: "all_empty_is_nil", wantNil: true},
		{name: "code_only", code: "x = 1", wantKey: map[string]interface{}{"code": "x = 1"}},
		{name: "language_only", lang: "go", wantKey: map[string]interface{}{"language": "go"}},
		{name: "title_only", title: "Snippet", wantKey: map[string]interface{}{"title": "Snippet"}},
		{
			name:    "false_line_numbers_still_sent",
			lineNum: &lineNumbersOff,
			wantKey: map[string]interface{}{"lineNumbersVisible": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := buildCodeWidgetData(tt.code, tt.lang, tt.title, tt.lineNum)

			if tt.wantNil {
				if data != nil {
					t.Fatalf("expected nil data, got %v", data)
				}
				return
			}
			if len(data) != len(tt.wantKey) {
				t.Fatalf("data = %v, want exactly %v", data, tt.wantKey)
			}
			for k, want := range tt.wantKey {
				if got := data[k]; got != want {
					t.Errorf("data[%q] = %v, want %v", k, got, want)
				}
			}
		})
	}
}

// TestParseCodeWidgetSummaries_SkipsMalformed asserts one unparseable entry
// does not discard the whole page.
func TestParseCodeWidgetSummaries_SkipsMalformed(t *testing.T) {
	entries := []json.RawMessage{
		json.RawMessage(`{"id":"cw-1","data":{"code":"ok","language":"go"}}`),
		json.RawMessage(`{"id": not-json`),
		json.RawMessage(`{"id":"cw-2","data":{"title":"Second"},"parent":{"id":"frame1"}}`),
	}

	got := parseCodeWidgetSummaries(entries)

	if len(got) != 2 {
		t.Fatalf("got %d summaries, want 2 (malformed entry skipped): %+v", len(got), got)
	}
	if got[0].ID != "cw-1" || got[1].ID != "cw-2" {
		t.Errorf("unexpected IDs: %q, %q", got[0].ID, got[1].ID)
	}
	if got[1].ParentID != "frame1" {
		t.Errorf("ParentID = %q, want 'frame1'", got[1].ParentID)
	}
}

// TestCodeWidgetLabel pins the create-message label fallback: title wins, and
// a missing title falls back to a 30-char code excerpt.
func TestCodeWidgetLabel(t *testing.T) {
	tests := []struct {
		name  string
		title string
		code  string
		want  string
	}{
		{"title_wins", "My Snippet", "x = 1", "My Snippet"},
		{"short_code_fallback", "", "x = 1", "x = 1"},
		{"long_code_truncated", "", strings.Repeat("a", 50), strings.Repeat("a", 27) + "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codeWidgetLabel(tt.title, tt.code); got != tt.want {
				t.Errorf("codeWidgetLabel(%q, ...) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}
