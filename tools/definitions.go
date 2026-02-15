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
//  3. Add a ToolSpec entry in AllTools
//  4. Register the method in handlers.go
package tools

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
}

// AllTools contains all registered Miro MCP tools.
// Tool descriptions are optimized for token efficiency and voice interaction.
var AllTools = []ToolSpec{
	// ==========================================================================
	// Board Tools
	// ==========================================================================
	{
		Name:     "miro_list_boards",
		Method:   "ListBoards",
		Title:    "List Boards",
		Category: "boards",
		ReadOnly: true,
		Description: `List Miro boards accessible to the user. Use board ID for subsequent operations. For a specific board by name, use ` + "`miro_find_board`" + ` instead.

VOICE-FRIENDLY: "Found 5 boards: Design Sprint, Product Roadmap, Team Retro..."`,
	},
	{
		Name:        "miro_get_board",
		Method:      "GetBoard",
		Title:       "Get Board Details",
		Category:    "boards",
		ReadOnly:    true,
		Description: `Get details of a specific Miro board (name, description, owner, creation date).`,
	},
	{
		Name:     "miro_create_board",
		Method:   "CreateBoard",
		Title:    "Create Board",
		Category: "boards",
		Description: `Create a new Miro board.

VOICE-FRIENDLY: "Created board 'Sprint Planning'"`,
	},
	{
		Name:     "miro_copy_board",
		Method:   "CopyBoard",
		Title:    "Copy Board",
		Category: "boards",
		Description: `Copy an existing Miro board.

VOICE-FRIENDLY: "Copied board to 'Sprint Planning Copy'"`,
	},
	{
		Name:        "miro_delete_board",
		Method:      "DeleteBoard",
		Title:       "Delete Board",
		Category:    "boards",
		Destructive: true,
		Description: `Delete a Miro board permanently.

WARNING: Cannot be undone. Use dry_run=true to preview first.`,
	},
	{
		Name:       "miro_update_board",
		Method:     "UpdateBoard",
		Title:      "Update Board",
		Category:   "boards",
		Idempotent: true,
		Description: `Update a Miro board's name or description. At least one field must be provided.

VOICE-FRIENDLY: "Updated board name to 'Sprint Planning Q1'"`,
	},

	// ==========================================================================
	// Create Tools - Sticky Notes, Shapes, Text
	// ==========================================================================
	{
		Name:     "miro_create_sticky",
		Method:   "CreateSticky",
		Title:    "Create Sticky Note",
		Category: "create",
		Description: `Create a sticky note on a Miro board. For multiple stickies in a grid, use miro_create_sticky_grid. For batch creation of mixed items, use miro_bulk_create.

USE WHEN: "add a sticky", "create note saying X", "put a yellow sticky"

VOICE-FRIENDLY: "Created yellow sticky 'Action item: Review design'"`,
	},
	{
		Name:     "miro_create_shape",
		Method:   "CreateShape",
		Title:    "Create Shape",
		Category: "create",
		Description: `Create a shape on a Miro board.

SHAPES: rectangle, round_rectangle, circle, triangle, rhombus, parallelogram, trapezoid, pentagon, hexagon, star, flow_chart_predefined_process, wedge_round_rectangle_callout`,
	},
	{
		Name:        "miro_create_text",
		Method:      "CreateText",
		Title:       "Create Text",
		Category:    "create",
		Description: `Create a text element on a Miro board.`,
	},
	{
		Name:        "miro_create_connector",
		Method:      "CreateConnector",
		Title:       "Create Connector",
		Category:    "create",
		Description: `Create a connector line between two items. Styles: straight, elbowed (default), curved. Caps: none, arrow, stealth, diamond, filled_diamond, oval, filled_oval, triangle, filled_triangle.`,
	},
	{
		Name:        "miro_create_frame",
		Method:      "CreateFrame",
		Title:       "Create Frame",
		Category:    "create",
		Description: `Create a frame container to group items visually. For logical grouping without a visual border, use miro_create_group.`,
	},
	{
		Name:     "miro_get_frame",
		Method:   "GetFrame",
		Title:    "Get Frame Details",
		Category: "read",
		ReadOnly: true,
		Description: `Get full details of a specific frame by ID. To get items inside the frame, use ` + "`miro_get_frame_items`" + `.

VOICE-FRIENDLY: "Frame 'Sprint Planning' is 800x600 with 12 items inside"`,
	},
	{
		Name:       "miro_update_frame",
		Method:     "UpdateFrame",
		Title:      "Update Frame",
		Category:   "update",
		Idempotent: true,
		Description: `Update a frame's title, position, size, or color. At least one field must be provided.

VOICE-FRIENDLY: "Updated frame title to 'Q1 Goals'"`,
	},
	{
		Name:        "miro_delete_frame",
		Method:      "DeleteFrame",
		Title:       "Delete Frame",
		Category:    "delete",
		Destructive: true,
		Description: `Delete a frame from a Miro board. Items inside are NOT deleted; they become ungrouped.

WARNING: Cannot be undone. Use dry_run=true to preview first.

VOICE-FRIENDLY: "Frame deleted successfully"`,
	},
	{
		Name:     "miro_get_frame_items",
		Method:   "GetFrameItems",
		Title:    "Get Frame Items",
		Category: "read",
		ReadOnly: true,
		Description: `Get all items contained within a specific frame. Filterable by type. For items in a logical group, use miro_get_group_items.

VOICE-FRIENDLY: "Frame has 8 items: 5 stickies, 2 shapes, 1 text"`,
	},

	// ==========================================================================
	// Bulk Operations
	// ==========================================================================
	{
		Name:     "miro_bulk_create",
		Method:   "BulkCreate",
		Title:    "Bulk Create Items",
		Category: "create",
		Description: `Create multiple items at once (max 20). For creating only stickies in a grid, use miro_create_sticky_grid instead.

USE WHEN: "add these 5 stickies", "create items for each of these", "batch add"

VOICE-FRIENDLY: "Created 5 items on the board"`,
	},
	{
		Name:     "miro_bulk_update",
		Method:   "BulkUpdate",
		Title:    "Bulk Update Items",
		Category: "update",
		Description: `Update multiple items at once (max 20). Only provide fields you want to change.

VOICE-FRIENDLY: "Updated 5 items on the board"`,
	},
	{
		Name:        "miro_bulk_delete",
		Method:      "BulkDelete",
		Title:       "Bulk Delete Items",
		Category:    "delete",
		Destructive: true,
		Description: `Delete multiple items at once (max 20).

WARNING: Cannot be undone. Use dry_run=true to preview first.

VOICE-FRIENDLY: "Deleted 5 items from the board"`,
	},

	// ==========================================================================
	// Read/List Tools
	// ==========================================================================
	{
		Name:     "miro_list_items",
		Method:   "ListItems",
		Title:    "List Board Items",
		Category: "read",
		ReadOnly: true,
		Description: `List items on a Miro board (max 50). For ALL items with auto-pagination, use miro_list_all_items. For text search, use miro_search_board.

USE WHEN: "what's on the board", "show all stickies", "list shapes"`,
	},
	{
		Name:     "miro_get_item",
		Method:   "GetItem",
		Title:    "Get Item Details",
		Category: "read",
		ReadOnly: true,
		Description: `Get full details of a specific item by ID. If you don't have the item ID, use ` + "`miro_search_board`" + ` to find it or ` + "`miro_list_items`" + ` to browse.

VOICE-FRIENDLY: "That sticky says 'Review Q4 goals' and was created by John yesterday"`,
	},
	{
		Name:     "miro_search_board",
		Method:   "SearchBoard",
		Title:    "Search Board Content",
		Category: "read",
		ReadOnly: true,
		Description: `Search for items containing specific text on a board (case-insensitive). For listing without search, use miro_list_items.

USE WHEN: "find items about X", "search for budget", "which stickies mention deadline"

VOICE-FRIENDLY: "Found 3 stickies mentioning 'budget'"`,
	},

	// ==========================================================================
	// Card, Image, Document, Embed Tools
	// ==========================================================================
	{
		Name:     "miro_create_card",
		Method:   "CreateCard",
		Title:    "Create Card",
		Category: "create",
		Description: `Create a card on a Miro board. Cards have title, description, and due dates. For cards with custom fields and status, use miro_create_app_card.

USE WHEN: "add a card", "create a task card", "card with due date"

VOICE-FRIENDLY: "Created card 'Review design specs'"`,
	},
	{
		Name:        "miro_create_image",
		Method:      "CreateImage",
		Title:       "Create Image",
		Category:    "create",
		Description: `Add an image to a Miro board from a URL. URL must be publicly accessible.`,
	},
	{
		Name:     "miro_get_image",
		Method:   "GetImage",
		Title:    "Get Image Details",
		Category: "read",
		ReadOnly: true,
		Description: `Get details of an image on a Miro board, including its Miro-hosted URL, title, and dimensions. Use the image_url to download or reference the image. For document items, use ` + "`miro_get_document`" + ` instead.

USE WHEN: "get image URL", "what image is this", "image details"

VOICE-FRIENDLY: "Image 'Logo' is 800x600 at position (100, 200)"`,
	},
	{
		Name:        "miro_create_document",
		Method:      "CreateDocument",
		Title:       "Create Document",
		Category:    "create",
		Description: `Add a document (PDF, etc.) to a Miro board from a URL. URL must be publicly accessible.`,
	},
	{
		Name:     "miro_get_document",
		Method:   "GetDocument",
		Title:    "Get Document Details",
		Category: "read",
		ReadOnly: true,
		Description: `Get details of a document on a Miro board, including its Miro-hosted URL and title. For image items, use ` + "`miro_get_image`" + ` instead.

USE WHEN: "get document details", "what document is this", "document URL"

VOICE-FRIENDLY: "Document 'Q4 Report' hosted at Miro"`,
	},
	{
		Name:        "miro_create_embed",
		Method:      "CreateEmbed",
		Title:       "Create Embed",
		Category:    "create",
		Description: `Embed external content on a Miro board. Supports YouTube, Vimeo, Figma, Google Docs, Loom, and more.`,
	},

	// ==========================================================================
	// Tag Tools
	// ==========================================================================
	{
		Name:     "miro_create_tag",
		Method:   "CreateTag",
		Title:    "Create Tag",
		Category: "tags",
		Description: `Create a tag on a Miro board. Colors: red, magenta, violet, blue, cyan, green, yellow, gray, light_green, dark_green, dark_blue, dark_gray, black.

VOICE-FRIENDLY: "Created red tag 'Urgent'"`,
	},
	{
		Name:        "miro_list_tags",
		Method:      "ListTags",
		Title:       "List Tags",
		Category:    "tags",
		ReadOnly:    true,
		Description: `List all tags on a Miro board.`,
	},
	{
		Name:        "miro_attach_tag",
		Method:      "AttachTag",
		Title:       "Attach Tag",
		Category:    "tags",
		Description: `Attach a tag to a sticky note or card. Tags only work on sticky_note and card items.`,
	},
	{
		Name:        "miro_detach_tag",
		Method:      "DetachTag",
		Title:       "Remove Tag",
		Category:    "tags",
		Description: `Remove a tag from a sticky note or card.`,
	},
	{
		Name:        "miro_get_item_tags",
		Method:      "GetItemTags",
		Title:       "Get Item Tags",
		Category:    "tags",
		ReadOnly:    true,
		Description: `List tags attached to a specific item.`,
	},
	{
		Name:     "miro_get_tag",
		Method:   "GetTag",
		Title:    "Get Tag",
		Category: "tags",
		ReadOnly: true,
		Description: `Get details of a specific tag by ID.

VOICE-FRIENDLY: "Tag 'Urgent' is red"`,
	},
	{
		Name:       "miro_update_tag",
		Method:     "UpdateTag",
		Title:      "Update Tag",
		Category:   "tags",
		Idempotent: true,
		Description: `Update a tag's title or color. At least one must be provided.

VOICE-FRIENDLY: "Updated tag to 'Done' with green color"`,
	},
	{
		Name:        "miro_delete_tag",
		Method:      "DeleteTag",
		Title:       "Delete Tag",
		Category:    "tags",
		Destructive: true,
		Description: `Delete a tag from a board. Removes the tag from all items.

WARNING: Cannot be undone. Use dry_run=true to preview first.

VOICE-FRIENDLY: "Tag deleted successfully"`,
	},

	// ==========================================================================
	// Pagination Tools
	// ==========================================================================
	{
		Name:     "miro_list_all_items",
		Method:   "ListAllItems",
		Title:    "List All Items (Paginated)",
		Category: "read",
		ReadOnly: true,
		Description: `Retrieve ALL items from a board with automatic pagination (up to 10000). For quick lookups (max 50), use miro_list_items instead.

USE WHEN: "get everything on board", "list all items", "export board contents"

VOICE-FRIENDLY: "Retrieved 847 items in 9 pages"`,
	},

	// ==========================================================================
	// Update/Delete Tools
	// ==========================================================================
	{
		Name:       "miro_update_item",
		Method:     "UpdateItem",
		Title:      "Update Item",
		Category:   "update",
		Idempotent: true,
		Description: `Update any item's content, position, or style. For sticky-specific options (color, shape), use ` + "`miro_update_sticky`" + `. For card fields, use ` + "`miro_update_card`" + `. For shape styling, use ` + "`miro_update_shape`" + `.

USE WHEN: "change sticky text", "move this item", "update the color"`,
	},
	{
		Name:       "miro_update_sticky",
		Method:     "UpdateSticky",
		Title:      "Update Sticky Note",
		Category:   "update",
		Idempotent: true,
		Description: `Update a sticky note with type-specific options (shape: square/rectangle, sticky colors). For generic updates, use miro_update_item.

USE WHEN: "change sticky color", "update sticky to square", "resize sticky note"

VOICE-FRIENDLY: "Updated sticky to yellow square"`,
	},
	{
		Name:       "miro_update_shape",
		Method:     "UpdateShape",
		Title:      "Update Shape",
		Category:   "update",
		Idempotent: true,
		Description: `Update a shape with type-specific options (fill_color, text_color, shape type). For generic updates, use miro_update_item.

VOICE-FRIENDLY: "Updated shape to blue circle"`,
	},
	{
		Name:       "miro_update_text",
		Method:     "UpdateText",
		Title:      "Update Text",
		Category:   "update",
		Idempotent: true,
		Description: `Update a text element (content, font_size, color, position).

VOICE-FRIENDLY: "Updated text to 'New Title'"`,
	},
	{
		Name:       "miro_update_card",
		Method:     "UpdateCard",
		Title:      "Update Card",
		Category:   "update",
		Idempotent: true,
		Description: `Update a card (title, description, due_date, position).

VOICE-FRIENDLY: "Updated card title to 'Review PR'"`,
	},
	{
		Name:       "miro_update_image",
		Method:     "UpdateImage",
		Title:      "Update Image",
		Category:   "update",
		Idempotent: true,
		Description: `Update an image (title, url, position, width).

VOICE-FRIENDLY: "Updated image title to 'Logo'"`,
	},
	{
		Name:       "miro_update_document",
		Method:     "UpdateDocument",
		Title:      "Update Document",
		Category:   "update",
		Idempotent: true,
		Description: `Update a document (title, url, position, width).

VOICE-FRIENDLY: "Updated document title"`,
	},
	{
		Name:       "miro_update_embed",
		Method:     "UpdateEmbed",
		Title:      "Update Embed",
		Category:   "update",
		Idempotent: true,
		Description: `Update an embed (url, mode: inline/modal, dimensions, position).

VOICE-FRIENDLY: "Updated embed settings"`,
	},
	{
		Name:        "miro_delete_item",
		Method:      "DeleteItem",
		Title:       "Delete Item",
		Category:    "delete",
		Destructive: true,
		Description: `Delete an item from a Miro board.

WARNING: Cannot be undone. Use dry_run=true to preview first.`,
	},
	{
		Name:        "miro_update_connector",
		Method:      "UpdateConnector",
		Title:       "Update Connector",
		Category:    "update",
		Idempotent:  true,
		Description: `Update a connector's style (straight/elbowed/curved), caps, caption, or color.`,
	},
	{
		Name:        "miro_delete_connector",
		Method:      "DeleteConnector",
		Title:       "Delete Connector",
		Category:    "delete",
		Destructive: true,
		Description: `Delete a connector from a Miro board.

WARNING: Cannot be undone. Use dry_run=true to preview first.

VOICE-FRIENDLY: "Connector deleted successfully"`,
	},
	{
		Name:     "miro_list_connectors",
		Method:   "ListConnectors",
		Title:    "List Connectors",
		Category: "read",
		ReadOnly: true,
		Description: `List all connectors (lines/arrows) on a Miro board.

VOICE-FRIENDLY: "Found 12 connectors on the board"`,
	},
	{
		Name:     "miro_get_connector",
		Method:   "GetConnector",
		Title:    "Get Connector Details",
		Category: "read",
		ReadOnly: true,
		Description: `Get full details of a specific connector by ID.

VOICE-FRIENDLY: "This connector links Item A to Item B with a curved arrow"`,
	},

	// ==========================================================================
	// Composite Tools
	// ==========================================================================
	{
		Name:     "miro_find_board",
		Method:   "FindBoardByNameTool",
		Title:    "Find Board by Name",
		Category: "boards",
		ReadOnly: true,
		Description: `Find a Miro board by name (case-insensitive, partial match). Returns board ID for subsequent operations. For listing all boards, use miro_list_boards.

USE WHEN: "find board named X", "get the Design Sprint board"

VOICE-FRIENDLY: "Found 'Design Sprint' board - ready to work on it"`,
	},
	{
		Name:     "miro_get_board_summary",
		Method:   "GetBoardSummary",
		Title:    "Get Board Summary",
		Category: "read",
		ReadOnly: true,
		Description: `Get board overview with item counts and statistics. For full content export, use miro_get_board_content instead.

USE WHEN: "summarize this board", "board stats", "what's the overview"

VOICE-FRIENDLY: "Design Sprint has 15 stickies, 8 shapes, and 3 frames - 26 items total"`,
	},
	{
		Name:     "miro_get_board_content",
		Method:   "GetBoardContent",
		Title:    "Get Board Content",
		Category: "read",
		ReadOnly: true,
		Description: `Get all board content for AI analysis and documentation generation. Returns items by type, frame hierarchy, connectors, and tags. For a quick summary, use miro_get_board_summary instead.

USE WHEN: "analyze this board", "generate documentation from board", "describe everything on this board"

VOICE-FRIENDLY: "Retrieved full content for 'Design Sprint': 26 items across 3 frames, 5 connectors, 2 tags"`,
	},
	{
		Name:     "miro_create_sticky_grid",
		Method:   "CreateStickyGrid",
		Title:    "Create Sticky Grid",
		Category: "create",
		Description: `Create multiple sticky notes arranged in a grid layout (max 50). For mixed item types, use miro_bulk_create.

USE WHEN: "add a grid of stickies", "create 6 stickies in rows", "make sticky notes for each idea"

VOICE-FRIENDLY: "Created 9 stickies in a 3x3 grid"`,
	},

	// ==========================================================================
	// Group Tools
	// ==========================================================================
	{
		Name:     "miro_create_group",
		Method:   "CreateGroup",
		Title:    "Group Items",
		Category: "create",
		Description: `Group multiple items together logically (minimum 2). Grouped items move and resize together. For a visible container with a border and title, use miro_create_frame.

VOICE-FRIENDLY: "Grouped 4 items together"`,
	},
	{
		Name:     "miro_ungroup",
		Method:   "Ungroup",
		Title:    "Ungroup Items",
		Category: "update",
		Description: `Remove a group, releasing items to move independently.

VOICE-FRIENDLY: "Items ungrouped successfully"`,
	},
	{
		Name:     "miro_list_groups",
		Method:   "ListGroups",
		Title:    "List Groups",
		Category: "read",
		ReadOnly: true,
		Description: `List all groups on a Miro board.

VOICE-FRIENDLY: "Found 3 groups on the board"`,
	},
	{
		Name:     "miro_get_group",
		Method:   "GetGroup",
		Title:    "Get Group Details",
		Category: "read",
		ReadOnly: true,
		Description: `Get details of a specific group by ID.

VOICE-FRIENDLY: "This group contains 4 items"`,
	},
	{
		Name:     "miro_get_group_items",
		Method:   "GetGroupItems",
		Title:    "Get Group Items",
		Category: "read",
		ReadOnly: true,
		Description: `Get items in a group with their details. For items inside a visual frame, use miro_get_frame_items.

VOICE-FRIENDLY: "Group has 4 items: 2 stickies, 1 shape, 1 text"`,
	},
	{
		Name:       "miro_update_group",
		Method:     "UpdateGroup",
		Title:      "Update Group",
		Category:   "update",
		Idempotent: true,
		Description: `Update a group's member items. Replaces all members; include existing IDs to keep them. Minimum 2 items.

VOICE-FRIENDLY: "Updated group with 5 items"`,
	},
	{
		Name:        "miro_delete_group",
		Method:      "DeleteGroup",
		Title:       "Delete Group",
		Category:    "delete",
		Destructive: true,
		Description: `Delete a group. Set delete_items=true to also delete items (default: items are ungrouped).

WARNING: Deleting items cannot be undone. Use dry_run=true to preview first.

VOICE-FRIENDLY: "Group deleted, items ungrouped"`,
	},

	// ==========================================================================
	// Board Member Tools
	// ==========================================================================
	{
		Name:     "miro_list_board_members",
		Method:   "ListBoardMembers",
		Title:    "List Board Members",
		Category: "read",
		ReadOnly: true,
		Description: `List all users who have access to a board.

VOICE-FRIENDLY: "This board has 5 members: 2 editors, 3 viewers"`,
	},
	{
		Name:     "miro_share_board",
		Method:   "ShareBoard",
		Title:    "Share Board",
		Category: "boards",
		Description: `Share a board with someone by email. Roles: viewer (default), commenter, editor.

VOICE-FRIENDLY: "Shared board with jane@example.com as editor"`,
	},
	{
		Name:     "miro_get_board_member",
		Method:   "GetBoardMember",
		Title:    "Get Board Member",
		Category: "read",
		ReadOnly: true,
		Description: `Get details of a specific board member.

VOICE-FRIENDLY: "John Smith has editor access"`,
	},
	{
		Name:        "miro_remove_board_member",
		Method:      "RemoveBoardMember",
		Title:       "Remove Board Member",
		Category:    "members",
		Destructive: true,
		Description: `Remove a member from a board.

WARNING: This revokes the member's access to the board.

VOICE-FRIENDLY: "Removed member from board"`,
	},
	{
		Name:       "miro_update_board_member",
		Method:     "UpdateBoardMember",
		Title:      "Update Board Member",
		Category:   "members",
		Idempotent: true,
		Description: `Update a board member's role (viewer, commenter, or editor).

VOICE-FRIENDLY: "Updated John's role to editor"`,
	},

	// ==========================================================================
	// Mindmap Tools
	// ==========================================================================
	{
		Name:        "miro_create_mindmap_node",
		Method:      "CreateMindmapNode",
		Title:       "Create Mindmap Node",
		Category:    "create",
		Description: `Create a mindmap node. Omit parent_id for root; add parent_id for children. node_view: "text" (default) or "bubble".`,
	},
	{
		Name:        "miro_get_mindmap_node",
		Method:      "GetMindmapNode",
		Title:       "Get Mindmap Node",
		Category:    "read",
		ReadOnly:    true,
		Description: `Get mindmap node details including content, hierarchy, and position. Uses v2-experimental API.`,
	},
	{
		Name:        "miro_list_mindmap_nodes",
		Method:      "ListMindmapNodes",
		Title:       "List Mindmap Nodes",
		Category:    "read",
		ReadOnly:    true,
		Description: `List all mindmap nodes on a board. Returns flat list; use parent_id to reconstruct hierarchy. Uses v2-experimental API.`,
	},
	{
		Name:        "miro_delete_mindmap_node",
		Method:      "DeleteMindmapNode",
		Title:       "Delete Mindmap Node",
		Category:    "delete",
		Destructive: true,
		Description: `Delete a mindmap node. Deleting a parent may affect children. Uses v2-experimental API.

WARNING: Cannot be undone. Use dry_run=true to preview first.`,
	},

	// ==========================================================================
	// Export Tools
	// ==========================================================================
	{
		Name:     "miro_get_board_picture",
		Method:   "GetBoardPicture",
		Title:    "Get Board Picture",
		Category: "export",
		ReadOnly: true,
		Description: `Get the preview image URL for a board. Works for all Miro plans. For full PDF/SVG exports, use the Enterprise export tools.

VOICE-FRIENDLY: "Got preview image for the board"`,
	},
	{
		Name:        "miro_create_export_job",
		Method:      "CreateExportJob",
		Title:       "Create Export Job",
		Category:    "export",
		Description: `Export boards to PDF, SVG, or HTML. ENTERPRISE ONLY. Returns job ID; use miro_get_export_job_status to monitor.`,
	},
	{
		Name:        "miro_get_export_job_status",
		Method:      "GetExportJobStatus",
		Title:       "Get Export Job Status",
		Category:    "export",
		ReadOnly:    true,
		Description: `Check export job progress. ENTERPRISE ONLY.`,
	},
	{
		Name:        "miro_get_export_job_results",
		Method:      "GetExportJobResults",
		Title:       "Get Export Job Results",
		Category:    "export",
		ReadOnly:    true,
		Description: `Get download links for completed export. ENTERPRISE ONLY. Links expire in 15 min; call again to regenerate.`,
	},

	// ==========================================================================
	// Audit Tools (Local Operations)
	// ==========================================================================
	{
		Name:        "miro_get_audit_log",
		Method:      "GetAuditLog",
		Title:       "Get Audit Log",
		Category:    "audit",
		ReadOnly:    true,
		Description: `Query local audit log for MCP tool executions (this session only). Filter by time range, tool, board, action type, or success/failure.`,
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

	// ==========================================================================
	// Webhook Tools - REMOVED (Miro sunset Dec 5, 2025)
	// ==========================================================================
	// Miro is discontinuing experimental webhooks on December 5, 2025.
	// The /v2-experimental/webhooks/board_subscriptions endpoints no longer work.
	// See: https://community.miro.com/developer-platform-and-apis-57/miro-webhooks-4281

	// ==========================================================================
	// Diagram Generation Tools (AI-Powered)
	// ==========================================================================
	{
		Name:     "miro_generate_diagram",
		Method:   "GenerateDiagram",
		Title:    "Generate Diagram from Code",
		Category: "diagrams",
		Description: `Generate diagram on Miro from Mermaid code. Creates shapes and connectors with auto-layout.

USE WHEN: "create flowchart", "generate diagram", "draw process flow", "sequence diagram"

TYPES: flowchart/graph, sequenceDiagram
FLOWCHART: A[rect] --> B{diamond} -->|label| C((circle))
SEQUENCE: participant A; A->>B: sync; A-->>B: async`,
	},

	// ==========================================================================
	// App Card Tools
	// ==========================================================================
	{
		Name:     "miro_create_app_card",
		Method:   "CreateAppCard",
		Title:    "Create App Card",
		Category: "create",
		Description: `Create an app card with custom fields and status indicators. For simple cards with due dates, use miro_create_card instead.

USE WHEN: "create an app card", "add a card with fields", "create a custom card"

VOICE-FRIENDLY: "Created app card 'Integration Status'"`,
	},
	{
		Name:     "miro_get_app_card",
		Method:   "GetAppCard",
		Title:    "Get App Card",
		Category: "read",
		ReadOnly: true,
		Description: `Get details of a specific app card by ID.

VOICE-FRIENDLY: "App card 'API Status' shows 3 custom fields"`,
	},
	{
		Name:     "miro_update_app_card",
		Method:   "UpdateAppCard",
		Title:    "Update App Card",
		Category: "update",
		Description: `Update an app card's title, description, status, or custom fields.

VOICE-FRIENDLY: "Updated app card status to 'connected'"`,
	},
	{
		Name:        "miro_delete_app_card",
		Method:      "DeleteAppCard",
		Title:       "Delete App Card",
		Category:    "delete",
		Destructive: true,
		Description: `Delete an app card from a Miro board.

WARNING: Cannot be undone. Use dry_run=true to preview first.

VOICE-FRIENDLY: "App card deleted successfully"`,
	},
}

// ptr is a helper to create a pointer to a value.
func ptr[T any](v T) *T {
	return &v
}
