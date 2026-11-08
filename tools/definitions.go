// Package tools provides MCP tool definitions and handlers for the Miro MCP server.
//
// This package defines all available tools that can be invoked through the
// Model Context Protocol (MCP). Each tool corresponds to a Miro API operation
// and includes metadata for LLM-friendly descriptions.
//
// # Tool Categories
//
//   - boards: List, create, copy, delete boards
//   - create: Create items (stickies, shapes, text, connectors, frames, etc.)
//   - read: List and get item details
//   - tags: Create, list, attach, and detach tags
//   - update: Modify existing items
//   - delete: Remove items from boards
//
// # Adding New Tools
//
// To add a new tool:
//  1. Add Args/Result types in miro/types.go
//  2. Add the method in miro/client.go
//  3. Add a ToolSpec entry to the matching definitions_*.go category file
//     (a new category also needs its slice listed in AllTools)
//  4. Register the method in handlers.go
package tools

import "slices"

// ToolSpec defines a tool's metadata for declarative registration.
type ToolSpec struct {
	// Name is the MCP tool name (e.g., "miro_list_boards")
	Name string

	// Method is the miro.Client method name (e.g., "ListBoards")
	Method string

	// Description is the tool description shown to LLMs
	Description string

	// Title is the human-readable tool title for annotations
	Title string

	// Category groups tools logically
	Category string

	// ReadOnly indicates the tool doesn't modify Miro state
	ReadOnly bool

	// Destructive indicates the tool can delete data
	Destructive bool

	// Idempotent indicates repeated calls have the same effect
	Idempotent bool

	// HeaderParams maps an input-schema property name to the Mcp-Param-*
	// header suffix that mirrors it over Streamable HTTP (SEP-2243
	// x-mcp-header). Example: {"board_id": "Board-Id"} declares that the
	// board_id argument travels as the Mcp-Param-Board-Id header, letting an
	// intermediary route or observe per-board traffic without reading the
	// JSON-RPC body. Only primitive (string, integer, boolean) top-level
	// properties are valid; registration panics on anything else because the
	// SDK silently ignores malformed annotations. Enforcement is strict for
	// clients negotiating >= 2026-07-28 over HTTP: when the annotated
	// argument is present, the header must be present and agree, or the
	// transport rejects the call with -32020 HeaderMismatch.
	HeaderParams map[string]string
}

// AllTools contains all registered Miro MCP tools, in registration order.
// Tool descriptions are optimized for token efficiency and voice interaction.
// Each category lives in its own definitions_*.go file; keep this list in
// the order agents should see the tools (discovery first).
//
// Webhook Tools - REMOVED (Miro sunset Dec 5, 2025)
// Miro is discontinuing experimental webhooks on December 5, 2025.
// The /v2-experimental/webhooks/board_subscriptions endpoints no longer work.
// See: https://community.miro.com/developer-platform-and-apis-57/miro-webhooks-4281
var AllTools = slices.Concat(
	discoveryTools,
	boardTools,
	createTools,
	bulkTools,
	readTools,
	cardMediaTools,
	tagTools,
	paginationTools,
	updateDeleteTools,
	compositeTools,
	groupTools,
	boardMemberTools,
	mindmapTools,
	codeWidgetTools,
	commentTools,
	canvasSVGTools,
	exportTools,
	auditTools,
	docFormatTools,
	tableTools,
	itemsByTagTools,
	uploadTools,
	updateFromFileTools,
	flowchartShapeTools,
	diagramTools,
	appCardTools,
)

// ptr is a helper to create a pointer to a value.
func ptr[T any](v T) *T {
	return &v
}
