package tools

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/olgasafonova/miro-mcp-server/miro"
)

// recoverPanic recovers from panics in tool handlers and converts them into a
// structured error with a correlation ID. The panic value and stack are logged
// server-side; only the correlation ID reaches the MCP caller.
//
// MUST be called as `defer h.recoverPanic(spec.Name, &err)` from a function
// with NAMED return values. Without named returns the deferred reassignment
// is a no-op and panics surface as silent fake-success responses.
func (h *HandlerRegistry) recoverPanic(toolName string, errPtr *error) {
	rec := recover()
	if rec == nil {
		return
	}
	corrID := newCorrelationID()
	h.logger.Error("Panic recovered",
		"tool", toolName,
		"correlation_id", corrID,
		"panic", rec,
		"stack", string(debug.Stack()))
	if errPtr != nil {
		*errPtr = fmt.Errorf("%s: internal error (correlation_id=%s)", toolName, corrID)
	}
}

// newCorrelationID returns a short hex string for log correlation. Falls back
// to a timestamp-based ID if crypto/rand is unavailable (vanishingly rare).
func newCorrelationID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("ts-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// logExecution logs tool execution with arg- and result-specific context.
func (h *HandlerRegistry) logExecution(spec ToolSpec, args, result any) {
	attrs := []any{"tool", spec.Name, "category", spec.Category}
	attrs = append(attrs, argAttrs(args)...)
	attrs = append(attrs, resultAttrs(result)...)
	h.logger.Info("Tool executed", attrs...)
}

// argAttrs extracts log attributes specific to known argument types. Returns
// nil for types that don't carry loggable context.
func argAttrs(args any) []any {
	if attrs := readArgAttrs(args); attrs != nil {
		return attrs
	}
	return writeArgAttrs(args)
}

// readArgAttrs covers read/list argument types.
func readArgAttrs(args any) []any {
	switch a := args.(type) {
	case miro.ListBoardsArgs:
		if a.Query != "" {
			return []any{"query", a.Query}
		}
	case miro.GetBoardArgs:
		return []any{"board_id", a.BoardID}
	case miro.ListItemsArgs:
		return []any{"board_id", a.BoardID, "type", a.Type}
	}
	return nil
}

// writeArgAttrs covers create/delete argument types.
func writeArgAttrs(args any) []any {
	switch a := args.(type) {
	case miro.CreateStickyArgs:
		return []any{"board_id", a.BoardID, "content_len", len(a.Content)}
	case miro.CreateShapeArgs:
		return []any{"board_id", a.BoardID, "shape", a.Shape}
	case miro.BulkCreateArgs:
		return []any{"board_id", a.BoardID, "items_count", len(a.Items)}
	case miro.DeleteItemArgs:
		return []any{"board_id", a.BoardID, "item_id", a.ItemID}
	case miro.GenerateDiagramArgs:
		return []any{"board_id", a.BoardID, "diagram_len", len(a.Diagram)}
	}
	return nil
}

// resultAttrs extracts log attributes specific to known result types. Returns
// nil for types that don't carry loggable context.
func resultAttrs(result any) []any {
	switch r := result.(type) {
	case miro.ListBoardsResult:
		return []any{"boards_count", r.Count}
	case miro.ListItemsResult:
		return []any{"items_count", r.Count}
	case miro.CreateStickyResult:
		return []any{"item_id", r.ID}
	case miro.CreateShapeResult:
		return []any{"item_id", r.ID}
	case miro.BulkCreateResult:
		return []any{"created", r.Created, "errors", len(r.Errors)}
	case miro.DeleteItemResult:
		return []any{"success", r.Success}
	case miro.GenerateDiagramResult:
		return []any{"nodes", r.NodesCreated, "connectors", r.ConnectorsCreated}
	}
	return nil
}
