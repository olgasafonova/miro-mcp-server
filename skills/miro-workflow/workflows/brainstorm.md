# Brainstorm Workflow

A radial idea board: a central topic with 6 colored stickies fanning around it. No frames; the layout is the structure.

## Triggers

Use this workflow when the user says:
- "Brainstorm board for <topic>"
- "Set up an ideation session"
- "Help me generate ideas about X"
- "Brainstorm <topic>"

## Tool sequence

1. **Pre-flight.** `miro_list_boards`; if user named an existing board, use it. Otherwise:
2. **`miro_create_board`** with `name = "Brainstorm: <topic>"`. Save `board_id`.
3. **`miro_create_shape`** at `(x=400, y=300)` with `shape="diamond"` (or `"rectangle"`), `content = "<topic>"`. This is the central anchor.
4. **`miro_bulk_create`** for the 6 radial stickies (each at a fixed position, no `parent`):
   - Top: `(x=400, y=0)`, color="orange"
   - Top-left: `(x=100, y=100)`, color="yellow"
   - Top-right: `(x=700, y=100)`, color="green"
   - Bottom-left: `(x=100, y=500)`, color="blue"
   - Bottom-right: `(x=700, y=500)`, color="pink"
   - Bottom: `(x=400, y=600)`, color="cyan"
   Each sticky's `content` can be empty (let the user fill them in) or a placeholder prompt like "Idea 1...".
5. **Connectors from the central diamond to each sticky.** For each of the 6 stickies, call `miro_create_connector` with `start_item_id = <diamond_id>`, `end_item_id = <sticky_id>`, `shape = "curved"`. This creates the visual "fan" effect.
6. **Return** the board URL.

## Layout

```
                  orange (400, 0)
                       │
       yellow ─── DIAMOND ─── green
       (100,100)  (400,300)  (700,100)
                       │
       blue ──────────────────── pink
       (100,500)              (700,500)
                       │
                  cyan (400, 600)
```

Total width: 800px. Total height: 600px.

Spatial details: see [../spatial-defaults.md](../spatial-defaults.md).

## Colors

The 6 stickies use ALL palette colors except gray. The point: visual variety to encourage idea diversity. There's no semantic mapping; each color is just a fresh slot.

| Position | Color |
|----------|-------|
| Top (12 o'clock) | orange |
| Top-left | yellow |
| Top-right | green |
| Bottom-left | blue |
| Bottom-right | pink |
| Bottom (6 o'clock) | cyan |

Central shape: diamond, default fill (no color). The shape is structural, not semantic.

Full palette: see [../color-conventions.md](../color-conventions.md).

## Personalization

- **More than 6 stickies:** Expand the ring. For 8 stickies, add positions at `(0, 300)` and `(800, 300)`; left and right midpoints. For 12, use 30° increments around a 300px-radius circle.
- **Sub-themes:** If the user has known themes ("ideas around speed, cost, quality"), label the stickies with those theme prompts instead of leaving them blank. Use one theme per color.
- **Add to existing board:** If `board_id` was passed, find an empty region first (`miro_get_board_summary` to see existing item bounds), then offset all coords accordingly.
- **Voice / quick capture:** If the user is brainstorming aloud, leave the stickies blank; the user will fill them in interactively.

## Variations

### Lotus blossom (8-petal radial)
Center + 8 outer stickies in 8 slots. Then each outer sticky becomes its own center for 8 sub-ideas. Heavier; creates 9 + 64 = 73 items. Use only if the user explicitly asks.

### Affinity grouping (post-brainstorm)
After the user adds ideas to the radial brainstorm, the next step is grouping. Use `miro_create_group` to cluster stickies by theme. Beyond this workflow's scope but worth mentioning to the user.

### Crazy 8s sketch (8 sketches in 8 minutes)
Use a 4×2 grid of frames instead of radial layout. Each frame is one sketch slot. This is closer to a story map than a brainstorm; route to a custom layout.

## Acceptance

- 1 central shape (diamond) at (400, 300) with the topic text.
- 6 stickies, one of each color (yellow, green, blue, pink, orange, cyan), positioned in a ring.
- 6 curved connectors, each from the diamond to one sticky.
- No `parent` on any item; they live on canvas root.
- Board URL returned.

## Anti-patterns

- Putting stickies inside a frame: defeats the radial layout. No frames in this workflow.
- All stickies the same color: kills the visual variety that makes brainstorms feel generative.
- Connectors with `shape="straight"`: less inviting than `"curved"`. Always curved here.
- Pre-filling all 6 stickies with content: removes the invitation for the user to think. Leave them empty or use generic prompts.
