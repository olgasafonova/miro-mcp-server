package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// =============================================================================
// Document Operations - Create, Get, Update
// =============================================================================

// CreateDocument creates a document on a board from a URL.
func (c *Client) CreateDocument(ctx context.Context, args CreateDocumentArgs) (CreateDocumentResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateDocumentResult{}, err
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

	// Invalidate items list cache
	c.cache.InvalidatePrefix("items:" + args.BoardID)

	return CreateDocumentResult{
		ID:      doc.ID,
		ItemURL: BuildItemURL(args.BoardID, doc.ID),
		Title:   title,
		Message: fmt.Sprintf("Added document '%s'", truncate(title, 30)),
	}, nil
}

// GetDocument retrieves details of a specific document item.
func (c *Client) GetDocument(ctx context.Context, args GetDocumentArgs) (GetDocumentResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetDocumentResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return GetDocumentResult{}, err
	}

	cacheKey := CacheKeyItem(args.BoardID, args.ItemID)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if result, ok := cached.(GetDocumentResult); ok {
			return result, nil
		}
	}

	respBody, err := c.request(ctx, http.MethodGet, "/boards/"+args.BoardID+"/documents/"+args.ItemID, nil)
	if err != nil {
		return GetDocumentResult{}, err
	}

	var resp struct {
		ID       string `json:"id"`
		Position *struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"position"`
		Geometry *struct {
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"geometry"`
		Data struct {
			Title       string `json:"title"`
			DocumentURL string `json:"documentUrl"`
		} `json:"data"`
		ParentID string `json:"parentId"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return GetDocumentResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	result := GetDocumentResult{
		ID:          resp.ID,
		Title:       resp.Data.Title,
		DocumentURL: resp.Data.DocumentURL,
		ParentID:    resp.ParentID,
		Message:     "Document retrieved successfully",
	}
	if resp.Position != nil {
		result.X = resp.Position.X
		result.Y = resp.Position.Y
	}
	if resp.Geometry != nil {
		result.Width = resp.Geometry.Width
		result.Height = resp.Geometry.Height
	}

	c.cache.Set(cacheKey, result, c.cacheConfig.ItemTTL)

	return result, nil
}

// buildUpdateDocumentBody assembles the PATCH body for a document update.
func buildUpdateDocumentBody(args UpdateDocumentArgs) map[string]interface{} {
	reqBody := make(map[string]interface{})

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

	if pos := buildPositionSection(args.X, args.Y); pos != nil {
		reqBody["position"] = pos
	}
	if args.Width != nil {
		reqBody["geometry"] = map[string]interface{}{"width": *args.Width}
	}
	applyParentField(reqBody, args.ParentID)
	return reqBody
}

// UpdateDocument updates a document using the dedicated documents endpoint.
func (c *Client) UpdateDocument(ctx context.Context, args UpdateDocumentArgs) (UpdateDocumentResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return UpdateDocumentResult{}, err
	}
	if err := ValidateItemID(args.ItemID); err != nil {
		return UpdateDocumentResult{}, fmt.Errorf("invalid item_id: %w", err)
	}

	reqBody := buildUpdateDocumentBody(args)
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
