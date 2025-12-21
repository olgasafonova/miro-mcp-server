package miro

import "time"

// =============================================================================
// Board Picture (All Plans)
// =============================================================================

// Picture represents a board's preview image.
type Picture struct {
	ID       string `json:"id,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
}

// GetBoardPictureArgs contains parameters for getting a board's picture.
type GetBoardPictureArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID to get picture for"`
}

// GetBoardPictureResult contains the board picture URL.
type GetBoardPictureResult struct {
	BoardID  string `json:"board_id"`
	ImageURL string `json:"image_url"`
	Message  string `json:"message"`
}

// =============================================================================
// Export Job Types (Enterprise Only)
// =============================================================================

// CreateExportJobArgs contains parameters for creating a board export job.
type CreateExportJobArgs struct {
	OrgID     string   `json:"org_id" jsonschema:"required" jsonschema_description:"Organization ID (Enterprise only)"`
	BoardIDs  []string `json:"board_ids" jsonschema:"required" jsonschema_description:"Board IDs to export (max 50)"`
	RequestID string   `json:"request_id,omitempty" jsonschema_description:"Unique request ID for idempotency (auto-generated if empty)"`
	Format    string   `json:"format,omitempty" jsonschema_description:"Export format: pdf, svg, or html (default: pdf)"`
}

// CreateExportJobResult contains the created export job details.
type CreateExportJobResult struct {
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Message   string `json:"message"`
}

// GetExportJobStatusArgs contains parameters for getting export job status.
type GetExportJobStatusArgs struct {
	OrgID string `json:"org_id" jsonschema:"required" jsonschema_description:"Organization ID"`
	JobID string `json:"job_id" jsonschema:"required" jsonschema_description:"Export job ID"`
}

// GetExportJobStatusResult contains the export job status.
type GetExportJobStatusResult struct {
	JobID          string    `json:"job_id"`
	Status         string    `json:"status"` // "in_progress", "completed", "failed"
	Progress       int       `json:"progress,omitempty"`
	BoardsTotal    int       `json:"boards_total,omitempty"`
	BoardsExported int       `json:"boards_exported,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	FinishedAt     time.Time `json:"finished_at,omitempty"`
	Message        string    `json:"message"`
}

// GetExportJobResultsArgs contains parameters for getting export job results.
type GetExportJobResultsArgs struct {
	OrgID string `json:"org_id" jsonschema:"required" jsonschema_description:"Organization ID"`
	JobID string `json:"job_id" jsonschema:"required" jsonschema_description:"Export job ID"`
}

// ExportedBoard contains export data for a single board.
type ExportedBoard struct {
	BoardID     string `json:"board_id"`
	BoardName   string `json:"board_name"`
	DownloadURL string `json:"download_url"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	FileSize    int64  `json:"file_size,omitempty"`
	Format      string `json:"format,omitempty"`
}

// GetExportJobResultsResult contains the export job results with download links.
type GetExportJobResultsResult struct {
	JobID     string          `json:"job_id"`
	Status    string          `json:"status"`
	Boards    []ExportedBoard `json:"boards"`
	ExpiresIn string          `json:"expires_in,omitempty"` // e.g., "15 minutes"
	Message   string          `json:"message"`
}
