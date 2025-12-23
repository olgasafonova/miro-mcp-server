package miro

// =============================================================================
// Group Types
// =============================================================================

// Group represents a group of items on a board.
type Group struct {
	ID    string   `json:"id"`
	Items []string `json:"items,omitempty"`
}

// =============================================================================
// Create Group
// =============================================================================

// CreateGroupArgs contains parameters for creating a group.
type CreateGroupArgs struct {
	BoardID string   `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemIDs []string `json:"item_ids" jsonschema:"required" jsonschema_description:"IDs of items to group together (minimum 2)"`
}

// CreateGroupResult contains the created group.
type CreateGroupResult struct {
	ID      string   `json:"id"`
	ItemIDs []string `json:"item_ids"`
	Message string   `json:"message"`
}

// =============================================================================
// Ungroup
// =============================================================================

// UngroupArgs contains parameters for ungrouping items.
type UngroupArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	GroupID string `json:"group_id" jsonschema:"required" jsonschema_description:"ID of the group to ungroup"`
}

// UngroupResult confirms ungrouping.
type UngroupResult struct {
	Success bool   `json:"success"`
	GroupID string `json:"group_id"`
	Message string `json:"message"`
}

// =============================================================================
// List Groups
// =============================================================================

// ListGroupsArgs contains parameters for listing groups on a board.
type ListGroupsArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"Max groups to return (default 50)"`
	Cursor  string `json:"cursor,omitempty" jsonschema_description:"Pagination cursor"`
}

// ListGroupsResult contains the list of groups.
type ListGroupsResult struct {
	Groups  []Group `json:"groups"`
	Count   int     `json:"count"`
	HasMore bool    `json:"has_more"`
	Cursor  string  `json:"cursor,omitempty"`
	Message string  `json:"message"`
}

// =============================================================================
// Get Group
// =============================================================================

// GetGroupArgs contains parameters for getting a specific group.
type GetGroupArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	GroupID string `json:"group_id" jsonschema:"required" jsonschema_description:"Group ID to retrieve"`
}

// GetGroupResult contains the group details.
type GetGroupResult struct {
	ID      string   `json:"id"`
	Items   []string `json:"items"`
	Message string   `json:"message"`
}

// =============================================================================
// Get Group Items
// =============================================================================

// GetGroupItemsArgs contains parameters for getting items in a group.
type GetGroupItemsArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	GroupID string `json:"group_id" jsonschema:"required" jsonschema_description:"Group ID"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"Max items to return (default 50)"`
	Cursor  string `json:"cursor,omitempty" jsonschema_description:"Pagination cursor"`
}

// GetGroupItemsResult contains the items in a group.
type GetGroupItemsResult struct {
	Items   []ItemSummary `json:"items"`
	Count   int           `json:"count"`
	HasMore bool          `json:"has_more"`
	Message string        `json:"message"`
}

// =============================================================================
// Update Group
// =============================================================================

// UpdateGroupArgs contains parameters for updating a group's items.
type UpdateGroupArgs struct {
	BoardID string   `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	GroupID string   `json:"group_id" jsonschema:"required" jsonschema_description:"Group ID to update"`
	ItemIDs []string `json:"item_ids" jsonschema:"required" jsonschema_description:"New list of item IDs for the group (replaces current items)"`
}

// UpdateGroupResult contains the updated group details.
type UpdateGroupResult struct {
	ID      string   `json:"id"`
	ItemIDs []string `json:"item_ids"`
	Message string   `json:"message"`
}

// =============================================================================
// Delete Group
// =============================================================================

// DeleteGroupArgs contains parameters for deleting a group.
type DeleteGroupArgs struct {
	BoardID     string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	GroupID     string `json:"group_id" jsonschema:"required" jsonschema_description:"Group ID to delete"`
	DeleteItems bool   `json:"delete_items,omitempty" jsonschema_description:"Also delete the items in the group (default: false, items are ungrouped)"`
}

// DeleteGroupResult confirms group deletion.
type DeleteGroupResult struct {
	Success bool   `json:"success"`
	GroupID string `json:"group_id"`
	Message string `json:"message"`
}
