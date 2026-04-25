# Spatial Defaults

Shared coordinate, size, and layout math for the Miro workflows. Pulled directly from the tested values in `prompts/prompts.go`.

## Coordinate system

- Origin `(0, 0)` is the canvas top-left for items at canvas root.
- Inside a frame, `(0, 0)` is the **frame's** top-left. Always pass `parent` (the frame ID) when placing an item inside a frame, then use frame-relative coordinates.
- Y increases downward.

## Universal sizes

| Element | Width | Height | Notes |
|---------|-------|--------|-------|
| Default frame (column) | 800 | 600 | Sprint board, retrospective |
| Compact frame (column) | 400 | 100–800 | Story map header, kanban |
| Sticky note | ~200 | ~200 | Miro auto-sizes; you don't set width |
| Title text | n/a | n/a | `font_size: 48`, centered |
| Section header text | n/a | n/a | `font_size: 24` or 36 |

## Gap math

- **Frame-to-frame horizontal gap:** 50px.
  - Default frame stride: `x = N * (800 + 50) = N * 850`
  - Compact frame stride: `x = N * (400 + 50) = N * 450`
- **Title text Y offset:** `y = -100` (above the first frame).
- **Sticky-to-sticky inside a frame:** 40px gap (Miro default rendering).

## Per-workflow column geometry

### Sprint board (4 columns, default frames)
```
Backlog       (x = 0,    width 800)
In Progress   (x = 850,  width 800)
In Review     (x = 1700, width 800)
Done          (x = 2550, width 800)
Title         (x = 1275, y = -100, font_size 48)
```
Total board width: ~3350px.

### Retrospective (3 columns, default frames)
```
What Went Well     (x = 0,    width 800)
What Could Improve (x = 850,  width 800)
Action Items       (x = 1700, width 800)
Title              (x = 850,  y = -100, font_size 48)
```
Total board width: ~2500px.

### Brainstorm (radial, no frames)
Central diamond shape at `(400, 300)`. 6 stickies arranged on a ring:
```
Top         (400, 0)    orange
Top-left    (100, 100)  yellow
Top-right   (700, 100)  green
Bottom-left (100, 500)  blue
Bottom-right (700, 500) pink
Bottom      (400, 600)  cyan
```
Connectors fan from the diamond to each sticky.

### Story map (compact frames + swimlanes)
```
Headers (y = 0,   height 100, blue background, width 400 each):
  Discovery     (x = 0)
  Onboarding    (x = 450)
  Core Usage    (x = 900)
  Growth        (x = 1350)

User tasks       (y = 150,  yellow stickies, 2-3 per activity)
Swimlane MVP     (y = 350,  green stickies)
Swimlane v1.0    (y = 550,  blue stickies)
Swimlane Future  (y = 750,  gray stickies)
```

### Kanban (compact tall frames)
```
Frames are width 400, height 800, gap 50.
N columns: x = i * 450 for i in 0..N-1
Frame fill colors cycle: gray → blue → yellow → green → pink → orange → cyan
Column header text: font_size 24, just above each frame
Title text: font_size 36, centered above the row
```

## Sticky placement inside a frame

When `parent_id` is set on a sticky:

- Use **frame-relative** coords. `(0, 0)` is the frame's top-left, NOT the canvas root.
- Leave a 40px margin from the frame edge.
- Stack stickies vertically: `y = 40 + (i * 240)` for the i-th sticky in a column.
- For grids inside a frame, prefer `miro_create_sticky_grid` over manual placement; it handles spacing.

## Connector placement

Connectors don't take coordinates. They take `start_item_id` and `end_item_id`. Always create the items first, collect their returned IDs, then create connectors in a second pass.

For radial layouts (brainstorm), use `shape: "curved"`. For flowcharts, use `shape: "elbowed"` or `"straight"` to match diagram conventions.

## Bulk creation tips

- For ≥ 5 same-type items, prefer `miro_bulk_create`.
- Pass full coordinate sets in the bulk request; don't post-update.
- Response order is NOT guaranteed; key items by your own correlation ID if downstream calls need to reference them.
- Keep bulk batches under 50 items per call to avoid rate limits.

## Why these defaults

The values in this file are not arbitrary; they're the same numbers the `miro-mcp-server` MCP prompts emit (`prompts/prompts.go`). Users who fire `/create-sprint-board` get the same layout the skill produces. Don't drift from these without good reason.
