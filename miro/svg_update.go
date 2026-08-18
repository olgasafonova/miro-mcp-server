package miro

import (
	"context"
	"fmt"
	"strings"
)

// =============================================================================
// SVG -> Board diff (update / delete / additive create, keyed on data-miro-id)
// =============================================================================
//
// Applies an SVG document to a board as a diff. Elements carrying data-miro-id
// are updated in place with PATCH semantics: geometry is restated as a unit
// (x, y, width, height together), fill maps to color, text content maps to
// content. Elements with data-miro-id and data-deleted="true" are deleted.
// Elements without data-miro-id are created, reusing the create dialect.
//
// Two failure classes, deliberately: a malformed document fails the whole
// request at the parse layer (nothing applied); a per-item semantic error
// lands in Failed while the rest of the batch applies. The output of
// miro_read_board_svg is re-submittable here — its elements carry data-miro-id
// and top-left rect geometry, exactly what this parser expects.

// svgUpdateOutcome accumulates the per-element results of one update call.
type svgUpdateOutcome struct {
	updated []SVGUpdatedItem
	deleted []string
	failed  []SVGFailedItem
	creates []svgElement
}

func (o *svgUpdateOutcome) fail(el svgElement, reason string) {
	o.failed = append(o.failed, SVGFailedItem{ID: el.miroID, Element: el.name, Reason: reason})
}

// svgUpdateArgs builds the PATCH payload for one identified element. Geometry
// is a unit: rects and ellipses restate all four values, text restates its
// anchor only (Miro sizes text by content).
func svgUpdateArgs(boardID string, el svgElement) (UpdateItemArgs, string) {
	args := UpdateItemArgs{BoardID: boardID, ItemID: el.miroID}
	x, y := el.x, el.y
	args.X, args.Y = &x, &y

	if el.name == "text" {
		content := el.text
		args.Content = &content
		return args, ""
	}
	if el.w <= 0 || el.h <= 0 {
		return args, "update requires full geometry (x, y, width, height restated as a unit)"
	}
	w, h := el.w, el.h
	args.Width, args.Height = &w, &h
	if el.fill != "" && el.fill != "none" {
		fill := el.fill
		args.Color = &fill
	}
	if el.text != "" {
		content := el.text
		args.Content = &content
	}
	return args, ""
}

// applySVGUpdate updates one identified element in place.
func (c *Client) applySVGUpdate(ctx context.Context, boardID string, el svgElement, out *svgUpdateOutcome) {
	args, reason := svgUpdateArgs(boardID, el)
	if reason != "" {
		out.fail(el, reason)
		return
	}
	if _, err := c.UpdateItem(ctx, args); err != nil {
		out.fail(el, err.Error())
		return
	}
	out.updated = append(out.updated, SVGUpdatedItem{ID: el.miroID, Element: el.name})
}

// applySVGDelete deletes one identified element. Connectors are not
// reachable through the generic items endpoint (DELETE there is a 404,
// verified live 18-08-2026), so lines route to DeleteConnector.
func (c *Client) applySVGDelete(ctx context.Context, boardID string, el svgElement, out *svgUpdateOutcome) {
	var err error
	if el.name == "line" {
		_, err = c.DeleteConnector(ctx, DeleteConnectorArgs{BoardID: boardID, ConnectorID: el.miroID})
	} else {
		_, err = c.DeleteItem(ctx, DeleteItemArgs{BoardID: boardID, ItemID: el.miroID})
	}
	if err != nil {
		out.fail(el, err.Error())
		return
	}
	out.deleted = append(out.deleted, el.miroID)
}

// routeSVGElement sends one parsed element to its diff action: delete, update,
// or (for elements with no data-miro-id) deferred creation.
func (c *Client) routeSVGElement(ctx context.Context, boardID string, el svgElement, out *svgUpdateOutcome) {
	switch {
	case el.miroID == "":
		out.creates = append(out.creates, el)
	case el.deleted:
		c.applySVGDelete(ctx, boardID, el, out)
	case el.name == "line":
		out.fail(el, "connectors cannot be updated through SVG; use miro_update_connector")
	default:
		c.applySVGUpdate(ctx, boardID, el, out)
	}
}

// validateUpdateFromSVGArgs checks the request bounds before any parsing.
func validateUpdateFromSVGArgs(args UpdateFromSVGArgs) error {
	if err := ValidateBoardID(args.BoardID); err != nil {
		return err
	}
	if strings.TrimSpace(args.SVG) == "" {
		return fmt.Errorf("svg is required")
	}
	if len(args.SVG) > maxSVGDocumentBytes {
		return fmt.Errorf("svg exceeds %d bytes (got %d)", maxSVGDocumentBytes, len(args.SVG))
	}
	return nil
}

// updateFromSVGMessage summarizes one applied diff.
func updateFromSVGMessage(out *svgUpdateOutcome, created, skipped int) string {
	return fmt.Sprintf("Updated %d, deleted %d, created %d item(s) (%d failed, %d skipped)",
		len(out.updated), len(out.deleted), created, len(out.failed), skipped)
}

// UpdateFromSVG applies an SVG document to a board as a diff keyed on
// data-miro-id: identified elements update in place, data-deleted elements
// are removed, unidentified elements are created.
func (c *Client) UpdateFromSVG(ctx context.Context, args UpdateFromSVGArgs) (UpdateFromSVGResult, error) {
	if err := validateUpdateFromSVGArgs(args); err != nil {
		return UpdateFromSVGResult{}, err
	}

	elements, skipped, err := parseSVGElements(args.SVG)
	if err != nil {
		return UpdateFromSVGResult{}, err
	}
	if len(elements) > maxSVGCreateElements {
		return UpdateFromSVGResult{}, fmt.Errorf("svg contains %d drawable elements; the cap is %d per call", len(elements), maxSVGCreateElements)
	}

	out := &svgUpdateOutcome{}
	for _, el := range elements {
		c.routeSVGElement(ctx, args.BoardID, el, out)
	}

	created, createSkips, createErr := c.createSVGElements(ctx, args.BoardID, out.creates, svgOffset{})
	skipped = append(skipped, createSkips...)
	result := UpdateFromSVGResult{
		Updated: out.updated,
		Deleted: out.deleted,
		Created: created,
		Failed:  out.failed,
		Skipped: skipped,
		Message: updateFromSVGMessage(out, len(created), len(skipped)),
	}
	if createErr != nil {
		return result, fmt.Errorf("update from svg: create pass failed after %d item(s): %w", len(created), createErr)
	}
	return result, nil
}
