package miro

// =============================================================================
// App Card Types
// =============================================================================
// App cards are special cards with custom fields and external app integration.

// AppCardField represents a custom field on an app card.
type AppCardField struct {
	Value     string `json:"value,omitempty"`
	FillColor string `json:"fillColor,omitempty"`
	TextColor string `json:"textColor,omitempty"`
	IconShape string `json:"iconShape,omitempty"` // round, square
	IconURL   string `json:"iconUrl,omitempty"`
}

// CreateAppCardArgs contains parameters for creating an app card.
type CreateAppCardArgs struct {
	BoardID     string         `json:"board_id" jsonschema:"Board ID"`
	Title       string         `json:"title" jsonschema:"App card title"`
	Description string         `json:"description,omitempty" jsonschema:"App card description"`
	Status      string         `json:"status,omitempty" jsonschema:"Status indicator: connected, disconnected, disabled"`
	Fields      []AppCardField `json:"fields,omitempty" jsonschema:"Custom fields (max 5)"`
	X           float64        `json:"x,omitempty" jsonschema:"X position"`
	Y           float64        `json:"y,omitempty" jsonschema:"Y position"`
	Width       float64        `json:"width,omitempty" jsonschema:"Card width (default 320)"`
	ParentID    string         `json:"parent_id,omitempty" jsonschema:"Frame ID to place card in"`
}

// CreateAppCardResult contains the created app card.
type CreateAppCardResult struct {
	ID          string `json:"id"`
	ItemURL     string `json:"item_url,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

// GetAppCardArgs contains parameters for getting an app card.
type GetAppCardArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"App card item ID"`
}

// GetAppCardResult contains the app card details.
type GetAppCardResult struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      string         `json:"status"`
	Fields      []AppCardField `json:"fields,omitempty"`
	Position    *Position      `json:"position,omitempty"`
	Geometry    *Geometry      `json:"geometry,omitempty"`
	CreatedAt   string         `json:"created_at,omitempty"`
	ModifiedAt  string         `json:"modified_at,omitempty"`
	Message     string         `json:"message"`
}

// UpdateAppCardArgs contains parameters for updating an app card.
type UpdateAppCardArgs struct {
	BoardID     string         `json:"board_id" jsonschema:"Board ID"`
	ItemID      string         `json:"item_id" jsonschema:"App card item ID"`
	Title       string         `json:"title,omitempty" jsonschema:"New title"`
	Description string         `json:"description,omitempty" jsonschema:"New description"`
	Status      string         `json:"status,omitempty" jsonschema:"Status: connected, disconnected, disabled"`
	Fields      []AppCardField `json:"fields,omitempty" jsonschema:"Updated custom fields (max 5)"`
	X           *float64       `json:"x,omitempty" jsonschema:"New X position"`
	Y           *float64       `json:"y,omitempty" jsonschema:"New Y position"`
	Width       *float64       `json:"width,omitempty" jsonschema:"New width"`
}

// UpdateAppCardResult contains the update result.
type UpdateAppCardResult struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// DeleteAppCardArgs contains parameters for deleting an app card.
type DeleteAppCardArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"App card item ID to delete"`
	DryRun  bool   `json:"dry_run,omitempty" jsonschema:"If true, returns preview without deleting"`
}

// DeleteAppCardResult contains the deletion result.
type DeleteAppCardResult struct {
	Success bool   `json:"success"`
	ItemID  string `json:"item_id"`
	Message string `json:"message"`
}
