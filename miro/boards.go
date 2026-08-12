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
// Board CRUD Operations
// =============================================================================

// ListBoards retrieves boards accessible to the user.
// clampBoardLimit normalizes a requested page size to the boards endpoint's
// bounds.
func clampBoardLimit(limit int) int {
	if limit > 0 && limit <= MaxBoardLimit {
		return limit
	}
	return DefaultBoardLimit
}

// buildListBoardsQuery assembles the query parameters for a boards listing.
// The team ID falls back to the client config when not supplied.
func (c *Client) buildListBoardsQuery(args ListBoardsArgs, limit int) url.Values {
	params := url.Values{}

	teamID := args.TeamID
	if teamID == "" && c.config != nil {
		teamID = c.config.TeamID
	}
	if teamID != "" {
		params.Set("team_id", teamID)
	}

	if args.Query != "" {
		params.Set("query", args.Query)
	}
	params.Set("limit", strconv.Itoa(limit))
	if args.Offset != "" {
		params.Set("offset", args.Offset)
	}
	return params
}

// summarizeBoard projects one full board object to a list summary. Owner and
// team are optional in the API response, so both stay nil/empty rather than
// being flattened to a zero-valued struct the caller would have to test.
func summarizeBoard(b Board) BoardSummary {
	s := BoardSummary{
		ID:          b.ID,
		Name:        b.Name,
		Description: b.Description,
		ViewLink:    b.ViewLink,
		CreatedAt:   formatOptionalRFC3339(b.CreatedAt),
		ModifiedAt:  formatOptionalRFC3339(b.ModifiedAt),
	}
	if b.Team != nil {
		s.TeamID = b.Team.ID
		s.TeamName = b.Team.Name
	}
	if b.Owner != nil {
		owner := *b.Owner
		s.Owner = &owner
	}
	return s
}

// summarizeBoards projects full board objects to list summaries.
func summarizeBoards(data []Board) []BoardSummary {
	boards := make([]BoardSummary, len(data))
	for i, b := range data {
		boards[i] = summarizeBoard(b)
	}
	return boards
}

func (c *Client) ListBoards(ctx context.Context, args ListBoardsArgs) (ListBoardsResult, error) {
	limit := clampBoardLimit(args.Limit)
	params := c.buildListBoardsQuery(args, limit)

	// params always carries at least the limit, so the query string is
	// unconditionally present.
	path := "/boards?" + params.Encode()

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ListBoardsResult{}, err
	}

	var resp struct {
		Data   []Board `json:"data"`
		Total  int     `json:"total,omitempty"`
		Size   int     `json:"size,omitempty"`
		Offset int     `json:"offset,omitempty"` // Miro API returns numeric offset
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ListBoardsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	boards := summarizeBoards(resp.Data)

	// Convert numeric offset to string for external API compatibility
	offsetStr := ""
	if resp.Offset > 0 {
		offsetStr = fmt.Sprintf("%d", resp.Offset)
	}

	return ListBoardsResult{
		Boards:  boards,
		Count:   len(boards),
		HasMore: resp.Offset > 0 && len(resp.Data) >= limit,
		Offset:  offsetStr,
	}, nil
}

// GetBoard retrieves a specific board by ID.
func (c *Client) GetBoard(ctx context.Context, args GetBoardArgs) (GetBoardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetBoardResult{}, err
	}

	// Check cache
	cacheKey := "board:" + args.BoardID
	if cached, ok := c.getCached(cacheKey); ok {
		return cached.(GetBoardResult), nil
	}

	respBody, err := c.request(ctx, http.MethodGet, "/boards/"+args.BoardID, nil)
	if err != nil {
		return GetBoardResult{}, err
	}

	var board Board
	if err := json.Unmarshal(respBody, &board); err != nil {
		return GetBoardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	result := GetBoardResult{Board: board}

	// Cache the result
	c.setCache(cacheKey, result)

	return result, nil
}

// boardFields collects the optional board attributes (name, description,
// team) that are shared by the create, copy, and update request bodies.
// Empty values are omitted.
func boardFields(name, description, teamID string) map[string]interface{} {
	reqBody := make(map[string]interface{})
	if name != "" {
		reqBody["name"] = name
	}
	if description != "" {
		reqBody["description"] = description
	}
	if teamID != "" {
		reqBody["teamId"] = teamID
	}
	return reqBody
}

// CreateBoard creates a new Miro board.
func (c *Client) CreateBoard(ctx context.Context, args CreateBoardArgs) (CreateBoardResult, error) {
	if args.Name == "" {
		return CreateBoardResult{}, fmt.Errorf("name is required")
	}

	reqBody := boardFields(args.Name, args.Description, args.TeamID)
	respBody, err := c.request(ctx, http.MethodPost, "/boards", reqBody)
	if err != nil {
		return CreateBoardResult{}, err
	}

	var board Board
	if err := json.Unmarshal(respBody, &board); err != nil {
		return CreateBoardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CreateBoardResult{
		ID:       board.ID,
		Name:     board.Name,
		ViewLink: board.ViewLink,
		Message:  fmt.Sprintf("Created board '%s'", board.Name),
	}, nil
}

// CopyBoard copies an existing board.
// Uses PUT /boards?copy_from={board_id} as per Miro API docs.
func (c *Client) CopyBoard(ctx context.Context, args CopyBoardArgs) (CopyBoardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CopyBoardResult{}, err
	}

	reqBody := boardFields(args.Name, args.Description, args.TeamID)

	// Miro API uses PUT /boards?copy_from={board_id} to copy boards
	path := "/boards?copy_from=" + url.QueryEscape(args.BoardID)
	respBody, err := c.request(ctx, http.MethodPut, path, reqBody)
	if err != nil {
		return CopyBoardResult{}, err
	}

	var board Board
	if err := json.Unmarshal(respBody, &board); err != nil {
		return CopyBoardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CopyBoardResult{
		ID:       board.ID,
		Name:     board.Name,
		ViewLink: board.ViewLink,
		Message:  fmt.Sprintf("Copied board to '%s'", board.Name),
	}, nil
}

// DeleteBoard deletes a board.
func (c *Client) DeleteBoard(ctx context.Context, args DeleteBoardArgs) (DeleteBoardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return DeleteBoardResult{}, err
	}

	// Dry-run mode: return preview without deleting
	if args.DryRun {
		return DeleteBoardResult{
			Success: true,
			BoardID: args.BoardID,
			Message: "[DRY RUN] Would delete board " + args.BoardID,
		}, nil
	}

	_, err := c.request(ctx, http.MethodDelete, "/boards/"+args.BoardID, nil)
	if err != nil {
		return DeleteBoardResult{
			Success: false,
			BoardID: args.BoardID,
			Message: fmt.Sprintf("Failed to delete board: %v", err),
		}, err
	}

	// Invalidate cache
	c.cache.Invalidate("board:" + args.BoardID)

	return DeleteBoardResult{
		Success: true,
		BoardID: args.BoardID,
		Message: "Board deleted successfully",
	}, nil
}

// UpdateBoard updates a board's name and/or description.
func (c *Client) UpdateBoard(ctx context.Context, args UpdateBoardArgs) (UpdateBoardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateBoardResult{}, err
	}
	if args.Name == "" && args.Description == "" {
		return UpdateBoardResult{}, fmt.Errorf("at least one of name or description is required")
	}

	reqBody := boardFields(args.Name, args.Description, "")
	respBody, err := c.request(ctx, http.MethodPatch, "/boards/"+args.BoardID, reqBody)
	if err != nil {
		return UpdateBoardResult{}, err
	}

	var board Board
	if err := json.Unmarshal(respBody, &board); err != nil {
		return UpdateBoardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate cache
	c.cache.Invalidate("board:" + args.BoardID)

	return UpdateBoardResult{
		ID:          board.ID,
		Name:        board.Name,
		Description: board.Description,
		ViewLink:    board.ViewLink,
		Message:     fmt.Sprintf("Updated board '%s'", board.Name),
	}, nil
}
