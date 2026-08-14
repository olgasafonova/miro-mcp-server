package miro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// =============================================================================
// Organization Audit Log (Miro Enterprise, GET /v2/audit/logs)
// =============================================================================
//
// Miro's own organization-wide audit trail, NOT this server's local execution
// log — that one lives in miro/audit/ behind GetAuditLog. The names are close
// enough to confuse, so both tool descriptions say which is which.
//
// Two API facts the implementation depends on:
//   - createdAfter and createdBefore are both REQUIRED; there is no default
//     window, and omitting either is a 400 rather than a full-range query
//   - the endpoint needs the auditlogs:read scope on an Enterprise plan, so a
//     403 here usually means plan or scope rather than a genuine permission
//     problem on a specific object

// wrapEnterpriseAuditErr adds a plan/scope hint to the ambiguous statuses,
// mirroring wrapExperimentalCommentErr.
//
// 403 and 404 are ambiguous on this endpoint: a non-Enterprise org, a token
// without auditlogs:read, and a genuinely absent resource are indistinguishable
// from the status alone. Other statuses are unambiguous and pass through
// untouched — in particular 400, which is what a bad time window returns and
// which a plan hint would actively mislead.
func wrapEnterpriseAuditErr(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	if apiErr.IsNotFound() || apiErr.IsForbidden() {
		return fmt.Errorf("%w (note: the organization audit log is a Miro Enterprise feature and needs the auditlogs:read scope; this is not the same as miro_get_audit_log, which reads this server's local execution log)", err)
	}
	return err
}

// clampOrgAuditLimit bounds a requested page size.
func clampOrgAuditLimit(n int) int {
	if n <= 0 {
		return DefaultOrgAuditLimit
	}
	if n > MaxOrgAuditLimit {
		return MaxOrgAuditLimit
	}
	return n
}

// normalizeOrgAuditSorting validates the sort order. Empty means the API
// default. An unrecognized value is rejected rather than silently dropped: a
// caller who asked for ASC and got DESC would misread the page.
func normalizeOrgAuditSorting(s string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "":
		return "", nil
	case OrgAuditSortAsc:
		return OrgAuditSortAsc, nil
	case OrgAuditSortDesc:
		return OrgAuditSortDesc, nil
	default:
		return "", fmt.Errorf("sorting must be %s or %s, got %q", OrgAuditSortAsc, OrgAuditSortDesc, s)
	}
}

// apiAuditEvent is the wire shape of one audit event.
type apiAuditEvent struct {
	ID        string `json:"id"`
	Event     string `json:"event"`
	Category  string `json:"category"`
	CreatedAt string `json:"createdAt"`
	CreatedBy *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Type  string `json:"type"`
	} `json:"createdBy"`
	Object *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"object"`
	Context *struct {
		IP   string `json:"ip"`
		Team *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
		Organization *struct {
			ID string `json:"id"`
		} `json:"organization"`
	} `json:"context"`
	Details map[string]any `json:"details"`
}

// summarizeOrgAuditEvent projects a wire event to the MCP-facing shape,
// flattening the nested context so a caller need not walk it.
func summarizeOrgAuditEvent(e apiAuditEvent) OrgAuditEvent {
	out := OrgAuditEvent{
		ID:        e.ID,
		Event:     e.Event,
		Category:  e.Category,
		CreatedAt: e.CreatedAt,
		Details:   e.Details,
	}
	if e.CreatedBy != nil {
		out.CreatedBy = &OrgAuditActor{
			ID:    e.CreatedBy.ID,
			Name:  e.CreatedBy.Name,
			Email: e.CreatedBy.Email,
			Type:  e.CreatedBy.Type,
		}
	}
	if e.Object != nil {
		out.Object = &OrgAuditTarget{ID: e.Object.ID, Name: e.Object.Name}
	}
	if e.Context != nil {
		out.IP = e.Context.IP
		if e.Context.Team != nil {
			out.TeamID = e.Context.Team.ID
			out.TeamName = e.Context.Team.Name
		}
		if e.Context.Organization != nil {
			out.OrganizationID = e.Context.Organization.ID
		}
	}
	return out
}

// buildOrgAuditQuery assembles the query string, validating the two required
// bounds up front so a missing one fails locally with a usable message rather
// than as an opaque 400.
func buildOrgAuditQuery(args GetOrgAuditLogsArgs) (string, error) {
	if strings.TrimSpace(args.CreatedAfter) == "" {
		return "", errors.New("created_after is required (ISO 8601, e.g. 2026-08-01T00:00:00Z); the API has no default time window")
	}
	if strings.TrimSpace(args.CreatedBefore) == "" {
		return "", errors.New("created_before is required (ISO 8601, e.g. 2026-08-14T00:00:00Z); the API has no default time window")
	}

	sorting, err := normalizeOrgAuditSorting(args.Sorting)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("createdAfter", args.CreatedAfter)
	params.Set("createdBefore", args.CreatedBefore)
	params.Set("limit", strconv.Itoa(clampOrgAuditLimit(args.Limit)))
	if args.Cursor != "" {
		params.Set("cursor", args.Cursor)
	}
	if sorting != "" {
		params.Set("sorting", sorting)
	}
	return params.Encode(), nil
}

// GetOrgAuditLogs fetches a page of Miro's organization audit log.
//
// Enterprise plan and the auditlogs:read scope are required; see
// wrapEnterpriseAuditErr for how that surfaces.
func (c *Client) GetOrgAuditLogs(ctx context.Context, args GetOrgAuditLogsArgs) (GetOrgAuditLogsResult, error) {
	query, err := buildOrgAuditQuery(args)
	if err != nil {
		return GetOrgAuditLogsResult{}, err
	}

	respBody, err := c.request(ctx, http.MethodGet, "/audit/logs?"+query, nil)
	if err != nil {
		return GetOrgAuditLogsResult{}, wrapEnterpriseAuditErr(err)
	}

	var resp struct {
		Data   []apiAuditEvent `json:"data"`
		Limit  int             `json:"limit"`
		Size   int             `json:"size"`
		Cursor string          `json:"cursor"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return GetOrgAuditLogsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	events := make([]OrgAuditEvent, 0, len(resp.Data))
	for _, wire := range resp.Data {
		events = append(events, summarizeOrgAuditEvent(wire))
	}

	result := GetOrgAuditLogsResult{
		Events:  events,
		Count:   len(events),
		Cursor:  resp.Cursor,
		HasMore: resp.Cursor != "",
	}
	if result.Count == 0 {
		result.Message = fmt.Sprintf("No audit events between %s and %s", args.CreatedAfter, args.CreatedBefore)
	}
	return result, nil
}
