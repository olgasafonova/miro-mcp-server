---
name: miro-boards
description: Create and manage Miro board content including stickies, shapes, connectors, diagrams, and mindmaps. Use for whiteboard collaboration, sprint planning, retrospectives, and brainstorming.
---

# Miro Board Management

## Trigger

Working with Miro whiteboards: creating boards, adding content, building diagrams, or running collaborative workflows like sprint planning or retrospectives.

## Workflow

1. **Discover boards** — Call `miro_list_boards` or `miro_find_board` to get the target board ID. Every subsequent operation needs a board ID.

2. **Read existing content** — Use `miro_get_board` for an overview, `miro_list_items` or `miro_list_all_items` for specific content, or `miro_search_board` to find items by text.

3. **Create items** — Use the appropriate creation tool:
   - `miro_create_sticky` for sticky notes (colors: yellow, green, blue, pink, orange, red, gray, cyan, purple)
   - `miro_create_shape` for shapes (rectangle, circle, triangle, rhombus, star, hexagon)
   - `miro_create_text` for text labels
   - `miro_create_frame` for grouping sections
   - `miro_create_connector` to connect items (requires source and target item IDs)
   - `miro_create_card` for structured cards with descriptions
   - `miro_create_mindmap_node` for mindmap branches
   - `miro_bulk_create` when adding multiple items at once

4. **Connect items** — Save item IDs from creation responses. Use `miro_create_connector` with start and end item IDs.

5. **Organize** — Use `miro_create_frame` to group related items, `miro_create_group` to lock items together, or `miro_attach_tag` to categorize.

## Workflow Templates

### Sprint Board
Create a frame for each column (Backlog, In Progress, Review, Done), then add stickies for tasks.

### Retrospective
Create three sections with colored stickies:
- Green: What went well
- Pink: What could improve
- Blue: Action items

### Brainstorming
Create a central topic shape, then radiate stickies outward. Use connectors to show relationships.

### Mermaid Diagrams
Use `miro_generate_diagram` to convert Mermaid syntax into visual diagrams on the board. Supports flowcharts and sequence diagrams.

## Guardrails

- Always start with `miro_list_boards` to get board IDs; never guess or hardcode them.
- Save returned item IDs for subsequent connection and update operations.
- Use `miro_bulk_create` for 3+ items instead of individual create calls.
- Keep confirmations short: "Added 3 stickies to Sprint Board" not lengthy descriptions.
- When updating items, read the current state first with `miro_get_item`.
