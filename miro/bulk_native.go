package miro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// =============================================================================
// Native Bulk Create (POST /v2/boards/{board_id}/items/bulk)
// =============================================================================
//
// Replaces the per-item fan-out with ONE request for up to 20 items. MaxBulkItems
// is already 20, the endpoint's own cap, so no chunking is needed: every request
// this server accepts fits in a single call.
//
// The endpoint is TRANSACTIONAL — "if any item's create operation fails, the
// create operation for all the remaining items also fails, and none of the items
// will be created." That conflicts with this server's per-item contract
// (Created/Errors/FailedItems/RetriableIDs), where 19 of 20 may land. The
// reconciliation, and the reason this is safe:
//
//   - happy path: one request instead of N
//   - a 400 means the batch was rejected on validation, and transactionality
//     guarantees NOTHING was created, so falling back to the fan-out cannot
//     double-create. The caller gets the same per-item detail as before, having
//     spent one extra request to find out which item was bad.
//   - any OTHER failure (429, 5xx, network, timeout) leaves the outcome UNKNOWN:
//     the transaction may have committed with the response lost. Falling back
//     there could duplicate every item, so it does not fall back. The items are
//     reported failed-and-retriable, which is the same signal the fan-out would
//     have produced and leaves the retry decision with the caller.
//
// Note the asymmetry is deliberate: fall back only where the API's own
// transactionality proves nothing landed.

// bulkCreateItemBody maps one BulkCreateItem onto the native wire body.
//
// Each type reuses the SAME builder as its typed Create* method rather than
// re-deriving the mapping. That is what keeps a bulk-created sticky identical to
// a singly-created one — in particular the sticky colour vocabulary, which is a
// named-value enum the API rejects hex for.
func bulkCreateItemBody(boardID string, item BulkCreateItem) (map[string]interface{}, error) {
	var (
		body map[string]interface{}
		err  error
	)

	switch item.Type {
	case "sticky_note":
		if item.Content == "" {
			return nil, errors.New("content is required")
		}
		body = buildStickyCreateBody(CreateStickyArgs{
			BoardID:  boardID,
			Content:  item.Content,
			X:        item.X,
			Y:        item.Y,
			Color:    item.Color,
			Width:    item.Width,
			ParentID: item.ParentID,
		})

	case "shape":
		body, err = bulkShapeBody(boardID, item)
		if err != nil {
			return nil, err
		}

	case "text":
		// Same fallback CreateText applies: a text item with no explicit
		// text_color takes the generic color field.
		textColor := item.TextColor
		if textColor == "" {
			textColor = item.Color
		}
		body, err = buildCreateTextBody(CreateTextArgs{
			BoardID:  boardID,
			Content:  item.Content,
			X:        item.X,
			Y:        item.Y,
			Width:    item.Width,
			FontSize: item.FontSize,
			Color:    textColor,
			ParentID: item.ParentID,
		})
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported item type: %s", item.Type)
	}

	// The native endpoint discriminates on a top-level "type"; the per-endpoint
	// builders above have no reason to set it, since their URL carries the type.
	body["type"] = item.Type
	return body, nil
}

// bulkShapeBody maps a shape BulkCreateItem onto the native wire body.
//
// Same style composition CreateShape performs: base body carries data,
// position, geometry and parent; style is assembled separately and
// attached. Reusing these three helpers rather than re-deriving is what
// keeps colour normalization and text alignment identical across paths.
func bulkShapeBody(boardID string, item BulkCreateItem) (map[string]interface{}, error) {
	shapeArgs := CreateShapeArgs{
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
		FontSize:          item.FontSize,
		ParentID:          item.ParentID,
	}
	if err := validateShapeCreateArgs(shapeArgs.BoardID, shapeArgs.Shape); err != nil {
		return nil, err
	}
	body := buildShapeBaseBody(shapeArgs.toCoreBody())
	style, err := buildCreateShapeStyle(shapeArgs.Color, shapeArgs.TextColor)
	if err != nil {
		return nil, err
	}
	if err := applyShapeTextAlign(style, shapeArgs.TextAlign, shapeArgs.TextAlignVertical); err != nil {
		return nil, err
	}
	if shapeArgs.FontSize > 0 {
		style["fontSize"] = strconv.Itoa(shapeArgs.FontSize)
	}
	if len(style) > 0 {
		body["style"] = style
	}
	return body, nil
}

// buildNativeBulkBody maps every item, or reports the first mapping failure.
//
// All-or-nothing on purpose: the endpoint is transactional, so sending a partial
// batch would silently drop the unmappable items. The caller falls back to the
// fan-out instead, where each bad item is reported individually.
func buildNativeBulkBody(boardID string, items []BulkCreateItem) ([]map[string]interface{}, error) {
	bodies := make([]map[string]interface{}, 0, len(items))
	for i, item := range items {
		body, err := bulkCreateItemBody(boardID, item)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i+1, err)
		}
		bodies = append(bodies, body)
	}
	return bodies, nil
}

// nativeBulkRejectedBatch reports whether err is the API rejecting the whole
// batch on validation, which transactionality proves created nothing.
//
// Only 400 qualifies. 429 and 5xx leave the outcome unknown, and treating them
// as "nothing was created" is exactly the assumption that would duplicate items.
func nativeBulkRejectedBatch(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusBadRequest
}

// createItemsNative issues the single bulk call and returns the created IDs.
func (c *Client) createItemsNative(ctx context.Context, boardID string, items []BulkCreateItem) ([]string, error) {
	bodies, err := buildNativeBulkBody(boardID, items)
	if err != nil {
		return nil, err
	}

	respBody, err := c.request(ctx, http.MethodPost, "/boards/"+boardID+"/items/bulk", bodies)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	ids := make([]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		ids = append(ids, item.ID)
	}
	return ids, nil
}

// allItemsFailed builds the result for a batch whose outcome is unknown.
//
// Every item is attributed the same cause, categorized through the same
// categorizeBulkError the fan-out uses, so retriability is decided by the HTTP
// status rather than asserted here. The retry decision stays with the caller.
func allItemsFailed(items []BulkCreateItem, cause error) BulkCreateResult {
	failed := make([]BulkItemError, 0, len(items))
	msgs := make([]string, 0, len(items))
	for i := range items {
		failed = append(failed, categorizeBulkError(i, "", cause))
		msgs = append(msgs, fmt.Sprintf("item %d: %v", i+1, cause))
	}
	retriable := retriableIndexes(failed)
	return BulkCreateResult{
		Created:      0,
		ItemIDs:      []string{},
		Errors:       msgs,
		FailedItems:  failed,
		RetriableIDs: retriable,
		Message: fmt.Sprintf(
			"Created 0 of %d items: the bulk call failed as one transaction. %s",
			len(items), transactionOutcomeNote(len(retriable))),
	}
}

// transactionOutcomeNote states what the caller can and cannot conclude. The
// honest answer is "almost certainly nothing was created, but not provably" —
// a committed transaction whose response was lost is indistinguishable here.
func transactionOutcomeNote(retriable int) string {
	if retriable == 0 {
		return "None were created."
	}
	return fmt.Sprintf(
		"None were created unless the response was lost in transit, so check the board before retrying; %d items can be retried", retriable)
}
