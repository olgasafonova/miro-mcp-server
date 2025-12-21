# Miro MCP Server

A Model Context Protocol (MCP) server for controlling Miro whiteboards with AI assistants. Built in Go for performance and single-binary deployment.

## Features

- **29 tools** for complete Miro control
- **Board management**: Create, copy, delete, find by name
- **Create items**: Sticky notes, shapes, text, connectors, frames, cards, images, documents, embeds
- **Bulk operations**: Create multiple items at once, sticky grids
- **Tags**: Create, attach, and manage tags
- **Token validation**: Fails fast with clear error if token is invalid
- **Rate limiting**: Semaphore-based (5 concurrent requests)
- **Caching**: 2-minute TTL for board data
- **Retry with backoff**: Handles rate limits gracefully
- **Dual transport**: stdio (default) + HTTP
- **Voice-optimized**: Short, speakable responses

## Quick Start

### 1. Get a Miro Access Token

1. Go to [Miro Developer Settings](https://miro.com/app/settings/user-profile/apps)
2. Create a new app or use an existing one
3. Install the app to your team
4. Copy the access token

### 2. Install

```bash
# Clone and build
git clone https://github.com/olgasafonova/miro-mcp-server.git
cd miro-mcp-server
go build -o miro-mcp-server .
```

### 3. Configure

Set your access token:

```bash
export MIRO_ACCESS_TOKEN="your-token-here"
```

### 4. Run

**Stdio mode** (for Claude Desktop, Cursor):
```bash
./miro-mcp-server
```

**HTTP mode** (for remote clients):
```bash
./miro-mcp-server -http :8080
```

## Claude Desktop Configuration

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "miro": {
      "command": "/path/to/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token-here"
      }
    }
  }
}
```

## Available Tools (29 total)

### Board Tools
| Tool | Description |
|------|-------------|
| `miro_list_boards` | List accessible boards |
| `miro_get_board` | Get board details |
| `miro_create_board` | Create a new board |
| `miro_copy_board` | Copy an existing board |
| `miro_delete_board` | Delete a board (destructive) |
| `miro_find_board` | Find board by name (fuzzy match) |
| `miro_get_board_summary` | Get board with item counts and stats |

### Create Tools
| Tool | Description |
|------|-------------|
| `miro_create_sticky` | Create a sticky note |
| `miro_create_shape` | Create a shape (rectangle, circle, etc.) |
| `miro_create_text` | Create a text element |
| `miro_create_connector` | Connect two items with a line |
| `miro_create_frame` | Create a frame container |
| `miro_create_card` | Create a card with title, description, due date |
| `miro_create_image` | Add an image from URL |
| `miro_create_document` | Add a document (PDF, etc.) from URL |
| `miro_create_embed` | Embed external content (YouTube, Figma, etc.) |
| `miro_bulk_create` | Create multiple items at once |
| `miro_create_sticky_grid` | Create stickies in a grid layout |

### Read Tools
| Tool | Description |
|------|-------------|
| `miro_list_items` | List items on a board (paginated) |
| `miro_list_all_items` | Retrieve ALL items with automatic pagination |
| `miro_get_item` | Get full details of a specific item |
| `miro_search_board` | Search for items by content |

### Tag Tools
| Tool | Description |
|------|-------------|
| `miro_create_tag` | Create a new tag on a board |
| `miro_list_tags` | List all tags on a board |
| `miro_attach_tag` | Attach a tag to a sticky note |
| `miro_detach_tag` | Remove a tag from a sticky note |
| `miro_get_item_tags` | List tags on a specific item |

### Modify Tools
| Tool | Description |
|------|-------------|
| `miro_update_item` | Update an item's content or position |
| `miro_delete_item` | Delete an item (destructive) |

## Example Usage

### Voice Commands
- "Add a yellow sticky saying 'Review PRs'"
- "Create 5 stickies for our action items"
- "Draw a rectangle for the header"
- "Connect the first box to the second"
- "What boards do I have?"

### Programmatic

```
User: Add a sticky note to my Design board saying "MVP feature"
Assistant: [Uses miro_list_boards to find Design board]
         [Uses miro_create_sticky with content "MVP feature"]
         Created yellow sticky "MVP feature" on Design board
```

## Sticky Note Colors

| Color | Name |
|-------|------|
| 🟡 | yellow (default) |
| 🟢 | green |
| 🔵 | blue |
| 🩷 | pink |
| 🟠 | orange |
| 🔴 | red |
| ⚫ | gray |
| 🩵 | cyan |
| 🟣 | purple |

## Shape Types

**Basic**: rectangle, round_rectangle, circle, triangle, rhombus

**Extended**: parallelogram, trapezoid, pentagon, hexagon, octagon, star

**Flowchart**: flow_chart_predefined_process, wedge_round_rectangle_callout

## Tag Colors

| Color | Name |
|-------|------|
| 🔴 | red |
| 🩷 | magenta |
| 🟣 | violet |
| 🔵 | blue |
| 🩵 | cyan |
| 🟢 | green |
| 🟡 | yellow |
| 🟠 | orange |
| ⚫ | gray |

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `MIRO_ACCESS_TOKEN` | Yes | Your Miro OAuth access token |
| `MIRO_TIMEOUT` | No | Request timeout (default: 30s) |
| `MIRO_USER_AGENT` | No | Custom user agent string |

## Voice Demo Use Case

This server is designed for "manage your Miro with your voice" scenarios:

1. **Brainstorming**: "Add stickies for each of these ideas: A, B, C, D"
2. **Retrospectives**: "Create a frame called 'What went well'"
3. **Diagramming**: "Draw a box, then another box below it, connect them"
4. **Organization**: "Move all the red stickies to the Done frame"

## License

MIT