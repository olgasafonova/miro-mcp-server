package miro

// =============================================================================
// Board Member Types
// =============================================================================

// BoardMember represents a member with access to a board.
type BoardMember struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Role  string `json:"role"` // "viewer", "commenter", "editor", "coowner", "owner"
}

// =============================================================================
// List Board Members
// =============================================================================

// ListBoardMembersArgs contains parameters for listing board members.
type ListBoardMembersArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"Max members to return (default 50)"`
	Offset  string `json:"offset,omitempty" jsonschema_description:"Pagination cursor"`
}

// ListBoardMembersResult contains the list of board members.
type ListBoardMembersResult struct {
	Members []BoardMember `json:"members"`
	Count   int           `json:"count"`
	HasMore bool          `json:"has_more"`
	Message string        `json:"message"`
}

// =============================================================================
// Share Board
// =============================================================================

// ShareBoardArgs contains parameters for sharing a board with a user.
type ShareBoardArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID to share"`
	Email   string `json:"email" jsonschema:"required" jsonschema_description:"Email address of the user to invite"`
	Role    string `json:"role,omitempty" jsonschema_description:"Access role: viewer, commenter, editor (default: viewer)"`
	Message string `json:"message,omitempty" jsonschema_description:"Optional message to include in the invitation"`
}

// ShareBoardResult confirms board sharing.
type ShareBoardResult struct {
	Success bool   `json:"success"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	Message string `json:"message"`
}
