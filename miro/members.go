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

// boardMembersPage is the envelope /boards/{id}/members returns.
type boardMembersPage struct {
	Data  []BoardMember `json:"data"`
	Total int           `json:"total,omitempty"`
	// Offset is the offset of the page just served, not the next one, so it is
	// parsed for completeness and never used as a cursor. See pagination_offset.go.
	Offset int `json:"offset,omitempty"`
}

// buildMemberListQuery assembles the query parameters for a member listing.
func buildMemberListQuery(args ListBoardMembersArgs, limit int) url.Values {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	if args.Offset != "" {
		params.Set("offset", args.Offset)
	}
	return params
}

// memberRequest bundles the method, path and optional body that travel
// together on every member call, the same shape tagItemRequest uses in tags.go.
type memberRequest struct {
	method string
	path   string
	body   interface{}
}

// fetchMemberJSON issues a request against the member endpoints and decodes
// the response into T. Both the listing envelope and the single-member object
// go through it, so the request-then-decode pair is written once.
func fetchMemberJSON[T any](ctx context.Context, c *Client, req memberRequest) (T, error) {
	var out T
	respBody, err := c.request(ctx, req.method, req.path, req.body)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return out, fmt.Errorf("failed to parse response: %w", err)
	}
	return out, nil
}

// memberPath is the single-member endpoint shared by read, update and remove.
func memberPath(boardID, memberID string) string {
	return "/boards/" + boardID + "/members/" + memberID
}

// ListBoardMembers retrieves members with access to a board.
func (c *Client) ListBoardMembers(ctx context.Context, args ListBoardMembersArgs) (ListBoardMembersResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListBoardMembersResult{}, err
	}

	requestedOffset, err := parseOffsetArg(args.Offset)
	if err != nil {
		return ListBoardMembersResult{}, err
	}

	limit := clampMemberLimit(args.Limit)
	path := "/boards/" + args.BoardID + "/members?" + buildMemberListQuery(args, limit).Encode()

	page, err := fetchMemberJSON[boardMembersPage](ctx, c, memberRequest{method: http.MethodGet, path: path})
	if err != nil {
		return ListBoardMembersResult{}, err
	}

	// The next page starts after the rows we asked for plus the rows we got.
	// Deriving it from the requested offset rather than the echoed one is what
	// keeps this correct on page one, where the echo is always 0.
	next := requestedOffset + len(page.Data)

	return ListBoardMembersResult{
		Members: page.Data,
		Count:   len(page.Data),
		Total:   page.Total,
		HasMore: offsetHasMore(next, page.Total, len(page.Data), limit),
		Offset:  nextOffsetString(next, page.Total, len(page.Data), limit),
		Message: boardMembersMessage(len(page.Data)),
	}, nil
}

// clampMemberLimit normalizes a requested page size. Unlike the items,
// connectors and groups endpoints, /boards/{id}/members accepts a page size
// below MinPagedLimit, so no floor is applied and a caller asking for five
// members receives five.
func clampMemberLimit(limit int) int {
	if limit > 0 && limit <= MaxItemLimitExtended {
		return limit
	}
	return DefaultItemLimit
}

// boardMembersMessage describes a member listing, with an explicit
// zero-result message.
func boardMembersMessage(count int) string {
	if count == 0 {
		return "No members found on this board"
	}
	return fmt.Sprintf("Found %d board members", count)
}

// validateMemberRef validates the (board, member) identifier pair shared by
// the member read/update/remove operations.
func validateMemberRef(boardID, memberID string) error {
	if err := ValidateBoardID(boardID); err != nil {
		return err
	}
	if memberID == "" {
		return fmt.Errorf("member_id is required")
	}
	return nil
}

// validateMemberRole rejects an empty or unrecognized access role.
func validateMemberRole(role string) error {
	if role == "" {
		return fmt.Errorf("role is required")
	}
	if !validMemberRoles[role] {
		return fmt.Errorf("invalid role '%s': must be viewer, commenter, or editor", role)
	}
	return nil
}

// validMemberRoles are the access roles a board can be shared at. Owner and
// coowner appear on existing members but cannot be granted through the API.
var validMemberRoles = map[string]bool{"viewer": true, "commenter": true, "editor": true}

// defaultMemberRole fills in the role an invitation gets when none was asked for.
func defaultMemberRole(role string) string {
	if role == "" {
		return "viewer"
	}
	return role
}

// memberDisplayName picks the most human label available for a member; the
// name is optional on the wire, in which case the email identifies them.
func memberDisplayName(member BoardMember) string {
	if member.Name == "" {
		return member.Email
	}
	return member.Name
}

// buildShareBody assembles the invitation payload. The message is optional.
func buildShareBody(email, role, message string) map[string]interface{} {
	reqBody := map[string]interface{}{
		"emails": []string{email},
		"role":   role,
	}
	if message != "" {
		reqBody["message"] = message
	}
	return reqBody
}

// ShareBoard shares a board with a user by email.
func (c *Client) ShareBoard(ctx context.Context, args ShareBoardArgs) (ShareBoardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ShareBoardResult{}, err
	}
	if args.Email == "" {
		return ShareBoardResult{}, fmt.Errorf("email is required")
	}

	role := defaultMemberRole(args.Role)
	if err := validateMemberRole(role); err != nil {
		return ShareBoardResult{}, err
	}

	reqBody := buildShareBody(args.Email, role, args.Message)
	if _, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/members", reqBody); err != nil {
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

// memberDetail projects a member onto the five fields the read and the update
// both return. UpdateBoardMemberResult declares the same five fields with the
// same JSON tags, so it converts from this directly; adding a field to either
// type breaks that conversion at compile time, which is the point.
func memberDetail(member BoardMember, messageFormat string) GetBoardMemberResult {
	return GetBoardMemberResult{
		ID:      member.ID,
		Name:    member.Name,
		Email:   member.Email,
		Role:    member.Role,
		Message: fmt.Sprintf(messageFormat, memberDisplayName(member), member.Role),
	}
}

// GetBoardMember retrieves a specific board member by ID.
func (c *Client) GetBoardMember(ctx context.Context, args GetBoardMemberArgs) (GetBoardMemberResult, error) {
	if err := validateMemberRef(args.BoardID, args.MemberID); err != nil {
		return GetBoardMemberResult{}, err
	}

	path := memberPath(args.BoardID, args.MemberID)
	member, err := fetchMemberJSON[BoardMember](ctx, c, memberRequest{method: http.MethodGet, path: path})
	if err != nil {
		return GetBoardMemberResult{}, err
	}

	return memberDetail(member, "Member '%s' has role '%s'"), nil
}

// RemoveBoardMember removes a member from a board.
func (c *Client) RemoveBoardMember(ctx context.Context, args RemoveBoardMemberArgs) (RemoveBoardMemberResult, error) {
	if err := validateMemberRef(args.BoardID, args.MemberID); err != nil {
		return RemoveBoardMemberResult{}, err
	}

	if _, err := c.request(ctx, http.MethodDelete, memberPath(args.BoardID, args.MemberID), nil); err != nil {
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
	if err := validateMemberRef(args.BoardID, args.MemberID); err != nil {
		return UpdateBoardMemberResult{}, err
	}
	if err := validateMemberRole(args.Role); err != nil {
		return UpdateBoardMemberResult{}, err
	}

	path := memberPath(args.BoardID, args.MemberID)
	member, err := fetchMemberJSON[BoardMember](ctx, c, memberRequest{
		method: http.MethodPatch,
		path:   path,
		body:   map[string]interface{}{"role": args.Role},
	})
	if err != nil {
		return UpdateBoardMemberResult{}, err
	}

	return UpdateBoardMemberResult(memberDetail(member, "Updated '%s' to role '%s'")), nil
}
