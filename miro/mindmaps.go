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

	// Build request body
	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"content": args.Content,
		},
	}

	// Set node view style
	if args.NodeView != "" {
		data := reqBody["data"].(map[string]interface{})
		data["nodeView"] = args.NodeView
	}

	// If parent is specified, this is a child node
	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	} else {
		// Root node - set position
		reqBody["position"] = map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/mind_map_nodes", reqBody)
	if err != nil {
		return CreateMindmapNodeResult{}, err
	}

	var node struct {
		ID   string `json:"id"`
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
		Parent *struct {
			ID string `json:"id"`
		} `json:"parent"`
	}
	if err := json.Unmarshal(respBody, &node); err != nil {
		return CreateMindmapNodeResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	result := CreateMindmapNodeResult{
		ID:      node.ID,
		Content: node.Data.Content,
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
