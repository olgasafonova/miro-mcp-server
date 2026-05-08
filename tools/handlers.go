package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/miro-mcp-server/miro"
	"github.com/olgasafonova/miro-mcp-server/miro/audit"
	"github.com/olgasafonova/miro-mcp-server/miro/desirepath"
)

// HandlerRegistry provides type-safe tool registration.
type HandlerRegistry struct {
	client         miro.MiroClient
	logger         *slog.Logger
	auditLogger    audit.Logger
	desireLogger   *desirepath.Logger
	normalizers    []desirepath.Normalizer
	userID         string          // Miro user ID from token validation
	userEmail      string          // Miro user email from token validation
	shareAllowlist *ShareAllowlist // Domain allowlist for miro_share_board
	handlers       map[string]func(server *mcp.Server, tool *mcp.Tool, spec ToolSpec)
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

// WithShareAllowlist sets the email-domain allowlist enforced by the
// miro_share_board handler. When nil or empty, every share invitation is
// rejected with a clear error instructing the operator to set
// MIRO_SHARE_ALLOWED_DOMAINS.
func (h *HandlerRegistry) WithShareAllowlist(allowlist *ShareAllowlist) *HandlerRegistry {
	h.shareAllowlist = allowlist
	return h
}

// ShareBoard validates the invitation target against the configured domain
// allowlist before delegating to the underlying Miro client. The allowlist is
// the server-side guardrail against prompt-injection-driven exfiltration:
// board content (stickies, cards, documents) must not be able to trick the
// agent into inviting arbitrary external parties.
func (h *HandlerRegistry) ShareBoard(ctx context.Context, args miro.ShareBoardArgs) (miro.ShareBoardResult, error) {
	allowlist := h.shareAllowlist
	if allowlist == nil {
		// Defensive: if no allowlist was wired, behave as if empty (fail closed).
		allowlist = NewShareAllowlist(nil, "not configured")
	}
	if err := allowlist.Validate(args.Email); err != nil {
		return miro.ShareBoardResult{
			Success: false,
			Email:   args.Email,
			Role:    args.Role,
			Message: err.Error(),
		}, err
	}
	return h.client.ShareBoard(ctx, args)
}

// WithDesirePathLogger enables desire path normalization and logging.
// Normalizers are applied in order to raw request arguments before
// they reach the typed handler, silently fixing common agent mistakes.
func (h *HandlerRegistry) WithDesirePathLogger(dpLogger *desirepath.Logger, normalizers []desirepath.Normalizer) *HandlerRegistry {
	h.desireLogger = dpLogger
	h.normalizers = normalizers
	return h
}

// RegisterAll registers every tool in AllTools with the MCP server. Equivalent
// to RegisterProfile(server, ProfileFull). Kept for backward compatibility
// with callers that don't know about profiles yet.
func (h *HandlerRegistry) RegisterAll(server *mcp.Server) {
	h.RegisterProfile(server, ProfileFull)
}

// RegisterProfile registers a subset of AllTools determined by the profile.
// ProfileFull (the default) registers everything. ProfileEssentials registers
// only the discovery meta-tool plus the curated EssentialsToolNames list;
// agents discover the rest via miro_tool_search on demand.
func (h *HandlerRegistry) RegisterProfile(server *mcp.Server, profile Profile) {
	specs := ToolsForProfile(profile)
	for _, spec := range specs {
		h.registerTool(server, spec)
	}
	h.logger.Info("Registered Miro tools",
		"profile", string(profile),
		"count", len(specs),
		"total_available", len(AllTools))
}

// buildHandlerMap composes the per-category dispatch tables into one map of
// method name → registration function. Adding a new tool requires only one
// entry in the appropriate per-category builder below.
func (h *HandlerRegistry) buildHandlerMap() map[string]func(*mcp.Server, *mcp.Tool, ToolSpec) {
	all := map[string]func(*mcp.Server, *mcp.Tool, ToolSpec){}
	for _, sub := range []map[string]func(*mcp.Server, *mcp.Tool, ToolSpec){
		h.boardHandlers(),
		h.itemCreateHandlers(),
		h.itemAccessHandlers(),
		h.tagHandlers(),
		h.structureHandlers(),
		h.contentHandlers(),
		h.localHandlers(),
	} {
		for k, v := range sub {
			all[k] = v
		}
	}
	return all
}

// boardHandlers returns dispatch entries for board-level operations: CRUD,
// composite views, members, sharing, export, and diagram generation.
func (h *HandlerRegistry) boardHandlers() map[string]func(*mcp.Server, *mcp.Tool, ToolSpec) {
	return map[string]func(*mcp.Server, *mcp.Tool, ToolSpec){
		"ListBoards":  makeHandler(h, h.client.ListBoards),
		"GetBoard":    makeHandler(h, h.client.GetBoard),
		"CreateBoard": makeHandler(h, h.client.CreateBoard),
		"CopyBoard":   makeHandler(h, h.client.CopyBoard),
		"DeleteBoard": makeHandler(h, h.client.DeleteBoard),
		"UpdateBoard": makeHandler(h, h.client.UpdateBoard),

		"FindBoardByNameTool": makeHandler(h, h.client.FindBoardByNameTool),
		"GetBoardSummary":     makeHandler(h, h.client.GetBoardSummary),
		"GetBoardContent":     makeHandler(h, h.client.GetBoardContent),

		"ListBoardMembers": makeHandler(h, h.client.ListBoardMembers),
		// ShareBoard routes through h.ShareBoard (not h.client.ShareBoard) so
		// the domain allowlist is enforced before the Miro API call.
		"ShareBoard":        makeHandler(h, h.ShareBoard),
		"GetBoardMember":    makeHandler(h, h.client.GetBoardMember),
		"RemoveBoardMember": makeHandler(h, h.client.RemoveBoardMember),
		"UpdateBoardMember": makeHandler(h, h.client.UpdateBoardMember),

		"GetBoardPicture":     makeHandler(h, h.client.GetBoardPicture),
		"CreateExportJob":     makeHandler(h, h.client.CreateExportJob),
		"GetExportJobStatus":  makeHandler(h, h.client.GetExportJobStatus),
		"GetExportJobResults": makeHandler(h, h.client.GetExportJobResults),

		"GenerateDiagram": makeHandler(h, h.client.GenerateDiagram),
	}
}

// itemCreateHandlers returns dispatch entries for creating items on a board.
func (h *HandlerRegistry) itemCreateHandlers() map[string]func(*mcp.Server, *mcp.Tool, ToolSpec) {
	return map[string]func(*mcp.Server, *mcp.Tool, ToolSpec){
		"CreateSticky":         makeHandler(h, h.client.CreateSticky),
		"CreateShape":          makeHandler(h, h.client.CreateShape),
		"CreateText":           makeHandler(h, h.client.CreateText),
		"CreateConnector":      makeHandler(h, h.client.CreateConnector),
		"CreateFrame":          makeHandler(h, h.client.CreateFrame),
		"CreateCard":           makeHandler(h, h.client.CreateCard),
		"CreateImage":          makeHandler(h, h.client.CreateImage),
		"CreateDocument":       makeHandler(h, h.client.CreateDocument),
		"CreateEmbed":          makeHandler(h, h.client.CreateEmbed),
		"CreateStickyGrid":     makeHandler(h, h.client.CreateStickyGrid),
		"CreateFlowchartShape": makeHandler(h, h.client.CreateFlowchartShape),
		"BulkCreate":           makeHandler(h, h.client.BulkCreate),
	}
}

// itemAccessHandlers returns dispatch entries for reading, updating, and
// deleting items.
func (h *HandlerRegistry) itemAccessHandlers() map[string]func(*mcp.Server, *mcp.Tool, ToolSpec) {
	return map[string]func(*mcp.Server, *mcp.Tool, ToolSpec){
		"ListItems":     makeHandler(h, h.client.ListItems),
		"GetItem":       makeHandler(h, h.client.GetItem),
		"GetImage":      makeHandler(h, h.client.GetImage),
		"GetDocument":   makeHandler(h, h.client.GetDocument),
		"SearchBoard":   makeHandler(h, h.client.SearchBoard),
		"ListAllItems":  makeHandler(h, h.client.ListAllItems),
		"GetItemsByTag": makeHandler(h, h.client.GetItemsByTag),

		"UpdateItem":     makeHandler(h, h.client.UpdateItem),
		"UpdateSticky":   makeHandler(h, h.client.UpdateSticky),
		"UpdateShape":    makeHandler(h, h.client.UpdateShape),
		"UpdateText":     makeHandler(h, h.client.UpdateText),
		"UpdateCard":     makeHandler(h, h.client.UpdateCard),
		"UpdateImage":    makeHandler(h, h.client.UpdateImage),
		"UpdateDocument": makeHandler(h, h.client.UpdateDocument),
		"UpdateEmbed":    makeHandler(h, h.client.UpdateEmbed),

		"DeleteItem": makeHandler(h, h.client.DeleteItem),
		"BulkUpdate": makeHandler(h, h.client.BulkUpdate),
		"BulkDelete": makeHandler(h, h.client.BulkDelete),
	}
}

// tagHandlers returns dispatch entries for tag CRUD and item attachment.
func (h *HandlerRegistry) tagHandlers() map[string]func(*mcp.Server, *mcp.Tool, ToolSpec) {
	return map[string]func(*mcp.Server, *mcp.Tool, ToolSpec){
		"CreateTag":   makeHandler(h, h.client.CreateTag),
		"ListTags":    makeHandler(h, h.client.ListTags),
		"AttachTag":   makeHandler(h, h.client.AttachTag),
		"DetachTag":   makeHandler(h, h.client.DetachTag),
		"GetItemTags": makeHandler(h, h.client.GetItemTags),
		"GetTag":      makeHandler(h, h.client.GetTag),
		"UpdateTag":   makeHandler(h, h.client.UpdateTag),
		"DeleteTag":   makeHandler(h, h.client.DeleteTag),
	}
}

// structureHandlers returns dispatch entries for board structure: connectors,
// groups, mindmap nodes, and frames.
func (h *HandlerRegistry) structureHandlers() map[string]func(*mcp.Server, *mcp.Tool, ToolSpec) {
	return map[string]func(*mcp.Server, *mcp.Tool, ToolSpec){
		"ListConnectors":  makeHandler(h, h.client.ListConnectors),
		"GetConnector":    makeHandler(h, h.client.GetConnector),
		"UpdateConnector": makeHandler(h, h.client.UpdateConnector),
		"DeleteConnector": makeHandler(h, h.client.DeleteConnector),

		"CreateGroup":   makeHandler(h, h.client.CreateGroup),
		"ListGroups":    makeHandler(h, h.client.ListGroups),
		"GetGroup":      makeHandler(h, h.client.GetGroup),
		"GetGroupItems": makeHandler(h, h.client.GetGroupItems),
		"UpdateGroup":   makeHandler(h, h.client.UpdateGroup),
		"DeleteGroup":   makeHandler(h, h.client.DeleteGroup),

		"CreateMindmapNode": makeHandler(h, h.client.CreateMindmapNode),
		"GetMindmapNode":    makeHandler(h, h.client.GetMindmapNode),
		"ListMindmapNodes":  makeHandler(h, h.client.ListMindmapNodes),
		"DeleteMindmapNode": makeHandler(h, h.client.DeleteMindmapNode),

		"GetFrame":      makeHandler(h, h.client.GetFrame),
		"UpdateFrame":   makeHandler(h, h.client.UpdateFrame),
		"DeleteFrame":   makeHandler(h, h.client.DeleteFrame),
		"GetFrameItems": makeHandler(h, h.client.GetFrameItems),
	}
}

// contentHandlers returns dispatch entries for app cards, doc formats,
// tables, and file uploads.
func (h *HandlerRegistry) contentHandlers() map[string]func(*mcp.Server, *mcp.Tool, ToolSpec) {
	return map[string]func(*mcp.Server, *mcp.Tool, ToolSpec){
		"CreateAppCard": makeHandler(h, h.client.CreateAppCard),
		"GetAppCard":    makeHandler(h, h.client.GetAppCard),
		"UpdateAppCard": makeHandler(h, h.client.UpdateAppCard),
		"DeleteAppCard": makeHandler(h, h.client.DeleteAppCard),

		"CreateDocFormat": makeHandler(h, h.client.CreateDocFormat),
		"GetDocFormat":    makeHandler(h, h.client.GetDocFormat),
		"UpdateDocFormat": makeHandler(h, h.client.UpdateDocFormat),
		"DeleteDocFormat": makeHandler(h, h.client.DeleteDocFormat),

		"ListTables": makeHandler(h, h.client.ListTables),
		"GetTable":   makeHandler(h, h.client.GetTable),

		"UploadImage":            makeHandler(h, h.client.UploadImage),
		"UploadDocument":         makeHandler(h, h.client.UploadDocument),
		"UpdateImageFromFile":    makeHandler(h, h.client.UpdateImageFromFile),
		"UpdateDocumentFromFile": makeHandler(h, h.client.UpdateDocumentFromFile),
	}
}

// localHandlers returns dispatch entries for tools handled locally on the
// server (no Miro API call): audit log, desire path report, tool discovery.
func (h *HandlerRegistry) localHandlers() map[string]func(*mcp.Server, *mcp.Tool, ToolSpec) {
	return map[string]func(*mcp.Server, *mcp.Tool, ToolSpec){
		"GetAuditLog":         makeHandler(h, h.GetAuditLog),
		"GetDesirePathReport": makeHandler(h, h.GetDesirePathReport),
		"SearchTools":         makeHandler(h, h.SearchTools),
	}
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
//
// The dispatcher closure uses NAMED return values so the deferred recoverPanic
// can reassign `err` on panic. Without named returns, Go cannot mutate the
// return values from a deferred function and a panic-then-recover would surface
// as `(nil, zero, nil)` to the MCP caller — looking like a successful empty
// response. See HG-1 in rules/code-review-prompts.md.
func registerTool[Args, Result any](
	h *HandlerRegistry,
	server *mcp.Server,
	tool *mcp.Tool,
	spec ToolSpec,
	method func(context.Context, Args) (Result, error),
) {
	mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, args Args) (res *mcp.CallToolResult, out Result, err error) {
		defer h.recoverPanic(spec.Name, &err)

		// Apply desire path normalization to raw arguments
		args = normalizeArgs(h, spec.Name, req, args)

		start := time.Now()
		result, methodErr := method(ctx, args)
		duration := time.Since(start)

		// Create and log audit event
		event := h.createAuditEvent(spec, executionResult{
			args:     args,
			result:   result,
			err:      methodErr,
			duration: duration,
		})
		if auditErr := h.auditLogger.Log(ctx, event); auditErr != nil {
			h.logger.Warn("Failed to log audit event", "error", auditErr)
		}

		if methodErr != nil {
			var zero Result
			return nil, zero, fmt.Errorf("%s failed: %w", spec.Name, methodErr)
		}

		h.logExecution(spec, args, result)
		return nil, result, nil
	})
}

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
	switch a := args.(type) {
	case miro.ListBoardsArgs:
		if a.Query != "" {
			return []any{"query", a.Query}
		}
	case miro.GetBoardArgs:
		return []any{"board_id", a.BoardID}
	case miro.CreateStickyArgs:
		return []any{"board_id", a.BoardID, "content_len", len(a.Content)}
	case miro.CreateShapeArgs:
		return []any{"board_id", a.BoardID, "shape", a.Shape}
	case miro.ListItemsArgs:
		return []any{"board_id", a.BoardID, "type", a.Type}
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
