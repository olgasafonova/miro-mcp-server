package miro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// =============================================================================
// Comment Operations (v2-experimental)
// =============================================================================
//
// The comments endpoints are live on the v2-experimental API but absent from
// Miro's OpenAPI spec (verified by live probe 13-08-2026). Wire facts the
// implementation relies on:
//   - POST  /boards/{id}/comments               opens a thread; content required
//   - GET   /boards/{id}/comments               offset-paginated envelope
//   - GET   /boards/{id}/comments/{cid}         one thread with messages[]
//   - POST  /boards/{id}/comments/{cid}/messages appends a reply
//   - PATCH /boards/{id}/comments/{cid}         {"resolved":bool}
//   - DELETE returns 405, so no delete surface is offered
//   - x/y in the create body are accepted but ignored (comment lands at 0,0);
//     itemId genuinely anchors the thread (position.type becomes "attached"),
//     so item_id is the only placement parameter exposed
//   - unknown body fields are silently accepted, so nothing here relies on
//     server-side rejection of typos

// DefaultCommentLimit is the page size when the caller doesn't specify one.
const DefaultCommentLimit = 20

// MaxCommentLimit caps a requested page size.
const MaxCommentLimit = 50

// wrapExperimentalCommentErr adds the experimental-availability hint to
// ambiguous 403/404 responses, mirroring wrapExperimentalCodeWidgetErr.
func wrapExperimentalCommentErr(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	if isExperimentalAccessDenied(apiErr) {
		return fmt.Errorf("%w (note: comments is a v2-experimental Miro API and may be unavailable for your account or plan)", err)
	}
	return err
}

// apiComment is the wire shape of a comment thread.
type apiComment struct {
	ID        string `json:"id"`
	Resolved  bool   `json:"resolved"`
	CreatedAt string `json:"createdAt"`
	CreatedBy *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"createdBy"`
	Position *struct {
		Type   string `json:"type"`
		ItemID string `json:"itemId"`
	} `json:"position"`
	Messages []struct {
		ID        string `json:"id"`
		Content   string `json:"content"`
		CreatedAt string `json:"createdAt"`
		CreatedBy *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"createdBy"`
	} `json:"messages"`
}

// summarizeComment projects a wire comment to the MCP-facing summary.
func summarizeComment(c apiComment) CommentSummary {
	s := CommentSummary{
		ID:        c.ID,
		Resolved:  c.Resolved,
		CreatedAt: c.CreatedAt,
		Messages:  make([]CommentMessage, 0, len(c.Messages)),
	}
	if c.CreatedBy != nil {
		s.CreatedBy = &UserInfo{ID: c.CreatedBy.ID, Name: c.CreatedBy.Name}
	}
	if c.Position != nil && c.Position.Type == "attached" {
		s.ItemID = c.Position.ItemID
	}
	for _, m := range c.Messages {
		msg := CommentMessage{ID: m.ID, Content: m.Content, CreatedAt: m.CreatedAt}
		if m.CreatedBy != nil {
			msg.CreatedBy = &UserInfo{ID: m.CreatedBy.ID, Name: m.CreatedBy.Name}
		}
		s.Messages = append(s.Messages, msg)
	}
	return s
}

// validateCommentThreadRef checks the board/comment ID pair shared by get,
// reply, and resolve.
func validateCommentThreadRef(boardID, commentID string) error {
	if err := ValidateBoardID(boardID); err != nil {
		return err
	}
	if commentID == "" {
		return fmt.Errorf("comment_id is required")
	}
	return ValidateItemID(commentID)
}

// validateCreateCommentArgs checks required fields and ID formats before a
// create request is issued.
func validateCreateCommentArgs(args CreateCommentArgs) error {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return err
	}
	if args.Content == "" {
		return fmt.Errorf("content is required")
	}
	if args.ItemID == "" {
		return nil
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return fmt.Errorf("invalid item_id: %w", err)
	}
	return nil
}

// buildCreateCommentBody assembles the create request body; itemId is only
// sent when the thread attaches to an item.
func buildCreateCommentBody(args CreateCommentArgs) map[string]interface{} {
	reqBody := map[string]interface{}{"content": args.Content}
	if args.ItemID != "" {
		reqBody["itemId"] = args.ItemID
	}
	return reqBody
}

// createCommentMessage phrases the confirmation for a new thread.
func createCommentMessage(itemID string) string {
	if itemID != "" {
		return fmt.Sprintf("Created comment thread attached to item %s", itemID)
	}
	return "Created comment thread"
}

// CreateComment opens a comment thread on a board, optionally attached to an
// item. Uses the v2-experimental API.
func (c *Client) CreateComment(ctx context.Context, args CreateCommentArgs) (CreateCommentResult, error) {
	if err := validateCreateCommentArgs(args); err != nil {
		return CreateCommentResult{}, err
	}

	respBody, err := c.requestExperimental(ctx, http.MethodPost, "/boards/"+args.BoardID+"/comments", buildCreateCommentBody(args))
	if err != nil {
		return CreateCommentResult{}, wrapExperimentalCommentErr(err)
	}

	var created apiComment
	if err := json.Unmarshal(respBody, &created); err != nil {
		return CreateCommentResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	summary := summarizeComment(created)
	return CreateCommentResult{
		ID:      created.ID,
		ItemID:  summary.ItemID,
		Message: createCommentMessage(summary.ItemID),
	}, nil
}

// clampCommentLimit normalizes a requested page size to the endpoint bounds.
func clampCommentLimit(limit int) int {
	if limit > 0 && limit <= MaxCommentLimit {
		return limit
	}
	return DefaultCommentLimit
}

// ListComments lists comment threads on a board with offset pagination.
// Uses the v2-experimental API.
func (c *Client) ListComments(ctx context.Context, args ListCommentsArgs) (ListCommentsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListCommentsResult{}, err
	}

	limit := clampCommentLimit(args.Limit)
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	if args.Offset > 0 {
		params.Set("offset", strconv.Itoa(args.Offset))
	}

	path := "/boards/" + args.BoardID + "/comments?" + params.Encode()
	respBody, err := c.requestExperimental(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ListCommentsResult{}, wrapExperimentalCommentErr(err)
	}

	var resp struct {
		Data   []apiComment `json:"data"`
		Total  int          `json:"total"`
		Offset int          `json:"offset"`
		Size   int          `json:"size"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ListCommentsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	comments := make([]CommentSummary, 0, len(resp.Data))
	for _, wire := range resp.Data {
		comments = append(comments, summarizeComment(wire))
	}

	result := ListCommentsResult{
		Comments: comments,
		Count:    len(comments),
		Total:    resp.Total,
		HasMore:  resp.Offset+len(comments) < resp.Total,
	}
	if result.Count == 0 {
		result.Message = "No comments on this board"
	}
	return result, nil
}

// GetComment fetches one comment thread with all its messages.
// Uses the v2-experimental API.
func (c *Client) GetComment(ctx context.Context, args GetCommentArgs) (GetCommentResult, error) {
	if err := validateCommentThreadRef(args.BoardID, args.CommentID); err != nil {
		return GetCommentResult{}, err
	}

	path := fmt.Sprintf("/boards/%s/comments/%s", args.BoardID, args.CommentID)
	respBody, err := c.requestExperimental(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetCommentResult{}, wrapExperimentalCommentErr(err)
	}

	var wire apiComment
	if err := json.Unmarshal(respBody, &wire); err != nil {
		return GetCommentResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	summary := summarizeComment(wire)
	return GetCommentResult{
		CommentSummary: summary,
		Message:        fmt.Sprintf("Comment thread with %d message(s)", len(summary.Messages)),
	}, nil
}

// ReplyComment appends a message to an existing comment thread.
// Uses the v2-experimental API.
func (c *Client) ReplyComment(ctx context.Context, args ReplyCommentArgs) (ReplyCommentResult, error) {
	if err := validateCommentThreadRef(args.BoardID, args.CommentID); err != nil {
		return ReplyCommentResult{}, err
	}
	if args.Content == "" {
		return ReplyCommentResult{}, fmt.Errorf("content is required")
	}

	path := fmt.Sprintf("/boards/%s/comments/%s/messages", args.BoardID, args.CommentID)
	respBody, err := c.requestExperimental(ctx, http.MethodPost, path, map[string]interface{}{
		"content": args.Content,
	})
	if err != nil {
		return ReplyCommentResult{}, wrapExperimentalCommentErr(err)
	}

	var wire apiComment
	if err := json.Unmarshal(respBody, &wire); err != nil {
		return ReplyCommentResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return ReplyCommentResult{
		ID:           wire.ID,
		MessageCount: len(wire.Messages),
		Message:      fmt.Sprintf("Replied; thread now has %d message(s)", len(wire.Messages)),
	}, nil
}

// ResolveComment resolves a comment thread, or reopens it when
// args.Resolved is explicitly false. Uses the v2-experimental API.
func (c *Client) ResolveComment(ctx context.Context, args ResolveCommentArgs) (ResolveCommentResult, error) {
	if err := validateCommentThreadRef(args.BoardID, args.CommentID); err != nil {
		return ResolveCommentResult{}, err
	}

	resolved := true
	if args.Resolved != nil {
		resolved = *args.Resolved
	}

	path := fmt.Sprintf("/boards/%s/comments/%s", args.BoardID, args.CommentID)
	respBody, err := c.requestExperimental(ctx, http.MethodPatch, path, map[string]interface{}{
		"resolved": resolved,
	})
	if err != nil {
		return ResolveCommentResult{}, wrapExperimentalCommentErr(err)
	}

	var wire apiComment
	if err := json.Unmarshal(respBody, &wire); err != nil {
		return ResolveCommentResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	verb := "Resolved"
	if !wire.Resolved {
		verb = "Reopened"
	}
	return ResolveCommentResult{
		ID:       wire.ID,
		Resolved: wire.Resolved,
		Message:  fmt.Sprintf("%s comment thread %s", verb, wire.ID),
	}, nil
}
