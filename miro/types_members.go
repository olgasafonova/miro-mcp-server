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

// =============================================================================
// Get Board Member
// =============================================================================

// GetBoardMemberArgs contains parameters for getting a specific board member.
type GetBoardMemberArgs struct {
	BoardID  string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	MemberID string `json:"member_id" jsonschema:"required" jsonschema_description:"Member ID to retrieve"`
}

// GetBoardMemberResult contains the board member details.
type GetBoardMemberResult struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
	Role    string `json:"role"`
	Message string `json:"message"`
}

// =============================================================================
// Remove Board Member
// =============================================================================

// RemoveBoardMemberArgs contains parameters for removing a board member.
type RemoveBoardMemberArgs struct {
	BoardID  string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	MemberID string `json:"member_id" jsonschema:"required" jsonschema_description:"Member ID to remove"`
}

// RemoveBoardMemberResult confirms member removal.
type RemoveBoardMemberResult struct {
	Success  bool   `json:"success"`
	MemberID string `json:"member_id"`
	Message  string `json:"message"`
}

// =============================================================================
// Update Board Member
// =============================================================================

// UpdateBoardMemberArgs contains parameters for updating a board member's role.
type UpdateBoardMemberArgs struct {
	BoardID  string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	MemberID string `json:"member_id" jsonschema:"required" jsonschema_description:"Member ID to update"`
	Role     string `json:"role" jsonschema:"required" jsonschema_description:"New role: viewer, commenter, or editor"`
}

// UpdateBoardMemberResult contains the updated member details.
type UpdateBoardMemberResult struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
	Role    string `json:"role"`
	Message string `json:"message"`
}
