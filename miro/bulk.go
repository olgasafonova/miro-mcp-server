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

// categorizeBulkError analyzes an error and returns a BulkItemError with appropriate categorization.
func categorizeBulkError(index int, itemID string, err error) BulkItemError {
	bulkErr := BulkItemError{
		Index:   index,
		ItemID:  itemID,
		Message: err.Error(),
	}

	// Check for API errors with status codes
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		bulkErr.StatusCode = apiErr.StatusCode

		switch {
		case apiErr.StatusCode == 429:
			bulkErr.ErrorType = "rate_limit"
			bulkErr.IsRetriable = true
		case apiErr.StatusCode == 404:
			bulkErr.ErrorType = "not_found"
			bulkErr.IsRetriable = false
		case apiErr.StatusCode >= 500:
			bulkErr.ErrorType = "server"
			bulkErr.IsRetriable = true
		case apiErr.StatusCode == 400:
			bulkErr.ErrorType = "validation"
			bulkErr.IsRetriable = false
		case apiErr.StatusCode == 401 || apiErr.StatusCode == 403:
			bulkErr.ErrorType = "auth"
			bulkErr.IsRetriable = false
		default:
			bulkErr.ErrorType = "api"
			bulkErr.IsRetriable = false
		}
		return bulkErr
	}

	// Check for validation errors
	var valErr *ValidationError
	if errors.As(err, &valErr) {
		bulkErr.ErrorType = "validation"
		bulkErr.IsRetriable = false
		return bulkErr
	}

	// Check for context errors (timeout, cancellation)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		bulkErr.ErrorType = "timeout"
		bulkErr.IsRetriable = true
		return bulkErr
	}

	// Check for network-related errors
	errStr := err.Error()
	if strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "EOF") {
		bulkErr.ErrorType = "network"
		bulkErr.IsRetriable = true
		return bulkErr
	}

	// Default: unknown error
	bulkErr.ErrorType = "unknown"
	bulkErr.IsRetriable = false
	return bulkErr
}

// BulkCreate creates multiple items in one operation.
// Items are created in parallel using goroutines, with concurrency
// controlled by the client's semaphore (MaxConcurrentRequests).
func (c *Client) BulkCreate(ctx context.Context, args BulkCreateArgs) (BulkCreateResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return BulkCreateResult{}, err
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

	// Extract IDs and categorize errors
	var itemIDs []string
	var errorMsgs []string
	var failedItems []BulkItemError
	var retriableIDs []int

	for _, r := range resultSlice {
		if r.err != nil {
			errorMsgs = append(errorMsgs, fmt.Sprintf("item %d: %v", r.index+1, r.err))

			// Categorize the error for better recovery
			bulkErr := categorizeBulkError(r.index, "", r.err)
			failedItems = append(failedItems, bulkErr)

			if bulkErr.IsRetriable {
				retriableIDs = append(retriableIDs, r.index)
			}
		} else if r.id != "" {
			itemIDs = append(itemIDs, r.id)
		}
	}

	// Build informative message
	message := fmt.Sprintf("Created %d of %d items", len(itemIDs), len(args.Items))
	if len(retriableIDs) > 0 {
		message += fmt.Sprintf(". %d items can be retried", len(retriableIDs))
	}

	return BulkCreateResult{
		Created:      len(itemIDs),
		ItemIDs:      itemIDs,
		ItemURLs:     BuildItemURLs(args.BoardID, itemIDs),
		Errors:       errorMsgs,
		FailedItems:  failedItems,
		RetriableIDs: retriableIDs,
		Message:      message,
	}, nil
}

// BulkUpdate updates multiple items in one operation.
// Items are updated in parallel using goroutines, with concurrency
// controlled by the client's semaphore (MaxConcurrentRequests).
func (c *Client) BulkUpdate(ctx context.Context, args BulkUpdateArgs) (BulkUpdateResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return BulkUpdateResult{}, err
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

	// Extract IDs and categorize errors
	var itemIDs []string
	var errorMsgs []string
	var failedItems []BulkItemError
	var retriableIDs []string

	for _, r := range resultSlice {
		if r.err != nil {
			errorMsgs = append(errorMsgs, fmt.Sprintf("item %d (%s): %v", r.index+1, r.id, r.err))

			// Categorize the error for better recovery
			bulkErr := categorizeBulkError(r.index, r.id, r.err)
			failedItems = append(failedItems, bulkErr)

			if bulkErr.IsRetriable {
				retriableIDs = append(retriableIDs, r.id)
			}
		} else if r.id != "" {
			itemIDs = append(itemIDs, r.id)
		}
	}

	// Build informative message
	message := fmt.Sprintf("Updated %d of %d items", len(itemIDs), len(args.Items))
	if len(retriableIDs) > 0 {
		message += fmt.Sprintf(". %d items can be retried", len(retriableIDs))
	}

	return BulkUpdateResult{
		Updated:      len(itemIDs),
		ItemIDs:      itemIDs,
		Errors:       errorMsgs,
		FailedItems:  failedItems,
		RetriableIDs: retriableIDs,
		Message:      message,
	}, nil
}

// BulkDelete deletes multiple items in one operation.
// Items are deleted in parallel using goroutines, with concurrency
// controlled by the client's semaphore (MaxConcurrentRequests).
func (c *Client) BulkDelete(ctx context.Context, args BulkDeleteArgs) (BulkDeleteResult, error) {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return BulkDeleteResult{}, err
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

	// Extract IDs and errors with categorization
	var itemIDs []string
	var errorMsgs []string
	var failedItems []BulkItemError
	var retriableIDs []string
	for _, r := range resultSlice {
		if r.err != nil {
			errorMsgs = append(errorMsgs, fmt.Sprintf("item %d (%s): %v", r.index+1, r.id, r.err))
			bulkErr := categorizeBulkError(r.index, r.id, r.err)
			failedItems = append(failedItems, bulkErr)
			if bulkErr.IsRetriable && r.id != "" {
				retriableIDs = append(retriableIDs, r.id)
			}
		} else if r.id != "" {
			itemIDs = append(itemIDs, r.id)
		}
	}

	return BulkDeleteResult{
		Deleted:      len(itemIDs),
		ItemIDs:      itemIDs,
		Errors:       errorMsgs,
		FailedItems:  failedItems,
		RetriableIDs: retriableIDs,
		Message:      fmt.Sprintf("Deleted %d of %d items", len(itemIDs), len(args.ItemIDs)),
	}, nil
}
