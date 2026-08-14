package miro

// =============================================================================
// Organization Audit Log Types (Miro Enterprise, GET /v2/audit/logs)
// =============================================================================
//
// Distinct from the types in types_audit.go, which describe THIS server's local
// execution log. These describe Miro's own organization-wide audit trail: who
// did what in the Miro workspace, including actions taken outside this server.
// The two share nothing but the word "audit".

// OrgAuditSortAsc and OrgAuditSortDesc are the sort orders the API accepts.
const (
	OrgAuditSortAsc  = "ASC"
	OrgAuditSortDesc = "DESC"
)

// DefaultOrgAuditLimit is the page size when the caller doesn't specify one.
const DefaultOrgAuditLimit = 50

// MaxOrgAuditLimit caps a requested page size.
const MaxOrgAuditLimit = 100

// GetOrgAuditLogsArgs specifies the query for Miro's organization audit log.
//
// created_after and created_before are both REQUIRED by the API — it has no
// default window. Miro retains 90 days; older events are only available via
// the CSV export in the Miro admin UI, which this tool does not wrap.
type GetOrgAuditLogsArgs struct {
	// CreatedAfter is the start of the window (required).
	CreatedAfter string `json:"created_after" jsonschema:"REQUIRED. Start of the time window (ISO 8601, e.g. 2026-08-01T00:00:00Z). Miro retains 90 days."`

	// CreatedBefore is the end of the window (required).
	CreatedBefore string `json:"created_before" jsonschema:"REQUIRED. End of the time window (ISO 8601, e.g. 2026-08-14T00:00:00Z)."`

	// Limit is the page size (default 50, max 100).
	Limit int `json:"limit,omitempty" jsonschema:"Max events per page (default 50, max 100)"`

	// Cursor continues a previous page.
	Cursor string `json:"cursor,omitempty" jsonschema:"Pagination cursor from a previous response"`

	// Sorting orders results by creation time.
	Sorting string `json:"sorting,omitempty" jsonschema:"Sort order by creation time: ASC or DESC (default DESC)"`
}

// OrgAuditActor is the user who performed an audited action.
type OrgAuditActor struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Type  string `json:"type,omitempty"`
}

// OrgAuditTarget is the object an audited action was performed on.
type OrgAuditTarget struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// OrgAuditEvent is one entry in Miro's organization audit log.
type OrgAuditEvent struct {
	ID        string          `json:"id"`
	Event     string          `json:"event"`
	Category  string          `json:"category,omitempty"`
	CreatedAt string          `json:"created_at,omitempty"`
	CreatedBy *OrgAuditActor  `json:"created_by,omitempty"`
	Object    *OrgAuditTarget `json:"object,omitempty"`

	// TeamID and TeamName come from the event's context, flattened because a
	// caller filtering by team should not have to walk a nested object.
	TeamID   string `json:"team_id,omitempty"`
	TeamName string `json:"team_name,omitempty"`

	// OrganizationID likewise comes from context.
	OrganizationID string `json:"organization_id,omitempty"`

	// IP is the source address recorded for the action.
	IP string `json:"ip,omitempty"`

	// Details is the event-specific payload. Left as raw JSON shape because it
	// varies per event type and Miro documents no schema for it; projecting it
	// would mean guessing.
	Details map[string]any `json:"details,omitempty"`
}

// GetOrgAuditLogsResult is one page of organization audit events.
type GetOrgAuditLogsResult struct {
	Events []OrgAuditEvent `json:"events"`

	// Count is the number of events in this page.
	Count int `json:"count"`

	// Cursor is the token for the next page. Empty means this is the last page.
	Cursor string `json:"cursor,omitempty"`

	// HasMore reports whether another page exists, so a caller does not have to
	// infer it from the cursor being non-empty.
	HasMore bool `json:"has_more"`

	// Message carries an explicit zero-result signal.
	Message string `json:"message,omitempty"`
}
