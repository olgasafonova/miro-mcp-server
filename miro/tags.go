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

// maxTagListPageSize is the upper bound Miro enforces on the board tag
// listing. The endpoint has no lower bound: limit=1 answers HTTP 200,
// verified live against api.miro.com on 18-09-2026, so unlike the items,
// connectors and groups routes this one takes no minimum-page-size floor.
const maxTagListPageSize = 50

// tagListRequest is one page of the board tag listing: which board, where to
// start, and how many rows. The page rules live on it as methods so ListTags
// reads as named steps rather than as a handful of loose ints.
type tagListRequest struct {
	boardID string
	limit   int
	offset  int
}

// validate rejects a negative offset instead of reading it as the first page,
// which would silently restart a walk the caller believed was advancing.
func (r tagListRequest) validate() error {
	if r.offset < 0 {
		return fmt.Errorf("invalid offset %d: must be a non-negative integer", r.offset)
	}
	return nil
}

// clamped applies the endpoint's page-size rules: a non-positive limit falls
// back to the cap, and a larger one is clamped to it.
func (r tagListRequest) clamped() tagListRequest {
	if r.limit <= 0 || r.limit > maxTagListPageSize {
		r.limit = maxTagListPageSize
	}
	return r
}

// path builds the /tags URL. The offset is left off the first page, keeping
// the common request the shape Miro's own examples use.
func (r tagListRequest) path() string {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(r.limit))
	if r.offset > 0 {
		params.Set("offset", strconv.Itoa(r.offset))
	}
	return "/boards/" + r.boardID + "/tags?" + params.Encode()
}

// ListTags retrieves one page of tag definitions from a board.
func (c *Client) ListTags(ctx context.Context, args ListTagsArgs) (ListTagsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListTagsResult{}, err
	}

	req := tagListRequest{boardID: args.BoardID, limit: args.Limit, offset: args.Offset}
	if err := req.validate(); err != nil {
		return ListTagsResult{}, err
	}
	req = req.clamped()

	respBody, err := c.request(ctx, http.MethodGet, req.path(), nil)
	if err != nil {
		return ListTagsResult{}, err
	}

	var resp struct {
		Data  []Tag `json:"data"`
		Total int   `json:"total,omitempty"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ListTagsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Ensure tags is never nil (MCP schema validation requires array, not null).
	tags := resp.Data
	if tags == nil {
		tags = []Tag{}
	}

	// The tags endpoint echoes back the offset of the page it just served, so
	// the next cursor comes from the offset we requested plus the rows we got.
	next := req.offset + len(tags)

	message := fmt.Sprintf("Found %d tags", len(tags))
	if len(tags) == 0 {
		message = "No tags on this board"
	}

	return ListTagsResult{
		Tags:    tags,
		Count:   len(tags),
		Total:   resp.Total,
		HasMore: offsetHasMore(next, resp.Total, len(tags), req.limit),
		Offset:  nextOffset(next, resp.Total, len(tags), req.limit),
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
