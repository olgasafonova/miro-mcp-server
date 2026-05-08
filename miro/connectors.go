package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// =============================================================================
// Connector Operations - List, Get, Create, Update, Delete
// =============================================================================

// ListConnectors returns a list of connectors on a board.
func (c *Client) ListConnectors(ctx context.Context, args ListConnectorsArgs) (ListConnectorsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListConnectorsResult{}, err
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit < 10 {
		limit = 10 // Miro API minimum for connectors
	}
	if limit > 100 {
		limit = 100
	}

	path := fmt.Sprintf("/boards/%s/connectors?limit=%d", args.BoardID, limit)
	if args.Cursor != "" {
		path += "&cursor=" + args.Cursor
	}

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ListConnectorsResult{}, err
	}

	var resp struct {
		Data   []Connector `json:"data"`
		Cursor string      `json:"cursor,omitempty"`
		Total  int         `json:"total,omitempty"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ListConnectorsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	connectors := make([]ConnectorSummary, len(resp.Data))
	for i, c := range resp.Data {
		connectors[i] = ConnectorSummary{
			ID:          c.ID,
			StartItemID: c.StartItem.ItemID,
			EndItemID:   c.EndItem.ItemID,
			Style:       c.Shape,
			Caption:     extractCaption(c.Captions),
		}
	}

	hasMore := resp.Cursor != ""
	return ListConnectorsResult{
		Connectors: connectors,
		Count:      len(connectors),
		HasMore:    hasMore,
		Cursor:     resp.Cursor,
		Message:    fmt.Sprintf("Found %d connectors", len(connectors)),
	}, nil
}

// extractCaption gets the text from the first caption if present.
func extractCaption(captions []Caption) string {
	if len(captions) > 0 {
		return captions[0].Content
	}
	return ""
}

// GetConnector retrieves a specific connector by ID.
func (c *Client) GetConnector(ctx context.Context, args GetConnectorArgs) (GetConnectorResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetConnectorResult{}, err
	}
	if args.ConnectorID == "" {
		return GetConnectorResult{}, fmt.Errorf("connector_id is required")
	}

	path := fmt.Sprintf("/boards/%s/connectors/%s", args.BoardID, args.ConnectorID)
	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetConnectorResult{}, err
	}

	var connector Connector
	if err := json.Unmarshal(respBody, &connector); err != nil {
		return GetConnectorResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	result := GetConnectorResult{
		ID:          connector.ID,
		StartItemID: connector.StartItem.ItemID,
		EndItemID:   connector.EndItem.ItemID,
		Style:       connector.Shape,
		Caption:     extractCaption(connector.Captions),
		Message:     "Retrieved connector details",
	}

	// Extract style details if present
	if connector.Style.StartStrokeCap != "" {
		result.StartCap = connector.Style.StartStrokeCap
	}
	if connector.Style.EndStrokeCap != "" {
		result.EndCap = connector.Style.EndStrokeCap
	}
	if connector.Style.Color != "" {
		result.Color = connector.Style.Color
	}

	// Format timestamps
	if !connector.CreatedAt.IsZero() {
		result.CreatedAt = connector.CreatedAt.Format(time.RFC3339)
	}
	if !connector.ModifiedAt.IsZero() {
		result.ModifiedAt = connector.ModifiedAt.Format(time.RFC3339)
	}

	return result, nil
}

// CreateConnector creates a connector between two items.
func (c *Client) CreateConnector(ctx context.Context, args CreateConnectorArgs) (CreateConnectorResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateConnectorResult{}, err
	}
	if args.StartItemID == "" || args.EndItemID == "" {
		return CreateConnectorResult{}, fmt.Errorf("start_item_id and end_item_id are required")
	}

	// Default style
	style := args.Style
	if style == "" {
		style = "elbowed"
	}

	reqBody := map[string]interface{}{
		"startItem": map[string]interface{}{
			"id": args.StartItemID,
		},
		"endItem": map[string]interface{}{
			"id": args.EndItemID,
		},
		"shape": style,
	}

	connectorStyle := make(map[string]interface{})
	if args.StartCap != "" {
		connectorStyle["startStrokeCap"] = args.StartCap
	}
	if args.EndCap != "" {
		connectorStyle["endStrokeCap"] = args.EndCap
	} else {
		connectorStyle["endStrokeCap"] = "arrow" // Default arrow at end
	}
	if len(connectorStyle) > 0 {
		reqBody["style"] = connectorStyle
	}

	if args.Caption != "" {
		reqBody["captions"] = []map[string]interface{}{
			{"content": args.Caption},
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/connectors", reqBody)
	if err != nil {
		return CreateConnectorResult{}, err
	}

	var connector Connector
	if err := json.Unmarshal(respBody, &connector); err != nil {
		return CreateConnectorResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate connectors cache
	c.cache.InvalidatePrefix("connectors:" + args.BoardID)

	return CreateConnectorResult{
		ID:      connector.ID,
		ItemURL: BuildItemURL(args.BoardID, connector.ID),
		Message: "Created connector between items",
	}, nil
}

// UpdateConnector updates an existing connector.
func (c *Client) UpdateConnector(ctx context.Context, args UpdateConnectorArgs) (UpdateConnectorResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateConnectorResult{}, err
	}
	if args.ConnectorID == "" {
		return UpdateConnectorResult{}, fmt.Errorf("connector_id is required")
	}

	reqBody := make(map[string]interface{})

	// Set shape (style) if provided
	if args.Style != "" {
		reqBody["shape"] = args.Style
	}

	// Build style object for caps and color
	connectorStyle := make(map[string]interface{})
	if args.StartCap != "" {
		connectorStyle["startStrokeCap"] = args.StartCap
	}
	if args.EndCap != "" {
		connectorStyle["endStrokeCap"] = args.EndCap
	}
	if args.Color != "" {
		strokeColor, err := normalizeColor(args.Color)
		if err != nil {
			return UpdateConnectorResult{}, fmt.Errorf("color: %w", err)
		}
		connectorStyle["strokeColor"] = strokeColor
	}
	if len(connectorStyle) > 0 {
		reqBody["style"] = connectorStyle
	}

	// Set caption if provided
	if args.Caption != "" {
		reqBody["captions"] = []map[string]interface{}{
			{"content": args.Caption},
		}
	}

	// If nothing to update, return error
	if len(reqBody) == 0 {
		return UpdateConnectorResult{}, fmt.Errorf("at least one update field is required")
	}

	path := fmt.Sprintf("/boards/%s/connectors/%s", args.BoardID, args.ConnectorID)

	_, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateConnectorResult{}, err
	}

	// Invalidate connectors cache
	c.cache.InvalidatePrefix("connectors:" + args.BoardID)

	return UpdateConnectorResult{
		Success: true,
		ID:      args.ConnectorID,
		Message: "Connector updated successfully",
	}, nil
}

// DeleteConnector removes a connector from a board.
func (c *Client) DeleteConnector(ctx context.Context, args DeleteConnectorArgs) (DeleteConnectorResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return DeleteConnectorResult{}, err
	}
	if args.ConnectorID == "" {
		return DeleteConnectorResult{}, fmt.Errorf("connector_id is required")
	}

	// Dry-run mode: return preview without deleting
	if args.DryRun {
		return DeleteConnectorResult{
			Success: true,
			ID:      args.ConnectorID,
			Message: "[DRY RUN] Would delete connector " + args.ConnectorID + " from board " + args.BoardID,
		}, nil
	}

	path := fmt.Sprintf("/boards/%s/connectors/%s", args.BoardID, args.ConnectorID)

	_, err := c.request(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return DeleteConnectorResult{
			Success: false,
			ID:      args.ConnectorID,
			Message: fmt.Sprintf("Failed to delete connector: %v", err),
		}, err
	}

	// Invalidate connectors cache
	c.cache.InvalidatePrefix("connectors:" + args.BoardID)

	return DeleteConnectorResult{
		Success: true,
		ID:      args.ConnectorID,
		Message: "Connector deleted successfully",
	}, nil
}
