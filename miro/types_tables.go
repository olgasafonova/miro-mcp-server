package miro

// =============================================================================
// Table Types (data_table_format)
// =============================================================================

// ListTablesArgs contains parameters for listing tables on a board.
type ListTablesArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Max tables to return (default 10, max 50)"`
	Cursor  string `json:"cursor,omitempty" jsonschema:"Pagination cursor from previous response"`
}

// ListTablesResult contains the tables found on a board.
type ListTablesResult struct {
	Tables  []TableItem `json:"tables"`
	Count   int         `json:"count"`
	Total   int         `json:"total"`
	Cursor  string      `json:"cursor,omitempty"`
	Message string      `json:"message"`
}

// TableItem represents a table on a Miro board.
type TableItem struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width,omitempty"`
	Height     float64 `json:"height,omitempty"`
	CreatedAt  string  `json:"created_at,omitempty"`
	ModifiedAt string  `json:"modified_at,omitempty"`
	CreatedBy  string  `json:"created_by,omitempty"`
	ModifiedBy string  `json:"modified_by,omitempty"`
	ItemURL    string  `json:"item_url,omitempty"`
}

// GetTableArgs contains parameters for getting a specific table's metadata.
type GetTableArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"Table item ID"`
}

// GetTableResult contains the table metadata.
type GetTableResult struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width,omitempty"`
	Height     float64 `json:"height,omitempty"`
	CreatedAt  string  `json:"created_at,omitempty"`
	ModifiedAt string  `json:"modified_at,omitempty"`
	CreatedBy  string  `json:"created_by,omitempty"`
	ModifiedBy string  `json:"modified_by,omitempty"`
	ParentID   string  `json:"parent_id,omitempty"`
	ItemURL    string  `json:"item_url,omitempty"`
	Message    string  `json:"message"`
}
