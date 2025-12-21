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
	client *miro.Client
	logger *slog.Logger
}

// NewHandlerRegistry creates a new handler registry.
func NewHandlerRegistry(client *miro.Client, logger *slog.Logger) *HandlerRegistry {
	return &HandlerRegistry{
		client: client,
		logger: logger,
	}
}

// RegisterAll registers all tools with the MCP server.
func (h *HandlerRegistry) RegisterAll(server *mcp.Server) {
	for _, spec := range AllTools {
		h.registerByName(server, spec)
	}
	h.logger.Info("Registered all Miro tools", "count", len(AllTools))
}

// registerByName dispatches to the correct typed registration function.
func (h *HandlerRegistry) registerByName(server *mcp.Server, spec ToolSpec) {
	tool := h.buildTool(spec)

	switch spec.Method {
	// Board tools
	case "ListBoards":
		h.register(server, tool, spec, h.client.ListBoards)
	case "GetBoard":
		h.register(server, tool, spec, h.client.GetBoard)

	// Create tools
	case "CreateSticky":
		h.register(server, tool, spec, h.client.CreateSticky)
	case "CreateShape":
		h.register(server, tool, spec, h.client.CreateShape)
	case "CreateText":
		h.register(server, tool, spec, h.client.CreateText)
	case "CreateConnector":
		h.register(server, tool, spec, h.client.CreateConnector)
	case "CreateFrame":
		h.register(server, tool, spec, h.client.CreateFrame)
	case "BulkCreate":
		h.register(server, tool, spec, h.client.BulkCreate)
	case "CreateCard":
		h.register(server, tool, spec, h.client.CreateCard)
	case "CreateImage":
		h.register(server, tool, spec, h.client.CreateImage)
	case "CreateDocument":
		h.register(server, tool, spec, h.client.CreateDocument)
	case "CreateEmbed":
		h.register(server, tool, spec, h.client.CreateEmbed)

	// Read tools
	case "ListItems":
		h.register(server, tool, spec, h.client.ListItems)
	case "GetItem":
		h.register(server, tool, spec, h.client.GetItem)
	case "SearchBoard":
		h.register(server, tool, spec, h.client.SearchBoard)
	case "ListAllItems":
		h.register(server, tool, spec, h.client.ListAllItems)

	// Tag tools
	case "CreateTag":
		h.register(server, tool, spec, h.client.CreateTag)
	case "ListTags":
		h.register(server, tool, spec, h.client.ListTags)
	case "AttachTag":
		h.register(server, tool, spec, h.client.AttachTag)
	case "DetachTag":
		h.register(server, tool, spec, h.client.DetachTag)
	case "GetItemTags":
		h.register(server, tool, spec, h.client.GetItemTags)

	// Update/Delete tools
	case "UpdateItem":
		h.register(server, tool, spec, h.client.UpdateItem)
	case "DeleteItem":
		h.register(server, tool, spec, h.client.DeleteItem)

	default:
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

// register is a generic helper that registers a tool with the MCP server.
func register[Args, Result any](
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

// Convenience function to call the generic register with method receiver
func (h *HandlerRegistry) register(server *mcp.Server, tool *mcp.Tool, spec ToolSpec, method any) {
	switch m := method.(type) {
	// Board tools
	case func(context.Context, miro.ListBoardsArgs) (miro.ListBoardsResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.GetBoardArgs) (miro.GetBoardResult, error):
		register(h, server, tool, spec, m)

	// Create tools
	case func(context.Context, miro.CreateStickyArgs) (miro.CreateStickyResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.CreateShapeArgs) (miro.CreateShapeResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.CreateTextArgs) (miro.CreateTextResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.CreateConnectorArgs) (miro.CreateConnectorResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.CreateFrameArgs) (miro.CreateFrameResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.BulkCreateArgs) (miro.BulkCreateResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.CreateCardArgs) (miro.CreateCardResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.CreateImageArgs) (miro.CreateImageResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.CreateDocumentArgs) (miro.CreateDocumentResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.CreateEmbedArgs) (miro.CreateEmbedResult, error):
		register(h, server, tool, spec, m)

	// Read tools
	case func(context.Context, miro.ListItemsArgs) (miro.ListItemsResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.GetItemArgs) (miro.GetItemResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.SearchBoardArgs) (miro.SearchBoardResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.ListAllItemsArgs) (miro.ListAllItemsResult, error):
		register(h, server, tool, spec, m)

	// Tag tools
	case func(context.Context, miro.CreateTagArgs) (miro.CreateTagResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.ListTagsArgs) (miro.ListTagsResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.AttachTagArgs) (miro.AttachTagResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.DetachTagArgs) (miro.DetachTagResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.GetItemTagsArgs) (miro.GetItemTagsResult, error):
		register(h, server, tool, spec, m)

	// Update/Delete tools
	case func(context.Context, miro.UpdateItemArgs) (miro.UpdateItemResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, miro.DeleteItemArgs) (miro.DeleteItemResult, error):
		register(h, server, tool, spec, m)

	default:
		h.logger.Error("Unknown method type, tool not registered", "tool", spec.Name)
	}
}
