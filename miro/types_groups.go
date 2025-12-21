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
