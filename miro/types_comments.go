package miro

// =============================================================================
// Comment Types (v2-experimental)
// =============================================================================
//
// A Miro comment is a thread: the create call opens it with a first message,
// replies append to messages[], and resolved is a thread-level flag. The
// endpoints are live but undocumented (absent from the OpenAPI spec); shapes
// below were captured from the wire on 13-08-2026.

// CommentMessage is one message inside a comment thread.
type CommentMessage struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt string    `json:"created_at,omitempty"`
	CreatedBy *UserInfo `json:"created_by,omitempty"`
}

// CommentSummary is a comment thread projected for MCP responses.
type CommentSummary struct {
	ID        string           `json:"id"`
	Resolved  bool             `json:"resolved"`
	CreatedAt string           `json:"created_at,omitempty"`
	CreatedBy *UserInfo        `json:"created_by,omitempty"`
	ItemID    string           `json:"item_id,omitempty"` // set when the thread is attached to an item
	Messages  []CommentMessage `json:"messages"`
}

// CreateCommentArgs contains parameters for opening a comment thread.
type CreateCommentArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID to comment on"`
	Content string `json:"content" jsonschema:"Text of the comment (first message of the thread)"`
	ItemID  string `json:"item_id,omitempty" jsonschema:"Optional item ID to attach the comment to. Omit for a board-level comment."`
}

// CreateCommentResult contains the created comment thread.
type CreateCommentResult struct {
	ID      string `json:"id"`
	ItemID  string `json:"item_id,omitempty"`
	Message string `json:"message"`
}

// ListCommentsArgs contains parameters for listing comment threads on a board.
type ListCommentsArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID to list comments from"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Max threads to return (default 20, max 50)"`
	Offset  int    `json:"offset,omitempty" jsonschema:"Zero-based offset for pagination"`
}

// ListCommentsResult contains the comment threads on a board.
type ListCommentsResult struct {
	Comments []CommentSummary `json:"comments"`
	Count    int              `json:"count"`
	Total    int              `json:"total"`
	HasMore  bool             `json:"has_more"`
	Message  string           `json:"message,omitempty"`
}

// GetCommentArgs contains parameters for fetching one comment thread.
type GetCommentArgs struct {
	BoardID   string `json:"board_id" jsonschema:"Board ID the comment belongs to"`
	CommentID string `json:"comment_id" jsonschema:"Comment thread ID"`
}

// GetCommentResult contains one comment thread with all its messages.
type GetCommentResult struct {
	CommentSummary
	Message string `json:"message"`
}

// ReplyCommentArgs contains parameters for replying to a comment thread.
type ReplyCommentArgs struct {
	BoardID   string `json:"board_id" jsonschema:"Board ID the comment belongs to"`
	CommentID string `json:"comment_id" jsonschema:"Comment thread ID to reply to"`
	Content   string `json:"content" jsonschema:"Text of the reply"`
}

// ReplyCommentResult contains the thread after the reply was appended.
type ReplyCommentResult struct {
	ID           string `json:"id"`
	MessageCount int    `json:"message_count"`
	Message      string `json:"message"`
}

// ResolveCommentArgs contains parameters for resolving or reopening a thread.
type ResolveCommentArgs struct {
	BoardID   string `json:"board_id" jsonschema:"Board ID the comment belongs to"`
	CommentID string `json:"comment_id" jsonschema:"Comment thread ID"`
	Resolved  *bool  `json:"resolved,omitempty" jsonschema:"true to resolve (default), false to reopen"`
}

// ResolveCommentResult confirms the new resolved state.
type ResolveCommentResult struct {
	ID       string `json:"id"`
	Resolved bool   `json:"resolved"`
	Message  string `json:"message"`
}
