package miro

// =============================================================================
// Create Image
// =============================================================================

// CreateImageArgs contains parameters for creating an image.
type CreateImageArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"Board ID"`
	URL      string  `json:"url" jsonschema:"URL of the image to add (must be publicly accessible)"`
	Title    string  `json:"title,omitempty" jsonschema:"Image title/alt text"`
	X        float64 `json:"x,omitempty" jsonschema:"X position"`
	Y        float64 `json:"y,omitempty" jsonschema:"Y position"`
	Width    float64 `json:"width,omitempty" jsonschema:"Image width (preserves aspect ratio)"`
	ParentID string  `json:"parent_id,omitempty" jsonschema:"Frame ID to place image in"`
}

// CreateImageResult contains the created image.
type CreateImageResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Message string `json:"message"`
}

// =============================================================================
// Get Image
// =============================================================================

// GetImageArgs contains parameters for retrieving an image item.
type GetImageArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	ItemID  string `json:"item_id" jsonschema:"Image item ID"`
}

// GetImageResult contains the image details.
type GetImageResult struct {
	ID       string  `json:"id"`
	Title    string  `json:"title,omitempty"`
	ImageURL string  `json:"image_url"`
	Width    float64 `json:"width,omitempty"`
	Height   float64 `json:"height,omitempty"`
	X        float64 `json:"x,omitempty"`
	Y        float64 `json:"y,omitempty"`
	ParentID string  `json:"parent_id,omitempty"`
	Message  string  `json:"message"`
}

// =============================================================================
// Update Image
// =============================================================================

// UpdateImageArgs contains parameters for updating an image via dedicated endpoint.
type UpdateImageArgs struct {
	BoardID  string   `json:"board_id" jsonschema:"Board ID"`
	ItemID   string   `json:"item_id" jsonschema:"Image ID to update"`
	Title    *string  `json:"title,omitempty" jsonschema:"New image title/alt text"`
	URL      *string  `json:"url,omitempty" jsonschema:"New image URL"`
	X        *float64 `json:"x,omitempty" jsonschema:"New X position"`
	Y        *float64 `json:"y,omitempty" jsonschema:"New Y position"`
	Width    *float64 `json:"width,omitempty" jsonschema:"New width (preserves aspect ratio)"`
	ParentID *string  `json:"parent_id,omitempty" jsonschema:"Move to frame (empty string removes from frame)"`
}

// UpdateImageResult contains the updated image details.
type UpdateImageResult struct {
	ID      string `json:"id"`
	Title   string `json:"title,omitempty"`
	URL     string `json:"url,omitempty"`
	Message string `json:"message"`
}
