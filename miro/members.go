package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// =============================================================================
// Board Member Operations
// =============================================================================

// ListBoardMembers retrieves members with access to a board.
func (c *Client) ListBoardMembers(ctx context.Context, args ListBoardMembersArgs) (ListBoardMembersResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListBoardMembersResult{}, err
	}

	params := url.Values{}
	limit := 50
	if args.Limit > 0 && args.Limit <= 100 {
		limit = args.Limit
	}
	params.Set("limit", strconv.Itoa(limit))
	if args.Offset != "" {
		params.Set("offset", args.Offset)
	}

	path := "/boards/" + args.BoardID + "/members?" + params.Encode()

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ListBoardMembersResult{}, err
	}

	var resp struct {
		Data   []BoardMember `json:"data"`
		Total  int           `json:"total,omitempty"`
		Offset int           `json:"offset,omitempty"` // Miro API returns numeric offset
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ListBoardMembersResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	message := fmt.Sprintf("Found %d board members", len(resp.Data))
	if len(resp.Data) == 0 {
		message = "No members found on this board"
	}

	return ListBoardMembersResult{
		Members: resp.Data,
		Count:   len(resp.Data),
		HasMore: resp.Offset > 0 && len(resp.Data) >= limit,
		Message: message,
	}, nil
}

// ShareBoard shares a board with a user by email.
func (c *Client) ShareBoard(ctx context.Context, args ShareBoardArgs) (ShareBoardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ShareBoardResult{}, err
	}
	if args.Email == "" {
		return ShareBoardResult{}, fmt.Errorf("email is required")
	}

	// Default role
	role := args.Role
	if role == "" {
		role = "viewer"
	}

	// Validate role
	validRoles := map[string]bool{"viewer": true, "commenter": true, "editor": true}
	if !validRoles[role] {
		return ShareBoardResult{}, fmt.Errorf("invalid role '%s': must be viewer, commenter, or editor", role)
	}

	reqBody := map[string]interface{}{
		"emails": []string{args.Email},
		"role":   role,
	}

	if args.Message != "" {
		reqBody["message"] = args.Message
	}

	_, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/members", reqBody)
	if err != nil {
		return ShareBoardResult{
			Success: false,
			Email:   args.Email,
			Role:    role,
			Message: fmt.Sprintf("Failed to share board: %v", err),
		}, err
	}

	return ShareBoardResult{
		Success: true,
		Email:   args.Email,
		Role:    role,
		Message: fmt.Sprintf("Shared board with %s as %s", args.Email, role),
	}, nil
}
