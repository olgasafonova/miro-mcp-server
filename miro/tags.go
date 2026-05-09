package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// =============================================================================
// Tag Operations
// =============================================================================

// CreateTag creates a tag on a board.
func (c *Client) CreateTag(ctx context.Context, args CreateTagArgs) (CreateTagResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateTagResult{}, err
	}
	if args.Title == "" {
		return CreateTagResult{}, fmt.Errorf("title is required")
	}

	// Color is required by Miro API; default to "blue" when unset.
	color := args.Color
	if color == "" {
		color = "blue"
	}

	reqBody := map[string]interface{}{
		"title":     args.Title,
		"fillColor": normalizeTagColor(color),
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/tags", reqBody)
	if err != nil {
		return CreateTagResult{}, err
	}

	var tag Tag
	if err := json.Unmarshal(respBody, &tag); err != nil {
		return CreateTagResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CreateTagResult{
		ID:      tag.ID,
		ItemURL: BuildItemURL(args.BoardID, tag.ID),
		Title:   tag.Title,
		Color:   tag.FillColor,
		Message: fmt.Sprintf("Created tag '%s'", args.Title),
	}, nil
}

// ListTags retrieves all tags from a board.
func (c *Client) ListTags(ctx context.Context, args ListTagsArgs) (ListTagsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListTagsResult{}, err
	}

	params := url.Values{}
	limit := DefaultItemLimit
	if args.Limit > 0 && args.Limit <= MaxItemLimit {
		limit = args.Limit
	}
	params.Set("limit", strconv.Itoa(limit))

	path := "/boards/" + args.BoardID + "/tags?" + params.Encode()

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ListTagsResult{}, err
	}

	var resp struct {
		Data []Tag `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ListTagsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	message := fmt.Sprintf("Found %d tags", len(resp.Data))
	if len(resp.Data) == 0 {
		message = "No tags on this board"
	}

	return ListTagsResult{
		Tags:    resp.Data,
		Count:   len(resp.Data),
		Message: message,
	}, nil
}

// tagItemRequest bundles the per-call inputs to runTagItemAction, the shared
// dispatcher that backs AttachTag and DetachTag.
type tagItemRequest struct {
	method      string // POST for attach, DELETE for detach
	successMsg  string // populated into the result on success
	failureVerb string // verb in "Failed to <verb> tag" failure message
	boardID     string
	itemID      string
	tagID       string
}

// validateTagItemRequest validates the (boardID, itemID, tagID) trio shared by
// AttachTag and DetachTag. tagID is required because the underlying endpoint
// encodes it as a query parameter.
func validateTagItemRequest(req tagItemRequest) error {
	if err := ValidateBoardID(req.boardID); err != nil {
		return err
	}
	if err := ValidateItemID(req.itemID); err != nil {
		return err
	}
	if req.tagID == "" {
		return fmt.Errorf("tag_id is required")
	}
	return nil
}

// tagItemPath returns the attach/detach endpoint path with tag_id encoded as
// a query parameter (Miro's per-item tag endpoint is the same URL regardless
// of HTTP method).
func tagItemPath(boardID, itemID, tagID string) string {
	return fmt.Sprintf("/boards/%s/items/%s?tag_id=%s", boardID, itemID, tagID)
}

// runTagItemAction validates args, executes the request, and returns the four
// fields the typed AttachTag/DetachTag wrappers need:
//   - filled: false on validation error (caller returns zero result + err)
//   - success: meaningful only when filled=true
//   - message: success or failure text to populate the result's Message field
//   - err: raw error from validation or the HTTP request
func (c *Client) runTagItemAction(ctx context.Context, req tagItemRequest) (filled bool, success bool, message string, err error) {
	if err := validateTagItemRequest(req); err != nil {
		return false, false, "", err
	}
	if _, err := c.request(ctx, req.method, tagItemPath(req.boardID, req.itemID, req.tagID), nil); err != nil {
		return true, false, fmt.Sprintf("Failed to %s tag: %v", req.failureVerb, err), err
	}
	return true, true, req.successMsg, nil
}

// AttachTag attaches a tag to an item (sticky note).
func (c *Client) AttachTag(ctx context.Context, args AttachTagArgs) (AttachTagResult, error) {
	filled, success, msg, err := c.runTagItemAction(ctx, tagItemRequest{
		method:      http.MethodPost,
		successMsg:  "Tag attached successfully",
		failureVerb: "attach",
		boardID:     args.BoardID,
		itemID:      args.ItemID,
		tagID:       args.TagID,
	})
	if !filled {
		return AttachTagResult{}, err
	}
	return AttachTagResult{Success: success, ItemID: args.ItemID, TagID: args.TagID, Message: msg}, err
}

// DetachTag removes a tag from an item.
func (c *Client) DetachTag(ctx context.Context, args DetachTagArgs) (DetachTagResult, error) {
	filled, success, msg, err := c.runTagItemAction(ctx, tagItemRequest{
		method:      http.MethodDelete,
		successMsg:  "Tag removed successfully",
		failureVerb: "detach",
		boardID:     args.BoardID,
		itemID:      args.ItemID,
		tagID:       args.TagID,
	})
	if !filled {
		return DetachTagResult{}, err
	}
	return DetachTagResult{Success: success, ItemID: args.ItemID, TagID: args.TagID, Message: msg}, err
}

// GetItemTags retrieves tags attached to an item.
func (c *Client) GetItemTags(ctx context.Context, args GetItemTagsArgs) (GetItemTagsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetItemTagsResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return GetItemTagsResult{}, err
	}

	path := fmt.Sprintf("/boards/%s/items/%s/tags", args.BoardID, args.ItemID)

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetItemTagsResult{}, err
	}

	var resp struct {
		Data []Tag `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return GetItemTagsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Ensure tags is never nil (MCP schema validation requires array, not null).
	tags := resp.Data
	if tags == nil {
		tags = []Tag{}
	}

	message := fmt.Sprintf("Item has %d tags", len(tags))
	if len(tags) == 0 {
		message = "No tags on this item"
	}

	return GetItemTagsResult{
		Tags:    tags,
		Count:   len(tags),
		ItemID:  args.ItemID,
		Message: message,
	}, nil
}

// getTagInternal retrieves a single tag by ID (internal helper).
func (c *Client) getTagInternal(ctx context.Context, boardID, tagID string) (Tag, error) {
	if boardID == "" {
		return Tag{}, fmt.Errorf("board_id is required")
	}
	if tagID == "" {
		return Tag{}, fmt.Errorf("tag_id is required")
	}

	path := fmt.Sprintf("/boards/%s/tags/%s", boardID, tagID)

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return Tag{}, err
	}

	var tag Tag
	if err := json.Unmarshal(respBody, &tag); err != nil {
		return Tag{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return tag, nil
}

// GetTag retrieves a single tag by ID.
func (c *Client) GetTag(ctx context.Context, args GetTagArgs) (GetTagResult, error) {
	tag, err := c.getTagInternal(ctx, args.BoardID, args.TagID)
	if err != nil {
		return GetTagResult{}, err
	}

	return GetTagResult{
		ID:      tag.ID,
		Title:   tag.Title,
		Color:   tag.FillColor,
		Message: fmt.Sprintf("Tag '%s'", tag.Title),
	}, nil
}

// tagUpdateFields bundles the inputs to resolveTagUpdateFields.
type tagUpdateFields struct {
	boardID, tagID, title, color string
}

// resolveTagUpdateFields fills in any unspecified field from the existing tag.
// Miro's PATCH /tags clears omitted fields, so a partial update needs the
// existing values for any field the caller did not supply.
func (c *Client) resolveTagUpdateFields(ctx context.Context, in tagUpdateFields) (string, string, error) {
	if in.title != "" && in.color != "" {
		return in.title, in.color, nil
	}
	existing, err := c.getTagInternal(ctx, in.boardID, in.tagID)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch existing tag: %w", err)
	}
	title, color := in.title, in.color
	if title == "" {
		title = existing.Title
	}
	if color == "" {
		color = existing.FillColor
	}
	return title, color, nil
}

// UpdateTag updates an existing tag on a board.
// When only color is provided, preserves the existing title (and vice versa).
func (c *Client) UpdateTag(ctx context.Context, args UpdateTagArgs) (UpdateTagResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateTagResult{}, err
	}
	if args.TagID == "" {
		return UpdateTagResult{}, fmt.Errorf("tag_id is required")
	}
	if args.Title == "" && args.Color == "" {
		return UpdateTagResult{}, fmt.Errorf("at least one of title or color is required")
	}

	title, color, err := c.resolveTagUpdateFields(ctx, tagUpdateFields{
		boardID: args.BoardID,
		tagID:   args.TagID,
		title:   args.Title,
		color:   args.Color,
	})
	if err != nil {
		return UpdateTagResult{}, err
	}

	reqBody := map[string]interface{}{
		"title":     title,
		"fillColor": normalizeTagColor(color),
	}

	path := fmt.Sprintf("/boards/%s/tags/%s", args.BoardID, args.TagID)

	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateTagResult{}, err
	}

	var tag Tag
	if err := json.Unmarshal(respBody, &tag); err != nil {
		return UpdateTagResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return UpdateTagResult{
		Success: true,
		ID:      tag.ID,
		Title:   tag.Title,
		Color:   tag.FillColor,
		Message: fmt.Sprintf("Updated tag '%s'", tag.Title),
	}, nil
}

// DeleteTag removes a tag from a board.
func (c *Client) DeleteTag(ctx context.Context, args DeleteTagArgs) (DeleteTagResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return DeleteTagResult{}, err
	}
	if args.TagID == "" {
		return DeleteTagResult{}, fmt.Errorf("tag_id is required")
	}

	if args.DryRun {
		return DeleteTagResult{
			Success: true,
			TagID:   args.TagID,
			Message: "[DRY RUN] Would delete tag " + args.TagID + " from board " + args.BoardID,
		}, nil
	}

	path := fmt.Sprintf("/boards/%s/tags/%s", args.BoardID, args.TagID)

	if _, err := c.request(ctx, http.MethodDelete, path, nil); err != nil {
		return DeleteTagResult{
			Success: false,
			TagID:   args.TagID,
			Message: fmt.Sprintf("Failed to delete tag: %v", err),
		}, err
	}

	return DeleteTagResult{
		Success: true,
		TagID:   args.TagID,
		Message: "Tag deleted successfully",
	}, nil
}

// maxTagItemsPageSize is the upper bound Miro enforces for the items-by-tag endpoint.
const maxTagItemsPageSize = 50

// clampTagItemsLimit applies the items-by-tag page-size rules: a non-positive
// limit defaults to the cap, a request larger than the cap is clamped to it.
func clampTagItemsLimit(requested int) int {
	if requested <= 0 || requested > maxTagItemsPageSize {
		return maxTagItemsPageSize
	}
	return requested
}

// buildTagItemsPath builds the /items?tag_id=... URL for GetItemsByTag.
func buildTagItemsPath(boardID, tagID string, limit, offset int) string {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("tag_id", tagID)
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	return "/boards/" + boardID + "/items?" + params.Encode()
}

// GetItemsByTag returns items on a board filtered by tag ID.
func (c *Client) GetItemsByTag(ctx context.Context, args GetItemsByTagArgs) (GetItemsByTagResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetItemsByTagResult{}, err
	}
	if args.TagID == "" {
		return GetItemsByTagResult{}, fmt.Errorf("tag_id is required")
	}

	limit := clampTagItemsLimit(args.Limit)
	respBody, err := c.request(ctx, http.MethodGet, buildTagItemsPath(args.BoardID, args.TagID, limit, args.Offset), nil)
	if err != nil {
		return GetItemsByTagResult{}, err
	}

	var resp struct {
		Data []json.RawMessage `json:"data"`
		Size int               `json:"size"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return GetItemsByTagResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	items := make([]ItemSummary, 0, len(resp.Data))
	for _, raw := range resp.Data {
		items = append(items, parseItemSummary(raw, false))
	}

	return GetItemsByTagResult{
		Items:   items,
		Count:   len(items),
		HasMore: len(items) >= limit,
		TagID:   args.TagID,
		Message: fmt.Sprintf("Found %d items with tag %s", len(items), args.TagID),
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// tagColorAliases maps caller-supplied color names (case-insensitive) to the
// canonical Miro tag color identifier. "grey" is folded into "gray" as an alias.
var tagColorAliases = map[string]string{
	"red":     "red",
	"magenta": "magenta",
	"violet":  "violet",
	"blue":    "blue",
	"cyan":    "cyan",
	"green":   "green",
	"yellow":  "yellow",
	"orange":  "orange",
	"gray":    "gray",
	"grey":    "gray",
}

// normalizeTagColor converts color names to Miro's expected format for tags.
// Unknown colors are returned unchanged so callers can pass hex codes through.
func normalizeTagColor(color string) string {
	if mapped, ok := tagColorAliases[strings.ToLower(color)]; ok {
		return mapped
	}
	return color
}
