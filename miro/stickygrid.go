package miro

import (
	"context"
	"fmt"
)

// =============================================================================
// Composite Create Operations
// =============================================================================

// stickyGridDefaults applies grid column and spacing defaults to args.
func stickyGridDefaults(args CreateStickyGridArgs) (columns int, spacing float64) {
	columns = args.Columns
	if columns <= 0 {
		columns = 3
	}
	spacing = args.Spacing
	if spacing == 0 {
		spacing = 220
	}
	return columns, spacing
}

// buildStickyGridItems converts contents into bulk-create items at the
// computed grid positions.
func buildStickyGridItems(args CreateStickyGridArgs, columns int, spacing float64) []BulkCreateItem {
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
	return items
}

// batchEnd returns the exclusive end index for a batch starting at i.
func batchEnd(i, batchSize, total int) int {
	end := i + batchSize
	if end > total {
		end = total
	}
	return end
}

// bulkCreateOneBatch creates a single batch and returns its item IDs.
func (c *Client) bulkCreateOneBatch(ctx context.Context, boardID string, items []BulkCreateItem) ([]string, error) {
	result, err := c.BulkCreate(ctx, BulkCreateArgs{
		BoardID: boardID,
		Items:   items,
	})
	if err != nil {
		return nil, err
	}
	return result.ItemIDs, nil
}

// bulkCreateInBatches calls BulkCreate in batches of batchSize and returns
// all created item IDs. An error after the first batch yields partial
// results; an error on the very first batch propagates.
func (c *Client) bulkCreateInBatches(ctx context.Context, boardID string, items []BulkCreateItem, batchSize int) ([]string, error) {
	var allIDs []string
	for i := 0; i < len(items); i += batchSize {
		ids, err := c.bulkCreateOneBatch(ctx, boardID, items[i:batchEnd(i, batchSize, len(items))])
		if err != nil {
			if len(allIDs) > 0 {
				return allIDs, nil
			}
			return nil, err
		}
		allIDs = append(allIDs, ids...)
	}
	return allIDs, nil
}

// CreateStickyGrid creates multiple sticky notes in a grid layout.
func (c *Client) CreateStickyGrid(ctx context.Context, args CreateStickyGridArgs) (CreateStickyGridResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return CreateStickyGridResult{}, err
	}
	if len(args.Contents) == 0 {
		return CreateStickyGridResult{}, fmt.Errorf("at least one content item is required")
	}
	if len(args.Contents) > 50 {
		return CreateStickyGridResult{}, fmt.Errorf("maximum 50 stickies per grid")
	}

	columns, spacing := stickyGridDefaults(args)
	items := buildStickyGridItems(args, columns, spacing)
	allIDs, err := c.bulkCreateInBatches(ctx, args.BoardID, items, 20)
	if err != nil {
		return CreateStickyGridResult{}, err
	}

	rows := (len(args.Contents) + columns - 1) / columns
	return CreateStickyGridResult{
		Created:  len(allIDs),
		ItemIDs:  allIDs,
		ItemURLs: BuildItemURLs(args.BoardID, allIDs),
		Rows:     rows,
		Columns:  columns,
		Message:  fmt.Sprintf("Created %d stickies in %dx%d grid", len(allIDs), columns, rows),
	}, nil
}
