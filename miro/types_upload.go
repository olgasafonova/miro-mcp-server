package miro

// =============================================================================
// Upload Image (File Upload)
// =============================================================================

// UploadImageArgs contains parameters for uploading a local image file.
type UploadImageArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"Board ID"`
	FilePath string  `json:"file_path" jsonschema:"Absolute path to the image file on disk"`
	Title    string  `json:"title,omitempty" jsonschema:"Image title/alt text"`
	X        float64 `json:"x,omitempty" jsonschema:"X position"`
	Y        float64 `json:"y,omitempty" jsonschema:"Y position"`
	ParentID string  `json:"parent_id,omitempty" jsonschema:"Frame ID to place image in"`
}

// UploadImageResult contains the uploaded image details.
type UploadImageResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Title   string `json:"title,omitempty"`
	Message string `json:"message"`
}

// =============================================================================
// Upload Document (File Upload)
// =============================================================================

// UploadDocumentArgs contains parameters for uploading a local document file.
type UploadDocumentArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"Board ID"`
	FilePath string  `json:"file_path" jsonschema:"Absolute path to the document file on disk"`
	Title    string  `json:"title,omitempty" jsonschema:"Document title"`
	X        float64 `json:"x,omitempty" jsonschema:"X position"`
	Y        float64 `json:"y,omitempty" jsonschema:"Y position"`
	ParentID string  `json:"parent_id,omitempty" jsonschema:"Frame ID to place document in"`
}

// UploadDocumentResult contains the uploaded document details.
type UploadDocumentResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Title   string `json:"title,omitempty"`
	Message string `json:"message"`
}

// =============================================================================
// Update Image from File (PATCH multipart)
// =============================================================================

// UpdateImageFromFileArgs contains parameters for replacing the file on an existing image item.
type UpdateImageFromFileArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"Board ID"`
	ItemID   string  `json:"item_id" jsonschema:"Image item ID to update"`
	FilePath string  `json:"file_path" jsonschema:"Absolute path to the new image file on disk"`
	Title    string  `json:"title,omitempty" jsonschema:"New image title/alt text"`
	X        float64 `json:"x,omitempty" jsonschema:"New X position"`
	Y        float64 `json:"y,omitempty" jsonschema:"New Y position"`
	ParentID string  `json:"parent_id,omitempty" jsonschema:"Frame ID to move image into"`
}

// UpdateImageFromFileResult contains the updated image details.
type UpdateImageFromFileResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Title   string `json:"title,omitempty"`
	Message string `json:"message"`
}

// =============================================================================
// Update Document from File (PATCH multipart)
// =============================================================================

// UpdateDocumentFromFileArgs contains parameters for replacing the file on an existing document item.
type UpdateDocumentFromFileArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"Board ID"`
	ItemID   string  `json:"item_id" jsonschema:"Document item ID to update"`
	FilePath string  `json:"file_path" jsonschema:"Absolute path to the new document file on disk"`
	Title    string  `json:"title,omitempty" jsonschema:"New document title"`
	X        float64 `json:"x,omitempty" jsonschema:"New X position"`
	Y        float64 `json:"y,omitempty" jsonschema:"New Y position"`
	ParentID string  `json:"parent_id,omitempty" jsonschema:"Frame ID to move document into"`
}

// UpdateDocumentFromFileResult contains the updated document details.
type UpdateDocumentFromFileResult struct {
	ID      string `json:"id"`
	ItemURL string `json:"item_url,omitempty"`
	Title   string `json:"title,omitempty"`
	Message string `json:"message"`
}
