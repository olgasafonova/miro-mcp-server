# Miro MCP Server

A Model Context Protocol (MCP) server for controlling Miro whiteboards with AI assistants. Designed for voice interaction and hands-free whiteboard management.

## Features

- **Create items**: Sticky notes, shapes, text, connectors, frames
- **Bulk operations**: Add multiple items at once (up to 20)
- **Board management**: List and browse boards
- **Voice-optimized**: Short, speakable responses for voice assistants

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

## Available Tools

### Board Tools
| Tool | Description |
|------|-------------|
| `miro_list_boards` | List accessible boards |
| `miro_get_board` | Get board details |

### Create Tools
| Tool | Description |
|------|-------------|
| `miro_create_sticky` | Create a sticky note |
| `miro_create_shape` | Create a shape (rectangle, circle, etc.) |
| `miro_create_text` | Create a text element |
| `miro_create_connector` | Connect two items with a line |
| `miro_create_frame` | Create a frame container |
| `miro_bulk_create` | Create multiple items at once |

### Modify Tools
| Tool | Description |
|------|-------------|
| `miro_list_items` | List items on a board |
| `miro_update_item` | Update an item's content or position |
| `miro_delete_item` | Delete an item |

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