package miro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// BaseURL is the Miro API base URL
	BaseURL = "https://api.miro.com/v2"

	// DefaultTimeout for API requests
	DefaultTimeout = 30 * time.Second

	// MaxConcurrentRequests limits parallel API calls
	MaxConcurrentRequests = 5

	// Rate limit: 100,000 credits per minute, but we'll be conservative
	DefaultRateLimit = 100 // requests per minute per user
)

// Config holds Miro client configuration.
type Config struct {
	// AccessToken is the OAuth access token (required)
	AccessToken string

	// Timeout for HTTP requests (default 30s)
	Timeout time.Duration

	// UserAgent for API requests
	UserAgent string
}

// LoadConfig creates a Config from environment variables.
func LoadConfig() (*Config, error) {
	token := getEnv("MIRO_ACCESS_TOKEN", "")
	if token == "" {
		return nil, fmt.Errorf("MIRO_ACCESS_TOKEN environment variable is required")
	}

	timeout := DefaultTimeout
	if t := getEnv("MIRO_TIMEOUT", ""); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	return &Config{
		AccessToken: token,
		Timeout:     timeout,
		UserAgent:   getEnv("MIRO_USER_AGENT", "miro-mcp-server/1.0"),
	}, nil
}

// getEnv returns environment variable value or default.
func getEnv(key, defaultVal string) string {
	if val := lookupEnv(key); val != "" {
		return val
	}
	return defaultVal
}

// lookupEnv is a variable to allow testing.
var lookupEnv = func(key string) string {
	// Import os in the actual implementation
	return ""
}

// Client handles communication with the Miro API.
type Client struct {
	config     *Config
	httpClient *http.Client
	logger     *slog.Logger

	// Rate limiting with semaphore
	semaphore chan struct{}

	// Simple response cache
	cache    sync.Map
	cacheTTL time.Duration
}

// cacheEntry holds cached data with expiration.
type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// NewClient creates a new Miro API client.
func NewClient(config *Config, logger *slog.Logger) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger:    logger,
		semaphore: make(chan struct{}, MaxConcurrentRequests),
		cacheTTL:  2 * time.Minute,
	}
}

// request makes an authenticated request to the Miro API.
func (c *Client) request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	// Acquire semaphore slot (rate limiting)
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while waiting for rate limiter: %w", ctx.Err())
	}

	// Build request URL
	reqURL := BaseURL + path

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)
	req.Header.Set("User-Agent", c.config.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("API error [%d %s]: %s", resp.StatusCode, apiErr.Code, apiErr.Message)
		}
		return nil, fmt.Errorf("API error [%d]: %s", resp.StatusCode, string(respBody))
	}

	c.logger.Debug("API request completed",
		"method", method,
		"path", path,
		"status", resp.StatusCode,
	)

	return respBody, nil
}

// getCached retrieves a cached value if valid.
func (c *Client) getCached(key string) (interface{}, bool) {
	if entry, ok := c.cache.Load(key); ok {
		ce := entry.(*cacheEntry)
		if time.Now().Before(ce.expiresAt) {
			return ce.data, true
		}
		c.cache.Delete(key)
	}
	return nil, false
}

// setCache stores a value in the cache.
func (c *Client) setCache(key string, data interface{}) {
	c.cache.Store(key, &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.cacheTTL),
	})
}

// =============================================================================
// Board Operations
// =============================================================================

// ListBoards retrieves boards accessible to the user.
func (c *Client) ListBoards(ctx context.Context, args ListBoardsArgs) (ListBoardsResult, error) {
	// Build query parameters
	params := url.Values{}
	if args.TeamID != "" {
		params.Set("team_id", args.TeamID)
	}
	if args.Query != "" {
		params.Set("query", args.Query)
	}
	limit := 20
	if args.Limit > 0 && args.Limit <= 50 {
		limit = args.Limit
	}
	params.Set("limit", strconv.Itoa(limit))
	if args.Offset != "" {
		params.Set("offset", args.Offset)
	}

	path := "/boards"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	respBody, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ListBoardsResult{}, err
	}

	var resp struct {
		Data   []Board `json:"data"`
		Total  int     `json:"total,omitempty"`
		Size   int     `json:"size,omitempty"`
		Offset string  `json:"offset,omitempty"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ListBoardsResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to summaries
	boards := make([]BoardSummary, len(resp.Data))
	for i, b := range resp.Data {
		boards[i] = BoardSummary{
			ID:          b.ID,
			Name:        b.Name,
			Description: b.Description,
			ViewLink:    b.ViewLink,
		}
		if b.Team != nil {
			boards[i].TeamName = b.Team.Name
		}
	}

	return ListBoardsResult{
		Boards:  boards,
		Count:   len(boards),
		HasMore: resp.Offset != "" && len(resp.Data) >= limit,
		Offset:  resp.Offset,
	}, nil
}

// GetBoard retrieves a specific board by ID.
func (c *Client) GetBoard(ctx context.Context, args GetBoardArgs) (GetBoardResult, error) {
	if args.BoardID == "" {
		return GetBoardResult{}, fmt.Errorf("board_id is required")
	}

	// Check cache
	cacheKey := "board:" + args.BoardID
	if cached, ok := c.getCached(cacheKey); ok {
		return cached.(GetBoardResult), nil
	}

	respBody, err := c.request(ctx, http.MethodGet, "/boards/"+args.BoardID, nil)
	if err != nil {
		return GetBoardResult{}, err
	}

	var board Board
	if err := json.Unmarshal(respBody, &board); err != nil {
		return GetBoardResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	result := GetBoardResult{Board: board}

	// Cache the result
	c.setCache(cacheKey, result)

	return result, nil
}

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
// Read Operations
// =============================================================================

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

// =============================================================================
// Card Operations
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

// =============================================================================
// Image Operations
// =============================================================================

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

// =============================================================================
// Document Operations
// =============================================================================

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

// =============================================================================
// Embed Operations
// =============================================================================

// CreateEmbed creates an embedded content item on a board.
func (c *Client) CreateEmbed(ctx context.Context, args CreateEmbedArgs) (CreateEmbedResult, error) {
	if args.BoardID == "" {
		return CreateEmbedResult{}, fmt.Errorf("board_id is required")
	}
	if args.URL == "" {
		return CreateEmbedResult{}, fmt.Errorf("url is required")
	}

	// Default dimensions
	width := args.Width
	if width == 0 {
		width = 400
	}
	height := args.Height
	if height == 0 {
		height = 300
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
		"geometry": map[string]interface{}{
			"width":  width,
			"height": height,
		},
	}

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

	reqBody := map[string]interface{}{
		"title": args.Title,
	}

	if args.Color != "" {
		reqBody["fillColor"] = normalizeTagColor(args.Color)
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
	limit := 50
	if args.Limit > 0 && args.Limit <= 50 {
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

	message := fmt.Sprintf("Item has %d tags", len(resp.Data))
	if len(resp.Data) == 0 {
		message = "No tags on this item"
	}

	return GetItemTagsResult{
		Tags:    resp.Data,
		Count:   len(resp.Data),
		ItemID:  args.ItemID,
		Message: message,
	}, nil
}

// =============================================================================
// Pagination Operations
// =============================================================================

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

// normalizeStickyColor converts color names to Miro's expected format.
func normalizeStickyColor(color string) string {
	// Miro uses specific color names
	colorMap := map[string]string{
		"yellow":      "light_yellow",
		"green":       "light_green",
		"blue":        "light_blue",
		"pink":        "light_pink",
		"purple":      "violet",
		"orange":      "orange",
		"red":         "red",
		"gray":        "gray",
		"grey":        "gray",
		"cyan":        "cyan",
		"dark_green":  "dark_green",
		"dark_blue":   "dark_blue",
		"black":       "black",
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
