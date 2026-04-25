# Color Conventions

Sticky-note and frame colors carry meaning. The conventions below match the values in `prompts/prompts.go` so the skill produces boards that look identical to those from the MCP prompts.

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
- A workflow not in the canonical 5 needs its own scheme; pick role-based colors from the palette table above and document the choice in the response.

Otherwise, match the conventions. Consistency between the skill, the MCP prompts, and the wider Miro community matters more than local creativity.
