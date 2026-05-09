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
// Generic Item Operations - List, Get, Update, Delete, ListAll
// =============================================================================

// ListItems retrieves items from a board.
// When detail_level=full, additional fields (style, geometry, timestamps, user info) are included.
func (c *Client) ListItems(ctx context.Context, args ListItemsArgs) (ListItemsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListItemsResult{}, err
	}

	respBody, err := c.request(ctx, http.MethodGet, buildListItemsPath(args), nil)
	if err != nil {
		return ListItemsResult{}, err
	}

	var resp struct {
		Data   []json.RawMessage `json:"data"`
		Cursor string            `json:"cursor,omitempty"`
		Size   int               `json:"size,omitempty"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ListItemsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	fullDetails := strings.EqualFold(args.DetailLevel, "full")
	items := make([]ItemSummary, 0, len(resp.Data))
	for _, raw := range resp.Data {
		item := parseItemSummary(raw, fullDetails)
		if item.ID != "" {
			items = append(items, item)
		}
	}

	return ListItemsResult{
		Items:   items,
		Count:   len(items),
		HasMore: resp.Cursor != "",
		Cursor:  resp.Cursor,
	}, nil
}

// buildListItemsPath assembles the items-list URL with query parameters,
// applying the limit fallback rules (default if unset, capped at MaxItemLimit).
func buildListItemsPath(args ListItemsArgs) string {
	params := url.Values{}
	if args.Type != "" {
		params.Set("type", args.Type)
	}
	limit := DefaultItemLimit
	if args.Limit > 0 && args.Limit <= MaxItemLimit {
		limit = args.Limit
	}
	params.Set("limit", strconv.Itoa(limit))
	if args.Cursor != "" {
		params.Set("cursor", args.Cursor)
	}

	path := "/boards/" + args.BoardID + "/items"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	return path
}

// rawItemPosition mirrors the JSON wire format for an item's position.
type rawItemPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// rawItemGeometry mirrors the JSON wire format for an item's geometry block.
type rawItemGeometry struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// rawItemStyle mirrors the JSON wire format for an item's style block.
type rawItemStyle struct {
	FillColor   string `json:"fillColor"`
	TextAlign   string `json:"textAlign"`
	BorderColor string `json:"borderColor"`
	FontSize    string `json:"fontSize"`
}

// rawItemUser mirrors the JSON wire format for createdBy / modifiedBy actors.
type rawItemUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// rawItemSummary mirrors the JSON wire format used by parseItemSummary.
// Extended fields (Geometry, Style, timestamps, actors) are only populated
// when ListItems is called with detail_level=full.
type rawItemSummary struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Position *rawItemPosition `json:"position"`
	ParentID string           `json:"parentId"`
	Data     struct {
		Content string `json:"content"`
		Title   string `json:"title"`
	} `json:"data"`
	Geometry   *rawItemGeometry `json:"geometry"`
	Style      *rawItemStyle    `json:"style"`
	CreatedAt  string           `json:"createdAt"`
	ModifiedAt string           `json:"modifiedAt"`
	CreatedBy  *rawItemUser     `json:"createdBy"`
	ModifiedBy *rawItemUser     `json:"modifiedBy"`
}

// parseItemSummary extracts an ItemSummary from raw JSON data.
// When fullDetails is true, additional fields are populated.
func parseItemSummary(raw json.RawMessage, fullDetails bool) ItemSummary {
	var base rawItemSummary
	if err := json.Unmarshal(raw, &base); err != nil {
		return ItemSummary{}
	}
	item := minimalItemSummary(base)
	if fullDetails {
		addItemFullDetails(&item, base)
	}
	return item
}

// minimalItemSummary builds the minimum-detail ItemSummary returned for
// every list response (regardless of detail_level).
func minimalItemSummary(base rawItemSummary) ItemSummary {
	content := base.Data.Content
	if content == "" {
		content = base.Data.Title
	}
	item := ItemSummary{
		ID:       base.ID,
		Type:     base.Type,
		Content:  content,
		ParentID: base.ParentID,
	}
	if base.Position != nil {
		item.X = base.Position.X
		item.Y = base.Position.Y
	}
	return item
}

// addItemFullDetails populates the extended fields when detail_level=full.
func addItemFullDetails(item *ItemSummary, base rawItemSummary) {
	if base.Geometry != nil {
		item.Width = base.Geometry.Width
		item.Height = base.Geometry.Height
	}
	if base.Style != nil {
		item.Style = &ItemStyleInfo{
			FillColor:   base.Style.FillColor,
			TextAlign:   base.Style.TextAlign,
			BorderColor: base.Style.BorderColor,
			FontSize:    base.Style.FontSize,
		}
	}
	item.CreatedAt = base.CreatedAt
	item.ModifiedAt = base.ModifiedAt
	if base.CreatedBy != nil {
		item.CreatedBy = &UserInfo{ID: base.CreatedBy.ID, Name: base.CreatedBy.Name}
	}
	if base.ModifiedBy != nil {
		item.ModifiedBy = &UserInfo{ID: base.ModifiedBy.ID, Name: base.ModifiedBy.Name}
	}
}

// rawItemDetail mirrors the JSON wire format used by GetItem.
type rawItemDetail struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Position *rawItemPosition `json:"position"`
	Geometry *rawItemGeometry `json:"geometry"`
	Data     struct {
		Content string `json:"content"`
		Title   string `json:"title"`
		Shape   string `json:"shape"`
	} `json:"data"`
	Style struct {
		FillColor string `json:"fillColor"`
	} `json:"style"`
	ParentID   string `json:"parentId"`
	CreatedAt  string `json:"createdAt"`
	ModifiedAt string `json:"modifiedAt"`
	CreatedBy  *struct {
		Name string `json:"name"`
	} `json:"createdBy"`
	ModifiedBy *struct {
		Name string `json:"name"`
	} `json:"modifiedBy"`
}

// GetItem retrieves detailed information about a specific item.
func (c *Client) GetItem(ctx context.Context, args GetItemArgs) (GetItemResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetItemResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return GetItemResult{}, err
	}

	cacheKey := CacheKeyItem(args.BoardID, args.ItemID)
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached.(GetItemResult), nil
	}

	respBody, err := c.request(ctx, http.MethodGet, "/boards/"+args.BoardID+"/items/"+args.ItemID, nil)
	if err != nil {
		return GetItemResult{}, err
	}

	var item rawItemDetail
	if err := json.Unmarshal(respBody, &item); err != nil {
		return GetItemResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	result := buildGetItemResult(item)
	c.cache.Set(cacheKey, result, c.cacheConfig.ItemTTL)
	return result, nil
}

// buildGetItemResult assembles the GetItemResult from a raw item record,
// folding the four optional-pointer-deref blocks into a flat assignment chain.
func buildGetItemResult(item rawItemDetail) GetItemResult {
	result := GetItemResult{
		ID:         item.ID,
		Type:       item.Type,
		Content:    item.Data.Content,
		Title:      item.Data.Title,
		Shape:      item.Data.Shape,
		Color:      item.Style.FillColor,
		ParentID:   item.ParentID,
		CreatedAt:  item.CreatedAt,
		ModifiedAt: item.ModifiedAt,
	}
	if item.Position != nil {
		result.X = item.Position.X
		result.Y = item.Position.Y
	}
	if item.Geometry != nil {
		result.Width = item.Geometry.Width
		result.Height = item.Geometry.Height
	}
	if item.CreatedBy != nil {
		result.CreatedBy = item.CreatedBy.Name
	}
	if item.ModifiedBy != nil {
		result.ModifiedBy = item.ModifiedBy.Name
	}
	return result
}

// UpdateItem updates an existing item.
func (c *Client) UpdateItem(ctx context.Context, args UpdateItemArgs) (UpdateItemResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateItemResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateItemResult{}, err
	}

	reqBody, err := buildUpdateItemBody(args)
	if err != nil {
		return UpdateItemResult{}, err
	}
	if len(reqBody) == 0 {
		return UpdateItemResult{
			Success: true,
			ItemID:  args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	if _, err := c.request(ctx, http.MethodPatch, "/boards/"+args.BoardID+"/items/"+args.ItemID, reqBody); err != nil {
		return UpdateItemResult{
			Success: false,
			ItemID:  args.ItemID,
			Message: fmt.Sprintf("Failed to update item: %v", err),
		}, err
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateItemResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "Item updated successfully",
	}, nil
}

// buildUpdateItemBody assembles the PATCH body for UpdateItem, including only
// the fields the caller supplied. Returns an empty map when the caller passed
// no updatable fields.
func buildUpdateItemBody(args UpdateItemArgs) (map[string]interface{}, error) {
	reqBody := make(map[string]interface{})

	if args.Content != nil {
		reqBody["data"] = map[string]interface{}{"content": *args.Content}
	}
	if pos := buildUpdatePosition(args.X, args.Y); pos != nil {
		reqBody["position"] = pos
	}
	if geom := buildUpdateGeometry(args.Width, args.Height); geom != nil {
		reqBody["geometry"] = geom
	}
	if args.Color != nil {
		fillColor, err := normalizeColor(*args.Color)
		if err != nil {
			return nil, fmt.Errorf("color: %w", err)
		}
		reqBody["style"] = map[string]interface{}{"fillColor": fillColor}
	}
	if args.ParentID != nil {
		// Empty string explicitly nulls parent (removes from frame).
		reqBody["parent"] = updateParentPayload(*args.ParentID)
	}
	return reqBody, nil
}

// buildUpdateGeometry returns a geometry payload for PATCH-style updates,
// or nil when both pointers are nil. Either pointer can be set independently.
func buildUpdateGeometry(width, height *float64) map[string]interface{} {
	if width == nil && height == nil {
		return nil
	}
	geom := make(map[string]interface{})
	if width != nil {
		geom["width"] = *width
	}
	if height != nil {
		geom["height"] = *height
	}
	return geom
}

// updateParentPayload returns the parent payload for an item update.
// An empty parentID maps to nil (removes the item from its parent frame);
// a non-empty parentID maps to {"id": parentID} (re-parents the item).
func updateParentPayload(parentID string) interface{} {
	if parentID == "" {
		return nil
	}
	return map[string]interface{}{"id": parentID}
}

// DeleteItem deletes an item from a board.
func (c *Client) DeleteItem(ctx context.Context, args DeleteItemArgs) (DeleteItemResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return DeleteItemResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return DeleteItemResult{}, err
	}

	// Dry-run mode: return preview without deleting
	if args.DryRun {
		return DeleteItemResult{
			Success: true,
			ItemID:  args.ItemID,
			Message: "[DRY RUN] Would delete item " + args.ItemID + " from board " + args.BoardID,
		}, nil
	}

	// Miro uses different endpoints for different item types
	// We'll try the generic items endpoint first
	_, err := c.request(ctx, http.MethodDelete, "/boards/"+args.BoardID+"/items/"+args.ItemID, nil)
	if err != nil {
		return DeleteItemResult{
			Success: false,
			ItemID:  args.ItemID,
			Message: fmt.Sprintf("Failed to delete item: %v", err),
		}, err
	}

	// Invalidate cache for this item and items list
	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return DeleteItemResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "Item deleted successfully",
	}, nil
}

// ListAllItems retrieves all items from a board with automatic pagination.
func (c *Client) ListAllItems(ctx context.Context, args ListAllItemsArgs) (ListAllItemsResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return ListAllItemsResult{}, err
	}

	maxItems := effectiveListAllMax(args.MaxItems)
	allItems, pageCount, truncated, err := c.collectAllItems(ctx, args, maxItems)
	if err != nil {
		return ListAllItemsResult{}, err
	}

	return ListAllItemsResult{
		Items:      allItems,
		Count:      len(allItems),
		TotalPages: pageCount,
		Truncated:  truncated,
		Message:    formatListAllMessage(len(allItems), pageCount, truncated),
	}, nil
}

// effectiveListAllMax resolves the effective per-call cap, honoring the default
// when the caller passed 0 and clamping the upper bound.
func effectiveListAllMax(requested int) int {
	if requested == 0 {
		return DefaultListAllMaxItems
	}
	if requested > MaxListAllItems {
		return MaxListAllItems
	}
	return requested
}

// collectAllItems pages through ListItems, accumulating up to maxItems and
// reporting whether the result was truncated at the cap.
func (c *Client) collectAllItems(ctx context.Context, args ListAllItemsArgs, maxItems int) ([]ItemSummary, int, bool, error) {
	var allItems []ItemSummary
	cursor := ""
	pageCount := 0

	for {
		result, err := c.ListItems(ctx, ListItemsArgs{
			BoardID:     args.BoardID,
			Type:        args.Type,
			Limit:       MaxItemLimitExtended,
			Cursor:      cursor,
			DetailLevel: args.DetailLevel,
		})
		if err != nil {
			return nil, 0, false, err
		}

		pageCount++
		allItems = append(allItems, result.Items...)

		if len(allItems) >= maxItems {
			return allItems[:maxItems], pageCount, true, nil
		}
		if !result.HasMore || result.Cursor == "" {
			return allItems, pageCount, false, nil
		}
		cursor = result.Cursor
	}
}

// formatListAllMessage produces the user-facing message for a ListAllItems result.
func formatListAllMessage(count, pageCount int, truncated bool) string {
	if truncated {
		return fmt.Sprintf("Retrieved %d items (truncated at max_items limit)", count)
	}
	return fmt.Sprintf("Retrieved %d items in %d pages", count, pageCount)
}
