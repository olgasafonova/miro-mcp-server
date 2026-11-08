package tools

// Update/Delete Tools
var updateDeleteTools = []ToolSpec{
	{
		Name:       "miro_update_item",
		Method:     "UpdateItem",
		Title:      "Update Item",
		Category:   "update",
		Idempotent: true,
		Description: `Update any item's content, position, or style. For sticky-specific options (color, shape), use ` + "`miro_update_sticky`" + `. For card fields, use ` + "`miro_update_card`" + `. For shape styling, use ` + "`miro_update_shape`" + `.

USE WHEN: "change sticky text", "move this item", "update the color"

RETURNS: Confirmation with item ID.`,
	},
	{
		Name:       "miro_update_sticky",
		Method:     "UpdateSticky",
		Title:      "Update Sticky Note",
		Category:   "update",
		Idempotent: true,
		Description: `Update a sticky note with type-specific options (shape: square/rectangle, sticky colors). For generic updates, use miro_update_item.

USE WHEN: "change sticky color", "update sticky to square", "resize sticky note"

RETURNS: Confirmation with item ID.

VOICE-FRIENDLY: "Updated sticky to yellow square"`,
	},
	{
		Name:       "miro_update_shape",
		Method:     "UpdateShape",
		Title:      "Update Shape",
		Category:   "update",
		Idempotent: true,
		Description: `Update a shape with type-specific options (fill_color, text_color, text alignment, shape type). For generic updates, use miro_update_item.

USE WHEN: "change shape color", "center the text on this shape", "update shape to circle", "resize this shape"

PARAMETERS:
- text_align: Horizontal text alignment: left, center, right.
- text_align_vertical: Vertical text alignment: top, middle, bottom.

RETURNS: Confirmation with item ID.

VOICE-FRIENDLY: "Updated shape to blue circle with centered text"`,
	},
	{
		Name:       "miro_update_text",
		Method:     "UpdateText",
		Title:      "Update Text",
		Category:   "update",
		Idempotent: true,
		Description: `Update a text element (content, font_size, color, position).

USE WHEN: "change the text", "update heading", "edit this text element"

RETURNS: Confirmation with item ID.

VOICE-FRIENDLY: "Updated text to 'New Title'"`,
	},
	{
		Name:       "miro_update_card",
		Method:     "UpdateCard",
		Title:      "Update Card",
		Category:   "update",
		Idempotent: true,
		Description: `Update a card (title, description, due_date, position).

USE WHEN: "update card title", "change due date", "edit this card"

RETURNS: Confirmation with item ID.

VOICE-FRIENDLY: "Updated card title to 'Review PR'"`,
	},
	{
		Name:       "miro_update_image",
		Method:     "UpdateImage",
		Title:      "Update Image",
		Category:   "update",
		Idempotent: true,
		Description: `Update an image (title, url, position, width).

USE WHEN: "rename this image", "move the image", "change image URL"

RETURNS: Confirmation with item ID.

VOICE-FRIENDLY: "Updated image title to 'Logo'"`,
	},
	{
		Name:       "miro_update_document",
		Method:     "UpdateDocument",
		Title:      "Update Document",
		Category:   "update",
		Idempotent: true,
		Description: `Update a document (title, url, position, width).

USE WHEN: "rename this document", "move the document", "change document URL"

RETURNS: Confirmation with item ID.

VOICE-FRIENDLY: "Updated document title"`,
	},
	{
		Name:       "miro_update_embed",
		Method:     "UpdateEmbed",
		Title:      "Update Embed",
		Category:   "update",
		Idempotent: true,
		Description: `Update an embed (url, mode: inline/modal, dimensions, position).

USE WHEN: "change embed URL", "switch embed to modal", "move this embed"

RETURNS: Confirmation with item ID.

VOICE-FRIENDLY: "Updated embed settings"`,
	},
	{
		Name:        "miro_delete_item",
		Method:      "DeleteItem",
		Title:       "Delete Item",
		Category:    "delete",
		Destructive: true,
		Description: `Delete an item from a Miro board.

USE WHEN: "delete this item", "remove this sticky", "get rid of this shape"

WARNING: Cannot be undone. Use dry_run=true to preview first.

RETURNS: Confirmation with deleted item ID.`,
	},
	{
		Name:       "miro_update_connector",
		Method:     "UpdateConnector",
		Title:      "Update Connector",
		Category:   "update",
		Idempotent: true,
		Description: `Update a connector's style (straight/elbowed/curved), caps, caption, or color.

USE WHEN: "change connector style", "update arrow caption", "restyle this line"

RETURNS: Confirmation with connector ID.`,
	},
	{
		Name:        "miro_delete_connector",
		Method:      "DeleteConnector",
		Title:       "Delete Connector",
		Category:    "delete",
		Destructive: true,
		Description: `Delete a connector from a Miro board.

USE WHEN: "delete this connector", "remove this line", "disconnect these items"

WARNING: Cannot be undone. Use dry_run=true to preview first.

RETURNS: Confirmation with deleted connector ID.

VOICE-FRIENDLY: "Connector deleted successfully"`,
	},
	{
		Name:     "miro_list_connectors",
		Method:   "ListConnectors",
		Title:    "List Connectors",
		Category: "read",
		ReadOnly: true,
		Description: `List all connectors (lines/arrows) on a Miro board.

USE WHEN: "show all connectors", "list arrows on board", "what's connected"

RETURNS: Array of connectors with IDs, start/end item IDs, style, and captions. Paginated via cursor.

VOICE-FRIENDLY: "Found 12 connectors on the board"`,
	},
	{
		Name:     "miro_get_connector",
		Method:   "GetConnector",
		Title:    "Get Connector Details",
		Category: "read",
		ReadOnly: true,
		Description: `Get full details of a specific connector by ID.

USE WHEN: "connector details", "what does this connector link", "show this arrow"

RETURNS: Connector ID, start/end item IDs, style, caps, caption, color, and timestamps.

VOICE-FRIENDLY: "This connector links Item A to Item B with a curved arrow"`,
	},
}
