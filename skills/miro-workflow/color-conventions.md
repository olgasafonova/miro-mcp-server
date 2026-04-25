# Color Conventions

Sticky-note and frame colors carry meaning. The conventions below match the values in `prompts/prompts.go` so the skill produces boards that look identical to those from the MCP prompts.

## Two color vocabularies

The Miro API uses **different color formats** for stickies vs. frames. Mixing them up causes silent rendering bugs (stickies) or hard API errors (frames):

- **Stickies** (`miro_create_sticky`, `miro_bulk_create` with `type: "sticky_note"`): pass a **named** color string. Valid: `yellow`, `light_yellow`, `light_green`, `green`, `dark_green`, `cyan`, `light_pink`, `pink`, `violet`, `red`, `light_blue`, `blue`, `dark_blue`, `gray`, `orange`, `black`.
- **Frames** (`miro_create_frame`): pass a **CSS hex** string for the `color` parameter (the field is `style.fillColor` in the Miro REST API). Named values are rejected with `2.0703 invalid hex string`.

### Named → hex translation (for frame fills)

Use the light pastel hex when the workflow names a role color for a frame:

| Role color | Hex | Notes |
|-----------|------|-------|
| green (positive) | `#A6E5BB` | retro "Went Well", sprint "Done", story-map MVP |
| pink (concern) | `#F5D0E8` | retro "Could Improve", sprint "In Review" |
| blue (active) | `#A6CCF5` | retro "Action Items", sprint "In Progress", story-map activity headers |
| yellow (neutral) | `#FFF8B4` | sprint "In Review" frame fill |
| gray (inactive) | `#E6E6E6` | sprint "Backlog" frame fill |
| orange (accent) | `#FFD4A3` | kanban column 6 |
| cyan (accent) | `#B4E5E5` | kanban column 7 |

These are pastel approximations of Miro's UI palette, picked for legibility against dark sticky-note text.

## Role-based palette

| Color | Role | Used for |
|-------|------|----------|
| **yellow** | neutral / default | tasks, user activities, generic stickies, action items |
| **green** | positive / done | "What Went Well", "Done", MVP releases, completed work |
| **pink** | concern / review | "What Could Improve", "In Review", risks |
| **blue** | active / structural | "In Progress", activity headers, v1.0 release, framing |
| **gray** | inactive / background | "Backlog", future, deferred, reference content |
| **orange** | accent | brainstorm spark, highlight, emphasis |
| **cyan** | accent | brainstorm spark, secondary highlight |

## Per-workflow color map

### Sprint board

**Frame backgrounds** (column hue):
- Backlog: gray
- In Progress: blue
- In Review: yellow
- Done: green

**Sticky colors inside frames**:
- Backlog: yellow stickies (default tasks)
- In Progress: blue stickies (matches column)
- In Review: pink stickies (concern/review hue)
- Done: green stickies (matches column)

### Retrospective

**Frame backgrounds**:
- What Went Well: green
- What Could Improve: pink
- Action Items: blue

**Sticky colors**:
- What Went Well: green stickies
- What Could Improve: pink stickies
- Action Items: yellow stickies (action = neutral task hue)

### Brainstorm

Six radial stickies, each a different color to encourage idea variety:
1. yellow
2. green
3. blue
4. pink
5. orange
6. cyan

Central diamond shape: no fill (default Miro shape color is fine).

### Story map

- Activity headers: blue frames
- User tasks: yellow stickies
- MVP swimlane: green stickies
- v1.0 swimlane: blue stickies
- Future swimlane: gray stickies

### Kanban

Frame backgrounds cycle through the palette in order:
`gray → blue → yellow → green → pink → orange → cyan`

For N columns, use the first N colors. The cycle wraps if N > 7.

Sample stickies in the first column: yellow (neutral, since the column hue varies).

## Accessibility note

Miro's sticky colors are not colorblind-safe out of the box. When the skill produces a board for an audience that may include colorblind users, encode meaning in **text labels** as well as color. Don't rely on color alone to distinguish "Went Well" from "Could Improve"; the stickies should still read correctly if printed in greyscale.

## When to deviate

These conventions are defaults, not rules. Deviate when:

- The user explicitly asks for different colors ("make all stickies blue").
- The board is for a brand-themed event and a specific palette is required.
- A workflow not in the 5 listed needs its own scheme; pick role-based colors from the palette table above and document the choice in the response.

Otherwise, match the conventions. Consistency between the skill, the MCP prompts, and the wider Miro community matters more than local creativity.
