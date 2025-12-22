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

// GetBoardMember retrieves a specific board member by ID.
func (c *Client) GetBoardMember(ctx context.Context, args GetBoardMemberArgs) (GetBoardMemberResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetBoardMemberResult{}, err
	}
	if args.MemberID == "" {
		return GetBoardMemberResult{}, fmt.Errorf("member_id is required")
	}

	path := "/boards/" + args.BoardID + "/members/" + args.MemberID

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetBoardMemberResult{}, err
	}

	var member BoardMember
	if err := json.Unmarshal(respBody, &member); err != nil {
		return GetBoardMemberResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	name := member.Name
	if name == "" {
		name = member.Email
	}

	return GetBoardMemberResult{
		ID:      member.ID,
		Name:    member.Name,
		Email:   member.Email,
		Role:    member.Role,
		Message: fmt.Sprintf("Member '%s' has role '%s'", name, member.Role),
	}, nil
}

// RemoveBoardMember removes a member from a board.
func (c *Client) RemoveBoardMember(ctx context.Context, args RemoveBoardMemberArgs) (RemoveBoardMemberResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return RemoveBoardMemberResult{}, err
	}
	if args.MemberID == "" {
		return RemoveBoardMemberResult{}, fmt.Errorf("member_id is required")
	}

	path := "/boards/" + args.BoardID + "/members/" + args.MemberID

	_, err := c.request(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return RemoveBoardMemberResult{
			Success:  false,
			MemberID: args.MemberID,
			Message:  fmt.Sprintf("Failed to remove member: %v", err),
		}, err
	}

	return RemoveBoardMemberResult{
		Success:  true,
		MemberID: args.MemberID,
		Message:  "Member removed from board",
	}, nil
}

// UpdateBoardMember updates a board member's role.
func (c *Client) UpdateBoardMember(ctx context.Context, args UpdateBoardMemberArgs) (UpdateBoardMemberResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateBoardMemberResult{}, err
	}
	if args.MemberID == "" {
		return UpdateBoardMemberResult{}, fmt.Errorf("member_id is required")
	}
	if args.Role == "" {
		return UpdateBoardMemberResult{}, fmt.Errorf("role is required")
	}

	// Validate role
	validRoles := map[string]bool{"viewer": true, "commenter": true, "editor": true}
	if !validRoles[args.Role] {
		return UpdateBoardMemberResult{}, fmt.Errorf("invalid role '%s': must be viewer, commenter, or editor", args.Role)
	}

	path := "/boards/" + args.BoardID + "/members/" + args.MemberID

	reqBody := map[string]interface{}{
		"role": args.Role,
	}

	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateBoardMemberResult{}, err
	}

	var member BoardMember
	if err := json.Unmarshal(respBody, &member); err != nil {
		return UpdateBoardMemberResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	name := member.Name
	if name == "" {
		name = member.Email
	}

	return UpdateBoardMemberResult{
		ID:      member.ID,
		Name:    member.Name,
		Email:   member.Email,
		Role:    member.Role,
		Message: fmt.Sprintf("Updated '%s' to role '%s'", name, member.Role),
	}, nil
}
