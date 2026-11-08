package tools

// Tag Tools
var tagTools = []ToolSpec{
	{
		Name:     "miro_create_tag",
		Method:   "CreateTag",
		Title:    "Create Tag",
		Category: "tags",
		Description: `Create a tag on a Miro board. Colors: red, magenta, violet, blue, cyan, green, yellow, gray, light_green, dark_green, dark_blue, dark_gray, black.

VOICE-FRIENDLY: "Created red tag 'Urgent'"`,
	},
	{
		Name:     "miro_list_tags",
		Method:   "ListTags",
		Title:    "List Tags",
		Category: "tags",
		ReadOnly: true,
		Description: `List tag definitions on a board with IDs, titles, and colors. Use tag IDs from this response with miro_attach_tag, miro_detach_tag, and miro_get_items_by_tag.

PAGING: returns up to 50 tags per call. Check has_more and pass the returned offset to read the rest; total reports how many exist.

USE WHEN: "show all tags", "what tags exist", "list labels", or before attaching a tag to get its ID

VOICE-FRIENDLY: "Board has 8 tags: Urgent (red), Done (green), Review (blue)..."`,
	},
	{
		Name:     "miro_attach_tag",
		Method:   "AttachTag",
		Title:    "Attach Tag",
		Category: "tags",
		Description: `Attach an existing tag to a sticky note or card. The tag must already exist; create it first with miro_create_tag if needed. Only sticky_note and card items support tags.

USE WHEN: "tag this sticky as Urgent", "add the Done label", "mark this card with Priority"

FAILS WHEN: tag_id doesn't exist on this board (list with miro_list_tags), item is not a sticky_note or card.

VOICE-FRIENDLY: "Tagged sticky with 'Urgent'"`,
	},
	{
		Name:     "miro_detach_tag",
		Method:   "DetachTag",
		Title:    "Remove Tag",
		Category: "tags",
		Description: `Remove a tag from a sticky note or card. The tag stays on the board for reuse; to delete it entirely, use miro_delete_tag.

USE WHEN: "remove the Urgent tag", "untag this card", "take off the Done label"`,
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

RETURNS: List of tags attached to the item.

RELATED: For the reverse lookup (all items with a specific tag), use miro_get_items_by_tag.`,
	},
	{
		Name:     "miro_get_tag",
		Method:   "GetTag",
		Title:    "Get Tag",
		Category: "tags",
		ReadOnly: true,
		Description: `Get details of a specific tag by ID.

USE WHEN: "tag details", "what color is this tag", "show tag info"

RETURNS: Tag ID, title, and color.

VOICE-FRIENDLY: "Tag 'Urgent' is red"`,
	},
	{
		Name:       "miro_update_tag",
		Method:     "UpdateTag",
		Title:      "Update Tag",
		Category:   "tags",
		Idempotent: true,
		Description: `Update a tag's title or color. At least one must be provided.

USE WHEN: "rename this tag", "change tag color", "update tag to green"

RETURNS: Confirmation with tag ID, updated title, and color.

VOICE-FRIENDLY: "Updated tag to 'Done' with green color"`,
	},
	{
		Name:        "miro_delete_tag",
		Method:      "DeleteTag",
		Title:       "Delete Tag",
		Category:    "tags",
		Destructive: true,
		Description: `Delete a tag from a board. Removes the tag from all items.

USE WHEN: "delete this tag", "remove tag from board", "get rid of tag X"

WARNING: Cannot be undone. Use dry_run=true to preview first.

RETURNS: Confirmation with deleted tag ID.

VOICE-FRIENDLY: "Tag deleted successfully"`,
	},
}

// Group Tools
var groupTools = []ToolSpec{
	{
		Name:     "miro_create_group",
		Method:   "CreateGroup",
		Title:    "Group Items",
		Category: "create",
		Description: `Group multiple items together logically (minimum 2). Grouped items move and resize together. For a visible container with a border and title, use miro_create_frame.

USE WHEN: "group these items", "bundle items together", "make a group"

RETURNS: Group ID and member item IDs.

VOICE-FRIENDLY: "Grouped 4 items together"`,
	},
	{
		Name:     "miro_list_groups",
		Method:   "ListGroups",
		Title:    "List Groups",
		Category: "read",
		ReadOnly: true,
		Description: `List all groups on a Miro board.

USE WHEN: "show all groups", "list groups on board", "what groups exist"

RETURNS: Array of group IDs. Paginated via cursor.

VOICE-FRIENDLY: "Found 3 groups on the board"`,
	},
	{
		Name:     "miro_get_group",
		Method:   "GetGroup",
		Title:    "Get Group Details",
		Category: "read",
		ReadOnly: true,
		Description: `Get details of a specific group by ID.

USE WHEN: "group details", "what's in this group", "show group info"

RETURNS: Group ID and member item IDs.

VOICE-FRIENDLY: "This group contains 4 items"`,
	},
	{
		Name:     "miro_get_group_items",
		Method:   "GetGroupItems",
		Title:    "Get Group Items",
		Category: "read",
		ReadOnly: true,
		Description: `Get items in a group with their details. For items inside a visual frame, use miro_get_frame_items.

USE WHEN: "list items in this group", "what items are grouped together", "show group members"

RETURNS: Array of items with IDs, types, and content.

VOICE-FRIENDLY: "Group has 4 items: 2 stickies, 1 shape, 1 text"`,
	},
	{
		Name:       "miro_update_group",
		Method:     "UpdateGroup",
		Title:      "Update Group",
		Category:   "update",
		Idempotent: true,
		Description: `Update a group's member items. Replaces all members; include existing IDs to keep them. Minimum 2 items.

USE WHEN: "add item to group", "remove item from group", "change group members"

RETURNS: Group ID and updated member item IDs.

FAILS WHEN: Fewer than 2 item IDs provided. Item IDs not found on the board.

VOICE-FRIENDLY: "Updated group with 5 items"`,
	},
	{
		Name:        "miro_delete_group",
		Method:      "DeleteGroup",
		Title:       "Delete Group",
		Category:    "delete",
		Destructive: true,
		Description: `Delete a group. Set delete_items=true to also delete items (default: items are released to move independently).

USE WHEN: deleting a group OR ungrouping items. With delete_items=false (default), items are ungrouped and remain on the board. With delete_items=true, both the group and its items are permanently deleted.

WARNING: Deleting items (delete_items=true) cannot be undone. Use dry_run=true to preview first.

RETURNS: Confirmation with deleted group ID.

VOICE-FRIENDLY: "Group deleted, items ungrouped"`,
	},
}

// Get Items By Tag
var itemsByTagTools = []ToolSpec{
	{
		Name:     "miro_get_items_by_tag",
		Method:   "GetItemsByTag",
		Title:    "Get Items By Tag",
		Category: "tags",
		ReadOnly: true,
		Description: `Get all items on a board that have a specific tag attached.

USE WHEN: User asks "show items tagged Urgent", "what's labeled Done", "find all items with this tag"

PARAMETERS:
- board_id: Required
- tag_id: Tag ID to filter by (required). Get tag IDs from list_tags.
- limit: Max items (default 50, max 50)
- offset: Pagination offset

RETURNS: List of items with IDs, types, and content that have the specified tag.

RELATED: Use miro_list_tags to get tag IDs. Use miro_get_item_tags for the reverse lookup (tags on a specific item). Use miro_attach_tag / miro_detach_tag to manage tag assignments.

VOICE-FRIENDLY: "Found 7 items tagged 'Urgent'"`,
	},
}
