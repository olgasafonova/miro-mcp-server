package miro

// =============================================================================
// List Connectors
// =============================================================================

// ListConnectorsArgs contains parameters for listing connectors on a board.
type ListConnectorsArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Max connectors to return (default 50, max 100)"`
	Cursor  string `json:"cursor,omitempty" jsonschema:"Pagination cursor"`
}

// ConnectorSummary represents a connector in list results.
type ConnectorSummary struct {
	ID          string `json:"id"`
	StartItemID string `json:"start_item_id"`
	EndItemID   string `json:"end_item_id"`
	Style       string `json:"style,omitempty"`
	Caption     string `json:"caption,omitempty"`
}

// ListConnectorsResult contains the list of connectors.
type ListConnectorsResult struct {
	Connectors []ConnectorSummary `json:"connectors"`
	Count      int                `json:"count"`
	HasMore    bool               `json:"has_more"`
	Cursor     string             `json:"cursor,omitempty"`
	Message    string             `json:"message"`
}

// =============================================================================
// Get Connector
// =============================================================================

// GetConnectorArgs contains parameters for getting a specific connector.
type GetConnectorArgs struct {
	BoardID     string `json:"board_id" jsonschema:"Board ID"`
	ConnectorID string `json:"connector_id" jsonschema:"Connector ID to retrieve"`
}

// GetConnectorResult contains the full connector details.
type GetConnectorResult struct {
	ID          string `json:"id"`
	StartItemID string `json:"start_item_id"`
	EndItemID   string `json:"end_item_id"`
	Style       string `json:"style,omitempty"`
	StartCap    string `json:"start_cap,omitempty"`
	EndCap      string `json:"end_cap,omitempty"`
	Caption     string `json:"caption,omitempty"`
	Color       string `json:"color,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	ModifiedAt  string `json:"modified_at,omitempty"`
	CreatedBy   string `json:"created_by,omitempty"`
	ModifiedBy  string `json:"modified_by,omitempty"`
	Message     string `json:"message"`
}

// =============================================================================
// Create Connector
// =============================================================================

// CreateConnectorArgs contains parameters for creating a connector.
type CreateConnectorArgs struct {
	BoardID     string `json:"board_id" jsonschema:"Board ID"`
	StartItemID string `json:"start_item_id" jsonschema:"ID of the item to connect from"`
	EndItemID   string `json:"end_item_id" jsonschema:"ID of the item to connect to"`
	Style       string `json:"style,omitempty" jsonschema:"Connector style: straight, elbowed, curved (default elbowed)"`
	StartCap    string `json:"start_cap,omitempty" jsonschema:"Start arrow: none, arrow, filled_arrow, diamond, etc."`
	EndCap      string `json:"end_cap,omitempty" jsonschema:"End arrow: none, arrow, filled_arrow, diamond, etc."`
	Caption     string `json:"caption,omitempty" jsonschema:"Text label on the connector"`
}

// CreateConnectorResult contains the created connector.
type CreateConnectorResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Message string `json:"message"`
}

// =============================================================================
// Update Connector
// =============================================================================

// UpdateConnectorArgs contains parameters for updating a connector.
type UpdateConnectorArgs struct {
	BoardID     string `json:"board_id" jsonschema:"Board ID"`
	ConnectorID string `json:"connector_id" jsonschema:"ID of the connector to update"`
	Style       string `json:"style,omitempty" jsonschema:"Connector style: straight, elbowed, curved"`
	StartCap    string `json:"start_cap,omitempty" jsonschema:"Start arrow: none, arrow, filled_arrow, diamond, etc."`
	EndCap      string `json:"end_cap,omitempty" jsonschema:"End arrow: none, arrow, filled_arrow, diamond, etc."`
	Caption     string `json:"caption,omitempty" jsonschema:"Text label on the connector"`
	Color       string `json:"color,omitempty" jsonschema:"Connector line color: 6-char hex like #1a1a1a or named (red, orange, yellow, green, blue, purple, pink, gray, white, black)"`
}

// UpdateConnectorResult confirms connector update.
type UpdateConnectorResult struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

// =============================================================================
// Delete Connector
// =============================================================================

// DeleteConnectorArgs contains parameters for deleting a connector.
type DeleteConnectorArgs struct {
	BoardID     string `json:"board_id" jsonschema:"Board ID"`
	ConnectorID string `json:"connector_id" jsonschema:"ID of the connector to delete"`
	DryRun      bool   `json:"dry_run,omitempty" jsonschema:"If true, returns preview without deleting"`
}

// DeleteConnectorResult confirms connector deletion.
type DeleteConnectorResult struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Message string `json:"message"`
}
