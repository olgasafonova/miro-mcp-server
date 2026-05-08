package miro

import (
	"context"
	"fmt"
)

// =============================================================================
// Composite Create Operations
// =============================================================================

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
		Created:  len(allIDs),
		ItemIDs:  allIDs,
		ItemURLs: BuildItemURLs(args.BoardID, allIDs),
		Rows:     rows,
		Columns:  columns,
		Message:  fmt.Sprintf("Created %d stickies in %dx%d grid", len(allIDs), columns, rows),
	}, nil
}
