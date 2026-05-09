package miro

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// =============================================================================
// Bulk Operations
// =============================================================================

// bulkResult holds the result of a single item creation in bulk operations.
type bulkResult struct {
	index int
	id    string
	err   error
}

// errorCategory describes how a failed bulk item should be classified.
type errorCategory struct {
	errorType   string
	isRetriable bool
}

// apiStatusCategories maps known HTTP status codes to bulk error categories.
var apiStatusCategories = map[int]errorCategory{
	400: {"validation", false},
	401: {"auth", false},
	403: {"auth", false},
	404: {"not_found", false},
	429: {"rate_limit", true},
}

// networkErrorMarkers are substrings that indicate a network-level failure.
var networkErrorMarkers = []string{"connection", "timeout", "network", "EOF"}

// categorizeAPIStatus maps an HTTP status code to a bulk error category.
func categorizeAPIStatus(status int) errorCategory {
	if c, ok := apiStatusCategories[status]; ok {
		return c
	}
	if status >= 500 {
		return errorCategory{"server", true}
	}
	return errorCategory{"api", false}
}

// hasNetworkErrorMarker reports whether the error message looks network-related.
func hasNetworkErrorMarker(err error) bool {
	msg := err.Error()
	for _, marker := range networkErrorMarkers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

// categorizeBulkError analyzes an error and returns a BulkItemError with appropriate categorization.
func categorizeBulkError(index int, itemID string, err error) BulkItemError {
	bulkErr := BulkItemError{
		Index:   index,
		ItemID:  itemID,
		Message: err.Error(),
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		cat := categorizeAPIStatus(apiErr.StatusCode)
		bulkErr.StatusCode = apiErr.StatusCode
		bulkErr.ErrorType = cat.errorType
		bulkErr.IsRetriable = cat.isRetriable
		return bulkErr
	}

	var valErr *ValidationError
	if errors.As(err, &valErr) {
		bulkErr.ErrorType = "validation"
		bulkErr.IsRetriable = false
		return bulkErr
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		bulkErr.ErrorType = "timeout"
		bulkErr.IsRetriable = true
		return bulkErr
	}

	if hasNetworkErrorMarker(err) {
		bulkErr.ErrorType = "network"
		bulkErr.IsRetriable = true
		return bulkErr
	}

	bulkErr.ErrorType = "unknown"
	bulkErr.IsRetriable = false
	return bulkErr
}

// validateBulkSize enforces the bulk operation size constraints shared by all bulk methods.
func validateBulkSize(boardID string, count int, itemNoun string) error {
	if err := ValidateBoardID(boardID); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("at least one %s is required", itemNoun)
	}
	if count > MaxBulkItems {
		return fmt.Errorf("maximum %d items per bulk operation", MaxBulkItems)
	}
	return nil
}

// runBulkInParallel runs an action concurrently on each item and returns ordered results.
// The semaphore in request() limits actual concurrency.
func runBulkInParallel[T any](ctx context.Context, items []T, action func(context.Context, int, T) (string, error)) []bulkResult {
	results := make(chan bulkResult, len(items))
	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		go func(idx int, it T) {
			defer wg.Done()
			id, err := action(ctx, idx, it)
			results <- bulkResult{index: idx, id: id, err: err}
		}(i, item)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	out := make([]bulkResult, len(items))
	for r := range results {
		out[r.index] = r
	}
	return out
}

// bulkAggregation holds the aggregated outcome of a parallel bulk run.
type bulkAggregation struct {
	successIDs  []string
	errorMsgs   []string
	failedItems []BulkItemError
}

// processBulkResults partitions results into successes, formatted error messages, and categorized failures.
func processBulkResults(resultSlice []bulkResult, formatErr func(idx int, id string, err error) string) bulkAggregation {
	var agg bulkAggregation
	for _, r := range resultSlice {
		if r.err != nil {
			agg.errorMsgs = append(agg.errorMsgs, formatErr(r.index, r.id, r.err))
			agg.failedItems = append(agg.failedItems, categorizeBulkError(r.index, r.id, r.err))
		} else if r.id != "" {
			agg.successIDs = append(agg.successIDs, r.id)
		}
	}
	return agg
}

// retriableIndexes returns indexes of failed items that should be retried.
// Used by BulkCreate, where item IDs do not exist before creation.
func retriableIndexes(failed []BulkItemError) []int {
	var out []int
	for _, f := range failed {
		if f.IsRetriable {
			out = append(out, f.Index)
		}
	}
	return out
}

// retriableItemIDs returns IDs of failed items that should be retried.
// Used by BulkUpdate and BulkDelete, where the item ID is known up front.
func retriableItemIDs(failed []BulkItemError) []string {
	var out []string
	for _, f := range failed {
		if f.IsRetriable && f.ItemID != "" {
			out = append(out, f.ItemID)
		}
	}
	return out
}

// formatBulkMessage produces a "<verb> N of M items" summary with an optional retriable suffix.
func formatBulkMessage(verb string, success, total, retriable int) string {
	msg := fmt.Sprintf("%s %d of %d items", verb, success, total)
	if retriable > 0 {
		msg += fmt.Sprintf(". %d items can be retried", retriable)
	}
	return msg
}

// createBulkItem dispatches creation for a single bulk item to the matching typed Create* method.
func createBulkItem(ctx context.Context, c *Client, boardID string, item BulkCreateItem) (string, error) {
	switch item.Type {
	case "sticky_note":
		result, err := c.CreateSticky(ctx, CreateStickyArgs{
			BoardID:  boardID,
			Content:  item.Content,
			X:        item.X,
			Y:        item.Y,
			Color:    item.Color,
			Width:    item.Width,
			ParentID: item.ParentID,
		})
		return result.ID, err

	case "shape":
		result, err := c.CreateShape(ctx, CreateShapeArgs{
			BoardID:           boardID,
			Shape:             item.Shape,
			Content:           item.Content,
			X:                 item.X,
			Y:                 item.Y,
			Width:             item.Width,
			Height:            item.Height,
			Color:             item.Color,
			TextColor:         item.TextColor,
			TextAlign:         item.TextAlign,
			TextAlignVertical: item.TextAlignVertical,
			ParentID:          item.ParentID,
		})
		return result.ID, err

	case "text":
		result, err := c.CreateText(ctx, CreateTextArgs{
			BoardID:  boardID,
			Content:  item.Content,
			X:        item.X,
			Y:        item.Y,
			Width:    item.Width,
			ParentID: item.ParentID,
		})
		return result.ID, err

	default:
		return "", fmt.Errorf("unsupported item type: %s", item.Type)
	}
}

// buildBulkUpdateArgs constructs UpdateItemArgs from a bulk update item, copying only non-nil fields.
func buildBulkUpdateArgs(boardID string, item BulkUpdateItem) UpdateItemArgs {
	args := UpdateItemArgs{
		BoardID: boardID,
		ItemID:  item.ItemID,
	}
	if item.Content != nil {
		args.Content = item.Content
	}
	if item.X != nil {
		args.X = item.X
	}
	if item.Y != nil {
		args.Y = item.Y
	}
	if item.Width != nil {
		args.Width = item.Width
	}
	if item.Height != nil {
		args.Height = item.Height
	}
	if item.Color != nil {
		args.Color = item.Color
	}
	if item.ParentID != nil {
		args.ParentID = item.ParentID
	}
	return args
}

// BulkCreate creates multiple items in one operation.
// Items are created in parallel using goroutines, with concurrency
// controlled by the client's semaphore (MaxConcurrentRequests).
func (c *Client) BulkCreate(ctx context.Context, args BulkCreateArgs) (BulkCreateResult, error) {
	if err := validateBulkSize(args.BoardID, len(args.Items), "item"); err != nil {
		return BulkCreateResult{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, BulkOperationTimeout)
	defer cancel()

	resultSlice := runBulkInParallel(ctx, args.Items, func(ctx context.Context, _ int, item BulkCreateItem) (string, error) {
		return createBulkItem(ctx, c, args.BoardID, item)
	})

	agg := processBulkResults(resultSlice, func(idx int, _ string, err error) string {
		return fmt.Sprintf("item %d: %v", idx+1, err)
	})
	retriableIDs := retriableIndexes(agg.failedItems)

	return BulkCreateResult{
		Created:      len(agg.successIDs),
		ItemIDs:      agg.successIDs,
		ItemURLs:     BuildItemURLs(args.BoardID, agg.successIDs),
		Errors:       agg.errorMsgs,
		FailedItems:  agg.failedItems,
		RetriableIDs: retriableIDs,
		Message:      formatBulkMessage("Created", len(agg.successIDs), len(args.Items), len(retriableIDs)),
	}, nil
}

// BulkUpdate updates multiple items in one operation.
// Items are updated in parallel using goroutines, with concurrency
// controlled by the client's semaphore (MaxConcurrentRequests).
func (c *Client) BulkUpdate(ctx context.Context, args BulkUpdateArgs) (BulkUpdateResult, error) {
	if err := validateBulkSize(args.BoardID, len(args.Items), "item"); err != nil {
		return BulkUpdateResult{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, BulkOperationTimeout)
	defer cancel()

	resultSlice := runBulkInParallel(ctx, args.Items, func(ctx context.Context, _ int, item BulkUpdateItem) (string, error) {
		_, err := c.UpdateItem(ctx, buildBulkUpdateArgs(args.BoardID, item))
		return item.ItemID, err
	})

	agg := processBulkResults(resultSlice, func(idx int, id string, err error) string {
		return fmt.Sprintf("item %d (%s): %v", idx+1, id, err)
	})
	retriableIDs := retriableItemIDs(agg.failedItems)

	return BulkUpdateResult{
		Updated:      len(agg.successIDs),
		ItemIDs:      agg.successIDs,
		Errors:       agg.errorMsgs,
		FailedItems:  agg.failedItems,
		RetriableIDs: retriableIDs,
		Message:      formatBulkMessage("Updated", len(agg.successIDs), len(args.Items), len(retriableIDs)),
	}, nil
}

// BulkDelete deletes multiple items in one operation.
// Items are deleted in parallel using goroutines, with concurrency
// controlled by the client's semaphore (MaxConcurrentRequests).
func (c *Client) BulkDelete(ctx context.Context, args BulkDeleteArgs) (BulkDeleteResult, error) {
	if err := validateBulkSize(args.BoardID, len(args.ItemIDs), "item_id"); err != nil {
		return BulkDeleteResult{}, err
	}

	if args.DryRun {
		return BulkDeleteResult{
			Deleted: len(args.ItemIDs),
			ItemIDs: args.ItemIDs,
			Message: fmt.Sprintf("[DRY RUN] Would delete %d items from board %s", len(args.ItemIDs), args.BoardID),
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, BulkOperationTimeout)
	defer cancel()

	resultSlice := runBulkInParallel(ctx, args.ItemIDs, func(ctx context.Context, _ int, itemID string) (string, error) {
		_, err := c.DeleteItem(ctx, DeleteItemArgs{
			BoardID: args.BoardID,
			ItemID:  itemID,
		})
		return itemID, err
	})

	agg := processBulkResults(resultSlice, func(idx int, id string, err error) string {
		return fmt.Sprintf("item %d (%s): %v", idx+1, id, err)
	})
	retriableIDs := retriableItemIDs(agg.failedItems)

	return BulkDeleteResult{
		Deleted:      len(agg.successIDs),
		ItemIDs:      agg.successIDs,
		Errors:       agg.errorMsgs,
		FailedItems:  agg.failedItems,
		RetriableIDs: retriableIDs,
		Message:      fmt.Sprintf("Deleted %d of %d items", len(agg.successIDs), len(args.ItemIDs)),
	}, nil
}
