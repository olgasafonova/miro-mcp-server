package tools

import (
	"encoding/json"
	"time"

	"github.com/olgasafonova/miro-mcp-server/miro/audit"
)

// executionResult bundles the dynamic outputs of one tool invocation, used to
// build an audit event without passing five positional arguments.
type executionResult struct {
	args     any
	result   any
	err      error
	duration time.Duration
}

// createAuditEvent assembles an audit event from a tool spec and the
// invocation outputs.
func (h *HandlerRegistry) createAuditEvent(spec ToolSpec, ex executionResult) audit.Event {
	event := audit.NewEvent(spec.Name, spec.Method, audit.DetectAction(spec.Method)).
		WithUser(h.userID, h.userEmail).
		WithDuration(ex.duration)

	applyInputContext(event, ex.args)
	applyOutcome(event, ex.result, ex.err)

	return event.Build()
}

// applyInputContext extracts identifying fields from request args into the
// event builder. No-op if args don't marshal to a map.
func applyInputContext(event *audit.EventBuilder, args any) {
	input := argsToMap(args)
	if input == nil {
		return
	}
	event.WithInput(input)
	if boardID, ok := input["board_id"].(string); ok {
		event.WithBoard(boardID)
	}
	if itemID, ok := input["item_id"].(string); ok {
		event.WithItem(itemID, "")
	}
}

// applyOutcome marks the event success or failure and, on success, extracts
// identifying fields from the result.
func applyOutcome(event *audit.EventBuilder, result any, err error) {
	if err != nil {
		event.Failure(err)
		return
	}
	event.Success()
	resultMap := argsToMap(result)
	if resultMap == nil {
		return
	}
	if itemID, ok := resultMap["id"].(string); ok {
		event.WithItem(itemID, "")
	}
	if created, ok := resultMap["created"].(float64); ok {
		event.WithItemCount(int(created))
	}
}

// argsToMap converts a struct to a map for audit logging via JSON round-trip.
// Returns nil on marshal/unmarshal error.
func argsToMap(v any) map[string]interface{} {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}
