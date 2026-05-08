package miro

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// =============================================================================
// Board Summary and Content (rich aggregations for AI consumers)
// =============================================================================

// GetBoardSummary retrieves a board with item counts and statistics.
func (c *Client) GetBoardSummary(ctx context.Context, args GetBoardSummaryArgs) (GetBoardSummaryResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetBoardSummaryResult{}, err
	}

	// Get board details
	board, err := c.GetBoard(ctx, GetBoardArgs(args))
	if err != nil {
		return GetBoardSummaryResult{}, fmt.Errorf("failed to get board: %w", err)
	}

	// Get items (first 100)
	items, err := c.ListItems(ctx, ListItemsArgs{BoardID: args.BoardID, Limit: 100})
	if err != nil {
		return GetBoardSummaryResult{}, fmt.Errorf("failed to list items: %w", err)
	}

	// Count items by type
	counts := make(map[string]int)
	for _, item := range items.Items {
		counts[item.Type]++
	}

	// Get recent items (first 5)
	recentItems := items.Items
	if len(recentItems) > 5 {
		recentItems = recentItems[:5]
	}

	return GetBoardSummaryResult{
		ID:          board.ID,
		Name:        board.Name,
		Description: board.Description,
		ViewLink:    board.ViewLink,
		ItemCounts:  counts,
		TotalItems:  items.Count,
		RecentItems: recentItems,
		Message:     fmt.Sprintf("Board '%s' has %d items", board.Name, items.Count),
	}, nil
}

// GetBoardContent retrieves comprehensive board data for AI analysis.
// This is designed to provide rich, structured data that an AI agent can
// analyze to generate documentation, summaries, or insights.
func (c *Client) GetBoardContent(ctx context.Context, args GetBoardContentArgs) (GetBoardContentResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GetBoardContentResult{}, err
	}

	// Set defaults
	maxItems := args.MaxItems
	if maxItems <= 0 {
		maxItems = 500
	}
	if maxItems > 2000 {
		maxItems = 2000
	}

	// Default to including connectors and tags
	includeConnectors := true
	includeTags := true
	if !args.IncludeConnectors {
		includeConnectors = false
	}
	if !args.IncludeTags {
		includeTags = false
	}

	// Get board details
	board, err := c.GetBoard(ctx, GetBoardArgs{BoardID: args.BoardID})
	if err != nil {
		return GetBoardContentResult{}, fmt.Errorf("failed to get board: %w", err)
	}

	// Get all items with full details
	allItems, err := c.ListAllItems(ctx, ListAllItemsArgs{
		BoardID:     args.BoardID,
		MaxItems:    maxItems,
		DetailLevel: "full",
	})
	if err != nil {
		return GetBoardContentResult{}, fmt.Errorf("failed to list items: %w", err)
	}

	// Build item counts and organize by type
	counts := make(map[string]int)
	itemsByType := ItemsByType{}
	itemMap := make(map[string]ItemSummary) // For connector lookups
	var allText []string
	totalChars := 0

	for _, item := range allItems.Items {
		counts[item.Type]++
		itemMap[item.ID] = item

		// Extract text content
		if item.Content != "" {
			allText = append(allText, item.Content)
			totalChars += len(item.Content)
		}

		// Organize by type
		switch item.Type {
		case "sticky_note":
			itemsByType.StickyNotes = append(itemsByType.StickyNotes, item)
		case "shape":
			itemsByType.Shapes = append(itemsByType.Shapes, item)
		case "text":
			itemsByType.Text = append(itemsByType.Text, item)
		case "card":
			itemsByType.Cards = append(itemsByType.Cards, item)
		case "image":
			itemsByType.Images = append(itemsByType.Images, item)
		case "document":
			itemsByType.Documents = append(itemsByType.Documents, item)
		case "embed":
			itemsByType.Embeds = append(itemsByType.Embeds, item)
		default:
			itemsByType.Other = append(itemsByType.Other, item)
		}
	}

	// Build frame hierarchy
	var frames []FrameContext
	for _, item := range allItems.Items {
		if item.Type == "frame" {
			frame := FrameContext{
				ID:     item.ID,
				Title:  item.Content,
				X:      item.X,
				Y:      item.Y,
				Width:  item.Width,
				Height: item.Height,
			}
			// Find children (items with this frame as parent)
			for _, child := range allItems.Items {
				if child.ParentID == item.ID {
					frame.Children = append(frame.Children, child)
				}
			}
			frames = append(frames, frame)
		}
	}

	// Format timestamps
	createdAt := ""
	if !board.CreatedAt.IsZero() {
		createdAt = board.CreatedAt.Format(time.RFC3339)
	}
	modifiedAt := ""
	if !board.ModifiedAt.IsZero() {
		modifiedAt = board.ModifiedAt.Format(time.RFC3339)
	}

	result := GetBoardContentResult{
		ID:          board.ID,
		Name:        board.Name,
		Description: board.Description,
		ViewLink:    board.ViewLink,
		CreatedAt:   createdAt,
		ModifiedAt:  modifiedAt,
		ItemCounts:  counts,
		TotalItems:  allItems.Count,
		ItemsByType: itemsByType,
		Frames:      frames,
		ContentSummary: ContentSummary{
			AllText:       allText,
			UniqueEntries: len(allText),
			TotalChars:    totalChars,
		},
		Truncated: allItems.Truncated,
	}

	// Get connectors if requested
	if includeConnectors {
		connectors, err := c.ListConnectors(ctx, ListConnectorsArgs{
			BoardID: args.BoardID,
			Limit:   100,
		})
		if err == nil {
			for _, conn := range connectors.Connectors {
				cc := ConnectorContext{
					ID:          conn.ID,
					StartItemID: conn.StartItemID,
					EndItemID:   conn.EndItemID,
					Caption:     conn.Caption,
				}
				// Add item types for context
				if startItem, ok := itemMap[conn.StartItemID]; ok {
					cc.StartItemType = startItem.Type
				}
				if endItem, ok := itemMap[conn.EndItemID]; ok {
					cc.EndItemType = endItem.Type
				}
				result.Connectors = append(result.Connectors, cc)
			}
		}
	}

	// Get tags if requested
	if includeTags {
		tags, err := c.ListTags(ctx, ListTagsArgs{BoardID: args.BoardID})
		if err == nil && len(tags.Tags) > 0 {
			// For each tag, we'd need to check which items have it
			// This is expensive, so we just return tag definitions for now
			for _, tag := range tags.Tags {
				result.Tags = append(result.Tags, TagContext{
					ID:    tag.ID,
					Title: tag.Title,
					Color: tag.FillColor,
				})
			}
		}
	}

	// Build message
	parts := []string{fmt.Sprintf("Board '%s' has %d items", board.Name, allItems.Count)}
	if len(frames) > 0 {
		parts = append(parts, fmt.Sprintf("%d frames", len(frames)))
	}
	if len(result.Connectors) > 0 {
		parts = append(parts, fmt.Sprintf("%d connectors", len(result.Connectors)))
	}
	if len(result.Tags) > 0 {
		parts = append(parts, fmt.Sprintf("%d tags", len(result.Tags)))
	}
	result.Message = strings.Join(parts, ", ")

	return result, nil
}
