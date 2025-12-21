package tools

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/miro-mcp-server/miro"
)

// HandlerRegistry provides type-safe tool registration.
type HandlerRegistry struct {
	client   miro.MiroClient
	logger   *slog.Logger
	handlers map[string]func(server *mcp.Server, tool *mcp.Tool, spec ToolSpec)
}

// NewHandlerRegistry creates a new handler registry.
func NewHandlerRegistry(client miro.MiroClient, logger *slog.Logger) *HandlerRegistry {
	h := &HandlerRegistry{
		client: client,
		logger: logger,
	}
	h.handlers = h.buildHandlerMap()
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

		// Update/Delete tools
		"UpdateItem": makeHandler(h, h.client.UpdateItem),
		"DeleteItem": makeHandler(h, h.client.DeleteItem),

		// Composite tools
		"FindBoardByNameTool": makeHandler(h, h.client.FindBoardByNameTool),
		"GetBoardSummary":     makeHandler(h, h.client.GetBoardSummary),
		"CreateStickyGrid":    makeHandler(h, h.client.CreateStickyGrid),

		// Group tools
		"CreateGroup": makeHandler(h, h.client.CreateGroup),
		"Ungroup":     makeHandler(h, h.client.Ungroup),

		// Board member tools
		"ListBoardMembers": makeHandler(h, h.client.ListBoardMembers),
		"ShareBoard":       makeHandler(h, h.client.ShareBoard),

		// Mindmap tools
		"CreateMindmapNode": makeHandler(h, h.client.CreateMindmapNode),
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
func registerTool[Args, Result any](
	h *HandlerRegistry,
	server *mcp.Server,
	tool *mcp.Tool,
	spec ToolSpec,
	method func(context.Context, Args) (Result, error),
) {
	mcp.AddTool(server, tool, func(ctx context.Context, _ *mcp.CallToolRequest, args Args) (*mcp.CallToolResult, Result, error) {
		defer h.recoverPanic(spec.Name)

		result, err := method(ctx, args)
		if err != nil {
			var zero Result
			return nil, zero, fmt.Errorf("%s failed: %w", spec.Name, err)
		}

		h.logExecution(spec, args, result)
		return nil, result, nil
	})
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
	}

	h.logger.Info("Tool executed", attrs...)
}

