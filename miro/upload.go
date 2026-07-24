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

// validImageExts is the allowlist of file extensions for image uploads.
var validImageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".webp": true, ".svg": true,
}

// validDocumentExts is the allowlist of file extensions for document uploads.
var validDocumentExts = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true,
	".ppt": true, ".pptx": true, ".xls": true, ".xlsx": true,
	".txt": true, ".rtf": true, ".csv": true,
}

const (
	imageExtsHint    = "supported: png, jpg, jpeg, gif, webp, svg"
	documentExtsHint = "supported: pdf, doc, docx, ppt, pptx, xls, xlsx, txt, rtf, csv"
	maxDocumentSize  = 6 * 1024 * 1024
)

// ValidateUploadPath checks that the given file path is under an allowed directory.
// Allowed directories are the current working directory and any directories listed
// in the MIRO_UPLOAD_ALLOWED_DIRS environment variable (comma-separated).
// Symlinks are resolved before checking.
func ValidateUploadPath(filePath string) (string, error) {
	resolved, err := resolveSymlinkPath(filePath)
	if err != nil {
		return "", err
	}
	if !pathUnderAnyAllowed(resolved, collectAllowedUploadDirs()) {
		return "", fmt.Errorf("file path %q is outside allowed directories", filePath)
	}
	return resolved, nil
}

// resolveSymlinkPath resolves the input to an absolute path with all symlinks
// dereferenced. Used both for the upload candidate and for allowed roots so the
// containment check operates on canonical paths.
func resolveSymlinkPath(path string) (string, error) {
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}
	return resolved, nil
}

// collectAllowedUploadDirs returns the set of directories that uploads are allowed
// to come from: the current working directory plus any in MIRO_UPLOAD_ALLOWED_DIRS.
func collectAllowedUploadDirs() []string {
	var allowed []string
	if cwd, ok := allowedCwd(); ok {
		allowed = append(allowed, cwd)
	}
	allowed = append(allowed, allowedDirsFromEnv("MIRO_UPLOAD_ALLOWED_DIRS")...)
	return allowed
}

// allowedCwd returns the symlink-resolved working directory, or false if it cannot
// be determined or resolved (in which case it is simply omitted from the allowlist).
func allowedCwd() (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	resolved, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return "", false
	}
	return resolved, true
}

// allowedDirsFromEnv parses a comma-separated list from envVar, resolves each
// entry's symlinks, and returns those that resolve cleanly. Unresolvable entries
// are dropped silently.
func allowedDirsFromEnv(envVar string) []string {
	raw := os.Getenv(envVar)
	if raw == "" {
		return nil
	}
	var out []string
	for _, entry := range strings.Split(raw, ",") {
		if resolved, ok := resolveAllowedDir(entry); ok {
			out = append(out, resolved)
		}
	}
	return out
}

// resolveAllowedDir resolves a single allowlist entry to an absolute, symlink-
// dereferenced path. Returns false if the entry is empty or cannot be resolved.
func resolveAllowedDir(raw string) (string, bool) {
	dir := strings.TrimSpace(raw)
	if dir == "" {
		return "", false
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", false
	}
	return resolved, true
}

// pathUnderAnyAllowed reports whether resolved equals or is contained beneath any
// of the allowed roots. The dir+sep prefix guard prevents sibling-prefix tricks
// (e.g. "/allowed-x" matching "/allowed").
func pathUnderAnyAllowed(resolved string, allowed []string) bool {
	sep := string(filepath.Separator)
	for _, dir := range allowed {
		if resolved == dir || strings.HasPrefix(resolved, dir+sep) {
			return true
		}
	}
	return false
}

// fileValidationOpts configures the shared per-file validator used by upload methods.
type fileValidationOpts struct {
	validExts map[string]bool
	kind      string // "image" | "document" — used in error messages
	hint      string // human-readable list of supported extensions
	maxSize   int64  // 0 means no limit
}

// validateUploadFile validates that filePath exists, is a regular file with an
// allowed extension and (optionally) within a size limit, and resolves it through
// ValidateUploadPath. Returns the resolved path on success.
func validateUploadFile(filePath string, opts fileValidationOpts) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file_path is required")
	}
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot access file: %w", err)
	}
	if fileInfo.IsDir() {
		return "", fmt.Errorf("file_path is a directory, not a file")
	}
	if opts.maxSize > 0 && fileInfo.Size() > opts.maxSize {
		return "", fmt.Errorf("file size %d bytes exceeds %d MB limit", fileInfo.Size(), opts.maxSize/(1024*1024))
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if !opts.validExts[ext] {
		return "", fmt.Errorf("unsupported %s format %q (%s)", opts.kind, ext, opts.hint)
	}
	return ValidateUploadPath(filePath)
}

// validateImageFile validates an image upload candidate.
func validateImageFile(filePath string) (string, error) {
	return validateUploadFile(filePath, fileValidationOpts{
		validExts: validImageExts,
		kind:      "image",
		hint:      imageExtsHint,
	})
}

// validateDocumentFile validates a document upload candidate, enforcing the 6 MB cap.
func validateDocumentFile(filePath string) (string, error) {
	return validateUploadFile(filePath, fileValidationOpts{
		validExts: validDocumentExts,
		kind:      "document",
		hint:      documentExtsHint,
		maxSize:   maxDocumentSize,
	})
}

// uploadAPIResponse is the shape returned by parseUploadResponse: the parsed item
// id, an effective title (server-provided or filename fallback), and the item URL
// built from the board id.
type uploadAPIResponse struct {
	ID      string
	Title   string
	ItemURL string
}

// parseUploadResponse decodes the JSON returned by image/document upload calls,
// resolves a fallback title from the supplied filename when the server response
// omits one, and computes the item URL.
func parseUploadResponse(respBody []byte, boardID, fallbackTitle string) (uploadAPIResponse, error) {
	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return uploadAPIResponse{}, fmt.Errorf("failed to parse response: %w", err)
	}
	title := resp.Data.Title
	if title == "" {
		title = fallbackTitle
	}
	return uploadAPIResponse{
		ID:      resp.ID,
		Title:   title,
		ItemURL: BuildItemURL(boardID, resp.ID),
	}, nil
}

// uploadFormOpts bundles the per-call form fields shared by upload and update-from-file
// multipart calls (title, position, parent).
type uploadFormOpts struct {
	title    string
	x, y     float64
	parentID string
}

// uploadKind bundles the per-item-type differences (API path segment, file
// validator, display noun) between image and document uploads.
type uploadKind struct {
	segment  string
	validate func(string) (string, error)
	noun     string
}

var (
	imageUploads    = uploadKind{segment: "images", validate: validateImageFile, noun: "image"}
	documentUploads = uploadKind{segment: "documents", validate: validateDocumentFile, noun: "document"}
)

// uploadRequest is one upload or update-from-file operation: the target kind
// and board, an optional existing item to replace, the local file, and the
// shared form fields.
type uploadRequest struct {
	kind     uploadKind
	boardID  string
	itemID   string // item whose file is being replaced; used when replace is set
	replace  bool   // true for update-from-file calls, false for new uploads
	filePath string
	form     uploadFormOpts
}

// message renders the user-facing confirmation for a completed operation.
func (r uploadRequest) message(title string) string {
	if r.replace {
		return fmt.Sprintf("Updated %s '%s' with new file", r.kind.noun, title)
	}
	return fmt.Sprintf("Uploaded %s '%s'", r.kind.noun, title)
}

// runUpload validates the request and performs the shared multipart flow.
// New uploads POST to the collection path; replacements PATCH the item path.
func (c *Client) runUpload(ctx context.Context, req uploadRequest) (uploadAPIResponse, error) {
	if err := ValidateBoardID(req.boardID); err != nil {
		return uploadAPIResponse{}, err
	}
	method := http.MethodPost
	path := "/boards/" + req.boardID + "/" + req.kind.segment
	if req.replace {
		if err := ValidateItemID(req.itemID); err != nil {
			return uploadAPIResponse{}, err
		}
		method = http.MethodPatch
		path += "/" + req.itemID
	}
	resolvedPath, err := req.kind.validate(req.filePath)
	if err != nil {
		return uploadAPIResponse{}, err
	}
	return c.uploadMultipart(ctx, multipartUploadCall{
		method:        method,
		path:          path,
		boardID:       req.boardID,
		filePath:      req.filePath,
		resolvedPath:  resolvedPath,
		form:          req.form,
		fallbackTitle: filepath.Base(req.filePath),
	})
}

// runUploadResult runs the upload and shapes the shared result type.
func (c *Client) runUploadResult(ctx context.Context, req uploadRequest) (UploadedItemResult, error) {
	parsed, err := c.runUpload(ctx, req)
	if err != nil {
		return UploadedItemResult{}, err
	}
	return UploadedItemResult{
		ID:      parsed.ID,
		ItemURL: parsed.ItemURL,
		Title:   parsed.Title,
		Message: req.message(parsed.Title),
	}, nil
}

// newUpload shapes a create-style upload request for the given kind.
func newUpload(kind uploadKind, boardID, filePath string, form uploadFormOpts) uploadRequest {
	return uploadRequest{kind: kind, boardID: boardID, filePath: filePath, form: form}
}

// withReplace marks the request as an update-from-file call targeting itemID.
func (r uploadRequest) withReplace(itemID string) uploadRequest {
	r.itemID = itemID
	r.replace = true
	return r
}

// formOf collects the shared optional form fields (title, position, parent).
func formOf(title string, x, y float64, parentID string) uploadFormOpts {
	return uploadFormOpts{title: title, x: x, y: y, parentID: parentID}
}

// UploadImage uploads a local image file to a Miro board.
func (c *Client) UploadImage(ctx context.Context, args UploadImageArgs) (UploadImageResult, error) {
	form := formOf(args.Title, args.X, args.Y, args.ParentID)
	return c.runUploadResult(ctx, newUpload(imageUploads, args.BoardID, args.FilePath, form))
}

// UploadDocument uploads a local document file to a Miro board.
func (c *Client) UploadDocument(ctx context.Context, args UploadDocumentArgs) (UploadDocumentResult, error) {
	form := formOf(args.Title, args.X, args.Y, args.ParentID)
	return c.runUploadResult(ctx, newUpload(documentUploads, args.BoardID, args.FilePath, form))
}

// UpdateImageFromFile replaces the file on an existing image item via PATCH multipart.
func (c *Client) UpdateImageFromFile(ctx context.Context, args UpdateImageFromFileArgs) (UpdateImageFromFileResult, error) {
	form := formOf(args.Title, args.X, args.Y, args.ParentID)
	req := newUpload(imageUploads, args.BoardID, args.FilePath, form).withReplace(args.ItemID)
	return c.runUploadResult(ctx, req)
}

// UpdateDocumentFromFile replaces the file on an existing document item via PATCH multipart.
func (c *Client) UpdateDocumentFromFile(ctx context.Context, args UpdateDocumentFromFileArgs) (UpdateDocumentFromFileResult, error) {
	form := formOf(args.Title, args.X, args.Y, args.ParentID)
	req := newUpload(documentUploads, args.BoardID, args.FilePath, form).withReplace(args.ItemID)
	return c.runUploadResult(ctx, req)
}

// multipartUploadCall bundles the per-call inputs for the shared upload skeleton.
type multipartUploadCall struct {
	method        string
	path          string
	boardID       string
	filePath      string
	resolvedPath  string
	form          uploadFormOpts
	fallbackTitle string
}

// uploadMultipart performs the shared file-open, multipart-encode, request,
// response-parse, and cache-invalidate sequence used by the four Upload*/Update*
// methods. The per-call differences (HTTP method, API path, fallback title) are
// supplied via multipartUploadCall.
func (c *Client) uploadMultipart(ctx context.Context, call multipartUploadCall) (uploadAPIResponse, error) {
	file, err := os.Open(call.resolvedPath)
	if err != nil {
		return uploadAPIResponse{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	body, contentType, err := buildMultipartBody(file, filepath.Base(call.filePath), call.form)
	if err != nil {
		return uploadAPIResponse{}, err
	}

	respBody, err := c.requestMultipart(ctx, multipartRequest{
		method:      call.method,
		path:        call.path,
		contentType: contentType,
		body:        body,
	})
	if err != nil {
		return uploadAPIResponse{}, err
	}

	parsed, err := parseUploadResponse(respBody, call.boardID, call.fallbackTitle)
	if err != nil {
		return uploadAPIResponse{}, err
	}

	c.cache.InvalidatePrefix("items:" + call.boardID)
	return parsed, nil
}

// buildMultipartBody creates the multipart form body shared by upload and update-from-file methods.
func buildMultipartBody(file *os.File, filename string, opts uploadFormOpts) (*bytes.Buffer, string, error) {
	dataBytes, err := buildUploadDataJSON(opts)
	if err != nil {
		return nil, "", err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writeMultipartDataField(writer, dataBytes); err != nil {
		return nil, "", err
	}
	if err := writeMultipartResource(writer, file, filename); err != nil {
		return nil, "", err
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &body, writer.FormDataContentType(), nil
}

// buildUploadDataJSON serializes the optional title/position/parent fields into
// the JSON payload Miro expects in the multipart "data" field.
func buildUploadDataJSON(opts uploadFormOpts) ([]byte, error) {
	dataJSON := map[string]interface{}{}
	if opts.title != "" {
		dataJSON["title"] = opts.title
	}
	if opts.x != 0 || opts.y != 0 {
		dataJSON["position"] = map[string]interface{}{
			"x":      opts.x,
			"y":      opts.y,
			"origin": "center",
		}
	}
	if opts.parentID != "" {
		dataJSON["parent"] = map[string]interface{}{"id": opts.parentID}
	}
	dataBytes, err := json.Marshal(dataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}
	return dataBytes, nil
}

// writeMultipartDataField writes the "data" form field containing the JSON payload.
func writeMultipartDataField(writer *multipart.Writer, dataBytes []byte) error {
	dataPart, err := writer.CreateFormField("data")
	if err != nil {
		return fmt.Errorf("failed to create data field: %w", err)
	}
	if _, err := dataPart.Write(dataBytes); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	return nil
}

// writeMultipartResource writes the "resource" form file containing the upload bytes.
func writeMultipartResource(writer *multipart.Writer, file *os.File, filename string) error {
	resourcePart, err := writer.CreateFormFile("resource", filename)
	if err != nil {
		return fmt.Errorf("failed to create resource field: %w", err)
	}
	if _, err := io.Copy(resourcePart, file); err != nil {
		return fmt.Errorf("failed to write file data: %w", err)
	}
	return nil
}
