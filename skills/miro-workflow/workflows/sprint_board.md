# Sprint Board Workflow

A 4-column tracker board for sprint work: Backlog → In Progress → In Review → Done.

## Triggers

Use this workflow when the user says:
- "Create a sprint board"
- "Set up sprint planning for sprint N"
- "Make a sprint kickoff board"
- "Build a Scrum board"

## Tool sequence

1. **Pre-flight.** `miro_list_boards` to confirm the workspace; if the user named an existing board, get its ID. Otherwise:
2. **`miro_create_board`** with `name = "<board_name>"`. Save `board_id`.
3. **`miro_create_text`** at `(x=1275, y=-100, font_size=48)` with `content = "Sprint <N> Planning"`. The title.
4. **`miro_create_frame`** four times (or one `miro_bulk_create` with 4 frames). Frame `color` is a CSS hex string (Miro API rejects named colors for frames):
   - Backlog: `(x=0, y=0, width=800, height=600, color="#E6E6E6")` (light gray)
   - In Progress: `(x=850, y=0, ..., color="#A6CCF5")` (light blue)
   - In Review: `(x=1700, y=0, ..., color="#FFF8B4")` (light yellow)
   - Done: `(x=2550, y=0, ..., color="#A6E5BB")` (light green)
   See [../color-conventions.md](../color-conventions.md) for the hex palette.
   Collect all 4 frame IDs from the response.
5. **`miro_bulk_create`** for starter stickies (or individual `miro_create_sticky` calls). Sticky `color` accepts named values:
   - Backlog frame: 3 `yellow` stickies with placeholder tasks at `(x=400, y=140)`, `(x=400, y=300)`, `(x=400, y=460)` — frame-relative, item-center placement, fits 3 stickies vertically in 600px height (use `width=160` on stickies to keep them from overlapping).
   - In Progress frame: 1 `light_blue` sticky "Current work" at `(x=400, y=300)`.
   - In Review frame: 1 `light_pink` sticky "Awaiting review" at `(x=400, y=300)`.
   - Done frame: 1 `light_green` sticky "Completed items" at `(x=400, y=300)`.
   Each sticky must include `parent_id = <frame_id>` for the column it belongs to. Frame-relative coords use the frame's top-left as `(0, 0)`; the sticky's CENTER is placed at the given (x, y).
6. **Return** the board URL: `https://miro.com/app/board/<board_id>/`.

## Layout

Total board width: ~3350px. Columns are 800 wide with 50px gaps. Title spans the center.

```
y = -100  [   Sprint N Planning (font 48)   ]
y = 0     [Backlog ][ In Prog ][ Review ][ Done ]
            gray      blue      yellow    green
            x=0       x=850     x=1700    x=2550
```

Spatial details: see [../spatial-defaults.md](../spatial-defaults.md).

## Colors

| Element | Color | Source |
|---------|-------|--------|
| Backlog frame | gray | inactive |
| In Progress frame | blue | active |
| In Review frame | yellow | neutral / attention |
| Done frame | green | positive / done |
| Backlog stickies | yellow | tasks (neutral) |
| In Progress stickies | blue | match column |
| In Review stickies | pink | concern hue |
| Done stickies | green | match column |

Color rationale: see [../color-conventions.md](../color-conventions.md).

## Personalization

Adapt to the user's request:

- **Team size N:** Add roughly N starter stickies in Backlog. Cap at 8 to keep the board readable.
- **Sprint number:** Substitute into the title (e.g., "Sprint 42 Planning"). If not provided, use "Sprint Planning" without a number.
- **Custom column names:** If the user asks for different columns ("we use Doing instead of In Progress"), keep the 4-column structure, swap the frame title text, keep the color mapping.
- **Empty start:** If the user says "no starter content", skip step 5 entirely.

## Acceptance

The produced board should:

- Have exactly 4 frames in a horizontal row, each 800x600.
- Have a centered title above the frames.
- Have stickies parented to the correct frames (no floating stickies on canvas root).
- Open in Miro at the returned URL with no overlapping items.

If any of those fail, fix before returning the URL.

## Common variations

- **5-column variant** (adds "Blocked"): use the `kanban` workflow instead; it handles N columns natively.
- **Issue-card variant** (use cards instead of stickies): replace `miro_create_sticky` with `miro_create_card`. Cards support title + description + assignee fields.
- **Long sprint with epics:** scale up; first row is 4 frames (small swimlanes per epic), each containing the 4-column structure. Beyond this skill's scope; route to the user as "we'd compose two boards" instead.

## Anti-patterns

- Skipping `parent_id` on stickies: they end up at canvas root, looking orphaned.
- Using canvas-absolute coords for stickies inside frames: stickies overlap the frame border or sit outside it.
- Adding more than 8 starter stickies per column: visual clutter; the user has to delete them anyway.
