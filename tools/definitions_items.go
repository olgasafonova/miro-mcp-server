package tools

// Create Tools - Sticky Notes, Shapes, Text
var createTools = []ToolSpec{
	{
		Name:     "miro_create_sticky",
		Method:   "CreateSticky",
		Title:    "Create Sticky Note",
		Category: "create",
		Description: `Create a sticky note on a Miro board. For multiple stickies in a grid, use miro_create_sticky_grid. For batch creation of mixed items, use miro_bulk_create.

USE WHEN: "add a sticky", "create note saying X", "put a yellow sticky"

RETURNS: Item ID, content, color, and view link.

FAILS WHEN: Content is empty. board_id not found.

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
- board_id: Required. Get from list_boards or find_board
- shape: Shape type (required, default: rectangle)
- content: Text inside shape
- color: Fill color. 6-char hex like "#FF5733" or named: red, orange, yellow, green, blue, purple, pink, gray, white, black.
- text_color: Text color, same format as color.
- text_align: Horizontal text alignment: left, center (default), right.
- text_align_vertical: Vertical text alignment: top, middle (default), bottom. For non-rectangular shapes (triangle, hexagon), 'middle' aligns to the bounding-box center, which is not the same as the visual centroid.
- x, y: Position (default: 0, 0)
- width, height: Size (default: 200, 200)

RETURNS: Item ID, shape type, position, size, and view link.

RELATED: For flowchart-specific stencil shapes (experimental API), use miro_create_flowchart_shape instead.

EXAMPLE:
{"board_id": "uXjVN1234", "shape": "circle", "content": "Start", "color": "green", "text_align": "center", "x": 0, "y": 0}`,
	},
	{
		Name:     "miro_create_text",
		Method:   "CreateText",
		Title:    "Create Text",
		Category: "create",
		Description: `Add free-floating text to a Miro board with no background or border. For notes with colored backgrounds, use miro_create_sticky. For rich Markdown documents, use miro_create_doc.

USE WHEN: "add a title", "put heading text", "write a label", "add text saying X"

RETURNS: Item ID, content, and view link.`,
	},
	{
		Name:     "miro_create_connector",
		Method:   "CreateConnector",
		Title:    "Create Connector",
		Category: "create",
		Description: `Create a connector line between two items. Styles: straight, elbowed (default), curved. Caps: none, arrow, stealth, diamond, filled_diamond, oval, filled_oval, triangle, filled_triangle.

USE WHEN: "connect these items", "draw a line from A to B", "link items together", "add an arrow"

RETURNS: Connector ID and view link.

FAILS WHEN: start_item_id or end_item_id don't exist on the board. Both endpoints are required.`,
	},
	{
		Name:     "miro_create_frame",
		Method:   "CreateFrame",
		Title:    "Create Frame",
		Category: "create",
		Description: `Create a frame container to group items visually. For logical grouping without a visual border, use miro_create_group.

USE WHEN: "create a frame", "add a container", "make a section for X"

PARAMETERS:
- color: Background color. Accepts a 6-char hex like "#006400" or a named color: red, orange, yellow, green, blue, purple, pink, gray, white, black. Note: frames use a smaller palette than stickies; sticky-only names like 'light_yellow' or 'light_green' are NOT valid for frames (use a hex value or a base named color instead).

RETURNS: Frame ID, title, and view link.

EXAMPLE:
{"board_id": "uXjVN1234", "title": "Q1 Goals", "width": 800, "height": 600, "color": "green"}`,
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

PARAMETERS:
- color: Background color. Accepts a 6-char hex like "#006400" or a named color: red, orange, yellow, green, blue, purple, pink, gray, white, black. Note: frames use a smaller palette than stickies; sticky-only names like 'light_yellow' or 'light_green' are NOT valid for frames (use a hex value or a base named color instead).

RETURNS: Confirmation with frame ID.

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

RETURNS: Confirmation with deleted frame ID.

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
}

// Bulk Operations
var bulkTools = []ToolSpec{
	{
		Name:     "miro_bulk_create",
		Method:   "BulkCreate",
		Title:    "Bulk Create Items",
		Category: "create",
		Description: `Create multiple items at once (max 20). For creating only stickies in a grid, use miro_create_sticky_grid instead.

USE WHEN: "add these 5 stickies", "create items for each of these", "batch add"

ITEM FIELDS:
- type (required): sticky_note, shape, text
- content, x, y, width, height, color, parent_id: standard fields
- shape: shape type (when type=shape)
- text_color: text color (shape and text items only)
- font_size: font size in points (shape and text items only; default 14)
- text_align: horizontal text alignment (shape items only): left, center, right
- text_align_vertical: vertical text alignment (shape items only): top, middle, bottom

RETURNS: Count of created items and their IDs (returned in input order even when API IDs are assigned out of order).

FAILS WHEN: More than 20 items. Empty items list. Individual items may fail while others succeed; check errors in response.

VOICE-FRIENDLY: "Created 5 items on the board"`,
	},
	{
		Name:     "miro_bulk_update",
		Method:   "BulkUpdate",
		Title:    "Bulk Update Items",
		Category: "update",
		Description: `Update multiple items at once (max 20). Only provide fields you want to change.

USE WHEN: "update positions for these items", "rename these stickies", "move several items into a frame", "center the text on these shapes".

ITEM FIELDS:
- item_id (required): ID of the item to update
- type (optional but recommended): 'shape', 'sticky_note', or 'text'. When set, the update routes to the type-specific endpoint and accepts type-specific fields like text_align. When omitted, falls back to the generic /items endpoint (cross-type fields only).
- content, x, y, width, height, color, parent_id: cross-type fields, accepted regardless of type
- text_color: shape and text items (requires type=shape or type=text)
- text_align: shape items (requires type=shape): left, center, right
- text_align_vertical: shape items (requires type=shape): top, middle, bottom

RETURNS: Count of updated items and their IDs.

FAILS WHEN: More than 20 items. Empty items list. Individual items may fail while others succeed; check errors in response.

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

RETURNS: Count of deleted items and their IDs.

FAILS WHEN: More than 20 items. Empty items list. Individual items may fail while others succeed; check errors in response.

VOICE-FRIENDLY: "Deleted 5 items from the board"`,
	},
}

// Read/List Tools
var readTools = []ToolSpec{
	{
		Name:         "miro_list_items",
		Method:       "ListItems",
		Title:        "List Board Items",
		Category:     "read",
		ReadOnly:     true,
		HeaderParams: map[string]string{"board_id": "Board-Id"},
		Description: `List items on a Miro board (max 50). For ALL items with auto-pagination, use miro_list_all_items. For text search, use miro_search_board.

USE WHEN: "what's on the board", "show all stickies", "list shapes"

CAVEAT: Miro's REST index lags items created moments ago by Miro AI or the official Miro MCP composer, and interactive widgets (polls, kanban, timelines) never appear in it. Missing items on a fresh board are index lag, not deletion.`,
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
}

// Card, Image, Document, Embed Tools
var cardMediaTools = []ToolSpec{
	{
		Name:     "miro_create_card",
		Method:   "CreateCard",
		Title:    "Create Card",
		Category: "create",
		Description: `Create a card on a Miro board. Cards have title, description, and due dates. For cards with custom fields and status, use miro_create_app_card.

USE WHEN: "add a card", "create a task card", "card with due date"

RETURNS: Card ID, title, and view link.

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

NOTE: The image URL must be publicly accessible. Private URLs won't work.

FAILS WHEN: URL is not publicly accessible or returns 404. board_id not found.

RELATED: To upload a local file instead, use miro_upload_image.`,
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
		Name:     "miro_create_document",
		Method:   "CreateDocument",
		Title:    "Create Document",
		Category: "create",
		Description: `Add a document (PDF, etc.) to a Miro board from a URL. URL must be publicly accessible.

USE WHEN: "add a document from URL", "put a PDF on the board", "add a reference document"

RETURNS: Document ID, title, and view link.

RELATED: To upload a local file instead, use miro_upload_document.`,
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
		Name:     "miro_create_embed",
		Method:   "CreateEmbed",
		Title:    "Create Embed",
		Category: "create",
		Description: `Embed external content as a live preview on a Miro board. Supports YouTube, Vimeo, Figma, Google Docs, Loom, and other oEmbed providers. For static images from URL, use miro_create_image. For document references from URL, use miro_create_document.

USE WHEN: "embed this YouTube video", "add a Figma link", "embed Google Doc", "put a Loom video on the board"

RETURNS: Embed ID, URL, provider name, and view link.`,
	},
}

// Pagination Tools
var paginationTools = []ToolSpec{
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
}

// Composite Tools
var compositeTools = []ToolSpec{
	{
		Name:     "miro_find_board",
		Method:   "FindBoardByNameTool",
		Title:    "Find Board by Name",
		Category: "boards",
		ReadOnly: true,
		Description: `Find a Miro board by name (case-insensitive, partial match). Returns board ID for subsequent operations. For listing all boards, use miro_list_boards.

USE WHEN: "find board named X", "get the Design Sprint board"

RETURNS: id, name, description, view_link, team_id, team_name, owner (id + name), created_at, modified_at. Answers "who owns this board" and "when was it last changed" without a follow-up miro_get_board.

MATCH QUALITY: the match field says how the board was reached - "exact", "prefix", "contains", or "none". A "none" result is the nearest candidate rather than a match: nothing with that name was found, and the board returned is a guess. Check match before acting on the board ID.

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

CAVEAT: counts come from Miro's REST index, which lags items created moments ago by Miro AI or the official Miro MCP composer, and excludes interactive widgets (polls, kanban, timelines) entirely. A count lower than what the board shows visually is index lag, not data loss.

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
}

// Flowchart Shape Tools (Experimental)
var flowchartShapeTools = []ToolSpec{
	{
		Name:     "miro_create_flowchart_shape",
		Method:   "CreateFlowchartShape",
		Title:    "Create Flowchart Shape",
		Category: "create",
		Description: `Create a flowchart shape using the experimental API. Supports additional stencil shapes beyond the standard shape tool.

USE WHEN: User says "create a flowchart shape", "add a process box", "draw a decision diamond for flowchart"

For standard shapes, use miro_create_shape instead. This tool uses the v2-experimental API for flowchart-specific stencil shapes.

PARAMETERS:
- board_id: Required
- shape: Shape type (required). Supports: rectangle, round_rectangle, circle, rhombus, parallelogram, trapezoid, pentagon, hexagon, star, flow_chart_predefined_process, wedge_round_rectangle_callout, etc.
- content: Text inside the shape
- x, y: Position
- width, height: Size (default 200x200)
- fill_color: Fill color (hex like #006400)
- border_color: Border color (hex like #000000)
- parent_id: Frame ID

RETURNS: Item ID, shape type, content, and view link.

NOTE: Uses v2-experimental API. Shape types may change when this moves to GA.`,
	},
}

// App Card Tools
var appCardTools = []ToolSpec{
	{
		Name:     "miro_create_app_card",
		Method:   "CreateAppCard",
		Title:    "Create App Card",
		Category: "create",
		Description: `Create an app card with custom fields and status indicators. For simple cards with due dates, use miro_create_card instead.

USE WHEN: "create an app card", "add a card with fields", "create a custom card"

RETURNS: App card ID and view link.

VOICE-FRIENDLY: "Created app card 'Integration Status'"`,
	},
	{
		Name:     "miro_get_app_card",
		Method:   "GetAppCard",
		Title:    "Get App Card",
		Category: "read",
		ReadOnly: true,
		Description: `Get details of a specific app card by ID.

RETURNS: App card ID, title, description, status, custom fields, and view link.

VOICE-FRIENDLY: "App card 'API Status' shows 3 custom fields"`,
	},
	{
		Name:     "miro_update_app_card",
		Method:   "UpdateAppCard",
		Title:    "Update App Card",
		Category: "update",
		Description: `Update an app card's title, description, status, or custom fields.

RETURNS: Confirmation with app card ID.

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

RETURNS: Confirmation with deleted app card ID.

VOICE-FRIENDLY: "App card deleted successfully"`,
	},
}
