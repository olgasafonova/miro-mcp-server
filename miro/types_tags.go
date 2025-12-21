package miro

// =============================================================================
// Tag Types
// =============================================================================

// Tag represents a tag that can be attached to items.
type Tag struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	FillColor string `json:"fillColor,omitempty"`
}

// =============================================================================
// Create Tag
// =============================================================================

// CreateTagArgs contains parameters for creating a tag.
type CreateTagArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Title   string `json:"title" jsonschema:"required" jsonschema_description:"Tag text (e.g., 'Urgent', 'Done', 'Review')"`
	Color   string `json:"color,omitempty" jsonschema_description:"Tag color: red, magenta, violet, blue, cyan, green, yellow, orange, gray"`
}

// CreateTagResult contains the created tag.
type CreateTagResult struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Color   string `json:"color"`
	Message string `json:"message"`
}

// =============================================================================
// List Tags
// =============================================================================

// ListTagsArgs contains parameters for listing tags on a board.
type ListTagsArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"Max tags to return (default 50)"`
}

// ListTagsResult contains the list of tags.
type ListTagsResult struct {
	Tags    []Tag  `json:"tags"`
	Count   int    `json:"count"`
	Message string `json:"message"`
}

// =============================================================================
// Attach Tag
// =============================================================================

// AttachTagArgs contains parameters for attaching a tag to an item.
type AttachTagArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"required" jsonschema_description:"ID of the item to tag (sticky note only)"`
	TagID   string `json:"tag_id" jsonschema:"required" jsonschema_description:"ID of the tag to attach"`
}

// AttachTagResult confirms tag attachment.
type AttachTagResult struct {
	Success bool   `json:"success"`
	ItemID  string `json:"item_id"`
	TagID   string `json:"tag_id"`
	Message string `json:"message"`
}

// =============================================================================
// Detach Tag
// =============================================================================

// DetachTagArgs contains parameters for removing a tag from an item.
type DetachTagArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"required" jsonschema_description:"ID of the item to untag"`
	TagID   string `json:"tag_id" jsonschema:"required" jsonschema_description:"ID of the tag to remove"`
}

// DetachTagResult confirms tag removal.
type DetachTagResult struct {
	Success bool   `json:"success"`
	ItemID  string `json:"item_id"`
	TagID   string `json:"tag_id"`
	Message string `json:"message"`
}

// =============================================================================
// Get Item Tags
// =============================================================================

// GetItemTagsArgs contains parameters for listing tags on an item.
type GetItemTagsArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"required" jsonschema_description:"ID of the item"`
}

// GetItemTagsResult contains tags attached to an item.
type GetItemTagsResult struct {
	Tags    []Tag  `json:"tags"`
	Count   int    `json:"count"`
	ItemID  string `json:"item_id"`
	Message string `json:"message"`
}
