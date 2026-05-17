package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// =============================================================================
// Mindmap Operations
// =============================================================================

// shouldSendMindmapPosition reports whether the request body should include
// an explicit position block. Root nodes (no parent) always carry position;
// child nodes only when X or Y is non-zero, to override default stacking.
func shouldSendMindmapPosition(args CreateMindmapNodeArgs) bool {
	if args.ParentID == "" {
		return true
	}
	return args.X != 0 || args.Y != 0
}

// CreateMindmapNode creates a mindmap node on a board.
func (c *Client) CreateMindmapNode(ctx context.Context, args CreateMindmapNodeArgs) (CreateMindmapNodeResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateMindmapNodeResult{}, err
	}
	if args.Content == "" {
		return CreateMindmapNodeResult{}, fmt.Errorf("content is required")
	}
	if err := ValidateContent(args.Content); err != nil {
		return CreateMindmapNodeResult{}, err
	}

	// Build request body with correct nested structure
	// Miro v2-experimental mindmap API uses: data.nodeView.data.content
	nodeViewData := map[string]interface{}{
		"content": args.Content,
	}

	nodeView := map[string]interface{}{
		"data": nodeViewData,
	}

	// Set node view type (text or bubble)
	if args.NodeView != "" {
		nodeView["type"] = args.NodeView
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"nodeView": nodeView,
		},
	}

	// Parent is sent for child nodes; root nodes have no parent.
	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	// Position is sent whenever explicit coordinates are supplied. Root nodes
	// always need it (default 0,0 if neither x nor y is given). For child nodes
	// it's optional: when supplied, it overrides the API's default
	// placement-on-top-of-parent so siblings don't all stack at the same point.
	if shouldSendMindmapPosition(args) {
		reqBody["position"] = map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		}
	}

	respBody, err := c.requestExperimental(ctx, http.MethodPost, "/boards/"+args.BoardID+"/mindmap_nodes", reqBody)
	if err != nil {
		return CreateMindmapNodeResult{}, err
	}

	var node struct {
		ID   string `json:"id"`
		Data struct {
			IsRoot   bool `json:"isRoot"`
			NodeView struct {
				Type string `json:"type"`
				Data struct {
					Content string `json:"content"`
				} `json:"data"`
			} `json:"nodeView"`
		} `json:"data"`
		Parent *struct {
			ID string `json:"id"`
		} `json:"parent"`
	}
	if err := json.Unmarshal(respBody, &node); err != nil {
		return CreateMindmapNodeResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract content from response (may have HTML tags added by API)
	content := node.Data.NodeView.Data.Content
	if content == "" {
		content = args.Content // fallback to input if response content is empty
	}

	result := CreateMindmapNodeResult{
		ID:      node.ID,
		ItemURL: BuildItemURL(args.BoardID, node.ID),
		Content: content,
		Message: fmt.Sprintf("Created mindmap node '%s'", truncateMindmap(args.Content, 30)),
	}
	if node.Parent != nil {
		result.ParentID = node.Parent.ID
	}

	return result, nil
}

// truncateMindmap shortens a string to max length with ellipsis.
// This is a local copy to avoid import cycles.
func truncateMindmap(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// GetMindmapNode retrieves a specific mindmap node.
func (c *Client) GetMindmapNode(ctx context.Context, args GetMindmapNodeArgs) (GetMindmapNodeResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetMindmapNodeResult{}, err
	}
	if args.NodeID == "" {
		return GetMindmapNodeResult{}, fmt.Errorf("node_id is required")
	}

	path := fmt.Sprintf("/boards/%s/mindmap_nodes/%s", args.BoardID, args.NodeID)
	respBody, err := c.requestExperimental(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetMindmapNodeResult{}, err
	}

	var node struct {
		ID       string `json:"id"`
		Position *struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"position"`
		Data struct {
			IsRoot   bool `json:"isRoot"`
			NodeView struct {
				Type string `json:"type"`
				Data struct {
					Content string `json:"content"`
				} `json:"data"`
			} `json:"nodeView"`
		} `json:"data"`
		Parent *struct {
			ID string `json:"id"`
		} `json:"parent"`
		Children []struct {
			ID string `json:"id"`
		} `json:"children"`
		CreatedAt  string `json:"createdAt"`
		ModifiedAt string `json:"modifiedAt"`
	}
	if err := json.Unmarshal(respBody, &node); err != nil {
		return GetMindmapNodeResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	result := GetMindmapNodeResult{
		ID:       node.ID,
		Content:  node.Data.NodeView.Data.Content,
		NodeView: node.Data.NodeView.Type,
		IsRoot:   node.Data.IsRoot,
		Message:  fmt.Sprintf("Retrieved mindmap node '%s'", truncateMindmap(node.Data.NodeView.Data.Content, 30)),
	}

	if node.Position != nil {
		result.X = node.Position.X
		result.Y = node.Position.Y
	}
	if node.Parent != nil {
		result.ParentID = node.Parent.ID
	}
	if len(node.Children) > 0 {
		result.ChildIDs = make([]string, len(node.Children))
		for i, child := range node.Children {
			result.ChildIDs[i] = child.ID
		}
	}
	result.CreatedAt = node.CreatedAt
	result.ModifiedAt = node.ModifiedAt

	return result, nil
}

// ListMindmapNodes retrieves all mindmap nodes on a board.
func (c *Client) ListMindmapNodes(ctx context.Context, args ListMindmapNodesArgs) (ListMindmapNodesResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListMindmapNodesResult{}, err
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	path := fmt.Sprintf("/boards/%s/mindmap_nodes?limit=%d", args.BoardID, limit)
	if args.Cursor != "" {
		path += "&cursor=" + args.Cursor
	}

	respBody, err := c.requestExperimental(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ListMindmapNodesResult{}, err
	}

	var resp struct {
		Data   []json.RawMessage `json:"data"`
		Cursor string            `json:"cursor,omitempty"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ListMindmapNodesResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	nodes := make([]MindmapNodeSummary, 0, len(resp.Data))
	for _, raw := range resp.Data {
		var node struct {
			ID   string `json:"id"`
			Data struct {
				IsRoot   bool `json:"isRoot"`
				NodeView struct {
					Data struct {
						Content string `json:"content"`
					} `json:"data"`
				} `json:"nodeView"`
			} `json:"data"`
			Parent *struct {
				ID string `json:"id"`
			} `json:"parent"`
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			continue
		}

		summary := MindmapNodeSummary{
			ID:      node.ID,
			Content: node.Data.NodeView.Data.Content,
			IsRoot:  node.Data.IsRoot,
		}
		if node.Parent != nil {
			summary.ParentID = node.Parent.ID
		}
		nodes = append(nodes, summary)
	}

	return ListMindmapNodesResult{
		Nodes:   nodes,
		Count:   len(nodes),
		HasMore: resp.Cursor != "",
		Cursor:  resp.Cursor,
		Message: fmt.Sprintf("Found %d mindmap nodes", len(nodes)),
	}, nil
}

// DeleteMindmapNode removes a mindmap node.
func (c *Client) DeleteMindmapNode(ctx context.Context, args DeleteMindmapNodeArgs) (DeleteMindmapNodeResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return DeleteMindmapNodeResult{}, err
	}
	if args.NodeID == "" {
		return DeleteMindmapNodeResult{}, fmt.Errorf("node_id is required")
	}

	// Dry-run mode: return preview without deleting
	if args.DryRun {
		return DeleteMindmapNodeResult{
			Success: true,
			ID:      args.NodeID,
			Message: "[DRY RUN] Would delete mindmap node " + args.NodeID + " from board " + args.BoardID,
		}, nil
	}

	path := fmt.Sprintf("/boards/%s/mindmap_nodes/%s", args.BoardID, args.NodeID)
	_, err := c.requestExperimental(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return DeleteMindmapNodeResult{
			Success: false,
			ID:      args.NodeID,
			Message: fmt.Sprintf("Failed to delete mindmap node: %v", err),
		}, err
	}

	// Invalidate items cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return DeleteMindmapNodeResult{
		Success: true,
		ID:      args.NodeID,
		Message: "Mindmap node deleted successfully",
	}, nil
}
