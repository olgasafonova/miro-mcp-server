package miro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// UploadImage uploads a local image file to a Miro board.
func (c *Client) UploadImage(ctx context.Context, args UploadImageArgs) (UploadImageResult, error) {
	if args.BoardID == "" {
		return UploadImageResult{}, fmt.Errorf("board_id is required")
	}
	if args.FilePath == "" {
		return UploadImageResult{}, fmt.Errorf("file_path is required")
	}

	// Validate file exists and is readable
	fileInfo, err := os.Stat(args.FilePath)
	if err != nil {
		return UploadImageResult{}, fmt.Errorf("cannot access file: %w", err)
	}
	if fileInfo.IsDir() {
		return UploadImageResult{}, fmt.Errorf("file_path is a directory, not a file")
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(args.FilePath))
	validExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".svg": true}
	if !validExts[ext] {
		return UploadImageResult{}, fmt.Errorf("unsupported image format %q (supported: png, jpg, jpeg, gif, webp, svg)", ext)
	}

	// Open file
	file, err := os.Open(args.FilePath)
	if err != nil {
		return UploadImageResult{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Build multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add JSON data part
	dataJSON := map[string]interface{}{}
	if args.Title != "" {
		dataJSON["title"] = args.Title
	}
	if args.X != 0 || args.Y != 0 {
		dataJSON["position"] = map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		}
	}
	if args.ParentID != "" {
		dataJSON["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	dataBytes, err := json.Marshal(dataJSON)
	if err != nil {
		return UploadImageResult{}, fmt.Errorf("failed to marshal data: %w", err)
	}

	dataPart, err := writer.CreateFormField("data")
	if err != nil {
		return UploadImageResult{}, fmt.Errorf("failed to create data field: %w", err)
	}
	if _, err := dataPart.Write(dataBytes); err != nil {
		return UploadImageResult{}, fmt.Errorf("failed to write data: %w", err)
	}

	// Add file resource part
	resourcePart, err := writer.CreateFormFile("resource", filepath.Base(args.FilePath))
	if err != nil {
		return UploadImageResult{}, fmt.Errorf("failed to create resource field: %w", err)
	}
	if _, err := io.Copy(resourcePart, file); err != nil {
		return UploadImageResult{}, fmt.Errorf("failed to write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return UploadImageResult{}, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Make the request
	respBody, err := c.requestMultipart(ctx, http.MethodPost,
		"/boards/"+args.BoardID+"/images",
		writer.FormDataContentType(), &body)
	if err != nil {
		return UploadImageResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UploadImageResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate items list cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	title := resp.Data.Title
	if title == "" {
		title = filepath.Base(args.FilePath)
	}

	return UploadImageResult{
		ID:      resp.ID,
		ItemURL: BuildItemURL(args.BoardID, resp.ID),
		Title:   title,
		Message: fmt.Sprintf("Uploaded image '%s'", title),
	}, nil
}
