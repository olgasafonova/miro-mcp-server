package tools

// Mindmap Tools
var mindmapTools = []ToolSpec{
	{
		Name:     "miro_create_mindmap_node",
		Method:   "CreateMindmapNode",
		Title:    "Create Mindmap Node",
		Category: "create",
		Description: `Create a mindmap node. Omit parent_id for root; add parent_id for children.

USE WHEN: "add a mindmap node", "create root node", "add child to mindmap"

LAYOUT: Multiple children of the same parent stack at the same default position if x/y are omitted. To avoid overlap when creating siblings, supply explicit x/y for each child (canvas-absolute coordinates). For a fully manual tree, prefer miro_create_shape + miro_create_connector with explicit positions.

PARAMETERS:
- node_view: Currently only "text" is reliably supported. Other values may return 400.
- x, y: Required for root nodes. Optional for children, but recommended when creating multiple siblings to override the API's default placement.

RETURNS: Node ID, parent node ID, and view link.

FAILS WHEN: parent_id references a non-existent node on the board. board_id not found.`,
	},
	{
		Name:     "miro_get_mindmap_node",
		Method:   "GetMindmapNode",
		Title:    "Get Mindmap Node",
		Category: "read",
		ReadOnly: true,
		Description: `Get mindmap node details including content, hierarchy, and position. Uses v2-experimental API.

USE WHEN: "mindmap node details", "what's in this node", "show node content"

RETURNS: Node ID, content, parent/child IDs, position, and root flag.`,
	},
	{
		Name:     "miro_list_mindmap_nodes",
		Method:   "ListMindmapNodes",
		Title:    "List Mindmap Nodes",
		Category: "read",
		ReadOnly: true,
		Description: `List all mindmap nodes on a board. Returns flat list; use parent_id to reconstruct hierarchy. Uses v2-experimental API.

RETURNS: Array of nodes with IDs, content, and parent IDs. Use parent_id to reconstruct hierarchy.`,
	},
	{
		Name:        "miro_delete_mindmap_node",
		Method:      "DeleteMindmapNode",
		Title:       "Delete Mindmap Node",
		Category:    "delete",
		Destructive: true,
		Description: `Delete a mindmap node. Deleting a parent may affect children. Uses v2-experimental API.

WARNING: Cannot be undone. Use dry_run=true to preview first.

RETURNS: Confirmation with deleted node ID.`,
	},
}

// Code Widget Tools (v2-experimental)
var codeWidgetTools = []ToolSpec{
	{
		Name:     "miro_create_code_widget",
		Method:   "CreateCodeWidget",
		Title:    "Create Code Widget",
		Category: "create",
		Description: `Create a code widget: a syntax-highlighted code snippet on the board. Uses v2-experimental API. May change or become unavailable before GA.

USE WHEN: "add a code snippet", "put this code on the board", "create a code block"

PARAMETERS:
- code: The source code (max 6000 characters). Required.
- language: For syntax highlighting (e.g. javascript, go, python). Optional.
- title: Widget header text (max 100 characters). Optional.

RETURNS: Widget ID and view link.

FAILS WHEN: code exceeds 6000 characters. board_id not found. Account lacks access to the experimental API.`,
	},
	{
		Name:     "miro_get_code_widget",
		Method:   "GetCodeWidget",
		Title:    "Get Code Widget",
		Category: "read",
		ReadOnly: true,
		Description: `Get code widget details including full source code, language, and position. Uses v2-experimental API. May change or become unavailable before GA.

USE WHEN: "show the code in this widget", "read that code snippet"

RETURNS: Widget ID, full code, language, title, line-number setting, position, size, and parent frame ID.`,
	},
	{
		Name:     "miro_list_code_widgets",
		Method:   "ListCodeWidgets",
		Title:    "List Code Widgets",
		Category: "read",
		ReadOnly: true,
		Description: `List code widgets on a board with truncated code previews. Uses v2-experimental API. May change or become unavailable before GA.

USE WHEN: "what code snippets are on this board", "find the code widgets"

RETURNS: Array of widgets with IDs, titles, languages, and 80-char code previews. Use miro_get_code_widget for full source. Paginated via cursor.`,
	},
	{
		Name:       "miro_update_code_widget",
		Method:     "UpdateCodeWidget",
		Title:      "Update Code Widget",
		Category:   "update",
		Idempotent: true,
		Description: `Update a code widget's content, language, title, line numbers, size, or parent frame. Uses v2-experimental API. May change or become unavailable before GA.

USE WHEN: "change the code in that widget", "fix the snippet", "retitle the code block"

NOTE: To move the widget to a new position, use miro_move_code_widget instead. Omitted fields keep their current values.

RETURNS: Confirmation with widget ID.`,
	},
	{
		Name:       "miro_move_code_widget",
		Method:     "MoveCodeWidget",
		Title:      "Move Code Widget",
		Category:   "update",
		Idempotent: true,
		Description: `Move a code widget to a new x/y position on the board. Uses v2-experimental API. May change or become unavailable before GA.

USE WHEN: "move the code widget", "reposition that snippet"

NOTE: Position only (center origin). To change content or size, use miro_update_code_widget instead.

RETURNS: Confirmation with widget ID and new coordinates.`,
	},
	{
		Name:        "miro_delete_code_widget",
		Method:      "DeleteCodeWidget",
		Title:       "Delete Code Widget",
		Category:    "delete",
		Destructive: true,
		Description: `Delete a code widget from a board. Uses v2-experimental API. May change or become unavailable before GA.

WARNING: Cannot be undone. Use dry_run=true to preview first.

RETURNS: Confirmation with deleted widget ID.`,
	},
}

// Comment Tools (v2-experimental)
var commentTools = []ToolSpec{
	{
		Name:     "miro_create_comment",
		Method:   "CreateComment",
		Title:    "Create Comment",
		Category: "comments",
		Description: `Open a comment thread on a board, optionally attached to a specific item. Uses a v2-experimental API that Miro ships undocumented; it may change or become unavailable without notice.

USE WHEN: "comment on this board", "leave feedback on that sticky", "add a note for the team on item X"

PARAMETERS:
- board_id: Required.
- content: Required. First message of the thread.
- item_id: Optional. Attaches the thread to an item; without it the comment lands at the board origin (the API ignores position coordinates on create).

RETURNS: Thread ID, and the item ID when attached.

VOICE-FRIENDLY: "Added a comment on the pricing sticky"`,
	},
	{
		Name:     "miro_list_comments",
		Method:   "ListComments",
		Title:    "List Comments",
		Category: "comments",
		ReadOnly: true,
		Description: `List comment threads on a board, each with its full message history, author, resolved state, and attached item ID. Uses a v2-experimental API that Miro ships undocumented; it may change or become unavailable without notice.

USE WHEN: "what feedback is on this board", "show unresolved comments", "read the comments"

PARAMETERS:
- board_id: Required.
- limit: Max threads per page (default 20, max 50). offset: zero-based pagination.

RETURNS: Threads with messages, count, total, has_more. Filter resolved client-side via each thread's resolved flag.

VOICE-FRIENDLY: "3 comment threads, 1 unresolved: 'move the CTA up' from Olga"`,
	},
	{
		Name:     "miro_get_comment",
		Method:   "GetComment",
		Title:    "Get Comment Thread",
		Category: "comments",
		ReadOnly: true,
		Description: `Get one comment thread with all its messages. Uses a v2-experimental API that Miro ships undocumented; it may change or become unavailable without notice.

USE WHEN: "show that comment thread", "what did they reply"

PARAMETERS:
- board_id, comment_id: Required. Get comment_id from miro_list_comments.

RETURNS: Thread with messages, author info, resolved state, attached item ID.`,
	},
	{
		Name:     "miro_reply_comment",
		Method:   "ReplyComment",
		Title:    "Reply to Comment",
		Category: "comments",
		Description: `Append a reply to an existing comment thread. Uses a v2-experimental API that Miro ships undocumented; it may change or become unavailable without notice.

USE WHEN: "reply to that comment", "answer the feedback"

PARAMETERS:
- board_id, comment_id: Required. Get comment_id from miro_list_comments.
- content: Required. Text of the reply.

RETURNS: Thread ID and new message count.

VOICE-FRIENDLY: "Replied to the thread - now 3 messages"`,
	},
	{
		Name:       "miro_resolve_comment",
		Method:     "ResolveComment",
		Title:      "Resolve Comment",
		Category:   "comments",
		Idempotent: true,
		Description: `Resolve a comment thread, or reopen it with resolved=false. Uses a v2-experimental API that Miro ships undocumented; it may change or become unavailable without notice.

USE WHEN: "mark that feedback as done", "resolve the comment", "reopen that thread"

PARAMETERS:
- board_id, comment_id: Required.
- resolved: Optional. Defaults to true; pass false to reopen.

RETURNS: Thread ID and its new resolved state.

VOICE-FRIENDLY: "Resolved - 2 threads still open"`,
	},
}
