# Story Map Workflow

A user story mapping board: activity headers across the top, user tasks below, release swimlanes at the bottom. Two axes: time (left to right) and release scope (top to bottom).

## Triggers

Use this workflow when the user says:
- "Create a user story map"
- "Set up story mapping for <product>"
- "Map the user journey for <product>"
- "Plan releases for <product> as a story map"

## Tool sequence

1. **Pre-flight.** `miro_list_boards`; use existing if named, else:
2. **`miro_create_board`** with `name = "<product> Story Map"`. Save `board_id`.
3. **`miro_create_text`** at `(x=675, y=-100, font_size=48)` with `content = "<product> Story Map"`. Title.
4. **`miro_bulk_create`** for 4 activity-header frames (compact, blue):
   - Discovery: `(x=0, y=0, width=400, height=100, fill_color="blue")`
   - Onboarding: `(x=450, y=0, ..., fill_color="blue")`
   - Core Usage: `(x=900, y=0, ..., fill_color="blue")`
   - Growth: `(x=1350, y=0, ..., fill_color="blue")`
   Collect frame IDs for downstream parenting.
5. **`miro_bulk_create`** for user-task stickies (yellow, parented to the activity frame above each):
   - 2-3 yellow stickies under each activity at `y=150`. Frame-relative coords inside each activity frame, OR canvas-absolute if you don't parent (parenting is preferred).
   Example: under Discovery, stickies at canvas `(40, 150)`, `(40, 250)` if not parented.
6. **`miro_create_text`** for swimlane labels:
   - "MVP" at `(x=-100, y=350, font_size=24)`
   - "v1.0" at `(x=-100, y=550, font_size=24)`
   - "Future" at `(x=-100, y=750, font_size=24)`
7. **`miro_bulk_create`** for swimlane stickies (each sticky at the y-coord of its swimlane, x-aligned with its activity):
   - MVP row: green stickies (only items needed for first release)
   - v1.0 row: blue stickies (next-release items)
   - Future row: gray stickies (deferred items)
8. **Return** the board URL.

## Layout

```
y = -100  [        <product> Story Map (font 48)        ]
y = 0     [Discovery][Onboarding][Core Usage][Growth   ] ← blue frames
            x=0       x=450      x=900       x=1350
y = 150   [tasks   ][tasks    ][tasks      ][tasks    ] ← yellow stickies
y = 350   MVP       [green     ][green      ][green    ] ← green stickies
y = 550   v1.0      [blue      ][blue       ][blue     ] ← blue stickies
y = 750   Future    [gray      ][gray       ][gray     ] ← gray stickies
```

Activity frames are 400px wide with 50px gaps. Swimlane stickies align horizontally with activity columns.

Spatial details: see [../spatial-defaults.md](../spatial-defaults.md).

## Colors

| Element | Color | Why |
|---------|-------|-----|
| Activity headers | blue (frame fill) | structural backbone |
| User tasks | yellow (sticky) | tasks (neutral) |
| MVP swimlane | green (sticky) | done in first release |
| v1.0 swimlane | blue (sticky) | active / planned |
| Future swimlane | gray (sticky) | inactive / deferred |
| Swimlane labels | (text, no fill) | reference only |

Full palette: see [../color-conventions.md](../color-conventions.md).

## Personalization

- **Number of activities:** Default is 4 (Discovery / Onboarding / Core Usage / Growth). For a different product, use the user-supplied activity names. Maintain 50px gap between activities. Frame width can stay at 400 even with more activities; the board scales horizontally.
- **Number of swimlanes:** Default is 3 (MVP / v1.0 / Future). User may specify "Phase 1 / Phase 2 / Phase 3" or "Now / Next / Later". Swap labels, keep color mapping (first=green, middle=blue, last=gray).
- **Number of tasks per activity:** Scale to product complexity. 2-3 default; up to 5 if the user specifies a complex flow.
- **Add to existing board:** If a `board_id` was passed, place the entire structure with a Y offset to clear existing content (`miro_get_board_summary` to find a clear region).

## Variations

### Customer journey map
Replace "activities" with "phases" (Awareness / Consideration / Decision / Retention) and replace "swimlanes" with "channels" (Web / Mobile / Email / Support). Same shape, different labels. Stickies stay yellow; channels can use cyan/orange/yellow/pink to differentiate.

### Impact map
Top row: goals (1 frame). Middle row: actors (4 frames). Stickies below: deliverables, color-coded by goal. Different geometry; route to a custom layout instead of using this workflow's defaults.

## Acceptance

- 4 activity frames in a horizontal row (or as many as the user requested).
- 2-3 yellow user-task stickies under each activity.
- 3 swimlane labels at the left margin.
- Stickies in each swimlane row, color-matched (green / blue / gray top-to-bottom).
- Board URL returned.

## Anti-patterns

- Forgetting swimlane labels: without "MVP / v1.0 / Future" text, the swimlane rows look like random sticky clusters.
- Mixing colors within a swimlane: breaks the visual encoding (a green sticky in the v1.0 row reads as a stale MVP item).
- All stickies on canvas root with no relationship to activities: the spatial logic disappears. Either parent to activity frames OR keep coords aligned (`x` of sticky = `x` of activity column).
- Activity frames too narrow (< 300px): user-task text gets cramped.
