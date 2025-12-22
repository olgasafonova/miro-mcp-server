package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// =============================================================================
// Board Picture (All Plans)
// =============================================================================

// GetBoardPicture retrieves the preview image URL for a board.
// This works for all Miro plans and provides a thumbnail of the board.
func (c *Client) GetBoardPicture(ctx context.Context, args GetBoardPictureArgs) (GetBoardPictureResult, error) {
	if args.BoardID == "" {
		return GetBoardPictureResult{}, fmt.Errorf("board_id is required")
	}

	// Get board details (which includes the picture)
	board, err := c.GetBoard(ctx, GetBoardArgs(args))
	if err != nil {
		return GetBoardPictureResult{}, fmt.Errorf("failed to get board: %w", err)
	}

	imageURL := ""
	if board.Picture != nil && board.Picture.ImageURL != "" {
		imageURL = board.Picture.ImageURL
	}

	if imageURL == "" {
		return GetBoardPictureResult{
			BoardID:  args.BoardID,
			ImageURL: "",
			Message:  "Board has no picture available",
		}, nil
	}

	return GetBoardPictureResult{
		BoardID:  args.BoardID,
		ImageURL: imageURL,
		Message:  "Board picture URL retrieved successfully",
	}, nil
}

// =============================================================================
// Export Jobs (Enterprise Only)
// =============================================================================

// CreateExportJob creates an export job for one or more boards.
// This is an Enterprise-only feature requiring the boards:export scope.
// Up to 50 boards can be exported in a single job.
func (c *Client) CreateExportJob(ctx context.Context, args CreateExportJobArgs) (CreateExportJobResult, error) {
	if args.OrgID == "" {
		return CreateExportJobResult{}, fmt.Errorf("org_id is required (Enterprise feature)")
	}
	if len(args.BoardIDs) == 0 {
		return CreateExportJobResult{}, fmt.Errorf("board_ids is required (at least one board)")
	}
	if len(args.BoardIDs) > 50 {
		return CreateExportJobResult{}, fmt.Errorf("maximum 50 boards per export job")
	}

	// Generate request ID if not provided
	requestID := args.RequestID
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Default format
	format := args.Format
	if format == "" {
		format = "pdf"
	}
	if format != "pdf" && format != "svg" && format != "html" {
		return CreateExportJobResult{}, fmt.Errorf("format must be pdf, svg, or html")
	}

	reqBody := map[string]interface{}{
		"boardIds":  args.BoardIDs,
		"requestId": requestID,
		"format":    format,
	}

	path := fmt.Sprintf("/orgs/%s/boards/export/jobs", args.OrgID)
	respBody, err := c.request(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return CreateExportJobResult{}, fmt.Errorf("failed to create export job (requires Enterprise plan): %w", err)
	}

	var resp struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return CreateExportJobResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CreateExportJobResult{
		JobID:     resp.ID,
		Status:    resp.Status,
		RequestID: resp.RequestID,
		Message:   fmt.Sprintf("Export job created for %d board(s)", len(args.BoardIDs)),
	}, nil
}

// GetExportJobStatus retrieves the status of an export job.
// This is an Enterprise-only feature.
func (c *Client) GetExportJobStatus(ctx context.Context, args GetExportJobStatusArgs) (GetExportJobStatusResult, error) {
	if args.OrgID == "" {
		return GetExportJobStatusResult{}, fmt.Errorf("org_id is required")
	}
	if args.JobID == "" {
		return GetExportJobStatusResult{}, fmt.Errorf("job_id is required")
	}

	path := fmt.Sprintf("/orgs/%s/boards/export/jobs/%s", args.OrgID, args.JobID)
	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetExportJobStatusResult{}, fmt.Errorf("failed to get export job status: %w", err)
	}

	var resp struct {
		ID             string `json:"id"`
		Status         string `json:"status"`
		Progress       int    `json:"progress,omitempty"`
		BoardsTotal    int    `json:"boardsTotal,omitempty"`
		BoardsExported int    `json:"boardsExported,omitempty"`
		CreatedAt      string `json:"createdAt,omitempty"`
		FinishedAt     string `json:"finishedAt,omitempty"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return GetExportJobStatusResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	message := fmt.Sprintf("Export job %s: %s", resp.ID, resp.Status)
	if resp.Status == "in_progress" && resp.BoardsTotal > 0 {
		message = fmt.Sprintf("Export job in progress: %d/%d boards exported", resp.BoardsExported, resp.BoardsTotal)
	}

	return GetExportJobStatusResult{
		JobID:          resp.ID,
		Status:         resp.Status,
		Progress:       resp.Progress,
		BoardsTotal:    resp.BoardsTotal,
		BoardsExported: resp.BoardsExported,
		Message:        message,
	}, nil
}

// GetExportJobResults retrieves the download links for a completed export job.
// This is an Enterprise-only feature.
// Download links are valid for 15 minutes; call again to regenerate if expired.
func (c *Client) GetExportJobResults(ctx context.Context, args GetExportJobResultsArgs) (GetExportJobResultsResult, error) {
	if args.OrgID == "" {
		return GetExportJobResultsResult{}, fmt.Errorf("org_id is required")
	}
	if args.JobID == "" {
		return GetExportJobResultsResult{}, fmt.Errorf("job_id is required")
	}

	path := fmt.Sprintf("/orgs/%s/boards/export/jobs/%s/results", args.OrgID, args.JobID)
	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetExportJobResultsResult{}, fmt.Errorf("failed to get export job results: %w", err)
	}

	var resp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Data   []struct {
			BoardID     string `json:"boardId"`
			BoardName   string `json:"boardName"`
			DownloadURL string `json:"downloadUrl"`
			ExpiresAt   string `json:"expiresAt,omitempty"`
			FileSize    int64  `json:"fileSize,omitempty"`
			Format      string `json:"format,omitempty"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return GetExportJobResultsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.Status != "completed" {
		return GetExportJobResultsResult{
			JobID:   resp.ID,
			Status:  resp.Status,
			Boards:  nil,
			Message: fmt.Sprintf("Export job is %s, results not yet available", resp.Status),
		}, nil
	}

	boards := make([]ExportedBoard, len(resp.Data))
	for i, b := range resp.Data {
		boards[i] = ExportedBoard{
			BoardID:     b.BoardID,
			BoardName:   b.BoardName,
			DownloadURL: b.DownloadURL,
			ExpiresAt:   b.ExpiresAt,
			FileSize:    b.FileSize,
			Format:      b.Format,
		}
	}

	return GetExportJobResultsResult{
		JobID:     resp.ID,
		Status:    resp.Status,
		Boards:    boards,
		ExpiresIn: "15 minutes",
		Message:   fmt.Sprintf("Export completed: %d board(s) ready for download", len(boards)),
	}, nil
}
