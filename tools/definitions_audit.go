package tools

// Audit Tools (Local Operations)
var auditTools = []ToolSpec{
	{
		Name:     "miro_get_audit_log",
		Method:   "GetAuditLog",
		Title:    "Get Audit Log",
		Category: "audit",
		ReadOnly: true,
		Description: `Query THIS SERVER's local execution log — which MCP tools ran here, when, and whether they succeeded. Covers this process only; it knows nothing about actions taken in Miro's UI or by other clients.

USE WHEN debugging what this server did. For who did what in the Miro workspace, use miro_get_org_audit_logs instead.

Filter by time range, tool, board, action type, or success/failure. RETURNS: Array of entries with timestamps, tool names, board IDs, and outcomes.`,
	},
	{
		Name:     "miro_get_org_audit_logs",
		Method:   "GetOrgAuditLogs",
		Title:    "Get Organization Audit Logs",
		Category: "audit",
		ReadOnly: true,
		Description: `Query MIRO's organization-wide audit log — who did what across the Miro workspace, including actions taken outside this server. ENTERPRISE ONLY; needs the auditlogs:read scope.

USE WHEN investigating workspace activity, access changes, or board history across users. For what this server itself executed, use miro_get_audit_log instead.

created_after and created_before are BOTH REQUIRED — the API has no default window. Miro retains 90 days; older events are only available via the CSV export in the Miro admin UI.

RETURNS: Cursor-paginated events with actor (name/email), target object, event type, category, team, and source IP.`,
	},
	{
		Name:     "miro_who_am_i",
		Method:   "WhoAmI",
		Title:    "Who Am I (Token Introspection)",
		Category: "read",
		ReadOnly: true,
		Description: `Introspect the access token this server is using — whose it is, which team and organization it is scoped to, and which scopes it carries (GET /v1/oauth-token).

USE WHEN a call returned 403 or you need to know what this token may do BEFORE blaming the endpoint: a missing scope (e.g. auditlogs:read) and a genuinely forbidden resource look identical from the status alone.

RETURNS: user, team, organization, application (each id + name), and the scopes list.`,
	},
	{
		Name:     "miro_get_desire_paths",
		Method:   "GetDesirePathReport",
		Title:    "Get Desire Path Report",
		Category: "audit",
		ReadOnly: true,
		Description: `Query desire path normalizations. Shows what agents tried to send and how it was auto-corrected (URLs in ID fields, camelCase keys, string numbers, etc.). USE WHEN reviewing tool usage patterns to improve descriptions or schemas.

Filter by tool name or normalizer rule. Returns top patterns and recent events.`,
	},
}
