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

// clampConnectorLimit normalizes a requested page size to the connector
// endpoint's bounds (Miro API minimum 10, maximum 50 — limit=51 and
// limit=100 both answer 400, verified live 18-08-2026).
func clampConnectorLimit(limit int) int {
	if limit <= 0 {
		limit = 50
	}
	if limit < 10 {
		return 10
	}
	if limit > MaxConnectorLimit {
		return MaxConnectorLimit
	}
	return limit
}

// summarizeConnectors projects full connector objects to list summaries.
func summarizeConnectors(data []Connector) []ConnectorSummary {
	connectors := make([]ConnectorSummary, len(data))
	for i, c := range data {
		connectors[i] = ConnectorSummary{
			ID:          c.ID,
			StartItemID: c.StartItem.ItemID,
			EndItemID:   c.EndItem.ItemID,
			Style:       c.Shape,
			Caption:     extractCaption(c.Captions),
		}
	}
	return connectors
}

// ListConnectors returns a list of connectors on a board.
func (c *Client) ListConnectors(ctx context.Context, args ListConnectorsArgs) (ListConnectorsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListConnectorsResult{}, err
	}

	limit := clampConnectorLimit(args.Limit)

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

	connectors := summarizeConnectors(resp.Data)

	return ListConnectorsResult{
		Connectors: connectors,
		Count:      len(connectors),
		HasMore:    resp.Cursor != "",
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

	return connectorDetails(connector), nil
}

// connectorDetails projects a full connector object onto the get result,
// including optional style details and formatted timestamps.
func connectorDetails(connector Connector) GetConnectorResult {
	result := GetConnectorResult{
		ID:          connector.ID,
		StartItemID: connector.StartItem.ItemID,
		EndItemID:   connector.EndItem.ItemID,
		Style:       connector.Shape,
		Caption:     extractCaption(connector.Captions),
		StartCap:    connector.Style.StartStrokeCap,
		EndCap:      connector.Style.EndStrokeCap,
		Color:       connector.Style.Color,
		Message:     "Retrieved connector details",
	}
	result.CreatedAt = formatTimestamp(connector.CreatedAt)
	result.ModifiedAt = formatTimestamp(connector.ModifiedAt)
	return result
}

// formatTimestamp renders a timestamp as RFC3339, or empty when unset.
func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// buildCreateConnectorBody assembles the request body for a connector
// create call, applying the default shape and end cap.
func buildCreateConnectorBody(args CreateConnectorArgs) map[string]interface{} {
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
	reqBody["style"] = connectorStyle

	if args.Caption != "" {
		reqBody["captions"] = []map[string]interface{}{
			{"content": args.Caption},
		}
	}
	return reqBody
}

// CreateConnector creates a connector between two items.
func (c *Client) CreateConnector(ctx context.Context, args CreateConnectorArgs) (CreateConnectorResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateConnectorResult{}, err
	}
	if args.StartItemID == "" || args.EndItemID == "" {
		return CreateConnectorResult{}, fmt.Errorf("start_item_id and end_item_id are required")
	}

	reqBody := buildCreateConnectorBody(args)
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

// buildUpdateConnectorBody assembles the request body for a connector
// update call; an empty map means no updatable field was supplied.
func buildUpdateConnectorBody(args UpdateConnectorArgs) (map[string]interface{}, error) {
	reqBody := make(map[string]interface{})
	if args.Style != "" {
		reqBody["shape"] = args.Style
	}

	connectorStyle, err := buildConnectorStyle(args)
	if err != nil {
		return nil, err
	}
	if len(connectorStyle) > 0 {
		reqBody["style"] = connectorStyle
	}

	if args.Caption != "" {
		reqBody["captions"] = []map[string]interface{}{
			{"content": args.Caption},
		}
	}
	return reqBody, nil
}

// buildConnectorStyle assembles the caps-and-color style block for an update.
func buildConnectorStyle(args UpdateConnectorArgs) (map[string]interface{}, error) {
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
			return nil, fmt.Errorf("color: %w", err)
		}
		connectorStyle["strokeColor"] = strokeColor
	}
	return connectorStyle, nil
}

// UpdateConnector updates an existing connector.
func (c *Client) UpdateConnector(ctx context.Context, args UpdateConnectorArgs) (UpdateConnectorResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateConnectorResult{}, err
	}
	if args.ConnectorID == "" {
		return UpdateConnectorResult{}, fmt.Errorf("connector_id is required")
	}

	reqBody, err := buildUpdateConnectorBody(args)
	if err != nil {
		return UpdateConnectorResult{}, err
	}
	if len(reqBody) == 0 {
		return UpdateConnectorResult{}, fmt.Errorf("at least one update field is required")
	}

	path := fmt.Sprintf("/boards/%s/connectors/%s", args.BoardID, args.ConnectorID)

	if _, err := c.request(ctx, http.MethodPatch, path, reqBody); err != nil {
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
