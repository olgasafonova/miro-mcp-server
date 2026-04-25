---
name: miro-workflow
version: 0.1.0
description: Compose Miro boards from natural-language requests by combining tools from miro-mcp-server. Use when the user asks to set up a sprint board, retrospective, brainstorming session, user story map, kanban, or any structured Miro layout. Do NOT use for single-tool calls (create one sticky), questions about Miro the product, or read-only board inspection.
allowed-tools:
  - mcp__miro__miro_list_boards
  - mcp__miro__miro_create_board
  - mcp__miro__miro_get_board
  - mcp__miro__miro_update_board
  - mcp__miro__miro_create_frame
  - mcp__miro__miro_update_frame
  - mcp__miro__miro_get_frame
  - mcp__miro__miro_get_frame_items
  - mcp__miro__miro_create_sticky
  - mcp__miro__miro_update_sticky
  - mcp__miro__miro_create_sticky_grid
  - mcp__miro__miro_create_text
  - mcp__miro__miro_update_text
  - mcp__miro__miro_create_shape
  - mcp__miro__miro_update_shape
  - mcp__miro__miro_create_flowchart_shape
  - mcp__miro__miro_create_connector
  - mcp__miro__miro_update_connector
  - mcp__miro__miro_list_connectors
  - mcp__miro__miro_create_card
  - mcp__miro__miro_update_card
  - mcp__miro__miro_create_mindmap_node
  - mcp__miro__miro_list_mindmap_nodes
  - mcp__miro__miro_create_group
  - mcp__miro__miro_get_group_items
  - mcp__miro__miro_bulk_create
  - mcp__miro__miro_list_items
  - mcp__miro__miro_get_board_summary
  - mcp__miro__miro_get_board_content
  - mcp__miro__miro_copy_board
  - mcp__miro__miro_find_board
---

# Miro Workflow Skill

Compose Miro boards from natural-language requests using the `miro-mcp-server` tools. The MCP exposes 91 atomic tools; this skill is the map.

---

## Trigger examples

Phrases that activate this skill (the routing decision happens via the `description` field; this list illustrates):

- "Set up a retro board for our team of 6"
- "Create a sprint planning board"
- "I need a brainstorm board for [topic]"
- "Build a user story map for [product]"
- "Make a kanban for our backlog"
- "Lay out a flowchart on Miro"

If the request is a single-tool call ("create one sticky note"), a question about Miro the product, or read-only inspection ("what's on this board?"), exit and let Claude handle it directly. The skill's job is composition, not single-call wrapping.

---

## The 5 canonical workflows

Each workflow is a tested composition with proven spatial defaults. Pick one based on intent:

| Workflow | Trigger Phrases | Detail File |
|----------|----------------|-------------|
| **Sprint Board** | "sprint planning", "sprint board", "sprint kickoff" | [workflows/sprint_board.md](workflows/sprint_board.md) |
| **Retrospective** | "retro", "retrospective", "what went well" | [workflows/retrospective.md](workflows/retrospective.md) |
| **Brainstorm** | "brainstorm", "ideation", "ideas around" | [workflows/brainstorm.md](workflows/brainstorm.md) |
| **Story Map** | "user story map", "story mapping", "user journey" | [workflows/story_map.md](workflows/story_map.md) |
| **Kanban** | "kanban", "task board", "workflow board" | [workflows/kanban.md](workflows/kanban.md) |

If the request doesn't match any of these, fall back to direct tool calls and ask the user what layout they want.

### Optional: seed boards

Before falling back to from-scratch construction, check if the user has imported a Miroverse template into their account that matches the requested workflow. See [seed-boards.md](seed-boards.md) for the lookup pattern. Seed boards are an optional power-user path; they produce designer-quality output but require one-time setup. The from-scratch workflows below work without any setup.

---

## Workflow selection guide

```
Goal: track sprint work?     → sprint_board (4 columns: Backlog, In Progress, Review, Done)
Goal: reflect on the team?   → retrospective (3 frames: Went Well, Could Improve, Action Items)
Goal: generate ideas?         → brainstorm (radial: central topic, 6 stickies in a ring)
Goal: map a product?         → story_map (header row + tasks + release swimlanes)
Goal: ongoing task flow?     → kanban (configurable column count)
```

Pick the workflow that matches the verb in the request (track, reflect, generate, map, manage). When in doubt, ask once.

---

## Universal pre-flight (always run first)

Before any workflow:

1. Call `miro_list_boards` to confirm the user's boards.
2. If the user named an existing board, get its ID. If not, create a fresh board with `miro_create_board`.
3. Save the `board_id` for every subsequent call.

Do NOT skip pre-flight. Without `board_id`, every other call fails.

---

## Spatial defaults (summary)

Full math in [spatial-defaults.md](spatial-defaults.md). Quick reference:

| Element | Default size | Default gap |
|---------|--------------|-------------|
| Frame (column) | 800 × 600 | 50px between frames |
| Sticky note | ~200 × 200 (Miro auto-sizes) | 40px between stickies |
| Connector | n/a | uses item IDs, not coords |
| Title text | font_size: 48 | 100px above first frame |

Origin (0, 0) is top-left in the board; the first frame sits at (0, 0) and successive frames stack to the right at `x = previous_x + 800 + 50`.

---

## Color conventions

Full table in [color-conventions.md](color-conventions.md). Quick map:

| Color | Used for |
|-------|----------|
| **yellow** | default, neutral content, tasks |
| **green** | positive (Went Well), Done, MVP |
| **pink** | concerns (Could Improve), Review |
| **blue** | In Progress, headers, activities |
| **gray** | Backlog, future, low priority |

Match the prompt-defined colors in `prompts/prompts.go` for consistency with users who fire the MCP prompts directly.

---

## Anti-patterns

These come up often. Avoid them.

1. **Stickies floating outside frames.** Always pass `parent_id` (the frame ID) when creating stickies inside a column. Otherwise the sticky lands at canvas root and looks orphaned.
2. **Wrong coordinate origin for parented items.** When `parent_id` is set, coordinates are **frame-relative** (NOT canvas-absolute). The frame's **top-left** is `(0, 0)` and the **item's CENTER** is placed at the given `(x, y)`. So for an 800×600 frame, a sticky at `(40, 40)` will overflow the frame's left and top edges by ~half the sticky's size. To stay fully inside an 800×600 frame: `x ∈ [100, 700]`, `y ∈ [114, 486]`. Center horizontally at `x = 400`.
3. **Frame `color` requires CSS hex, not a named color.** The Miro API returns `2.0703 invalid hex string` for named colors (`"green"`, `"blue"`, etc.) on frames. Pass hex like `"#A6E5BB"`. Stickies, in contrast, accept named values like `"light_green"`, `"yellow"`, `"light_pink"`. See [color-conventions.md](color-conventions.md) for the named→hex translation table.
4. **Missing `board_id`.** Every create/update call needs it. Re-confirm after `miro_create_board`.
5. **Sticky text > 280 chars.** Miro truncates. Break into multiple stickies if longer.
6. **Hand-rolled flowchart connectors with raw shapes.** Use `miro_create_flowchart_shape` (auto-sized for diagrams) instead of `miro_create_shape` when building flowcharts.
7. **Bulk creates without ordering.** `miro_bulk_create` is fast but doesn't guarantee item order in the response. Don't assume index N = the Nth created item.
8. **Skipping the title text.** Every workflow board gets a title at the top (font_size: 48). Without it, boards look unfinished.

---

## Troubleshooting

Common errors when calling the miro-mcp-server tools, with cause and fix.

| Error | Cause | Fix |
|-------|-------|-----|
| `401 Unauthorized` | `MIRO_ACCESS_TOKEN` missing or expired | Tell the user to set the env var; restart the MCP host |
| `404 Not Found` on `board_id` | Wrong board ID, or board outside the user's team | Re-run `miro_list_boards` to confirm the ID; ask the user which team |
| `429 Too Many Requests` | Too many tool calls in a short window | Use `miro_bulk_create` instead of individual calls; back off 5s and retry |
| `400 Bad Request` on sticky create | Invalid `color` name or `parent_id` not found | Check color is one of: yellow, green, blue, pink, gray, orange, cyan; confirm parent frame exists before creating children |
| Connector create fails with "item not found" | Bulk-create response order isn't guaranteed; the connector ran before the item ID was stable | Always create items, await the response, collect IDs, THEN create connectors in a second pass |
| Sticky text shows "..." (truncated) | Text exceeds Miro's ~280 char limit | Split the content into multiple stickies; keep each under 280 chars |
| Frame title doesn't appear | `title` was passed as empty string | Always provide a title; Miro renders the title bar regardless |
| Items overlap visually | Used canvas-absolute coords instead of frame-relative when `parent_id` was set | Use `(0, 0)` to mean "frame's top-left" when parented; not the canvas origin |
| Board URL returns 403 | Board is in a team the user isn't a member of | Confirm the user's team via `miro_list_boards`; recreate the board in their team |

If an error doesn't match anything above, return the raw error message to the user with the relevant tool name and the params you sent. Don't fabricate a diagnosis.

---

## Bulk creation guidance

When a workflow needs more than ~5 items of the same type (e.g., 12 stickies in a sprint board), prefer `miro_bulk_create` over individual calls. Single round-trip, lower rate-limit pressure. Trade-off: response order isn't guaranteed, so don't rely on indices for downstream connector calls; use returned IDs.

For workflows that need connectors between items, create items first (collect IDs), then connectors in a second pass.

---

## Common building blocks

All 5 workflows share these pieces:

### Title
```
miro_create_text(board_id, content="<Workflow Name>", x=center, y=-100, font_size=48)
```
Centered above the first frame. Always present.

### Column frames
```
miro_create_frame(board_id, title="<Column>", x=N*850, y=0, width=800, height=600, fill_color="<color>")
```
N is the column index (0, 1, 2, ...). 850 = 800 frame + 50 gap.

### Stickies inside a frame
```
miro_create_sticky(board_id, parent_id=<frame_id>, content="...", color="<color>", x=relative_x, y=relative_y)
```
`parent_id` is critical. Inside the frame, `(0, 0)` is the frame's top-left.

### Connectors between items
```
miro_create_connector(board_id, start_item_id=<id_a>, end_item_id=<id_b>, shape="curved")
```
Use IDs from prior creation calls. Connectors don't need coordinates.

---

## Output expectation

After completing any workflow, return:

1. The board URL (formed from `board_id`: `https://miro.com/app/board/<id>/`)
2. A one-line summary: "Created [workflow name] with [N frames, M stickies, K connectors]"
3. Any items skipped due to errors (with reason)

Never close out without the URL; that's the user's primary deliverable.

---

## Pairing with MCP prompts

The `miro-mcp-server` ships 5 MCP prompts (`create-sprint-board`, `create-retrospective`, `create-brainstorm`, `create-story-map`, `create-kanban`) that produce procedural instructions identical to these workflows. The skill's job is to fire when the user phrases a request *naturally* (without invoking a slash-prompt). Both routes converge on the same compositions; that's intentional.

If the user explicitly fires `/create-retrospective`, defer to the prompt. The skill is for the implicit ask.

---

## Best practices

### ✅ Do

- Run `miro_list_boards` first to confirm the user's workspace.
- Set `parent` on stickies that belong inside frames.
- Use color conventions consistently (green = positive, pink = concern, etc.).
- Bulk-create when N ≥ 5 same-type items.
- Return the board URL at the end.

### ❌ Don't

- Skip the title text.
- Use canvas-absolute coordinates for items inside frames.
- Create connectors before the items they connect exist.
- Assume bulk-create response order.
- Fire 91 tool calls when 5 will do.

---

## Detailed workflow files

For each canonical workflow (from-scratch construction):

- [workflows/sprint_board.md](workflows/sprint_board.md): 4-column tracker
- [workflows/retrospective.md](workflows/retrospective.md): 3-column reflection
- [workflows/brainstorm.md](workflows/brainstorm.md): radial idea ring
- [workflows/story_map.md](workflows/story_map.md): header + tasks + swimlanes
- [workflows/kanban.md](workflows/kanban.md): N-column workflow board

Optional power-user path:

- [seed-boards.md](seed-boards.md): copy + personalize Miroverse templates the user has imported

Read the relevant detail file before composing the tool sequence.
