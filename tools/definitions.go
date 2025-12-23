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
// Tool descriptions are optimized for voice interaction:
// - Short, action-oriented names
// - Clear, speakable result descriptions
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
		Description: `List Miro boards accessible to the user.

USE WHEN: User asks "show my boards", "what boards do I have", "find board named X"

RETURNS: Board names, IDs, and view links. Use board ID for subsequent operations.

VOICE-FRIENDLY: "Found 5 boards: Design Sprint, Product Roadmap, Team Retro..."`,
	},
	{
		Name:     "miro_get_board",
		Method:   "GetBoard",
		Title:    "Get Board Details",
		Category: "boards",
		ReadOnly: true,
		Description: `Get details of a specific Miro board.

USE WHEN: User asks "tell me about board X", "what's on this board"

RETURNS: Board name, description, owner, and creation date.`,
	},
	{
		Name:     "miro_create_board",
		Method:   "CreateBoard",
		Title:    "Create Board",
		Category: "boards",
		Description: `Create a new Miro board.

USE WHEN: User says "create a new board", "make a board called X", "new whiteboard"

PARAMETERS:
- name: Board name (required)
- description: Board description
- team_id: Team ID to create board in

VOICE-FRIENDLY: "Created board 'Sprint Planning'"`,
	},
	{
		Name:     "miro_copy_board",
		Method:   "CopyBoard",
		Title:    "Copy Board",
		Category: "boards",
		Description: `Copy an existing Miro board.

USE WHEN: User says "copy this board", "duplicate board X", "make a copy of the board"

PARAMETERS:
- board_id: Board to copy (required)
- name: Name for the copy (defaults to "Copy of {original}")
- description: Description for the copy
- team_id: Team ID to copy board to

VOICE-FRIENDLY: "Copied board to 'Sprint Planning Copy'"`,
	},
	{
		Name:        "miro_delete_board",
		Method:      "DeleteBoard",
		Title:       "Delete Board",
		Category:    "boards",
		Destructive: true,
		Description: `Delete a Miro board permanently.

USE WHEN: User says "delete this board", "remove board X"

PARAMETERS:
- board_id: Board to delete (required)

WARNING: This action cannot be undone. The board and all its contents will be permanently deleted.`,
	},
	{
		Name:       "miro_update_board",
		Method:     "UpdateBoard",
		Title:      "Update Board",
		Category:   "boards",
		Idempotent: true,
		Description: `Update a Miro board's name or description.

USE WHEN: User says "rename the board", "change board name to X", "update board description"

PARAMETERS:
- board_id: Required
- name: New board name
- description: New board description

NOTE: At least one of name or description must be provided.

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
		Description: `Create a sticky note on a Miro board.

USE WHEN: User says "add a sticky", "create note saying X", "put a yellow sticky with X"

PARAMETERS:
- board_id: Required. Get from list_boards
- content: The text on the sticky (required)
- color: yellow, green, blue, pink, orange, red, gray, cyan, purple
- x, y: Position (default 0,0)
- parent_id: Frame ID to place it in

VOICE-FRIENDLY: "Created yellow sticky 'Action item: Review design'"`,
	},
	{
		Name:     "miro_create_shape",
		Method:   "CreateShape",
		Title:    "Create Shape",
		Category: "create",
		Description: `Create a shape on a Miro board.

USE WHEN: User says "add a rectangle", "draw a circle", "create a box for X"

SHAPE TYPES:
- Basic: rectangle, round_rectangle, circle, triangle, rhombus
- Flow: parallelogram, trapezoid, pentagon, hexagon, star
- Flowchart: flow_chart_predefined_process, wedge_round_rectangle_callout

PARAMETERS:
- board_id: Required
- shape: Shape type (required)
- content: Text inside shape
- color: Fill color
- width, height: Size in pixels (default 200x200)`,
	},
	{
		Name:     "miro_create_text",
		Method:   "CreateText",
		Title:    "Create Text",
		Category: "create",
		Description: `Create a text element on a Miro board.

USE WHEN: User says "add text", "write a title", "put label X"

PARAMETERS:
- board_id: Required
- content: The text content (required)
- font_size: Size in points
- color: Text color
- x, y: Position`,
	},
	{
		Name:     "miro_create_connector",
		Method:   "CreateConnector",
		Title:    "Create Connector",
		Category: "create",
		Description: `Create a connector line between two items.

USE WHEN: User says "connect these items", "draw arrow from X to Y", "link boxes"

PARAMETERS:
- board_id: Required
- start_item_id: ID of source item (required)
- end_item_id: ID of target item (required)
- style: straight, elbowed (default), curved
- start_cap: Arrow style at start (none, arrow, stealth, diamond, filled_diamond, oval, filled_oval, triangle, filled_triangle)
- end_cap: Arrow style at end (none, arrow, stealth, diamond, filled_diamond, oval, filled_oval, triangle, filled_triangle). Default: arrow
- caption: Label on the connector`,
	},
	{
		Name:     "miro_create_frame",
		Method:   "CreateFrame",
		Title:    "Create Frame",
		Category: "create",
		Description: `Create a frame container to group items.

USE WHEN: User says "create a section for X", "add a frame", "make a container"

PARAMETERS:
- board_id: Required
- title: Frame title
- width, height: Size (default 800x600)
- color: Background color
- x, y: Position`,
	},
	{
		Name:     "miro_get_frame",
		Method:   "GetFrame",
		Title:    "Get Frame Details",
		Category: "read",
		ReadOnly: true,
		Description: `Get full details of a specific frame by ID.

USE WHEN: User asks "show me this frame", "frame details", "what's in this frame"

PARAMETERS:
- board_id: Required
- frame_id: Required

RETURNS: Frame details including title, position, size, color, child count, and timestamps.

VOICE-FRIENDLY: "Frame 'Sprint Planning' is 800x600 with 12 items inside"`,
	},
	{
		Name:       "miro_update_frame",
		Method:     "UpdateFrame",
		Title:      "Update Frame",
		Category:   "update",
		Idempotent: true,
		Description: `Update an existing frame's title, position, size, or color.

USE WHEN: User says "rename the frame", "resize this frame", "move the frame", "change frame color"

PARAMETERS:
- board_id: Required
- frame_id: Required
- title: New frame title
- x, y: New position
- width, height: New size
- color: New background color

NOTE: At least one update field must be provided.

VOICE-FRIENDLY: "Updated frame title to 'Q1 Goals'"`,
	},
	{
		Name:        "miro_delete_frame",
		Method:      "DeleteFrame",
		Title:       "Delete Frame",
		Category:    "delete",
		Destructive: true,
		Description: `Delete a frame from a Miro board.

USE WHEN: User says "remove this frame", "delete the frame"

PARAMETERS:
- board_id: Required
- frame_id: Required

WARNING: This action cannot be undone. Items inside the frame are NOT deleted - they become ungrouped.

VOICE-FRIENDLY: "Frame deleted successfully"`,
	},
	{
		Name:     "miro_get_frame_items",
		Method:   "GetFrameItems",
		Title:    "Get Frame Items",
		Category: "read",
		ReadOnly: true,
		Description: `Get all items contained within a specific frame.

USE WHEN: User asks "what's inside this frame", "list frame contents", "show items in frame"

PARAMETERS:
- board_id: Required
- frame_id: Required
- type: Filter by item type (sticky_note, shape, text, card, image)
- limit: Max items to return (default 50, max 100)

RETURNS: List of items with IDs, types, and content.

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
		Description: `Create multiple items at once (max 20).

USE WHEN: User says "add these 5 stickies", "create items for each of these", "batch add"

PARAMETERS:
- board_id: Required
- items: Array of items to create, each with:
  - type: sticky_note, shape, or text
  - content: Text content
  - shape: Shape type (for shapes)
  - x, y: Position
  - color: Item color

VOICE-FRIENDLY: "Created 5 items on the board"`,
	},
	{
		Name:     "miro_bulk_update",
		Method:   "BulkUpdate",
		Title:    "Bulk Update Items",
		Category: "update",
		Description: `Update multiple items at once (max 20).

USE WHEN: User says "update these items", "move all these stickies", "change color of these shapes"

PARAMETERS:
- board_id: Required
- items: Array of item updates, each with:
  - item_id: ID of item to update (required)
  - content: New text content
  - x, y: New position
  - width, height: New size
  - color: New color
  - parent_id: New frame ID (empty string to remove from frame)

NOTE: Only provide fields you want to change. Null/missing fields are ignored.

VOICE-FRIENDLY: "Updated 5 items on the board"`,
	},
	{
		Name:     "miro_bulk_delete",
		Method:   "BulkDelete",
		Title:    "Bulk Delete Items",
		Category: "delete",
		Destructive: true,
		Description: `Delete multiple items at once (max 20).

USE WHEN: User says "delete these items", "remove all these stickies", "clear these shapes"

PARAMETERS:
- board_id: Required
- item_ids: Array of item IDs to delete (max 20)

WARNING: This action cannot be undone.

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
		Description: `List items on a Miro board.

USE WHEN: User asks "what's on the board", "show me all stickies", "list shapes"

PARAMETERS:
- board_id: Required
- type: Filter by type (sticky_note, shape, text, connector, frame)
- limit: Max items (default 50, max 50)

RETURNS: Item IDs, types, content, and positions.`,
	},
	{
		Name:     "miro_get_item",
		Method:   "GetItem",
		Title:    "Get Item Details",
		Category: "read",
		ReadOnly: true,
		Description: `Get full details of a specific item by ID.

USE WHEN: User asks "read that sticky", "what does item X say", "show me details of that shape"

PARAMETERS:
- board_id: Required
- item_id: Required (ID of the item to retrieve)

RETURNS: Full item details including content, position, size, color, creator, and timestamps.

VOICE-FRIENDLY: "That sticky says 'Review Q4 goals' and was created by John yesterday"`,
	},
	{
		Name:     "miro_search_board",
		Method:   "SearchBoard",
		Title:    "Search Board Content",
		Category: "read",
		ReadOnly: true,
		Description: `Search for items containing specific text on a board.

USE WHEN: User asks "find items about X", "search for budget", "which stickies mention deadline"

PARAMETERS:
- board_id: Required
- query: Text to search for (required)
- type: Filter by type (sticky_note, shape, text, frame)
- limit: Max results (default 20, max 50)

RETURNS: Matching items with content snippets highlighting the match.

VOICE-FRIENDLY: "Found 3 stickies mentioning 'budget': 'Q4 budget review', 'Budget approval needed', 'Marketing budget'"`,
	},

	// ==========================================================================
	// Card, Image, Document, Embed Tools
	// ==========================================================================
	{
		Name:     "miro_create_card",
		Method:   "CreateCard",
		Title:    "Create Card",
		Category: "create",
		Description: `Create a card on a Miro board. Cards are like enhanced sticky notes with title, description, and due dates.

USE WHEN: User says "add a card", "create a task card", "add card with due date"

PARAMETERS:
- board_id: Required
- title: Card title (required)
- description: Card body text
- due_date: Due date in ISO format (e.g., 2024-12-31)
- x, y: Position
- width: Card width (default 320)

VOICE-FRIENDLY: "Created card 'Review design specs'"`,
	},
	{
		Name:     "miro_create_image",
		Method:   "CreateImage",
		Title:    "Create Image",
		Category: "create",
		Description: `Add an image to a Miro board from a URL.

USE WHEN: User says "add an image", "insert picture from URL", "put this image on the board"

PARAMETERS:
- board_id: Required
- url: Image URL (must be publicly accessible, required)
- title: Alt text / title
- width: Image width (preserves aspect ratio)
- x, y: Position

NOTE: The image URL must be publicly accessible. Private URLs won't work.`,
	},
	{
		Name:     "miro_create_document",
		Method:   "CreateDocument",
		Title:    "Create Document",
		Category: "create",
		Description: `Add a document (PDF, etc.) to a Miro board from a URL.

USE WHEN: User says "add a PDF", "embed this document", "put document on board"

PARAMETERS:
- board_id: Required
- url: Document URL (required)
- title: Document title
- width: Preview width
- x, y: Position

NOTE: Supports PDF and other document formats. URL must be publicly accessible.`,
	},
	{
		Name:     "miro_create_embed",
		Method:   "CreateEmbed",
		Title:    "Create Embed",
		Category: "create",
		Description: `Embed external content (YouTube, Figma, Google Docs, etc.) on a Miro board.

USE WHEN: User says "embed this video", "add YouTube link", "embed Figma design", "add Google Doc"

PARAMETERS:
- board_id: Required
- url: URL to embed (required)
- mode: "inline" (default) or "modal"
- width, height: Embed dimensions
- x, y: Position

SUPPORTED: YouTube, Vimeo, Figma, Google Docs, Loom, and many more.`,
	},

	// ==========================================================================
	// Tag Tools
	// ==========================================================================
	{
		Name:     "miro_create_tag",
		Method:   "CreateTag",
		Title:    "Create Tag",
		Category: "tags",
		Description: `Create a tag on a Miro board. Tags can be attached to sticky notes.

USE WHEN: User says "create a tag", "add label called X", "make an Urgent tag"

PARAMETERS:
- board_id: Required
- title: Tag text (required, e.g., "Urgent", "Done", "Review")
- color: red, magenta, violet, blue, cyan, green, yellow, gray, light_green, dark_green, dark_blue, dark_gray, black

VOICE-FRIENDLY: "Created red tag 'Urgent'"`,
	},
	{
		Name:     "miro_list_tags",
		Method:   "ListTags",
		Title:    "List Tags",
		Category: "tags",
		ReadOnly: true,
		Description: `List all tags on a Miro board.

USE WHEN: User asks "what tags exist", "show me all labels", "list available tags"

PARAMETERS:
- board_id: Required

RETURNS: List of tags with IDs, titles, and colors.`,
	},
	{
		Name:     "miro_attach_tag",
		Method:   "AttachTag",
		Title:    "Attach Tag",
		Category: "tags",
		Description: `Attach a tag to a sticky note or card.

USE WHEN: User says "tag this sticky as Urgent", "add Done label to item", "mark as reviewed"

PARAMETERS:
- board_id: Required
- item_id: Sticky note or card ID (required)
- tag_id: Tag ID (required, get from list_tags or create_tag)

IMPORTANT: Tags can ONLY be attached to sticky_note or card items. Shapes, text, frames, images, and other item types CANNOT be tagged. If user asks to tag a shape/star/circle, explain this limitation.`,
	},
	{
		Name:     "miro_detach_tag",
		Method:   "DetachTag",
		Title:    "Remove Tag",
		Category: "tags",
		Description: `Remove a tag from a sticky note.

USE WHEN: User says "remove tag from sticky", "untag this item", "remove Urgent label"

PARAMETERS:
- board_id: Required
- item_id: Sticky note ID (required)
- tag_id: Tag ID to remove (required)`,
	},
	{
		Name:     "miro_get_item_tags",
		Method:   "GetItemTags",
		Title:    "Get Item Tags",
		Category: "tags",
		ReadOnly: true,
		Description: `List tags attached to a specific item.

USE WHEN: User asks "what tags are on this sticky", "show labels for this item"

PARAMETERS:
- board_id: Required
- item_id: Item ID (required)

RETURNS: List of tags attached to the item.`,
	},
	{
		Name:     "miro_get_tag",
		Method:   "GetTag",
		Title:    "Get Tag",
		Category: "tags",
		ReadOnly: true,
		Description: `Get details of a specific tag by ID.

USE WHEN: User asks "show tag details", "what's this tag", "get tag info"

PARAMETERS:
- board_id: Required
- tag_id: Tag ID (required)

RETURNS: Tag ID, title, and color.

VOICE-FRIENDLY: "Tag 'Urgent' is red"`,
	},
	{
		Name:       "miro_update_tag",
		Method:     "UpdateTag",
		Title:      "Update Tag",
		Category:   "tags",
		Idempotent: true,
		Description: `Update an existing tag's title or color.

USE WHEN: User says "rename the tag", "change tag color", "update the Urgent tag to red"

PARAMETERS:
- board_id: Required
- tag_id: Tag ID to update (required)
- title: New tag text
- color: New color (red, magenta, violet, blue, cyan, green, yellow, gray, light_green, dark_green, dark_blue, dark_gray, black)

NOTE: At least one of title or color must be provided.

VOICE-FRIENDLY: "Updated tag to 'Done' with green color"`,
	},
	{
		Name:        "miro_delete_tag",
		Method:      "DeleteTag",
		Title:       "Delete Tag",
		Category:    "tags",
		Destructive: true,
		Description: `Delete a tag from a board. This removes the tag from all items it was attached to.

USE WHEN: User says "delete this tag", "remove the Urgent tag", "get rid of that label"

PARAMETERS:
- board_id: Required
- tag_id: Tag ID to delete (required)

WARNING: This removes the tag from all items and cannot be undone.

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
		Description: `Retrieve ALL items from a board with automatic pagination. Use for large boards.

USE WHEN: User asks "get everything on board", "list all items", "export board contents"

PARAMETERS:
- board_id: Required
- type: Filter by type (sticky_note, shape, text, etc.)
- max_items: Max items to fetch (default 500, max 10000)

NOTE: This handles pagination automatically. Use regular list_items for quick lookups.

VOICE-FRIENDLY: "Retrieved 847 items in 9 pages"`,
	},

	// ==========================================================================
	// Update/Delete Tools
	// ==========================================================================
	{
		Name:     "miro_update_item",
		Method:   "UpdateItem",
		Title:    "Update Item",
		Category: "update",
		Idempotent: true,
		Description: `Update an existing item's content, position, or style.

USE WHEN: User says "change sticky text to X", "move this item", "update the color"

PARAMETERS:
- board_id: Required
- item_id: Required
- content: New text content
- x, y: New position
- width, height: New size
- color: New color
- parent_id: Move to a frame (empty string to remove from frame)`,
	},
	{
		Name:        "miro_delete_item",
		Method:      "DeleteItem",
		Title:       "Delete Item",
		Category:    "delete",
		Destructive: true,
		Description: `Delete an item from a Miro board.

USE WHEN: User says "remove this sticky", "delete that shape", "get rid of X"

PARAMETERS:
- board_id: Required
- item_id: Required

WARNING: This action cannot be undone.`,
	},
	{
		Name:       "miro_update_connector",
		Method:     "UpdateConnector",
		Title:      "Update Connector",
		Category:   "update",
		Idempotent: true,
		Description: `Update an existing connector's style, arrows, or label.

USE WHEN: User says "change the arrow style", "update connector color", "add label to the line"

PARAMETERS:
- board_id: Required
- connector_id: Connector ID to update (required)
- style: Line style (straight, elbowed, curved)
- start_cap: Arrow at start (none, arrow, stealth, diamond, filled_diamond, oval, filled_oval, triangle, filled_triangle)
- end_cap: Arrow at end (none, arrow, stealth, diamond, filled_diamond, oval, filled_oval, triangle, filled_triangle)
- caption: Text label on the connector
- color: Line color (hex code)

NOTE: At least one update field must be provided.

VOICE-FRIENDLY: "Updated connector to curved style with arrow"`,
	},
	{
		Name:        "miro_delete_connector",
		Method:      "DeleteConnector",
		Title:       "Delete Connector",
		Category:    "delete",
		Destructive: true,
		Description: `Delete a connector line from a Miro board.

USE WHEN: User says "remove this line", "delete the connection", "disconnect these items"

PARAMETERS:
- board_id: Required
- connector_id: Connector ID to delete (required)

WARNING: This action cannot be undone.

VOICE-FRIENDLY: "Connector deleted successfully"`,
	},
	{
		Name:     "miro_list_connectors",
		Method:   "ListConnectors",
		Title:    "List Connectors",
		Category: "read",
		ReadOnly: true,
		Description: `List all connectors (lines/arrows) on a Miro board.

USE WHEN: User asks "show me all connections", "list connectors", "what's connected on this board"

PARAMETERS:
- board_id: Required
- limit: Max connectors to return (default 50, min 10, max 100)

RETURNS: List of connectors with IDs, connected item IDs, style, and any labels.

VOICE-FRIENDLY: "Found 12 connectors on the board"`,
	},
	{
		Name:     "miro_get_connector",
		Method:   "GetConnector",
		Title:    "Get Connector Details",
		Category: "read",
		ReadOnly: true,
		Description: `Get full details of a specific connector by ID.

USE WHEN: User asks "show me this connection", "details of that arrow", "what does this line connect"

PARAMETERS:
- board_id: Required
- connector_id: Required

RETURNS: Connector details including connected items, style, arrow types, color, and timestamps.

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
		Description: `Find a Miro board by name. No need to know the board ID!

USE WHEN: User says "find board named X", "get the Design Sprint board", "open my Planning board"

PARAMETERS:
- name: Board name to search for (required, case-insensitive, supports partial matching)

RETURNS: Board ID, name, and view link. Use the ID for subsequent operations.

VOICE-FRIENDLY: "Found 'Design Sprint' board - ready to work on it"`,
	},
	{
		Name:     "miro_get_board_summary",
		Method:   "GetBoardSummary",
		Title:    "Get Board Summary",
		Category: "read",
		ReadOnly: true,
		Description: `Get a comprehensive summary of a Miro board with item counts and statistics.

USE WHEN: User asks "summarize this board", "what's the overview", "board stats"

PARAMETERS:
- board_id: Required

RETURNS: Board name, description, item counts by type, total items, and 5 recent items.

VOICE-FRIENDLY: "Design Sprint has 15 stickies, 8 shapes, and 3 frames - 26 items total"`,
	},
	{
		Name:     "miro_create_sticky_grid",
		Method:   "CreateStickyGrid",
		Title:    "Create Sticky Grid",
		Category: "create",
		Description: `Create multiple sticky notes arranged in a grid layout.

USE WHEN: User says "add a grid of stickies", "create 6 stickies in rows", "make sticky notes for each of these ideas"

PARAMETERS:
- board_id: Required
- contents: Array of text for each sticky (required, max 50)
- columns: Number of columns (default 3)
- color: Color for all stickies (yellow, green, blue, pink, etc.)
- start_x, start_y: Grid starting position
- spacing: Gap between stickies (default 220)
- parent_id: Frame to place in

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
		Description: `Group multiple items together on a board. Grouped items move and resize together.

USE WHEN: User says "group these items", "combine these shapes", "make a group from these stickies"

PARAMETERS:
- board_id: Required
- item_ids: Array of item IDs to group (required, minimum 2 items)

NOTE: At least 2 items are required to create a group. Use list_items to find item IDs.

VOICE-FRIENDLY: "Grouped 4 items together"`,
	},
	{
		Name:        "miro_ungroup",
		Method:      "Ungroup",
		Title:       "Ungroup Items",
		Category:    "update",
		Description: `Remove a group, releasing its items to be moved independently.

USE WHEN: User says "ungroup these", "separate the group", "break apart the group"

PARAMETERS:
- board_id: Required
- group_id: ID of the group to ungroup (required)

VOICE-FRIENDLY: "Items ungrouped successfully"`,
	},
	{
		Name:     "miro_list_groups",
		Method:   "ListGroups",
		Title:    "List Groups",
		Category: "read",
		ReadOnly: true,
		Description: `List all groups on a Miro board.

USE WHEN: User asks "what groups exist", "show me all groups", "list groups on the board"

PARAMETERS:
- board_id: Required
- limit: Max groups to return (default 50, max 100)

RETURNS: List of groups with IDs and member item IDs.

VOICE-FRIENDLY: "Found 3 groups on the board"`,
	},
	{
		Name:     "miro_get_group",
		Method:   "GetGroup",
		Title:    "Get Group Details",
		Category: "read",
		ReadOnly: true,
		Description: `Get details of a specific group by ID.

USE WHEN: User asks "show me this group", "what's in this group", "group details"

PARAMETERS:
- board_id: Required
- group_id: Required

RETURNS: Group ID and list of item IDs in the group.

VOICE-FRIENDLY: "This group contains 4 items"`,
	},
	{
		Name:     "miro_get_group_items",
		Method:   "GetGroupItems",
		Title:    "Get Group Items",
		Category: "read",
		ReadOnly: true,
		Description: `Get the items in a group with their details.

USE WHEN: User asks "what items are in this group", "show group contents", "list items in group"

PARAMETERS:
- board_id: Required
- group_id: Required
- limit: Max items to return (default 50, max 100)

RETURNS: List of items with IDs, types, and content.

VOICE-FRIENDLY: "Group has 4 items: 2 stickies, 1 shape, 1 text"`,
	},
	{
		Name:        "miro_delete_group",
		Method:      "DeleteGroup",
		Title:       "Delete Group",
		Category:    "delete",
		Destructive: true,
		Description: `Delete a group from a board. Optionally delete the items in the group too.

USE WHEN: User says "delete this group", "remove the group"

PARAMETERS:
- board_id: Required
- group_id: Required
- delete_items: If true, also delete the items (default: false, items are ungrouped)

WARNING: Deleting items cannot be undone.

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

USE WHEN: User asks "who has access to this board", "show board members", "list collaborators"

PARAMETERS:
- board_id: Required
- limit: Max members to return (default 50)

RETURNS: Member names, emails, and roles (viewer, commenter, editor, coowner, owner).

VOICE-FRIENDLY: "This board has 5 members: 2 editors, 3 viewers"`,
	},
	{
		Name:     "miro_share_board",
		Method:   "ShareBoard",
		Title:    "Share Board",
		Category: "boards",
		Description: `Share a board with someone by email. Sends an invitation to collaborate.

USE WHEN: User says "share board with John", "add jane@example.com to the board", "invite someone to the board"

PARAMETERS:
- board_id: Required
- email: Email address of the person to invite (required)
- role: Access level - viewer, commenter, or editor (default: viewer)
- message: Optional invitation message

VOICE-FRIENDLY: "Shared board with jane@example.com as editor"`,
	},
	{
		Name:     "miro_get_board_member",
		Method:   "GetBoardMember",
		Title:    "Get Board Member",
		Category: "read",
		ReadOnly: true,
		Description: `Get details of a specific board member.

USE WHEN: User asks "who is this member", "show member details", "what role does X have"

PARAMETERS:
- board_id: Required
- member_id: Required

RETURNS: Member name, email, and role.

VOICE-FRIENDLY: "John Smith has editor access"`,
	},
	{
		Name:        "miro_remove_board_member",
		Method:      "RemoveBoardMember",
		Title:       "Remove Board Member",
		Category:    "members",
		Destructive: true,
		Description: `Remove a member from a board. They will lose access.

USE WHEN: User says "remove John from the board", "revoke access for X", "kick from board"

PARAMETERS:
- board_id: Required
- member_id: Required

WARNING: This revokes the member's access to the board.

VOICE-FRIENDLY: "Removed member from board"`,
	},
	{
		Name:       "miro_update_board_member",
		Method:     "UpdateBoardMember",
		Title:      "Update Board Member",
		Category:   "members",
		Idempotent: true,
		Description: `Update a board member's role.

USE WHEN: User says "change John's role to editor", "make X a viewer", "promote to editor"

PARAMETERS:
- board_id: Required
- member_id: Required
- role: New role (viewer, commenter, or editor)

VOICE-FRIENDLY: "Updated John's role to editor"`,
	},

	// ==========================================================================
	// Mindmap Tools
	// ==========================================================================
	{
		Name:     "miro_create_mindmap_node",
		Method:   "CreateMindmapNode",
		Title:    "Create Mindmap Node",
		Category: "create",
		Description: `Create a mindmap node on a board. Mindmaps are hierarchical diagrams for brainstorming and idea organization.

USE WHEN: User says "create a mindmap", "add mindmap node", "add a branch to the mindmap"

PARAMETERS:
- board_id: Required
- content: Node text content (required)
- parent_id: ID of parent node (omit for root node)
- node_view: Style - "text" (default) or "bubble"
- x, y: Position (only for root nodes without parent)

WORKFLOW:
1. Create root node first (without parent_id)
2. Add child nodes with parent_id pointing to root
3. Continue adding nested children as needed

VOICE-FRIENDLY: "Created mindmap node 'Project Ideas' as root"`,
	},
	{
		Name:     "miro_get_mindmap_node",
		Method:   "GetMindmapNode",
		Title:    "Get Mindmap Node",
		Category: "read",
		ReadOnly: true,
		Description: `Get details of a specific mindmap node including content, hierarchy, and position.

USE WHEN: User asks "show mindmap node details", "what's in this node", "get node info"

PARAMETERS:
- board_id: Required
- node_id: Mindmap node ID (required)

RETURNS: Node ID, content, node view style, whether it's root, parent ID, child IDs, position, timestamps.

NOTE: Uses v2-experimental Miro API endpoint.

VOICE-FRIENDLY: "Mindmap node 'Marketing Strategy' has 3 children"`,
	},
	{
		Name:     "miro_list_mindmap_nodes",
		Method:   "ListMindmapNodes",
		Title:    "List Mindmap Nodes",
		Category: "read",
		ReadOnly: true,
		Description: `List all mindmap nodes on a board.

USE WHEN: User asks "show all mindmap nodes", "list mindmap", "what's in the mindmap"

PARAMETERS:
- board_id: Required
- limit: Max nodes to return (default 50, max 100)
- cursor: Pagination cursor

RETURNS: List of nodes with IDs, content, root status, and parent relationships.

NOTE: Uses v2-experimental Miro API endpoint. Returns nodes flat - use parent_id to reconstruct hierarchy.

VOICE-FRIENDLY: "Found 12 mindmap nodes - 1 root, 11 children"`,
	},
	{
		Name:        "miro_delete_mindmap_node",
		Method:      "DeleteMindmapNode",
		Title:       "Delete Mindmap Node",
		Category:    "delete",
		Destructive: true,
		Description: `Delete a mindmap node from a board.

USE WHEN: User says "remove this mindmap node", "delete node", "remove from mindmap"

PARAMETERS:
- board_id: Required
- node_id: Mindmap node ID to delete (required)

WARNING: This action cannot be undone. Deleting a parent node may affect child nodes.

NOTE: Uses v2-experimental Miro API endpoint.

VOICE-FRIENDLY: "Mindmap node deleted successfully"`,
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
		Description: `Get the preview image URL for a board. Works for all Miro plans.

USE WHEN: User says "get board thumbnail", "show board preview", "get picture of the board"

PARAMETERS:
- board_id: Required

RETURNS: URL to the board's preview image. This is a thumbnail/snapshot of the board.

NOTE: This works for all Miro plans. For full PDF/SVG exports, use the Enterprise export tools.

VOICE-FRIENDLY: "Got preview image for the board"`,
	},
	{
		Name:     "miro_create_export_job",
		Method:   "CreateExportJob",
		Title:    "Create Export Job",
		Category: "export",
		Description: `Create an export job to export one or more boards to PDF, SVG, or HTML.

USE WHEN: User says "export board as PDF", "download board", "export to PDF", "backup board"

PARAMETERS:
- org_id: Organization ID (required, Enterprise feature)
- board_ids: Array of board IDs to export (required, max 50)
- format: Export format - pdf (default), svg, or html
- request_id: Unique ID for idempotency (auto-generated if not provided)

ENTERPRISE ONLY: This feature requires an Enterprise Miro plan with eDiscovery enabled.

RETURNS: Job ID and status. Use get_export_job_status to monitor progress.

VOICE-FRIENDLY: "Started export job for 3 boards"`,
	},
	{
		Name:     "miro_get_export_job_status",
		Method:   "GetExportJobStatus",
		Title:    "Get Export Job Status",
		Category: "export",
		ReadOnly: true,
		Description: `Check the status of a board export job.

USE WHEN: User asks "is the export done", "check export status", "how's the export going"

PARAMETERS:
- org_id: Organization ID (required)
- job_id: Export job ID from create_export_job (required)

RETURNS: Job status (in_progress, completed, failed), progress percentage, and boards exported count.

ENTERPRISE ONLY: This feature requires an Enterprise Miro plan.

VOICE-FRIENDLY: "Export is 75% complete - 3 of 4 boards done"`,
	},
	{
		Name:     "miro_get_export_job_results",
		Method:   "GetExportJobResults",
		Title:    "Get Export Job Results",
		Category: "export",
		ReadOnly: true,
		Description: `Get download links for a completed export job.

USE WHEN: User says "get export download link", "download the exported boards", "where's my export"

PARAMETERS:
- org_id: Organization ID (required)
- job_id: Export job ID (required)

RETURNS: Download URLs for each exported board. Links expire in 15 minutes.

ENTERPRISE ONLY: This feature requires an Enterprise Miro plan.

NOTE: If links expired, call this again to regenerate them.

VOICE-FRIENDLY: "Export ready - 4 boards available for download"`,
	},

	// ==========================================================================
	// Audit Tools (Local Operations)
	// ==========================================================================
	{
		Name:     "miro_get_audit_log",
		Method:   "GetAuditLog",
		Title:    "Get Audit Log",
		Category: "audit",
		ReadOnly: true,
		Description: `Query the local audit log for MCP tool executions.

USE WHEN: User asks "what operations have been performed", "show recent activity", "audit trail"

PARAMETERS:
- since: Return events after this time (ISO 8601 format)
- until: Return events before this time (ISO 8601 format)
- tool: Filter by tool name (e.g., miro_create_sticky)
- board_id: Filter by board ID
- action: Filter by action type (create, read, update, delete, export, auth)
- success: Filter by success status (true/false)
- limit: Maximum events to return (default 50, max 500)

RETURNS: List of audit events with timestamps, tools, actions, and results.

NOTE: This queries the LOCAL audit log of this MCP server session, not Miro's enterprise audit logs.

VOICE-FRIENDLY: "Found 15 operations in the last hour - 12 creates, 3 reads"`,
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
		Description: `Generate a visual diagram on a Miro board from Mermaid code. Supports flowcharts and sequence diagrams. Automatically creates shapes and connectors with proper layout.

USE WHEN: User says "create a flowchart", "generate diagram from this code", "draw process flow", "visualize this workflow", "create sequence diagram"

SUPPORTED DIAGRAM TYPES:
1. Flowcharts (flowchart/graph)
2. Sequence diagrams (sequenceDiagram)

FLOWCHART SYNTAX:
` + "```" + `
flowchart TB
    A[Start] --> B{Is it working?}
    B -->|Yes| C[Great!]
    B -->|No| D[Debug]
    D --> B
` + "```" + `

SEQUENCE DIAGRAM SYNTAX:
` + "```" + `
sequenceDiagram
    participant A as Alice
    participant B as Bob
    A->>B: Hello Bob!
    B-->>A: Hi Alice!
    A->>B: How are you?
` + "```" + `

FLOWCHART SHAPES:
- [text] = Rectangle
- (text) = Rounded rectangle
- {text} = Diamond (decision)
- ((text)) = Circle
- {{text}} = Hexagon

SEQUENCE DIAGRAM ELEMENTS:
- participant X = Declare participant
- actor X = Declare actor (circle shape)
- A->>B: text = Sync message (solid arrow)
- A-->>B: text = Async message (dotted arrow)
- A-xB: text = Lost message (cross end)

PARAMETERS:
- board_id: Required. Board to create diagram on
- diagram: Required. Mermaid diagram code
- start_x, start_y: Starting position (default: 0, 0)
- node_width: Width of each node (default: 180)
- parent_id: Frame ID to place diagram inside

RETURNS: Count of created nodes, connectors, and their IDs.

VOICE-FRIENDLY: "Created sequence diagram with 3 participants and 5 messages"`,
	},

	// ==========================================================================
	// App Card Tools
	// ==========================================================================
	{
		Name:     "miro_create_app_card",
		Method:   "CreateAppCard",
		Title:    "Create App Card",
		Category: "create",
		Description: `Create an app card on a Miro board. App cards are special cards with custom fields and status indicators.

USE WHEN: User says "create an app card", "add a card with fields", "create a custom card"

PARAMETERS:
- board_id: Required
- title: Card title (required)
- description: Card body text
- status: Status indicator (connected, disconnected, disabled)
- fields: Array of custom fields (max 5), each with value, fillColor, textColor
- x, y: Position
- width: Card width (default 320)
- parent_id: Frame ID to place card in

VOICE-FRIENDLY: "Created app card 'Integration Status'"`,
	},
	{
		Name:     "miro_get_app_card",
		Method:   "GetAppCard",
		Title:    "Get App Card",
		Category: "read",
		ReadOnly: true,
		Description: `Get details of a specific app card by ID.

USE WHEN: User asks "show app card details", "what's in this app card"

PARAMETERS:
- board_id: Required
- item_id: App card ID (required)

RETURNS: App card details including title, description, status, custom fields, position.

VOICE-FRIENDLY: "App card 'API Status' shows 3 custom fields"`,
	},
	{
		Name:     "miro_update_app_card",
		Method:   "UpdateAppCard",
		Title:    "Update App Card",
		Category: "update",
		Description: `Update an existing app card's content, status, or fields.

USE WHEN: User says "update the app card", "change card status", "modify card fields"

PARAMETERS:
- board_id: Required
- item_id: App card ID (required)
- title: New title
- description: New description
- status: New status (connected, disconnected, disabled)
- fields: Updated custom fields array
- x, y: New position
- width: New width

NOTE: Only provide fields you want to change.

VOICE-FRIENDLY: "Updated app card status to 'connected'"`,
	},
	{
		Name:        "miro_delete_app_card",
		Method:      "DeleteAppCard",
		Title:       "Delete App Card",
		Category:    "delete",
		Destructive: true,
		Description: `Delete an app card from a Miro board.

USE WHEN: User says "remove the app card", "delete that app card"

PARAMETERS:
- board_id: Required
- item_id: App card ID to delete (required)

WARNING: This action cannot be undone.

VOICE-FRIENDLY: "App card deleted successfully"`,
	},
}

// ptr is a helper to create a pointer to a value.
func ptr[T any](v T) *T {
	return &v
}
