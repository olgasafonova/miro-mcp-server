package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/olgasafonova/miro-mcp-server/miro"
	"github.com/olgasafonova/miro-mcp-server/miro/audit"
)

// GetAuditLog queries the local audit log for events matching the request
// filters and returns them in the API response shape.
func (h *HandlerRegistry) GetAuditLog(ctx context.Context, args miro.GetAuditLogArgs) (miro.GetAuditLogResult, error) {
	since, until, err := parseTimeRange(args.Since, args.Until)
	if err != nil {
		return miro.GetAuditLogResult{}, err
	}

	opts := audit.QueryOptions{
		Since:   since,
		Until:   until,
		Tool:    args.Tool,
		BoardID: args.BoardID,
		Action:  audit.Action(args.Action),
		Success: args.Success,
		Limit:   clampLimit(args.Limit, 50, 500),
	}

	result, err := h.auditLogger.Query(ctx, opts)
	if err != nil {
		return miro.GetAuditLogResult{}, fmt.Errorf("audit query failed: %w", err)
	}

	events := convertAuditEvents(result.Events)
	return miro.GetAuditLogResult{
		Events:  events,
		Total:   result.Total,
		HasMore: result.HasMore,
		Message: fmt.Sprintf("Found %d audit events", len(events)),
	}, nil
}

// parseTimeRange parses the optional RFC3339 since/until timestamps. Empty
// inputs yield the zero time.
func parseTimeRange(since, until string) (time.Time, time.Time, error) {
	sinceT, err := parseOptionalRFC3339(since, "since")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	untilT, err := parseOptionalRFC3339(until, "until")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return sinceT, untilT, nil
}

// parseOptionalRFC3339 parses an RFC3339 timestamp; an empty string returns
// the zero time without error.
func parseOptionalRFC3339(s, label string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid '%s' time format: %w", label, err)
	}
	return t, nil
}

// clampLimit applies a default and maximum to a requested page size.
// Non-positive requests fall to defaultLimit; oversized requests are capped
// at maxLimit.
func clampLimit(requested, defaultLimit, maxLimit int) int {
	if requested <= 0 {
		return defaultLimit
	}
	if requested > maxLimit {
		return maxLimit
	}
	return requested
}

// convertAuditEvents maps internal audit events to the API response type.
func convertAuditEvents(events []audit.Event) []miro.AuditLogEvent {
	out := make([]miro.AuditLogEvent, len(events))
	for i, e := range events {
		out[i] = miro.AuditLogEvent{
			ID:         e.ID,
			Timestamp:  e.Timestamp,
			Tool:       e.Tool,
			Action:     string(e.Action),
			BoardID:    e.BoardID,
			ItemID:     e.ItemID,
			Success:    e.Success,
			Error:      e.Error,
			DurationMs: e.DurationMs,
		}
	}
	return out
}
