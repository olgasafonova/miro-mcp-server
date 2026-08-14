package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// orgAuditSample is a two-event page shaped the way GET /v2/audit/logs answers:
// a cursor-list envelope, actor under createdBy, and team/org/ip buried in a
// nested context object.
const orgAuditSample = `{
  "type": "cursor-list",
  "limit": 50,
  "size": 2,
  "cursor": "2026-08-01T09:30:10.840687Z#1234567890123456789-DDB",
  "data": [
    {
      "id": "evt-1",
      "event": "board_created",
      "category": "boards",
      "createdAt": "2026-08-12T09:14:00Z",
      "createdBy": {"type": "user", "id": "u-1", "name": "Olga Safonova", "email": "olga@example.com"},
      "object": {"id": "b-1", "name": "Design Sprint"},
      "context": {
        "ip": "203.0.113.7",
        "team": {"id": "t-1", "name": "Design"},
        "organization": {"id": "org-1"}
      },
      "details": {"boardPolicy": "private"}
    },
    {
      "id": "evt-2",
      "event": "user_invited",
      "category": "users",
      "createdAt": "2026-08-12T10:02:00Z"
    }
  ]
}`

// newOrgAuditServer captures the query string the client sent and replies with
// the supplied body.
func newOrgAuditServer(t *testing.T, body string, gotQuery *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gotQuery != nil {
			*gotQuery = r.URL.RawQuery
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
}

// TestGetOrgAuditLogs_FlattensContext pins the projection. The API buries team,
// organization and IP inside a nested context object; a caller filtering by
// team should not have to walk it, so those are flattened onto the event.
func TestGetOrgAuditLogs_FlattensContext(t *testing.T) {
	server := newOrgAuditServer(t, orgAuditSample, nil)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetOrgAuditLogs(context.Background(), GetOrgAuditLogsArgs{
		CreatedAfter:  "2026-08-01T00:00:00Z",
		CreatedBefore: "2026-08-14T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("GetOrgAuditLogs: %v", err)
	}

	if result.Count != 2 {
		t.Fatalf("Count = %d, want 2", result.Count)
	}

	first := result.Events[0]
	if first.TeamID != "t-1" || first.TeamName != "Design" {
		t.Errorf("team = %q/%q, want t-1/Design", first.TeamID, first.TeamName)
	}
	if first.OrganizationID != "org-1" {
		t.Errorf("organization = %q, want org-1", first.OrganizationID)
	}
	if first.IP != "203.0.113.7" {
		t.Errorf("ip = %q, want 203.0.113.7", first.IP)
	}
	if first.CreatedBy == nil || first.CreatedBy.Email != "olga@example.com" {
		t.Errorf("createdBy = %+v, want the actor with email", first.CreatedBy)
	}
	if first.Object == nil || first.Object.Name != "Design Sprint" {
		t.Errorf("object = %+v, want the target board", first.Object)
	}
	if first.Details["boardPolicy"] != "private" {
		t.Errorf("details = %v, want boardPolicy preserved", first.Details)
	}

	// An event with no context, actor or object must project cleanly rather
	// than panic or invent empty structs.
	second := result.Events[1]
	if second.CreatedBy != nil || second.Object != nil {
		t.Errorf("sparse event gained structs: createdBy=%+v object=%+v", second.CreatedBy, second.Object)
	}
	if second.TeamID != "" || second.IP != "" {
		t.Errorf("sparse event gained context: team=%q ip=%q", second.TeamID, second.IP)
	}

	// Cursor drives HasMore; the caller should not infer paging state itself.
	if !result.HasMore || result.Cursor == "" {
		t.Errorf("HasMore=%v Cursor=%q, want true and non-empty", result.HasMore, result.Cursor)
	}
}

// TestGetOrgAuditLogs_RequiresTimeWindow pins that the two mandatory bounds
// fail locally with a usable message. The API returns an opaque 400 for a
// missing bound, and an agent that has to decode a 400 to learn a parameter was
// required will burn a turn doing it.
func TestGetOrgAuditLogs_RequiresTimeWindow(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()
	client := newTestClientWithServer(server.URL)

	for name, args := range map[string]GetOrgAuditLogsArgs{
		"missing after":  {CreatedBefore: "2026-08-14T00:00:00Z"},
		"missing before": {CreatedAfter: "2026-08-01T00:00:00Z"},
		"both missing":   {},
		"blank after":    {CreatedAfter: "   ", CreatedBefore: "2026-08-14T00:00:00Z"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := client.GetOrgAuditLogs(context.Background(), args)
			if err == nil {
				t.Fatal("expected an error for an incomplete time window")
			}
			if !strings.Contains(err.Error(), "required") {
				t.Errorf("error = %v, want it to say the field is required", err)
			}
		})
	}

	if called {
		t.Error("client issued an HTTP request despite an invalid time window")
	}
}

// TestGetOrgAuditLogs_QueryString pins the wire parameter names. They are
// camelCase (createdAfter) while the tool arguments are snake_case
// (created_after), so a rename on either side breaks silently — the API ignores
// unknown query parameters and would answer with an unfiltered page.
func TestGetOrgAuditLogs_QueryString(t *testing.T) {
	var got string
	server := newOrgAuditServer(t, `{"data":[],"size":0}`, &got)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.GetOrgAuditLogs(context.Background(), GetOrgAuditLogsArgs{
		CreatedAfter:  "2026-08-01T00:00:00Z",
		CreatedBefore: "2026-08-14T00:00:00Z",
		Limit:         10,
		Cursor:        "cur-1",
		Sorting:       "asc",
	}); err != nil {
		t.Fatalf("GetOrgAuditLogs: %v", err)
	}

	for _, want := range []string{
		"createdAfter=2026-08-01T00%3A00%3A00Z",
		"createdBefore=2026-08-14T00%3A00%3A00Z",
		"limit=10",
		"cursor=cur-1",
		"sorting=ASC", // normalized to the enum the API accepts
	} {
		if !strings.Contains(got, want) {
			t.Errorf("query %q missing %q", got, want)
		}
	}
}

// TestGetOrgAuditLogs_LimitAndSorting covers the two normalizers.
func TestGetOrgAuditLogs_LimitAndSorting(t *testing.T) {
	t.Run("limit clamps", func(t *testing.T) {
		for in, want := range map[int]int{
			0:    DefaultOrgAuditLimit,
			-5:   DefaultOrgAuditLimit,
			10:   10,
			1000: MaxOrgAuditLimit,
		} {
			if got := clampOrgAuditLimit(in); got != want {
				t.Errorf("clampOrgAuditLimit(%d) = %d, want %d", in, got, want)
			}
		}
	})

	t.Run("unknown sorting is rejected", func(t *testing.T) {
		// Rejected rather than dropped: a caller who asked for ASC and
		// silently got the API default would misread the page order.
		if _, err := normalizeOrgAuditSorting("sideways"); err == nil {
			t.Error("expected an error for an unrecognized sort order")
		}
		if got, err := normalizeOrgAuditSorting(""); err != nil || got != "" {
			t.Errorf("empty sorting = %q, %v; want passthrough", got, err)
		}
	})
}

// TestGetOrgAuditLogs_EnterpriseHintByStatus pins which statuses get the
// plan/scope hint. 403 and 404 are ambiguous here — non-Enterprise org, missing
// auditlogs:read scope, and a genuinely absent resource look identical. Other
// statuses are unambiguous, and 400 in particular must NOT get the hint: that
// is what a malformed time window returns, and a plan hint would send the
// caller chasing the wrong problem.
func TestGetOrgAuditLogs_EnterpriseHintByStatus(t *testing.T) {
	tests := []struct {
		status   int
		wantHint bool
	}{
		{http.StatusForbidden, true},
		{http.StatusNotFound, true},
		{http.StatusBadRequest, false},
		{http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "boom"})
			}))
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			_, err := client.GetOrgAuditLogs(context.Background(), GetOrgAuditLogsArgs{
				CreatedAfter:  "2026-08-01T00:00:00Z",
				CreatedBefore: "2026-08-14T00:00:00Z",
			})
			if err == nil {
				t.Fatalf("expected an error for status %d", tt.status)
			}
			if got := strings.Contains(err.Error(), "Enterprise"); got != tt.wantHint {
				t.Errorf("hint present = %v, want %v (err: %v)", got, tt.wantHint, err)
			}
		})
	}
}

// TestGetOrgAuditLogs_EmptyPageIsExplicit pins the zero-result signal. An agent
// cannot tell an empty array from a silently failed call, so the result names
// the window it searched.
func TestGetOrgAuditLogs_EmptyPageIsExplicit(t *testing.T) {
	server := newOrgAuditServer(t, `{"data":[],"size":0}`, nil)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetOrgAuditLogs(context.Background(), GetOrgAuditLogsArgs{
		CreatedAfter:  "2026-08-01T00:00:00Z",
		CreatedBefore: "2026-08-14T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("GetOrgAuditLogs: %v", err)
	}
	if result.Count != 0 || result.HasMore {
		t.Errorf("Count=%d HasMore=%v, want 0 and false", result.Count, result.HasMore)
	}
	if !strings.Contains(result.Message, "2026-08-01T00:00:00Z") {
		t.Errorf("Message = %q, want it to name the searched window", result.Message)
	}
}
