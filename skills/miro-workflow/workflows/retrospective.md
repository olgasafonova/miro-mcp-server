# Retrospective Workflow

A 3-column reflection board: What Went Well / What Could Improve / Action Items.

## Triggers

Use this workflow when the user says:
- "Create a retro board"
- "Set up a retrospective"
- "What went well, what could improve" board
- "Sprint retrospective for [team]"

## Tool sequence

1. **Pre-flight.** `miro_list_boards` to find an existing board if the user named one; otherwise:
2. **`miro_create_board`** with `name = "<team> Retrospective"`. Save `board_id`.
3. **`miro_create_text`** at `(x=850, y=-100, font_size=48)` with `content = "<team> Retrospective"`. The title spans the center.
4. **`miro_create_frame`** three times (or `miro_bulk_create`):
   - What Went Well: `(x=0, y=0, width=800, height=600, color="#A6E5BB")` (light green)
   - What Could Improve: `(x=850, y=0, ..., color="#F5D0E8")` (light pink)
   - Action Items: `(x=1700, y=0, ..., color="#A6CCF5")` (light blue)
   Frame `color` parameter takes a CSS hex string (Miro API rejects named colors for frames). See [../color-conventions.md](../color-conventions.md) for the hex palette.
   Collect frame IDs.
5. **`miro_bulk_create`** for starter stickies (parented to the matching frame). Sticky `color` accepts named values (`light_green`, `light_pink`, `yellow`).
   - What Went Well: 2 `light_green` stickies, e.g., "Team collaboration was excellent", "Shipped on time".
   - What Could Improve: 2 `light_pink` stickies, e.g., "Code review turnaround", "Mid-sprint scope creep".
   - Action Items: 2 `yellow` stickies, e.g., "Schedule weekly sync", "Define WIP limit".
   Frame-relative coords (frame top-left is `(0, 0)`; the sticky's CENTER is placed at the given coord): `(x=400, y=175)` and `(x=400, y=425)` give a clean 2-sticky stack inside an 800×600 frame.
6. **Return** the board URL.

## Layout

Total board width: ~2500px. Three 800px columns with 50px gaps.

```
y = -100  [   <team> Retrospective (font 48)   ]
y = 0     [Went Well ][Could Improve][Action Items]
            green       pink           blue
            x=0         x=850          x=1700
```

Spatial details: see [../spatial-defaults.md](../spatial-defaults.md).

## Colors

| Element | Color | Why |
|---------|-------|-----|
| Went Well frame | green | positive |
| Could Improve frame | pink | concern |
| Action Items frame | blue | active / structural |
| Went Well stickies | green | match column |
| Could Improve stickies | pink | match column |
| Action Items stickies | yellow | tasks (neutral); actions ≠ concerns |

Note: Action stickies are yellow, NOT blue. The frame itself is blue (structural cue), but the items inside are tasks and use the neutral task hue.

Full palette: see [../color-conventions.md](../color-conventions.md).

## Personalization

- **Team name:** Substitute into the title and board name (e.g., "Platform Team Retrospective"). If no team name given, default to "Team".
- **Sprint number:** Add to the title (e.g., "Platform Team Sprint 42 Retro"). Optional.
- **More starter prompts:** If the user wants the retro pre-seeded with discussion prompts ("seed it with last sprint's themes"), add up to 4 stickies per column.
- **Add to existing board:** If `board_id` was passed, skip step 2. Place the 3 frames in empty space (the user may need to drag them).

## Variations

### Glad / Sad / Mad (3-emotion variant)
Same shape, different column titles and colors:
- Glad: yellow frame (warmth)
- Sad: blue frame (calm)
- Mad: pink frame (heat)
Stickies all yellow (neutral content).

### Start / Stop / Continue
- Start: green frame (begin)
- Stop: pink frame (end)
- Continue: blue frame (carry forward)
Frame-relative sticky placement same as default.

### 4Ls (Liked / Learned / Lacked / Longed for)
Switch to a 4-column layout. Use the sprint board's 4-column geometry (x=0, 850, 1700, 2550) but keep retrospective coloring. Stickies are yellow throughout.

## Acceptance

- 3 frames in a horizontal row, 800×600 each.
- Centered title above.
- Stickies parented to correct frames, none floating.
- Color matches: green/pink/blue frames, green/pink/yellow stickies.
- Board URL returned.

## Anti-patterns

- Action Items stickies in pink: pink reads as "concern", not "task". Use yellow.
- Frames overlapping due to forgotten 50px gap.
- Forgetting `parent_id` on stickies (they orphan).
- Title text not centered (center column is at x=850, so title at x=850 with `align="center"` works).
