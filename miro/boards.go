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
func (c *Client) ListBoards(ctx context.Context, args ListBoardsArgs) (ListBoardsResult, error) {
	// Build query parameters
	params := url.Values{}

	// Use TeamID from args, or fall back to config's TeamID
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
	limit := DefaultBoardLimit
	if args.Limit > 0 && args.Limit <= MaxBoardLimit {
		limit = args.Limit
	}
	params.Set("limit", strconv.Itoa(limit))
	if args.Offset != "" {
		params.Set("offset", args.Offset)
	}

	path := "/boards"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

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

	// Convert to summaries
	boards := make([]BoardSummary, len(resp.Data))
	for i, b := range resp.Data {
		boards[i] = BoardSummary{
			ID:          b.ID,
			Name:        b.Name,
			Description: b.Description,
			ViewLink:    b.ViewLink,
		}
		if b.Team != nil {
			boards[i].TeamName = b.Team.Name
		}
	}

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

// CreateBoard creates a new Miro board.
func (c *Client) CreateBoard(ctx context.Context, args CreateBoardArgs) (CreateBoardResult, error) {
	if args.Name == "" {
		return CreateBoardResult{}, fmt.Errorf("name is required")
	}

	reqBody := map[string]interface{}{
		"name": args.Name,
	}

	if args.Description != "" {
		reqBody["description"] = args.Description
	}

	if args.TeamID != "" {
		reqBody["teamId"] = args.TeamID
	}

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

	reqBody := make(map[string]interface{})

	if args.Name != "" {
		reqBody["name"] = args.Name
	}
	if args.Description != "" {
		reqBody["description"] = args.Description
	}
	if args.TeamID != "" {
		reqBody["teamId"] = args.TeamID
	}

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

	reqBody := make(map[string]interface{})
	if args.Name != "" {
		reqBody["name"] = args.Name
	}
	if args.Description != "" {
		reqBody["description"] = args.Description
	}

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
