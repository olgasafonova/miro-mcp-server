package tools

// Doc Format Tools (Markdown Documents)
var docFormatTools = []ToolSpec{
	{
		Name:     "miro_create_doc",
		Method:   "CreateDocFormat",
		Title:    "Create Doc Format Item",
		Category: "create",
		Description: `Create a rich text document on a Miro board from Markdown content.

USE WHEN: User says "add a document", "create a doc from markdown", "put markdown on the board"

PARAMETERS:
- board_id: Required
- content: Markdown text (required). Supports headings, lists, bold, italic, links, code blocks.
- x, y: Position
- parent_id: Frame ID to place doc in

EXAMPLE:
{"board_id": "uXjVN1234", "content": "# Sprint Goals\n- Ship v2.0\n- Fix critical bugs"}

RELATED: Use miro_get_doc to read doc content. Use miro_delete_doc to remove. For URL-based documents, use miro_create_document instead.

VOICE-FRIENDLY: "Created doc format item on board"`,
	},
	{
		Name:     "miro_get_doc",
		Method:   "GetDocFormat",
		Title:    "Get Doc Format Item",
		Category: "read",
		ReadOnly: true,
		Description: `Get details of a doc format item by ID.

USE WHEN: User asks "show me that document", "what's in this doc", "read the document"

PARAMETERS:
- board_id: Required
- item_id: Doc format item ID (required)

RETURNS: Document content (Markdown), position, timestamps.

RELATED: Use miro_create_doc to create new documents. Use miro_delete_doc to remove.`,
	},
	{
		Name:        "miro_delete_doc",
		Method:      "DeleteDocFormat",
		Title:       "Delete Doc Format Item",
		Category:    "delete",
		Destructive: true,
		Description: `Delete a doc format item from a Miro board.

USE WHEN: User says "remove the document", "delete that doc"

PARAMETERS:
- board_id: Required
- item_id: Doc format item ID to delete (required)
- dry_run: If true, returns preview without deleting (optional)

WARNING: This action cannot be undone.
Use dry_run=true to preview what will be deleted before executing.

RELATED: Use miro_get_doc to inspect before deleting. Use miro_create_doc to create new documents.`,
	},
	{
		Name:       "miro_update_doc",
		Method:     "UpdateDocFormat",
		Title:      "Update Doc Format Item",
		Category:   "update",
		Idempotent: true,
		Description: `Update a doc format item's Markdown content. Supports two modes: full content replacement or find-and-replace.

USE WHEN: "edit the document", "update doc content", "change the text in that doc", "replace X with Y in the document"

NOT FOR: Creating new documents (use ` + "`miro_create_doc`" + `). Updating position only (use ` + "`miro_update_item`" + `).

PARAMETERS:
- board_id: Required
- item_id: Doc format item ID (required). Get from miro_get_doc or miro_list_items.
- content: New Markdown content (for full replacement mode)
- old_content: Text to find (for find-and-replace mode)
- new_content: Replacement text (for find-and-replace mode)
- replace_all: Replace all occurrences (default: first only)

MODE 1 - Full replacement:
{"board_id": "uXjVN1234", "item_id": "345876...", "content": "# New Title\n\nNew content"}

MODE 2 - Find and replace:
{"board_id": "uXjVN1234", "item_id": "345876...", "old_content": "Draft", "new_content": "Final", "replace_all": true}

NOTE: The item ID changes after update because Miro's API requires delete+recreate. The new ID is returned. Position is preserved.

RETURNS: New item ID, old item ID, updated content, view link.

RELATED: Use miro_get_doc to read before editing. Use miro_create_doc to create new documents.

VOICE-FRIENDLY: "Updated document content"`,
	},
}

// Table Tools (data_table_format)
var tableTools = []ToolSpec{
	{
		Name:     "miro_list_tables",
		Method:   "ListTables",
		Title:    "List Tables",
		Category: "read",
		ReadOnly: true,
		Description: `List tables (data_table_format items) on a Miro board. Returns table metadata: ID, position, size, and timestamps. Use the table ID with miro_get_table for details.

USE WHEN: "find tables on this board", "list all tables", "does this board have tables", "show me the tables"

NOT FOR: Reading table row data or column definitions. The Miro REST API provides table metadata only. For full table content, open the board in Miro.

PARAMETERS:
- board_id: Required
- limit: Max tables to return (default 10, max 50)
- cursor: Pagination cursor from previous response

RETURNS: Table items with IDs, positions, sizes, and timestamps.

VOICE-FRIENDLY: "Found 3 tables on the board"`,
	},
	{
		Name:     "miro_get_table",
		Method:   "GetTable",
		Title:    "Get Table Details",
		Category: "read",
		ReadOnly: true,
		Description: `Get metadata for a specific table (data_table_format item) by ID. Returns position, size, parent frame, and timestamps.

USE WHEN: "get table details", "where is that table", "table info"

NOT FOR: Reading table row data or column definitions. The Miro REST API provides table metadata only.

PARAMETERS:
- board_id: Required
- item_id: Table item ID (required). Get from miro_list_tables or miro_list_items with type filter.

RETURNS: Table ID, position, size, parent frame ID, timestamps, and view link.

RELATED: Use miro_list_tables to discover tables. Use miro_list_items with type "data_table_format" as an alternative.

VOICE-FRIENDLY: "Table is at position (100, 200), size 400x300"`,
	},
}

// Upload Tools (File Upload)
var uploadTools = []ToolSpec{
	{
		Name:     "miro_upload_image",
		Method:   "UploadImage",
		Title:    "Upload Image from File",
		Category: "create",
		Description: `Upload a local image file to a Miro board.

USE WHEN: User says "upload this image", "add screenshot to board", "upload png/jpg/gif/svg file". Use this for image files (png, jpg, gif, webp, svg). For documents (pdf, docx, pptx), use miro_upload_document instead.

PARAMETERS:
- board_id: Required
- file_path: Absolute path to the image file (required). Supports: png, jpg, jpeg, gif, webp, svg.
- title: Image title/alt text
- x, y: Position
- parent_id: Frame ID to place image in

NOTE: The file must exist on the local filesystem. For remote images, use miro_create_image with a URL instead.

RELATED: To upload a document file (pdf, docx, etc.), use miro_upload_document.

VOICE-FRIENDLY: "Uploaded image 'screenshot.png' to board"`,
	},

	{
		Name:     "miro_upload_document",
		Method:   "UploadDocument",
		Title:    "Upload Document from File",
		Category: "create",
		Description: `Upload a local document file to a Miro board.

USE WHEN: User says "upload this document", "add PDF to board", "upload spreadsheet/presentation file". Use this for document files (pdf, doc, docx, ppt, pptx, xls, xlsx, txt, rtf, csv). For images (png, jpg, gif), use miro_upload_image instead.

PARAMETERS:
- board_id: Required
- file_path: Absolute path to the document file (required). Supports: pdf, doc, docx, ppt, pptx, xls, xlsx, txt, rtf, csv. Max 6 MB.
- title: Document title
- x, y: Position
- parent_id: Frame ID to place document in

NOTE: The file must exist on the local filesystem. For remote documents, use miro_create_document with a URL instead.

RELATED: To upload a local image instead, use miro_upload_image.

VOICE-FRIENDLY: "Uploaded document 'report.pdf' to board"`,
	},
}

// Update from File Tools (PATCH multipart)
var updateFromFileTools = []ToolSpec{
	{
		Name:     "miro_update_image_from_file",
		Method:   "UpdateImageFromFile",
		Title:    "Replace Image File",
		Category: "update",
		Description: `Replace the file on an existing image item with a new local image file.

USE WHEN: User says "replace this image", "swap the screenshot", "update the image file". Use this to change the file on an existing image item without creating a new one. For updating metadata only (title, position), use miro_update_image instead.

PARAMETERS:
- board_id: Required
- item_id: Required. The existing image item to update.
- file_path: Absolute path to the new image file (required). Supports: png, jpg, jpeg, gif, webp, svg.
- title: New image title/alt text
- x, y: New position
- parent_id: Frame ID to move image into

NOTE: The item must already exist as an image. The file must exist on the local filesystem.

RELATED: To create a new image from file, use miro_upload_image. To update metadata only, use miro_update_image.

VOICE-FRIENDLY: "Replaced image file on item"`,
	},
	{
		Name:     "miro_update_document_from_file",
		Method:   "UpdateDocumentFromFile",
		Title:    "Replace Document File",
		Category: "update",
		Description: `Replace the file on an existing document item with a new local document file.

USE WHEN: User says "replace this document", "update the PDF", "swap the file on this document". Use this to change the file on an existing document item without creating a new one. For updating metadata only (title, position), use miro_update_document instead.

PARAMETERS:
- board_id: Required
- item_id: Required. The existing document item to update.
- file_path: Absolute path to the new document file (required). Supports: pdf, doc, docx, ppt, pptx, xls, xlsx, txt, rtf, csv. Max 6 MB.
- title: New document title
- x, y: New position
- parent_id: Frame ID to move document into

NOTE: The item must already exist as a document. The file must exist on the local filesystem.

RELATED: To create a new document from file, use miro_upload_document. To update metadata only, use miro_update_document.

VOICE-FRIENDLY: "Replaced document file on item"`,
	},
}
