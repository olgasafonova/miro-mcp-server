package tools

// Canvas SVG Tools (local transform)
var canvasSVGTools = []ToolSpec{
	{
		Name:     "miro_read_board_svg",
		Method:   "ReadBoardSVG",
		Title:    "Read Board as SVG",
		Category: "read",
		ReadOnly: true,
		Description: `Render a board's items as an SVG document, computed locally from item geometry (no export job, no external service). Frames render as dashed outlines, shapes and stickies as filled rects/ellipses with labels, text as text, connectors as lines between item centers. Every element carries data-miro-id and data-miro-type attributes linking it back to the board item, and the document is directly re-submittable to miro_update_from_svg after editing.

USE WHEN: "show me the board as SVG", "show me just this frame", "vector snapshot of the layout", "diff two boards visually", or feeding a board's spatial layout to a tool that reads SVG.

NOT FOR: pixel-accurate rendering (use miro_get_board_picture) or full content analysis (use miro_get_board_content). This is a spatial approximation: images render as placeholder rects and rich styling is reduced to fill colors.

PARAMETERS:
- board_id: Required. max_items: cap on items fetched (default 500, max 2000).
- frame_id: Optional. Scope the render to one frame and its children; child coordinates come back relative to the frame's top-left corner, with the frame outline at (0,0).

RETURNS: svg (the document), item_count, skipped (items with no visual mapping), truncated.`,
	},
	{
		Name:     "miro_create_from_svg",
		Method:   "CreateFromSVG",
		Title:    "Create Items from SVG",
		Category: "create",
		Description: `Create board items from an SVG document, parsed locally. Supported elements: rect (-> rectangle shape; rx>0 -> round_rectangle; data-type="sticky" -> sticky note with data-content text; data-type="frame" -> frame with data-title), circle and ellipse (-> circle shape), polygon with 3 points (-> triangle shape), text (-> text item), image with a public href (-> image item), line with data-start/data-end referencing other elements' id attributes (-> connector, created after the referenced items; data-caption -> label), g with transform="translate(x,y)" (offset applied to children, nesting supported). Fill colors carry over. Unsupported elements (path, multi-point polygon, ...) are itemized in the response as skipped, never silently dropped.

USE WHEN: "put this SVG on the board", "recreate this wireframe as board items", importing a diagram from a tool that exports SVG.

NOT FOR: pixel-faithful SVG import (Miro has no vector item type; this maps to native shapes), Mermaid diagrams (use miro_generate_diagram), or editing existing items (use miro_update_from_svg).

PARAMETERS:
- board_id, svg: Required. SVG source, max 1 MiB, max 200 drawable elements per call.
- offset_x, offset_y: Optional placement offset for the whole batch.

RETURNS: created (id, type, source element per item), skipped (element + reason), count. On a mid-batch failure the created list still names every item that landed, so the caller can verify or clean up.`,
	},
	{
		Name:        "miro_update_from_svg",
		Method:      "UpdateFromSVG",
		Title:       "Update Items from SVG",
		Category:    "update",
		Destructive: true,
		Description: `Apply an SVG document to a board as a diff keyed on data-miro-id, parsed locally. Elements carrying data-miro-id update the matching item in place: geometry is restated as a unit (x, y, width, height together; a partial restatement fails that item), fill -> color, text content -> content. WARNING: elements with data-miro-id AND data-deleted="true" permanently delete the item; confirm with the user first. Elements WITHOUT data-miro-id are created additively, same dialect as miro_create_from_svg. The output of miro_read_board_svg is directly re-submittable here: edit it and send it back.

Two failure classes: a malformed document (bad XML, unescaped &) fails the whole request with nothing applied; a per-item semantic error (unknown id, partial geometry) lands in the failed list while the rest of the batch applies.

USE WHEN: "move these items", "recolor the stickies in this layout", "apply my edited SVG back to the board", iterating on a layout read via miro_read_board_svg.

NOT FOR: first-time creation (use miro_create_from_svg) or connector edits (use miro_update_connector; connectors are create-only in this dialect).

PARAMETERS:
- board_id, svg: Required. SVG source, max 1 MiB, max 200 drawable elements per call.

RETURNS: updated (id + element), deleted (ids), created (additive items), failed (id + reason per semantic failure), skipped (element + reason).`,
	},
}

// Export Tools
var exportTools = []ToolSpec{
	{
		Name:     "miro_get_board_picture",
		Method:   "GetBoardPicture",
		Title:    "Get Board Picture",
		Category: "export",
		ReadOnly: true,
		Description: `Get the preview image URL for a board. Works for all Miro plans. For full PDF/SVG exports, use the Enterprise export tools.

RETURNS: Preview image URL for the board.

VOICE-FRIENDLY: "Got preview image for the board"`,
	},
	{
		Name:     "miro_create_export_job",
		Method:   "CreateExportJob",
		Title:    "Create Export Job",
		Category: "export",
		Description: `Export boards to PDF, SVG, or HTML. ENTERPRISE ONLY.

RETURNS: Export job ID and initial status. Poll with miro_get_export_job_status.

FAILS WHEN: Not on Enterprise plan. No board_ids provided. More than 50 boards. Invalid format (must be pdf, svg, or html).`,
	},
	{
		Name:     "miro_get_export_job_status",
		Method:   "GetExportJobStatus",
		Title:    "Get Export Job Status",
		Category: "export",
		ReadOnly: true,
		Description: `Check the progress of a board export job. Call after miro_create_export_job. ENTERPRISE ONLY.

USE WHEN: Polling an export job started with miro_create_export_job. Call repeatedly until status is "completed" or "failed".

PARAMETERS:
- org_id: Required. Same organization ID used in miro_create_export_job.
- job_id: Required. Job ID returned by miro_create_export_job.

RETURNS: Job ID, status (in_progress, completed, failed), progress percentage, and boards exported/total count.

NEXT STEPS BY STATUS:
- in_progress: Wait a few seconds, then poll again
- completed: Call miro_get_export_job_results to get download links
- failed: Export failed; check board IDs and permissions

FAILS WHEN: Not on Enterprise plan. Invalid org_id or job_id.

VOICE-FRIENDLY: "Export 50% complete: 5 of 10 boards exported"`,
	},
	{
		Name:        "miro_get_export_job_results",
		Method:      "GetExportJobResults",
		Title:       "Get Export Job Results",
		Category:    "export",
		ReadOnly:    true,
		Description: `Get download links for completed export. ENTERPRISE ONLY. Links expire in 15 min; call again to regenerate.`,
	},
}

// Diagram Generation Tools (AI-Powered)
var diagramTools = []ToolSpec{
	{
		Name:     "miro_generate_diagram",
		Method:   "GenerateDiagram",
		Title:    "Generate Diagram from Code",
		Category: "diagrams",
		Description: `Generate a diagram on a Miro board from Mermaid code. Parses locally (no external service), creates shapes and connectors with auto-layout.

USE WHEN: "create a flowchart", "generate diagram from code", "draw a process flow", "make a sequence diagram"

NOT FOR: Freeform shape placement (use miro_create_shape). Mindmaps (use miro_create_mindmap_node).

SUPPORTED SYNTAX:
- flowchart/graph (directions: TB, BT, LR, RL): A[rect] --> B{diamond} -->|label| C((circle))
- sequenceDiagram: participant A; A->>B: sync; A-->>B: async

OUTPUT MODES:
- discrete (default): Individual shapes and connectors, no container
- grouped: All items in a logical group for easy move/delete
- framed: All items inside a titled frame

PARAMETERS:
- board_id: Required
- diagram: Mermaid code (required). Must start with "flowchart TB", "graph LR", or "sequenceDiagram".
- output_mode: "discrete" (default), "grouped", or "framed"
- use_stencils: true for professional flowchart symbols (terminator, process, decision, I/O)
- start_x, start_y: Position offset (default: 0, 0)
- parent_id: Frame ID to place diagram inside

RETURNS: Counts of created nodes/connectors/frames, all item IDs and view links, diagram dimensions. In grouped/framed mode, also returns container ID.

FAILS WHEN: Invalid Mermaid syntax (returns line number and fix suggestion). Missing header (must start with flowchart/graph/sequenceDiagram). Input exceeds 50KB or 500 lines.

VOICE-FRIENDLY: "Created flowchart with 6 shapes and 5 connectors"`,
	},
	{
		Name:     "miro_list_diagrams",
		Method:   "ListDiagrams",
		Title:    "List Diagrams",
		Category: "diagrams",
		ReadOnly: true,
		Description: `List native diagram items on a Miro board (diagrams made with Miro's own diagramming tools). Returns diagram metadata: ID, title, position, size, and timestamps.

USE WHEN: "find diagrams on this board", "list all diagrams", "does this board have diagrams"

NOT FOR: Diagrams built from shapes and connectors by miro_generate_diagram — those are regular items, not native diagram items; use miro_list_items. Creating diagrams: the REST API is read-only here (POST returns 405); creation happens in Miro's UI or hosted tooling.

PARAMETERS:
- board_id: Required
- limit: Max diagrams to return (default 10, max 50)
- cursor: Pagination cursor from previous response

RETURNS: Diagram items with IDs, titles, positions, sizes, and timestamps.

RELATED: Use miro_get_diagram for a single diagram's details. Use miro_generate_diagram to draw a new diagram from Mermaid code (as shapes, not a native diagram item).

VOICE-FRIENDLY: "Found 2 diagrams on the board"`,
	},
	{
		Name:     "miro_get_diagram",
		Method:   "GetDiagram",
		Title:    "Get Diagram Details",
		Category: "diagrams",
		ReadOnly: true,
		Description: `Get metadata for a specific native diagram item by ID. Returns title, position, size, parent frame, and timestamps.

USE WHEN: "get diagram details", "where is that diagram", "diagram info"

NOT FOR: Reading a diagram's internal nodes and edges — the REST API exposes item metadata only. For full diagram content, open the board in Miro.

PARAMETERS:
- board_id: Required
- item_id: Diagram item ID (required). Get from miro_list_diagrams.

RETURNS: Diagram ID, title, position, size, parent frame ID, timestamps, and view link.

RELATED: Use miro_list_diagrams to discover diagrams on a board.

VOICE-FRIENDLY: "Diagram 'Architecture' is at position (100, 200), size 1200x700"`,
	},
}
