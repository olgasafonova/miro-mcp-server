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

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
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

	// Check if full details are requested
	fullDetails := strings.EqualFold(args.DetailLevel, "full")

	// Parse items into summaries
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

// parseItemSummary extracts an ItemSummary from raw JSON data.
// When fullDetails is true, additional fields are populated.
func parseItemSummary(raw json.RawMessage, fullDetails bool) ItemSummary {
	// Base structure for minimal fields
	var base struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Position *struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"position"`
		ParentID string `json:"parentId"`
		Data     struct {
			Content string `json:"content"`
			Title   string `json:"title"`
		} `json:"data"`
		// Extended fields (only parsed when fullDetails=true)
		Geometry *struct {
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"geometry"`
		Style *struct {
			FillColor   string `json:"fillColor"`
			TextAlign   string `json:"textAlign"`
			BorderColor string `json:"borderColor"`
			FontSize    string `json:"fontSize"`
		} `json:"style"`
		CreatedAt  string `json:"createdAt"`
		ModifiedAt string `json:"modifiedAt"`
		CreatedBy  *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"createdBy"`
		ModifiedBy *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"modifiedBy"`
	}

	if err := json.Unmarshal(raw, &base); err != nil {
		return ItemSummary{}
	}

	// Build minimal summary
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

	// Add extended fields when full details requested
	if fullDetails {
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
			item.CreatedBy = &UserInfo{
				ID:   base.CreatedBy.ID,
				Name: base.CreatedBy.Name,
			}
		}
		if base.ModifiedBy != nil {
			item.ModifiedBy = &UserInfo{
				ID:   base.ModifiedBy.ID,
				Name: base.ModifiedBy.Name,
			}
		}
	}

	return item
}

// GetItem retrieves detailed information about a specific item.
func (c *Client) GetItem(ctx context.Context, args GetItemArgs) (GetItemResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetItemResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return GetItemResult{}, err
	}

	// Check cache first
	cacheKey := CacheKeyItem(args.BoardID, args.ItemID)
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached.(GetItemResult), nil
	}

	respBody, err := c.request(ctx, http.MethodGet, "/boards/"+args.BoardID+"/items/"+args.ItemID, nil)
	if err != nil {
		return GetItemResult{}, err
	}

	// Parse generic item response
	var item struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Position *struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"position"`
		Geometry *struct {
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"geometry"`
		Data struct {
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

	if err := json.Unmarshal(respBody, &item); err != nil {
		return GetItemResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

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

	// Cache the result
	c.cache.Set(cacheKey, result, c.cacheConfig.ItemTTL)

	return result, nil
}

// UpdateItem updates an existing item.
func (c *Client) UpdateItem(ctx context.Context, args UpdateItemArgs) (UpdateItemResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateItemResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateItemResult{}, err
	}

	// Build update body - only include provided fields
	reqBody := make(map[string]interface{})

	if args.Content != nil {
		reqBody["data"] = map[string]interface{}{
			"content": *args.Content,
		}
	}

	if args.X != nil || args.Y != nil {
		pos := map[string]interface{}{"origin": "center"}
		if args.X != nil {
			pos["x"] = *args.X
		}
		if args.Y != nil {
			pos["y"] = *args.Y
		}
		reqBody["position"] = pos
	}

	if args.Width != nil || args.Height != nil {
		geom := make(map[string]interface{})
		if args.Width != nil {
			geom["width"] = *args.Width
		}
		if args.Height != nil {
			geom["height"] = *args.Height
		}
		reqBody["geometry"] = geom
	}

	if args.Color != nil {
		fillColor, err := normalizeColor(*args.Color)
		if err != nil {
			return UpdateItemResult{}, fmt.Errorf("color: %w", err)
		}
		reqBody["style"] = map[string]interface{}{
			"fillColor": fillColor,
		}
	}

	if args.ParentID != nil {
		if *args.ParentID == "" {
			reqBody["parent"] = nil // Remove from frame
		} else {
			reqBody["parent"] = map[string]interface{}{
				"id": *args.ParentID,
			}
		}
	}

	if len(reqBody) == 0 {
		return UpdateItemResult{
			Success: true,
			ItemID:  args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	_, err := c.request(ctx, http.MethodPatch, "/boards/"+args.BoardID+"/items/"+args.ItemID, reqBody)
	if err != nil {
		return UpdateItemResult{
			Success: false,
			ItemID:  args.ItemID,
			Message: fmt.Sprintf("Failed to update item: %v", err),
		}, err
	}

	// Invalidate cache for this item
	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateItemResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "Item updated successfully",
	}, nil
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

	maxItems := args.MaxItems
	if maxItems == 0 {
		maxItems = DefaultListAllMaxItems
	}
	if maxItems > MaxListAllItems {
		maxItems = MaxListAllItems
	}

	var allItems []ItemSummary
	cursor := ""
	pageCount := 0
	truncated := false

	for {
		result, err := c.ListItems(ctx, ListItemsArgs{
			BoardID:     args.BoardID,
			Type:        args.Type,
			Limit:       MaxItemLimitExtended, // Max per page
			Cursor:      cursor,
			DetailLevel: args.DetailLevel,
		})
		if err != nil {
			return ListAllItemsResult{}, err
		}

		pageCount++
		allItems = append(allItems, result.Items...)

		// Check if we've hit the max items limit
		if len(allItems) >= maxItems {
			allItems = allItems[:maxItems]
			truncated = true
			break
		}

		// Check if there are more pages
		if !result.HasMore || result.Cursor == "" {
			break
		}
		cursor = result.Cursor
	}

	message := fmt.Sprintf("Retrieved %d items in %d pages", len(allItems), pageCount)
	if truncated {
		message = fmt.Sprintf("Retrieved %d items (truncated at max_items limit)", len(allItems))
	}

	return ListAllItemsResult{
		Items:      allItems,
		Count:      len(allItems),
		TotalPages: pageCount,
		Truncated:  truncated,
		Message:    message,
	}, nil
}
