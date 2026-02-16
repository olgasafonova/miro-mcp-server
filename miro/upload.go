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

	body, contentType, err := buildMultipartBody(file, filepath.Base(args.FilePath), args.Title, args.X, args.Y, args.ParentID)
	if err != nil {
		return UploadImageResult{}, err
	}

	// Make the request
	respBody, err := c.requestMultipart(ctx, http.MethodPost,
		"/boards/"+args.BoardID+"/images",
		contentType, body)
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

// UploadDocument uploads a local document file to a Miro board.
func (c *Client) UploadDocument(ctx context.Context, args UploadDocumentArgs) (UploadDocumentResult, error) {
	if args.BoardID == "" {
		return UploadDocumentResult{}, fmt.Errorf("board_id is required")
	}
	if args.FilePath == "" {
		return UploadDocumentResult{}, fmt.Errorf("file_path is required")
	}

	// Validate file exists and is readable
	fileInfo, err := os.Stat(args.FilePath)
	if err != nil {
		return UploadDocumentResult{}, fmt.Errorf("cannot access file: %w", err)
	}
	if fileInfo.IsDir() {
		return UploadDocumentResult{}, fmt.Errorf("file_path is a directory, not a file")
	}

	// Validate file size (max 6 MB per Miro API)
	const maxSize = 6 * 1024 * 1024
	if fileInfo.Size() > maxSize {
		return UploadDocumentResult{}, fmt.Errorf("file size %d bytes exceeds 6 MB limit", fileInfo.Size())
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(args.FilePath))
	validExts := map[string]bool{".pdf": true, ".doc": true, ".docx": true, ".ppt": true, ".pptx": true, ".xls": true, ".xlsx": true, ".txt": true, ".rtf": true, ".csv": true}
	if !validExts[ext] {
		return UploadDocumentResult{}, fmt.Errorf("unsupported document format %q (supported: pdf, doc, docx, ppt, pptx, xls, xlsx, txt, rtf, csv)", ext)
	}

	// Open file
	file, err := os.Open(args.FilePath)
	if err != nil {
		return UploadDocumentResult{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	body, contentType, err := buildMultipartBody(file, filepath.Base(args.FilePath), args.Title, args.X, args.Y, args.ParentID)
	if err != nil {
		return UploadDocumentResult{}, err
	}

	// Make the request
	respBody, err := c.requestMultipart(ctx, http.MethodPost,
		"/boards/"+args.BoardID+"/documents",
		contentType, body)
	if err != nil {
		return UploadDocumentResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UploadDocumentResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate items list cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	title := resp.Data.Title
	if title == "" {
		title = filepath.Base(args.FilePath)
	}

	return UploadDocumentResult{
		ID:      resp.ID,
		ItemURL: BuildItemURL(args.BoardID, resp.ID),
		Title:   title,
		Message: fmt.Sprintf("Uploaded document '%s'", title),
	}, nil
}

// UpdateImageFromFile replaces the file on an existing image item via PATCH multipart.
func (c *Client) UpdateImageFromFile(ctx context.Context, args UpdateImageFromFileArgs) (UpdateImageFromFileResult, error) {
	if args.BoardID == "" {
		return UpdateImageFromFileResult{}, fmt.Errorf("board_id is required")
	}
	if args.ItemID == "" {
		return UpdateImageFromFileResult{}, fmt.Errorf("item_id is required")
	}
	if args.FilePath == "" {
		return UpdateImageFromFileResult{}, fmt.Errorf("file_path is required")
	}

	fileInfo, err := os.Stat(args.FilePath)
	if err != nil {
		return UpdateImageFromFileResult{}, fmt.Errorf("cannot access file: %w", err)
	}
	if fileInfo.IsDir() {
		return UpdateImageFromFileResult{}, fmt.Errorf("file_path is a directory, not a file")
	}

	ext := strings.ToLower(filepath.Ext(args.FilePath))
	validExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".svg": true}
	if !validExts[ext] {
		return UpdateImageFromFileResult{}, fmt.Errorf("unsupported image format %q (supported: png, jpg, jpeg, gif, webp, svg)", ext)
	}

	file, err := os.Open(args.FilePath)
	if err != nil {
		return UpdateImageFromFileResult{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	body, contentType, err := buildMultipartBody(file, filepath.Base(args.FilePath), args.Title, args.X, args.Y, args.ParentID)
	if err != nil {
		return UpdateImageFromFileResult{}, err
	}

	respBody, err := c.requestMultipart(ctx, http.MethodPatch,
		"/boards/"+args.BoardID+"/images/"+args.ItemID,
		contentType, body)
	if err != nil {
		return UpdateImageFromFileResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateImageFromFileResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidatePrefix("items:" + args.BoardID)

	title := resp.Data.Title
	if title == "" {
		title = filepath.Base(args.FilePath)
	}

	return UpdateImageFromFileResult{
		ID:      resp.ID,
		ItemURL: BuildItemURL(args.BoardID, resp.ID),
		Title:   title,
		Message: fmt.Sprintf("Updated image '%s' with new file", title),
	}, nil
}

// UpdateDocumentFromFile replaces the file on an existing document item via PATCH multipart.
func (c *Client) UpdateDocumentFromFile(ctx context.Context, args UpdateDocumentFromFileArgs) (UpdateDocumentFromFileResult, error) {
	if args.BoardID == "" {
		return UpdateDocumentFromFileResult{}, fmt.Errorf("board_id is required")
	}
	if args.ItemID == "" {
		return UpdateDocumentFromFileResult{}, fmt.Errorf("item_id is required")
	}
	if args.FilePath == "" {
		return UpdateDocumentFromFileResult{}, fmt.Errorf("file_path is required")
	}

	fileInfo, err := os.Stat(args.FilePath)
	if err != nil {
		return UpdateDocumentFromFileResult{}, fmt.Errorf("cannot access file: %w", err)
	}
	if fileInfo.IsDir() {
		return UpdateDocumentFromFileResult{}, fmt.Errorf("file_path is a directory, not a file")
	}

	const maxSize = 6 * 1024 * 1024
	if fileInfo.Size() > maxSize {
		return UpdateDocumentFromFileResult{}, fmt.Errorf("file size %d bytes exceeds 6 MB limit", fileInfo.Size())
	}

	ext := strings.ToLower(filepath.Ext(args.FilePath))
	validExts := map[string]bool{".pdf": true, ".doc": true, ".docx": true, ".ppt": true, ".pptx": true, ".xls": true, ".xlsx": true, ".txt": true, ".rtf": true, ".csv": true}
	if !validExts[ext] {
		return UpdateDocumentFromFileResult{}, fmt.Errorf("unsupported document format %q (supported: pdf, doc, docx, ppt, pptx, xls, xlsx, txt, rtf, csv)", ext)
	}

	file, err := os.Open(args.FilePath)
	if err != nil {
		return UpdateDocumentFromFileResult{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	body, contentType, err := buildMultipartBody(file, filepath.Base(args.FilePath), args.Title, args.X, args.Y, args.ParentID)
	if err != nil {
		return UpdateDocumentFromFileResult{}, err
	}

	respBody, err := c.requestMultipart(ctx, http.MethodPatch,
		"/boards/"+args.BoardID+"/documents/"+args.ItemID,
		contentType, body)
	if err != nil {
		return UpdateDocumentFromFileResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateDocumentFromFileResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidatePrefix("items:" + args.BoardID)

	title := resp.Data.Title
	if title == "" {
		title = filepath.Base(args.FilePath)
	}

	return UpdateDocumentFromFileResult{
		ID:      resp.ID,
		ItemURL: BuildItemURL(args.BoardID, resp.ID),
		Title:   title,
		Message: fmt.Sprintf("Updated document '%s' with new file", title),
	}, nil
}

// buildMultipartBody creates the multipart form body shared by upload and update-from-file methods.
func buildMultipartBody(file *os.File, filename, title string, x, y float64, parentID string) (*bytes.Buffer, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	dataJSON := map[string]interface{}{}
	if title != "" {
		dataJSON["title"] = title
	}
	if x != 0 || y != 0 {
		dataJSON["position"] = map[string]interface{}{
			"x":      x,
			"y":      y,
			"origin": "center",
		}
	}
	if parentID != "" {
		dataJSON["parent"] = map[string]interface{}{
			"id": parentID,
		}
	}

	dataBytes, err := json.Marshal(dataJSON)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal data: %w", err)
	}

	dataPart, err := writer.CreateFormField("data")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create data field: %w", err)
	}
	if _, err := dataPart.Write(dataBytes); err != nil {
		return nil, "", fmt.Errorf("failed to write data: %w", err)
	}

	resourcePart, err := writer.CreateFormFile("resource", filename)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create resource field: %w", err)
	}
	if _, err := io.Copy(resourcePart, file); err != nil {
		return nil, "", fmt.Errorf("failed to write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &body, writer.FormDataContentType(), nil
}
