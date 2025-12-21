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
// Item Operations
// =============================================================================

// ListItems retrieves items from a board.
func (c *Client) ListItems(ctx context.Context, args ListItemsArgs) (ListItemsResult, error) {
	if args.BoardID == "" {
		return ListItemsResult{}, fmt.Errorf("board_id is required")
	}

	params := url.Values{}
	if args.Type != "" {
		params.Set("type", args.Type)
	}
	limit := 50
	if args.Limit > 0 && args.Limit <= 100 {
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

	// Parse items into summaries
	items := make([]ItemSummary, 0, len(resp.Data))
	for _, raw := range resp.Data {
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
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &base); err != nil {
			continue
		}

		item := ItemSummary{
			ID:       base.ID,
			Type:     base.Type,
			Content:  base.Data.Content,
			ParentID: base.ParentID,
		}
		if base.Position != nil {
			item.X = base.Position.X
			item.Y = base.Position.Y
		}
		items = append(items, item)
	}

	return ListItemsResult{
		Items:   items,
		Count:   len(items),
		HasMore: resp.Cursor != "",
		Cursor:  resp.Cursor,
	}, nil
}

// GetItem retrieves detailed information about a specific item.
func (c *Client) GetItem(ctx context.Context, args GetItemArgs) (GetItemResult, error) {
	if args.BoardID == "" {
		return GetItemResult{}, fmt.Errorf("board_id is required")
	}
	if args.ItemID == "" {
		return GetItemResult{}, fmt.Errorf("item_id is required")
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

	return result, nil
}

// UpdateItem updates an existing item.
func (c *Client) UpdateItem(ctx context.Context, args UpdateItemArgs) (UpdateItemResult, error) {
	if args.BoardID == "" {
		return UpdateItemResult{}, fmt.Errorf("board_id is required")
	}
	if args.ItemID == "" {
		return UpdateItemResult{}, fmt.Errorf("item_id is required")
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
		reqBody["style"] = map[string]interface{}{
			"fillColor": *args.Color,
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

	return UpdateItemResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "Item updated successfully",
	}, nil
}

// DeleteItem deletes an item from a board.
func (c *Client) DeleteItem(ctx context.Context, args DeleteItemArgs) (DeleteItemResult, error) {
	if args.BoardID == "" {
		return DeleteItemResult{}, fmt.Errorf("board_id is required")
	}
	if args.ItemID == "" {
		return DeleteItemResult{}, fmt.Errorf("item_id is required")
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

	return DeleteItemResult{
		Success: true,
		ItemID:  args.ItemID,
		Message: "Item deleted successfully",
	}, nil
}

// BulkCreate creates multiple items in one operation.
func (c *Client) BulkCreate(ctx context.Context, args BulkCreateArgs) (BulkCreateResult, error) {
	if args.BoardID == "" {
		return BulkCreateResult{}, fmt.Errorf("board_id is required")
	}
	if len(args.Items) == 0 {
		return BulkCreateResult{}, fmt.Errorf("at least one item is required")
	}
	if len(args.Items) > 20 {
		return BulkCreateResult{}, fmt.Errorf("maximum 20 items per bulk operation")
	}

	// Create items sequentially (Miro doesn't have a true bulk API)
	var itemIDs []string
	var errors []string

	for i, item := range args.Items {
		var id string
		var err error

		switch item.Type {
		case "sticky_note":
			result, e := c.CreateSticky(ctx, CreateStickyArgs{
				BoardID:  args.BoardID,
				Content:  item.Content,
				X:        item.X,
				Y:        item.Y,
				Color:    item.Color,
				Width:    item.Width,
				ParentID: item.ParentID,
			})
			id, err = result.ID, e

		case "shape":
			result, e := c.CreateShape(ctx, CreateShapeArgs{
				BoardID:  args.BoardID,
				Shape:    item.Shape,
				Content:  item.Content,
				X:        item.X,
				Y:        item.Y,
				Width:    item.Width,
				Height:   item.Height,
				Color:    item.Color,
				ParentID: item.ParentID,
			})
			id, err = result.ID, e

		case "text":
			result, e := c.CreateText(ctx, CreateTextArgs{
				BoardID:  args.BoardID,
				Content:  item.Content,
				X:        item.X,
				Y:        item.Y,
				Width:    item.Width,
				ParentID: item.ParentID,
			})
			id, err = result.ID, e

		default:
			err = fmt.Errorf("unsupported item type: %s", item.Type)
		}

		if err != nil {
			errors = append(errors, fmt.Sprintf("item %d: %v", i+1, err))
		} else if id != "" {
			itemIDs = append(itemIDs, id)
		}
	}

	return BulkCreateResult{
		Created: len(itemIDs),
		ItemIDs: itemIDs,
		Errors:  errors,
		Message: fmt.Sprintf("Created %d of %d items", len(itemIDs), len(args.Items)),
	}, nil
}

// ListAllItems retrieves all items from a board with automatic pagination.
func (c *Client) ListAllItems(ctx context.Context, args ListAllItemsArgs) (ListAllItemsResult, error) {
	if args.BoardID == "" {
		return ListAllItemsResult{}, fmt.Errorf("board_id is required")
	}

	maxItems := args.MaxItems
	if maxItems == 0 {
		maxItems = 500
	}
	if maxItems > 10000 {
		maxItems = 10000
	}

	var allItems []ItemSummary
	cursor := ""
	pageCount := 0
	truncated := false

	for {
		result, err := c.ListItems(ctx, ListItemsArgs{
			BoardID: args.BoardID,
			Type:    args.Type,
			Limit:   100, // Max per page
			Cursor:  cursor,
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

// SearchBoard searches for items containing specific text.
func (c *Client) SearchBoard(ctx context.Context, args SearchBoardArgs) (SearchBoardResult, error) {
	if args.BoardID == "" {
		return SearchBoardResult{}, fmt.Errorf("board_id is required")
	}
	if args.Query == "" {
		return SearchBoardResult{}, fmt.Errorf("query is required")
	}

	limit := 50
	if args.Limit > 0 && args.Limit < 50 {
		limit = args.Limit
	}

	// Fetch items from the board
	params := url.Values{}
	if args.Type != "" {
		params.Set("type", args.Type)
	}
	params.Set("limit", strconv.Itoa(limit))

	path := "/boards/" + args.BoardID + "/items"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return SearchBoardResult{}, err
	}

	var resp struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return SearchBoardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Search through items for matching content
	queryLower := strings.ToLower(args.Query)
	var matches []ItemMatch

	for _, raw := range resp.Data {
		var item struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Position *struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
			} `json:"position"`
			Data struct {
				Content string `json:"content"`
				Title   string `json:"title"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}

		// Check content and title for matches
		content := item.Data.Content
		if content == "" {
			content = item.Data.Title
		}

		if content != "" && strings.Contains(strings.ToLower(content), queryLower) {
			match := ItemMatch{
				ID:      item.ID,
				Type:    item.Type,
				Content: content,
				Snippet: createSnippet(content, args.Query, 50),
			}
			if item.Position != nil {
				match.X = item.Position.X
				match.Y = item.Position.Y
			}
			matches = append(matches, match)
		}
	}

	message := fmt.Sprintf("Found %d items matching '%s'", len(matches), args.Query)
	if len(matches) == 0 {
		message = fmt.Sprintf("No items found matching '%s'", args.Query)
	}

	return SearchBoardResult{
		Matches: matches,
		Count:   len(matches),
		Query:   args.Query,
		Message: message,
	}, nil
}

// createSnippet creates a text snippet around the matched query.
func createSnippet(content, query string, contextLen int) string {
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)

	idx := strings.Index(lowerContent, lowerQuery)
	if idx == -1 {
		return truncate(content, contextLen*2)
	}

	start := idx - contextLen
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + contextLen
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}
