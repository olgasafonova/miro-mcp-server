package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// Create Operations - Stickies, Shapes, Text, Connectors, Frames
// =============================================================================

// CreateSticky creates a sticky note on a board.
func (c *Client) CreateSticky(ctx context.Context, args CreateStickyArgs) (CreateStickyResult, error) {
	if args.BoardID == "" {
		return CreateStickyResult{}, fmt.Errorf("board_id is required")
	}
	if args.Content == "" {
		return CreateStickyResult{}, fmt.Errorf("content is required")
	}

	// Build request body
	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"content": args.Content,
			"shape":   "square",
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
	}

	// Add style if color specified
	if args.Color != "" {
		reqBody["style"] = map[string]interface{}{
			"fillColor": normalizeStickyColor(args.Color),
		}
	}

	// Add geometry if width specified
	if args.Width > 0 {
		reqBody["geometry"] = map[string]interface{}{
			"width": args.Width,
		}
	}

	// Add parent if specified
	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/sticky_notes", reqBody)
	if err != nil {
		return CreateStickyResult{}, err
	}

	var sticky StickyNote
	if err := json.Unmarshal(respBody, &sticky); err != nil {
		return CreateStickyResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CreateStickyResult{
		ID:      sticky.ID,
		Content: sticky.Data.Content,
		Color:   sticky.Style.FillColor,
		Message: fmt.Sprintf("Created sticky note '%s'", truncate(args.Content, 30)),
	}, nil
}

// CreateShape creates a shape on a board.
func (c *Client) CreateShape(ctx context.Context, args CreateShapeArgs) (CreateShapeResult, error) {
	if args.BoardID == "" {
		return CreateShapeResult{}, fmt.Errorf("board_id is required")
	}
	if args.Shape == "" {
		return CreateShapeResult{}, fmt.Errorf("shape type is required")
	}

	// Default dimensions
	width := args.Width
	if width == 0 {
		width = 200
	}
	height := args.Height
	if height == 0 {
		height = 200
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"shape":   args.Shape,
			"content": args.Content,
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
		"geometry": map[string]interface{}{
			"width":  width,
			"height": height,
		},
	}

	if args.Color != "" {
		reqBody["style"] = map[string]interface{}{
			"fillColor": args.Color,
		}
	}

	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/shapes", reqBody)
	if err != nil {
		return CreateShapeResult{}, err
	}

	var shape Shape
	if err := json.Unmarshal(respBody, &shape); err != nil {
		return CreateShapeResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CreateShapeResult{
		ID:      shape.ID,
		Shape:   shape.Data.Shape,
		Content: shape.Data.Content,
		Message: fmt.Sprintf("Created %s shape", args.Shape),
	}, nil
}

// CreateText creates a text item on a board.
func (c *Client) CreateText(ctx context.Context, args CreateTextArgs) (CreateTextResult, error) {
	if args.BoardID == "" {
		return CreateTextResult{}, fmt.Errorf("board_id is required")
	}
	if args.Content == "" {
		return CreateTextResult{}, fmt.Errorf("content is required")
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"content": args.Content,
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
	}

	style := make(map[string]interface{})
	if args.FontSize > 0 {
		style["fontSize"] = strconv.Itoa(args.FontSize)
	}
	if args.Color != "" {
		style["color"] = args.Color
	}
	if len(style) > 0 {
		reqBody["style"] = style
	}

	if args.Width > 0 {
		reqBody["geometry"] = map[string]interface{}{
			"width": args.Width,
		}
	}

	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/texts", reqBody)
	if err != nil {
		return CreateTextResult{}, err
	}

	var text TextItem
	if err := json.Unmarshal(respBody, &text); err != nil {
		return CreateTextResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CreateTextResult{
		ID:      text.ID,
		Content: text.Data.Content,
		Message: fmt.Sprintf("Created text '%s'", truncate(args.Content, 30)),
	}, nil
}

// ListConnectors returns a list of connectors on a board.
func (c *Client) ListConnectors(ctx context.Context, args ListConnectorsArgs) (ListConnectorsResult, error) {
	if args.BoardID == "" {
		return ListConnectorsResult{}, fmt.Errorf("board_id is required")
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit < 10 {
		limit = 10 // Miro API minimum for connectors
	}
	if limit > 100 {
		limit = 100
	}

	path := fmt.Sprintf("/boards/%s/connectors?limit=%d", args.BoardID, limit)
	if args.Cursor != "" {
		path += "&cursor=" + args.Cursor
	}

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ListConnectorsResult{}, err
	}

	var resp struct {
		Data   []Connector `json:"data"`
		Cursor string      `json:"cursor,omitempty"`
		Total  int         `json:"total,omitempty"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ListConnectorsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	connectors := make([]ConnectorSummary, len(resp.Data))
	for i, c := range resp.Data {
		connectors[i] = ConnectorSummary{
			ID:          c.ID,
			StartItemID: c.StartItem.ItemID,
			EndItemID:   c.EndItem.ItemID,
			Style:       c.Shape,
			Caption:     extractCaption(c.Captions),
		}
	}

	hasMore := resp.Cursor != ""
	return ListConnectorsResult{
		Connectors: connectors,
		Count:      len(connectors),
		HasMore:    hasMore,
		Cursor:     resp.Cursor,
		Message:    fmt.Sprintf("Found %d connectors", len(connectors)),
	}, nil
}

// extractCaption gets the text from the first caption if present.
func extractCaption(captions []Caption) string {
	if len(captions) > 0 {
		return captions[0].Content
	}
	return ""
}

// GetConnector retrieves a specific connector by ID.
func (c *Client) GetConnector(ctx context.Context, args GetConnectorArgs) (GetConnectorResult, error) {
	if args.BoardID == "" {
		return GetConnectorResult{}, fmt.Errorf("board_id is required")
	}
	if args.ConnectorID == "" {
		return GetConnectorResult{}, fmt.Errorf("connector_id is required")
	}

	path := fmt.Sprintf("/boards/%s/connectors/%s", args.BoardID, args.ConnectorID)
	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return GetConnectorResult{}, err
	}

	var connector Connector
	if err := json.Unmarshal(respBody, &connector); err != nil {
		return GetConnectorResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	result := GetConnectorResult{
		ID:          connector.ID,
		StartItemID: connector.StartItem.ItemID,
		EndItemID:   connector.EndItem.ItemID,
		Style:       connector.Shape,
		Caption:     extractCaption(connector.Captions),
		Message:     "Retrieved connector details",
	}

	// Extract style details if present
	if connector.Style.StartStrokeCap != "" {
		result.StartCap = connector.Style.StartStrokeCap
	}
	if connector.Style.EndStrokeCap != "" {
		result.EndCap = connector.Style.EndStrokeCap
	}
	if connector.Style.Color != "" {
		result.Color = connector.Style.Color
	}

	// Format timestamps
	if !connector.CreatedAt.IsZero() {
		result.CreatedAt = connector.CreatedAt.Format(time.RFC3339)
	}
	if !connector.ModifiedAt.IsZero() {
		result.ModifiedAt = connector.ModifiedAt.Format(time.RFC3339)
	}

	return result, nil
}

// CreateConnector creates a connector between two items.
func (c *Client) CreateConnector(ctx context.Context, args CreateConnectorArgs) (CreateConnectorResult, error) {
	if args.BoardID == "" {
		return CreateConnectorResult{}, fmt.Errorf("board_id is required")
	}
	if args.StartItemID == "" || args.EndItemID == "" {
		return CreateConnectorResult{}, fmt.Errorf("start_item_id and end_item_id are required")
	}

	// Default style
	style := args.Style
	if style == "" {
		style = "elbowed"
	}

	reqBody := map[string]interface{}{
		"startItem": map[string]interface{}{
			"id": args.StartItemID,
		},
		"endItem": map[string]interface{}{
			"id": args.EndItemID,
		},
		"shape": style,
	}

	connectorStyle := make(map[string]interface{})
	if args.StartCap != "" {
		connectorStyle["startStrokeCap"] = args.StartCap
	}
	if args.EndCap != "" {
		connectorStyle["endStrokeCap"] = args.EndCap
	} else {
		connectorStyle["endStrokeCap"] = "arrow" // Default arrow at end
	}
	if len(connectorStyle) > 0 {
		reqBody["style"] = connectorStyle
	}

	if args.Caption != "" {
		reqBody["captions"] = []map[string]interface{}{
			{"content": args.Caption},
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/connectors", reqBody)
	if err != nil {
		return CreateConnectorResult{}, err
	}

	var connector Connector
	if err := json.Unmarshal(respBody, &connector); err != nil {
		return CreateConnectorResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CreateConnectorResult{
		ID:      connector.ID,
		Message: "Created connector between items",
	}, nil
}

// UpdateConnector updates an existing connector.
func (c *Client) UpdateConnector(ctx context.Context, args UpdateConnectorArgs) (UpdateConnectorResult, error) {
	if args.BoardID == "" {
		return UpdateConnectorResult{}, fmt.Errorf("board_id is required")
	}
	if args.ConnectorID == "" {
		return UpdateConnectorResult{}, fmt.Errorf("connector_id is required")
	}

	reqBody := make(map[string]interface{})

	// Set shape (style) if provided
	if args.Style != "" {
		reqBody["shape"] = args.Style
	}

	// Build style object for caps and color
	connectorStyle := make(map[string]interface{})
	if args.StartCap != "" {
		connectorStyle["startStrokeCap"] = args.StartCap
	}
	if args.EndCap != "" {
		connectorStyle["endStrokeCap"] = args.EndCap
	}
	if args.Color != "" {
		connectorStyle["strokeColor"] = args.Color
	}
	if len(connectorStyle) > 0 {
		reqBody["style"] = connectorStyle
	}

	// Set caption if provided
	if args.Caption != "" {
		reqBody["captions"] = []map[string]interface{}{
			{"content": args.Caption},
		}
	}

	// If nothing to update, return error
	if len(reqBody) == 0 {
		return UpdateConnectorResult{}, fmt.Errorf("at least one update field is required")
	}

	path := fmt.Sprintf("/boards/%s/connectors/%s", args.BoardID, args.ConnectorID)

	_, err := c.request(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return UpdateConnectorResult{}, err
	}

	return UpdateConnectorResult{
		Success: true,
		ID:      args.ConnectorID,
		Message: "Connector updated successfully",
	}, nil
}

// DeleteConnector removes a connector from a board.
func (c *Client) DeleteConnector(ctx context.Context, args DeleteConnectorArgs) (DeleteConnectorResult, error) {
	if args.BoardID == "" {
		return DeleteConnectorResult{}, fmt.Errorf("board_id is required")
	}
	if args.ConnectorID == "" {
		return DeleteConnectorResult{}, fmt.Errorf("connector_id is required")
	}

	path := fmt.Sprintf("/boards/%s/connectors/%s", args.BoardID, args.ConnectorID)

	_, err := c.request(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return DeleteConnectorResult{
			Success: false,
			ID:      args.ConnectorID,
			Message: fmt.Sprintf("Failed to delete connector: %v", err),
		}, err
	}

	return DeleteConnectorResult{
		Success: true,
		ID:      args.ConnectorID,
		Message: "Connector deleted successfully",
	}, nil
}

// CreateFrame creates a frame container on a board.
func (c *Client) CreateFrame(ctx context.Context, args CreateFrameArgs) (CreateFrameResult, error) {
	if args.BoardID == "" {
		return CreateFrameResult{}, fmt.Errorf("board_id is required")
	}

	// Default dimensions
	width := args.Width
	if width == 0 {
		width = 800
	}
	height := args.Height
	if height == 0 {
		height = 600
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"title":  args.Title,
			"format": "custom",
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
		"geometry": map[string]interface{}{
			"width":  width,
			"height": height,
		},
	}

	if args.Color != "" {
		reqBody["style"] = map[string]interface{}{
			"fillColor": args.Color,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/frames", reqBody)
	if err != nil {
		return CreateFrameResult{}, err
	}

	var frame Frame
	if err := json.Unmarshal(respBody, &frame); err != nil {
		return CreateFrameResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CreateFrameResult{
		ID:      frame.ID,
		Title:   frame.Data.Title,
		Message: fmt.Sprintf("Created frame '%s'", args.Title),
	}, nil
}

// =============================================================================
// Create Operations - Cards, Images, Documents, Embeds
// =============================================================================

// CreateCard creates a card on a board.
func (c *Client) CreateCard(ctx context.Context, args CreateCardArgs) (CreateCardResult, error) {
	if args.BoardID == "" {
		return CreateCardResult{}, fmt.Errorf("board_id is required")
	}
	if args.Title == "" {
		return CreateCardResult{}, fmt.Errorf("title is required")
	}

	// Default width
	width := args.Width
	if width == 0 {
		width = 320
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"title":       args.Title,
			"description": args.Description,
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
		"geometry": map[string]interface{}{
			"width": width,
		},
	}

	if args.DueDate != "" {
		data := reqBody["data"].(map[string]interface{})
		data["dueDate"] = args.DueDate
	}

	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/cards", reqBody)
	if err != nil {
		return CreateCardResult{}, err
	}

	var card Card
	if err := json.Unmarshal(respBody, &card); err != nil {
		return CreateCardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CreateCardResult{
		ID:      card.ID,
		Title:   card.Data.Title,
		Message: fmt.Sprintf("Created card '%s'", truncate(args.Title, 30)),
	}, nil
}

// CreateImage creates an image on a board from a URL.
func (c *Client) CreateImage(ctx context.Context, args CreateImageArgs) (CreateImageResult, error) {
	if args.BoardID == "" {
		return CreateImageResult{}, fmt.Errorf("board_id is required")
	}
	if args.URL == "" {
		return CreateImageResult{}, fmt.Errorf("url is required")
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"url": args.URL,
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
	}

	if args.Title != "" {
		data := reqBody["data"].(map[string]interface{})
		data["title"] = args.Title
	}

	if args.Width > 0 {
		reqBody["geometry"] = map[string]interface{}{
			"width": args.Width,
		}
	}

	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/images", reqBody)
	if err != nil {
		return CreateImageResult{}, err
	}

	var image Image
	if err := json.Unmarshal(respBody, &image); err != nil {
		return CreateImageResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	title := image.Data.Title
	if title == "" {
		title = "image"
	}

	return CreateImageResult{
		ID:      image.ID,
		Title:   title,
		URL:     image.Data.ImageURL,
		Message: fmt.Sprintf("Added image '%s'", truncate(title, 30)),
	}, nil
}

// CreateDocument creates a document on a board from a URL.
func (c *Client) CreateDocument(ctx context.Context, args CreateDocumentArgs) (CreateDocumentResult, error) {
	if args.BoardID == "" {
		return CreateDocumentResult{}, fmt.Errorf("board_id is required")
	}
	if args.URL == "" {
		return CreateDocumentResult{}, fmt.Errorf("url is required")
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"url": args.URL,
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
	}

	if args.Title != "" {
		data := reqBody["data"].(map[string]interface{})
		data["title"] = args.Title
	}

	if args.Width > 0 {
		reqBody["geometry"] = map[string]interface{}{
			"width": args.Width,
		}
	}

	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/documents", reqBody)
	if err != nil {
		return CreateDocumentResult{}, err
	}

	var doc Document
	if err := json.Unmarshal(respBody, &doc); err != nil {
		return CreateDocumentResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	title := doc.Data.Title
	if title == "" {
		title = "document"
	}

	return CreateDocumentResult{
		ID:      doc.ID,
		Title:   title,
		Message: fmt.Sprintf("Added document '%s'", truncate(title, 30)),
	}, nil
}

// CreateEmbed creates an embedded content item on a board.
func (c *Client) CreateEmbed(ctx context.Context, args CreateEmbedArgs) (CreateEmbedResult, error) {
	if args.BoardID == "" {
		return CreateEmbedResult{}, fmt.Errorf("board_id is required")
	}
	if args.URL == "" {
		return CreateEmbedResult{}, fmt.Errorf("url is required")
	}

	mode := args.Mode
	if mode == "" {
		mode = "inline"
	}

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"url":  args.URL,
			"mode": mode,
		},
		"position": map[string]interface{}{
			"x":      args.X,
			"y":      args.Y,
			"origin": "center",
		},
	}

	// For embeds with fixed aspect ratio (like YouTube), only send width
	// Miro will calculate height automatically. Sending both causes an error.
	if args.Width > 0 {
		reqBody["geometry"] = map[string]interface{}{
			"width": args.Width,
		}
	} else if args.Height > 0 {
		reqBody["geometry"] = map[string]interface{}{
			"height": args.Height,
		}
	}
	// If neither specified, let Miro use defaults

	if args.ParentID != "" {
		reqBody["parent"] = map[string]interface{}{
			"id": args.ParentID,
		}
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/embeds", reqBody)
	if err != nil {
		return CreateEmbedResult{}, err
	}

	var embed Embed
	if err := json.Unmarshal(respBody, &embed); err != nil {
		return CreateEmbedResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return CreateEmbedResult{
		ID:       embed.ID,
		URL:      embed.Data.URL,
		Provider: embed.Data.ProviderName,
		Message:  fmt.Sprintf("Embedded content from %s", embed.Data.ProviderName),
	}, nil
}

// =============================================================================
// Composite Create Operations
// =============================================================================

// CreateStickyGrid creates multiple sticky notes in a grid layout.
func (c *Client) CreateStickyGrid(ctx context.Context, args CreateStickyGridArgs) (CreateStickyGridResult, error) {
	if args.BoardID == "" {
		return CreateStickyGridResult{}, fmt.Errorf("board_id is required")
	}
	if len(args.Contents) == 0 {
		return CreateStickyGridResult{}, fmt.Errorf("at least one content item is required")
	}
	if len(args.Contents) > 50 {
		return CreateStickyGridResult{}, fmt.Errorf("maximum 50 stickies per grid")
	}

	// Defaults
	columns := args.Columns
	if columns <= 0 {
		columns = 3
	}
	spacing := args.Spacing
	if spacing == 0 {
		spacing = 220
	}

	// Build items for bulk create
	items := make([]BulkCreateItem, len(args.Contents))
	for i, content := range args.Contents {
		row := i / columns
		col := i % columns
		items[i] = BulkCreateItem{
			Type:     "sticky_note",
			Content:  content,
			X:        args.StartX + float64(col)*spacing,
			Y:        args.StartY + float64(row)*spacing,
			Color:    args.Color,
			ParentID: args.ParentID,
		}
	}

	// Create in batches of 20
	var allIDs []string
	for i := 0; i < len(items); i += 20 {
		end := i + 20
		if end > len(items) {
			end = len(items)
		}

		result, err := c.BulkCreate(ctx, BulkCreateArgs{
			BoardID: args.BoardID,
			Items:   items[i:end],
		})
		if err != nil {
			// Return partial results if some succeeded
			if len(allIDs) > 0 {
				break
			}
			return CreateStickyGridResult{}, err
		}
		allIDs = append(allIDs, result.ItemIDs...)
	}

	rows := (len(args.Contents) + columns - 1) / columns

	return CreateStickyGridResult{
		Created: len(allIDs),
		ItemIDs: allIDs,
		Rows:    rows,
		Columns: columns,
		Message: fmt.Sprintf("Created %d stickies in %dx%d grid", len(allIDs), columns, rows),
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// normalizeStickyColor converts color names to Miro's expected format.
func normalizeStickyColor(color string) string {
	// Miro uses specific color names
	colorMap := map[string]string{
		"yellow":     "light_yellow",
		"green":      "light_green",
		"blue":       "light_blue",
		"pink":       "light_pink",
		"purple":     "violet",
		"orange":     "orange",
		"red":        "red",
		"gray":       "gray",
		"grey":       "gray",
		"cyan":       "cyan",
		"dark_green": "dark_green",
		"dark_blue":  "dark_blue",
		"black":      "black",
	}

	lower := strings.ToLower(color)
	if mapped, ok := colorMap[lower]; ok {
		return mapped
	}
	return color // Return as-is if not in map
}

// truncate shortens a string to max length with ellipsis.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
