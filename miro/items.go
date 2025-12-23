package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
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
	if args.BoardID == "" {
		return DeleteItemResult{}, fmt.Errorf("board_id is required")
	}
	if args.ItemID == "" {
		return DeleteItemResult{}, fmt.Errorf("item_id is required")
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

// bulkResult holds the result of a single item creation in bulk operations.
type bulkResult struct {
	index int
	id    string
	err   error
}

// BulkCreate creates multiple items in one operation.
// Items are created in parallel using goroutines, with concurrency
// controlled by the client's semaphore (MaxConcurrentRequests).
func (c *Client) BulkCreate(ctx context.Context, args BulkCreateArgs) (BulkCreateResult, error) {
	if args.BoardID == "" {
		return BulkCreateResult{}, fmt.Errorf("board_id is required")
	}
	if len(args.Items) == 0 {
		return BulkCreateResult{}, fmt.Errorf("at least one item is required")
	}
	if len(args.Items) > MaxBulkItems {
		return BulkCreateResult{}, fmt.Errorf("maximum %d items per bulk operation", MaxBulkItems)
	}

	// Add timeout for bulk operations to prevent hanging
	ctx, cancel := context.WithTimeout(ctx, BulkOperationTimeout)
	defer cancel()

	// Create items in parallel - semaphore in request() limits actual concurrency
	results := make(chan bulkResult, len(args.Items))
	var wg sync.WaitGroup

	for i, item := range args.Items {
		wg.Add(1)
		go func(idx int, it BulkCreateItem) {
			defer wg.Done()

			var id string
			var err error

			switch it.Type {
			case "sticky_note":
				result, e := c.CreateSticky(ctx, CreateStickyArgs{
					BoardID:  args.BoardID,
					Content:  it.Content,
					X:        it.X,
					Y:        it.Y,
					Color:    it.Color,
					Width:    it.Width,
					ParentID: it.ParentID,
				})
				id, err = result.ID, e

			case "shape":
				result, e := c.CreateShape(ctx, CreateShapeArgs{
					BoardID:  args.BoardID,
					Shape:    it.Shape,
					Content:  it.Content,
					X:        it.X,
					Y:        it.Y,
					Width:    it.Width,
					Height:   it.Height,
					Color:    it.Color,
					ParentID: it.ParentID,
				})
				id, err = result.ID, e

			case "text":
				result, e := c.CreateText(ctx, CreateTextArgs{
					BoardID:  args.BoardID,
					Content:  it.Content,
					X:        it.X,
					Y:        it.Y,
					Width:    it.Width,
					ParentID: it.ParentID,
				})
				id, err = result.ID, e

			default:
				err = fmt.Errorf("unsupported item type: %s", it.Type)
			}

			results <- bulkResult{index: idx, id: id, err: err}
		}(i, item)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results maintaining order
	resultSlice := make([]bulkResult, len(args.Items))
	for r := range results {
		resultSlice[r.index] = r
	}

	// Extract IDs and errors
	var itemIDs []string
	var errors []string
	for _, r := range resultSlice {
		if r.err != nil {
			errors = append(errors, fmt.Sprintf("item %d: %v", r.index+1, r.err))
		} else if r.id != "" {
			itemIDs = append(itemIDs, r.id)
		}
	}

	return BulkCreateResult{
		Created: len(itemIDs),
		ItemIDs: itemIDs,
		Errors:  errors,
		Message: fmt.Sprintf("Created %d of %d items", len(itemIDs), len(args.Items)),
	}, nil
}

// BulkUpdate updates multiple items in one operation.
// Items are updated in parallel using goroutines, with concurrency
// controlled by the client's semaphore (MaxConcurrentRequests).
func (c *Client) BulkUpdate(ctx context.Context, args BulkUpdateArgs) (BulkUpdateResult, error) {
	if args.BoardID == "" {
		return BulkUpdateResult{}, fmt.Errorf("board_id is required")
	}
	if len(args.Items) == 0 {
		return BulkUpdateResult{}, fmt.Errorf("at least one item is required")
	}
	if len(args.Items) > MaxBulkItems {
		return BulkUpdateResult{}, fmt.Errorf("maximum %d items per bulk operation", MaxBulkItems)
	}

	// Add timeout for bulk operations to prevent hanging
	ctx, cancel := context.WithTimeout(ctx, BulkOperationTimeout)
	defer cancel()

	// Update items in parallel - semaphore in request() limits actual concurrency
	results := make(chan bulkResult, len(args.Items))
	var wg sync.WaitGroup

	for i, item := range args.Items {
		wg.Add(1)
		go func(idx int, it BulkUpdateItem) {
			defer wg.Done()

			// Build update args
			updateArgs := UpdateItemArgs{
				BoardID: args.BoardID,
				ItemID:  it.ItemID,
			}
			if it.Content != nil {
				updateArgs.Content = it.Content
			}
			if it.X != nil {
				updateArgs.X = it.X
			}
			if it.Y != nil {
				updateArgs.Y = it.Y
			}
			if it.Width != nil {
				updateArgs.Width = it.Width
			}
			if it.Height != nil {
				updateArgs.Height = it.Height
			}
			if it.Color != nil {
				updateArgs.Color = it.Color
			}
			if it.ParentID != nil {
				updateArgs.ParentID = it.ParentID
			}

			_, err := c.UpdateItem(ctx, updateArgs)
			results <- bulkResult{index: idx, id: it.ItemID, err: err}
		}(i, item)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results maintaining order
	resultSlice := make([]bulkResult, len(args.Items))
	for r := range results {
		resultSlice[r.index] = r
	}

	// Extract IDs and errors
	var itemIDs []string
	var errors []string
	for _, r := range resultSlice {
		if r.err != nil {
			errors = append(errors, fmt.Sprintf("item %d (%s): %v", r.index+1, r.id, r.err))
		} else if r.id != "" {
			itemIDs = append(itemIDs, r.id)
		}
	}

	return BulkUpdateResult{
		Updated: len(itemIDs),
		ItemIDs: itemIDs,
		Errors:  errors,
		Message: fmt.Sprintf("Updated %d of %d items", len(itemIDs), len(args.Items)),
	}, nil
}

// BulkDelete deletes multiple items in one operation.
// Items are deleted in parallel using goroutines, with concurrency
// controlled by the client's semaphore (MaxConcurrentRequests).
func (c *Client) BulkDelete(ctx context.Context, args BulkDeleteArgs) (BulkDeleteResult, error) {
	if args.BoardID == "" {
		return BulkDeleteResult{}, fmt.Errorf("board_id is required")
	}
	if len(args.ItemIDs) == 0 {
		return BulkDeleteResult{}, fmt.Errorf("at least one item_id is required")
	}
	if len(args.ItemIDs) > MaxBulkItems {
		return BulkDeleteResult{}, fmt.Errorf("maximum %d items per bulk operation", MaxBulkItems)
	}

	// Dry-run mode: return preview without deleting
	if args.DryRun {
		return BulkDeleteResult{
			Deleted: len(args.ItemIDs),
			ItemIDs: args.ItemIDs,
			Message: fmt.Sprintf("[DRY RUN] Would delete %d items from board %s", len(args.ItemIDs), args.BoardID),
		}, nil
	}

	// Add timeout for bulk operations to prevent hanging
	ctx, cancel := context.WithTimeout(ctx, BulkOperationTimeout)
	defer cancel()

	// Delete items in parallel - semaphore in request() limits actual concurrency
	results := make(chan bulkResult, len(args.ItemIDs))
	var wg sync.WaitGroup

	for i, itemID := range args.ItemIDs {
		wg.Add(1)
		go func(idx int, id string) {
			defer wg.Done()

			_, err := c.DeleteItem(ctx, DeleteItemArgs{
				BoardID: args.BoardID,
				ItemID:  id,
			})
			results <- bulkResult{index: idx, id: id, err: err}
		}(i, itemID)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results maintaining order
	resultSlice := make([]bulkResult, len(args.ItemIDs))
	for r := range results {
		resultSlice[r.index] = r
	}

	// Extract IDs and errors
	var itemIDs []string
	var errors []string
	for _, r := range resultSlice {
		if r.err != nil {
			errors = append(errors, fmt.Sprintf("item %d (%s): %v", r.index+1, r.id, r.err))
		} else if r.id != "" {
			itemIDs = append(itemIDs, r.id)
		}
	}

	return BulkDeleteResult{
		Deleted: len(itemIDs),
		ItemIDs: itemIDs,
		Errors:  errors,
		Message: fmt.Sprintf("Deleted %d of %d items", len(itemIDs), len(args.ItemIDs)),
	}, nil
}

// ListAllItems retrieves all items from a board with automatic pagination.
func (c *Client) ListAllItems(ctx context.Context, args ListAllItemsArgs) (ListAllItemsResult, error) {
	if args.BoardID == "" {
		return ListAllItemsResult{}, fmt.Errorf("board_id is required")
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
			BoardID: args.BoardID,
			Type:    args.Type,
			Limit:   MaxItemLimitExtended, // Max per page
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

	limit := DefaultSearchLimit
	if args.Limit > 0 && args.Limit < MaxSearchLimit {
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

// =============================================================================
// Type-Specific Update Operations
// =============================================================================

// UpdateSticky updates a sticky note using the dedicated sticky_notes endpoint.
func (c *Client) UpdateSticky(ctx context.Context, args UpdateStickyArgs) (UpdateStickyResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateStickyResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateStickyResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := make(map[string]interface{})

	// Build data section
	data := make(map[string]interface{})
	if args.Content != nil {
		data["content"] = *args.Content
	}
	if args.Shape != nil {
		data["shape"] = *args.Shape
	}
	if len(data) > 0 {
		reqBody["data"] = data
	}

	// Build style section
	if args.Color != nil {
		reqBody["style"] = map[string]interface{}{
			"fillColor": *args.Color,
		}
	}

	// Build position section
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

	// Build geometry section
	if args.Width != nil {
		reqBody["geometry"] = map[string]interface{}{
			"width": *args.Width,
		}
	}

	// Build parent section
	if args.ParentID != nil {
		if *args.ParentID == "" {
			reqBody["parent"] = nil
		} else {
			reqBody["parent"] = map[string]interface{}{"id": *args.ParentID}
		}
	}

	if len(reqBody) == 0 {
		return UpdateStickyResult{
			ID:      args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	path := "/boards/" + args.BoardID + "/sticky_notes/" + args.ItemID
	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateStickyResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Content string `json:"content"`
			Shape   string `json:"shape"`
		} `json:"data"`
		Style struct {
			FillColor string `json:"fillColor"`
		} `json:"style"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateStickyResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateStickyResult{
		ID:      resp.ID,
		Content: resp.Data.Content,
		Shape:   resp.Data.Shape,
		Color:   resp.Style.FillColor,
		Message: "Sticky note updated successfully",
	}, nil
}

// UpdateShape updates a shape using the dedicated shapes endpoint.
func (c *Client) UpdateShape(ctx context.Context, args UpdateShapeArgs) (UpdateShapeResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateShapeResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateShapeResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := make(map[string]interface{})

	// Build data section
	data := make(map[string]interface{})
	if args.Content != nil {
		data["content"] = *args.Content
	}
	if args.ShapeType != nil {
		data["shape"] = *args.ShapeType
	}
	if len(data) > 0 {
		reqBody["data"] = data
	}

	// Build style section
	style := make(map[string]interface{})
	if args.Color != nil {
		style["fillColor"] = *args.Color
	}
	if args.TextColor != nil {
		style["fontColor"] = *args.TextColor
	}
	if len(style) > 0 {
		reqBody["style"] = style
	}

	// Build position section
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

	// Build geometry section
	geom := make(map[string]interface{})
	if args.Width != nil {
		geom["width"] = *args.Width
	}
	if args.Height != nil {
		geom["height"] = *args.Height
	}
	if len(geom) > 0 {
		reqBody["geometry"] = geom
	}

	// Build parent section
	if args.ParentID != nil {
		if *args.ParentID == "" {
			reqBody["parent"] = nil
		} else {
			reqBody["parent"] = map[string]interface{}{"id": *args.ParentID}
		}
	}

	if len(reqBody) == 0 {
		return UpdateShapeResult{
			ID:      args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	path := "/boards/" + args.BoardID + "/shapes/" + args.ItemID
	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateShapeResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Content string `json:"content"`
			Shape   string `json:"shape"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateShapeResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateShapeResult{
		ID:        resp.ID,
		ShapeType: resp.Data.Shape,
		Content:   resp.Data.Content,
		Message:   "Shape updated successfully",
	}, nil
}

// UpdateText updates a text item using the dedicated text endpoint.
func (c *Client) UpdateText(ctx context.Context, args UpdateTextArgs) (UpdateTextResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateTextResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateTextResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := make(map[string]interface{})

	// Build data section
	if args.Content != nil {
		reqBody["data"] = map[string]interface{}{
			"content": *args.Content,
		}
	}

	// Build style section
	style := make(map[string]interface{})
	if args.FontSize != nil {
		style["fontSize"] = fmt.Sprintf("%d", *args.FontSize)
	}
	if args.TextAlign != nil {
		style["textAlign"] = *args.TextAlign
	}
	if args.Color != nil {
		style["color"] = *args.Color
	}
	if len(style) > 0 {
		reqBody["style"] = style
	}

	// Build position section
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

	// Build geometry section
	if args.Width != nil {
		reqBody["geometry"] = map[string]interface{}{
			"width": *args.Width,
		}
	}

	// Build parent section
	if args.ParentID != nil {
		if *args.ParentID == "" {
			reqBody["parent"] = nil
		} else {
			reqBody["parent"] = map[string]interface{}{"id": *args.ParentID}
		}
	}

	if len(reqBody) == 0 {
		return UpdateTextResult{
			ID:      args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	path := "/boards/" + args.BoardID + "/texts/" + args.ItemID
	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateTextResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
		Style struct {
			FontSize string `json:"fontSize"`
		} `json:"style"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateTextResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	var fontSize int
	fmt.Sscanf(resp.Style.FontSize, "%d", &fontSize)

	return UpdateTextResult{
		ID:       resp.ID,
		Content:  resp.Data.Content,
		FontSize: fontSize,
		Message:  "Text updated successfully",
	}, nil
}

// UpdateCard updates a card using the dedicated cards endpoint.
func (c *Client) UpdateCard(ctx context.Context, args UpdateCardArgs) (UpdateCardResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateCardResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateCardResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := make(map[string]interface{})

	// Build data section
	data := make(map[string]interface{})
	if args.Title != nil {
		data["title"] = *args.Title
	}
	if args.Description != nil {
		data["description"] = *args.Description
	}
	if args.DueDate != nil {
		if *args.DueDate == "" {
			data["dueDate"] = nil
		} else {
			data["dueDate"] = *args.DueDate
		}
	}
	if len(data) > 0 {
		reqBody["data"] = data
	}

	// Build position section
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

	// Build geometry section
	if args.Width != nil {
		reqBody["geometry"] = map[string]interface{}{
			"width": *args.Width,
		}
	}

	// Build parent section
	if args.ParentID != nil {
		if *args.ParentID == "" {
			reqBody["parent"] = nil
		} else {
			reqBody["parent"] = map[string]interface{}{"id": *args.ParentID}
		}
	}

	if len(reqBody) == 0 {
		return UpdateCardResult{
			ID:      args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	path := "/boards/" + args.BoardID + "/cards/" + args.ItemID
	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateCardResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			DueDate     string `json:"dueDate"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateCardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateCardResult{
		ID:          resp.ID,
		Title:       resp.Data.Title,
		Description: resp.Data.Description,
		DueDate:     resp.Data.DueDate,
		Message:     "Card updated successfully",
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

// UpdateImage updates an image using the dedicated images endpoint.
func (c *Client) UpdateImage(ctx context.Context, args UpdateImageArgs) (UpdateImageResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateImageResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateImageResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := make(map[string]interface{})

	// Build data section
	data := make(map[string]interface{})
	if args.Title != nil {
		data["title"] = *args.Title
	}
	if args.URL != nil {
		data["url"] = *args.URL
	}
	if len(data) > 0 {
		reqBody["data"] = data
	}

	// Build position section
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

	// Build geometry section
	if args.Width != nil {
		reqBody["geometry"] = map[string]interface{}{
			"width": *args.Width,
		}
	}

	// Build parent section
	if args.ParentID != nil {
		if *args.ParentID == "" {
			reqBody["parent"] = nil
		} else {
			reqBody["parent"] = map[string]interface{}{"id": *args.ParentID}
		}
	}

	if len(reqBody) == 0 {
		return UpdateImageResult{
			ID:      args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	path := "/boards/" + args.BoardID + "/images/" + args.ItemID
	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateImageResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Title    string `json:"title"`
			ImageURL string `json:"imageUrl"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateImageResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateImageResult{
		ID:      resp.ID,
		Title:   resp.Data.Title,
		URL:     resp.Data.ImageURL,
		Message: "Image updated successfully",
	}, nil
}

// UpdateDocument updates a document using the dedicated documents endpoint.
func (c *Client) UpdateDocument(ctx context.Context, args UpdateDocumentArgs) (UpdateDocumentResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateDocumentResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateDocumentResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := make(map[string]interface{})

	// Build data section
	data := make(map[string]interface{})
	if args.Title != nil {
		data["title"] = *args.Title
	}
	if args.URL != nil {
		data["documentUrl"] = *args.URL
	}
	if len(data) > 0 {
		reqBody["data"] = data
	}

	// Build position section
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

	// Build geometry section
	if args.Width != nil {
		reqBody["geometry"] = map[string]interface{}{
			"width": *args.Width,
		}
	}

	// Build parent section
	if args.ParentID != nil {
		if *args.ParentID == "" {
			reqBody["parent"] = nil
		} else {
			reqBody["parent"] = map[string]interface{}{"id": *args.ParentID}
		}
	}

	if len(reqBody) == 0 {
		return UpdateDocumentResult{
			ID:      args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	path := "/boards/" + args.BoardID + "/documents/" + args.ItemID
	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateDocumentResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateDocumentResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateDocumentResult{
		ID:      resp.ID,
		Title:   resp.Data.Title,
		Message: "Document updated successfully",
	}, nil
}

// UpdateEmbed updates an embed using the dedicated embeds endpoint.
func (c *Client) UpdateEmbed(ctx context.Context, args UpdateEmbedArgs) (UpdateEmbedResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateEmbedResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateEmbedResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := make(map[string]interface{})

	// Build data section
	data := make(map[string]interface{})
	if args.URL != nil {
		data["url"] = *args.URL
	}
	if args.Mode != nil {
		data["mode"] = *args.Mode
	}
	if len(data) > 0 {
		reqBody["data"] = data
	}

	// Build position section
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

	// Build geometry section
	geom := make(map[string]interface{})
	if args.Width != nil {
		geom["width"] = *args.Width
	}
	if args.Height != nil {
		geom["height"] = *args.Height
	}
	if len(geom) > 0 {
		reqBody["geometry"] = geom
	}

	// Build parent section
	if args.ParentID != nil {
		if *args.ParentID == "" {
			reqBody["parent"] = nil
		} else {
			reqBody["parent"] = map[string]interface{}{"id": *args.ParentID}
		}
	}

	if len(reqBody) == 0 {
		return UpdateEmbedResult{
			ID:      args.ItemID,
			Message: "No changes specified",
		}, nil
	}

	path := "/boards/" + args.BoardID + "/embeds/" + args.ItemID
	respBody, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateEmbedResult{}, err
	}

	var resp struct {
		ID   string `json:"id"`
		Data struct {
			URL         string `json:"url"`
			ProviderURL string `json:"providerUrl"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return UpdateEmbedResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.cache.InvalidateItem(args.BoardID, args.ItemID)

	return UpdateEmbedResult{
		ID:       resp.ID,
		URL:      resp.Data.URL,
		Provider: resp.Data.ProviderURL,
		Message:  "Embed updated successfully",
	}, nil
}
