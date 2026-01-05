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
	if args.BoardID == "" {
		return CreateTagResult{}, fmt.Errorf("board_id is required")
	}
	if args.Title == "" {
		return CreateTagResult{}, fmt.Errorf("title is required")
	}

	// Color is required by Miro API, default to "blue" if not specified
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
	if args.BoardID == "" {
		return ListTagsResult{}, fmt.Errorf("board_id is required")
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

// AttachTag attaches a tag to an item (sticky note).
func (c *Client) AttachTag(ctx context.Context, args AttachTagArgs) (AttachTagResult, error) {
	if args.BoardID == "" {
		return AttachTagResult{}, fmt.Errorf("board_id is required")
	}
	if args.ItemID == "" {
		return AttachTagResult{}, fmt.Errorf("item_id is required")
	}
	if args.TagID == "" {
		return AttachTagResult{}, fmt.Errorf("tag_id is required")
	}

	path := fmt.Sprintf("/boards/%s/items/%s?tag_id=%s", args.BoardID, args.ItemID, args.TagID)

	_, err := c.request(ctx, http.MethodPost, path, nil)
	if err != nil {
		return AttachTagResult{
			Success: false,
			ItemID:  args.ItemID,
			TagID:   args.TagID,
			Message: fmt.Sprintf("Failed to attach tag: %v", err),
		}, err
	}

	return AttachTagResult{
		Success: true,
		ItemID:  args.ItemID,
		TagID:   args.TagID,
		Message: "Tag attached successfully",
	}, nil
}

// DetachTag removes a tag from an item.
func (c *Client) DetachTag(ctx context.Context, args DetachTagArgs) (DetachTagResult, error) {
	if args.BoardID == "" {
		return DetachTagResult{}, fmt.Errorf("board_id is required")
	}
	if args.ItemID == "" {
		return DetachTagResult{}, fmt.Errorf("item_id is required")
	}
	if args.TagID == "" {
		return DetachTagResult{}, fmt.Errorf("tag_id is required")
	}

	path := fmt.Sprintf("/boards/%s/items/%s?tag_id=%s", args.BoardID, args.ItemID, args.TagID)

	_, err := c.request(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return DetachTagResult{
			Success: false,
			ItemID:  args.ItemID,
			TagID:   args.TagID,
			Message: fmt.Sprintf("Failed to detach tag: %v", err),
		}, err
	}

	return DetachTagResult{
		Success: true,
		ItemID:  args.ItemID,
		TagID:   args.TagID,
		Message: "Tag removed successfully",
	}, nil
}

// GetItemTags retrieves tags attached to an item.
func (c *Client) GetItemTags(ctx context.Context, args GetItemTagsArgs) (GetItemTagsResult, error) {
	if args.BoardID == "" {
		return GetItemTagsResult{}, fmt.Errorf("board_id is required")
	}
	if args.ItemID == "" {
		return GetItemTagsResult{}, fmt.Errorf("item_id is required")
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

	// Ensure tags is never nil (MCP schema validation requires array, not null)
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

// UpdateTag updates an existing tag on a board.
// When only color is provided, preserves the existing title (and vice versa).
func (c *Client) UpdateTag(ctx context.Context, args UpdateTagArgs) (UpdateTagResult, error) {
	if args.BoardID == "" {
		return UpdateTagResult{}, fmt.Errorf("board_id is required")
	}
	if args.TagID == "" {
		return UpdateTagResult{}, fmt.Errorf("tag_id is required")
	}
	if args.Title == "" && args.Color == "" {
		return UpdateTagResult{}, fmt.Errorf("at least one of title or color is required")
	}

	// If doing a partial update, fetch existing tag to preserve unspecified fields
	// Miro's API clears fields that aren't included in the PATCH request
	title := args.Title
	color := args.Color

	if title == "" || color == "" {
		existingTag, err := c.getTagInternal(ctx, args.BoardID, args.TagID)
		if err != nil {
			return UpdateTagResult{}, fmt.Errorf("failed to fetch existing tag: %w", err)
		}
		if title == "" {
			title = existingTag.Title
		}
		if color == "" {
			color = existingTag.FillColor
		}
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
	if args.BoardID == "" {
		return DeleteTagResult{}, fmt.Errorf("board_id is required")
	}
	if args.TagID == "" {
		return DeleteTagResult{}, fmt.Errorf("tag_id is required")
	}

	// Dry-run mode: return preview without deleting
	if args.DryRun {
		return DeleteTagResult{
			Success: true,
			TagID:   args.TagID,
			Message: "[DRY RUN] Would delete tag " + args.TagID + " from board " + args.BoardID,
		}, nil
	}

	path := fmt.Sprintf("/boards/%s/tags/%s", args.BoardID, args.TagID)

	_, err := c.request(ctx, http.MethodDelete, path, nil)
	if err != nil {
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

// =============================================================================
// Helper Functions
// =============================================================================

// normalizeTagColor converts color names to Miro's expected format for tags.
func normalizeTagColor(color string) string {
	colorMap := map[string]string{
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

	lower := strings.ToLower(color)
	if mapped, ok := colorMap[lower]; ok {
		return mapped
	}
	return color
}
