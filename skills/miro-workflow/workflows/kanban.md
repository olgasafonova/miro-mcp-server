# Kanban Workflow

A configurable N-column workflow board: stickies as cards, frames as columns, color-coded by stage.

## Triggers

Use this workflow when the user says:
- "Make a kanban board"
- "Set up a task board"
- "Workflow board with columns: <names>"
- "Kanban for <project>"

## Tool sequence

1. **Pre-flight.** `miro_list_boards`; if the user named an existing board, use its ID. Otherwise:
2. **`miro_create_board`** with `name = "<board_name>"`. Save `board_id`.
3. **Determine columns.** Default: `["To Do", "In Progress", "Review", "Done"]`. If the user provided custom names ("Doing, Blocked, Done"), use those. Number of columns N ≥ 2, ≤ 7 (palette has 7 colors before cycling).
4. **`miro_create_text`** for the title at `(x = (N * 450) / 2 - 100, y = -100, font_size = 36)` with `content = "<board_name>"`. Title sits centered above the row.
5. **`miro_bulk_create`** for N column frames. For each column index `i` (0-based):
   - Position: `(x = i * 450, y = 0, width = 400, height = 800)`
   - Fill color (`color` parameter, CSS hex): cycles through `["#E6E6E6", "#A6CCF5", "#FFF8B4", "#A6E5BB", "#F5D0E8", "#FFD4A3", "#B4E5E5"]` by `i % 7` (gray → blue → yellow → green → pink → orange → cyan, all light pastels). Miro API rejects named colors for frames.
   - Title: the column name
   Collect frame IDs.
6. **`miro_create_text`** for each column header at `(x = i * 450 + 100, y = -50, font_size = 24)` with the column name. Headers sit just above each frame.
7. **`miro_bulk_create`** for sample stickies in the FIRST column only:
   - 2-3 `yellow` stickies, parented to the first frame
   - Frame-relative coords (frame top-left is `(0, 0)`; sticky CENTER is placed at the given coord). For a 400×800 frame: `(x=200, y=140)`, `(x=200, y=400)`, `(x=200, y=660)` — centered horizontally, evenly distributed vertically. Set sticky `width=160` to keep them from overlapping.
   - Sample text: "Sample task 1", "Sample task 2", "Sample task 3"
8. **Return** the board URL.

## Layout

For 4 columns (default):

```
y = -100  [        <board_name> (font 36)        ]
y = -50   [To Do   ][In Prog ][Review  ][Done    ] ← column headers
y = 0     ┌────────┬────────┬────────┬────────┐
          │  gray  │  blue  │ yellow │ green  │
          │ frame  │ frame  │ frame  │ frame  │
y = 800   └────────┴────────┴────────┴────────┘
```

Column stride: 450px (400 frame + 50 gap). Columns are tall (800px) to fit a stack of stickies.

Spatial details: see [../spatial-defaults.md](../spatial-defaults.md).

## Colors

Frames cycle through the palette in this order: `gray → blue → yellow → green → pink → orange → cyan → gray (wraps)`.

Sample stickies in the first column: yellow (neutral, since column hue varies). New stickies users add later can take any color; kanban is loose about sticky color.

| Column index | Color |
|--------------|-------|
| 0 (first) | gray |
| 1 | blue |
| 2 | yellow |
| 3 | green |
| 4 | pink |
| 5 | orange |
| 6 | cyan |

Full palette: see [../color-conventions.md](../color-conventions.md).

## Personalization

- **Custom column names:** Pass them in. Color cycle stays the same regardless of column meaning. (User can update colors after the fact if they want a different scheme.)
- **More than 4 columns:** Up to 7 stay in palette; beyond that, cycle wraps. Boards wider than 7 columns get unwieldy; flag this to the user and suggest splitting into two boards.
- **WIP limits:** Users often want a number on each column ("3 max"). Add a small text label above each column header at `(x = i * 450 + 100, y = -80)` with `content = "WIP: <N>"`. Optional.
- **Empty board:** If the user says "no sample stickies", skip step 7.
- **Add to existing board:** If `board_id` was passed, place the row of frames at a Y offset to clear existing content.

## Variations

### Personal kanban (3 columns)
`["To Do", "Doing", "Done"]`. Same shape, narrower board.

### Team kanban with swimlanes
N columns × M rows. Each (column, row) is a frame. Beyond this workflow's scope; route to a custom layout.

### Cards instead of stickies
For Jira-style task tracking, use `miro_create_card` instead of `miro_create_sticky`. Cards have title + description + assignee fields.

## Acceptance

- N frames in a horizontal row, each 400×800.
- Each frame has a column header text above it (font 24).
- Title text at the top, centered (font 36).
- 2-3 yellow sample stickies in the first column, parented correctly.
- Frame fills cycle through the palette per `i % 7`.
- Board URL returned.

## Anti-patterns

- Forgetting column headers: the frames look anonymous without text labels.
- Sample stickies in every column: looks like fake activity. Only seed the first column.
- Overlapping frame coords (forgot the 50px gap): visually broken.
- Stickies on canvas root instead of parented to a frame: they don't move with the column when the user drags it.
- More than 7 columns without warning the user: cycle wraps and meaning gets lost.
