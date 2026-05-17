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

	maxItems := clampBoardContentMaxItems(args.MaxItems)

	board, err := c.GetBoard(ctx, GetBoardArgs{BoardID: args.BoardID})
	if err != nil {
		return GetBoardContentResult{}, fmt.Errorf("failed to get board: %w", err)
	}

	allItems, err := c.ListAllItems(ctx, ListAllItemsArgs{
		BoardID:     args.BoardID,
		MaxItems:    maxItems,
		DetailLevel: "full",
	})
	if err != nil {
		return GetBoardContentResult{}, fmt.Errorf("failed to list items: %w", err)
	}

	agg := aggregateBoardItems(allItems.Items)
	frames := buildFrameHierarchy(allItems.Items)

	result := assembleBoardContentResult(board, allItems, agg, frames)

	if args.IncludeConnectors {
		result.Connectors = c.loadConnectorContexts(ctx, args.BoardID, agg.itemMap)
	}
	if args.IncludeTags {
		result.Tags = c.loadTagContexts(ctx, args.BoardID)
	}

	result.Message = buildBoardContentMessage(board.Name, allItems.Count, len(frames), len(result.Connectors), len(result.Tags))

	return result, nil
}

// clampBoardContentMaxItems applies the [1, 2000] window with a 500 default.
func clampBoardContentMaxItems(requested int) int {
	const (
		defaultMax = 500
		hardCap    = 2000
	)
	if requested <= 0 {
		return defaultMax
	}
	if requested > hardCap {
		return hardCap
	}
	return requested
}

// boardItemAggregation bundles the per-item rollups computed in a single pass
// over the board's items.
type boardItemAggregation struct {
	counts      map[string]int
	itemsByType ItemsByType
	itemMap     map[string]ItemSummary
	allText     []string
	totalChars  int
}

// aggregateBoardItems walks the items once, building counts, items-by-type,
// an ID->item lookup map (for connector enrichment), and the text rollup.
func aggregateBoardItems(items []ItemSummary) boardItemAggregation {
	agg := boardItemAggregation{
		counts:  make(map[string]int),
		itemMap: make(map[string]ItemSummary),
	}
	for _, item := range items {
		agg.counts[item.Type]++
		agg.itemMap[item.ID] = item
		if item.Content != "" {
			agg.allText = append(agg.allText, item.Content)
			agg.totalChars += len(item.Content)
		}
		appendItemByType(&agg.itemsByType, item)
	}
	return agg
}

// appendItemByType pushes item onto the matching slice in itemsByType,
// falling back to Other for unknown types.
func appendItemByType(by *ItemsByType, item ItemSummary) {
	switch item.Type {
	case "sticky_note":
		by.StickyNotes = append(by.StickyNotes, item)
	case "shape":
		by.Shapes = append(by.Shapes, item)
	case "text":
		by.Text = append(by.Text, item)
	case "card":
		by.Cards = append(by.Cards, item)
	case "image":
		by.Images = append(by.Images, item)
	case "document":
		by.Documents = append(by.Documents, item)
	case "embed":
		by.Embeds = append(by.Embeds, item)
	default:
		by.Other = append(by.Other, item)
	}
}

// childrenOf returns items whose ParentID matches the given frame ID.
func childrenOf(items []ItemSummary, frameID string) []ItemSummary {
	var children []ItemSummary
	for _, child := range items {
		if child.ParentID == frameID {
			children = append(children, child)
		}
	}
	return children
}

// frameContextFromItem builds a FrameContext from a frame item plus its
// child items.
func frameContextFromItem(item ItemSummary, items []ItemSummary) FrameContext {
	return FrameContext{
		ID:       item.ID,
		Title:    item.Content,
		X:        item.X,
		Y:        item.Y,
		Width:    item.Width,
		Height:   item.Height,
		Children: childrenOf(items, item.ID),
	}
}

// buildFrameHierarchy returns a FrameContext per frame item, with each frame's
// children populated (items whose ParentID points at the frame).
func buildFrameHierarchy(items []ItemSummary) []FrameContext {
	var frames []FrameContext
	for _, item := range items {
		if item.Type == "frame" {
			frames = append(frames, frameContextFromItem(item, items))
		}
	}
	return frames
}

// formatOptionalRFC3339 returns t formatted as RFC3339, or "" if t is zero.
func formatOptionalRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// assembleBoardContentResult builds the result struct from the components
// gathered upstream. Connectors and tags are filled in by the caller after
// this returns.
func assembleBoardContentResult(board GetBoardResult, allItems ListAllItemsResult, agg boardItemAggregation, frames []FrameContext) GetBoardContentResult {
	return GetBoardContentResult{
		ID:          board.ID,
		Name:        board.Name,
		Description: board.Description,
		ViewLink:    board.ViewLink,
		CreatedAt:   formatOptionalRFC3339(board.CreatedAt),
		ModifiedAt:  formatOptionalRFC3339(board.ModifiedAt),
		ItemCounts:  agg.counts,
		TotalItems:  allItems.Count,
		ItemsByType: agg.itemsByType,
		Frames:      frames,
		ContentSummary: ContentSummary{
			AllText:       agg.allText,
			UniqueEntries: len(agg.allText),
			TotalChars:    agg.totalChars,
		},
		Truncated: allItems.Truncated,
	}
}

// loadConnectorContexts fetches connectors for the board and projects them
// onto the lighter ConnectorContext shape used by GetBoardContent. Returns
// nil on fetch error (errors are intentionally swallowed; connectors are an
// enrichment, not load-bearing).
func (c *Client) loadConnectorContexts(ctx context.Context, boardID string, itemMap map[string]ItemSummary) []ConnectorContext {
	connectors, err := c.ListConnectors(ctx, ListConnectorsArgs{
		BoardID: boardID,
		Limit:   100,
	})
	if err != nil {
		return nil
	}
	out := make([]ConnectorContext, 0, len(connectors.Connectors))
	for _, conn := range connectors.Connectors {
		cc := ConnectorContext{
			ID:          conn.ID,
			StartItemID: conn.StartItemID,
			EndItemID:   conn.EndItemID,
			Caption:     conn.Caption,
		}
		if startItem, ok := itemMap[conn.StartItemID]; ok {
			cc.StartItemType = startItem.Type
		}
		if endItem, ok := itemMap[conn.EndItemID]; ok {
			cc.EndItemType = endItem.Type
		}
		out = append(out, cc)
	}
	return out
}

// loadTagContexts fetches tag definitions for the board. Per-item tag
// membership is intentionally not computed (would require an extra API call
// per tag). Returns nil on fetch error.
func (c *Client) loadTagContexts(ctx context.Context, boardID string) []TagContext {
	tags, err := c.ListTags(ctx, ListTagsArgs{BoardID: boardID})
	if err != nil {
		return nil
	}
	out := make([]TagContext, 0, len(tags.Tags))
	for _, tag := range tags.Tags {
		out = append(out, TagContext{
			ID:    tag.ID,
			Title: tag.Title,
			Color: tag.FillColor,
		})
	}
	return out
}

// buildBoardContentMessage formats the human-readable summary line.
// Frames / connectors / tags are appended only when present.
func buildBoardContentMessage(boardName string, totalItems, framesCount, connectorsCount, tagsCount int) string {
	parts := []string{fmt.Sprintf("Board '%s' has %d items", boardName, totalItems)}
	if framesCount > 0 {
		parts = append(parts, fmt.Sprintf("%d frames", framesCount))
	}
	if connectorsCount > 0 {
		parts = append(parts, fmt.Sprintf("%d connectors", connectorsCount))
	}
	if tagsCount > 0 {
		parts = append(parts, fmt.Sprintf("%d tags", tagsCount))
	}
	return strings.Join(parts, ", ")
}
