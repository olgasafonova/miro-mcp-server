package miro

// =============================================================================
// Create Embed
// =============================================================================

// CreateEmbedArgs contains parameters for creating an embed.
type CreateEmbedArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"Board ID"`
	URL      string  `json:"url" jsonschema:"URL to embed (YouTube, Vimeo, Figma, Google Docs, etc.)"`
	Mode     string  `json:"mode,omitempty" jsonschema:"Display mode: inline (default) or modal"`
	X        float64 `json:"x,omitempty" jsonschema:"X position"`
	Y        float64 `json:"y,omitempty" jsonschema:"Y position"`
	Width    float64 `json:"width,omitempty" jsonschema:"Embed width (default 400)"`
	Height   float64 `json:"height,omitempty" jsonschema:"Embed height (default 300)"`
	ParentID string  `json:"parent_id,omitempty" jsonschema:"Frame ID to place embed in"`
}

// CreateEmbedResult contains the created embed.
type CreateEmbedResult struct {
	ID       string `json:"id"`
	ItemURL  string `json:"item_url,omitempty"`
	URL      string `json:"url"`
	Provider string `json:"provider,omitempty"`
	Message  string `json:"message"`
}

// =============================================================================
// Update Embed
// =============================================================================

// UpdateEmbedArgs contains parameters for updating an embed via dedicated endpoint.
type UpdateEmbedArgs struct {
	BoardID  string   `json:"board_id" jsonschema:"Board ID"`
	ItemID   string   `json:"item_id" jsonschema:"Embed ID to update"`
	URL      *string  `json:"url,omitempty" jsonschema:"New embed URL"`
	Mode     *string  `json:"mode,omitempty" jsonschema:"Display mode: inline or modal"`
	X        *float64 `json:"x,omitempty" jsonschema:"New X position"`
	Y        *float64 `json:"y,omitempty" jsonschema:"New Y position"`
	Width    *float64 `json:"width,omitempty" jsonschema:"New embed width"`
	Height   *float64 `json:"height,omitempty" jsonschema:"New embed height"`
	ParentID *string  `json:"parent_id,omitempty" jsonschema:"Move to frame (empty string removes from frame)"`
}

// UpdateEmbedResult contains the updated embed details.
type UpdateEmbedResult struct {
	ID       string `json:"id"`
	URL      string `json:"url,omitempty"`
	Provider string `json:"provider,omitempty"`
	Message  string `json:"message"`
}
