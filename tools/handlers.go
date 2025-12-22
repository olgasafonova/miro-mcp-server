package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/miro-mcp-server/miro"
	"github.com/olgasafonova/miro-mcp-server/miro/audit"
)

// HandlerRegistry provides type-safe tool registration.
type HandlerRegistry struct {
	client      miro.MiroClient
	logger      *slog.Logger
	auditLogger audit.Logger
	userID      string // Miro user ID from token validation
	userEmail   string // Miro user email from token validation
	handlers    map[string]func(server *mcp.Server, tool *mcp.Tool, spec ToolSpec)
}

// NewHandlerRegistry creates a new handler registry.
func NewHandlerRegistry(client miro.MiroClient, logger *slog.Logger) *HandlerRegistry {
	h := &HandlerRegistry{
		client:      client,
		logger:      logger,
		auditLogger: audit.NewNoopLogger(), // Default to noop
	}
	h.handlers = h.buildHandlerMap()
	return h
}

// WithAuditLogger sets the audit logger for the registry.
func (h *HandlerRegistry) WithAuditLogger(auditLogger audit.Logger) *HandlerRegistry {
	h.auditLogger = auditLogger
	return h
}

// WithUser sets the user info for audit events.
func (h *HandlerRegistry) WithUser(userID, userEmail string) *HandlerRegistry {
	h.userID = userID
	h.userEmail = userEmail
	return h
}

// RegisterAll registers all tools with the MCP server.
func (h *HandlerRegistry) RegisterAll(server *mcp.Server) {
	for _, spec := range AllTools {
		h.registerTool(server, spec)
	}
	h.logger.Info("Registered all Miro tools", "count", len(AllTools))
}

// buildHandlerMap creates a map of method names to registration functions.
// Adding a new tool requires only one entry here.
func (h *HandlerRegistry) buildHandlerMap() map[string]func(*mcp.Server, *mcp.Tool, ToolSpec) {
	return map[string]func(*mcp.Server, *mcp.Tool, ToolSpec){
		// Board tools
		"ListBoards":  makeHandler(h, h.client.ListBoards),
		"GetBoard":    makeHandler(h, h.client.GetBoard),
		"CreateBoard": makeHandler(h, h.client.CreateBoard),
		"CopyBoard":   makeHandler(h, h.client.CopyBoard),
		"DeleteBoard": makeHandler(h, h.client.DeleteBoard),
		"UpdateBoard": makeHandler(h, h.client.UpdateBoard),

		// Create tools
		"CreateSticky":    makeHandler(h, h.client.CreateSticky),
		"CreateShape":     makeHandler(h, h.client.CreateShape),
		"CreateText":      makeHandler(h, h.client.CreateText),
		"CreateConnector": makeHandler(h, h.client.CreateConnector),
		"CreateFrame":     makeHandler(h, h.client.CreateFrame),
		"BulkCreate":      makeHandler(h, h.client.BulkCreate),
		"CreateCard":      makeHandler(h, h.client.CreateCard),
		"CreateImage":     makeHandler(h, h.client.CreateImage),
		"CreateDocument":  makeHandler(h, h.client.CreateDocument),
		"CreateEmbed":     makeHandler(h, h.client.CreateEmbed),

		// Read tools
		"ListItems":    makeHandler(h, h.client.ListItems),
		"GetItem":      makeHandler(h, h.client.GetItem),
		"SearchBoard":  makeHandler(h, h.client.SearchBoard),
		"ListAllItems": makeHandler(h, h.client.ListAllItems),

		// Tag tools
		"CreateTag":   makeHandler(h, h.client.CreateTag),
		"ListTags":    makeHandler(h, h.client.ListTags),
		"AttachTag":   makeHandler(h, h.client.AttachTag),
		"DetachTag":   makeHandler(h, h.client.DetachTag),
		"GetItemTags": makeHandler(h, h.client.GetItemTags),
		"UpdateTag":   makeHandler(h, h.client.UpdateTag),
		"DeleteTag":   makeHandler(h, h.client.DeleteTag),

		// Connector tools
		"ListConnectors":  makeHandler(h, h.client.ListConnectors),
		"GetConnector":    makeHandler(h, h.client.GetConnector),
		"UpdateConnector": makeHandler(h, h.client.UpdateConnector),
		"DeleteConnector": makeHandler(h, h.client.DeleteConnector),

		// Update/Delete tools
		"UpdateItem": makeHandler(h, h.client.UpdateItem),
		"DeleteItem": makeHandler(h, h.client.DeleteItem),

		// Composite tools
		"FindBoardByNameTool": makeHandler(h, h.client.FindBoardByNameTool),
		"GetBoardSummary":     makeHandler(h, h.client.GetBoardSummary),
		"CreateStickyGrid":    makeHandler(h, h.client.CreateStickyGrid),

		// Group tools
		"CreateGroup":    makeHandler(h, h.client.CreateGroup),
		"Ungroup":        makeHandler(h, h.client.Ungroup),
		"ListGroups":     makeHandler(h, h.client.ListGroups),
		"GetGroup":       makeHandler(h, h.client.GetGroup),
		"GetGroupItems":  makeHandler(h, h.client.GetGroupItems),
		"DeleteGroup":    makeHandler(h, h.client.DeleteGroup),

		// Board member tools
		"ListBoardMembers":   makeHandler(h, h.client.ListBoardMembers),
		"ShareBoard":         makeHandler(h, h.client.ShareBoard),
		"GetBoardMember":     makeHandler(h, h.client.GetBoardMember),
		"RemoveBoardMember":  makeHandler(h, h.client.RemoveBoardMember),
		"UpdateBoardMember":  makeHandler(h, h.client.UpdateBoardMember),

		// Mindmap tools
		"CreateMindmapNode": makeHandler(h, h.client.CreateMindmapNode),

		// Diagram generation tools
		"GenerateDiagram": makeHandler(h, h.client.GenerateDiagram),

		// Export tools
		"GetBoardPicture":     makeHandler(h, h.client.GetBoardPicture),
		"CreateExportJob":     makeHandler(h, h.client.CreateExportJob),
		"GetExportJobStatus":  makeHandler(h, h.client.GetExportJobStatus),
		"GetExportJobResults": makeHandler(h, h.client.GetExportJobResults),

		// Audit tools (local, not Miro API)
		"GetAuditLog": makeHandler(h, h.GetAuditLog),

		// App card tools
		"CreateAppCard": makeHandler(h, h.client.CreateAppCard),
		"GetAppCard":    makeHandler(h, h.client.GetAppCard),
		"UpdateAppCard": makeHandler(h, h.client.UpdateAppCard),
		"DeleteAppCard": makeHandler(h, h.client.DeleteAppCard),

		// Webhook tools - REMOVED (Miro sunset Dec 5, 2025)
		// The /v2-experimental/webhooks/board_subscriptions endpoints no longer work.
	}
}

// GetAuditLog queries the local audit log.
func (h *HandlerRegistry) GetAuditLog(ctx context.Context, args miro.GetAuditLogArgs) (miro.GetAuditLogResult, error) {
	// Parse time range
	var since, until time.Time
	if args.Since != "" {
		t, err := time.Parse(time.RFC3339, args.Since)
		if err != nil {
			return miro.GetAuditLogResult{}, fmt.Errorf("invalid 'since' time format: %w", err)
		}
		since = t
	}
	if args.Until != "" {
		t, err := time.Parse(time.RFC3339, args.Until)
		if err != nil {
			return miro.GetAuditLogResult{}, fmt.Errorf("invalid 'until' time format: %w", err)
		}
		until = t
	}

	// Set defaults
	limit := args.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	// Build query options
	opts := audit.QueryOptions{
		Since:   since,
		Until:   until,
		Tool:    args.Tool,
		BoardID: args.BoardID,
		Action:  audit.Action(args.Action),
		Success: args.Success,
		Limit:   limit,
	}

	// Execute query
	result, err := h.auditLogger.Query(ctx, opts)
	if err != nil {
		return miro.GetAuditLogResult{}, fmt.Errorf("audit query failed: %w", err)
	}

	// Convert to response type
	events := make([]miro.AuditLogEvent, len(result.Events))
	for i, e := range result.Events {
		events[i] = miro.AuditLogEvent{
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

	return miro.GetAuditLogResult{
		Events:  events,
		Total:   result.Total,
		HasMore: result.HasMore,
		Message: fmt.Sprintf("Found %d audit events", len(events)),
	}, nil
}

// makeHandler creates a registration function for a typed client method.
func makeHandler[Args, Result any](
	h *HandlerRegistry,
	method func(context.Context, Args) (Result, error),
) func(*mcp.Server, *mcp.Tool, ToolSpec) {
	return func(server *mcp.Server, tool *mcp.Tool, spec ToolSpec) {
		registerTool(h, server, tool, spec, method)
	}
}

// registerTool looks up and calls the registration function for a tool spec.
func (h *HandlerRegistry) registerTool(server *mcp.Server, spec ToolSpec) {
	tool := h.buildTool(spec)
	if handler, ok := h.handlers[spec.Method]; ok {
		handler(server, tool, spec)
	} else {
		h.logger.Error("Unknown method, tool not registered", "method", spec.Method, "tool", spec.Name)
	}
}

// buildTool creates an mcp.Tool from a ToolSpec.
func (h *HandlerRegistry) buildTool(spec ToolSpec) *mcp.Tool {
	annotations := &mcp.ToolAnnotations{
		Title:          spec.Title,
		ReadOnlyHint:   spec.ReadOnly,
		IdempotentHint: spec.Idempotent,
	}
	if spec.Destructive {
		annotations.DestructiveHint = ptr(true)
	}

	return &mcp.Tool{
		Name:        spec.Name,
		Description: spec.Description,
		Annotations: annotations,
	}
}

// registerTool is a generic helper that registers a tool with the MCP server.
func registerTool[Args, Result any](
	h *HandlerRegistry,
	server *mcp.Server,
	tool *mcp.Tool,
	spec ToolSpec,
	method func(context.Context, Args) (Result, error),
) {
	mcp.AddTool(server, tool, func(ctx context.Context, _ *mcp.CallToolRequest, args Args) (*mcp.CallToolResult, Result, error) {
		defer h.recoverPanic(spec.Name)

		start := time.Now()
		result, err := method(ctx, args)
		duration := time.Since(start)

		// Create and log audit event
		event := h.createAuditEvent(spec, args, result, err, duration)
		if auditErr := h.auditLogger.Log(ctx, event); auditErr != nil {
			h.logger.Warn("Failed to log audit event", "error", auditErr)
		}

		if err != nil {
			var zero Result
			return nil, zero, fmt.Errorf("%s failed: %w", spec.Name, err)
		}

		h.logExecution(spec, args, result)
		return nil, result, nil
	})
}

// createAuditEvent creates an audit event from tool execution details.
func (h *HandlerRegistry) createAuditEvent(spec ToolSpec, args, result any, err error, duration time.Duration) audit.Event {
	event := audit.NewEvent(spec.Name, spec.Method, audit.DetectAction(spec.Method)).
		WithUser(h.userID, h.userEmail).
		WithDuration(duration)

	// Extract board_id and item_id from args
	if input := argsToMap(args); input != nil {
		event.WithInput(input)
		if boardID, ok := input["board_id"].(string); ok {
			event.WithBoard(boardID)
		}
		if itemID, ok := input["item_id"].(string); ok {
			event.WithItem(itemID, "")
		}
	}

	// Mark success or failure
	if err != nil {
		event.Failure(err)
	} else {
		event.Success()
		// Extract item_id and count from result
		if resultMap := argsToMap(result); resultMap != nil {
			if itemID, ok := resultMap["id"].(string); ok {
				event.WithItem(itemID, "")
			}
			if created, ok := resultMap["created"].(float64); ok {
				event.WithItemCount(int(created))
			}
		}
	}

	return event.Build()
}

// argsToMap converts a struct to a map for audit logging.
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

// recoverPanic recovers from panics in tool handlers.
func (h *HandlerRegistry) recoverPanic(toolName string) {
	if rec := recover(); rec != nil {
		h.logger.Error("Panic recovered",
			"tool", toolName,
			"panic", rec,
			"stack", string(debug.Stack()))
	}
}

// logExecution logs tool execution details.
func (h *HandlerRegistry) logExecution(spec ToolSpec, args, result any) {
	attrs := []any{"tool", spec.Name, "category", spec.Category}

	// Add context from specific arg types
	switch a := args.(type) {
	case miro.ListBoardsArgs:
		if a.Query != "" {
			attrs = append(attrs, "query", a.Query)
		}
	case miro.GetBoardArgs:
		attrs = append(attrs, "board_id", a.BoardID)
	case miro.CreateStickyArgs:
		attrs = append(attrs, "board_id", a.BoardID, "content_len", len(a.Content))
	case miro.CreateShapeArgs:
		attrs = append(attrs, "board_id", a.BoardID, "shape", a.Shape)
	case miro.ListItemsArgs:
		attrs = append(attrs, "board_id", a.BoardID, "type", a.Type)
	case miro.BulkCreateArgs:
		attrs = append(attrs, "board_id", a.BoardID, "items_count", len(a.Items))
	case miro.DeleteItemArgs:
		attrs = append(attrs, "board_id", a.BoardID, "item_id", a.ItemID)
	case miro.GenerateDiagramArgs:
		attrs = append(attrs, "board_id", a.BoardID, "diagram_len", len(a.Diagram))
	}

	// Add context from result types
	switch r := result.(type) {
	case miro.ListBoardsResult:
		attrs = append(attrs, "boards_count", r.Count)
	case miro.ListItemsResult:
		attrs = append(attrs, "items_count", r.Count)
	case miro.CreateStickyResult:
		attrs = append(attrs, "item_id", r.ID)
	case miro.CreateShapeResult:
		attrs = append(attrs, "item_id", r.ID)
	case miro.BulkCreateResult:
		attrs = append(attrs, "created", r.Created, "errors", len(r.Errors))
	case miro.DeleteItemResult:
		attrs = append(attrs, "success", r.Success)
	case miro.GenerateDiagramResult:
		attrs = append(attrs, "nodes", r.NodesCreated, "connectors", r.ConnectorsCreated)
	}

	h.logger.Info("Tool executed", attrs...)
}

