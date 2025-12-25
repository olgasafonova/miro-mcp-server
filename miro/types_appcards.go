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
	BoardID     string         `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Title       string         `json:"title" jsonschema:"required" jsonschema_description:"App card title"`
	Description string         `json:"description,omitempty" jsonschema_description:"App card description"`
	Status      string         `json:"status,omitempty" jsonschema_description:"Status indicator: connected, disconnected, disabled"`
	Fields      []AppCardField `json:"fields,omitempty" jsonschema_description:"Custom fields (max 5)"`
	X           float64        `json:"x,omitempty" jsonschema_description:"X position"`
	Y           float64        `json:"y,omitempty" jsonschema_description:"Y position"`
	Width       float64        `json:"width,omitempty" jsonschema_description:"Card width (default 320)"`
	ParentID    string         `json:"parent_id,omitempty" jsonschema_description:"Frame ID to place card in"`
}

// CreateAppCardResult contains the created app card.
type CreateAppCardResult struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

// GetAppCardArgs contains parameters for getting an app card.
type GetAppCardArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"required" jsonschema_description:"App card item ID"`
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
	BoardID     string         `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID      string         `json:"item_id" jsonschema:"required" jsonschema_description:"App card item ID"`
	Title       string         `json:"title,omitempty" jsonschema_description:"New title"`
	Description string         `json:"description,omitempty" jsonschema_description:"New description"`
	Status      string         `json:"status,omitempty" jsonschema_description:"Status: connected, disconnected, disabled"`
	Fields      []AppCardField `json:"fields,omitempty" jsonschema_description:"Updated custom fields (max 5)"`
	X           *float64       `json:"x,omitempty" jsonschema_description:"New X position"`
	Y           *float64       `json:"y,omitempty" jsonschema_description:"New Y position"`
	Width       *float64       `json:"width,omitempty" jsonschema_description:"New width"`
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
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"required" jsonschema_description:"App card item ID to delete"`
	DryRun  bool   `json:"dry_run,omitempty" jsonschema_description:"If true, returns preview without deleting"`
}

// DeleteAppCardResult contains the deletion result.
type DeleteAppCardResult struct {
	Success bool   `json:"success"`
	ItemID  string `json:"item_id"`
	Message string `json:"message"`
}
