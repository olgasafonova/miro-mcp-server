// Package tools provides MCP tool definitions for the Miro MCP server.
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
- start_cap: Arrow style at start (none, arrow, filled_arrow, diamond)
- end_cap: Arrow style at end (default: arrow)
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
- limit: Max items (default 50, max 100)

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
}

// ptr is a helper to create a pointer to a value.
func ptr[T any](v T) *T {
	return &v
}
