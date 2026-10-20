package main

// serverInstructions provides guidance for LLMs on how to use the Miro tools
const serverInstructions = `# Miro MCP Server - Voice-Friendly Whiteboard Control

## Quick Reference

### Creating Items
- miro_create_sticky: Add sticky notes ("add a yellow sticky saying Action Items")
- miro_create_shape: Add shapes ("create a rectangle for the header")
- miro_create_text: Add text labels ("write a title 'Sprint Goals'")
- miro_create_connector: Connect items ("draw an arrow from box A to box B")
- miro_create_frame: Create containers ("make a section for brainstorming")
- miro_bulk_create: Add multiple items ("add these 5 sticky notes")

### Reading
- miro_list_boards: Find boards ("show my boards", "find Design Sprint board")
- miro_get_board: Get board details
- miro_list_items: List items on a board ("what's on this board", "show all stickies")

### Modifying
- miro_update_item: Change items ("update sticky text", "move this shape")
- miro_delete_item: Remove items ("delete that sticky")

## MCP Resources

Access board content directly via resource URIs:
- miro://board/{board_id} - Get board summary with metadata and item counts
- miro://board/{board_id}/items - Get all items on a board
- miro://board/{board_id}/frames - Get all frames on a board

## MCP Prompts (Workflow Templates)

Use prompts for common workflows:
- create-sprint-board: Create sprint planning board with standard columns
- create-retrospective: Create retrospective with What Went Well/Could Improve/Action Items
- create-brainstorm: Set up brainstorming session with central topic
- create-story-map: Create user story mapping board
- create-kanban: Create kanban board with customizable columns

## Workflow

1. ALWAYS start with miro_list_boards to get board IDs
2. Use the board ID in all subsequent operations
3. When creating items, save the returned item IDs for connections

## Voice Interaction Tips

- Give SHORT confirmations: "Added 3 stickies to Design board"
- When listing, be concise: "Found 5 boards: Design, Retro, Planning..."
- For errors, explain simply: "Couldn't find that board. Try 'list my boards' first"

## Colors for Stickies

yellow, green, blue, pink, orange, red, gray, cyan, purple, dark_green, dark_blue, black

## Common Shapes

rectangle, round_rectangle, circle, triangle, rhombus, star, hexagon, pentagon

## Example Voice Commands -> Tool Mapping

| User Says | Use This Tool |
|-----------|---------------|
| "Add a sticky saying Review PRs" | miro_create_sticky |
| "Create 5 stickies for action items" | miro_bulk_create |
| "Draw a box around these ideas" | miro_create_frame |
| "Connect the first box to the second" | miro_create_connector |
| "What boards do I have?" | miro_list_boards |
| "What's on the Design board?" | miro_list_items |
| "Delete that last sticky" | miro_delete_item |

## Authentication

Requires MIRO_ACCESS_TOKEN environment variable.
Get a token at: https://miro.com/app/settings/user-profile/apps

Or use OAuth: ./miro-mcp-server auth login
`
